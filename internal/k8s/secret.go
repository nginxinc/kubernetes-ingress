/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8s

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	api_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// createSecretHandlers builds the handler funcs for secrets
func createSecretHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret := obj.(*v1.Secret)
			if !secrets.IsSupportedSecretType(secret.Type) {
				glog.V(3).Infof("Ignoring Secret %v of unsupported type %v", secret.Name, secret.Type)
				return
			}
			glog.V(3).Infof("Adding Secret: %v", secret.Name)
			lbc.AddSyncQueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			secret, isSecr := obj.(*v1.Secret)
			if !isSecr {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				secret, ok = deletedState.Obj.(*v1.Secret)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Secret object: %v", deletedState.Obj)
					return
				}
			}
			if !secrets.IsSupportedSecretType(secret.Type) {
				glog.V(3).Infof("Ignoring Secret %v of unsupported type %v", secret.Name, secret.Type)
				return
			}

			glog.V(3).Infof("Removing Secret: %v", secret.Name)
			lbc.AddSyncQueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			// A secret cannot change its type. That's why we only need to check the type of the current secret.
			curSecret := cur.(*v1.Secret)
			if !secrets.IsSupportedSecretType(curSecret.Type) {
				glog.V(3).Infof("Ignoring Secret %v of unsupported type %v", curSecret.Name, curSecret.Type)
				return
			}

			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Secret %v changed, syncing", cur.(*v1.Secret).Name)
				lbc.AddSyncQueue(cur)
			}
		},
	}
}

// addSecretHandler adds the handler for secrets to the controller
func (nsi *namespacedInformer) addSecretHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := nsi.secretInformerFactory.Core().V1().Secrets().Informer()
	informer.AddEventHandler(handlers) //nolint:errcheck,gosec
	nsi.secretLister = informer.GetStore()

	nsi.cacheSyncs = append(nsi.cacheSyncs, informer.HasSynced)
}

func (lbc *LoadBalancerController) syncSecret(task task) {
	key := task.Key
	var obj interface{}
	var secrExists bool
	var err error

	namespace, name, err := ParseNamespaceName(key)
	if err != nil {
		glog.Warningf("Secret key %v is invalid: %v", key, err)
		return
	}
	obj, secrExists, err = lbc.getNamespacedInformer(namespace).secretLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	resources := lbc.configuration.FindResourcesForSecret(namespace, name)

	if lbc.areCustomResourcesEnabled {
		secretPols := lbc.getPoliciesForSecret(namespace, name)
		for _, pol := range secretPols {
			resources = append(resources, lbc.configuration.FindResourcesForPolicy(pol.Namespace, pol.Name)...)
		}

		resources = removeDuplicateResources(resources)
	}

	glog.V(2).Infof("Found %v Resources with Secret %v", len(resources), key)

	if !secrExists {
		lbc.secretStore.DeleteSecret(key)

		glog.V(2).Infof("Deleting Secret: %v\n", key)

		if len(resources) > 0 {
			lbc.handleRegularSecretDeletion(resources)
		}
		if lbc.isSpecialSecret(key) {
			glog.Warningf("A special TLS Secret %v was removed. Retaining the Secret.", key)
		}
		return
	}

	glog.V(2).Infof("Adding / Updating Secret: %v\n", key)

	secret := obj.(*api_v1.Secret)

	lbc.secretStore.AddOrUpdateSecret(secret)

	if lbc.isSpecialSecret(key) {
		lbc.handleSpecialSecretUpdate(secret)
		// we don't return here in case the special secret is also used in resources.
	}

	if len(resources) > 0 {
		lbc.handleSecretUpdate(secret, resources)
	}
}

func (lbc *LoadBalancerController) isSpecialSecret(secretName string) bool {
	return secretName == lbc.defaultServerSecret || secretName == lbc.wildcardTLSSecret
}

func (lbc *LoadBalancerController) handleSecretUpdate(secret *api_v1.Secret, resources []Resource) {
	secretNsName := secret.Namespace + "/" + secret.Name

	var warnings configs.Warnings
	var addOrUpdateErr error

	resourceExes := lbc.createExtendedResources(resources)

	warnings, addOrUpdateErr = lbc.configurator.AddOrUpdateResources(resourceExes, !lbc.configurator.DynamicSSLReloadEnabled())
	if addOrUpdateErr != nil {
		glog.Errorf("Error when updating Secret %v: %v", secretNsName, addOrUpdateErr)
		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "%v was updated, but not applied: %v", secretNsName, addOrUpdateErr)
	}

	lbc.updateResourcesStatusAndEvents(resources, warnings, addOrUpdateErr)
}

func (lbc *LoadBalancerController) handleRegularSecretDeletion(resources []Resource) {
	resourceExes := lbc.createExtendedResources(resources)

	warnings, addOrUpdateErr := lbc.configurator.AddOrUpdateResources(resourceExes, true)

	lbc.updateResourcesStatusAndEvents(resources, warnings, addOrUpdateErr)
}

func (lbc *LoadBalancerController) handleSpecialSecretUpdate(secret *api_v1.Secret) {
	var specialSecretsToUpdate []string
	secretNsName := secret.Namespace + "/" + secret.Name
	err := secrets.ValidateTLSSecret(secret)
	if err != nil {
		glog.Errorf("Couldn't validate the special Secret %v: %v", secretNsName, err)
		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "Rejected", "the special Secret %v was rejected, using the previous version: %v", secretNsName, err)
		return
	}

	if secretNsName == lbc.defaultServerSecret {
		specialSecretsToUpdate = append(specialSecretsToUpdate, configs.DefaultServerSecretName)
	}
	if secretNsName == lbc.wildcardTLSSecret {
		specialSecretsToUpdate = append(specialSecretsToUpdate, configs.WildcardSecretName)
	}

	err = lbc.configurator.AddOrUpdateSpecialTLSSecrets(secret, specialSecretsToUpdate)
	if err != nil {
		glog.Errorf("Error when updating the special Secret %v: %v", secretNsName, err)
		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "the special Secret %v was updated, but not applied: %v", secretNsName, err)
		return
	}

	lbc.recorder.Eventf(secret, api_v1.EventTypeNormal, "Updated", "the special Secret %v was updated", secretNsName)
}

func (lbc *LoadBalancerController) addAPIKeySecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.APIKey == nil {
			continue
		}

		secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.APIKey.ClientSecret)
		secretRef := lbc.secretStore.GetSecret(secretKey)

		secretRefs[secretKey] = secretRef

		if secretRef.Error != nil {
			return secretRef.Error
		}

	}
	return nil
}

func (lbc *LoadBalancerController) addOIDCSecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.OIDC == nil {
			continue
		}

		secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.OIDC.ClientSecret)
		secretRef := lbc.secretStore.GetSecret(secretKey)

		secretRefs[secretKey] = secretRef

		if secretRef.Error != nil {
			return secretRef.Error
		}
	}
	return nil
}

func (lbc *LoadBalancerController) addEgressMTLSSecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.EgressMTLS == nil {
			continue
		}
		if pol.Spec.EgressMTLS.TLSSecret != "" {
			secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.EgressMTLS.TLSSecret)
			secretRef := lbc.secretStore.GetSecret(secretKey)

			secretRefs[secretKey] = secretRef

			if secretRef.Error != nil {
				return secretRef.Error
			}
		}
		if pol.Spec.EgressMTLS.TrustedCertSecret != "" {
			secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.EgressMTLS.TrustedCertSecret)
			secretRef := lbc.secretStore.GetSecret(secretKey)

			secretRefs[secretKey] = secretRef

			if secretRef.Error != nil {
				return secretRef.Error
			}
		}
	}

	return nil
}

func (lbc *LoadBalancerController) addIngressMTLSSecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.IngressMTLS == nil {
			continue
		}

		secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.IngressMTLS.ClientCertSecret)
		secretRef := lbc.secretStore.GetSecret(secretKey)

		secretRefs[secretKey] = secretRef

		return secretRef.Error
	}

	return nil
}

func (lbc *LoadBalancerController) addJWTSecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.JWTAuth == nil {
			continue
		}

		if pol.Spec.JWTAuth.JwksURI != "" {
			continue
		}

		secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.JWTAuth.Secret)
		secretRef := lbc.secretStore.GetSecret(secretKey)

		secretRefs[secretKey] = secretRef

		if secretRef.Error != nil {
			return secretRef.Error
		}
	}

	return nil
}

func (lbc *LoadBalancerController) addBasicSecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.BasicAuth == nil {
			continue
		}

		secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.BasicAuth.Secret)
		secretRef := lbc.secretStore.GetSecret(secretKey)

		secretRefs[secretKey] = secretRef

		if secretRef.Error != nil {
			return secretRef.Error
		}
	}

	return nil
}

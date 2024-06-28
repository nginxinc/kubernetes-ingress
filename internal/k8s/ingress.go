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

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	"golang.org/x/exp/maps"
	api_v1 "k8s.io/api/core/v1"
	discovery_v1 "k8s.io/api/discovery/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// createIngressHandlers builds the handler funcs for ingresses
func createIngressHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ingress := obj.(*networking.Ingress)
			glog.V(3).Infof("Adding Ingress: %v", ingress.Name)
			lbc.AddSyncQueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			ingress, isIng := obj.(*networking.Ingress)
			if !isIng {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				ingress, ok = deletedState.Obj.(*networking.Ingress)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Ingress object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing Ingress: %v", ingress.Name)
			lbc.AddSyncQueue(obj)
		},
		UpdateFunc: func(old, current interface{}) {
			c := current.(*networking.Ingress)
			o := old.(*networking.Ingress)
			if hasChanges(o, c) {
				glog.V(3).Infof("Ingress %v changed, syncing", c.Name)
				lbc.AddSyncQueue(c)
			}
		},
	}
}

// addIngressHandler adds the handler for ingresses to the controller
func (nsi *namespacedInformer) addIngressHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := nsi.sharedInformerFactory.Networking().V1().Ingresses().Informer()
	informer.AddEventHandler(handlers) //nolint:errcheck,gosec
	nsi.ingressLister = storeToIngressLister{Store: informer.GetStore()}

	nsi.cacheSyncs = append(nsi.cacheSyncs, informer.HasSynced)
}

func (lbc *LoadBalancerController) syncIngress(task task) {
	key := task.Key
	var ing *networking.Ingress
	var ingExists bool
	var err error

	ns, _, _ := cache.SplitMetaNamespaceKey(key)
	ing, ingExists, err = lbc.getNamespacedInformer(ns).ingressLister.GetByKeySafe(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []ResourceChange
	var problems []ConfigurationProblem

	if !ingExists {
		glog.V(2).Infof("Deleting Ingress: %v\n", key)

		changes, problems = lbc.configuration.DeleteIngress(key)
	} else {
		glog.V(2).Infof("Adding or Updating Ingress: %v\n", key)

		changes, problems = lbc.configuration.AddOrUpdateIngress(ing)
	}

	lbc.processChanges(changes)
	lbc.processProblems(problems)
}

// nolint:gocyclo
func (lbc *LoadBalancerController) createIngressEx(ing *networking.Ingress, validHosts map[string]bool, validMinionPaths map[string]bool) *configs.IngressEx {
	var endps []string
	ingEx := &configs.IngressEx{
		Ingress:          ing,
		ValidHosts:       validHosts,
		ValidMinionPaths: validMinionPaths,
	}

	ingEx.SecretRefs = make(map[string]*secrets.SecretReference)

	for _, tls := range ing.Spec.TLS {
		secretName := tls.SecretName
		secretKey := ing.Namespace + "/" + secretName

		secretRef := lbc.secretStore.GetSecret(secretKey)
		if secretRef.Error != nil {
			glog.Warningf("Error trying to get the secret %v for Ingress %v: %v", secretName, ing.Name, secretRef.Error)
		}

		ingEx.SecretRefs[secretName] = secretRef
	}

	if basicAuth, exists := ingEx.Ingress.Annotations[configs.BasicAuthSecretAnnotation]; exists {
		secretName := basicAuth
		secretKey := ing.Namespace + "/" + secretName

		secretRef := lbc.secretStore.GetSecret(secretKey)
		if secretRef.Error != nil {
			glog.Warningf("Error trying to get the secret %v for Ingress %v/%v: %v", secretName, ing.Namespace, ing.Name, secretRef.Error)
		}

		ingEx.SecretRefs[secretName] = secretRef
	}

	if lbc.isNginxPlus {
		if jwtKey, exists := ingEx.Ingress.Annotations[configs.JWTKeyAnnotation]; exists {
			secretName := jwtKey
			secretKey := ing.Namespace + "/" + secretName

			secretRef := lbc.secretStore.GetSecret(secretKey)
			if secretRef.Error != nil {
				glog.Warningf("Error trying to get the secret %v for Ingress %v/%v: %v", secretName, ing.Namespace, ing.Name, secretRef.Error)
			}

			ingEx.SecretRefs[secretName] = secretRef
		}
		if lbc.appProtectEnabled {
			if apPolicyAntn, exists := ingEx.Ingress.Annotations[configs.AppProtectPolicyAnnotation]; exists {
				policy, err := lbc.getAppProtectPolicy(ing)
				if err != nil {
					glog.Warningf("Error Getting App Protect policy %v for Ingress %v/%v: %v", apPolicyAntn, ing.Namespace, ing.Name, err)
				} else {
					ingEx.AppProtectPolicy = policy
				}
			}

			if apLogConfAntn, exists := ingEx.Ingress.Annotations[configs.AppProtectLogConfAnnotation]; exists {
				logConf, err := lbc.getAppProtectLogConfAndDst(ing)
				if err != nil {
					glog.Warningf("Error Getting App Protect Log Config %v for Ingress %v/%v: %v", apLogConfAntn, ing.Namespace, ing.Name, err)
				} else {
					ingEx.AppProtectLogs = logConf
				}
			}
		}

		if lbc.appProtectDosEnabled {
			if dosProtectedAnnotationValue, exists := ingEx.Ingress.Annotations[configs.AppProtectDosProtectedAnnotation]; exists {
				dosResEx, err := lbc.dosConfiguration.GetValidDosEx(ing.Namespace, dosProtectedAnnotationValue)
				if err != nil {
					glog.Warningf("Error Getting Dos Protected Resource %v for Ingress %v/%v: %v", dosProtectedAnnotationValue, ing.Namespace, ing.Name, err)
				}
				if dosResEx != nil {
					ingEx.DosEx = dosResEx
				}
			}
		}
	}

	ingEx.Endpoints = make(map[string][]string)
	ingEx.HealthChecks = make(map[string]*api_v1.Probe)
	ingEx.ExternalNameSvcs = make(map[string]bool)
	ingEx.PodsByIP = make(map[string]configs.PodInfo)
	hasUseClusterIP := ingEx.Ingress.Annotations[configs.UseClusterIPAnnotation] == "true"

	if ing.Spec.DefaultBackend != nil {
		podEndps := []podEndpoint{}
		var external bool
		svc, err := lbc.getServiceForIngressBackend(ing.Spec.DefaultBackend, ing.Namespace)
		if err != nil {
			glog.V(3).Infof("Error getting service %v: %v", ing.Spec.DefaultBackend.Service.Name, err)
		} else {
			podEndps, external, err = lbc.getEndpointsForIngressBackend(ing.Spec.DefaultBackend, svc)
			if err == nil && external && lbc.isNginxPlus {
				ingEx.ExternalNameSvcs[svc.Name] = true
			}
		}

		if err != nil {
			glog.Warningf("Error retrieving endpoints for the service %v: %v", ing.Spec.DefaultBackend.Service.Name, err)
		}

		if svc != nil && !external && hasUseClusterIP {
			if ing.Spec.DefaultBackend.Service.Port.Number == 0 {
				for _, port := range svc.Spec.Ports {
					if port.Name == ing.Spec.DefaultBackend.Service.Port.Name {
						ing.Spec.DefaultBackend.Service.Port.Number = port.Port
						break
					}
				}
			}
			endps = []string{ipv6SafeAddrPort(svc.Spec.ClusterIP, ing.Spec.DefaultBackend.Service.Port.Number)}
		} else {
			endps = getIPAddressesFromEndpoints(podEndps)
		}

		// endps is empty if there was any error before this point
		ingEx.Endpoints[ing.Spec.DefaultBackend.Service.Name+configs.GetBackendPortAsString(ing.Spec.DefaultBackend.Service.Port)] = endps

		if lbc.isNginxPlus && lbc.isHealthCheckEnabled(ing) {
			healthCheck := lbc.getHealthChecksForIngressBackend(ing.Spec.DefaultBackend, ing.Namespace)
			if healthCheck != nil {
				ingEx.HealthChecks[ing.Spec.DefaultBackend.Service.Name+configs.GetBackendPortAsString(ing.Spec.DefaultBackend.Service.Port)] = healthCheck
			}
		}

		if (lbc.isNginxPlus && lbc.isPrometheusEnabled) || lbc.isLatencyMetricsEnabled {
			for _, endpoint := range podEndps {
				ingEx.PodsByIP[endpoint.Address] = configs.PodInfo{
					Name:         endpoint.PodName,
					MeshPodOwner: endpoint.MeshPodOwner,
				}
			}
		}
	}

	for _, rule := range ing.Spec.Rules {
		if !validHosts[rule.Host] {
			glog.V(3).Infof("Skipping host %s for Ingress %s", rule.Host, ing.Name)
			continue
		}

		// check if rule has any paths
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			path := path // address gosec G601
			podEndps := []podEndpoint{}
			if validMinionPaths != nil && !validMinionPaths[path.Path] {
				glog.V(3).Infof("Skipping path %s for minion Ingress %s", path.Path, ing.Name)
				continue
			}

			var external bool
			svc, err := lbc.getServiceForIngressBackend(&path.Backend, ing.Namespace)
			if err != nil {
				glog.V(3).Infof("Error getting service %v: %v", &path.Backend.Service.Name, err)
			} else {
				podEndps, external, err = lbc.getEndpointsForIngressBackend(&path.Backend, svc)
				if err == nil && external && lbc.isNginxPlus {
					ingEx.ExternalNameSvcs[svc.Name] = true
				}
			}

			if err != nil {
				glog.Warningf("Error retrieving endpoints for the service %v: %v", path.Backend.Service.Name, err)
			}

			if svc != nil && !external && hasUseClusterIP {
				if path.Backend.Service.Port.Number == 0 {
					for _, port := range svc.Spec.Ports {
						if port.Name == path.Backend.Service.Port.Name {
							path.Backend.Service.Port.Number = port.Port
							break
						}
					}
				}
				endps = []string{ipv6SafeAddrPort(svc.Spec.ClusterIP, path.Backend.Service.Port.Number)}
			} else {
				endps = getIPAddressesFromEndpoints(podEndps)
			}

			// endps is empty if there was any error before this point
			ingEx.Endpoints[path.Backend.Service.Name+configs.GetBackendPortAsString(path.Backend.Service.Port)] = endps

			// Pull active health checks from k8 api
			if lbc.isNginxPlus && lbc.isHealthCheckEnabled(ing) {
				healthCheck := lbc.getHealthChecksForIngressBackend(&path.Backend, ing.Namespace)
				if healthCheck != nil {
					ingEx.HealthChecks[path.Backend.Service.Name+configs.GetBackendPortAsString(path.Backend.Service.Port)] = healthCheck
				}
			}

			if lbc.isNginxPlus || lbc.isLatencyMetricsEnabled {
				for _, endpoint := range podEndps {
					ingEx.PodsByIP[endpoint.Address] = configs.PodInfo{
						Name:         endpoint.PodName,
						MeshPodOwner: endpoint.MeshPodOwner,
					}
				}
			}
		}
	}

	return ingEx
}

func (lbc *LoadBalancerController) createMergeableIngresses(ingConfig *IngressConfiguration) *configs.MergeableIngresses {
	// for master Ingress, validMinionPaths are nil
	masterIngressEx := lbc.createIngressEx(ingConfig.Ingress, ingConfig.ValidHosts, nil)

	var minions []*configs.IngressEx

	for _, m := range ingConfig.Minions {
		minions = append(minions, lbc.createIngressEx(m.Ingress, ingConfig.ValidHosts, m.ValidPaths))
	}

	return &configs.MergeableIngresses{
		Master:  masterIngressEx,
		Minions: minions,
	}
}

func (lbc *LoadBalancerController) getHealthChecksForIngressBackend(backend *networking.IngressBackend, namespace string) *api_v1.Probe {
	svc, err := lbc.getServiceForIngressBackend(backend, namespace)
	if err != nil {
		glog.V(3).Infof("Error getting service %v: %v", backend.Service.Name, err)
		return nil
	}
	svcPort := lbc.getServicePortForIngressPort(backend.Service.Port, svc)
	if svcPort == nil {
		return nil
	}
	var pods []*api_v1.Pod
	nsi := lbc.getNamespacedInformer(svc.Namespace)
	pods, err = nsi.podLister.ListByNamespace(svc.Namespace, labels.Set(svc.Spec.Selector).AsSelector())
	if err != nil {
		glog.V(3).Infof("Error fetching pods for namespace %v: %v", svc.Namespace, err)
		return nil
	}
	return findProbeForPods(pods, svcPort)
}

func (lbc *LoadBalancerController) getExternalEndpointsForIngressBackend(backend *networking.IngressBackend, svc *api_v1.Service) []podEndpoint {
	address := fmt.Sprintf("%s:%d", svc.Spec.ExternalName, backend.Service.Port.Number)
	endpoints := []podEndpoint{
		{
			Address: address,
			PodName: "",
		},
	}
	return endpoints
}

func (lbc *LoadBalancerController) getEndpointsForIngressBackend(backend *networking.IngressBackend, svc *api_v1.Service) (result []podEndpoint, isExternal bool, err error) {
	var endpointSlices []discovery_v1.EndpointSlice
	endpointSlices, err = lbc.getNamespacedInformer(svc.Namespace).endpointSliceLister.GetServiceEndpointSlices(svc)
	if err != nil {
		if svc.Spec.Type == api_v1.ServiceTypeExternalName {
			if !lbc.isNginxPlus {
				return nil, false, fmt.Errorf("type ExternalName Services feature is only available in NGINX Plus")
			}
			result = lbc.getExternalEndpointsForIngressBackend(backend, svc)
			return result, true, nil
		}
		glog.V(3).Infof("Error getting endpoints for service %s from the cache: %v", svc.Name, err)
		return nil, false, err
	}

	result, err = lbc.getEndpointsForPortFromEndpointSlices(endpointSlices, backend.Service.Port, svc)
	if err != nil {
		glog.V(3).Infof("Error getting endpointslices for service %s port %v: %v", svc.Name, configs.GetBackendPortAsString(backend.Service.Port), err)
		return nil, false, err
	}
	return result, false, nil
}

func (lbc *LoadBalancerController) getServicePortForIngressPort(backendPort networking.ServiceBackendPort, svc *api_v1.Service) *api_v1.ServicePort {
	for _, port := range svc.Spec.Ports {
		if (backendPort.Name == "" && port.Port == backendPort.Number) || port.Name == backendPort.Name {
			return &port
		}
	}
	return nil
}

func (lbc *LoadBalancerController) getServiceForIngressBackend(backend *networking.IngressBackend, namespace string) (*api_v1.Service, error) {
	svcKey := namespace + "/" + backend.Service.Name
	var svcObj interface{}
	var svcExists bool
	var err error

	svcObj, svcExists, err = lbc.getNamespacedInformer(namespace).svcLister.GetByKey(svcKey)
	if err != nil {
		return nil, err
	}

	if svcExists {
		return svcObj.(*api_v1.Service), nil
	}

	return nil, fmt.Errorf("service %s doesn't exist", svcKey)
}

func (lbc *LoadBalancerController) updateIngressMetrics() {
	counters := lbc.configurator.GetIngressCounts()
	for nType, count := range counters {
		lbc.metricsCollector.SetIngresses(nType, count)
	}
}

func (lbc *LoadBalancerController) getEndpointsForPortFromEndpointSlices(endpointSlices []discovery_v1.EndpointSlice, backendPort networking.ServiceBackendPort, svc *api_v1.Service) ([]podEndpoint, error) {
	var targetPort int32
	var err error

	for _, port := range svc.Spec.Ports {
		if (backendPort.Name == "" && port.Port == backendPort.Number) || port.Name == backendPort.Name {
			targetPort, err = lbc.getTargetPort(port, svc)
			if err != nil {
				return nil, fmt.Errorf("error determining target port for port %v in Ingress: %w", backendPort, err)
			}
			break
		}
	}

	if targetPort == 0 {
		return nil, fmt.Errorf("no port %v in service %s", backendPort, svc.Name)
	}

	makePodEndpoints := func(port int32, epx []discovery_v1.Endpoint) []podEndpoint {
		endpointSet := make(map[podEndpoint]struct{})

		for _, ep := range epx {
			for _, addr := range ep.Addresses {
				address := ipv6SafeAddrPort(addr, port)
				podEndpoint := podEndpoint{
					Address: address,
				}
				if ep.TargetRef != nil {
					parentType, parentName := lbc.getPodOwnerTypeAndNameFromAddress(ep.TargetRef.Namespace, ep.TargetRef.Name)
					podEndpoint.OwnerType = parentType
					podEndpoint.OwnerName = parentName
					podEndpoint.PodName = ep.TargetRef.Name
				}
				endpointSet[podEndpoint] = struct{}{}
			}
		}
		return maps.Keys(endpointSet)
	}

	endpoints := makePodEndpoints(targetPort, filterReadyEndpointsFrom(selectEndpointSlicesForPort(targetPort, endpointSlices)))
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpointslices for target port %v in service %s", targetPort, svc.Name)
	}
	return endpoints, nil
}

// isHealthCheckEnabled checks if health checks are enabled so we can only query pods if enabled.
func (lbc *LoadBalancerController) isHealthCheckEnabled(ing *networking.Ingress) bool {
	if healthCheckEnabled, exists, err := configs.GetMapKeyAsBool(ing.Annotations, "nginx.com/health-checks", ing); exists {
		if err != nil {
			glog.Error(err)
		}
		return healthCheckEnabled
	}
	return false
}

func (lbc *LoadBalancerController) getPodOwnerTypeAndNameFromAddress(ns, name string) (parentType, parentName string) {
	var obj interface{}
	var exists bool
	var err error

	obj, exists, err = lbc.getNamespacedInformer(ns).podLister.GetByKey(fmt.Sprintf("%s/%s", ns, name))
	if err != nil {
		glog.Warningf("could not get pod by key %s/%s: %v", ns, name, err)
		return "", ""
	}
	if exists {
		pod := obj.(*api_v1.Pod)
		return getPodOwnerTypeAndName(pod)
	}
	return "", ""
}

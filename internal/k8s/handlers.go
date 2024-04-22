package k8s

import (
	"fmt"
	"reflect"
	"sort"

	discovery_v1 "k8s.io/api/discovery/v1"

	"github.com/jinzhu/copier"
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/dos/v1beta1"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/cache"

	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// createConfigMapHandlers builds the handler funcs for config maps
func createConfigMapHandlers(lbc *LoadBalancerController, name string) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			configMap := obj.(*v1.ConfigMap)
			if configMap.Name == name {
				glog.V(3).Infof("Adding ConfigMap: %v", configMap.Name)
				lbc.AddSyncQueue(obj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			configMap, isConfigMap := obj.(*v1.ConfigMap)
			if !isConfigMap {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				configMap, ok = deletedState.Obj.(*v1.ConfigMap)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-ConfigMap object: %v", deletedState.Obj)
					return
				}
			}
			if configMap.Name == name {
				glog.V(3).Infof("Removing ConfigMap: %v", configMap.Name)
				lbc.AddSyncQueue(obj)
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				configMap := cur.(*v1.ConfigMap)
				if configMap.Name == name {
					glog.V(3).Infof("ConfigMap %v changed, syncing", cur.(*v1.ConfigMap).Name)
					lbc.AddSyncQueue(cur)
				}
			}
		},
	}
}

// createEndpointSliceHandlers builds the handler funcs for EndpointSlices
func createEndpointSliceHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpointSlice := obj.(*discovery_v1.EndpointSlice)
			glog.V(3).Infof("Adding EndpointSlice: %v", endpointSlice.Name)
			lbc.AddSyncQueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			endpointSlice, isEndpointSlice := obj.(*discovery_v1.EndpointSlice)
			if !isEndpointSlice {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				endpointSlice, ok = deletedState.Obj.(*discovery_v1.EndpointSlice)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-EndpointSlice object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing EndpointSlice: %v", endpointSlice.Name)
			lbc.AddSyncQueue(obj)
		}, UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("EndpointSlice %v changed, syncing", cur.(*discovery_v1.EndpointSlice).Name)
				lbc.AddSyncQueue(cur)
			}
		},
	}
}

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

// createServiceHandlers builds the handler funcs for services.
//
// In the update handlers below we catch two cases:
// (1) the service is the external service
// (2) the service had a change like a change of the port field of a service port (for such a change Kubernetes doesn't
// update the corresponding endpoints resource, that we monitor as well)
// or a change of the externalName field of an ExternalName service.
//
// In both cases we enqueue the service to be processed by syncService
func createServiceHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)

			glog.V(3).Infof("Adding service: %v", svc.Name)
			lbc.AddSyncQueue(svc)
		},
		DeleteFunc: func(obj interface{}) {
			svc, isSvc := obj.(*v1.Service)
			if !isSvc {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				svc, ok = deletedState.Obj.(*v1.Service)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Service object: %v", deletedState.Obj)
					return
				}
			}

			glog.V(3).Infof("Removing service: %v", svc.Name)
			lbc.AddSyncQueue(svc)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				curSvc := cur.(*v1.Service)
				if lbc.IsExternalServiceForStatus(curSvc) {
					lbc.AddSyncQueue(curSvc)
					return
				}
				oldSvc := old.(*v1.Service)
				if hasServiceChanges(oldSvc, curSvc) {
					glog.V(3).Infof("Service %v changed, syncing", curSvc.Name)
					lbc.AddSyncQueue(curSvc)
				}
			}
		},
	}
}

type portSort []v1.ServicePort

func (a portSort) Len() int {
	return len(a)
}

func (a portSort) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a portSort) Less(i, j int) bool {
	if a[i].Name == a[j].Name {
		return a[i].Port < a[j].Port
	}
	return a[i].Name < a[j].Name
}

// hasServicedChanged checks if the service has changed based on custom rules we define (eg. port).
func hasServiceChanges(oldSvc, curSvc *v1.Service) bool {
	if hasServicePortChanges(oldSvc.Spec.Ports, curSvc.Spec.Ports) {
		return true
	}
	if hasServiceExternalNameChanges(oldSvc, curSvc) {
		return true
	}
	return false
}

// hasServiceExternalNameChanges only compares Service.Spec.Externalname for Type ExternalName services.
func hasServiceExternalNameChanges(oldSvc, curSvc *v1.Service) bool {
	return curSvc.Spec.Type == v1.ServiceTypeExternalName && oldSvc.Spec.ExternalName != curSvc.Spec.ExternalName
}

// hasServicePortChanges only compares ServicePort.Name and .Port.
func hasServicePortChanges(oldServicePorts []v1.ServicePort, curServicePorts []v1.ServicePort) bool {
	if len(oldServicePorts) != len(curServicePorts) {
		return true
	}

	sort.Sort(portSort(oldServicePorts))
	sort.Sort(portSort(curServicePorts))

	for i := range oldServicePorts {
		if oldServicePorts[i].Port != curServicePorts[i].Port ||
			oldServicePorts[i].Name != curServicePorts[i].Name {
			return true
		}
	}
	return false
}

func createVirtualServerHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			vs := obj.(*conf_v1.VirtualServer)
			glog.V(3).Infof("Adding VirtualServer: %v", vs.Name)
			lbc.AddSyncQueue(vs)
		},
		DeleteFunc: func(obj interface{}) {
			vs, isVs := obj.(*conf_v1.VirtualServer)
			if !isVs {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				vs, ok = deletedState.Obj.(*conf_v1.VirtualServer)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-VirtualServer object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing VirtualServer: %v", vs.Name)
			lbc.AddSyncQueue(vs)
		},
		UpdateFunc: func(old, cur interface{}) {
			curVs := cur.(*conf_v1.VirtualServer)
			oldVs := old.(*conf_v1.VirtualServer)

			if lbc.weightChangesDynamicReload {
				var curVsCopy, oldVsCopy conf_v1.VirtualServer
				err := copier.CopyWithOption(&curVsCopy, curVs, copier.Option{DeepCopy: true})
				if err != nil {
					glog.V(3).Infof("Error copying VirtualServer %v: %v for Dynamic Weight Changes", curVs.Name, err)
					return
				}

				err = copier.CopyWithOption(&oldVsCopy, oldVs, copier.Option{DeepCopy: true})
				if err != nil {
					glog.V(3).Infof("Error copying VirtualServer %v: %v for Dynamic Weight Changes", oldVs.Name, err)
					return
				}

				zeroOutVirtualServerSplitWeights(&curVsCopy)
				zeroOutVirtualServerSplitWeights(&oldVsCopy)

				if reflect.DeepEqual(oldVsCopy.Spec, curVsCopy.Spec) {
					lbc.processVSWeightChangesDynamicReload(oldVs, curVs)
					return
				}

			}

			if !reflect.DeepEqual(oldVs.Spec, curVs.Spec) {
				glog.V(3).Infof("VirtualServer %v changed, syncing", curVs.Name)
				lbc.AddSyncQueue(curVs)
			}
		},
	}
}

func createVirtualServerRouteHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			vsr := obj.(*conf_v1.VirtualServerRoute)
			glog.V(3).Infof("Adding VirtualServerRoute: %v", vsr.Name)
			lbc.AddSyncQueue(vsr)
		},
		DeleteFunc: func(obj interface{}) {
			vsr, isVsr := obj.(*conf_v1.VirtualServerRoute)
			if !isVsr {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				vsr, ok = deletedState.Obj.(*conf_v1.VirtualServerRoute)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-VirtualServerRoute object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing VirtualServerRoute: %v", vsr.Name)
			lbc.AddSyncQueue(vsr)
		},
		UpdateFunc: func(old, cur interface{}) {
			curVsr := cur.(*conf_v1.VirtualServerRoute)
			oldVsr := old.(*conf_v1.VirtualServerRoute)

			if lbc.weightChangesDynamicReload {
				var curVsrCopy, oldVsrCopy conf_v1.VirtualServerRoute
				err := copier.CopyWithOption(&curVsrCopy, curVsr, copier.Option{DeepCopy: true})
				if err != nil {
					glog.V(3).Infof("Error copying VirtualServerRoute %v: %v for Dynamic Weight Changes", curVsr.Name, err)
					return
				}

				err = copier.CopyWithOption(&oldVsrCopy, oldVsr, copier.Option{DeepCopy: true})
				if err != nil {
					glog.V(3).Infof("Error copying VirtualServerRoute %v: %v for Dynamic Weight Changes", oldVsr.Name, err)
					return
				}

				zeroOutVirtualServerRouteSplitWeights(&curVsrCopy)
				zeroOutVirtualServerRouteSplitWeights(&oldVsrCopy)

				if reflect.DeepEqual(oldVsrCopy.Spec, curVsrCopy.Spec) {
					lbc.processVSRWeightChangesDynamicReload(oldVsr, curVsr)
					return
				}

			}

			if !reflect.DeepEqual(oldVsr.Spec, curVsr.Spec) {
				glog.V(3).Infof("VirtualServerRoute %v changed, syncing", curVsr.Name)
				lbc.AddSyncQueue(curVsr)
			}
		},
	}
}

func createGlobalConfigurationHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			gc := obj.(*conf_v1.GlobalConfiguration)
			glog.V(3).Infof("Adding GlobalConfiguration: %v", gc.Name)
			lbc.AddSyncQueue(gc)
		},
		DeleteFunc: func(obj interface{}) {
			gc, isGc := obj.(*conf_v1.GlobalConfiguration)
			if !isGc {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				gc, ok = deletedState.Obj.(*conf_v1.GlobalConfiguration)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-GlobalConfiguration object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing GlobalConfiguration: %v", gc.Name)
			lbc.AddSyncQueue(gc)
		},
		UpdateFunc: func(old, cur interface{}) {
			curGc := cur.(*conf_v1.GlobalConfiguration)
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("GlobalConfiguration %v changed, syncing", curGc.Name)
				lbc.AddSyncQueue(curGc)
			}
		},
	}
}

func createTransportServerHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ts := obj.(*conf_v1.TransportServer)
			glog.V(3).Infof("Adding TransportServer: %v", ts.Name)
			lbc.AddSyncQueue(ts)
		},
		DeleteFunc: func(obj interface{}) {
			ts, isTs := obj.(*conf_v1.TransportServer)
			if !isTs {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				ts, ok = deletedState.Obj.(*conf_v1.TransportServer)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-TransportServer object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing TransportServer: %v", ts.Name)
			lbc.AddSyncQueue(ts)
		},
		UpdateFunc: func(old, cur interface{}) {
			curTs := cur.(*conf_v1.TransportServer)
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("TransportServer %v changed, syncing", curTs.Name)
				lbc.AddSyncQueue(curTs)
			}
		},
	}
}

func createPolicyHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pol := obj.(*conf_v1.Policy)
			glog.V(3).Infof("Adding Policy: %v", pol.Name)
			lbc.AddSyncQueue(pol)
		},
		DeleteFunc: func(obj interface{}) {
			pol, isPol := obj.(*conf_v1.Policy)
			if !isPol {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				pol, ok = deletedState.Obj.(*conf_v1.Policy)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Policy object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing Policy: %v", pol.Name)
			lbc.AddSyncQueue(pol)
		},
		UpdateFunc: func(old, cur interface{}) {
			curPol := cur.(*conf_v1.Policy)
			oldPol := old.(*conf_v1.Policy)
			if !reflect.DeepEqual(oldPol.Spec, curPol.Spec) {
				glog.V(3).Infof("Policy %v changed, syncing", curPol.Name)
				lbc.AddSyncQueue(curPol)
			}
		},
	}
}

func createIngressLinkHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			link := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding IngressLink: %v", link.GetName())
			lbc.AddSyncQueue(link)
		},
		DeleteFunc: func(obj interface{}) {
			link, isUnstructured := obj.(*unstructured.Unstructured)

			if !isUnstructured {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				link, ok = deletedState.Obj.(*unstructured.Unstructured)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Unstructured object: %v", deletedState.Obj)
					return
				}
			}

			glog.V(3).Infof("Removing IngressLink: %v", link.GetName())
			lbc.AddSyncQueue(link)
		},
		UpdateFunc: func(old, cur interface{}) {
			oldLink := old.(*unstructured.Unstructured)
			curLink := cur.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldLink, curLink)
			if err != nil {
				glog.V(3).Infof("Error when comparing IngressLinks: %v", err)
				lbc.AddSyncQueue(curLink)
			}
			if different {
				glog.V(3).Infof("IngressLink %v changed, syncing", oldLink.GetName())
				lbc.AddSyncQueue(curLink)
			}
		},
	}
}

func createAppProtectPolicyHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pol := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding AppProtectPolicy: %v", pol.GetName())
			lbc.AddSyncQueue(pol)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			oldPol := oldObj.(*unstructured.Unstructured)
			newPol := obj.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldPol, newPol)
			if err != nil {
				glog.V(3).Infof("Error when comparing policy %v", err)
				lbc.AddSyncQueue(newPol)
			}
			if different {
				glog.V(3).Infof("ApPolicy %v changed, syncing", oldPol.GetName())
				lbc.AddSyncQueue(newPol)
			}
		},
		DeleteFunc: func(obj interface{}) {
			lbc.AddSyncQueue(obj)
		},
	}
	return handlers
}

// areResourcesDifferent returns true if the resources are different based on their spec.
func areResourcesDifferent(oldresource, resource *unstructured.Unstructured) (bool, error) {
	oldSpec, found, err := unstructured.NestedMap(oldresource.Object, "spec")
	if !found {
		glog.V(3).Infof("Warning, oldspec has unexpected format")
	}
	if err != nil {
		return false, err
	}
	spec, found, err := unstructured.NestedMap(resource.Object, "spec")
	if err != nil {
		return false, err
	}
	if !found {
		return false, fmt.Errorf("spec has unexpected format")
	}
	eq := reflect.DeepEqual(oldSpec, spec)
	if eq {
		glog.V(3).Infof("New spec of %v same as old spec", oldresource.GetName())
	}
	return !eq, nil
}

func createAppProtectLogConfHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			conf := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding AppProtectLogConf: %v", conf.GetName())
			lbc.AddSyncQueue(conf)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			oldConf := oldObj.(*unstructured.Unstructured)
			newConf := obj.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldConf, newConf)
			if err != nil {
				glog.V(3).Infof("Error when comparing LogConfs %v", err)
				lbc.AddSyncQueue(newConf)
			}
			if different {
				glog.V(3).Infof("ApLogConf %v changed, syncing", oldConf.GetName())
				lbc.AddSyncQueue(newConf)
			}
		},
		DeleteFunc: func(obj interface{}) {
			lbc.AddSyncQueue(obj)
		},
	}
	return handlers
}

func createAppProtectUserSigHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sig := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding AppProtectUserSig: %v", sig.GetName())
			lbc.AddSyncQueue(sig)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			oldSig := oldObj.(*unstructured.Unstructured)
			newSig := obj.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldSig, newSig)
			if err != nil {
				glog.V(3).Infof("Error when comparing UserSigs %v", err)
				lbc.AddSyncQueue(newSig)
			}
			if different {
				glog.V(3).Infof("ApUserSig %v changed, syncing", oldSig.GetName())
				lbc.AddSyncQueue(newSig)
			}
		},
		DeleteFunc: func(obj interface{}) {
			lbc.AddSyncQueue(obj)
		},
	}
	return handlers
}

func createAppProtectDosPolicyHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pol := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding AppProtectDosPolicy: %v", pol.GetName())
			lbc.AddSyncQueue(pol)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			oldPol := oldObj.(*unstructured.Unstructured)
			newPol := obj.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldPol, newPol)
			if err != nil {
				glog.V(3).Infof("Error when comparing policy %v", err)
				lbc.AddSyncQueue(newPol)
			}
			if different {
				glog.V(3).Infof("ApDosPolicy %v changed, syncing", oldPol.GetName())
				lbc.AddSyncQueue(newPol)
			}
		},
		DeleteFunc: func(obj interface{}) {
			lbc.AddSyncQueue(obj)
		},
	}
	return handlers
}

func createAppProtectDosLogConfHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			conf := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding AppProtectDosLogConf: %v", conf.GetName())
			lbc.AddSyncQueue(conf)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			oldConf := oldObj.(*unstructured.Unstructured)
			newConf := obj.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldConf, newConf)
			if err != nil {
				glog.V(3).Infof("Error when comparing DosLogConfs %v", err)
				lbc.AddSyncQueue(newConf)
			}
			if different {
				glog.V(3).Infof("ApDosLogConf %v changed, syncing", oldConf.GetName())
				lbc.AddSyncQueue(newConf)
			}
		},
		DeleteFunc: func(obj interface{}) {
			lbc.AddSyncQueue(obj)
		},
	}
	return handlers
}

func createAppProtectDosProtectedResourceHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			conf := obj.(*v1beta1.DosProtectedResource)
			glog.V(3).Infof("Adding DosProtectedResource: %v", conf.GetName())
			lbc.AddSyncQueue(conf)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			oldConf := oldObj.(*v1beta1.DosProtectedResource)
			newConf := obj.(*v1beta1.DosProtectedResource)

			if !reflect.DeepEqual(oldConf.Spec, newConf.Spec) {
				glog.V(3).Infof("DosProtectedResource %v changed, syncing", oldConf.GetName())
				lbc.AddSyncQueue(newConf)
			}
		},
		DeleteFunc: func(obj interface{}) {
			lbc.AddSyncQueue(obj)
		},
	}
	return handlers
}

// createNamespaceHandlers builds the handler funcs for namespaces
func createNamespaceHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ns := obj.(*v1.Namespace)
			glog.V(3).Infof("Adding Namespace to list of watched Namespaces: %v", ns.Name)
			lbc.AddSyncQueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			ns, isNs := obj.(*v1.Namespace)
			if !isNs {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				ns, ok = deletedState.Obj.(*v1.Namespace)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Namespace object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing Namespace from list of watched Namespaces: %v", ns.Name)
			lbc.AddSyncQueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Namespace %v changed, syncing", cur.(*v1.Namespace).Name)
				lbc.AddSyncQueue(cur)
			}
		},
	}
}

func zeroOutVirtualServerSplitWeights(vs *conf_v1.VirtualServer) {
	for _, route := range vs.Spec.Routes {
		for _, match := range route.Matches {
			if len(match.Splits) == 2 {
				match.Splits[0].Weight = 0
				match.Splits[1].Weight = 0
			}
		}

		if len(route.Splits) == 2 {
			route.Splits[0].Weight = 0
			route.Splits[1].Weight = 0
		}
	}
}

func zeroOutVirtualServerRouteSplitWeights(vs *conf_v1.VirtualServerRoute) {
	for _, route := range vs.Spec.Subroutes {
		for _, match := range route.Matches {
			if len(match.Splits) == 2 {
				match.Splits[0].Weight = 0
				match.Splits[1].Weight = 0
			}
		}

		if len(route.Splits) == 2 {
			route.Splits[0].Weight = 0
			route.Splits[1].Weight = 0
		}
	}
}

package k8s

import (
	"reflect"
	"sort"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"

	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
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

// createEndpointHandlers builds the handler funcs for endpoints
func createEndpointHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoint := obj.(*v1.Endpoints)
			glog.V(3).Infof("Adding endpoints: %v", endpoint.Name)
			lbc.AddSyncQueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			endpoint, isEndpoint := obj.(*v1.Endpoints)
			if !isEndpoint {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				endpoint, ok = deletedState.Obj.(*v1.Endpoints)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Endpoints object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing endpoints: %v", endpoint.Name)
			lbc.AddSyncQueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Endpoints %v changed, syncing", cur.(*v1.Endpoints).Name)
				lbc.AddSyncQueue(cur)
			}
		},
	}
}

// createIngressHandlers builds the handler funcs for ingresses
func createIngressHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ingress := obj.(*v1beta1.Ingress)
			if !lbc.IsNginxIngress(ingress) {
				glog.Infof("Ignoring Ingress %v based on Annotation %v", ingress.Name, ingressClassKey)
				return
			}
			glog.V(3).Infof("Adding Ingress: %v", ingress.Name)
			lbc.AddSyncQueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			ingress, isIng := obj.(*v1beta1.Ingress)
			if !isIng {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				ingress, ok = deletedState.Obj.(*v1beta1.Ingress)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Ingress object: %v", deletedState.Obj)
					return
				}
			}
			if !lbc.IsNginxIngress(ingress) {
				return
			}
			if isMinion(ingress) {
				master, err := lbc.FindMasterForMinion(ingress)
				if err != nil {
					glog.Infof("Ignoring Ingress %v(Minion): %v", ingress.Name, err)
					return
				}
				glog.V(3).Infof("Removing Ingress: %v(Minion) for %v(Master)", ingress.Name, master.Name)
				lbc.AddSyncQueue(master)
			} else {
				glog.V(3).Infof("Removing Ingress: %v", ingress.Name)
				lbc.AddSyncQueue(obj)
			}
		},
		UpdateFunc: func(old, current interface{}) {
			c := current.(*v1beta1.Ingress)
			o := old.(*v1beta1.Ingress)
			if !lbc.IsNginxIngress(c) {
				return
			}
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
			if err := lbc.ValidateSecret(secret); err != nil {
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
			if err := lbc.ValidateSecret(secret); err != nil {
				return
			}

			glog.V(3).Infof("Removing Secret: %v", secret.Name)
			lbc.AddSyncQueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			errOld := lbc.ValidateSecret(old.(*v1.Secret))
			errCur := lbc.ValidateSecret(cur.(*v1.Secret))
			if errOld != nil && errCur != nil {
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
// In the handlers below, we need to enqueue Ingress or other affected resources
// so that we can catch a change like a change of the port field of a service port (for such a change Kubernetes doesn't
// update the corresponding endpoints resource, that we monitor as well)
// or a change of the externalName field of an ExternalName service.
func createServiceHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)
			if lbc.IsExternalServiceForStatus(svc) {
				lbc.AddSyncQueue(svc)
				return
			}
			glog.V(3).Infof("Adding service: %v", svc.Name)
			lbc.EnqueueIngressForService(svc)

			if lbc.areCustomResourcesEnabled {
				lbc.EnqueueVirtualServersForService(svc)
				lbc.EnqueueTransportServerForService(svc)
			}
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
			if lbc.IsExternalServiceForStatus(svc) {
				lbc.AddSyncQueue(svc)
				return
			}

			glog.V(3).Infof("Removing service: %v", svc.Name)
			lbc.EnqueueIngressForService(svc)

			if lbc.areCustomResourcesEnabled {
				lbc.EnqueueVirtualServersForService(svc)
				lbc.EnqueueTransportServerForService(svc)
			}
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
					lbc.EnqueueIngressForService(curSvc)

					if lbc.areCustomResourcesEnabled {
						lbc.EnqueueVirtualServersForService(curSvc)
						lbc.EnqueueTransportServerForService(curSvc)
					}
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
			if !lbc.IsNginxIngress(vs.Spec.IngressClass) {
				glog.Infof("Ignoring VS %v based on class %v", vs.Name, vs.Spec.IngressClass)
				return
			}
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
			if !lbc.IsNginxIngress(vs.Spec.IngressClass) {
				glog.Infof("Ignoring VS %v based on class %v", vs.Name, vs.Spec.IngressClass)
				return
			}
			glog.V(3).Infof("Removing VirtualServer: %v", vs.Name)
			lbc.AddSyncQueue(vs)
		},
		UpdateFunc: func(old, cur interface{}) {
			curVs := cur.(*conf_v1.VirtualServer)
			oldVs := old.(*conf_v1.VirtualServer)
			if !lbc.IsNginxIngress(curVs.Spec.IngressClass) {
				glog.Infof("Ignoring VS %v based on class %v", curVs.Name, curVs.Spec.IngressClass)
				return
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
			if !lbc.IsNginxIngress(vsr.Spec.IngressClass) {
				glog.Infof("Ignoring VSR %v based on class %v", vsr.Name, vsr.Spec.IngressClass)
				return
			}
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
			if !lbc.IsNginxIngress(vsr.Spec.IngressClass) {
				glog.Infof("Ignoring VSR %v based on class %v", vsr.Name, vsr.Spec.IngressClass)
				return
			}
			glog.V(3).Infof("Removing VirtualServerRoute: %v", vsr.Name)
			lbc.AddSyncQueue(vsr)
		},
		UpdateFunc: func(old, cur interface{}) {
			curVsr := cur.(*conf_v1.VirtualServerRoute)
			oldVsr := old.(*conf_v1.VirtualServerRoute)
			if !lbc.IsNginxIngress(curVsr.Spec.IngressClass) {
				glog.Infof("Ignoring VSR %v based on class %v", curVsr.Name, curVsr.Spec.IngressClass)
				return
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
			gc := obj.(*conf_v1alpha1.GlobalConfiguration)
			glog.V(3).Infof("Adding GlobalConfiguration: %v", gc.Name)
			lbc.AddSyncQueue(gc)
		},
		DeleteFunc: func(obj interface{}) {
			gc, isGc := obj.(*conf_v1alpha1.GlobalConfiguration)
			if !isGc {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				gc, ok = deletedState.Obj.(*conf_v1alpha1.GlobalConfiguration)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-GlobalConfiguration object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing GlobalConfiguration: %v", gc.Name)
			lbc.AddSyncQueue(gc)
		},
		UpdateFunc: func(old, cur interface{}) {
			curGc := cur.(*conf_v1alpha1.GlobalConfiguration)
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
			ts := obj.(*conf_v1alpha1.TransportServer)
			glog.V(3).Infof("Adding TransportServer: %v", ts.Name)
			lbc.AddSyncQueue(ts)
		},
		DeleteFunc: func(obj interface{}) {
			ts, isTs := obj.(*conf_v1alpha1.TransportServer)
			if !isTs {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				ts, ok = deletedState.Obj.(*conf_v1alpha1.TransportServer)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-TransportServer object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing TransportServer: %v", ts.Name)
			lbc.AddSyncQueue(ts)
		},
		UpdateFunc: func(old, cur interface{}) {
			curTs := cur.(*conf_v1alpha1.TransportServer)
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("TransportServer %v changed, syncing", curTs.Name)
				lbc.AddSyncQueue(curTs)
			}
		},
	}
}

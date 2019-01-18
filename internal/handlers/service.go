package handlers

import (
	"reflect"
	"sort"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/controller"
	"k8s.io/client-go/tools/cache"

	api_v1 "k8s.io/api/core/v1"
)

// CreateServiceHandlers builds the handler funcs for services
func CreateServiceHandlers(lbc *controller.LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*api_v1.Service)
			if lbc.IsExternalServiceForStatus(svc) {
				lbc.AddSyncQueue(svc)
				return
			}
			glog.V(3).Infof("Adding service: %v", svc.Name)
			lbc.EnqueueIngressForService(svc)
		},
		DeleteFunc: func(obj interface{}) {
			svc, isSvc := obj.(*api_v1.Service)
			if !isSvc {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				svc, ok = deletedState.Obj.(*api_v1.Service)
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

		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				curSvc := cur.(*api_v1.Service)
				if lbc.IsExternalServiceForStatus(curSvc) {
					lbc.AddSyncQueue(curSvc)
					return
				}
				oldSvc := old.(*api_v1.Service)
				if hasServiceChanges(oldSvc, curSvc) {
					glog.V(3).Infof("Service %v changed, syncing", curSvc.Name)
					lbc.EnqueueIngressForService(curSvc)
				}
			}
		},
	}
}

type portSort []api_v1.ServicePort

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
func hasServiceChanges(oldSvc, curSvc *api_v1.Service) bool {
	if hasServicePortChanges(oldSvc.Spec.Ports, curSvc.Spec.Ports) {
		return true
	}
	if hasServiceExternalNameChanges(oldSvc, curSvc) {
		return true
	}
	return false
}

// hasServiceExternalNameChanges only compares Service.Spec.Externalname for Type ExternalName services.
func hasServiceExternalNameChanges(oldSvc, curSvc *api_v1.Service) bool {
	return curSvc.Spec.Type == api_v1.ServiceTypeExternalName && oldSvc.Spec.ExternalName != curSvc.Spec.ExternalName
}

// hasServicePortChanges only compares ServicePort.Name and .Port.
func hasServicePortChanges(oldServicePorts []api_v1.ServicePort, curServicePorts []api_v1.ServicePort) bool {
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

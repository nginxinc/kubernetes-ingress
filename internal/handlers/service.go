package handlers

import (
	"reflect"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/controller"
	"k8s.io/client-go/tools/cache"

	api_v1 "k8s.io/api/core/v1"
)

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
				svc := cur.(*api_v1.Service)
				if lbc.IsExternalServiceForStatus(svc) {
					lbc.AddSyncQueue(svc)
					return
				}
				glog.V(3).Infof("Service %v changed, syncing", svc.Name)
				lbc.EnqueueIngressForService(svc)
			}
		},
	}
}

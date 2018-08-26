package handlers

import (
	"reflect"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/controller"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func CreateEndpointHandlers(lbc *controller.LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoint := obj.(*api_v1.Endpoints)
			glog.V(3).Infof("Adding endpoints: %v", endpoint.Name)
			lbc.AddSyncQueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			endpoint, isEndpoint := obj.(*api_v1.Endpoints)
			if !isEndpoint {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				endpoint, ok = deletedState.Obj.(*api_v1.Endpoints)
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
				glog.V(3).Infof("Endpoints %v changed, syncing", cur.(*api_v1.Endpoints).Name)
				lbc.AddSyncQueue(cur)
			}
		},
	}
}

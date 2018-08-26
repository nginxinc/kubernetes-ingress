package handlers

import (
	"reflect"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/controller"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func CreateConfigMapHandlers(lbc *controller.LoadBalancerController, name string) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			configMap := obj.(*api_v1.ConfigMap)
			if configMap.Name == name {
				glog.V(3).Infof("Adding ConfigMap: %v", configMap.Name)
				lbc.AddSyncQueue(obj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			configMap, isConfigMap := obj.(*api_v1.ConfigMap)
			if !isConfigMap {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				configMap, ok = deletedState.Obj.(*api_v1.ConfigMap)
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
				configMap := cur.(*api_v1.ConfigMap)
				if configMap.Name == name {
					glog.V(3).Infof("ConfigMap %v changed, syncing", cur.(*api_v1.ConfigMap).Name)
					lbc.AddSyncQueue(cur)
				}
			}
		},
	}
}

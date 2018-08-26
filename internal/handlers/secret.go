package handlers

import (
	"reflect"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/controller"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func CreateSecretHandlers(lbc *controller.LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret := obj.(*api_v1.Secret)
			if err := lbc.ValidateSecret(secret); err != nil {
				return
			}
			nsname := secret.Namespace + "/" + secret.Name
			if nsname == lbc.GetDefaultServerSecret() {
				glog.V(3).Infof("Adding default server Secret: %v", secret.Name)
				lbc.AddSyncQueue(obj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			secret, isSecr := obj.(*api_v1.Secret)
			if !isSecr {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				secret, ok = deletedState.Obj.(*api_v1.Secret)
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
			errOld := lbc.ValidateSecret(old.(*api_v1.Secret))
			errCur := lbc.ValidateSecret(cur.(*api_v1.Secret))
			if errOld != nil && errCur != nil {
				return
			}

			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Secret %v changed, syncing", cur.(*api_v1.Secret).Name)
				lbc.AddSyncQueue(cur)
			}
		},
	}
}

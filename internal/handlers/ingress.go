package handlers

import (
	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/controller"
	"github.com/nginxinc/kubernetes-ingress/internal/utils"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

func CreateIngressHandlers(lbc *controller.LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ingress := obj.(*extensions.Ingress)
			if !lbc.IsNginxIngress(ingress) {
				glog.Infof("Ignoring Ingress %v based on Annotation %v", ingress.Name, lbc.GetIngressClassKey())
				return
			}
			glog.V(3).Infof("Adding Ingress: %v", ingress.Name)
			lbc.AddSyncQueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			ingress, isIng := obj.(*extensions.Ingress)
			if !isIng {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				ingress, ok = deletedState.Obj.(*extensions.Ingress)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Ingress object: %v", deletedState.Obj)
					return
				}
			}
			if !lbc.IsNginxIngress(ingress) {
				return
			}
			if utils.IsMinion(ingress) {
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
			c := current.(*extensions.Ingress)
			o := old.(*extensions.Ingress)
			if !lbc.IsNginxIngress(c) {
				return
			}
			if utils.HasChanges(o, c) {
				glog.V(3).Infof("Ingress %v changed, syncing", c.Name)
				lbc.AddSyncQueue(c)
			}
		},
	}
}

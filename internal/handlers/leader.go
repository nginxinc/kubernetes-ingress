package handlers

import (
	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/controller"
	"k8s.io/client-go/tools/leaderelection"
)

func CreateLeaderHandler(lbc *controller.LoadBalancerController) leaderelection.LeaderCallbacks {
	return leaderelection.LeaderCallbacks{
		OnStartedLeading: func(stop <-chan struct{}) {
			glog.V(3).Info("started leading, updating ingress status")
			ingresses, mergeableIngresses := lbc.GetManagedIngresses()
			err := lbc.UpdateManagedAndMergeableIngresses(ingresses, mergeableIngresses)
			if err != nil {
				glog.V(3).Infof("error updating status when starting leading: %v", err)
			}
		},
	}
}

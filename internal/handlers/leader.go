package handlers

import (
	"context"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/controller"
	"k8s.io/client-go/tools/leaderelection"
)

// CreateLeaderHandler builds the handler funcs for leader handling
func CreateLeaderHandler(lbc *controller.LoadBalancerController) leaderelection.LeaderCallbacks {
	return leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			glog.V(3).Info("started leading, updating ingress status")
			ingresses, mergeableIngresses := lbc.GetManagedIngresses()
			err := lbc.UpdateManagedAndMergeableIngresses(ingresses, mergeableIngresses)
			if err != nil {
				glog.V(3).Infof("error updating status when starting leading: %v", err)
			}
		},
	}
}

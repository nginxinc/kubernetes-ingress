package k8s

import (
	"context"
	"os"
	"time"

	"github.com/golang/glog"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
)

// newLeaderElector creates a new LeaderElection and returns the Elector.
func newLeaderElector(client kubernetes.Interface, callbacks leaderelection.LeaderCallbacks, namespace string, lockName string) (*leaderelection.LeaderElector, error) {
	podName := os.Getenv("POD_NAME")

	broadcaster := record.NewBroadcaster()
	hostname, _ := os.Hostname()

	source := v1.EventSource{Component: "nginx-ingress-leader-elector", Host: hostname}
	recorder := broadcaster.NewRecorder(scheme.Scheme, source)

	lc := resourcelock.ResourceLockConfig{
		Identity:      podName,
		EventRecorder: recorder,
	}

	leaseMeta := metav1.ObjectMeta{
		Namespace: namespace,
		Name:      lockName,
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta:  leaseMeta,
		Client:     client.CoordinationV1(),
		LockConfig: lc,
	}

	ttl := 30 * time.Second
	return leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:          lock,
			LeaseDuration: ttl,
			RenewDeadline: ttl / 2,
			RetryPeriod:   ttl / 4,
			Callbacks:     callbacks,
		})
}

// createLeaderHandler builds the handler funcs for leader handling
func createLeaderHandler(lbc *LoadBalancerController) leaderelection.LeaderCallbacks {
	return leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			glog.V(3).Info("started leading")
			// Closing this channel allows the leader to start the telemetry reporting process
			if lbc.telemetryChan != nil {
				close(lbc.telemetryChan)
			}
			if lbc.reportIngressStatus {
				ingresses := lbc.configuration.GetResourcesWithFilter(resourceFilter{Ingresses: true})

				glog.V(3).Infof("Updating status for %v Ingresses", len(ingresses))

				err := lbc.statusUpdater.UpdateExternalEndpointsForResources(ingresses)
				if err != nil {
					glog.V(3).Infof("error updating status when starting leading: %v", err)
				}
			}

			if lbc.areCustomResourcesEnabled {
				glog.V(3).Info("updating VirtualServer and VirtualServerRoutes status")

				err := lbc.updateVirtualServersStatusFromEvents()
				if err != nil {
					glog.V(3).Infof("error updating VirtualServers status when starting leading: %v", err)
				}

				err = lbc.updateVirtualServerRoutesStatusFromEvents()
				if err != nil {
					glog.V(3).Infof("error updating VirtualServerRoutes status when starting leading: %v", err)
				}

				err = lbc.updatePoliciesStatus()
				if err != nil {
					glog.V(3).Infof("error updating Policies status when starting leading: %v", err)
				}

				err = lbc.updateTransportServersStatusFromEvents()
				if err != nil {
					glog.V(3).Infof("error updating TransportServers status when starting leading: %v", err)
				}
			}
		},
		OnStoppedLeading: func() {
			glog.V(3).Info("stopped leading")
		},
	}
}

package k8s

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/spiffe/go-spiffe/workload"
)

type spiffeController struct {
	watcher *spiffeWatcher
	client  *workload.X509SVIDClient
}

// NewSpiffeController creates the spiffeWatcher and the Spiffe Workload API Client,
// returns an error if the client cannot connect to the Spire Agent.
func NewSpiffeController(sync func(*workload.X509SVIDs), spireAgentAddr string) (*spiffeController, error) {
	watcher := &spiffeWatcher{sync: sync}
	client, err := workload.NewX509SVIDClient(watcher, workload.WithAddr("unix://"+spireAgentAddr))
	if err != nil {
		return nil, fmt.Errorf("failed to create Spiffe Workload API Client: %w", err)
	}
	sc := &spiffeController{
		watcher: watcher,
		client:  client,
	}
	return sc, nil
}

// Start starts the Spiffe Workload API Client and waits for the Spiffe certs to be written to disk.
// If the certs are not available after 30 seconds an error is returned.
// On success, calls onStart function and kicks off the Spiffe Controller's run loop.
func (sc *spiffeController) Start(stopCh <-chan struct{}, onStart func()) error {
	glog.V(3).Info("Starting SPIFFE Workload API Client")
	err := sc.client.Start()
	if err != nil {
		return fmt.Errorf("failed to start Spiffe Workload API Client: %w", err)
	}
	timeout := time.After(30 * time.Second)
	duration := 100 * time.Millisecond
	for {
		if sc.watcher.synced {
			glog.V(3).Info("initial SPIFFE trust bundle written to disk")
			break
		}
		select {
		case <-timeout:
			return errors.New("timed out waiting for SPIFFE trust bundle")
		case <-stopCh:
			return sc.client.Stop()
		default:
			break
		}
		time.Sleep(duration)
	}
	onStart()
	go sc.Run(stopCh)
	return nil
}

// Run waits until a message is sent on the stop channel and stops the Spiffe Workload API Client.
func (sc *spiffeController) Run(stopCh <-chan struct{}) {
	<-stopCh
	err := sc.client.Stop()
	if err != nil {
		glog.Errorf("failed to stop Spiffe Workload API Client: %v", err)
	}
}

// spiffeWatcher is a sample implementation of the workload.X509SVIDWatcher interface
type spiffeWatcher struct {
	sync   func(*workload.X509SVIDs)
	synced bool
}

// UpdateX509SVIDs is run every time an SVID is updated
func (w *spiffeWatcher) UpdateX509SVIDs(svids *workload.X509SVIDs) {
	for _, svid := range svids.SVIDs {
		glog.V(3).Infof("SVID updated for spiffeID: %q", svid.SPIFFEID)
	}
	w.sync(svids)
	w.synced = true
}

// OnError is run when the client runs into an error
func (w *spiffeWatcher) OnError(err error) {
	if strings.Contains(err.Error(), "PermissionDenied") {
		glog.V(3).Infof("X509SVIDClient still waiting for certificates: %v", err)
		return
	}
	glog.Fatal(err)
}

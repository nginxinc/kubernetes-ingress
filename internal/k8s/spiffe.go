package k8s

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client wraps the workloadapi.Client
type Client interface {
	WatchX509Context(context.Context, workloadapi.X509ContextWatcher) error
	Close() error
}

// SpiffeCertFetcher fetches certs from the X509 SPIFFE Workload API.
type SpiffeCertFetcher struct {
	client     Client
	watcher    *spiffeWatcher
	watchErrCh chan error
}

// NewSpiffeCertFetcher creates the spiffeWatcher and the Spiffe Workload API Client,
// returns an error if the client cannot connect to the Spire Agent.
func NewSpiffeCertFetcher(sync func(*workloadapi.X509Context), spireAgentAddr string) (*SpiffeCertFetcher, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := workloadapi.New(ctx, workloadapi.WithAddr("unix://"+spireAgentAddr))
	if err != nil {
		return nil, fmt.Errorf("could not create SPIFFE Workload API Client: %w", err)
	}

	return &SpiffeCertFetcher{
		watchErrCh: make(chan error),
		client:     client,
		watcher:    &spiffeWatcher{sync: sync},
	}, nil
}

// Start starts the Spiffe Workload API Client and waits for the Spiffe certs to be written to disk.
// If the certs are not available after 30 seconds an error is returned.
// On success, calls onStart function and kicks off the SpiffeCertFetcher's run loop.
func (sc *SpiffeCertFetcher) Start(ctx context.Context, onStart func()) error {
	glog.V(3).Info("Starting SPIFFE Workload API Client")

	go func() {
		defer func() {
			if err := sc.client.Close(); err != nil && status.Code(err) != codes.Canceled {
				glog.V(3).Info("error closing SPIFFE Workload API Client: ", err)
			}
		}()
		err := sc.client.WatchX509Context(ctx, sc.watcher)
		if err != nil && status.Code(err) != codes.Canceled {
			sc.watchErrCh <- err
		}
	}()

	stopCh := ctx.Done()
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
		case err := <-sc.watchErrCh:
			return fmt.Errorf("error waiting for initial trust bundle: %w", err)
		case <-stopCh:
			return sc.client.Close()
		default:
		}
		time.Sleep(duration)
	}
	onStart()
	go sc.Run(stopCh)
	return nil
}

// Run waits until a message is sent on the stop channel and stops the Spiffe Workload API Client.
func (sc *SpiffeCertFetcher) Run(stopCh <-chan struct{}) {
	<-stopCh
	if err := sc.client.Close(); err != nil {
		glog.Errorf("failed to stop Spiffe Workload API Client: %v", err)
	}
}

// spiffeWatcher is a sample implementation of the workload.X509ContextWatcher interface
type spiffeWatcher struct {
	sync   func(*workloadapi.X509Context)
	synced bool
}

// OnX509ContextUpdate is called when a new X.509 Context is fetched from the SPIFFE Workload API.
func (w *spiffeWatcher) OnX509ContextUpdate(svidResponse *workloadapi.X509Context) {
	glog.V(3).Infof("SVID updated for for spiffeID: %q\n", svidResponse.DefaultSVID().ID)
	w.sync(svidResponse)
	w.synced = true
}

// OnX509WatchError is called when there is an error watching the X.509 Contexts from the SPIFFE Workload API.
func (w *spiffeWatcher) OnX509ContextWatchError(err error) {
	msg := "For more information check the logs of the Spire agents and server."
	switch status.Code(err) {
	case codes.Unavailable:
		glog.V(3).Infof("X509SVIDClient cannot connect to the Spire agent: %v. %s", err, msg)
	case codes.PermissionDenied:
		glog.V(3).Infof("X509SVIDClient still waiting for certificates: %v. %s", err, msg)
	case codes.Canceled:
		return
	default:
		glog.V(3).Infof("X509SVIDClient error: %v. %s", err, msg)
	}
}

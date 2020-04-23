package workload

import (
	"context"
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"sync"
	"time"

	"github.com/spiffe/go-spiffe/proto/spiffe/workload"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// X509SVIDs is an X.509 SVID response from the SPIFFE Workload API.
type X509SVIDs struct {
	// SVIDs is a list of X509SVID messages, each of which includes a single
	// SPIFFE Verifiable Identity Document, along with its private key and bundle.
	SVIDs []*X509SVID

	// CRL is a list of revoked certificates.
	// Unimplemented.
	CRL *pkix.CertificateList
}

// Default returns the default SVID (the first in the list).
//
// See the SPIFFE Workload API standard Section 5.3
// (https://github.com/spiffe/spiffe/blob/master/standards/SPIFFE_Workload_API.md#53-default-identity)
func (x *X509SVIDs) Default() *X509SVID {
	return x.SVIDs[0]
}

// SVID is an X.509 SPIFFE Verifiable Identity Document.
//
// See https://github.com/spiffe/spiffe/blob/master/standards/X509-SVID.md
type X509SVID struct {
	SPIFFEID                  string
	PrivateKey                crypto.Signer
	Certificates              []*x509.Certificate
	TrustBundle               []*x509.Certificate
	TrustBundlePool           *x509.CertPool
	FederatedTrustBundles     map[string][]*x509.Certificate
	FederatedTrustBundlePools map[string]*x509.CertPool
}

// X509SVIDWatcher is implemented by consumers who wish to be updated on SVID changes.
type X509SVIDWatcher interface {
	// UpdateX509SVIDs indicates to the Watcher that the SVID has been updated
	UpdateX509SVIDs(*X509SVIDs)

	// OnError indicates an error occurred.
	OnError(err error)
}

// X509SVIDClient interacts with the SPIFFE Workload API.
type X509SVIDClient struct {
	watcher      X509SVIDWatcher
	dialer       *Dialer
	wg           sync.WaitGroup
	ctx          context.Context
	cancelFn     func()
	backoff      *backoff
	stateManager *clientStateManager
}

// NewX509SVIDClient returns a new Workload API client for X.509 SVIDs.
func NewX509SVIDClient(watcher X509SVIDWatcher, opts ...DialOption) (*X509SVIDClient, error) {
	dialer, err := NewDialer(opts...)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &X509SVIDClient{
		dialer:       dialer,
		watcher:      watcher,
		ctx:          ctx,
		cancelFn:     cancel,
		backoff:      newBackoff(),
		stateManager: newClientStateManager(),
	}, nil
}

// Start starts the client.
//
// The client will always start, and users should rely on the watcher
// interface to receives updates on the client's status.
//
// It is an error to call Start() more than once. Calling Start() after
// Stop() is not supported.
func (c *X509SVIDClient) Start() error {
	if err := c.stateManager.StartIfStartable(); err != nil {
		return fmt.Errorf("spiffe/workload: %v", err)
	}
	c.wg.Add(1)
	go c.run()
	return nil
}

// Stop stops the client and waits for the watch loop to end.
func (c *X509SVIDClient) Stop() error {
	if err := c.stateManager.StopIfStoppable(); err != nil {
		return fmt.Errorf("spiffe/workload: %v", err)
	}
	c.cancelFn()
	c.wg.Wait()
	return nil
}

func (c *X509SVIDClient) run() {
	defer c.wg.Done()

	conn := c.newConn()
	if conn == nil {
		return
	}
	defer conn.Close()

	for {
		if done := c.watch(conn); done {
			return
		}
	}
}

// establishes a new persistent connection, returns nil if a connection can't be created
func (c *X509SVIDClient) newConn() *grpc.ClientConn {
	for {
		conn, err := c.dialer.DialContext(c.ctx)
		if err != nil {
			if done := c.handleError(err); done {
				return nil
			}
			continue
		}
		c.backoff.Reset()
		return conn
	}
}

// handles an error, applies backoff, and returns true if the context has been canceled
func (c *X509SVIDClient) handleError(err error) (done bool) {
	if status.Code(err) == codes.Canceled {
		return true
	}
	c.watcher.OnError(err)
	select {
	case <-time.After(c.backoff.Duration()):
		return false
	case <-c.ctx.Done():
		return true
	}
}

// creates single watch for the connection and returns whether we should stop watching
func (c *X509SVIDClient) watch(conn *grpc.ClientConn) bool {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()
	stream, err := c.newX509SVIDStream(ctx, conn)
	if err != nil {
		return c.handleError(err)
	}
	if err := c.handleX509SVIDStream(stream); err != nil {
		return c.handleError(err)
	}
	return false
}

func (c *X509SVIDClient) newX509SVIDStream(ctx context.Context, conn *grpc.ClientConn) (workload.SpiffeWorkloadAPI_FetchX509SVIDClient, error) {
	workloadClient := workload.NewSpiffeWorkloadAPIClient(conn)
	header := metadata.Pairs("workload.spiffe.io", "true")
	grpcCtx := metadata.NewOutgoingContext(ctx, header)
	return workloadClient.FetchX509SVID(grpcCtx, &workload.X509SVIDRequest{})
}

func (c *X509SVIDClient) handleX509SVIDStream(stream workload.SpiffeWorkloadAPI_FetchX509SVIDClient) error {
	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}
		svids, err := protoToX509SVIDs(resp)
		if err != nil {
			c.watcher.OnError(err)
			continue
		}
		c.backoff.Reset()
		c.watcher.UpdateX509SVIDs(svids)
	}
}

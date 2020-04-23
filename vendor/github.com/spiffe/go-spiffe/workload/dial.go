package workload

import (
	"context"
	"errors"

	"google.golang.org/grpc"
)

type dialConfig struct {
	addr string
	opts []grpc.DialOption
}

// DialOption is a function type used to configure a Dialer.
type DialOption func(*dialConfig)

// WithAddr returns a DialOption that sets the dial address to the given value.
func WithAddr(addr string) DialOption {
	return func(c *dialConfig) {
		c.addr = addr
	}
}

// WithGRPCOptions returns a DialOption that appends the given gRPC DialOptions.
func WithGRPCOptions(opts ...grpc.DialOption) DialOption {
	return func(c *dialConfig) {
		c.opts = append(c.opts, opts...)
	}
}

// Dialer type is used to create client gRPC connections.
type Dialer struct {
	target string
	opts   []grpc.DialOption
}

// NewDialer creates a Dialer configured according to the given DialOption list.
func NewDialer(opts ...DialOption) (*Dialer, error) {
	c := new(dialConfig)
	for _, opt := range opts {
		opt(c)
	}

	if c.addr == "" {
		var ok bool
		c.addr, ok = GetDefaultAddress()
		if !ok {
			return nil, errors.New("workload endpoint socket address is not configured")
		}
	}

	target, err := parseTargetFromAddr(c.addr)
	if err != nil {
		return nil, err
	}

	return &Dialer{
		target: target,
		opts:   c.opts,
	}, nil
}

// Dial calls DialContext using a background context.
func (d *Dialer) Dial() (*grpc.ClientConn, error) {
	return d.DialContext(context.Background())
}

// DialContext is a wrapper of grpc.DialContext that uses the target and dial
// options defined in Dialer.
func (d *Dialer) DialContext(ctx context.Context) (*grpc.ClientConn, error) {
	// append the insecure option since the workload endpoint is by
	// definition insecure.
	dialOpts := append([]grpc.DialOption{}, d.opts...)
	dialOpts = append(dialOpts, grpc.WithInsecure())

	return grpc.DialContext(ctx, d.target, dialOpts...)
}

// Dial creates a gRPC client connection using a background context and the
// given dial options.
func Dial(opts ...DialOption) (*grpc.ClientConn, error) {
	dialer, err := NewDialer(opts...)
	if err != nil {
		return nil, err
	}
	return dialer.Dial()
}

// DialContext creates a gRPC client connection using the given context and dial
// options.
func DialContext(ctx context.Context, opts ...DialOption) (*grpc.ClientConn, error) {
	dialer, err := NewDialer(opts...)
	if err != nil {
		return nil, err
	}
	return dialer.DialContext(ctx)
}

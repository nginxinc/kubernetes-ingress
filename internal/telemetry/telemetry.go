// Package telemetry provides functionality for collecting and exporting NIC telemetry data.
package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Export takes context and data and sends it to the Otel endpoint.
func Export(_ context.Context, _ TraceData) error {
	// Note: exporting functionality will be implemented in a separate module.
	return nil
}

// TraceData holds collected NIC telemetry data.
type TraceData struct {
	// Numer of VirtualServers
	VSCount int
	// Number of TransportServers
	TSCount int

	// TODO
	// Add more fields for NIC data points
}

// Option is a functional option used for configuring TraceReporter.
type Option func(*Collector) error

// WithTimePeriod configures reporting time on TraceReporter.
func WithTimePeriod(period string) Option {
	return func(c *Collector) error {
		d, err := time.ParseDuration(period)
		if err != nil {
			return err
		}
		c.Period = d
		return nil
	}
}

// Collector is NIC telemetry data collector.
type Collector struct {
	Period time.Duration

	mu   sync.Mutex
	Data TraceData
}

// NewCollector takes 0 or more options and creates a new TraceReporter.
// If no options are provided, NewReporter returns TraceReporter
// configured to gather data every 24h.
func NewCollector(opts ...Option) (*Collector, error) {
	c := Collector{
		Period: 24 * time.Hour,
		Data:   TraceData{},
	}
	for _, o := range opts {
		if err := o(&c); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

// BuildReport takes context and builds report from gathered telemetry data.
func (c *Collector) BuildReport(_ context.Context) error {
	glog.V(3).Info("Building telemetry report")
	dt := TraceData{}

	// TODO: Implement handling and logging errors for each collected data point

	c.mu.Lock()
	c.Data = dt

	glog.V(3).Infof("%+v", c.Data)
	c.mu.Unlock()
	return nil
}

// Collect runs data builder.
func (c *Collector) Collect(ctx context.Context) {
	glog.V(3).Info("Collecting telemetry data")
	if err := c.BuildReport(ctx); err != nil {
		glog.Errorf("Error exporting telemetry data: %v", err)
	}
}

// GetVSCount returns number of VirtualServers in watched namespaces.
//
// Note: this is a placeholder function.
func (c *Collector) GetVSCount() int {
	// Placeholder function
	return 0
}

// GetTSCount returns number of TransportServers in watched namespaces.
//
// Note: this is a placeholder function.
func (c *Collector) GetTSCount() int {
	// Placeholder function
	return 0
}

// Run starts running NIC Telemetry Collector.
//
// This is a placeholder for implementing collector runner.
func (c *Collector) Run(ctx context.Context) {
	wait.JitterUntilWithContext(ctx, c.Collect, c.Period, 0.1, true)
}

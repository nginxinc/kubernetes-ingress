// Package telemetry provides functionality for collecting and exporting NIC telemetry data.
package telemetry

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
)

// DiscardExporter is a temporary exporter
// for discarding collected telemetry data.
var DiscardExporter = Exporter{Endpoint: io.Discard}

// Exporter represents a temporary telemetry data exporter.
type Exporter struct {
	Endpoint io.Writer
}

// Export takes context and trace data and writes to the endpoint.
func (e *Exporter) Export(_ context.Context, td TraceData) error {
	// Note: exporting functionality will be implemented in a separate module.
	fmt.Fprintf(e.Endpoint, "%+v", td)
	return nil
}

// TraceData holds collected NIC telemetry data.
type TraceData struct {
	// Count of VirtualServers
	VSCount int
	// Count of TransportServers
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

// WithExporter configures telemetry collector to use given exporter.
//
// This may change in the future when we use exporter implemented
// in the external module.
func WithExporter(e Exporter) Option {
	return func(c *Collector) error {
		c.Exporter = e
		return nil
	}
}

// Collector is NIC telemetry data collector.
type Collector struct {
	Period time.Duration

	// Exporter is a temp exporter for exporting telemetry data.
	// The concrete implementation will be implemented in a separate module.
	Exporter Exporter
}

// NewCollector takes 0 or more options and creates a new TraceReporter.
// If no options are provided, NewReporter returns TraceReporter
// configured to gather data every 24h.
func NewCollector(opts ...Option) (*Collector, error) {
	c := Collector{
		Period:   24 * time.Hour,
		Exporter: DiscardExporter, // Use DiscardExporter until the real exporter is available.
	}
	for _, o := range opts {
		if err := o(&c); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

// BuildReport takes context and builds report from gathered telemetry data.
func (c *Collector) BuildReport(context.Context) (TraceData, error) {
	dt := TraceData{}

	// TODO: Implement handling and logging errors for each collected data point

	return dt, nil
}

// Collect collects and exports telemetry data.
// It exports data using provided exporter.
func (c *Collector) Collect(ctx context.Context) {
	glog.V(3).Info("Collecting telemetry data")
	traceData, err := c.BuildReport(ctx)
	if err != nil {
		glog.Errorf("Error collecting telemetry data: %v", err)
	}
	err = c.Exporter.Export(ctx, traceData)
	if err != nil {
		glog.Errorf("Error exporting telemetry data: %v", err)
	}
	glog.V(3).Infof("Exported telemetry data: %x", traceData)
}

// Start starts running NIC Telemetry Collector.
func (c *Collector) Start(ctx context.Context) {
	wait.JitterUntilWithContext(ctx, c.Collect, c.Period, 0.1, true)
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

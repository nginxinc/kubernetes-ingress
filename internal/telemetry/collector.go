// Package telemetry provides functionality for collecting and exporting NIC telemetry data.
package telemetry

import (
	"context"
	"time"

	k8s_nginx "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
)

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

	// Configuration for the collector.
	Config CollectorConfig
}

// CollectorConfig contains configuration options for a Collector
type CollectorConfig struct {
	// K8sClientReader is a kubernetes client.
	K8sClientReader kubernetes.Interface

	// CustomK8sClientReader is a kubernetes client for our CRDs.
	// CustomK8sClientReader v1.K8sV1Interface
	CustomK8sClientReader k8s_nginx.Interface

	// List of watched namespaces
	Namespaces []string
}

// NewCollector takes 0 or more options and creates a new TraceReporter.
// If no options are provided, NewReporter returns TraceReporter
// configured to gather data every 24h.
func NewCollector(cfg CollectorConfig, opts ...Option) (*Collector, error) {
	c := Collector{
		Period:   24 * time.Hour,
		Exporter: DiscardExporter, // Use DiscardExporter until the real exporter is available.
		Config:   cfg,
	}
	for _, o := range opts {
		if err := o(&c); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

// Start starts running NIC Telemetry Collector.
func (c *Collector) Start(ctx context.Context) {
	wait.JitterUntilWithContext(ctx, c.Collect, c.Period, 0.1, true)
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
	glog.V(3).Infof("Exported telemetry data: %+v", traceData)
}

// BuildReport takes context and builds report from gathered telemetry data.
func (c *Collector) BuildReport(ctx context.Context) (TraceData, error) {
	dt := TraceData{}
	var err error

	for _, namespace := range c.Config.Namespaces {
		dt.VirtualServers += c.GetVirtualServerCount(ctx, namespace)
		dt.TransportServers += c.GetTransportServerCount(ctx, namespace)
	}

	return dt, err
}

// SetConfig is used to overwrite the existing config for the Collector.
func (c *Collector) SetConfig(cfg CollectorConfig) {
	c.Config = cfg
}

// GetVirtualServerCount returns the number of VirtualServers in a namespace.
func (c *Collector) GetVirtualServerCount(ctx context.Context, namespace string) int {
	count, err := c.Config.CustomK8sClientReader.
		K8sV1().
		VirtualServers(namespace).
		List(ctx, meta_v1.ListOptions{})
	if err != nil {
		glog.Errorf("unable to list VirtualServers in namespace %s Error: %x", namespace, err)
	}
	return len(count.Items)
}

// GetTransportServerCount returns the number of TransportServers in a namespace.
func (c *Collector) GetTransportServerCount(ctx context.Context, namespace string) int {
	count, err := c.Config.CustomK8sClientReader.
		K8sV1().
		TransportServers(namespace).
		List(ctx, meta_v1.ListOptions{})
	if err != nil {
		glog.Errorf("unable to list TransportServers in namespace %s Error: %x", namespace, err)
	}
	return len(count.Items)
}

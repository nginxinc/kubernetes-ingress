// Package telemetry provides functionality for collecting and exporting NIC telemetry data.
package telemetry

import (
	"context"
	"io"
	"runtime"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"

	k8s_nginx "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"
)

// Option is a functional option used for configuring TraceReporter.
type Option func(*Collector) error

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
	// Note: May not need this client.
	CustomK8sClientReader k8s_nginx.Interface

	// Period to collect telemetry
	Period time.Duration

	Configurator *configs.Configurator

	// Version represents NIC version.
	Version string
}

// NewCollector takes 0 or more options and creates a new TraceReporter.
// If no options are provided, NewReporter returns TraceReporter
// configured to gather data every 24h.
func NewCollector(cfg CollectorConfig, opts ...Option) (*Collector, error) {
	c := Collector{
		Exporter: &StdoutExporter{Endpoint: io.Discard},
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
	wait.JitterUntilWithContext(ctx, c.Collect, c.Config.Period, 0.1, true)
}

// Collect collects and exports telemetry data.
// It exports data using provided exporter.
func (c *Collector) Collect(ctx context.Context) {
	glog.V(3).Info("Collecting telemetry data")
	data, err := c.BuildReport(ctx)
	if err != nil {
		glog.Errorf("Error collecting telemetry data: %v", err)
	}
	err = c.Exporter.Export(ctx, data)
	if err != nil {
		glog.Errorf("Error exporting telemetry data: %v", err)
	}
	glog.V(3).Infof("Exported telemetry data: %+v", data)
}

// BuildReport takes context and builds report from gathered telemetry data.
func (c *Collector) BuildReport(ctx context.Context) (Data, error) {
	pm := ProjectMeta{
		Name:    "NIC",
		Version: c.Config.Version,
	}
	d := Data{
		ProjectMeta: pm,
		Arch:        runtime.GOARCH,
	}

	var err error

	if c.Config.Configurator != nil {
		vsCount, vsrCount := c.Config.Configurator.GetVirtualServerCounts()
		d.VirtualServers, d.VirtualServerRoutes = int64(vsCount), int64(vsrCount)
		d.TransportServers = int64(c.Config.Configurator.GetTransportServerCounts())
	}

	if d.NodeCount, err = c.NodeCount(ctx); err != nil {
		glog.Errorf("Error collecting telemetry data: Nodes: %v", err)
	}

	if d.ClusterID, err = c.ClusterID(ctx); err != nil {
		glog.Errorf("Error collecting telemetry data: ClusterID: %v", err)
	}

	if d.K8sVersion, err = c.K8sVersion(); err != nil {
		glog.Errorf("Error collecting telemetry data: K8s Version: %v", err)
	}
	return d, err
}

package telemetry

import (
	"context"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	jitterFactor = 0.1
	sliding      = true
)

type Reporter interface {
	Start(ctx context.Context)
}

type TraceTelemetryReporterConfig struct {
	Data            Data
	Exporter        Exporter
	ReportingPeriod time.Duration
}

type TraceTelemetryReporter struct {
	config TraceTelemetryReporterConfig
}

func NewTelemetryReporter(config TraceTelemetryReporterConfig) *TraceTelemetryReporter {
	return &TraceTelemetryReporter{
		config: config,
	}
}

func (t *TraceTelemetryReporter) Start(ctx context.Context) {
	wait.JitterUntilWithContext(ctx, t.report, t.config.ReportingPeriod, jitterFactor, sliding)
}

func (t *TraceTelemetryReporter) report(ctx context.Context) {
	glog.V(3).Infof("Collecting Telemetry Data")
	// Gather data here
	t.setVirtualServerCount()
	t.setTransportServerCount()

	if err := t.config.Exporter.Export(ctx, t.config.Data); err != nil {
		glog.Errorf("Error exporting telemetry data: %v", err)
	}
}

func (t *TraceTelemetryReporter) setVirtualServerCount() {
	// Placeholder function
}

func (t *TraceTelemetryReporter) setTransportServerCount() {
	// Placeholder function
}

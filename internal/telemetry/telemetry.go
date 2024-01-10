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

// Reporter is an interface that represents a telemetry style reporter
type Reporter interface {
	Start(ctx context.Context)
}

// TraceTelemetryReporterConfig contains configuration data for the Telemetry Reporter
type TraceTelemetryReporterConfig struct {
	Data            Data
	Exporter        Exporter
	ReportingPeriod time.Duration
}

// TraceTelemetryReporter reports telemety data that will be exported as a trace
type TraceTelemetryReporter struct {
	config TraceTelemetryReporterConfig
}

// NewTelemetryReporter creates a new TraceTelemetryReporter
func NewTelemetryReporter(config TraceTelemetryReporterConfig) *TraceTelemetryReporter {
	return &TraceTelemetryReporter{
		config: config,
	}
}

// Start starts the telemetry reporting job
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

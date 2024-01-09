package telemetry

import (
	"context"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

const (
	jitterFactor = 0.1  // If the period is 10 seconds, the jitter will be up to 1 second.
	sliding      = true // The period with jitter will be calculated after each report() call.
)

type Reporter interface {
	Start(ctx context.Context)
}

type TraceTelemetryReporterConfig struct {
	Data     Data
	Exporter Exporter
	Period   time.Duration
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
	glog.V(1).Info("Starting Telemetry Job...")
	wait.JitterUntilWithContext(ctx, t.report, t.config.Period, jitterFactor, sliding)
	glog.V(1).Info("Stopping Telemetry Job...")
}

func (t *TraceTelemetryReporter) report(ctx context.Context) {
	// Gather data here
	t.setProductName()
	t.setProductVersion()

	t.config.Exporter.Export(ctx, t.config.Data)
}

func (t *TraceTelemetryReporter) setProductVersion() {
	// Placeholder function
}

func (t *TraceTelemetryReporter) setProductName() {
	// Placeholder function
}

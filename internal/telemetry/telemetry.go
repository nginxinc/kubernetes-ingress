package telemetry

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

type Data struct {
}

type Exporter interface {
	Export(data Data)
}

type StdOutExporter struct {
}

func (s *StdOutExporter) Export(data Data) {
	fmt.Printf("Exporting data %v", data)
}

type Reporter interface {
	report(ctx context.Context)
	Start(ctx context.Context)
}

type TraceTelemetryReporter struct {
	exporter Exporter
	period   time.Duration
}

func NewTelemetryReporter(reportingPeriod time.Duration, exporter Exporter) *TraceTelemetryReporter {
	return &TraceTelemetryReporter{
		exporter: exporter,
		period:   reportingPeriod,
	}
}

func (t *TraceTelemetryReporter) Start(ctx context.Context) {
	glog.V(1).Info("Starting Telemetry Job...")
	wait.UntilWithContext(ctx, t.report, t.period)
	glog.V(1).Info("Stopping Telemetry Job...")
}

// ctx is blank for now during POC.
func (t *TraceTelemetryReporter) report(_ context.Context) {
	t.exporter.Export(Data{})

	glog.V(1).Info("Data exported...")
}

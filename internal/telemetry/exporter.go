package telemetry

import (
	"context"

	"github.com/golang/glog"
)

// Data represents the telemetry data that will be exported
type Data struct{}

// Exporter defines an interface for telemetry exporters
type Exporter interface {
	Export(ctx context.Context, data Data) error
}

// LogExporter is an exporter that will log out exported data
type LogExporter struct{}

// NewLogExporter creates a new logging exporter
func NewLogExporter() *LogExporter {
	return &LogExporter{}
}

// Export will send exported data level 3 logs
func (s *LogExporter) Export(_ context.Context, data Data) error {
	glog.V(3).Infof("Exporting data %v", data)
	return nil
}

package telemetry

import (
	"context"
	"fmt"
	"github.com/nginxinc/telemetry-exporter/pkg/telemetry"
	"io"
)

// Exporter interface for exporters.
type Exporter interface {
	Export(ctx context.Context, data telemetry.Exportable) error
}

// StdoutExporter represents a temporary telemetry data exporter.
type StdoutExporter struct {
	Endpoint io.Writer
}

// Export takes context and trace data and writes to the endpoint.
func (e *StdoutExporter) Export(_ context.Context, data telemetry.Exportable) error {
	fmt.Fprintf(e.Endpoint, "%+v", data)
	return nil
}

// Data holds collected telemetry data.
type Data struct {
	telemetry.Data
	NICResourceCounts
}

// NICResourceCounts holds a count of NIC specific resource.
type NICResourceCounts struct {
	VirtualServers      int64
	VirtualServerRoutes int64
	TransportServers    int64
}

package telemetry

import (
	"context"
	"fmt"
	"io"

	"go.opentelemetry.io/otel/attribute"
)

// Exporter interface for exporters.
type Exporter interface {
	// TODO Change Data to Exportable.
	Export(ctx context.Context, data Data) error
}

// StdoutExporter represents a temporary telemetry data exporter.
type StdoutExporter struct {
	Endpoint io.Writer
}

// Export takes context and trace data and writes to the endpoint.
func (e *StdoutExporter) Export(_ context.Context, data Data) error {
	fmt.Fprintf(e.Endpoint, "%+v", data)
	return nil
}

// Data holds collected telemetry data.
type Data struct {
	ProjectMeta
	NICResourceCounts
	NodeCount  int64
	ClusterID  string
	K8sVersion string
	Arch       string
}

// ProjectMeta holds metadata for the project.
type ProjectMeta struct {
	Name    string
	Version string
}

// NICResourceCounts holds a count of NIC specific resource.
type NICResourceCounts struct {
	VirtualServers      int64
	VirtualServerRoutes int64
	TransportServers    int64
}

// Attributes is a placeholder function.
// This ensures that Data is of type Exportable
func (d *Data) Attributes() []attribute.KeyValue {
	return nil
}

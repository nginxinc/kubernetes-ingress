package telemetry

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"io"
)

type Exporter interface {
	//TODO Change Data to Exportable.
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
	ProjectMeta       ProjectMeta
	NICResourceCounts NICResourceCounts
}

type ProjectMeta struct {
	Name    string
	Version string
}

type NICResourceCounts struct {
	VirtualServers      int
	VirtualServerRoutes int
	TransportServers    int
}

// Attributes is a placeholder function.
// This ensures that Data is of type Exportable
func (d *Data) Attributes() []attribute.KeyValue {
	return nil
}

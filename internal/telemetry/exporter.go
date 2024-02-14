package telemetry

import (
	"context"
	"fmt"
	"io"
)

// DiscardExporter is a temporary exporter
// for discarding collected telemetry data.
var DiscardExporter = Exporter{Endpoint: io.Discard}

// Exporter represents a temporary telemetry data exporter.
type Exporter struct {
	Endpoint io.Writer
}

// Export takes context and trace data and writes to the endpoint.
func (e *Exporter) Export(_ context.Context, td TraceData) error {
	// Note: exporting functionality will be implemented in a separate module.
	fmt.Fprintf(e.Endpoint, "%+v", td)
	return nil
}

// TraceData holds collected telemetry data.
type TraceData struct {
	VirtualServers int

	VirtualServerRoutes int

	TransportServers int
}

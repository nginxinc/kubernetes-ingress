package telemetry

import (
	"context"
	"fmt"
	"io"

	tel "github.com/nginxinc/telemetry-exporter/pkg/telemetry"
)

// Exporter interface for exporters.
type Exporter interface {
	Export(ctx context.Context, data tel.Exportable) error
}

// StdoutExporter represents a temporary telemetry data exporter.
type StdoutExporter struct {
	Endpoint io.Writer
}

// Export takes context and trace data and writes to the endpoint.
func (e *StdoutExporter) Export(_ context.Context, data tel.Exportable) error {
	fmt.Fprintf(e.Endpoint, "%+v", data)
	return nil
}

// Data holds collected telemetry data.
//
//go:generate go run -tags=generator github.com/nginxinc/telemetry-exporter/cmd/generator -type Data -scheme -scheme-protocol=NICProductTelemetry -scheme-df-datatype=nic-product-telemetry -scheme-namespace=ingress.nginx.com
type Data struct {
	tel.Data
	NICResourceCounts
}

// NICResourceCounts holds a count of NIC specific resource.
//
//go:generate go run -tags=generator github.com/nginxinc/telemetry-exporter/cmd/generator -type NICResourceCounts
type NICResourceCounts struct {
	// VirtualServer is the number of VirtualServer managed by the Ingress Controller.
	VirtualServers int64
	// VirtualServerRoutes is the number of VirtualServerRoutes managed by the Ingress Controller.
	VirtualServerRoutes int64
	// TransportServers is the number of TransportServers managed by the Ingress Controller.
	TransportServers int64
}

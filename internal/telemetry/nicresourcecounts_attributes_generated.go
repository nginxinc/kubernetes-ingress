package telemetry

/*
This is a generated file. DO NOT EDIT.
*/

import (
	"go.opentelemetry.io/otel/attribute"

	ngxTelemetry "github.com/nginxinc/telemetry-exporter/pkg/telemetry"
)

func (d *NICResourceCounts) Attributes() []attribute.KeyValue {
	var attrs []attribute.KeyValue

	attrs = append(attrs, attribute.Int64("VirtualServers", d.VirtualServers))
	attrs = append(attrs, attribute.Int64("VirtualServerRoutes", d.VirtualServerRoutes))
	attrs = append(attrs, attribute.Int64("TransportServers", d.TransportServers))
	attrs = append(attrs, attribute.Int64("Replicas", d.Replicas))

	return attrs
}

var _ ngxTelemetry.Exportable = (*NICResourceCounts)(nil)

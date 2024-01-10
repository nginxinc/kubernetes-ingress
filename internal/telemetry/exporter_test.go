package telemetry

import (
	"context"
	"testing"
)

func TestExportData(t *testing.T) {
	t.Parallel()

	exporter := NewStdOutExporter()

	err := exporter.Export(context.Background(), Data{})

	if err != nil {
		t.Fatalf("Expeceted no error, but got %s", err.Error())
	}
}

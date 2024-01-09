package telemetry

import (
	"context"
	"testing"
)
import . "github.com/onsi/gomega"

func TestExportData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	exporter := NewStdOutExporter()

	err := exporter.Export(context.Background(), Data{})

	g.Expect(err).To(BeNil())
}

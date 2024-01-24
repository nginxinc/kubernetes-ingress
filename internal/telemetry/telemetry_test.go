package telemetry_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/telemetry"
)

func TestCreateNewDefaultCollector(t *testing.T) {
	t.Parallel()

	c, err := telemetry.NewCollector()
	if err != nil {
		t.Fatal(err)
	}

	want := 24.0
	got := c.Period.Hours()

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCreateNewCollectorWithCustomReportingPeriod(t *testing.T) {
	t.Parallel()

	c, err := telemetry.NewCollector(telemetry.WithTimePeriod("4h"))
	if err != nil {
		t.Fatal(err)
	}

	want := 4.0
	got := c.Period.Hours()

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCreateNewCollectorWithCustomExporter(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := telemetry.Exporter{Endpoint: buf}

	c, err := telemetry.NewCollector(telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	want := "{VSCount:0 TSCount:0}"
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

package telemetry_test

import (
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

	wantData := telemetry.TraceData{}
	gotData := c.Data

	if !cmp.Equal(wantData, gotData) {
		t.Error(cmp.Diff(wantData, gotData))
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

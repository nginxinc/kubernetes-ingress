package telemetry

import (
	"context"
	"reflect"
	"testing"
)

type MockTelemetryReport struct {
	data Data
}

func (m *MockTelemetryReport) Start(_ context.Context) {
	m.data = Data{}
}

func TestCollectData(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mtr := &MockTelemetryReport{
		Data{},
	}
	expectedData := Data{}
	mtr.Start(ctx)

	if !reflect.DeepEqual(mtr.data, expectedData) {
		t.Fatalf("expected %v, but got %v", expectedData, mtr.data)
	}
}

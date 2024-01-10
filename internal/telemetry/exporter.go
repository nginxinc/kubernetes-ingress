package telemetry

import (
	"context"
	"github.com/golang/glog"
)

type Data struct {
}

type Exporter interface {
	Export(ctx context.Context, data Data) error
}

type StdOutExporter struct {
}

func NewStdOutExporter() *StdOutExporter {
	return &StdOutExporter{}
}

func (s *StdOutExporter) Export(_ context.Context, data Data) error {
	glog.V(3).Infof("Exporting data %v", data)
	return nil
}

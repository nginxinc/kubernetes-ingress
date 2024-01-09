package telemetry

import (
	"context"
	"github.com/golang/glog"
)

type Data struct {
}

type Exporter interface {
	Export(ctx context.Context, data Data)
}

type StdOutExporter struct {
}

func NewStdOutExporter() *StdOutExporter {
	return &StdOutExporter{}
}

func (s *StdOutExporter) Export(_ context.Context, data Data) error {
	glog.V(1).Infof("Exporting data %v", data)
	return nil
}

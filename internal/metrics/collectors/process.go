package collectors

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ProcessCollector is an interface for the metrics of NGINX Processes
type ProcessCollector interface {
	SetWorkerProcessCount(curr string, wp int)
	Register(register *prometheus.Registry) error
}

// ProcessMetricsCollector implements prometheus.Collector interface
type ProcessMetricsCollector struct {
	prevConfVer string
	// metrics
	processTotal *prometheus.GaugeVec
}

// NewProcessMetricCollector creates a new ProcessMetricCollector
func NewProcessMetricCollector(constLabels map[string]string) *ProcessMetricsCollector {
	labelNames := []string{"oldGeneration", "currentGeneration"}
	pt := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "controller_nginx_worker_processes_total",
			Namespace:   metricsNamespace,
			Help:        "Number of NGINX worker processes",
			ConstLabels: constLabels,
		},
		labelNames,
	)

	return &ProcessMetricsCollector{
		prevConfVer:  "0",
		processTotal: pt,
	}
}

// SetWorkerProcessCount sets the number of NGINX worker processes
func (pc *ProcessMetricsCollector) SetWorkerProcessCount(currentConfigVersion string, workerProcesses int) {
	pc.processTotal.WithLabelValues(pc.prevConfVer, currentConfigVersion).Set(float64(workerProcesses))

	pc.prevConfVer = currentConfigVersion
}

// Describe implements prometheus.Collector interface Describe method
func (pc *ProcessMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	pc.processTotal.Describe(ch)
}

// Collect implements prometheus.Collector interface Collect method
func (pc *ProcessMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	pc.processTotal.Collect(ch)
}

// Register registers all the metrics of the collector
func (pc *ProcessMetricsCollector) Register(registry *prometheus.Registry) error {
	return registry.Register(pc)
}

// ProcessFakeCollector us a fake collector that will implement ProcessCollector interface
type ProcessFakeCollector struct{}

// NewProcessFakeCollector creates a faje collector that will implement ProcessCollector interface
func NewProcessFakeCollector() *ProcessFakeCollector {
	return &ProcessFakeCollector{}
}

// Register implements a fake Register
func (pc *ProcessFakeCollector) Register(registry *prometheus.Registry) error { return nil }

// SetWorkerProcessCount implements a fake SetWorkerProcessCount
func (pc *ProcessFakeCollector) SetWorkerProcessCount(current string, wp int) {}

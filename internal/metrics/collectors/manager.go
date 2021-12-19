package collectors

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ManagerCollector is an interface for the metrics of the Nginx Manager
type ManagerCollector interface {
	IncNginxReloadCount(isEndPointUpdate bool)
	IncNginxReloadErrors()
	UpdateLastReloadTime(ms time.Duration)
	Register(registry *prometheus.Registry) error
}

// LocalManagerMetricsCollector implements NginxManagerCollector interface and prometheus.Collector interface
type LocalManagerMetricsCollector struct {
	// Metrics
	reloadsTotal     *prometheus.CounterVec
	reloadsError     prometheus.Counter
	lastReloadStatus prometheus.Gauge
	lastReloadTime   prometheus.Gauge
}

// NewLocalManagerMetricsCollector creates a new LocalManagerMetricsCollector
func NewLocalManagerMetricsCollector(constLabels map[string]string) *LocalManagerMetricsCollector {
	nc := &LocalManagerMetricsCollector{
		reloadsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "nginx_reloads_total",
				Namespace:   metricsNamespace,
				Help:        "Number of successful NGINX reloads",
				ConstLabels: constLabels,
			},
			[]string{"reason"},
		),
		reloadsError: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name:        "nginx_reload_errors_total",
				Namespace:   metricsNamespace,
				Help:        "Number of unsuccessful NGINX reloads",
				ConstLabels: constLabels,
			},
		),
		lastReloadStatus: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "nginx_last_reload_status",
				Namespace:   metricsNamespace,
				Help:        "Status of the last NGINX reload",
				ConstLabels: constLabels,
			},
		),
		lastReloadTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name:        "nginx_last_reload_milliseconds",
				Namespace:   metricsNamespace,
				Help:        "Duration in milliseconds of the last NGINX reload",
				ConstLabels: constLabels,
			},
		),
	}
	nc.reloadsTotal.WithLabelValues("other")
	nc.reloadsTotal.WithLabelValues("endpoints")
	return nc
}

// IncNginxReloadCount increments the counter of successful NGINX reloads and sets the last reload status to true
func (nc *LocalManagerMetricsCollector) IncNginxReloadCount(isEndPointUpdate bool) {
	var label string
	if isEndPointUpdate {
		label = "endpoints"
	} else {
		label = "other"
	}
	nc.reloadsTotal.WithLabelValues(label).Inc()
	nc.updateLastReloadStatus(true)
}

// IncNginxReloadErrors increments the counter of NGINX reload errors and sets the last reload status to false
func (nc *LocalManagerMetricsCollector) IncNginxReloadErrors() {
	nc.reloadsError.Inc()
	nc.updateLastReloadStatus(false)
}

// updateLastReloadStatus updates the last NGINX reload status metric
func (nc *LocalManagerMetricsCollector) updateLastReloadStatus(up bool) {
	var status float64
	if up {
		status = 1.0
	}
	nc.lastReloadStatus.Set(status)
}

// UpdateLastReloadTime updates the last NGINX reload time
func (nc *LocalManagerMetricsCollector) UpdateLastReloadTime(duration time.Duration) {
	nc.lastReloadTime.Set(float64(duration / time.Millisecond))
}

// Describe implements prometheus.Collector interface Describe method
func (nc *LocalManagerMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	nc.reloadsTotal.Describe(ch)
	nc.reloadsError.Describe(ch)
	nc.lastReloadStatus.Describe(ch)
	nc.lastReloadTime.Describe(ch)
}

// Collect implements the prometheus.Collector interface Collect method
func (nc *LocalManagerMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	nc.reloadsTotal.Collect(ch)
	nc.reloadsError.Collect(ch)
	nc.lastReloadStatus.Collect(ch)
	nc.lastReloadTime.Collect(ch)
}

// Register registers all the metrics of the collector
func (nc *LocalManagerMetricsCollector) Register(registry *prometheus.Registry) error {
	return registry.Register(nc)
}

// ManagerFakeCollector is a fake collector that will implement ManagerCollector interface
type ManagerFakeCollector struct{}

// NewManagerFakeCollector creates a fake collector that implements ManagerCollector interface
func NewManagerFakeCollector() *ManagerFakeCollector {
	return &ManagerFakeCollector{}
}

// Register implements a fake Register
func (nc *ManagerFakeCollector) Register(_ *prometheus.Registry) error { return nil }

// IncNginxReloadCount implements a fake IncNginxReloadCount
func (nc *ManagerFakeCollector) IncNginxReloadCount(_ bool) {}

// IncNginxReloadErrors implements a fake IncNginxReloadErrors
func (nc *ManagerFakeCollector) IncNginxReloadErrors() {}

// UpdateLastReloadTime implements a fake UpdateLastReloadTime
func (nc *ManagerFakeCollector) UpdateLastReloadTime(_ time.Duration) {}

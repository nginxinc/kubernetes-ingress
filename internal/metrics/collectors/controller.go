package collectors

import "github.com/prometheus/client_golang/prometheus"

var labelNamesController = []string{"type"}

// ControllerCollector is an interface for the metrics of the Controller
type ControllerCollector interface {
	SetIngress(ingressType string, count int)
	SetVirtualServer(count int)
	SetVirtualServerRoute(count int)
	Register(registry *prometheus.Registry) error
}

// ControllerMetricsCollector implements the ControllerCollector interface and prometheus.Collector interface
type ControllerMetricsCollector struct {
	ingressesTotal           *prometheus.GaugeVec
	virtualServersTotal      prometheus.Gauge
	virtualServerRoutesTotal prometheus.Gauge
}

// NewControllerMetricsCollector creates a new ControllerMetricsCollector
func NewControllerMetricsCollector() *ControllerMetricsCollector {
	ingResTotal := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "ingress_resources_total",
			Namespace: metricsNamespace,
			Help:      "Number of handled ingress resources",
		},
		labelNamesController,
	)

	vsResTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "virtualserver_resources_total",
			Namespace: metricsNamespace,
			Help:      "Number of handled VirtualServer resources",
		},
	)

	vsrResTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "virtualserverroute_resources_total",
			Namespace: metricsNamespace,
			Help:      "Number of handled VirtualServerRoute resources",
		},
	)

	return &ControllerMetricsCollector{
		ingressesTotal:           ingResTotal,
		virtualServersTotal:      vsResTotal,
		virtualServerRoutesTotal: vsrResTotal,
	}
}

// SetIngress sets the value of the ingress resources gauge for a given type
func (cc *ControllerMetricsCollector) SetIngress(ingressType string, count int) {
	cc.ingressesTotal.WithLabelValues(ingressType).Set(float64(count))
}

// SetVirtualServer sets the value of the VirtualServer resources gauge
func (cc *ControllerMetricsCollector) SetVirtualServer(count int) {
	cc.virtualServersTotal.Set(float64(count))
}

// SetVirtualServerRoute sets the value of the VirtualServerRoute resources gauge
func (cc *ControllerMetricsCollector) SetVirtualServerRoute(count int) {
	cc.virtualServerRoutesTotal.Set(float64(count))
}

// Describe implements prometheus.Collector interface Describe method
func (cc *ControllerMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	cc.ingressesTotal.Describe(ch)
	cc.virtualServersTotal.Describe(ch)
	cc.virtualServerRoutesTotal.Describe(ch)
}

// Collect implements the prometheus.Collector interface Collect method
func (cc *ControllerMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	cc.ingressesTotal.Collect(ch)
	cc.virtualServersTotal.Collect(ch)
	cc.virtualServerRoutesTotal.Collect(ch)
}

// Register registers all the metrics of the collector
func (cc *ControllerMetricsCollector) Register(registry *prometheus.Registry) error {
	return registry.Register(cc)
}

// ControllerFakeCollector is a fake collector that implements the ControllerCollector interface
type ControllerFakeCollector struct{}

// NewControllerFakeCollector creates a fake collector that implements the ControllerCollector interface
func NewControllerFakeCollector() *ControllerFakeCollector {
	return &ControllerFakeCollector{}
}

// Register implements a fake Register
func (cc *ControllerFakeCollector) Register(registry *prometheus.Registry) error { return nil }

// SetIngress implements a fake SetIngress
func (cc *ControllerFakeCollector) SetIngress(ingressType string, count int) {}

// SetVirtualServer implements a fake SetVirtualServer
func (cc *ControllerFakeCollector) SetVirtualServer(count int) {}

// SetVirtualServerRoute implements a fake SetVirtualServerRoute
func (cc *ControllerFakeCollector) SetVirtualServerRoute(count int) {}

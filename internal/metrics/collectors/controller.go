package collectors

import "github.com/prometheus/client_golang/prometheus"

var labelNamesController = []string{"type"}

// ControllerCollector is an interface for the metrics of the Controller
type ControllerCollector interface {
	SetIngresses(ingressType string, count int)
	SetVirtualServers(count int)
	SetVirtualServerRoutes(count int)
	Register(registry *prometheus.Registry) error
}

// ControllerMetricsCollector implements the ControllerCollector interface and prometheus.Collector interface
type ControllerMetricsCollector struct {
	crdsEnabled              bool
	ingressesTotal           *prometheus.GaugeVec
	virtualServersTotal      prometheus.Gauge
	virtualServerRoutesTotal prometheus.Gauge
}

// NewControllerMetricsCollector creates a new ControllerMetricsCollector
func NewControllerMetricsCollector(crdsEnabled bool, constLabels map[string]string) *ControllerMetricsCollector {
	ingResTotal := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "ingress_resources_total",
			Namespace:   metricsNamespace,
			Help:        "Number of handled ingress resources",
			ConstLabels: constLabels,
		},
		labelNamesController,
	)

	if !crdsEnabled {
		return &ControllerMetricsCollector{ingressesTotal: ingResTotal}
	}

	vsResTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "virtualserver_resources_total",
			Namespace:   metricsNamespace,
			Help:        "Number of handled VirtualServer resources",
			ConstLabels: constLabels,
		},
	)

	vsrResTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "virtualserverroute_resources_total",
			Namespace:   metricsNamespace,
			Help:        "Number of handled VirtualServerRoute resources",
			ConstLabels: constLabels,
		},
	)

	return &ControllerMetricsCollector{
		crdsEnabled:              true,
		ingressesTotal:           ingResTotal,
		virtualServersTotal:      vsResTotal,
		virtualServerRoutesTotal: vsrResTotal,
	}
}

// SetIngresses sets the value of the ingress resources gauge for a given type
func (cc *ControllerMetricsCollector) SetIngresses(ingressType string, count int) {
	cc.ingressesTotal.WithLabelValues(ingressType).Set(float64(count))
}

// SetVirtualServers sets the value of the VirtualServer resources gauge
func (cc *ControllerMetricsCollector) SetVirtualServers(count int) {
	cc.virtualServersTotal.Set(float64(count))
}

// SetVirtualServerRoutes sets the value of the VirtualServerRoute resources gauge
func (cc *ControllerMetricsCollector) SetVirtualServerRoutes(count int) {
	cc.virtualServerRoutesTotal.Set(float64(count))
}

// Describe implements prometheus.Collector interface Describe method
func (cc *ControllerMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	cc.ingressesTotal.Describe(ch)
	if cc.crdsEnabled {
		cc.virtualServersTotal.Describe(ch)
		cc.virtualServerRoutesTotal.Describe(ch)
	}
}

// Collect implements the prometheus.Collector interface Collect method
func (cc *ControllerMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	cc.ingressesTotal.Collect(ch)
	if cc.crdsEnabled {
		cc.virtualServersTotal.Collect(ch)
		cc.virtualServerRoutesTotal.Collect(ch)
	}
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

// SetIngresses implements a fake SetIngresses
func (cc *ControllerFakeCollector) SetIngresses(ingressType string, count int) {}

// SetVirtualServers implements a fake SetVirtualServers
func (cc *ControllerFakeCollector) SetVirtualServers(count int) {}

// SetVirtualServerRoutes implements a fake SetVirtualServerRoutes
func (cc *ControllerFakeCollector) SetVirtualServerRoutes(count int) {}

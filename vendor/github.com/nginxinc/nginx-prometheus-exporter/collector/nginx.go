package collector

import (
	"log"
	"sync"

	"github.com/nginxinc/nginx-prometheus-exporter/client"
	"github.com/prometheus/client_golang/prometheus"
)

// NginxCollector collects NGINX metrics. It implements prometheus.Collector interface.
type NginxCollector struct {
	nginxClient *client.NginxClient
	metrics     map[string]*prometheus.Desc
	upMetric    prometheus.Gauge
	mutex       sync.Mutex
}

// NewNginxCollector creates an NginxCollector.
func NewNginxCollector(nginxClient *client.NginxClient, namespace string) *NginxCollector {
	return &NginxCollector{
		nginxClient: nginxClient,
		metrics: map[string]*prometheus.Desc{
			"connections_active":   newGlobalMetric(namespace, "connections_active", "Active client connections"),
			"connections_accepted": newGlobalMetric(namespace, "connections_accepted", "Accepted client connections"),
			"connections_handled":  newGlobalMetric(namespace, "connections_handled", "Handled client connections"),
			"connections_reading":  newGlobalMetric(namespace, "connections_reading", "Connections where NGINX is reading the request header"),
			"connections_writing":  newGlobalMetric(namespace, "connections_writing", "Connections where NGINX is writing the response back to the client"),
			"connections_waiting":  newGlobalMetric(namespace, "connections_waiting", "Idle client connections"),
			"http_requests_total":  newGlobalMetric(namespace, "http_requests_total", "Total http requests"),
		},
		upMetric: newUpMetric(namespace),
	}
}

// Describe sends the super-set of all possible descriptors of NGINX metrics
// to the provided channel.
func (c *NginxCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upMetric.Desc()

	for _, m := range c.metrics {
		ch <- m
	}
}

// Collect fetches metrics from NGINX and sends them to the provided channel.
func (c *NginxCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock() // To protect metrics from concurrent collects
	defer c.mutex.Unlock()

	stats, err := c.nginxClient.GetStubStats()
	if err != nil {
		c.upMetric.Set(nginxDown)
		ch <- c.upMetric
		log.Printf("Error getting stats: %v", err)
		return
	}

	c.upMetric.Set(nginxUp)
	ch <- c.upMetric

	ch <- prometheus.MustNewConstMetric(c.metrics["connections_active"],
		prometheus.GaugeValue, float64(stats.Connections.Active))
	ch <- prometheus.MustNewConstMetric(c.metrics["connections_accepted"],
		prometheus.CounterValue, float64(stats.Connections.Accepted))
	ch <- prometheus.MustNewConstMetric(c.metrics["connections_handled"],
		prometheus.CounterValue, float64(stats.Connections.Handled))
	ch <- prometheus.MustNewConstMetric(c.metrics["connections_reading"],
		prometheus.GaugeValue, float64(stats.Connections.Reading))
	ch <- prometheus.MustNewConstMetric(c.metrics["connections_writing"],
		prometheus.GaugeValue, float64(stats.Connections.Writing))
	ch <- prometheus.MustNewConstMetric(c.metrics["connections_waiting"],
		prometheus.GaugeValue, float64(stats.Connections.Waiting))
	ch <- prometheus.MustNewConstMetric(c.metrics["http_requests_total"],
		prometheus.CounterValue, float64(stats.Requests))
}

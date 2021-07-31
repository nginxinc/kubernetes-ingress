package collector

import (
	"github.com/prometheus/client_golang/prometheus"
)

const nginxUp = 1
const nginxDown = 0

func newGlobalMetric(namespace string, metricName string, docString string, constLabels map[string]string) *prometheus.Desc {
	return prometheus.NewDesc(namespace+"_"+metricName, docString, nil, constLabels)
}

func newUpMetric(namespace string, constLabels map[string]string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Name:        "up",
		Help:        "Status of the last metric scrape",
		ConstLabels: constLabels,
	})
}

// MergeLabels merges two maps of labels.
func MergeLabels(a map[string]string, b map[string]string) map[string]string {
	c := make(map[string]string)

	for k, v := range a {
		c[k] = v
	}
	for k, v := range b {
		c[k] = v
	}

	return c
}

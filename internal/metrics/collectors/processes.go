package collectors

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

// NginxProcessesMetricsCollector implements prometheus.Collector interface
type NginxProcessesMetricsCollector struct {
	workerProcessTotal *prometheus.GaugeVec
}

// NewNginxProcessesMetricsCollector creates a new NginxProcessMetricsCollector
func NewNginxProcessesMetricsCollector(constLabels map[string]string) *NginxProcessesMetricsCollector {
	return &NginxProcessesMetricsCollector{
		workerProcessTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "nginx_worker_processes_total",
				Namespace:   metricsNamespace,
				Help:        "Number of NGINX worker processes",
				ConstLabels: constLabels,
			},
			[]string{"generation"},
		),
	}
}

// updateWorkerProcessCount sets the number of NGINX worker processes
func (pc *NginxProcessesMetricsCollector) updateWorkerProcessCount() {
	currWorkerProcesses, prevWorkerPrcesses, err := getWorkerProcesses()
	if err != nil {
		glog.Errorf("unable to collect process metrics : %v", err)
		return
	}
	pc.workerProcessTotal.WithLabelValues("current").Set(float64(currWorkerProcesses))
	pc.workerProcessTotal.WithLabelValues("old").Set(float64(prevWorkerPrcesses))
}

func getWorkerProcesses() (int, int, error) {
	var workerProcesses int
	var prevWorkerProcesses int

	procFolders, err := ioutil.ReadDir("/proc")
	if err != nil {
		return 0, 0, fmt.Errorf("unable to read directory /proc : %w", err)
	}

	for _, folder := range procFolders {
		_, err := strconv.Atoi(folder.Name())
		if err != nil {
			continue
		}

		cmdlineFile := fmt.Sprintf("/proc/%v/cmdline", folder.Name())
		content, err := ioutil.ReadFile(cmdlineFile)
		if err != nil {
			return 0, 0, fmt.Errorf("unable to read file %v: %w", cmdlineFile, err)
		}

		text := string(bytes.TrimRight(content, "\x00"))
		if text == "nginx: worker process" {
			workerProcesses++
		} else if text == "nginx: worker process is shutting down" {
			prevWorkerProcesses++
		}
	}
	return workerProcesses, prevWorkerProcesses, nil
}

// Collect implements the prometheus.Collector interface Collect method
func (pc *NginxProcessesMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	pc.updateWorkerProcessCount()
	pc.workerProcessTotal.Collect(ch)
}

// Describe implements prometheus.Collector interface Describe method
func (pc *NginxProcessesMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	pc.workerProcessTotal.Describe(ch)
}

// Register registers all the metrics of the collector
func (pc *NginxProcessesMetricsCollector) Register(registry *prometheus.Registry) error {
	return registry.Register(pc)
}

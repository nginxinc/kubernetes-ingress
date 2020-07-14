package collectors

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

// ManagerCollector is an interface for the metrics of the Nginx Manager
type ManagerCollector interface {
	IncNginxReloadCount()
	IncNginxReloadErrors()
	UpdateLastReloadTime(ms time.Duration)
	UpdateWorkerProcessCount(confVersion string)
	Register(registry *prometheus.Registry) error
}

// LocalManagerMetricsCollector implements NginxManagerCollector interface and prometheus.Collector interface
type LocalManagerMetricsCollector struct {
	// Metrics
	reloadsTotal          prometheus.Counter
	reloadsError          prometheus.Counter
	lastReloadStatus      prometheus.Gauge
	lastReloadTime        prometheus.Gauge
	workerProcessTotal    *prometheus.GaugeVec
	oldWorkerProcessTotal float64
}

// NewLocalManagerMetricsCollector creates a new LocalManagerMetricsCollector
func NewLocalManagerMetricsCollector(constLabels map[string]string) *LocalManagerMetricsCollector {
	labelNames := []string{"generation"}
	nc := &LocalManagerMetricsCollector{
		reloadsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name:        "nginx_reloads_total",
				Namespace:   metricsNamespace,
				Help:        "Number of successful NGINX reloads",
				ConstLabels: constLabels,
			},
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

		workerProcessTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "controller_nginx_worker_processes_total",
				Namespace:   metricsNamespace,
				Help:        "Number of NGINX worker processes",
				ConstLabels: constLabels,
			},
			labelNames,
		),
	}
	return nc
}

// IncNginxReloadCount increments the counter of successful NGINX reloads and sets the last reload status to true
func (nc *LocalManagerMetricsCollector) IncNginxReloadCount() {
	nc.reloadsTotal.Inc()
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

// UpdateWorkerProcessCount sets the number of NGINX worker processes
func (nc *LocalManagerMetricsCollector) UpdateWorkerProcessCount(configVersion string) {
	workerProcesses := 0
	procFolders, err := ioutil.ReadDir("/proc")
	if err != nil {
		glog.Errorf("error %v", err)
	}
	var processes = []string
	for _, f := range procFolders {
		u, err := user.LookupId(fmt.Sprint(f.Sys().(*syscall.Stat_t).Uid))
		if err != nil {
			glog.Errorf("Error %v", err)
		}
		if u.Name == "nginx user" {
			// store pids of the processes
			processes = append(processes, f.Name())
		}
	}
	for _, file := range processes {
		statusFile := fmt.Sprintf("/proc/%v/status", file)
		f, err := os.Open(statusFile)
		if err != nil {
			glog.Errorf("Error in opening file %v", statusFile)
		}

		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "PPid") {
				words := strings.Split(scanner.Text(), "\t")
				ppid := words[len(words)-1]

				if ppid != "0" {
					workerProcesses++
				}
			}
		}
	}
	nc.workerProcessTotal.WithLabelValues(configVersion).Set(float64(workerProcesses))
	nc.workerProcessTotal.WithLabelValues("old").Set(nc.oldWorkerProcessTotal)
	nc.oldWorkerProcessTotal = float64(workerProcesses)
}

// Describe implements prometheus.Collector interface Describe method
func (nc *LocalManagerMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	nc.reloadsTotal.Describe(ch)
	nc.reloadsError.Describe(ch)
	nc.lastReloadStatus.Describe(ch)
	nc.lastReloadTime.Describe(ch)
	nc.workerProcessTotal.Describe(ch)
}

// Collect implements the prometheus.Collector interface Collect method
func (nc *LocalManagerMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	nc.reloadsTotal.Collect(ch)
	nc.reloadsError.Collect(ch)
	nc.lastReloadStatus.Collect(ch)
	nc.lastReloadTime.Collect(ch)
	nc.workerProcessTotal.Collect(ch)
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
func (nc *ManagerFakeCollector) Register(registry *prometheus.Registry) error { return nil }

// IncNginxReloadCount implements a fake IncNginxReloadCount
func (nc *ManagerFakeCollector) IncNginxReloadCount() {}

// IncNginxReloadErrors implements a fake IncNginxReloadErrors
func (nc *ManagerFakeCollector) IncNginxReloadErrors() {}

// UpdateLastReloadTime implements a fake UpdateLastReloadTime
func (nc *ManagerFakeCollector) UpdateLastReloadTime(ms time.Duration) {}

// UpdateWorkerProcessCount implements a fake UpdateWorkerPorcessCount
func (nc *ManagerFakeCollector) UpdateWorkerProcessCount(confVersion string) {
}

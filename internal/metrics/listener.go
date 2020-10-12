package metrics

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	prometheusClient "github.com/nginxinc/nginx-prometheus-exporter/client"
	nginxCollector "github.com/nginxinc/nginx-prometheus-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metricsEndpoint is the path where prometheus metrics will be exposed
const metricsEndpoint = "/metrics"

// NewNginxMetricsClient creates an NginxClient to fetch stats from NGINX over an unix socket
func NewNginxMetricsClient(httpClient *http.Client) (*prometheusClient.NginxClient, error) {
	return prometheusClient.NewNginxClient(httpClient, "http://config-status/stub_status")
}

// RunPrometheusListenerForNginx runs an http server to expose Prometheus metrics for NGINX
func RunPrometheusListenerForNginx(port int, client *prometheusClient.NginxClient, registry *prometheus.Registry, constLabels map[string]string) {
	registry.MustRegister(nginxCollector.NewNginxCollector(client, "nginx_ingress_nginx", constLabels))
	runServer(strconv.Itoa(port), registry)
}

// RunPrometheusListenerForNginxPlus runs an http server to expose Prometheus metrics for NGINX Plus
func RunPrometheusListenerForNginxPlus(port int, nginxPlusCollector prometheus.Collector, registry *prometheus.Registry) {
	registry.MustRegister(nginxPlusCollector)
	runServer(strconv.Itoa(port), registry)
}

func runServer(port string, registry prometheus.Gatherer) {
	http.Handle(metricsEndpoint, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
			<head><title>NGINX Ingress Controller</title></head>
			<body>
			<h1>NGINX Ingress Controller</h1>
			<p><a href='/metrics'>Metrics</a></p>
			</body>
			</html>`))
		if err != nil {
			glog.Warningf("Error while sending a response for the '/' path: %v", err)
		}
	})
	address := fmt.Sprintf(":%v", port)
	glog.Infof("Starting Prometheus listener on: %v%v", address, metricsEndpoint)
	glog.Fatal("Error in Prometheus listener server: ", http.ListenAndServe(address, nil))
}

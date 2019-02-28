package metrics

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	sdkClient "github.com/nginxinc/nginx-plus-go-sdk/client"
	prometheusClient "github.com/nginxinc/nginx-prometheus-exporter/client"
	"github.com/nginxinc/nginx-prometheus-exporter/collector"
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
func RunPrometheusListenerForNginx(port int, client *prometheusClient.NginxClient) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector.NewNginxCollector(client, "nginx"))
	runServer(strconv.Itoa(port), registry)
}

// RunPrometheusListenerForNginxPlus runs an http server to expose Prometheus metrics for NGINX Plus
func RunPrometheusListenerForNginxPlus(port int, plusClient *sdkClient.NginxClient) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector.NewNginxPlusCollector(plusClient, "nginxplus"))
	runServer(strconv.Itoa(port), registry)
}

func runServer(port string, registry prometheus.Gatherer) {
	http.Handle(metricsEndpoint, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>NGINX Ingress Controller</title></head>
			<body>
			<h1>NGINX Ingress Controller</h1>
			<p><a href='/metrics'>Metrics</a></p>
			</body>
			</html>`))
	})
	address := fmt.Sprintf(":%v", port)
	glog.Infof("Starting Prometheus listener on: %v%v", address, metricsEndpoint)
	glog.Fatal("Error in Prometheus listener server: ", http.ListenAndServe(address, nil))
}

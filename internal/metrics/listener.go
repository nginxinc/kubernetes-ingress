package metrics

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	prometheusClient "github.com/nginxinc/nginx-prometheus-exporter/client"
	nginxCollector "github.com/nginxinc/nginx-prometheus-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	api_v1 "k8s.io/api/core/v1"
)

// metricsEndpoint is the path where prometheus metrics will be exposed
const metricsEndpoint = "/metrics"

// NewNginxMetricsClient creates an NginxClient to fetch stats from NGINX over an unix socket
func NewNginxMetricsClient(httpClient *http.Client) (*prometheusClient.NginxClient, error) {
	return prometheusClient.NewNginxClient(httpClient, "http://config-status/stub_status")
}

// RunPrometheusListenerForNginx runs an http server to expose Prometheus metrics for NGINX
func RunPrometheusListenerForNginx(port int, client *prometheusClient.NginxClient, registry *prometheus.Registry, constLabels map[string]string, prometheusSecret *api_v1.Secret) {
	registry.MustRegister(nginxCollector.NewNginxCollector(client, "nginx_ingress_nginx", constLabels))
	runServer(strconv.Itoa(port), registry, prometheusSecret)
}

// RunPrometheusListenerForNginxPlus runs an http server to expose Prometheus metrics for NGINX Plus
func RunPrometheusListenerForNginxPlus(port int, nginxPlusCollector prometheus.Collector, registry *prometheus.Registry, prometheusSecret *api_v1.Secret) {
	registry.MustRegister(nginxPlusCollector)
	runServer(strconv.Itoa(port), registry, prometheusSecret)
}

func runServer(port string, registry prometheus.Gatherer, prometheusSecret *api_v1.Secret) {
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
	if prometheusSecret == nil {
		glog.Fatal("Error in Prometheus listener server: ", http.ListenAndServe(address, nil))
	} else {
		// Unfortunately, http.ListenAndServeTLS() takes a filename instead of cert/key data, so we
		// Write the cert and key to a temporary file. We create a unique file name to prevent collisions.
		certFileName := "nginx-prometheus.cert"
		keyFileName := "nginx-prometheus.key"
		certFile, err := writeTempFile(prometheusSecret.Data[api_v1.TLSCertKey], certFileName)
		if err != nil {
			glog.Fatal("failed to create cert file for prometheus: %w", err)
		}

		keyFile, err := writeTempFile(prometheusSecret.Data[api_v1.TLSPrivateKeyKey], keyFileName)
		if err != nil {
			glog.Fatal("failed to create key file for prometheus: %w", err)
		}

		glog.Fatal("Error in Prometheus listener server: ", http.ListenAndServeTLS(address, certFile.Name(), keyFile.Name(), nil))
	}
}

func writeTempFile(data []byte, name string) (*os.File, error) {
	f, err := ioutil.TempFile("", name)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	err = f.Chmod(nginx.TLSSecretFileMode)
	if err != nil {
		return nil, fmt.Errorf("couldn't change the mode of the temp file %v: %w", f.Name(), err)
	}

	_, err = f.Write(data)
	if err != nil {
		return f, fmt.Errorf("failed to write to temp file: %w", err)
	}

	return f, nil
}

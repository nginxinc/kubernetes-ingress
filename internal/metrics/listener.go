package metrics

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang/glog"
	prometheusClient "github.com/nginxinc/nginx-prometheus-exporter/client"
	nginxCollector "github.com/nginxinc/nginx-prometheus-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	v1 "k8s.io/api/core/v1"
)

// NewNginxMetricsClient creates an NginxClient to fetch stats from NGINX over an unix socket
func NewNginxMetricsClient(httpClient *http.Client) *prometheusClient.NginxClient {
	return prometheusClient.NewNginxClient(httpClient, "http://config-status/stub_status")
}

// RunPrometheusListenerForNginx runs an http server to expose Prometheus metrics for NGINX
func RunPrometheusListenerForNginx(port int, client *prometheusClient.NginxClient, registry *prometheus.Registry, constLabels map[string]string, prometheusSecret *v1.Secret) {
	infoLevel := new(promlog.AllowedLevel)
	err := infoLevel.Set("info")
	if err != nil {
		glog.Error("Error setting prometheus exporter log level")
	}
	promlogConfig := &promlog.Config{
		Level: infoLevel,
	}

	logger := promlog.New(promlogConfig)
	registry.MustRegister(nginxCollector.NewNginxCollector(client, "nginx_ingress_nginx", constLabels, logger))
	runServer(strconv.Itoa(port), registry, prometheusSecret)
}

// RunPrometheusListenerForNginxPlus runs an http server to expose Prometheus metrics for NGINX Plus
func RunPrometheusListenerForNginxPlus(port int, nginxPlusCollector prometheus.Collector, registry *prometheus.Registry, prometheusSecret *v1.Secret) {
	registry.MustRegister(nginxPlusCollector)
	runServer(strconv.Itoa(port), registry, prometheusSecret)
}

// runServer starts the metrics server.
func runServer(port string, registry prometheus.Gatherer, prometheusSecret *v1.Secret) {
	addr := fmt.Sprintf(":%s", port)
	s, err := NewServer(addr, registry, prometheusSecret)
	if err != nil {
		glog.Fatal(err)
	}
	glog.Infof("Starting prometheus listener on: %s/metrics", addr)
	glog.Fatal(s.ListenAndServe())
}

// Server holds information about NIC metrics server.
type Server struct {
	Server   *http.Server
	URL      string
	Registry prometheus.Gatherer
}

// NewServer creates HTTP server for serving NIC metrics for Prometheus.
//
// Metrics are exposed on the `/metrics` endpoint.
func NewServer(addr string, registry prometheus.Gatherer, secret *v1.Secret) (*Server, error) {
	s := Server{
		Server: &http.Server{
			Addr:         addr,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		URL:      fmt.Sprintf("http://%s/", addr),
		Registry: registry,
	}
	// Secrets are read from K8s API. If the secret for Prometheus is present
	// we configure Metrics Server with the key and cert.
	if secret != nil {
		tlsCert, err := makeCert(secret)
		if err != nil {
			return nil, fmt.Errorf("unable to create TLS cert: %w", err)
		}
		s.Server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			MinVersion:   tls.VersionTLS12,
		}
		s.URL = fmt.Sprintf("https://%s/", addr)
	}
	return &s, nil
}

// Home is a handler for serving metrics home page.
func (s *Server) Home(w http.ResponseWriter, r *http.Request) { //nolint:revive
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`<html>
			<head><title>NGINX Ingress Controller</title></head>
			<body>
			<h1>NGINX Ingress Controller</h1>
			<p><a href='/metrics'>Metrics</a></p>
			</body>
			</html>`))
	if err != nil {
		glog.Errorf("error while sending a response for the '/' path: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

// ListenAndServe starts metrics server.
func (s *Server) ListenAndServe() error {
	mux := chi.NewRouter()
	mux.Get("/", s.Home)
	mux.Handle("/metrics", promhttp.HandlerFor(s.Registry, promhttp.HandlerOpts{}))
	s.Server.Handler = mux
	if s.Server.TLSConfig != nil {
		return s.Server.ListenAndServeTLS("", "")
	}
	return s.Server.ListenAndServe()
}

// Shutdown shuts down metrics server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

// makeCert takes K8s Secret and returns tls Certificate for the server.
// It errors if either cert, or key are not present in the Secret.
func makeCert(s *v1.Secret) (tls.Certificate, error) {
	cert, ok := s.Data[v1.TLSCertKey]
	if !ok {
		return tls.Certificate{}, errors.New("missing tls cert")
	}
	key, ok := s.Data[v1.TLSPrivateKeyKey]
	if !ok {
		return tls.Certificate{}, errors.New("missing tls key")
	}
	return tls.X509KeyPair(cert, key)
}

package healthcheck

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/go-chi/chi"
	"github.com/go-chi/httprate"
	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/nginx-plus-go-client/client"
	"k8s.io/utils/strings/slices"
)

// RunHealtcheckServer takes config params, creates the health server and starts it.
// It errors if the server can't be started or provided secret is not valid
// (tls certificate cannot be created) and the health service with TLS support can't start.
func RunHealtcheckServer(port string, nc *client.NginxClient, cnf *configs.Configurator, secret *v1.Secret) error {
	getUpstreamsForHost := UpstreamsForHost(cnf)
	getUpstreamsFromNginx := NginxUpstreams(nc)

	if secret == nil {
		healthServer := http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      API(getUpstreamsForHost, getUpstreamsFromNginx),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		if err := healthServer.ListenAndServe(); err != nil {
			return fmt.Errorf("starting healthcheck server: %w", err)
		}
	}

	tlsCert, err := makeCert(secret)
	if err != nil {
		return fmt.Errorf("creating tls cert %w", err)
	}

	healthServer := http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: API(getUpstreamsForHost, getUpstreamsFromNginx),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			MinVersion:   tls.VersionTLS10,
		},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if err := healthServer.ListenAndServeTLS("", ""); err != nil {
		return fmt.Errorf("starting healthcheck tls server: %w", err)
	}
	return nil
}

// makeCert takes k8s Secret and returns tls Certificate for the server.
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

// API constructs an http.Handler with all healtcheck routes.
func API(upstreamsForHost func(string) []string, upstreamsFromNginx func() (*client.Upstreams, error)) http.Handler {
	health := HealthHandler{
		UpstreamsForHost: upstreamsForHost,
		NginxUpstreams:   upstreamsFromNginx,
	}
	mux := chi.NewRouter()
	mux.Use(httprate.Limit(10, 1*time.Second, httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
	})),
	)
	mux.MethodFunc(http.MethodGet, "/probe/{hostname}", health.Retrieve)
	return mux
}

// UpstreamsForHost takes configurator and returns a func
// that is reposnsible for retrieving upstreams for the given hostname.
func UpstreamsForHost(cnf *configs.Configurator) func(hostname string) []string {
	return func(hostname string) []string {
		return cnf.GetUpstreamsforHost(hostname)
	}
}

// NginxUpstreams takes an instance of NGNX client and returns
// a func that returns all upstreams.
func NginxUpstreams(nc *client.NginxClient) func() (*client.Upstreams, error) {
	return func() (*client.Upstreams, error) {
		upstreams, err := nc.GetUpstreams()
		if err != nil {
			return nil, err
		}
		return upstreams, nil
	}
}

// HealthHandler holds dependency for its method(s).
type HealthHandler struct {
	UpstreamsForHost func(host string) []string
	NginxUpstreams   func() (*client.Upstreams, error)
}

// Retrieve finds health stats for the host identified by a hostname in the request URL.
func (h *HealthHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "hostname")

	upstreamNames := h.UpstreamsForHost(hostname)
	if len(upstreamNames) == 0 {
		glog.Errorf("no upstreams for hostname %s or hostname does not exist", hostname)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	upstreams, err := h.NginxUpstreams()
	if err != nil {
		glog.Errorf("error retrieving upstreams for host: %s", hostname)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stats := countStats(upstreams, upstreamNames)
	data, err := json.Marshal(stats)
	if err != nil {
		glog.Error("error marshaling result", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch stats.Up {
	case 0:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusOK)
	}
	if _, err = w.Write(data); err != nil {
		glog.Error("error writing result", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

// HostStats holds information about total, up and
// unhealthy number of 'peers' associated with the
// given host.
type HostStats struct {
	Total     int
	Up        int
	Unhealthy int
}

// countStats calculates and returns statistics.
func countStats(upstreams *client.Upstreams, upstreamNames []string) HostStats {
	total, up := 0, 0
	for name, u := range *upstreams {
		if slices.Contains(upstreamNames, name) {
			for _, p := range u.Peers {
				total++
				if strings.ToLower(p.State) != "up" {
					continue
				}
				up++
			}
		}
	}
	unhealthy := total - up
	return HostStats{
		Total:     total,
		Up:        up,
		Unhealthy: unhealthy,
	}
}

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
	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/nginx-plus-go-client/client"
	"k8s.io/utils/strings/slices"
)

// RunHealtcheckServer takes configs and starts healtcheck service.
func RunHealtcheckServer(port string, nc *client.NginxClient, cnf *configs.Configurator, secret *v1.Secret) error {
	if secret == nil {
		healthServer := http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      API(nc, cnf),
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
		Handler: API(nc, cnf),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
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
func API(client *client.NginxClient, cnf *configs.Configurator) http.Handler {
	health := HealthHandler{
		client: client,
		cnf:    cnf,
	}
	mux := chi.NewRouter()
	mux.MethodFunc(http.MethodGet, "/", health.Info)
	mux.MethodFunc(http.MethodGet, "/healthcheck/{hostname}", health.Retrieve)
	return mux
}

// ===================================================================================
// Handlers
// ===================================================================================

type HealthHandler struct {
	client *client.NginxClient
	cnf    *configs.Configurator
}

// Info returns basic information about healthcheck service.
func (h *HealthHandler) Info(w http.ResponseWriter, r *http.Request) {
	// for now it is a placeholder for the response body
	// we would return to a caller on GET request to '/'
	info := struct {
		Info    string
		Version string
		Usage   string
	}{
		Info:    "extended healthcheck endpoint",
		Version: "0.1",
		Usage:   "/{hostname}",
	}
	data, err := json.Marshal(info)
	if err != nil {
		glog.Error("error marshalling result", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		glog.Error("error writing result", err)
	}
}

// Retrieve finds health stats for a host identified by an hostname in the request URL.
func (h *HealthHandler) Retrieve(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "hostname")

	upstreamNames := h.cnf.GetUpstreamsforHost(hostname)
	if len(upstreamNames) == 0 {
		glog.Errorf("no upstreams for hostname %s or hostname does not exist", hostname)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	upstreams, err := h.client.GetUpstreams()
	if err != nil {
		glog.Errorf("error retriving upstreams for host: %s", hostname)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stats := countStats(upstreams, upstreamNames)
	data, err := json.Marshal(stats)
	if err != nil {
		glog.Error("error marshalling result", err)
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
	}
}

func (h *HealthHandler) Status(w http.ResponseWriter, r *http.Request) {
	// possible implement website that currently is served from listener.go ?
	w.WriteHeader(http.StatusNotImplemented)
}

type hostStats struct {
	Total     int // Total number of configured servers (peers)
	Up        int // The number of servers (peers) with 'up' status
	Unhealthy int // The number of servers (peers) with 'down' status
}

// countStats calculates and returns statistics.
func countStats(upstreams *client.Upstreams, upstreamNames []string) hostStats {
	total, up := 0, 0

	for name, u := range *upstreams {
		if slices.Contains(upstreamNames, name) {
			for _, p := range u.Peers {
				total++
				if strings.ToLower(p.State) == "up" {
					up++
				}
			}
		}
	}
	unhealthy := total - up
	return hostStats{
		Total:     total,
		Up:        up,
		Unhealthy: unhealthy,
	}
}

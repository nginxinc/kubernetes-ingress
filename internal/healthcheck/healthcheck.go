// Package healthcheck provides primitives for running deep healtcheck service.
package healthcheck

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/go-chi/chi"
	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/nginx-plus-go-client/client"
	"k8s.io/utils/strings/slices"
)

// RunHealthCheck starts the deep healthcheck service.
func RunHealthCheck(port int, plusClient *client.NginxClient, cnf *configs.Configurator, healthProbeTLSSecret *v1.Secret) {
	addr := fmt.Sprintf(":%s", strconv.Itoa(port))
	hs, err := NewHealthServer(addr, plusClient, cnf, healthProbeTLSSecret)
	if err != nil {
		glog.Fatal(err)
	}
	glog.Infof("Starting Service Insight listener on: %v%v", addr, "/probe")
	glog.Fatal(hs.ListenAndServe())
}

// HealthServer holds data required for running
// the healthcheck server.
type HealthServer struct {
	Server           *http.Server
	URL              string
	UpstreamsForHost func(host string) []string
	NginxUpstreams   func() (*client.Upstreams, error)
}

// NewHealthServer creates Health Server. If secret is provided,
// the server is configured with TLS Config.
func NewHealthServer(addr string, nc *client.NginxClient, cnf *configs.Configurator, secret *v1.Secret) (*HealthServer, error) {
	hs := HealthServer{
		Server: &http.Server{
			Addr:         addr,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		URL:              fmt.Sprintf("http://%s/", addr),
		UpstreamsForHost: cnf.GetUpstreamsforHost,
		NginxUpstreams:   nc.GetUpstreams,
	}

	if secret != nil {
		tlsCert, err := makeCert(secret)
		if err != nil {
			return nil, fmt.Errorf("unable to create TLS cert: %w", err)
		}
		hs.Server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
			MinVersion:   tls.VersionTLS12,
		}
		hs.URL = fmt.Sprintf("https://%s/", addr)
	}
	return &hs, nil
}

// ListenAndServe starts healthcheck server.
func (hs *HealthServer) ListenAndServe() error {
	mux := chi.NewRouter()
	mux.Get("/probe/{hostname}", hs.Retrieve)
	hs.Server.Handler = mux
	if hs.Server.TLSConfig != nil {
		return hs.Server.ListenAndServeTLS("", "")
	}
	return hs.Server.ListenAndServe()
}

// Shutdown shuts down healthcheck server.
func (hs *HealthServer) Shutdown(ctx context.Context) error {
	return hs.Server.Shutdown(ctx)
}

// Retrieve finds health stats for the host identified by a hostname in the request URL.
func (hs *HealthServer) Retrieve(w http.ResponseWriter, r *http.Request) {
	hostname := chi.URLParam(r, "hostname")
	host := sanitize(hostname)

	upstreamNames := hs.UpstreamsForHost(host)
	if len(upstreamNames) == 0 {
		glog.Errorf("no upstreams for requested hostname %s or hostname does not exist", host)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	upstreams, err := hs.NginxUpstreams()
	if err != nil {
		glog.Errorf("error retrieving upstreams for requested hostname: %s", host)
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

func sanitize(s string) string {
	hostname := strings.TrimSpace(s)
	hostname = strings.ReplaceAll(hostname, "\n", "")
	hostname = strings.ReplaceAll(hostname, "\r", "")
	return hostname
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

// HostStats holds information about total, up and
// unhealthy number of 'peers' associated with the
// given host.
type HostStats struct {
	Total     int
	Up        int
	Unhealthy int
}

// countStats calculates and returns statistics for a host.
func countStats(upstreams *client.Upstreams, upstreamNames []string) HostStats {
	total, up := 0, 0
	for name, u := range *upstreams {
		if !slices.Contains(upstreamNames, name) {
			continue
		}
		for _, p := range u.Peers {
			total++
			if strings.ToLower(p.State) != "up" {
				continue
			}
			up++
		}
	}

	unhealthy := total - up
	return HostStats{
		Total:     total,
		Up:        up,
		Unhealthy: unhealthy,
	}
}

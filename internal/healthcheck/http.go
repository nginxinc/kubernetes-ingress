package healthcheck

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/nginx-plus-go-client/client"
	"k8s.io/utils/strings/slices"
)

// RunHealtcheckServer takes configs and starts healtcheck service.
func RunHealtcheckServer(port string, nc *client.NginxClient, cnf *configs.Configurator) error {
	healthServer := http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: API(nc, cnf),

		// For now hardcoded!
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if err := healthServer.ListenAndServe(); err != nil {
		glog.Error("error starting healtcheck server", err)
		return fmt.Errorf("starting healtcheck server: %w", err)
	}
	return nil
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

	upstreams, err := h.client.GetUpstreams()
	if err != nil {
		glog.Errorf("error retriving upstreams for host: %s", hostname)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	upstreamNames := h.cnf.GetUpstreamsforHost(hostname)

	stats := countStats(upstreams, upstreamNames)

	data, err := json.Marshal(stats)
	if err != nil {
		glog.Error("error marshalling result", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// NOTE:
	// Here we need logic to setup correct header depending on the stats result!
	// current one is initial implementation.
	// Do ww have to calculate ratio of total/down to return a different status?
	// TBD!
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
	// possible implement website that currently is server from listener.go ?
	w.WriteHeader(http.StatusNotImplemented)
}

type hostStats struct {
	Total     int // Total number of configured servers (peers)
	Up        int // The number of servers (peers) with 'up' status
	Unhealthy int // The number of servers (peers) with 'down' status
}

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

// ListenAndServeTLS is a drop-in replacement for a default
// ListenAndServeTLS func from the http std library. The func
// takes not paths to a cert and a key but slices of bytes representing
// content of the files used by the original http.ListenAndServeTLS function.
func ListenAndServeTLS(addr string, cert, key []byte, handler http.Handler) error {
	tlsCert, err := LoadX509KeyPair(cert, key)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		},
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return server.ListenAndServeTLS("", "")
}

// LoadX509KeyPair is a drop-in replacement for the LoadX509KeyPair
// from the http package from the std library. It takes not files
// containing a cert and a key but a slice of bytes representing
// the content of the corresponding files.
func LoadX509KeyPair(cert, key []byte) (tls.Certificate, error) {
	return tls.X509KeyPair(cert, key)
}

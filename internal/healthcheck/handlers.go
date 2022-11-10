package healthcheck

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/nginx-plus-go-client/client"
	"k8s.io/utils/strings/slices"
)

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
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		glog.Error("error writing result", err)
	}
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

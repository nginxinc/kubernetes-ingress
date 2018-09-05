package plus

import (
	"net/http"

	"github.com/golang/glog"
)

type NginxAPIController struct {
	client *NginxClient
	local  bool
}

type ServerConfig struct {
	MaxFails    int64
	FailTimeout string
	SlowStart   string
}

func NewNginxAPIController(httpClient *http.Client, endpoint string, local bool) (*NginxAPIController, error) {
	client, err := NewNginxClient(httpClient, endpoint)
	if !local && err != nil {
		return nil, err
	}
	nginx := &NginxAPIController{client: client, local: local}
	return nginx, nil
}

func (nginx *NginxAPIController) UpdateServers(upstream string, servers []string, config ServerConfig) error {
	if nginx.local {
		glog.V(3).Infof("Updating endpoints of %v: %v\n", upstream, servers)
		return nil
	}

	var upsServers []UpstreamServer
	for _, s := range servers {
		upsServers = append(upsServers, UpstreamServer{
			Server:      s,
			MaxFails:    config.MaxFails,
			FailTimeout: config.FailTimeout,
			SlowStart:   config.SlowStart,
		})
	}

	added, removed, err := nginx.client.UpdateHTTPServers(upstream, upsServers)
	if err != nil {
		glog.V(3).Infof("Couldn't update servers of %v upstream: %v", upstream, err)
		return err
	}

	glog.V(3).Infof("Updated servers of %v; Added: %v, Removed: %v", upstream, added, removed)
	return nil
}

package plus

import (
	"net/http"

	"github.com/golang/glog"
)

type NginxAPIController struct {
	client *NginxClient
	local  bool
}

func NewNginxAPIController(httpClient *http.Client, endpoint string, local bool) (*NginxAPIController, error) {
	client, err := NewNginxClient(httpClient, endpoint)
	if !local && err != nil {
		return nil, err
	}
	nginx := &NginxAPIController{client: client, local: local}
	return nginx, nil
}

func (nginx *NginxAPIController) UpdateServers(upstream string, servers []string) error {
	if nginx.local {
		glog.V(3).Infof("Updating endpoints of %v: %v\n", upstream, servers)
		return nil
	}
	added, removed, err := nginx.client.UpdateHTTPServers(upstream, servers)
	if err != nil {
		glog.V(3).Infof("Couldn't update servers of %v upstream: %v", upstream, err)
		return err
	}

	glog.V(3).Infof("Updated servers of %v; Added: %v, Removed: %v", upstream, added, removed)
	return nil
}

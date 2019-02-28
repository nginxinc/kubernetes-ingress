package plus

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	plus "github.com/nginxinc/nginx-plus-go-sdk/client"
)

// NginxAPIController works with the NGINX API
type NginxAPIController struct {
	client     *plus.NginxClient
	httpClient *http.Client
	local      bool
}

// ServerConfig holds the config data
type ServerConfig struct {
	MaxFails    int
	FailTimeout string
	SlowStart   string
}

// NewNginxAPIController creates an instance of NginxAPIController
func NewNginxAPIController(httpClient *http.Client, endpoint string, local bool) (*NginxAPIController, error) {
	client, err := plus.NewNginxClient(httpClient, endpoint)
	if !local && err != nil {
		return nil, err
	}
	nginx := &NginxAPIController{client: client, httpClient: httpClient, local: local}
	return nginx, nil
}

// verifyConfigVersion is used to check if the worker process that the API client is connected
// to is using the latest version of nginx config. This way we avoid making changes on
// a worker processes that is being shut down.
func verifyConfigVersion(httpClient *http.Client, configVersion int) error {
	req, err := http.NewRequest("GET", "http://nginx-plus-api/configVersionCheck", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("x-expected-config-version", fmt.Sprintf("%v", configVersion))
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error doing request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("API returned non-success status: %v", resp.StatusCode)
	}
	return nil
}

// UpdateServers updates upstream servers
func (nginx *NginxAPIController) UpdateServers(upstream string, servers []string, config ServerConfig, configVersion int) error {
	if nginx.local {
		glog.V(3).Infof("Updating endpoints of %v: %v\n", upstream, servers)
		return nil
	}

	err := verifyConfigVersion(nginx.httpClient, configVersion)
	if err != nil {
		return fmt.Errorf("error verifying config version: %v", err)
	}
	glog.V(3).Infof("API has the correct config version: %v.", configVersion)

	var upsServers []plus.UpstreamServer
	for _, s := range servers {
		upsServers = append(upsServers, plus.UpstreamServer{
			Server:      s,
			MaxFails:    config.MaxFails,
			FailTimeout: config.FailTimeout,
			SlowStart:   config.SlowStart,
		})
	}

	added, removed, err := nginx.client.UpdateHTTPServers(upstream, upsServers)
	if err != nil {
		glog.V(3).Infof("Couldn't update servers of %v upstream: %v", upstream, err)
		return fmt.Errorf("error updating servers of %v upstream: %v", upstream, err)
	}

	glog.V(3).Infof("Updated servers of %v; Added: %v, Removed: %v", upstream, added, removed)
	return nil
}

// GetClientPlus returns the internal client for NGINX Plus API to reuse it outside the package
func (nginx *NginxAPIController) GetClientPlus() *plus.NginxClient {
	if nginx != nil && nginx.client != nil {
		return nginx.client
	}
	return nil
}

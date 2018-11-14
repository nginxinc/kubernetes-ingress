package verify

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
)

// Client is a client for verifying the config version.
type Client struct {
	client     *http.Client
	maxRetries int
}

// NewClient returns a new client pointed at the config version socket.
func NewClient() *Client {
	return &Client{
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", "/var/run/nginx-config-version.sock")
				},
			},
		},
		maxRetries: 160,
	}
}

// GetConfigVersion get version number that we put in the nginx config to verify that we're using
// the correct config.
func (c *Client) GetConfigVersion() (int, error) {
	resp, err := c.client.Get("http://config-version/configVersion")
	if err != nil {
		return 0, fmt.Errorf("error getting client: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("non-200 response: %v", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read the response body: %v", err)
	}
	v, err := strconv.Atoi(string(body))
	if err != nil {
		return 0, fmt.Errorf("error converting string to int: %v", err)
	}
	return v, nil
}

// WaitForCorrectVersion calls the config version endpoint until it gets the expectedVersion,
// which ensures that a new worker process has been started for that config version.
func (c *Client) WaitForCorrectVersion(expectedVersion int) error {
	sleep := 25 * time.Millisecond
	for i := 1; i <= c.maxRetries; i++ {
		time.Sleep(sleep)

		version, err := c.GetConfigVersion()
		if err != nil {
			glog.V(3).Infof("Unable to fetch version: %v", err)
			continue
		}
		if version == expectedVersion {
			glog.V(3).Infof("success, version %v ensured. iterations: %v. took: %v", expectedVersion, i, time.Duration(i)*sleep)
			return nil
		}
	}
	return fmt.Errorf("could not get expected version: %v", expectedVersion)
}

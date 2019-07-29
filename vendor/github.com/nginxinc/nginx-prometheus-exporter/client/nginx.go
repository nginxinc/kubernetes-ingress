package client

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// NginxClient allows you to fetch NGINX metrics from the stub_status page.
type NginxClient struct {
	apiEndpoint string
	httpClient  *http.Client
}

// StubStats represents NGINX stub_status metrics.
type StubStats struct {
	Connections StubConnections
	Requests    int64
}

// StubConnections represents connections related metrics.
type StubConnections struct {
	Active   int64
	Accepted int64
	Handled  int64
	Reading  int64
	Writing  int64
	Waiting  int64
}

// NewNginxClient creates an NginxClient.
func NewNginxClient(httpClient *http.Client, apiEndpoint string) (*NginxClient, error) {
	client := &NginxClient{
		apiEndpoint: apiEndpoint,
		httpClient:  httpClient,
	}

	_, err := client.GetStubStats()
	return client, err
}

// GetStubStats fetches the stub_status metrics.
func (client *NginxClient) GetStubStats() (*StubStats, error) {
	resp, err := client.httpClient.Get(client.apiEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get %v: %v", client.apiEndpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected %v response, got %v", http.StatusOK, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response body: %v", err)
	}

	var stats StubStats
	err = parseStubStats(body, &stats)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response body %q: %v", string(body), err)
	}

	return &stats, nil
}

func parseStubStats(data []byte, stats *StubStats) error {
	dataStr := string(data)

	parts := strings.Split(dataStr, "\n")
	if len(parts) != 5 {
		return fmt.Errorf("invalid input %q", dataStr)
	}

	activeConsParts := strings.Split(strings.TrimSpace(parts[0]), " ")
	if len(activeConsParts) != 3 {
		return fmt.Errorf("invalid input for active connections %q", parts[0])
	}

	actCons, err := strconv.ParseInt(activeConsParts[2], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid input for active connections %q: %v", activeConsParts[2], err)
	}
	stats.Connections.Active = actCons

	miscParts := strings.Split(strings.TrimSpace(parts[2]), " ")
	if len(miscParts) != 3 {
		return fmt.Errorf("invalid input for connections and requests %q", parts[2])
	}

	acceptedCons, err := strconv.ParseInt(miscParts[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid input for accepted connections %q: %v", miscParts[0], err)
	}
	stats.Connections.Accepted = acceptedCons

	handledCons, err := strconv.ParseInt(miscParts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid input for handled connections %q: %v", miscParts[1], err)
	}
	stats.Connections.Handled = handledCons

	requests, err := strconv.ParseInt(miscParts[2], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid input for requests %q: %v", miscParts[2], err)
	}
	stats.Requests = requests

	consParts := strings.Split(strings.TrimSpace(parts[3]), " ")
	if len(consParts) != 6 {
		return fmt.Errorf("invalid input for connections %q", parts[3])
	}

	readingCons, err := strconv.ParseInt(consParts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid input for reading connections %q: %v", consParts[1], err)
	}
	stats.Connections.Reading = readingCons

	writingCons, err := strconv.ParseInt(consParts[3], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid input for writing connections %q: %v", consParts[3], err)
	}
	stats.Connections.Writing = writingCons

	waitingCons, err := strconv.ParseInt(consParts[5], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid input for waiting connections %q: %v", consParts[5], err)
	}
	stats.Connections.Waiting = waitingCons

	return nil
}

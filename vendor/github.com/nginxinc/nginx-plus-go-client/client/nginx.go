package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
)

const (
	// APIVersion is a version of NGINX Plus API.
	APIVersion = 5

	pathNotFoundCode  = "PathNotFound"
	streamContext     = true
	httpContext       = false
	defaultServerPort = "80"
)

// Default values for servers in Upstreams.
var (
	defaultMaxConns    = 0
	defaultMaxFails    = 1
	defaultFailTimeout = "10s"
	defaultSlowStart   = "0s"
	defaultBackup      = false
	defaultDown        = false
	defaultWeight      = 1
)

// NginxClient lets you access NGINX Plus API.
type NginxClient struct {
	apiEndpoint string
	httpClient  *http.Client
}

type versions []int

// UpstreamServer lets you configure HTTP upstreams.
type UpstreamServer struct {
	ID          int    `json:"id,omitempty"`
	Server      string `json:"server"`
	MaxConns    *int   `json:"max_conns,omitempty"`
	MaxFails    *int   `json:"max_fails,omitempty"`
	FailTimeout string `json:"fail_timeout,omitempty"`
	SlowStart   string `json:"slow_start,omitempty"`
	Route       string `json:"route,omitempty"`
	Backup      *bool  `json:"backup,omitempty"`
	Down        *bool  `json:"down,omitempty"`
	Drain       bool   `json:"drain,omitempty"`
	Weight      *int   `json:"weight,omitempty"`
	Service     string `json:"service,omitempty"`
}

// StreamUpstreamServer lets you configure Stream upstreams.
type StreamUpstreamServer struct {
	ID          int    `json:"id,omitempty"`
	Server      string `json:"server"`
	MaxConns    *int   `json:"max_conns,omitempty"`
	MaxFails    *int   `json:"max_fails,omitempty"`
	FailTimeout string `json:"fail_timeout,omitempty"`
	SlowStart   string `json:"slow_start,omitempty"`
	Backup      *bool  `json:"backup,omitempty"`
	Down        *bool  `json:"down,omitempty"`
	Weight      *int   `json:"weight,omitempty"`
	Service     string `json:"service,omitempty"`
}

type apiErrorResponse struct {
	Error     apiError
	RequestID string `json:"request_id"`
	Href      string
}

func (resp *apiErrorResponse) toString() string {
	return fmt.Sprintf("error.status=%v; error.text=%v; error.code=%v; request_id=%v; href=%v",
		resp.Error.Status, resp.Error.Text, resp.Error.Code, resp.RequestID, resp.Href)
}

type apiError struct {
	Status int
	Text   string
	Code   string
}

type internalError struct {
	apiError
	err string
}

// Error allows internalError to match the Error interface.
func (internalError *internalError) Error() string {
	return internalError.err
}

// Wrap is a way of including current context while preserving previous error information,
// similar to `return fmt.Errorf("error doing foo, err: %v", err)` but for our internalError type.
func (internalError *internalError) Wrap(err string) *internalError {
	internalError.err = fmt.Sprintf("%v. %v", err, internalError.err)
	return internalError
}

// Stats represents NGINX Plus stats fetched from the NGINX Plus API.
// https://nginx.org/en/docs/http/ngx_http_api_module.html
type Stats struct {
	NginxInfo         NginxInfo
	Connections       Connections
	HTTPRequests      HTTPRequests
	SSL               SSL
	ServerZones       ServerZones
	Upstreams         Upstreams
	StreamServerZones StreamServerZones
	StreamUpstreams   StreamUpstreams
	StreamZoneSync    *StreamZoneSync
	LocationZones     LocationZones
	Resolvers         Resolvers
}

// NginxInfo contains general information about NGINX Plus.
type NginxInfo struct {
	Version         string
	Build           string
	Address         string
	Generation      uint64
	LoadTimestamp   string `json:"load_timestamp"`
	Timestamp       string
	ProcessID       uint64 `json:"pid"`
	ParentProcessID uint64 `json:"ppid"`
}

// Connections represents connection related stats.
type Connections struct {
	Accepted uint64
	Dropped  uint64
	Active   uint64
	Idle     uint64
}

// HTTPRequests represents HTTP request related stats.
type HTTPRequests struct {
	Total   uint64
	Current uint64
}

// SSL represents SSL related stats.
type SSL struct {
	Handshakes       uint64
	HandshakesFailed uint64 `json:"handshakes_failed"`
	SessionReuses    uint64 `json:"session_reuses"`
}

// ServerZones is map of server zone stats by zone name
type ServerZones map[string]ServerZone

// ServerZone represents server zone related stats.
type ServerZone struct {
	Processing uint64
	Requests   uint64
	Responses  Responses
	Discarded  uint64
	Received   uint64
	Sent       uint64
}

// StreamServerZones is map of stream server zone stats by zone name.
type StreamServerZones map[string]StreamServerZone

// StreamServerZone represents stream server zone related stats.
type StreamServerZone struct {
	Processing  uint64
	Connections uint64
	Sessions    Sessions
	Discarded   uint64
	Received    uint64
	Sent        uint64
}

// StreamZoneSync represents the sync information per each shared memory zone and the sync information per node in a cluster
type StreamZoneSync struct {
	Zones  map[string]SyncZone
	Status StreamZoneSyncStatus
}

// SyncZone represents the synchronization status of a shared memory zone
type SyncZone struct {
	RecordsPending uint64 `json:"records_pending"`
	RecordsTotal   uint64 `json:"records_total"`
}

// StreamZoneSyncStatus represents the status of a shared memory zone
type StreamZoneSyncStatus struct {
	BytesIn     uint64 `json:"bytes_in"`
	MsgsIn      uint64 `json:"msgs_in"`
	MsgsOut     uint64 `json:"msgs_out"`
	BytesOut    uint64 `json:"bytes_out"`
	NodesOnline uint64 `json:"nodes_online"`
}

// Responses represents HTTP response related stats.
type Responses struct {
	Responses1xx uint64 `json:"1xx"`
	Responses2xx uint64 `json:"2xx"`
	Responses3xx uint64 `json:"3xx"`
	Responses4xx uint64 `json:"4xx"`
	Responses5xx uint64 `json:"5xx"`
	Total        uint64
}

// Sessions represents stream session related stats.
type Sessions struct {
	Sessions2xx uint64 `json:"2xx"`
	Sessions4xx uint64 `json:"4xx"`
	Sessions5xx uint64 `json:"5xx"`
	Total       uint64
}

// Upstreams is a map of upstream stats by upstream name.
type Upstreams map[string]Upstream

// Upstream represents upstream related stats.
type Upstream struct {
	Peers      []Peer
	Keepalives int
	Zombies    int
	Zone       string
	Queue      Queue
}

// StreamUpstreams is a map of stream upstream stats by upstream name.
type StreamUpstreams map[string]StreamUpstream

// StreamUpstream represents stream upstream related stats.
type StreamUpstream struct {
	Peers   []StreamPeer
	Zombies int
	Zone    string
}

// Queue represents queue related stats for an upstream.
type Queue struct {
	Size      int
	MaxSize   int `json:"max_size"`
	Overflows uint64
}

// Peer represents peer (upstream server) related stats.
type Peer struct {
	ID           int
	Server       string
	Service      string
	Name         string
	Backup       bool
	Weight       int
	State        string
	Active       uint64
	MaxConns     int `json:"max_conns"`
	Requests     uint64
	Responses    Responses
	Sent         uint64
	Received     uint64
	Fails        uint64
	Unavail      uint64
	HealthChecks HealthChecks `json:"health_checks"`
	Downtime     uint64
	Downstart    string
	Selected     string
	HeaderTime   uint64 `json:"header_time"`
	ResponseTime uint64 `json:"response_time"`
}

// StreamPeer represents peer (stream upstream server) related stats.
type StreamPeer struct {
	ID            int
	Server        string
	Service       string
	Name          string
	Backup        bool
	Weight        int
	State         string
	Active        uint64
	MaxConns      int `json:"max_conns"`
	Connections   uint64
	ConnectTime   int    `json:"connect_time"`
	FirstByteTime int    `json:"first_byte_time"`
	ResponseTime  uint64 `json:"response_time"`
	Sent          uint64
	Received      uint64
	Fails         uint64
	Unavail       uint64
	HealthChecks  HealthChecks `json:"health_checks"`
	Downtime      uint64
	Downstart     string
	Selected      string
}

// HealthChecks represents health check related stats for a peer.
type HealthChecks struct {
	Checks     uint64
	Fails      uint64
	Unhealthy  uint64
	LastPassed bool `json:"last_passed"`
}

// LocationZones represents location_zones related stats
type LocationZones map[string]LocationZone

// Resolvers represents resolvers related stats
type Resolvers map[string]Resolver

// LocationZone represents location_zones related stats
type LocationZone struct {
	Requests  int64
	Responses Responses
	Discarded int64
	Received  int64
	Sent      int64
}

// Resolver represents resolvers related stats
type Resolver struct {
	Requests  ResolverRequests  `json:"requests"`
	Responses ResolverResponses `json:"responses"`
}

// ResolverRequests represents resolver requests
type ResolverRequests struct {
	Name int64
	Srv  int64
	Addr int64
}

// ResolverResponses represents resolver responses
type ResolverResponses struct {
	Noerror  int64
	Formerr  int64
	Servfail int64
	Nxdomain int64
	Notimp   int64
	Refused  int64
	Timedout int64
	Unknown  int64
}

// NewNginxClient creates an NginxClient.
func NewNginxClient(httpClient *http.Client, apiEndpoint string) (*NginxClient, error) {
	versions, err := getAPIVersions(httpClient, apiEndpoint)
	if err != nil {
		return nil, fmt.Errorf("error accessing the API: %v", err)
	}

	found := false
	for _, v := range *versions {
		if v == APIVersion {
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("API version %v of the client is not supported by API versions of NGINX Plus: %v", APIVersion, *versions)
	}

	return &NginxClient{
		apiEndpoint: apiEndpoint,
		httpClient:  httpClient,
	}, nil
}

func getAPIVersions(httpClient *http.Client, endpoint string) (*versions, error) {
	resp, err := httpClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("%v is not accessible: %v", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v is not accessible: expected %v response, got %v", endpoint, http.StatusOK, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error while reading body of the response: %v", err)
	}

	var vers versions
	err = json.Unmarshal(body, &vers)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling versions, got %q response: %v", string(body), err)
	}

	return &vers, nil
}

func createResponseMismatchError(respBody io.ReadCloser) *internalError {
	apiErrResp, err := readAPIErrorResponse(respBody)
	if err != nil {
		return &internalError{
			err: fmt.Sprintf("failed to read the response body: %v", err),
		}
	}

	return &internalError{
		err:      apiErrResp.toString(),
		apiError: apiErrResp.Error,
	}
}

func readAPIErrorResponse(respBody io.ReadCloser) (*apiErrorResponse, error) {
	body, err := ioutil.ReadAll(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response body: %v", err)
	}

	var apiErr apiErrorResponse
	err = json.Unmarshal(body, &apiErr)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling apiErrorResponse: got %q response: %v", string(body), err)
	}

	return &apiErr, nil
}

// CheckIfUpstreamExists checks if the upstream exists in NGINX. If the upstream doesn't exist, it returns the error.
func (client *NginxClient) CheckIfUpstreamExists(upstream string) error {
	_, err := client.GetHTTPServers(upstream)
	return err
}

// GetHTTPServers returns the servers of the upstream from NGINX.
func (client *NginxClient) GetHTTPServers(upstream string) ([]UpstreamServer, error) {
	path := fmt.Sprintf("http/upstreams/%v/servers", upstream)

	var servers []UpstreamServer
	err := client.get(path, &servers)
	if err != nil {
		return nil, fmt.Errorf("failed to get the HTTP servers of upstream %v: %v", upstream, err)
	}

	return servers, nil
}

// AddHTTPServer adds the server to the upstream.
func (client *NginxClient) AddHTTPServer(upstream string, server UpstreamServer) error {
	id, err := client.getIDOfHTTPServer(upstream, server.Server)
	if err != nil {
		return fmt.Errorf("failed to add %v server to %v upstream: %v", server.Server, upstream, err)
	}
	if id != -1 {
		return fmt.Errorf("failed to add %v server to %v upstream: server already exists", server.Server, upstream)
	}

	path := fmt.Sprintf("http/upstreams/%v/servers/", upstream)
	err = client.post(path, &server)
	if err != nil {
		return fmt.Errorf("failed to add %v server to %v upstream: %v", server.Server, upstream, err)
	}

	return nil
}

// DeleteHTTPServer the server from the upstream.
func (client *NginxClient) DeleteHTTPServer(upstream string, server string) error {
	id, err := client.getIDOfHTTPServer(upstream, server)
	if err != nil {
		return fmt.Errorf("failed to remove %v server from  %v upstream: %v", server, upstream, err)
	}
	if id == -1 {
		return fmt.Errorf("failed to remove %v server from %v upstream: server doesn't exist", server, upstream)
	}

	path := fmt.Sprintf("http/upstreams/%v/servers/%v", upstream, id)
	err = client.delete(path, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to remove %v server from %v upstream: %v", server, upstream, err)
	}

	return nil
}

// UpdateHTTPServers updates the servers of the upstream.
// Servers that are in the slice, but don't exist in NGINX will be added to NGINX.
// Servers that aren't in the slice, but exist in NGINX, will be removed from NGINX.
// Servers that are in the slice and exist in NGINX, but have different parameters, will be updated.
func (client *NginxClient) UpdateHTTPServers(upstream string, servers []UpstreamServer) (added []UpstreamServer, deleted []UpstreamServer, updated []UpstreamServer, err error) {
	serversInNginx, err := client.GetHTTPServers(upstream)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %v", upstream, err)
	}

	// We assume port 80 if no port is set for servers.
	var formattedServers []UpstreamServer
	for _, server := range servers {
		server.Server = addPortToServer(server.Server)
		formattedServers = append(formattedServers, server)
	}

	toAdd, toDelete, toUpdate := determineUpdates(formattedServers, serversInNginx)

	for _, server := range toAdd {
		err := client.AddHTTPServer(upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %v", upstream, err)
		}
	}

	for _, server := range toDelete {
		err := client.DeleteHTTPServer(upstream, server.Server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %v", upstream, err)
		}
	}

	for _, server := range toUpdate {
		err := client.UpdateHTTPServer(upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update servers of %v upstream: %v", upstream, err)
		}
	}

	return toAdd, toDelete, toUpdate, nil
}

// haveSameParameters checks if a given server has the same parameters as a server already present in NGINX. Order matters
func haveSameParameters(newServer UpstreamServer, serverNGX UpstreamServer) bool {
	newServer.ID = serverNGX.ID

	if serverNGX.MaxConns != nil && newServer.MaxConns == nil {
		newServer.MaxConns = &defaultMaxConns
	}

	if serverNGX.MaxFails != nil && newServer.MaxFails == nil {
		newServer.MaxFails = &defaultMaxFails
	}

	if serverNGX.FailTimeout != "" && newServer.FailTimeout == "" {
		newServer.FailTimeout = defaultFailTimeout
	}

	if serverNGX.SlowStart != "" && newServer.SlowStart == "" {
		newServer.SlowStart = defaultSlowStart
	}

	if serverNGX.Backup != nil && newServer.Backup == nil {
		newServer.Backup = &defaultBackup
	}

	if serverNGX.Down != nil && newServer.Down == nil {
		newServer.Down = &defaultDown
	}

	if serverNGX.Weight != nil && newServer.Weight == nil {
		newServer.Weight = &defaultWeight
	}

	return reflect.DeepEqual(newServer, serverNGX)
}

func determineUpdates(updatedServers []UpstreamServer, nginxServers []UpstreamServer) (toAdd []UpstreamServer, toRemove []UpstreamServer, toUpdate []UpstreamServer) {
	for _, server := range updatedServers {
		updateFound := false
		for _, serverNGX := range nginxServers {
			if server.Server == serverNGX.Server && !haveSameParameters(server, serverNGX) {
				server.ID = serverNGX.ID
				updateFound = true
				break
			}
		}
		if updateFound {
			toUpdate = append(toUpdate, server)
		}
	}

	for _, server := range updatedServers {
		found := false
		for _, serverNGX := range nginxServers {
			if server.Server == serverNGX.Server {
				found = true
				break
			}
		}
		if !found {
			toAdd = append(toAdd, server)
		}
	}

	for _, serverNGX := range nginxServers {
		found := false
		for _, server := range updatedServers {
			if serverNGX.Server == server.Server {
				found = true
				break
			}
		}
		if !found {
			toRemove = append(toRemove, serverNGX)
		}
	}

	return
}

func (client *NginxClient) getIDOfHTTPServer(upstream string, name string) (int, error) {
	servers, err := client.GetHTTPServers(upstream)
	if err != nil {
		return -1, fmt.Errorf("error getting id of server %v of upstream %v: %v", name, upstream, err)
	}

	for _, s := range servers {
		if s.Server == name {
			return s.ID, nil
		}
	}

	return -1, nil
}

func (client *NginxClient) get(path string, data interface{}) error {
	url := fmt.Sprintf("%v/%v/%v", client.apiEndpoint, APIVersion, path)
	resp, err := client.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get %v: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return createResponseMismatchError(resp.Body).Wrap(fmt.Sprintf(
			"expected %v response, got %v",
			http.StatusOK, resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read the response body: %v", err)
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return fmt.Errorf("error unmarshaling response %q: %v", string(body), err)
	}
	return nil
}

func (client *NginxClient) post(path string, input interface{}) error {
	url := fmt.Sprintf("%v/%v/%v", client.apiEndpoint, APIVersion, path)

	jsonInput, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshall input: %v", err)
	}

	resp, err := client.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonInput))
	if err != nil {
		return fmt.Errorf("failed to post %v: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return createResponseMismatchError(resp.Body).Wrap(fmt.Sprintf(
			"expected %v response, got %v",
			http.StatusCreated, resp.StatusCode))
	}

	return nil
}

func (client *NginxClient) delete(path string, expectedStatusCode int) error {
	path = fmt.Sprintf("%v/%v/%v/", client.apiEndpoint, APIVersion, path)

	req, err := http.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create a delete request: %v", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatusCode {
		return createResponseMismatchError(resp.Body).Wrap(fmt.Sprintf(
			"failed to complete delete request: expected %v response, got %v",
			expectedStatusCode, resp.StatusCode))
	}
	return nil
}

func (client *NginxClient) patch(path string, input interface{}, expectedStatusCode int) error {
	path = fmt.Sprintf("%v/%v/%v/", client.apiEndpoint, APIVersion, path)

	jsonInput, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshall input: %v", err)
	}

	req, err := http.NewRequest(http.MethodPatch, path, bytes.NewBuffer(jsonInput))
	if err != nil {
		return fmt.Errorf("failed to create a patch request: %v", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create patch request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatusCode {
		return createResponseMismatchError(resp.Body).Wrap(fmt.Sprintf(
			"failed to complete patch request: expected %v response, got %v",
			expectedStatusCode, resp.StatusCode))
	}
	return nil
}

// CheckIfStreamUpstreamExists checks if the stream upstream exists in NGINX. If the upstream doesn't exist, it returns the error.
func (client *NginxClient) CheckIfStreamUpstreamExists(upstream string) error {
	_, err := client.GetStreamServers(upstream)
	return err
}

// GetStreamServers returns the stream servers of the upstream from NGINX.
func (client *NginxClient) GetStreamServers(upstream string) ([]StreamUpstreamServer, error) {
	path := fmt.Sprintf("stream/upstreams/%v/servers", upstream)

	var servers []StreamUpstreamServer
	err := client.get(path, &servers)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream servers of upstream server %v: %v", upstream, err)
	}
	return servers, nil
}

// AddStreamServer adds the stream server to the upstream.
func (client *NginxClient) AddStreamServer(upstream string, server StreamUpstreamServer) error {
	id, err := client.getIDOfStreamServer(upstream, server.Server)
	if err != nil {
		return fmt.Errorf("failed to add %v stream server to %v upstream: %v", server.Server, upstream, err)
	}
	if id != -1 {
		return fmt.Errorf("failed to add %v stream server to %v upstream: server already exists", server.Server, upstream)
	}

	path := fmt.Sprintf("stream/upstreams/%v/servers/", upstream)
	err = client.post(path, &server)
	if err != nil {
		return fmt.Errorf("failed to add %v stream server to %v upstream: %v", server.Server, upstream, err)
	}
	return nil
}

// DeleteStreamServer the server from the upstream.
func (client *NginxClient) DeleteStreamServer(upstream string, server string) error {
	id, err := client.getIDOfStreamServer(upstream, server)
	if err != nil {
		return fmt.Errorf("failed to remove %v stream server from  %v upstream: %v", server, upstream, err)
	}
	if id == -1 {
		return fmt.Errorf("failed to remove %v stream server from %v upstream: server doesn't exist", server, upstream)
	}

	path := fmt.Sprintf("stream/upstreams/%v/servers/%v", upstream, id)
	err = client.delete(path, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to remove %v stream server from %v upstream: %v", server, upstream, err)
	}
	return nil
}

// UpdateStreamServers updates the servers of the upstream.
// Servers that are in the slice, but don't exist in NGINX will be added to NGINX.
// Servers that aren't in the slice, but exist in NGINX, will be removed from NGINX.
// Servers that are in the slice and exist in NGINX, but have different parameters, will be updated.
func (client *NginxClient) UpdateStreamServers(upstream string, servers []StreamUpstreamServer) (added []StreamUpstreamServer, deleted []StreamUpstreamServer, updated []StreamUpstreamServer, err error) {
	serversInNginx, err := client.GetStreamServers(upstream)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %v", upstream, err)
	}

	var formattedServers []StreamUpstreamServer
	for _, server := range servers {
		server.Server = addPortToServer(server.Server)
		formattedServers = append(formattedServers, server)
	}

	toAdd, toDelete, toUpdate := determineStreamUpdates(formattedServers, serversInNginx)

	for _, server := range toAdd {
		err := client.AddStreamServer(upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %v", upstream, err)
		}
	}

	for _, server := range toDelete {
		err := client.DeleteStreamServer(upstream, server.Server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %v", upstream, err)
		}
	}

	for _, server := range toUpdate {
		err := client.UpdateStreamServer(upstream, server)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to update stream servers of %v upstream: %v", upstream, err)
		}
	}

	return toAdd, toDelete, toUpdate, nil
}

func (client *NginxClient) getIDOfStreamServer(upstream string, name string) (int, error) {
	servers, err := client.GetStreamServers(upstream)
	if err != nil {
		return -1, fmt.Errorf("error getting id of stream server %v of upstream %v: %v", name, upstream, err)
	}

	for _, s := range servers {
		if s.Server == name {
			return s.ID, nil
		}
	}

	return -1, nil
}

// haveSameParametersForStream checks if a given server has the same parameters as a server already present in NGINX. Order matters
func haveSameParametersForStream(newServer StreamUpstreamServer, serverNGX StreamUpstreamServer) bool {
	newServer.ID = serverNGX.ID
	if serverNGX.MaxConns != nil && newServer.MaxConns == nil {
		newServer.MaxConns = &defaultMaxConns
	}

	if serverNGX.MaxFails != nil && newServer.MaxFails == nil {
		newServer.MaxFails = &defaultMaxFails
	}

	if serverNGX.FailTimeout != "" && newServer.FailTimeout == "" {
		newServer.FailTimeout = defaultFailTimeout
	}

	if serverNGX.SlowStart != "" && newServer.SlowStart == "" {
		newServer.SlowStart = defaultSlowStart
	}

	if serverNGX.Backup != nil && newServer.Backup == nil {
		newServer.Backup = &defaultBackup
	}

	if serverNGX.Down != nil && newServer.Down == nil {
		newServer.Down = &defaultDown
	}

	if serverNGX.Weight != nil && newServer.Weight == nil {
		newServer.Weight = &defaultWeight
	}

	return reflect.DeepEqual(newServer, serverNGX)
}

func determineStreamUpdates(updatedServers []StreamUpstreamServer, nginxServers []StreamUpstreamServer) (toAdd []StreamUpstreamServer, toRemove []StreamUpstreamServer, toUpdate []StreamUpstreamServer) {
	for _, server := range updatedServers {
		updateFound := false
		for _, serverNGX := range nginxServers {
			if server.Server == serverNGX.Server && !haveSameParametersForStream(server, serverNGX) {
				server.ID = serverNGX.ID
				updateFound = true
				break
			}
		}
		if updateFound {
			toUpdate = append(toUpdate, server)
		}
	}

	for _, server := range updatedServers {
		found := false
		for _, serverNGX := range nginxServers {
			if server.Server == serverNGX.Server {
				found = true
				break
			}
		}
		if !found {
			toAdd = append(toAdd, server)
		}
	}

	for _, serverNGX := range nginxServers {
		found := false
		for _, server := range updatedServers {
			if serverNGX.Server == server.Server {
				found = true
				break
			}
		}
		if !found {
			toRemove = append(toRemove, serverNGX)
		}
	}

	return
}

// GetStats gets connection, request, ssl, zone, stream zone, upstream and stream upstream related stats from the NGINX Plus API.
func (client *NginxClient) GetStats() (*Stats, error) {
	info, err := client.getNginxInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats %v", err)
	}

	cons, err := client.getConnections()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	requests, err := client.getHTTPRequests()
	if err != nil {
		return nil, fmt.Errorf("Failed to get stats: %v", err)
	}

	ssl, err := client.getSSL()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	zones, err := client.getServerZones()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	upstreams, err := client.getUpstreams()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	streamZones, err := client.getStreamServerZones()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	streamUpstreams, err := client.getStreamUpstreams()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	streamZoneSync, err := client.getStreamZoneSync()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	locationZones, err := client.getLocationZones()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	resolvers, err := client.getResolvers()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %v", err)
	}

	return &Stats{
		NginxInfo:         *info,
		Connections:       *cons,
		HTTPRequests:      *requests,
		SSL:               *ssl,
		ServerZones:       *zones,
		StreamServerZones: *streamZones,
		Upstreams:         *upstreams,
		StreamUpstreams:   *streamUpstreams,
		StreamZoneSync:    streamZoneSync,
		LocationZones:     *locationZones,
		Resolvers:         *resolvers,
	}, nil
}

func (client *NginxClient) getNginxInfo() (*NginxInfo, error) {
	var info NginxInfo
	err := client.get("nginx", &info)
	if err != nil {
		return nil, fmt.Errorf("failed to get info: %v", err)
	}
	return &info, nil
}

func (client *NginxClient) getConnections() (*Connections, error) {
	var cons Connections
	err := client.get("connections", &cons)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %v", err)
	}
	return &cons, nil
}

func (client *NginxClient) getHTTPRequests() (*HTTPRequests, error) {
	var requests HTTPRequests
	err := client.get("http/requests", &requests)
	if err != nil {
		return nil, fmt.Errorf("failed to get http requests: %v", err)
	}
	return &requests, nil
}

func (client *NginxClient) getSSL() (*SSL, error) {
	var ssl SSL
	err := client.get("ssl", &ssl)
	if err != nil {
		return nil, fmt.Errorf("failed to get ssl: %v", err)
	}
	return &ssl, nil
}

func (client *NginxClient) getServerZones() (*ServerZones, error) {
	var zones ServerZones
	err := client.get("http/server_zones", &zones)
	if err != nil {
		return nil, fmt.Errorf("failed to get server zones: %v", err)
	}
	return &zones, err
}

func (client *NginxClient) getStreamServerZones() (*StreamServerZones, error) {
	var zones StreamServerZones
	err := client.get("stream/server_zones", &zones)
	if err != nil {
		if err, ok := err.(*internalError); ok {
			if err.Code == pathNotFoundCode {
				return &zones, nil
			}
		}
		return nil, fmt.Errorf("failed to get stream server zones: %v", err)
	}
	return &zones, err
}

func (client *NginxClient) getUpstreams() (*Upstreams, error) {
	var upstreams Upstreams
	err := client.get("http/upstreams", &upstreams)
	if err != nil {
		return nil, fmt.Errorf("failed to get upstreams: %v", err)
	}
	return &upstreams, nil
}

func (client *NginxClient) getStreamUpstreams() (*StreamUpstreams, error) {
	var upstreams StreamUpstreams
	err := client.get("stream/upstreams", &upstreams)
	if err != nil {
		if err, ok := err.(*internalError); ok {
			if err.Code == pathNotFoundCode {
				return &upstreams, nil
			}
		}
		return nil, fmt.Errorf("failed to get stream upstreams: %v", err)
	}
	return &upstreams, nil
}

func (client *NginxClient) getStreamZoneSync() (*StreamZoneSync, error) {
	var streamZoneSync StreamZoneSync
	err := client.get("stream/zone_sync", &streamZoneSync)
	if err != nil {
		if err, ok := err.(*internalError); ok {
			if err.Code == pathNotFoundCode {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("failed to get stream zone sync: %v", err)
	}

	return &streamZoneSync, err
}

func (client *NginxClient) getLocationZones() (*LocationZones, error) {
	var locationZones LocationZones
	err := client.get("http/location_zones", &locationZones)
	if err != nil {
		return nil, fmt.Errorf("failed to get location zones: %v", err)
	}

	return &locationZones, err
}

func (client *NginxClient) getResolvers() (*Resolvers, error) {
	var resolvers Resolvers
	err := client.get("resolvers", &resolvers)
	if err != nil {
		return nil, fmt.Errorf("failed to get resolvers: %v", err)
	}

	return &resolvers, err
}

// KeyValPairs are the key-value pairs stored in a zone.
type KeyValPairs map[string]string

// KeyValPairsByZone are the KeyValPairs for all zones, by zone name.
type KeyValPairsByZone map[string]KeyValPairs

// GetKeyValPairs fetches key/value pairs for a given HTTP zone.
func (client *NginxClient) GetKeyValPairs(zone string) (KeyValPairs, error) {
	return client.getKeyValPairs(zone, httpContext)
}

// GetStreamKeyValPairs fetches key/value pairs for a given Stream zone.
func (client *NginxClient) GetStreamKeyValPairs(zone string) (KeyValPairs, error) {
	return client.getKeyValPairs(zone, streamContext)
}

func (client *NginxClient) getKeyValPairs(zone string, stream bool) (KeyValPairs, error) {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return nil, fmt.Errorf("zone required")
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	var keyValPairs KeyValPairs
	err := client.get(path, &keyValPairs)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyvals for %v/%v zone: %v", base, zone, err)
	}
	return keyValPairs, nil
}

// GetAllKeyValPairs fetches all key/value pairs for all HTTP zones.
func (client *NginxClient) GetAllKeyValPairs() (KeyValPairsByZone, error) {
	return client.getAllKeyValPairs(httpContext)
}

// GetAllStreamKeyValPairs fetches all key/value pairs for all Stream zones.
func (client *NginxClient) GetAllStreamKeyValPairs() (KeyValPairsByZone, error) {
	return client.getAllKeyValPairs(streamContext)
}

func (client *NginxClient) getAllKeyValPairs(stream bool) (KeyValPairsByZone, error) {
	base := "http"
	if stream {
		base = "stream"
	}

	path := fmt.Sprintf("%v/keyvals", base)
	var keyValPairsByZone KeyValPairsByZone
	err := client.get(path, &keyValPairsByZone)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyvals for all %v zones: %v", base, err)
	}
	return keyValPairsByZone, nil
}

// AddKeyValPair adds a new key/value pair to a given HTTP zone.
func (client *NginxClient) AddKeyValPair(zone string, key string, val string) error {
	return client.addKeyValPair(zone, key, val, httpContext)
}

// AddStreamKeyValPair adds a new key/value pair to a given Stream zone.
func (client *NginxClient) AddStreamKeyValPair(zone string, key string, val string) error {
	return client.addKeyValPair(zone, key, val, streamContext)
}

func (client *NginxClient) addKeyValPair(zone string, key string, val string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return fmt.Errorf("zone required")
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	input := KeyValPairs{key: val}
	err := client.post(path, &input)
	if err != nil {
		return fmt.Errorf("failed to add key value pair for %v/%v zone: %v", base, zone, err)
	}
	return nil
}

// ModifyKeyValPair modifies the value of an existing key in a given HTTP zone.
func (client *NginxClient) ModifyKeyValPair(zone string, key string, val string) error {
	return client.modifyKeyValPair(zone, key, val, httpContext)
}

// ModifyStreamKeyValPair modifies the value of an existing key in a given Stream zone.
func (client *NginxClient) ModifyStreamKeyValPair(zone string, key string, val string) error {
	return client.modifyKeyValPair(zone, key, val, streamContext)
}

func (client *NginxClient) modifyKeyValPair(zone string, key string, val string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return fmt.Errorf("zone required")
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	input := KeyValPairs{key: val}
	err := client.patch(path, &input, http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed to update key value pair for %v/%v zone: %v", base, zone, err)
	}
	return nil
}

// DeleteKeyValuePair deletes the key/value pair for a key in a given HTTP zone.
func (client *NginxClient) DeleteKeyValuePair(zone string, key string) error {
	return client.deleteKeyValuePair(zone, key, httpContext)
}

// DeleteStreamKeyValuePair deletes the key/value pair for a key in a given Stream zone.
func (client *NginxClient) DeleteStreamKeyValuePair(zone string, key string) error {
	return client.deleteKeyValuePair(zone, key, streamContext)
}

// To delete a key/value pair you set the value to null via the API,
// then NGINX+ will delete the key.
func (client *NginxClient) deleteKeyValuePair(zone string, key string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return fmt.Errorf("zone required")
	}

	// map[string]string can't have a nil value so we use a different type here.
	keyval := make(map[string]interface{})
	keyval[key] = nil

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	err := client.patch(path, &keyval, http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed to remove key values pair for %v/%v zone: %v", base, zone, err)
	}
	return nil
}

// DeleteKeyValPairs deletes all the key-value pairs in a given HTTP zone.
func (client *NginxClient) DeleteKeyValPairs(zone string) error {
	return client.deleteKeyValPairs(zone, httpContext)
}

// DeleteStreamKeyValPairs deletes all the key-value pairs in a given Stream zone.
func (client *NginxClient) DeleteStreamKeyValPairs(zone string) error {
	return client.deleteKeyValPairs(zone, streamContext)
}

func (client *NginxClient) deleteKeyValPairs(zone string, stream bool) error {
	base := "http"
	if stream {
		base = "stream"
	}
	if zone == "" {
		return fmt.Errorf("zone required")
	}

	path := fmt.Sprintf("%v/keyvals/%v", base, zone)
	err := client.delete(path, http.StatusNoContent)
	if err != nil {
		return fmt.Errorf("failed to remove all key value pairs for %v/%v zone: %v", base, zone, err)
	}
	return nil
}

// UpdateHTTPServer updates the server of the upstream.
func (client *NginxClient) UpdateHTTPServer(upstream string, server UpstreamServer) error {
	path := fmt.Sprintf("http/upstreams/%v/servers/%v", upstream, server.ID)
	server.ID = 0
	err := client.patch(path, &server, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to update %v server to %v upstream: %v", server.Server, upstream, err)
	}

	return nil
}

// UpdateStreamServer updates the stream server of the upstream.
func (client *NginxClient) UpdateStreamServer(upstream string, server StreamUpstreamServer) error {
	path := fmt.Sprintf("stream/upstreams/%v/servers/%v", upstream, server.ID)
	server.ID = 0
	err := client.patch(path, &server, http.StatusOK)
	if err != nil {
		return fmt.Errorf("failed to update %v stream server to %v upstream: %v", server.Server, upstream, err)
	}

	return nil
}

func addPortToServer(server string) string {
	if len(strings.Split(server, ":")) == 2 {
		return server
	}

	if len(strings.Split(server, "]:")) == 2 {
		return server
	}

	if strings.HasPrefix(server, "unix:") {
		return server
	}

	return fmt.Sprintf("%v:%v", server, defaultServerPort)
}

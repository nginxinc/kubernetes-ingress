package plus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// APIVersion is a version of NGINX Plus API
const APIVersion = 2

// NginxClient lets you add/remove servers to/from NGINX Plus via its API
type NginxClient struct {
	apiEndpoint string
	httpClient  *http.Client
}

type peers struct {
	Peers []peer
}

type peer struct {
	ID     int
	Server string
}

type versions []int

type upstreamServer struct {
	Server string `json:"server"`
}

type apiErrorResponse struct {
	Path      string
	Method    string
	Error     apiError
	RequestID string `json:"request_id"`
	Href      string
}

func (resp *apiErrorResponse) toString() string {
	return fmt.Sprintf("path=%v; method=%v; error.status=%v; error.text=%v; error.code=%v; request_id=%v; href=%v",
		resp.Path, resp.Method, resp.Error.Status, resp.Error.Text, resp.Error.Code, resp.RequestID, resp.Href)
}

type apiError struct {
	Status int
	Text   string
	Code   string
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

func createResponseMismatchError(respBody io.ReadCloser, mainErr error) error {
	apiErr, err := readAPIErrorResponse(respBody)
	if err != nil {
		return fmt.Errorf("%v; failed to read the response body: %v", mainErr, err)
	}

	return fmt.Errorf("%v; error: %v", mainErr, apiErr.toString())
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
	_, err := client.getUpstreamPeers(upstream)
	return err
}

func (client *NginxClient) getUpstreamPeers(upstream string) (*peers, error) {
	url := fmt.Sprintf("%v/%v/http/upstreams/%v", client.apiEndpoint, APIVersion, upstream)

	resp, err := client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the API to get upstream %v info: %v", upstream, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		mainErr := fmt.Errorf("upstream %v is invalid:  expected %v response, got %v", upstream, http.StatusOK, resp.StatusCode)
		return nil, createResponseMismatchError(resp.Body, mainErr)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response body with upstream %v info: %v", upstream, err)
	}

	var prs peers
	err = json.Unmarshal(body, &prs)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling upstream %v: got %q response: %v", upstream, string(body), err)
	}

	return &prs, nil
}

// AddHTTPServer adds the server to the upstream.
func (client *NginxClient) AddHTTPServer(upstream string, server string) error {
	id, err := client.getIDOfHTTPServer(upstream, server)

	if err != nil {
		return fmt.Errorf("failed to add %v server to %v upstream: %v", server, upstream, err)
	}
	if id != -1 {
		return fmt.Errorf("failed to add %v server to %v upstream: server already exists", server, upstream)
	}

	upsServer := upstreamServer{
		Server: server,
	}

	jsonServer, err := json.Marshal(upsServer)
	if err != nil {
		return fmt.Errorf("error marshalling upstream server %v: %v", upsServer, err)
	}

	url := fmt.Sprintf("%v/%v/http/upstreams/%v/servers/", client.apiEndpoint, APIVersion, upstream)

	resp, err := client.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonServer))

	if err != nil {
		return fmt.Errorf("failed to add %v server to %v upstream: %v", server, upstream, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		mainErr := fmt.Errorf("failed to add %v server to %v upstream: expected %v response, got %v",
			server, upstream, http.StatusCreated, resp.StatusCode)
		return createResponseMismatchError(resp.Body, mainErr)
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
		return fmt.Errorf("failed to remove %v server from %v upstream: server doesn't exists", server, upstream)
	}

	url := fmt.Sprintf("%v/%v/http/upstreams/%v/servers/%v", client.apiEndpoint, APIVersion, upstream, id)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create a request: %v", err)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove %v server from %v upstream: %v", server, upstream, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		mainErr := fmt.Errorf("failed to remove %v server from %v upstream: expected %v response, got %v",
			server, upstream, http.StatusOK, resp.StatusCode)
		return createResponseMismatchError(resp.Body, mainErr)
	}

	return nil
}

// UpdateHTTPServers updates the servers of the upstream.
// Servers that are in the slice, but don't exist in NGINX will be added to NGINX.
// Servers that aren't in the slice, but exist in NGINX, will be removed from NGINX.
func (client *NginxClient) UpdateHTTPServers(upstream string, servers []string) ([]string, []string, error) {
	serversInNginx, err := client.GetHTTPServers(upstream)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update servers of %v upstream: %v", upstream, err)
	}

	toAdd, toDelete := determineUpdates(servers, serversInNginx)

	for _, server := range toAdd {
		err := client.AddHTTPServer(upstream, server)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to update servers of %v upstream: %v", upstream, err)
		}
	}

	for _, server := range toDelete {
		err := client.DeleteHTTPServer(upstream, server)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to update servers of %v upstream: %v", upstream, err)
		}
	}

	return toAdd, toDelete, nil
}

func determineUpdates(updatedServers []string, nginxServers []string) (toAdd []string, toRemove []string) {
	for _, server := range updatedServers {
		found := false
		for _, serverNGX := range nginxServers {
			if server == serverNGX {
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
			if serverNGX == server {
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

// GetHTTPServers returns the servers of the upsteam from NGINX.
func (client *NginxClient) GetHTTPServers(upstream string) ([]string, error) {
	peers, err := client.getUpstreamPeers(upstream)
	if err != nil {
		return nil, fmt.Errorf("error getting servers of %v upstream: %v", upstream, err)
	}

	var servers []string
	for _, peer := range peers.Peers {
		servers = append(servers, peer.Server)
	}

	return servers, nil
}

func (client *NginxClient) getIDOfHTTPServer(upstream string, name string) (int, error) {
	peers, err := client.getUpstreamPeers(upstream)
	if err != nil {
		return -1, fmt.Errorf("error getting id of server %v of upstream %v: %v", name, upstream, err)
	}

	for _, p := range peers.Peers {
		if p.Server == name {
			return p.ID, nil
		}
	}

	return -1, nil
}

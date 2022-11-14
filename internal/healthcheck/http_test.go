package healthcheck_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/healthcheck"
	"github.com/nginxinc/nginx-plus-go-client/client"
)

func TestHealthCheckServer_ReturnsValidStatsAndValidHTTPCodeForAllPeersUpOnValidHostname(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/probe/bar.tea.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp := httptest.NewRecorder()

	h := healthcheck.API(getUpstreamsForHost, getUpstreamsFromNGINXAllUp)
	h.ServeHTTP(resp, req)

	if !cmp.Equal(http.StatusOK, resp.Code) {
		t.Error(cmp.Diff(http.StatusOK, resp.Code))
	}

	want := healthcheck.HostStats{
		Total:     3,
		Up:        3,
		Unhealthy: 0,
	}
	var got healthcheck.HostStats
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestHealthCheckServer_ReturnsValidStatsAndValidHTTPCodeForAllPeersDownOnValidHostname(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/probe/bar.tea.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp := httptest.NewRecorder()

	h := healthcheck.API(getUpstreamsForHost, getUpstreamsFromNGINXAllUnhealthy)
	h.ServeHTTP(resp, req)

	if !cmp.Equal(http.StatusServiceUnavailable, resp.Code) {
		t.Error(cmp.Diff(http.StatusServiceUnavailable, resp.Code))
	}

	want := healthcheck.HostStats{
		Total:     3,
		Up:        0,
		Unhealthy: 3,
	}

	var got healthcheck.HostStats
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestHealthCheckServer_ReturnsValidStatsAndCorrectHTTPStatusCodeForPartOfPeersDownOnValidHostname(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/probe/bar.tea.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp := httptest.NewRecorder()

	h := healthcheck.API(getUpstreamsForHost, getUpstreamsFromNGINXPartiallyUp)
	h.ServeHTTP(resp, req)

	if !cmp.Equal(http.StatusOK, resp.Code) {
		t.Error(cmp.Diff(http.StatusOK, resp.Code))
	}

	want := healthcheck.HostStats{
		Total:     3,
		Up:        1,
		Unhealthy: 2,
	}

	var got healthcheck.HostStats
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestHealthCheckServer_RespondsWithHTTPErrCodeOnNotExistingHostname(t *testing.T) {
	t.Parallel()

	// 'foo.mocha.org' represents not existing hostname
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/probe/foo.mocha.org", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp := httptest.NewRecorder()

	h := healthcheck.API(getUpstreamsForHost, getUpstreamsFromNGINXNotExistingHost)
	h.ServeHTTP(resp, req)

	if !cmp.Equal(http.StatusNotFound, resp.Code) {
		t.Error(cmp.Diff(http.StatusServiceUnavailable, resp.Code))
	}
}

func TestHealthCheckServer_RespondsWithCorrectHTTPStatusCodeOnErrorFromNGINXAPI(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/probe/foo.tea.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp := httptest.NewRecorder()

	h := healthcheck.API(getUpstreamsForHost, getUpstreamsFromNGINXErrorFromAPI)
	h.ServeHTTP(resp, req)

	if !cmp.Equal(http.StatusInternalServerError, resp.Code) {
		t.Error(cmp.Diff(http.StatusInternalServerError, resp.Code))
	}
}

// getUpstreamsForHost is a helper func faking response from IC.
func getUpstreamsForHost(host string) []string {
	upstreams := map[string][]string{
		"foo.tea.com": {"upstream1", "upstream2"},
		"bar.tea.com": {"upstream1"},
	}
	u, ok := upstreams[host]
	if !ok {
		return []string{}
	}
	return u
}

// getUpstreamsFromNGINXAllUP is a helper func used
// for faking response data from NGINX API. It responds
// with all upstreams and 'peers' in 'Up' state.
//
// Upstreams retrieved using NGINX API client:
// foo.tea.com -> upstream1, upstream2
// bar.tea.com -> upstream2
func getUpstreamsFromNGINXAllUp() (*client.Upstreams, error) {
	ups := client.Upstreams{
		"upstream1": client.Upstream{
			Peers: []client.Peer{
				{State: "Up"},
				{State: "Up"},
				{State: "Up"},
			},
		},
		"upstream2": client.Upstream{
			Peers: []client.Peer{
				{State: "Up"},
				{State: "Up"},
				{State: "Up"},
			},
		},
		"upstream3": client.Upstream{
			Peers: []client.Peer{
				{State: "Up"},
				{State: "Up"},
				{State: "Up"},
			},
		},
	}
	return &ups, nil
}

// getUpstreamsFromNGINXAllUnhealthy is a helper func used
// for faking response data from NGINX API. It responds
// with all upstreams and 'peers' in 'Down' (Unhealthy) state.
//
// Upstreams retrieved using NGINX API client:
// foo.tea.com -> upstream1, upstream2
// bar.tea.com -> upstream2
func getUpstreamsFromNGINXAllUnhealthy() (*client.Upstreams, error) {
	ups := client.Upstreams{
		"upstream1": client.Upstream{
			Peers: []client.Peer{
				{State: "Down"},
				{State: "Down"},
				{State: "Down"},
			},
		},
		"upstream2": client.Upstream{
			Peers: []client.Peer{
				{State: "Down"},
				{State: "Down"},
				{State: "Down"},
			},
		},
		"upstream3": client.Upstream{
			Peers: []client.Peer{
				{State: "Down"},
				{State: "Down"},
				{State: "Down"},
			},
		},
	}
	return &ups, nil
}

// getUpstreamsFromNGINXPartiallyUp is a helper func used
// for faking response data from NGINX API. It responds
// with some upstreams and 'peers' in 'Down' (Unhealthy) state,
// and some upstreams and 'peers' in 'Up' state.
//
// Upstreams retrieved using NGINX API client
// foo.tea.com -> upstream1, upstream2
// bar.tea.com -> upstream2
func getUpstreamsFromNGINXPartiallyUp() (*client.Upstreams, error) {
	ups := client.Upstreams{
		"upstream1": client.Upstream{
			Peers: []client.Peer{
				{State: "Down"},
				{State: "Down"},
				{State: "Up"},
			},
		},
		"upstream2": client.Upstream{
			Peers: []client.Peer{
				{State: "Down"},
				{State: "Down"},
				{State: "Up"},
			},
		},
		"upstream3": client.Upstream{
			Peers: []client.Peer{
				{State: "Down"},
				{State: "Up"},
				{State: "Down"},
			},
		},
	}
	return &ups, nil
}

// getUpstreamsFromNGINXNotExistingHost is a helper func used
// for faking response data from NGINX API. It responds
// with empty upstreams on a request for not existing host.
func getUpstreamsFromNGINXNotExistingHost() (*client.Upstreams, error) {
	ups := client.Upstreams{}
	return &ups, nil
}

// getUpstreamsFromNGINXErrorFromAPI is a helper func used
// for faking err response from NGINX API client.
func getUpstreamsFromNGINXErrorFromAPI() (*client.Upstreams, error) {
	return nil, errors.New("nginx api error")
}

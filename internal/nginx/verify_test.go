package nginx

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

type Transport struct{}

func (c Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(bytes.NewBufferString("42")),
		Header:     make(http.Header),
	}, nil
}

func getTestHTTPClient() *http.Client {
	ts := Transport{}
	tClient := &http.Client{
		Transport: ts,
	}
	return tClient
}

func TestVerifyClient(t *testing.T) {
	c := verifyClient{
		client:  getTestHTTPClient(),
		timeout: 25 * time.Millisecond,
	}

	configVersion, err := c.GetConfigVersion()
	if err != nil {
		t.Errorf("error getting config version: %v", err)
	}
	if configVersion != 42 {
		t.Errorf("got bad config version, expected 42 got %v", configVersion)
	}

	err = c.WaitForCorrectVersion(43)
	if err == nil {
		t.Error("expected error from WaitForCorrectVersion ")
	}
	err = c.WaitForCorrectVersion(42)
	if err != nil {
		t.Errorf("error waiting for config version: %v", err)
	}
}

func TestConfigWriter(t *testing.T) {
	cw, err := newVerifyConfigGenerator()
	if err != nil {
		t.Fatalf("error instantiating ConfigWriter: %v", err)
	}
	config, err := cw.GenerateVersionConfig(1, true)
	if err != nil {
		t.Errorf("error generating version config: %v", err)
	}
	if !strings.Contains(string(config), "configVersion") {
		t.Errorf("configVersion endpoint not set. config contents: %v", string(config))
	}
	if !strings.Contains(string(config), "opentracing off") {
		t.Errorf("opentracing directive missing when is enabled. config contents: %v", string(config))
	}
}

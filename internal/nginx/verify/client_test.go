package verify

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
)

type Transport struct {
}

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
	c := Client{
		client:     getTestHTTPClient(),
		maxRetries: 1,
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

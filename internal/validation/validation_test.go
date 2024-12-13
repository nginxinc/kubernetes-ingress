package validation

import (
	"strings"
	"testing"
)

func TestValidatePort_IsValidOnValidInput(t *testing.T) {
	t.Parallel()

	ports := []string{"1", "65535"}
	for _, p := range ports {
		if err := ValidatePort(p); err != nil {
			t.Error(err)
		}
	}
}

func TestValidatePort_ErrorsOnInvalidString(t *testing.T) {
	t.Parallel()

	if err := ValidatePort(""); err == nil {
		t.Error("want error, got nil")
	}
}

func TestValidatePort_ErrorsOnInvalidRange(t *testing.T) {
	t.Parallel()

	ports := []string{"0", "-1", "65536"}
	for _, p := range ports {
		if err := ValidatePort(p); err == nil {
			t.Error("want error, got nil")
		}
	}
}

func TestValidateHost(t *testing.T) {
	t.Parallel()
	// Positive test cases
	posHosts := []string{
		"10.10.1.1:443",
		"10.10.1.1",
		"123.112.224.43:443",
		"172.120.3.222",
		"localhost:80",
		"localhost",
		"myhost:54321",
		"myhost",
		"my-host:54321",
		"my-host",
		"dns.test.svc.cluster.local:8443",
		"cluster.local:8443",
		"product.example.com",
		"product.example.com:443",
	}

	// Negative test cases item, expected error message
	negHosts := [][]string{
		{"NotValid", "not a valid host"},
		{"-cluster.local:514", "not a valid host"},
		{"10.10.1.1:99999", "not a valid port number"},
		{"333.333.333.333", "not a valid host"},
	}

	for _, tCase := range posHosts {
		err := ValidateHost(tCase)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	}

	for _, nTCase := range negHosts {
		err := ValidateHost(nTCase[0])
		if err == nil {
			t.Errorf("got no error expected error containing '%s'", nTCase[1])
		} else {
			if !strings.Contains(err.Error(), nTCase[1]) {
				t.Errorf("got '%v', expected: '%s'", err, nTCase[1])
			}
		}
	}
}

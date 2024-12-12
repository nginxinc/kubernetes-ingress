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
		"10.10.1.1:514",
		"localhost:514",
		"dns.test.svc.cluster.local:514",
		"cluster.local:514",
		"dash-test.cluster.local:514",
		"product.example.com",
	}

	// Negative test cases item, expected error message
	negHosts := [][]string{
		{"NotValid", "invalid host: NotValid"},
		{"cluster.local", "invalid host: cluster.local"},
		{"-cluster.local:514", "invalid host: -cluster.local:514"},
		{"10.10.1.1:99999", "not a valid port number"},
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

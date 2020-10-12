package k8s

import (
	"strings"
	"testing"
)

func TestValidateAppProtectLogDestinationAnnotation(t *testing.T) {
	// Positive test cases
	var posDstAntns = []string{"stderr", "syslog:server=localhost:9000", "syslog:server=10.1.1.2:9000", "/var/log/ap.log"}

	// Negative test cases item, expected error message
	var negDstAntns = [][]string{
		{"stdout", "Log Destination did not follow format"},
		{"syslog:server=localhost:99999", "not a valid port number"},
		{"syslog:server=999.99.99.99:5678", "is not a valid ip address"},
	}

	for _, tCase := range posDstAntns {
		err := ValidateAppProtectLogDestinationAnnotation(tCase)
		if err != nil {
			t.Errorf("got %v expected nil", err)
		}
	}
	for _, nTCase := range negDstAntns {
		err := ValidateAppProtectLogDestinationAnnotation(nTCase[0])
		if err == nil {
			t.Errorf("got no error expected error containing %s", nTCase[1])
		} else {
			if !strings.Contains(err.Error(), nTCase[1]) {
				t.Errorf("got %v expected to contain: %s", err, nTCase[1])
			}
		}
	}
}

package validation

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestValidateAppProtectDosAccessLogDest(t *testing.T) {
	// Positive test cases
	posDstAntns := []string{"10.10.1.1:514"}

	// Negative test cases item, expected error message
	negDstAntns := [][]string{
		{"NotValid", "Error parsing App Protect Dos Access Log Dest config: Destination must follow format: <ip-address>:<port> Log Destination did not follow format"},
		{"10.10.1.1:99999", "not a valid port number"},
		{"999.99.99.99:5678", "is not a valid ip address"},
	}

	for _, tCase := range posDstAntns {
		err := ValidateAppProtectDosAccessLogDest(tCase)
		if err != nil {
			t.Errorf("got %v expected nil", err)
		}
	}

	for _, nTCase := range negDstAntns {
		err := ValidateAppProtectDosAccessLogDest(nTCase[0])
		if err == nil {
			t.Errorf("got no error expected error containing %s", nTCase[1])
		} else {
			if !strings.Contains(err.Error(), nTCase[1]) {
				t.Errorf("got %v expected to contain: %s", err, nTCase[1])
			}
		}
	}
}

func TestValidateAppProtectDosLogConf(t *testing.T) {
	tests := []struct {
		logConf   *unstructured.Unstructured
		expectErr bool
		msg       string
	}{
		{
			logConf: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"content": map[string]interface{}{},
						"filter":  map[string]interface{}{},
					},
				},
			},
			expectErr: false,
			msg:       "valid log conf",
		},
		{
			logConf: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"filter": map[string]interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid log conf with no content field",
		},
		{
			logConf: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"content": map[string]interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid log conf with no filter field",
		},
		{
			logConf: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"something": map[string]interface{}{
						"content": map[string]interface{}{},
						"filter":  map[string]interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid log conf with no spec field",
		},
	}

	for _, test := range tests {
		err := ValidateAppProtectDosLogConf(test.logConf)
		if test.expectErr && err == nil {
			t.Errorf("validateAppProtectDosLogConf() returned no error for the case of %s", test.msg)
		}
		if !test.expectErr && err != nil {
			t.Errorf("validateAppProtectDosLogConf() returned unexpected error %v for the case of %s", err, test.msg)
		}
	}
}

func TestValidateAppProtectDosPolicy(t *testing.T) {
	tests := []struct {
		policy    *unstructured.Unstructured
		expectErr bool
		msg       string
	}{
		{
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{},
				},
			},
			expectErr: false,
			msg:       "valid policy",
		},
		{
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"something": map[string]interface{}{},
				},
			},
			expectErr: true,
			msg:       "invalid policy with no spec field",
		},
	}

	for _, test := range tests {
		err := ValidateAppProtectDosPolicy(test.policy)
		if test.expectErr && err == nil {
			t.Errorf("validateAppProtectPolicy() returned no error for the case of %s", test.msg)
		}
		if !test.expectErr && err != nil {
			t.Errorf("validateAppProtectPolicy() returned unexpected error %v for the case of %s", err, test.msg)
		}
	}
}

func TestValidateAppProtectDosName(t *testing.T) {
	// Positive test cases
	posDstAntns := []string{"example.com"}

	// Negative test cases item, expected error message
	negDstAntns := [][]string{
		{"very very very very very very very very very very very very very very very very very very long Name", "App Protect Dos Name max length is 64"},
	}

	for _, tCase := range posDstAntns {
		err := ValidateAppProtectDosName(tCase)
		if err != nil {
			t.Errorf("got %v expected nil", err)
		}
	}

	for _, nTCase := range negDstAntns {
		err := ValidateAppProtectDosName(nTCase[0])
		if err == nil {
			t.Errorf("got no error expected error containing %s", nTCase[1])
		} else {
			if !strings.Contains(err.Error(), nTCase[1]) {
				t.Errorf("got %v expected to contain: %s", err, nTCase[1])
			}
		}
	}
}

func TestValidateAppProtectDosMonitor(t *testing.T) {
	// Positive test cases
	posDstAntns := []string{"example.com", "https://example.com/good_path"}

	// Negative test cases item, expected error message
	negDstAntns := [][]string{
		{"http://example.com/%", "App Protect Dos Monitor must have valid URL"},
	}

	for _, tCase := range posDstAntns {
		err := ValidateAppProtectDosMonitor(tCase)
		if err != nil {
			t.Errorf("got %v expected nil", err)
		}
	}

	for _, nTCase := range negDstAntns {
		err := ValidateAppProtectDosMonitor(nTCase[0])
		if err == nil {
			t.Errorf("got no error expected error containing %s", nTCase[1])
		} else {
			if !strings.Contains(err.Error(), nTCase[1]) {
				t.Errorf("got %v expected to contain: %s", err, nTCase[1])
			}
		}
	}
}

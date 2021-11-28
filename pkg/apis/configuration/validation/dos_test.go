package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/dos/v1beta1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestValidateDosProtectedResource(t *testing.T) {
	tests := []struct {
		protected *v1beta1.DosProtectedResource
		expectErr string
		msg       string
	}{
		{
			protected: &v1beta1.DosProtectedResource{},
			expectErr: "error validating DosProtectedResource:  missing value for field: name",
			msg:       "empty resource",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{},
			},
			expectErr: "error validating DosProtectedResource:  missing value for field: name",
			msg:       "empty spec",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
				},
			},
			expectErr: "error validating DosProtectedResource:  missing value for field: dosAccessLogDest",
			msg:       "only name specified",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "example.com",
					},
				},
			},
			expectErr: "error validating DosProtectedResource:  missing value for field: dosAccessLogDest",
			msg:       "name and apDosMonitor specified",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "exabad-$%^$-example.com",
					},
				},
			},
			expectErr: "error validating DosProtectedResource:  invalid field: apDosMonitor err: app Protect Dos Monitor must have valid URL",
			msg:       "invalid apDosMonitor specified",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "example.com",
					},
					DosAccessLogDest: "example.service.com:123",
				},
			},
			msg: "name, dosAccessLogDest and apDosMonitor specified",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "example.com",
					},
					DosAccessLogDest: "bad&$%^logdest",
				},
			},
			expectErr: "error validating DosProtectedResource:  invalid field: dosAccessLogDest err: invalid log destination: bad&$%^logdest, must follow format: <ip-address | localhost | dns name>:<port> or stderr",
			msg:       "invalid DosAccessLogDest specified",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "example.com",
					},
					DosAccessLogDest: "example.service.com:123",
					ApDosPolicy:      "ns/name",
				},
			},
			expectErr: "",
			msg:       "required fields and apdospolicy specified",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "example.com",
					},
					DosAccessLogDest: "example.service.com:123",
					ApDosPolicy:      "bad$%^name",
				},
			},
			expectErr: "error validating DosProtectedResource:  invalid field: apDosPolicy err: reference name is invalid: bad$%^name",
			msg:       "invalid apdospolicy specified",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "example.com",
					},
					DosAccessLogDest: "example.service.com:123",
					DosSecurityLog:   &v1beta1.DosSecurityLog{},
				},
			},
			expectErr: "error validating DosProtectedResource:  invalid field: dosSecurityLog/dosLogDest err: invalid log destination: , must follow format: <ip-address | localhost | dns name>:<port> or stderr",
			msg:       "empty DosSecurityLog specified",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "example.com",
					},
					DosAccessLogDest: "example.service.com:123",
					DosSecurityLog: &v1beta1.DosSecurityLog{
						DosLogDest: "service.org:123",
					},
				},
			},
			expectErr: "error validating DosProtectedResource:  invalid field: dosSecurityLog/apDosLogConf err: reference name is invalid: ",
			msg:       "DosSecurityLog with missing apDosLogConf",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "example.com",
					},
					DosAccessLogDest: "example.service.com:123",
					DosSecurityLog: &v1beta1.DosSecurityLog{
						DosLogDest:   "service.org:123",
						ApDosLogConf: "bad$%^$%name",
					},
				},
			},
			expectErr: "error validating DosProtectedResource:  invalid field: dosSecurityLog/apDosLogConf err: reference name is invalid: bad$%^$%name",
			msg:       "DosSecurityLog with invalid apDosLogConf",
		},
		{
			protected: &v1beta1.DosProtectedResource{
				Spec: v1beta1.DosProtectedResourceSpec{
					Name: "name",
					ApDosMonitor: &v1beta1.ApDosMonitor{
						Uri: "example.com",
					},
					DosAccessLogDest: "example.service.com:123",
					DosSecurityLog: &v1beta1.DosSecurityLog{
						DosLogDest:   "service.org:123",
						ApDosLogConf: "ns/name",
					},
				},
			},
			expectErr: "",
			msg:       "DosSecurityLog with valid apDosLogConf",
		},
	}

	for _, test := range tests {
		err := ValidateDosProtectedResource(test.protected)
		if err != nil {
			if test.expectErr == "" {
				t.Errorf("ValidateDosProtectedResource() returned unexpected error: '%v' for the case of: '%s'", err, test.msg)
				continue
			}
			if test.expectErr != err.Error() {
				t.Errorf("ValidateDosProtectedResource() returned error for the case of '%s', expected err: '%s' got err: '%s'", test.msg, test.expectErr, err.Error())
			}
		} else {
			if test.expectErr != "" {
				t.Errorf("ValidateDosProtectedResource() failed to return expected error: '%v' for the case of: '%s'", test.expectErr, test.msg)
			}
		}
	}
}

func TestValidateAppProtectDosAccessLogDest(t *testing.T) {
	// Positive test cases
	posDstAntns := []string{
		"10.10.1.1:514",
		"localhost:514",
		"dns.test.svc.cluster.local:514",
		"cluster.local:514",
		"dash-test.cluster.local:514",
	}

	// Negative test cases item, expected error message
	negDstAntns := [][]string{
		{"NotValid", "invalid log destination: NotValid, must follow format: <ip-address | localhost | dns name>:<port> or stderr"},
		{"cluster.local", "invalid log destination: cluster.local, must follow format: <ip-address | localhost | dns name>:<port> or stderr"},
		{"-cluster.local:514", "invalid log destination: -cluster.local:514, must follow format: <ip-address | localhost | dns name>:<port> or stderr"},
		{"10.10.1.1:99999", "not a valid port number"},
	}

	for _, tCase := range posDstAntns {
		err := validateAppProtectDosLogDest(tCase)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	}

	for _, nTCase := range negDstAntns {
		err := validateAppProtectDosLogDest(nTCase[0])
		if err == nil {
			t.Errorf("got no error expected error containing '%s'", nTCase[1])
		} else {
			if !strings.Contains(err.Error(), nTCase[1]) {
				t.Errorf("got '%v', expected: '%s'", err, nTCase[1])
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
	posDstAntns := []string{"example.com", "\\\"example.com\\\""}

	// Negative test cases item, expected error message
	negDstAntns := [][]string{
		{"very very very very very very very very very very very very very very very very very very long Name", fmt.Sprintf(`app Protect Dos Name max length is %v`, MaxNameLength)},
		{"example.com\\", "must have all '\"' (double quotes) escaped and must not end with an unescaped '\\' (backslash) (e.g. 'protected-object-one', regex used for validation is '([^\"\\\\]|\\\\.)*')"},
		{"\"example.com\"", "must have all '\"' (double quotes) escaped and must not end with an unescaped '\\' (backslash) (e.g. 'protected-object-one', regex used for validation is '([^\"\\\\]|\\\\.)*')"},
	}

	for _, tCase := range posDstAntns {
		err := validateAppProtectDosName(tCase)
		if err != nil {
			t.Errorf("got %v expected nil", err)
		}
	}

	for _, nTCase := range negDstAntns {
		err := validateAppProtectDosName(nTCase[0])
		if err == nil {
			t.Errorf("got no error expected error containing %s", nTCase[1])
		} else {
			if !strings.Contains(err.Error(), nTCase[1]) {
				t.Errorf("got '%v'\n expected: '%s'\n", err, nTCase[1])
			}
		}
	}
}

func TestValidateAppProtectDosMonitor(t *testing.T) {
	// Positive test cases
	posDstAntns := []v1beta1.ApDosMonitor{
		{
			Uri:      "example.com",
			Protocol: "http1",
			Timeout:  5,
		},
		{
			Uri:      "https://example.com/good_path",
			Protocol: "http2",
			Timeout:  10,
		},		
		{
			Uri:      "https://example.com/good_path",
			Protocol: "grpc",
			Timeout:  10,
		},
	}
	negDstAntns := []struct {
		apDosMonitor v1beta1.ApDosMonitor
		msg          string
	}{
		{
			apDosMonitor: v1beta1.ApDosMonitor{
				Uri:      "http://example.com/%",
				Protocol: "http1",
				Timeout:  5,
			},
			msg: "app Protect Dos Monitor must have valid URL",
		},
		{
			apDosMonitor: v1beta1.ApDosMonitor{
				Uri:      "http://example.com/\\",
				Protocol: "http1",
				Timeout:  5,
			},
			msg: "must have all '\"' (double quotes) escaped and must not end with an unescaped '\\' (backslash) (e.g. 'http://www.example.com', regex used for validation is '([^\"\\\\]|\\\\.)*')",
		},
		{
			apDosMonitor: v1beta1.ApDosMonitor{
				Uri:      "example.com",
				Protocol: "http3",
				Timeout:  5,
			},
			msg: "app Protect Dos Monitor Protocol must be: dosMonitorProtocol: Invalid value: \"http3\": 'http3' contains an invalid NGINX parameter. Accepted parameters are: http1, http2, grpc",
		},
	}

	for _, tCase := range posDstAntns {
		err := validateAppProtectDosMonitor(tCase)
		if err != nil {
			t.Errorf("got %v expected nil", err)
		}
	}

	for _, nTCase := range negDstAntns {
		err := validateAppProtectDosMonitor(nTCase.apDosMonitor)
		if err == nil {
			t.Errorf("got no error expected error containing %s", nTCase.msg)
		} else {
			if !strings.Contains(err.Error(), nTCase.msg) {
				t.Errorf("got %v expected to contain: %s", err, nTCase.msg)
			}
		}
	}
}

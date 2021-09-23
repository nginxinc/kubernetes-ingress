package appprotect_common

import (
	"reflect"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestValidateRequiredFields(t *testing.T) {
	tests := []struct {
		obj        *unstructured.Unstructured
		fieldsList [][]string
		expectErr  bool
		msg        string
	}{
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": map[string]interface{}{},
					"b": map[string]interface{}{},
				},
			},
			fieldsList: [][]string{{"a"}, {"b"}},
			expectErr:  false,
			msg:        "valid object with 2 fields",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": map[string]interface{}{},
				},
			},
			fieldsList: [][]string{{"a"}, {"b"}},
			expectErr:  true,
			msg:        "invalid object with a missing field",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": map[string]interface{}{},
					"x": map[string]interface{}{},
				},
			},
			fieldsList: [][]string{{"a"}, {"b"}},
			expectErr:  true,
			msg:        "invalid object with a wrong field",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": map[string]interface{}{
						"b": map[string]interface{}{},
					},
				},
			},
			fieldsList: [][]string{{"a", "b"}},
			expectErr:  false,
			msg:        "valid object with nested field",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": map[string]interface{}{
						"x": map[string]interface{}{},
					},
				},
			},
			fieldsList: [][]string{{"a", "b"}},
			expectErr:  true,
			msg:        "invalid object with a wrong nested field",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			fieldsList: nil,
			expectErr:  false,
			msg:        "valid object with no validation",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": "wrong-type", // must be map[string]interface{}
				},
			},
			fieldsList: [][]string{{"a"}},
			expectErr:  true,
			msg:        "invalid object with a field of wrong type",
		},
	}

	for _, test := range tests {
		err := ValidateRequiredFields(test.obj, test.fieldsList)
		if test.expectErr && err == nil {
			t.Errorf("ValidateRequiredFields() returned no error for the case of %s", test.msg)
		}
		if !test.expectErr && err != nil {
			t.Errorf("ValidateRequiredFields() returned unexpected error %v for the case of %s", err, test.msg)
		}
	}
}

func TestValidateRequiredSlices(t *testing.T) {
	tests := []struct {
		obj        *unstructured.Unstructured
		fieldsList [][]string
		expectErr  bool
		msg        string
	}{
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": []interface{}{},
					"b": []interface{}{},
				},
			},
			fieldsList: [][]string{{"a"}, {"b"}},
			expectErr:  false,
			msg:        "valid object with 2 fields",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": []interface{}{},
				},
			},
			fieldsList: [][]string{{"a"}, {"b"}},
			expectErr:  true,
			msg:        "invalid object with a field",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": []interface{}{},
					"x": []interface{}{},
				},
			},
			fieldsList: [][]string{{"a"}, {"b"}},
			expectErr:  true,
			msg:        "invalid object with a wrong field",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": map[string]interface{}{
						"b": []interface{}{},
					},
				},
			},
			fieldsList: [][]string{{"a", "b"}},
			expectErr:  false,
			msg:        "valid object with nested field",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": map[string]interface{}{
						"x": []interface{}{},
					},
				},
			},
			fieldsList: [][]string{{"a", "b"}},
			expectErr:  true,
			msg:        "invalid object with a wrong nested field",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			fieldsList: nil,
			expectErr:  false,
			msg:        "valid object with no validation",
		},
		{
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"a": "wrong-type", // must be [string]interface{}
				},
			},
			fieldsList: [][]string{{"a"}},
			expectErr:  true,
			msg:        "invalid object with a field of wrong type",
		},
	}

	for _, test := range tests {
		err := ValidateRequiredSlices(test.obj, test.fieldsList)
		if test.expectErr && err == nil {
			t.Errorf("ValidateRequiredSlices() returned no error for the case of %s", test.msg)
		}
		if !test.expectErr && err != nil {
			t.Errorf("ValidateRequiredSlices() returned unexpected error %v for the case of %s", err, test.msg)
		}
	}
}

func TestValidateAppProtectLogDestinationAnnotation(t *testing.T) {
	// Positive test cases
	posDstAntns := []string{"stderr", "syslog:server=localhost:9000", "syslog:server=10.1.1.2:9000"}

	// Negative test cases item, expected error message
	negDstAntns := [][]string{
		{"stdout", "Log Destination did not follow format"},
		{"syslog:server=localhost:99999", "not a valid port number"},
		{"syslog:server=999.99.99.99:5678", "is not a valid ip address"},
		{"/var/log/ap.log", "Error parsing App Protect Log config: Destination must follow format: syslog:server=<ip-address | localhost>:<port> or stderr"},
	}

	for _, tCase := range posDstAntns {
		err := ValidateAppProtectLogDestination(tCase)
		if err != nil {
			t.Errorf("got %v expected nil", err)
		}
	}
	for _, nTCase := range negDstAntns {
		err := ValidateAppProtectLogDestination(nTCase[0])
		if err == nil {
			t.Errorf("got no error expected error containing %s", nTCase[1])
		} else {
			if !strings.Contains(err.Error(), nTCase[1]) {
				t.Errorf("got %v expected to contain: %s", err, nTCase[1])
			}
		}
	}
}

func TestParseResourceReferenceAnnotation(t *testing.T) {
	tests := []struct {
		ns, antn, expected string
	}{
		{
			ns:       "default",
			antn:     "resource",
			expected: "default/resource",
		},
		{
			ns:       "default",
			antn:     "ns-1/resource",
			expected: "ns-1/resource",
		},
	}

	for _, test := range tests {
		result := ParseResourceReferenceAnnotation(test.ns, test.antn)
		if result != test.expected {
			t.Errorf("ParseResourceReferenceAnnotation(%q,%q) returned %q but expected %q", test.ns, test.antn, result, test.expected)
		}
	}
}

func TestGenNsName(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "default",
				"name":      "resource",
			},
		},
	}

	expected := "default/resource"

	result := GetNsName(obj)
	if result != expected {
		t.Errorf("GetNsName() returned %q but expected %q", result, expected)
	}
}

func TestParseResourceReferenceAnnotationList(t *testing.T) {
	namespace := "test_ns"
	tests := []struct {
		annotation string
		expected   []string
		msg        string
	}{
		{
			annotation: "test",
			expected:   []string{namespace + "/test"},
			msg:        "single resource no namespace",
		},
		{
			annotation: "different_ns/test",
			expected:   []string{"different_ns/test"},
			msg:        "single resource with namespace",
		},
		{
			annotation: "test,test1",
			expected:   []string{namespace + "/test", namespace + "/test1"},
			msg:        "multiple resource no namespace",
		},
		{
			annotation: "different_ns/test,different_ns/test1",
			expected:   []string{"different_ns/test", "different_ns/test1"},
			msg:        "multiple resource with namespaces",
		},
		{
			annotation: "different_ns/test,test1",
			expected:   []string{"different_ns/test", namespace + "/test1"},
			msg:        "multiple resource with mixed namespaces",
		},
	}
	for _, test := range tests {
		result := ParseResourceReferenceAnnotationList(namespace, test.annotation)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("Error in test case %s: got: %v, expected: %v", test.msg, result, test.expected)
		}
	}
}

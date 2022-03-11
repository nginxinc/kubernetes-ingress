package appprotectcommon

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseResourceReferenceAnnotation(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

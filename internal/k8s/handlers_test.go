package k8s

import (
	"errors"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestHasServicePortChanges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a      []v1.ServicePort
		b      []v1.ServicePort
		result bool
		reason string
	}{
		{
			[]v1.ServicePort{},
			[]v1.ServicePort{},
			false,
			"Empty should report no changes",
		},
		{
			[]v1.ServicePort{{
				Port: 80,
			}},
			[]v1.ServicePort{{
				Port: 8080,
			}},
			true,
			"Different Ports",
		},
		{
			[]v1.ServicePort{{
				Port: 80,
			}},
			[]v1.ServicePort{{
				Port: 80,
			}},
			false,
			"Same Ports",
		},
		{
			[]v1.ServicePort{{
				Name: "asdf",
				Port: 80,
			}},
			[]v1.ServicePort{{
				Name: "asdf",
				Port: 80,
			}},
			false,
			"Same Port and Name",
		},
		{
			[]v1.ServicePort{{
				Name: "foo",
				Port: 80,
			}},
			[]v1.ServicePort{{
				Name: "bar",
				Port: 80,
			}},
			true,
			"Different Name same Port",
		},
		{
			[]v1.ServicePort{{
				Name: "foo",
				Port: 8080,
			}},
			[]v1.ServicePort{{
				Name: "bar",
				Port: 80,
			}},
			true,
			"Different Name different Port",
		},
		{
			[]v1.ServicePort{{
				Name: "foo",
			}},
			[]v1.ServicePort{{
				Name: "fooo",
			}},
			true,
			"Very similar Name",
		},
		{
			[]v1.ServicePort{{
				Name: "asdf",
				Port: 80,
				TargetPort: intstr.IntOrString{
					IntVal: 80,
				},
			}},
			[]v1.ServicePort{{
				Name: "asdf",
				Port: 80,
				TargetPort: intstr.IntOrString{
					IntVal: 8080,
				},
			}},
			false,
			"TargetPort should be ignored",
		},
		{
			[]v1.ServicePort{{
				Name: "foo",
			}, {
				Name: "bar",
			}},
			[]v1.ServicePort{{
				Name: "foo",
			}, {
				Name: "bar",
			}},
			false,
			"Multiple same names",
		},
		{
			[]v1.ServicePort{{
				Name: "foo",
			}, {
				Name: "bar",
			}},
			[]v1.ServicePort{{
				Name: "foo",
			}, {
				Name: "bars",
			}},
			true,
			"Multiple different names",
		},
		{
			[]v1.ServicePort{{
				Name: "foo",
			}, {
				Port: 80,
			}},
			[]v1.ServicePort{{
				Port: 80,
			}, {
				Name: "foo",
			}},
			false,
			"Some names some ports",
		},
	}

	for _, c := range cases {
		if c.result != hasServicePortChanges(c.a, c.b) {
			t.Errorf("hasServicePortChanges returned %v, but expected %v for %q case", c.result, !c.result, c.reason)
		}
	}
}

func TestAreResourcesDifferent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		oldR, newR *unstructured.Unstructured
		expected   bool
		expectErr  error
		msg        string
	}{
		{
			oldR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": true, // wrong type
				},
			},
			newR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{},
				},
			},
			expected:  false,
			expectErr: errors.New(`.spec accessor error: true is of the type bool, expected map[string]interface{}`),
			msg:       "invalid old resource",
		},
		{
			oldR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{},
				},
			},
			newR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": true, // wrong type
				},
			},
			expected:  false,
			expectErr: errors.New(`.spec accessor error: true is of the type bool, expected map[string]interface{}`),
			msg:       "invalid new resource",
		},
		{
			oldR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{},
				},
			},
			newR: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			expected:  false,
			expectErr: errors.New(`spec has unexpected format`),
			msg:       "new resource with missing spec",
		},
		{
			oldR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"field": "a",
					},
				},
			},
			newR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"field": "a",
					},
				},
			},
			expected:  false,
			expectErr: nil,
			msg:       "equal resources",
		},
		{
			oldR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"field": "a",
					},
				},
			},
			newR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"field": "b",
					},
				},
			},
			expected:  true,
			expectErr: nil,
			msg:       "not equal resources",
		},
		{
			oldR: &unstructured.Unstructured{
				Object: map[string]interface{}{},
			},
			newR: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"field": "b",
					},
				},
			},
			expected:  true,
			expectErr: nil,
			msg:       "not equal resources with first resource missing spec",
		},
	}

	for _, test := range tests {
		result, err := areResourcesDifferent(test.oldR, test.newR)
		if result != test.expected {
			t.Errorf("areResourcesDifferent() returned %v but expected %v for the case of %s", result, test.expected, test.msg)
		}
		if test.expectErr != nil {
			if err == nil {
				t.Errorf("areResourcesDifferent() returned no error for the case of %s", test.msg)
			} else if test.expectErr.Error() != err.Error() {
				t.Errorf("areResourcesDifferent() returned an unexpected error '%v' for the case of %s", err, test.msg)
			}
		}
		if test.expectErr == nil && err != nil {
			t.Errorf("areResourcesDifferent() returned unexpected error %v for the case of %s", err, test.msg)
		}
	}
}

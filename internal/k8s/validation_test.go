package k8s

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateIngress(t *testing.T) {
	tests := []struct {
		ing            *networking.Ingress
		expectedErrors []string
		msg            string
	}{
		{
			ing: &networking.Ingress{
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host: "example.com",
						},
					},
				},
			},
			expectedErrors: nil,
			msg:            "valid input",
		},
		{
			ing: &networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{
						"nginx.org/mergeable-ingress-type": "invalid",
					},
				},
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host: "",
						},
					},
				},
			},
			expectedErrors: []string{
				`annotations.nginx.org/mergeable-ingress-type: Invalid value: "invalid": must be one of: 'master' or 'minion'`,
				"spec.rules[0].host: Required value",
			},
			msg: "invalid ingress",
		},
		{
			ing: &networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{
						"nginx.org/mergeable-ingress-type": "master",
					},
				},
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host: "example.com",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path: "/",
										},
									},
								},
							},
						},
					},
				},
			},
			expectedErrors: []string{
				"spec.rules[0].http.paths: Too many: 1: must have at most 0 items",
			},
			msg: "invalid master",
		},
		{
			ing: &networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{
						"nginx.org/mergeable-ingress-type": "minion",
					},
				},
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host:             "example.com",
							IngressRuleValue: networking.IngressRuleValue{},
						},
					},
				},
			},
			expectedErrors: []string{
				"spec.rules[0].http.paths: Required value: must include at least one path",
			},
			msg: "invalid minion",
		},
	}

	for _, test := range tests {
		allErrs := validateIngress(test.ing)
		assertion := assertErrors("validateIngress()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func TestValidateIngressAnnotations(t *testing.T) {
	tests := []struct {
		annotations    map[string]string
		expectedErrors []string
		msg            string
	}{
		{
			annotations:    map[string]string{},
			expectedErrors: nil,
			msg:            "valid input",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
			},
			expectedErrors: nil,
			msg:            "valid input with master annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "minion",
			},
			expectedErrors: nil,
			msg:            "valid input with minion annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "",
			},
			expectedErrors: []string{
				"annotations.nginx.org/mergeable-ingress-type: Required value",
			},
			msg: "invalid mergeable type annotation 1",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "abc",
			},
			expectedErrors: []string{
				`annotations.nginx.org/mergeable-ingress-type: Invalid value: "abc": must be one of: 'master' or 'minion'`,
			},
			msg: "invalid mergeable type annotation 2",
		},
	}

	for _, test := range tests {
		allErrs := validateIngressAnnotations(test.annotations, field.NewPath("annotations"))
		assertion := assertErrors("validateIngressAnnotations()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func TestValidateIngressSpec(t *testing.T) {
	tests := []struct {
		spec           *networking.IngressSpec
		expectedErrors []string
		msg            string
	}{
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
				},
			},
			expectedErrors: nil,
			msg:            "valid input",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{},
			},
			expectedErrors: []string{
				"spec.rules: Required value",
			},
			msg: "zero rules",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "",
					},
				},
			},
			expectedErrors: []string{
				"spec.rules[0].host: Required value",
			},
			msg: "empty host",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
					{
						Host: "foo.example.com",
					},
				},
			},
			expectedErrors: []string{
				`spec.rules[1].host: Duplicate value: "foo.example.com"`,
			},
			msg: "duplicated host",
		},
	}

	for _, test := range tests {
		allErrs := validateIngressSpec(test.spec, field.NewPath("spec"))
		assertion := assertErrors("validateIngressSpec()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func TestValidateMasterSpec(t *testing.T) {
	tests := []struct {
		spec           *networking.IngressSpec
		expectedErrors []string
		msg            string
	}{
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{},
							},
						},
					},
				},
			},
			expectedErrors: nil,
			msg:            "valid input",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
					{
						Host: "bar.example.com",
					},
				},
			},
			expectedErrors: []string{
				"spec.rules: Too many: 2: must have at most 1 items",
			},
			msg: "too many hosts",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: "/",
									},
								},
							},
						},
					},
				},
			},
			expectedErrors: []string{
				"spec.rules[0].http.paths: Too many: 1: must have at most 0 items",
			},
			msg: "too many paths",
		},
	}

	for _, test := range tests {
		allErrs := validateMasterSpec(test.spec, field.NewPath("spec"))
		assertion := assertErrors("validateMasterSpec()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func TestValidateMinionSpec(t *testing.T) {
	tests := []struct {
		spec           *networking.IngressSpec
		expectedErrors []string
		msg            string
	}{
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: "/",
									},
								},
							},
						},
					},
				},
			},
			expectedErrors: nil,
			msg:            "valid input",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
					{
						Host: "bar.example.com",
					},
				},
			},
			expectedErrors: []string{
				"spec.rules: Too many: 2: must have at most 1 items",
			},
			msg: "too many hosts",
		},
		{
			spec: &networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{},
							},
						},
					},
				},
			},
			expectedErrors: []string{
				"spec.rules[0].http.paths: Required value: must include at least one path",
			},
			msg: "too few paths",
		},
		{
			spec: &networking.IngressSpec{
				TLS: []networking.IngressTLS{
					{
						Hosts: []string{"foo.example.com"},
					},
				},
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: "/",
									},
								},
							},
						},
					},
				},
			},
			expectedErrors: []string{
				"spec.tls: Too many: 1: must have at most 0 items",
			},
			msg: "tls is forbidden",
		},
	}

	for _, test := range tests {
		allErrs := validateMinionSpec(test.spec, field.NewPath("spec"))
		assertion := assertErrors("validateMinionSpec()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func assertErrors(funcName string, msg string, allErrs field.ErrorList, expectedErrors []string) string {
	errors := errorListToStrings(allErrs)
	if !reflect.DeepEqual(errors, expectedErrors) {
		result := strings.Join(errors, "\n")
		expected := strings.Join(expectedErrors, "\n")

		return fmt.Sprintf("%s returned \n%s \nbut expected \n%s \nfor the case of %s", funcName, result, expected, msg)
	}

	return ""
}

func errorListToStrings(list field.ErrorList) []string {
	var result []string

	for _, e := range list {
		result = append(result, e.Error())
	}

	return result
}

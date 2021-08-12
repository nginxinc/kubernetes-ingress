package k8s

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateIngress(t *testing.T) {
	tests := []struct {
		ing                   *networking.Ingress
		isPlus                bool
		appProtectEnabled     bool
		internalRoutesEnabled bool
		expectedErrors        []string
		msg                   string
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
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid input",
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
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
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
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
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
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"spec.rules[0].http.paths: Required value: must include at least one path",
			},
			msg: "invalid minion",
		},
	}

	for _, test := range tests {
		allErrs := validateIngress(test.ing, test.isPlus, test.appProtectEnabled, test.internalRoutesEnabled)
		assertion := assertErrors("validateIngress()", test.msg, allErrs, test.expectedErrors)
		if assertion != "" {
			t.Error(assertion)
		}
	}
}

func TestValidateNginxIngressAnnotations(t *testing.T) {
	tests := []struct {
		annotations           map[string]string
		specServices          map[string]bool
		isPlus                bool
		appProtectEnabled     bool
		internalRoutesEnabled bool
		expectedErrors        []string
		msg                   string
	}{
		{
			annotations:           map[string]string{},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid no annotations",
		},

		{
			annotations: map[string]string{
				"nginx.org/lb-method":              "invalid_method",
				"nginx.org/mergeable-ingress-type": "invalid",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/lb-method: Invalid value: "invalid_method": Invalid load balancing method: "invalid_method"`,
				`annotations.nginx.org/mergeable-ingress-type: Invalid value: "invalid": must be one of: 'master' or 'minion'`,
			},
			msg: "invalid multiple annotations messages in alphabetical order",
		},

		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "master",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid input with master annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "minion",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid input with minion annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.org/mergeable-ingress-type: Required value",
			},
			msg: "invalid mergeable type annotation 1",
		},
		{
			annotations: map[string]string{
				"nginx.org/mergeable-ingress-type": "abc",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/mergeable-ingress-type: Invalid value: "abc": must be one of: 'master' or 'minion'`,
			},
			msg: "invalid mergeable type annotation 2",
		},

		{
			annotations: map[string]string{
				"nginx.org/lb-method": "random",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/lb-method annotation, nginx normal",
		},
		{
			annotations: map[string]string{
				"nginx.org/lb-method": "least_time header",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/lb-method: Invalid value: "least_time header": Invalid load balancing method: "least_time header"`,
			},
			msg: "invalid nginx.org/lb-method annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.org/lb-method": "invalid_method",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/lb-method: Invalid value: "invalid_method": Invalid load balancing method: "invalid_method"`,
			},
			msg: "invalid nginx.org/lb-method annotation",
		},

		{
			annotations: map[string]string{
				"nginx.com/health-checks": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/health-checks: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/health-checks annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/health-checks annotation",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.com/health-checks: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid nginx.com/health-checks annotation",
		},

		{
			annotations: map[string]string{
				"nginx.com/health-checks-mandatory": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/health-checks-mandatory annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":           "true",
				"nginx.com/health-checks-mandatory": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/health-checks-mandatory annotation",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":           "true",
				"nginx.com/health-checks-mandatory": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.com/health-checks-mandatory: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid nginx.com/health-checks-mandatory, must be a boolean",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks-mandatory": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory: Forbidden: related annotation nginx.com/health-checks: must be set",
			},
			msg: "invalid nginx.com/health-checks-mandatory, related annotation nginx.com/health-checks not set",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":           "false",
				"nginx.com/health-checks-mandatory": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory: Forbidden: related annotation nginx.com/health-checks: must be true",
			},
			msg: "invalid nginx.com/health-checks-mandatory nginx.com/health-checks is not true",
		},

		{
			annotations: map[string]string{
				"nginx.com/health-checks-mandatory-queue": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory-queue: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/health-checks-mandatory-queue annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":                 "true",
				"nginx.com/health-checks-mandatory":       "true",
				"nginx.com/health-checks-mandatory-queue": "5",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/health-checks-mandatory-queue annotation",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":                 "true",
				"nginx.com/health-checks-mandatory":       "true",
				"nginx.com/health-checks-mandatory-queue": "not_a_number",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.com/health-checks-mandatory-queue: Invalid value: "not_a_number": must be a non-negative integer`,
			},
			msg: "invalid nginx.com/health-checks-mandatory-queue, must be a number",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks-mandatory-queue": "5",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory-queue: Forbidden: related annotation nginx.com/health-checks-mandatory: must be set",
			},
			msg: "invalid nginx.com/health-checks-mandatory-queue, related annotation nginx.com/health-checks-mandatory not set",
		},
		{
			annotations: map[string]string{
				"nginx.com/health-checks":                 "true",
				"nginx.com/health-checks-mandatory":       "false",
				"nginx.com/health-checks-mandatory-queue": "5",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/health-checks-mandatory-queue: Forbidden: related annotation nginx.com/health-checks-mandatory: must be true",
			},
			msg: "invalid nginx.com/health-checks-mandatory-queue nginx.com/health-checks-mandatory is not true",
		},

		{
			annotations: map[string]string{
				"nginx.com/slow-start": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/slow-start: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/slow-start annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.com/slow-start": "60s",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/slow-start annotation",
		},
		{
			annotations: map[string]string{
				"nginx.com/slow-start": "not_a_time",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.com/slow-start: Invalid value: "not_a_time": must be a time`,
			},
			msg: "invalid nginx.com/slow-start annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/server-tokens": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/server-tokens annotation, nginx",
		},
		{
			annotations: map[string]string{
				"nginx.org/server-tokens": "custom_setting",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/server-tokens annotation, nginx plus",
		},
		{
			annotations: map[string]string{
				"nginx.org/server-tokens": "custom_setting",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/server-tokens: Invalid value: "custom_setting": must be a boolean`,
			},
			msg: "invalid nginx.org/server-tokens annotation, must be a boolean",
		},

		{
			annotations: map[string]string{
				"nginx.org/server-snippets": "snippet-1",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/server-snippets annotation, single-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/server-snippets": "snippet-1\nsnippet-2\nsnippet-3",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/server-snippets annotation, multi-value",
		},

		{
			annotations: map[string]string{
				"nginx.org/location-snippets": "snippet-1",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/location-snippets annotation, single-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/location-snippets": "snippet-1\nsnippet-2\nsnippet-3",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/location-snippets annotation, multi-value",
		},

		{
			annotations: map[string]string{
				"nginx.org/proxy-connect-timeout": "10s",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-connect-timeout annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/proxy-connect-timeout": "not_a_time",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/proxy-connect-timeout: Invalid value: "not_a_time": must be a time`,
			},
			msg: "invalid nginx.org/proxy-connect-timeout annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/proxy-read-timeout": "10s",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-read-timeout annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/proxy-read-timeout": "not_a_time",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/proxy-read-timeout: Invalid value: "not_a_time": must be a time`,
			},
			msg: "invalid nginx.org/proxy-read-timeout annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/proxy-send-timeout": "10s",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-send-timeout annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/proxy-send-timeout": "not_a_time",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/proxy-send-timeout: Invalid value: "not_a_time": must be a time`,
			},
			msg: "invalid nginx.org/proxy-send-timeout annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/proxy-hide-headers": "header-1",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-hide-headers annotation, single-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/proxy-hide-headers": "header-1,header-2,header-3",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-hide-headers annotation, multi-value",
		},

		{
			annotations: map[string]string{
				"nginx.org/proxy-pass-headers": "header-1",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-pass-headers annotation, single-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/proxy-pass-headers": "header-1,header-2,header-3",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-pass-headers annotation, multi-value",
		},

		{
			annotations: map[string]string{
				"nginx.org/client-max-body-size": "16M",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/client-max-body-size annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/client-max-body-size": "not_an_offset",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/client-max-body-size: Invalid value: "not_an_offset": must be an offset`,
			},
			msg: "invalid nginx.org/client-max-body-size annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/redirect-to-https": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/redirect-to-https annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/redirect-to-https": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/redirect-to-https: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid nginx.org/redirect-to-https annotation",
		},

		{
			annotations: map[string]string{
				"ingress.kubernetes.io/ssl-redirect": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid ingress.kubernetes.io/ssl-redirect annotation",
		},
		{
			annotations: map[string]string{
				"ingress.kubernetes.io/ssl-redirect": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.ingress.kubernetes.io/ssl-redirect: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid ingress.kubernetes.io/ssl-redirect annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/proxy-buffering": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-buffering annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/proxy-buffering": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/proxy-buffering: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid nginx.org/proxy-buffering annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/hsts": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/hsts annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/hsts: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid nginx.org/hsts annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/hsts":         "true",
				"nginx.org/hsts-max-age": "120",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/hsts-max-age annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts":         "false",
				"nginx.org/hsts-max-age": "120",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/hsts-max-age nginx.org/hsts can be false",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts":         "true",
				"nginx.org/hsts-max-age": "not_a_number",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/hsts-max-age: Invalid value: "not_a_number": must be an integer`,
			},
			msg: "invalid nginx.org/hsts-max-age, must be a number",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts-max-age": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.org/hsts-max-age: Forbidden: related annotation nginx.org/hsts: must be set",
			},
			msg: "invalid nginx.org/hsts-max-age, related annotation nginx.org/hsts not set",
		},

		{
			annotations: map[string]string{
				"nginx.org/hsts":                    "true",
				"nginx.org/hsts-include-subdomains": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/hsts-include-subdomains annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts":                    "false",
				"nginx.org/hsts-include-subdomains": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/hsts-include-subdomains, nginx.org/hsts can be false",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts":                    "true",
				"nginx.org/hsts-include-subdomains": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/hsts-include-subdomains: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid nginx.org/hsts-include-subdomains, must be a boolean",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts-include-subdomains": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.org/hsts-include-subdomains: Forbidden: related annotation nginx.org/hsts: must be set",
			},
			msg: "invalid nginx.org/hsts-include-subdomains, related annotation nginx.org/hsts not set",
		},

		{
			annotations: map[string]string{
				"nginx.org/hsts":              "true",
				"nginx.org/hsts-behind-proxy": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/hsts-behind-proxy annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts":              "false",
				"nginx.org/hsts-behind-proxy": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/hsts-behind-proxy, nginx.org/hsts can be false",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts":              "true",
				"nginx.org/hsts-behind-proxy": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/hsts-behind-proxy: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid nginx.org/hsts-behind-proxy, must be a boolean",
		},
		{
			annotations: map[string]string{
				"nginx.org/hsts-behind-proxy": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.org/hsts-behind-proxy: Forbidden: related annotation nginx.org/hsts: must be set",
			},
			msg: "invalid nginx.org/hsts-behind-proxy, related annotation nginx.org/hsts not set",
		},

		{
			annotations: map[string]string{
				"nginx.org/proxy-buffers": "8 8k",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-buffers annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/proxy-buffers": "not_a_proxy_buffers_spec",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/proxy-buffers: Invalid value: "not_a_proxy_buffers_spec": must be a proxy buffer spec`,
			},
			msg: "invalid nginx.org/proxy-buffers annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/proxy-buffer-size": "16k",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-buffer-size annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/proxy-buffer-size": "not_a_size",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/proxy-buffer-size: Invalid value: "not_a_size": must be a size`,
			},
			msg: "invalid nginx.org/proxy-buffer-size annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/proxy-max-temp-file-size": "128M",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/proxy-max-temp-file-size annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/proxy-max-temp-file-size": "not_a_size",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/proxy-max-temp-file-size: Invalid value: "not_a_size": must be a size`,
			},
			msg: "invalid nginx.org/proxy-max-temp-file-size annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/upstream-zone-size": "512k",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/upstream-zone-size annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/upstream-zone-size": "not a size",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/upstream-zone-size: Invalid value: "not a size": must be a size`,
			},
			msg: "invalid nginx.org/upstream-zone-size annotation",
		},

		{
			annotations: map[string]string{
				"nginx.com/jwt-realm": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/jwt-realm: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/jwt-realm annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.com/jwt-realm": "my-jwt-realm",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/jwt-realm annotation",
		},

		{
			annotations: map[string]string{
				"nginx.com/jwt-key": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/jwt-key: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/jwt-key annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.com/jwt-key": "my-jwk",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/jwt-key annotation",
		},

		{
			annotations: map[string]string{
				"nginx.com/jwt-token": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/jwt-token: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/jwt-token annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.com/jwt-token": "$cookie_auth_token",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/jwt-token annotation",
		},

		{
			annotations: map[string]string{
				"nginx.com/jwt-login-url": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/jwt-login-url: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/jwt-login-url annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.com/jwt-login-url": "https://login.example.com",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/jwt-login-url annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/listen-ports": "80,8080,9090",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/listen-ports annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/listen-ports": "not_a_port_list",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/listen-ports: Invalid value: "not_a_port_list": must be a comma-separated list of port numbers`,
			},
			msg: "invalid nginx.org/listen-ports annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/listen-ports-ssl": "443,8443",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/listen-ports-ssl annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/listen-ports-ssl": "not_a_port_list",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/listen-ports-ssl: Invalid value: "not_a_port_list": must be a comma-separated list of port numbers`,
			},
			msg: "invalid nginx.org/listen-ports-ssl annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/keepalive": "1000",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/keepalive annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/keepalive": "not_a_number",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/keepalive: Invalid value: "not_a_number": must be an integer`,
			},
			msg: "invalid nginx.org/keepalive annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/max-fails": "5",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/max-fails annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/max-fails": "-100",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/max-fails: Invalid value: "-100": must be a non-negative integer`,
			},
			msg: "invalid nginx.org/max-fails annotation, negative number",
		},
		{
			annotations: map[string]string{
				"nginx.org/max-fails": "not_a_number",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/max-fails: Invalid value: "not_a_number": must be a non-negative integer`,
			},
			msg: "invalid nginx.org/max-fails annotation, not a number",
		},

		{
			annotations: map[string]string{
				"nginx.org/max-conns": "10",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/max-conns annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/max-conns": "-100",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/max-conns: Invalid value: "-100": must be a non-negative integer`,
			},
			msg: "invalid nginx.org/max-conns annotation, negative number",
		},
		{
			annotations: map[string]string{
				"nginx.org/max-conns": "not_a_number",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/max-conns: Invalid value: "not_a_number": must be a non-negative integer`,
			},
			msg: "invalid nginx.org/max-conns annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/fail-timeout": "10s",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/fail-timeout annotation",
		},
		{
			annotations: map[string]string{
				"nginx.org/fail-timeout": "not_a_time",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/fail-timeout: Invalid value: "not_a_time": must be a time`,
			},
			msg: "invalid nginx.org/fail-timeout annotation",
		},

		{
			annotations: map[string]string{
				"appprotect.f5.com/app-protect-enable": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.appprotect.f5.com/app-protect-enable: Forbidden: annotation requires AppProtect",
			},
			msg: "invalid appprotect.f5.com/app-protect-enable annotation, requires app protect",
		},
		{
			annotations: map[string]string{
				"appprotect.f5.com/app-protect-enable": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     true,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid appprotect.f5.com/app-protect-enable annotation",
		},
		{
			annotations: map[string]string{
				"appprotect.f5.com/app-protect-enable": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     true,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.appprotect.f5.com/app-protect-enable: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid appprotect.f5.com/app-protect-enable annotation",
		},

		{
			annotations: map[string]string{
				"appprotect.f5.com/app-protect-security-log-enable": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.appprotect.f5.com/app-protect-security-log-enable: Forbidden: annotation requires AppProtect",
			},
			msg: "invalid appprotect.f5.com/app-protect-security-log-enable annotation, requires app protect",
		},
		{
			annotations: map[string]string{
				"appprotect.f5.com/app-protect-security-log-enable": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     true,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid appprotect.f5.com/app-protect-security-log-enable annotation",
		},
		{
			annotations: map[string]string{
				"appprotect.f5.com/app-protect-security-log-enable": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     true,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.appprotect.f5.com/app-protect-security-log-enable: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid appprotect.f5.com/app-protect-security-log-enable annotation",
		},

		{
			annotations: map[string]string{
				"nsm.nginx.com/internal-route": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nsm.nginx.com/internal-route: Forbidden: annotation requires Internal Routes enabled",
			},
			msg: "invalid nsm.nginx.com/internal-route annotation, requires internal routes",
		},
		{
			annotations: map[string]string{
				"nsm.nginx.com/internal-route": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: true,
			expectedErrors:        nil,
			msg:                   "valid nsm.nginx.com/internal-route annotation",
		},
		{
			annotations: map[string]string{
				"nsm.nginx.com/internal-route": "not_a_boolean",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: true,
			expectedErrors: []string{
				`annotations.nsm.nginx.com/internal-route: Invalid value: "not_a_boolean": must be a boolean`,
			},
			msg: "invalid nsm.nginx.com/internal-route annotation",
		},

		{
			annotations: map[string]string{
				"nginx.org/websocket-services": "service-1",
			},
			specServices: map[string]bool{
				"service-1": true,
			},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/websocket-services annotation, single-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/websocket-services": "service-1,service-2",
			},
			specServices: map[string]bool{
				"service-1": true,
				"service-2": true,
			},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/websocket-services annotation, multi-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/websocket-services": "service-1,service-2",
			},
			specServices: map[string]bool{
				"service-1": true,
			},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/websocket-services: Invalid value: "service-1,service-2": must be a comma-separated list of services. The following services were not found: service-2`,
			},
			msg: "invalid nginx.org/websocket-services annotation, service does not exist",
		},

		{
			annotations: map[string]string{
				"nginx.org/ssl-services": "service-1",
			},
			specServices: map[string]bool{
				"service-1": true,
			},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/ssl-services annotation, single-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/ssl-services": "service-1,service-2",
			},
			specServices: map[string]bool{
				"service-1": true,
				"service-2": true,
			},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/ssl-services annotation, multi-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/ssl-services": "service-1,service-2",
			},
			specServices: map[string]bool{
				"service-1": true,
			},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/ssl-services: Invalid value: "service-1,service-2": must be a comma-separated list of services. The following services were not found: service-2`,
			},
			msg: "invalid nginx.org/ssl-services annotation, service does not exist",
		},

		{
			annotations: map[string]string{
				"nginx.org/grpc-services": "service-1",
			},
			specServices: map[string]bool{
				"service-1": true,
			},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/grpc-services annotation, single-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/grpc-services": "service-1,service-2",
			},
			specServices: map[string]bool{
				"service-1": true,
				"service-2": true,
			},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/grpc-services annotation, multi-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/grpc-services": "service-1,service-2",
			},
			specServices: map[string]bool{
				"service-1": true,
			},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.org/grpc-services: Invalid value: "service-1,service-2": must be a comma-separated list of services. The following services were not found: service-2`,
			},
			msg: "invalid nginx.org/grpc-services annotation, service does not exist",
		},

		{
			annotations: map[string]string{
				"nginx.org/rewrites": "serviceName=service-1 rewrite=rewrite-1",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/rewrites annotation, single-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/rewrites": "serviceName=service-1 rewrite=rewrite-1;serviceName=service-2 rewrite=rewrite-2",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.org/rewrites annotation, multi-value",
		},
		{
			annotations: map[string]string{
				"nginx.org/rewrites": "not_a_rewrite",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: true,
			expectedErrors: []string{
				`annotations.nginx.org/rewrites: Invalid value: "not_a_rewrite": must be a semicolon-separated list of rewrites`,
			},
			msg: "invalid nginx.org/rewrites annotation",
		},

		{
			annotations: map[string]string{
				"nginx.com/sticky-cookie-services": "true",
			},
			specServices:          map[string]bool{},
			isPlus:                false,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				"annotations.nginx.com/sticky-cookie-services: Forbidden: annotation requires NGINX Plus",
			},
			msg: "invalid nginx.com/sticky-cookie-services annotation, nginx plus only",
		},
		{
			annotations: map[string]string{
				"nginx.com/sticky-cookie-services": "serviceName=service-1 srv_id expires=1h path=/service-1",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/sticky-cookie-services annotation, single-value",
		},
		{
			annotations: map[string]string{
				"nginx.com/sticky-cookie-services": "serviceName=service-1 srv_id expires=1h path=/service-1;serviceName=service-2 srv_id expires=2h path=/service-2",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors:        nil,
			msg:                   "valid nginx.com/sticky-cookie-services annotation, multi-value",
		},
		{
			annotations: map[string]string{
				"nginx.com/sticky-cookie-services": "not_a_rewrite",
			},
			specServices:          map[string]bool{},
			isPlus:                true,
			appProtectEnabled:     false,
			internalRoutesEnabled: false,
			expectedErrors: []string{
				`annotations.nginx.com/sticky-cookie-services: Invalid value: "not_a_rewrite": must be a semicolon-separated list of sticky services`,
			},
			msg: "invalid nginx.com/sticky-cookie-services annotation",
		},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			allErrs := validateIngressAnnotations(
				test.annotations,
				test.specServices,
				test.isPlus,
				test.appProtectEnabled,
				test.internalRoutesEnabled,
				field.NewPath("annotations"),
			)
			assertion := assertErrors("validateIngressAnnotations()", test.msg, allErrs, test.expectedErrors)
			if assertion != "" {
				t.Error(assertion)
			}
		})
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
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: "/",
										Backend: networking.IngressBackend{
											Service: &networking.IngressServiceBackend{},
										},
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
				DefaultBackend: &networking.IngressBackend{
					Service: &networking.IngressServiceBackend{},
				},
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
				},
			},
			expectedErrors: nil,
			msg:            "valid input with default backend",
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
		{
			spec: &networking.IngressSpec{
				DefaultBackend: &networking.IngressBackend{
					Resource: &v1.TypedLocalObjectReference{},
				},
				Rules: []networking.IngressRule{
					{
						Host: "foo.example.com",
					},
				},
			},
			expectedErrors: []string{
				"spec.defaultBackend.resource: Forbidden: resource backends are not supported",
			},
			msg: "invalid default backend",
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
										Backend: networking.IngressBackend{
											Resource: &v1.TypedLocalObjectReference{},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedErrors: []string{
				"spec.rules[0].http.path[0].backend.resource: Forbidden: resource backends are not supported",
			},
			msg: "invalid backend",
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

func TestGetSpecServices(t *testing.T) {
	tests := []struct {
		spec     networking.IngressSpec
		expected map[string]bool
		msg      string
	}{
		{
			spec: networking.IngressSpec{
				DefaultBackend: &networking.IngressBackend{
					Service: &networking.IngressServiceBackend{
						Name: "svc1",
					},
				},
				Rules: []networking.IngressRule{
					{
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path: "/",
										Backend: networking.IngressBackend{
											Service: &networking.IngressServiceBackend{
												Name: "svc2",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]bool{
				"svc1": true,
				"svc2": true,
			},
			msg: "services are referenced",
		},
		{
			spec: networking.IngressSpec{
				DefaultBackend: &networking.IngressBackend{},
				Rules: []networking.IngressRule{
					{
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Path:    "/",
										Backend: networking.IngressBackend{},
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]bool{},
			msg:      "services are not referenced",
		},
	}

	for _, test := range tests {
		result := getSpecServices(test.spec)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("getSpecServices() returned %v but expected %v for the case of %s", result, test.expected, test.msg)
		}
	}
}

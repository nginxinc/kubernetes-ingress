package validation

import (
	"reflect"
	"testing"

	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateVirtualServer(t *testing.T) {
	virtualServer := v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
		Spec: v1.VirtualServerSpec{
			Host: "example.com",
			TLS: &v1.TLS{
				Secret: "abc",
			},
			Upstreams: []v1.Upstream{
				{
					Name:      "first",
					Service:   "service-1",
					LBMethod:  "random",
					Port:      80,
					MaxFails:  createPointerFromInt(8),
					MaxConns:  createPointerFromInt(16),
					Keepalive: createPointerFromInt(32),
				},
				{
					Name:    "second",
					Service: "service-2",
					Port:    80,
				},
			},
			Routes: []v1.Route{
				{
					Path: "/first",
					Action: &v1.Action{
						Pass: "first",
					},
				},
				{
					Path: "/second",
					Action: &v1.Action{
						Pass: "second",
					},
				},
			},
		},
	}

	err := ValidateVirtualServer(&virtualServer, false)
	if err != nil {
		t.Errorf("ValidateVirtualServer() returned error %v for valid input %v", err, virtualServer)
	}
}

func TestValidateHost(t *testing.T) {
	validHosts := []string{
		"hello",
		"example.com",
		"hello-world-1",
	}

	for _, h := range validHosts {
		allErrs := validateHost(h, field.NewPath("host"))
		if len(allErrs) > 0 {
			t.Errorf("validateHost(%q) returned errors %v for valid input", h, allErrs)
		}
	}

	invalidHosts := []string{
		"",
		"*",
		"..",
		".example.com",
		"-hello-world-1",
	}

	for _, h := range invalidHosts {
		allErrs := validateHost(h, field.NewPath("host"))
		if len(allErrs) == 0 {
			t.Errorf("validateHost(%q) returned no errors for invalid input", h)
		}
	}
}

func TestValidatePolicies(t *testing.T) {
	tests := []struct {
		policies []v1.PolicyReference
		msg      string
	}{
		{
			policies: []v1.PolicyReference{
				{
					Name: "my-policy",
				},
			},
			msg: "name without namespace",
		},
		{
			policies: []v1.PolicyReference{
				{
					Name:      "my-policy",
					Namespace: "nginx-ingress",
				},
			},
			msg: "name with namespace",
		},
		{
			policies: []v1.PolicyReference{
				{
					Name:      "my-policy",
					Namespace: "default",
				},
				{
					Name:      "my-policy",
					Namespace: "test",
				},
			},
			msg: "same name different namespaces",
		},
	}

	for _, test := range tests {
		allErrs := validatePolicies(test.policies, field.NewPath("policies"), "default")
		if len(allErrs) > 0 {
			t.Errorf("validatePolicies() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidatePoliciesFails(t *testing.T) {
	tests := []struct {
		policies []v1.PolicyReference
		msg      string
	}{
		{
			policies: []v1.PolicyReference{
				{
					Name: "",
				},
			},
			msg: "missing name",
		},
		{
			policies: []v1.PolicyReference{
				{
					Name: "-invalid",
				},
			},
			msg: "invalid name",
		},
		{
			policies: []v1.PolicyReference{
				{
					Name:      "my-policy",
					Namespace: "-invalid",
				},
			},
			msg: "invalid namespace",
		},
		{
			policies: []v1.PolicyReference{
				{
					Name:      "my-policy",
					Namespace: "default",
				},
				{
					Name:      "my-policy",
					Namespace: "default",
				},
			},
			msg: "duplicated policies",
		},
		{
			policies: []v1.PolicyReference{
				{
					Name:      "my-policy",
					Namespace: "default",
				},
				{
					Name: "my-policy",
				},
			},
			msg: "duplicated policies with inferred namespace",
		},
	}

	for _, test := range tests {
		allErrs := validatePolicies(test.policies, field.NewPath("policies"), "default")
		if len(allErrs) == 0 {
			t.Errorf("validatePolicies() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateTLS(t *testing.T) {
	validTLSes := []*v1.TLS{
		nil,
		{
			Secret: "",
		},
		{
			Secret: "my-secret",
		},
		{
			Secret:   "my-secret",
			Redirect: &v1.TLSRedirect{},
		},
		{
			Secret: "my-secret",
			Redirect: &v1.TLSRedirect{
				Enable: true,
			},
		},
		{
			Secret: "my-secret",
			Redirect: &v1.TLSRedirect{
				Enable:  true,
				Code:    createPointerFromInt(302),
				BasedOn: "scheme",
			},
		},
		{
			Secret: "my-secret",
			Redirect: &v1.TLSRedirect{
				Enable: true,
				Code:   createPointerFromInt(307),
			},
		},
	}

	for _, tls := range validTLSes {
		allErrs := validateTLS(tls, field.NewPath("tls"))
		if len(allErrs) > 0 {
			t.Errorf("validateTLS() returned errors %v for valid input %v", allErrs, tls)
		}
	}

	invalidTLSes := []*v1.TLS{
		{
			Secret: "-",
		},
		{
			Secret: "a/b",
		},
		{
			Secret: "my-secret",
			Redirect: &v1.TLSRedirect{
				Enable:  true,
				Code:    createPointerFromInt(305),
				BasedOn: "scheme",
			},
		},
		{
			Secret: "my-secret",
			Redirect: &v1.TLSRedirect{
				Enable:  true,
				Code:    createPointerFromInt(301),
				BasedOn: "invalidScheme",
			},
		},
	}

	for _, tls := range invalidTLSes {
		allErrs := validateTLS(tls, field.NewPath("tls"))
		if len(allErrs) == 0 {
			t.Errorf("validateTLS() returned no errors for invalid input %v", tls)
		}
	}
}

func TestValidateUpstreams(t *testing.T) {
	tests := []struct {
		upstreams             []v1.Upstream
		expectedUpstreamNames sets.String
		msg                   string
	}{
		{
			upstreams:             []v1.Upstream{},
			expectedUpstreamNames: sets.String{},
			msg:                   "no upstreams",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:                     "upstream1",
					Service:                  "test-1",
					Port:                     80,
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   5,
					MaxConns:                 createPointerFromInt(16),
				},
				{
					Name:                     "upstream2",
					Subselector:              map[string]string{"version": "test"},
					Service:                  "test-2",
					Port:                     80,
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   5,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
				"upstream2": {},
			},
			msg: "2 valid upstreams",
		},
	}
	isPlus := false
	for _, test := range tests {
		allErrs, resultUpstreamNames := validateUpstreams(test.upstreams, field.NewPath("upstreams"), isPlus)
		if len(allErrs) > 0 {
			t.Errorf("validateUpstreams() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
		if !resultUpstreamNames.Equal(test.expectedUpstreamNames) {
			t.Errorf("validateUpstreams() returned %v expected %v for the case of %s", resultUpstreamNames, test.expectedUpstreamNames, test.msg)
		}
	}
}

func TestValidateUpstreamsFails(t *testing.T) {
	tests := []struct {
		upstreams             []v1.Upstream
		expectedUpstreamNames sets.String
		msg                   string
	}{
		{
			upstreams: []v1.Upstream{
				{
					Name:                     "@upstream1",
					Service:                  "test-1",
					Port:                     80,
					ProxyNextUpstream:        "http_502",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   5,
				},
			},
			expectedUpstreamNames: sets.String{},
			msg:                   "invalid upstream name",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:                     "upstream1",
					Service:                  "@test-1",
					Port:                     80,
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   5,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid service",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:                     "upstream1",
					Service:                  "test-1",
					Port:                     0,
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   5,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid port",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:                     "upstream1",
					Service:                  "test-1",
					Port:                     80,
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   5,
				},
				{
					Name:                     "upstream1",
					Service:                  "test-2",
					Port:                     80,
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   5,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "duplicated upstreams",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:                     "upstream1",
					Service:                  "test-1",
					Port:                     80,
					ProxyNextUpstream:        "https_504",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   5,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid next upstream syntax",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:                     "upstream1",
					Service:                  "test-1",
					Port:                     80,
					ProxyNextUpstream:        "http_504",
					ProxyNextUpstreamTimeout: "-2s",
					ProxyNextUpstreamTries:   5,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid upstream timeout value",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:                     "upstream1",
					Service:                  "test-1",
					Port:                     80,
					ProxyNextUpstream:        "https_504",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   -1,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid upstream tries value",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:     "upstream1",
					Service:  "test-1",
					Port:     80,
					MaxConns: createPointerFromInt(-1),
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "negative value for MaxConns",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:              "upstream1",
					Service:           "test-1",
					Port:              80,
					ClientMaxBodySize: "7mins",
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid value for ClientMaxBodySize",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:    "upstream1",
					Service: "test-1",
					Port:    80,
					ProxyBuffers: &v1.UpstreamBuffers{
						Number: -1,
						Size:   "1G",
					},
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid value for ProxyBuffers",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:            "upstream1",
					Service:         "test-1",
					Port:            80,
					ProxyBufferSize: "1G",
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid value for ProxyBufferSize",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:        "upstream1",
					Service:     "test-1",
					Subselector: map[string]string{"\\$invalidkey": "test"},
					Port:        80,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid key for subselector",
		},
		{
			upstreams: []v1.Upstream{
				{
					Name:        "upstream1",
					Service:     "test-1",
					Subselector: map[string]string{"version": "test=fail"},
					Port:        80,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid value for subselector",
		},
	}

	isPlus := false
	for _, test := range tests {
		allErrs, resultUpstreamNames := validateUpstreams(test.upstreams, field.NewPath("upstreams"), isPlus)
		if len(allErrs) == 0 {
			t.Errorf("validateUpstreams() returned no errors for the case of %s", test.msg)
		}
		if !resultUpstreamNames.Equal(test.expectedUpstreamNames) {
			t.Errorf("validateUpstreams() returned %v expected %v for the case of %s", resultUpstreamNames, test.expectedUpstreamNames, test.msg)
		}
	}
}

func TestValidateNextUpstream(t *testing.T) {
	tests := []struct {
		inputS string
	}{
		{
			inputS: "error timeout",
		},
		{
			inputS: "http_404 timeout",
		},
	}
	for _, test := range tests {
		allErrs := validateNextUpstream(test.inputS, field.NewPath("next-upstreams"))
		if len(allErrs) > 0 {
			t.Errorf("validateNextUpstream(%q) returned errors %v for valid input.", test.inputS, allErrs)
		}
	}
}

func TestValidateNextUpstreamFails(t *testing.T) {
	tests := []struct {
		inputS string
	}{
		{
			inputS: "error error",
		},
		{
			inputS: "https_404",
		},
	}
	for _, test := range tests {
		allErrs := validateNextUpstream(test.inputS, field.NewPath("next-upstreams"))
		if len(allErrs) == 0 {
			t.Errorf("validateNextUpstream(%q) didn't return errors %v for invalid input.", test.inputS, allErrs)
		}
	}
}

func TestValidateDNS1035Label(t *testing.T) {
	validNames := []string{
		"test",
		"test-123",
	}

	for _, name := range validNames {
		allErrs := validateDNS1035Label(name, field.NewPath("name"))
		if len(allErrs) > 0 {
			t.Errorf("validateDNS1035Label(%q) returned errors %v for valid input", name, allErrs)
		}
	}

	invalidNames := []string{
		"",
		"123",
		"test.123",
	}

	for _, name := range invalidNames {
		allErrs := validateDNS1035Label(name, field.NewPath("name"))
		if len(allErrs) == 0 {
			t.Errorf("validateDNS1035Label(%q) returned no errors for invalid input", name)
		}
	}
}

func TestValidateVirtualServerRoutes(t *testing.T) {
	tests := []struct {
		routes        []v1.Route
		upstreamNames sets.String
		msg           string
	}{
		{
			routes:        []v1.Route{},
			upstreamNames: sets.String{},
			msg:           "no routes",
		},
		{
			routes: []v1.Route{
				{
					Path: "/",
					Action: &v1.Action{
						Pass: "test",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			msg: "valid route",
		},
	}

	for _, test := range tests {
		allErrs := validateVirtualServerRoutes(test.routes, field.NewPath("routes"), test.upstreamNames)
		if len(allErrs) > 0 {
			t.Errorf("validateVirtualServerRoutes() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidateVirtualServerRoutesFails(t *testing.T) {
	tests := []struct {
		routes        []v1.Route
		upstreamNames sets.String
		msg           string
	}{
		{
			routes: []v1.Route{
				{
					Path: "/test",
					Action: &v1.Action{
						Pass: "test-1",
					},
				},
				{
					Path: "/test",
					Action: &v1.Action{
						Pass: "test-2",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "duplicated paths",
		},

		{
			routes: []v1.Route{
				{
					Path:   "",
					Action: nil,
				},
			},
			upstreamNames: map[string]sets.Empty{},
			msg:           "invalid route",
		},
	}

	for _, test := range tests {
		allErrs := validateVirtualServerRoutes(test.routes, field.NewPath("routes"), test.upstreamNames)
		if len(allErrs) == 0 {
			t.Errorf("validateVirtualServerRoutes() returned no errors for the case of %s", test.msg)
		}
	}
}

func TestValidateRoute(t *testing.T) {
	tests := []struct {
		route                 v1.Route
		upstreamNames         sets.String
		isRouteFieldForbidden bool
		msg                   string
	}{
		{
			route: v1.Route{

				Path: "/",
				Action: &v1.Action{
					Pass: "test",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "valid route with upstream",
		},
		{
			route: v1.Route{
				Path: "/",
				Splits: []v1.Split{
					{
						Weight: 90,
						Action: &v1.Action{
							Pass: "test-1",
						},
					},
					{
						Weight: 10,
						Action: &v1.Action{
							Pass: "test-2",
						},
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "valid upstream with splits",
		},
		{
			route: v1.Route{
				Path: "/",
				Matches: []v1.Match{
					{
						Conditions: []v1.Condition{
							{
								Header: "x-version",
								Value:  "test-1",
							},
						},
						Action: &v1.Action{
							Pass: "test-1",
						},
					},
				},
				Action: &v1.Action{
					Pass: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "valid action with matches",
		},
		{
			route: v1.Route{

				Path:  "/",
				Route: "default/test",
			},
			upstreamNames:         map[string]sets.Empty{},
			isRouteFieldForbidden: false,
			msg:                   "valid route with route",
		},
	}

	for _, test := range tests {
		allErrs := validateRoute(test.route, field.NewPath("route"), test.upstreamNames, test.isRouteFieldForbidden)
		if len(allErrs) > 0 {
			t.Errorf("validateRoute() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidateRouteFails(t *testing.T) {
	tests := []struct {
		route                 v1.Route
		upstreamNames         sets.String
		isRouteFieldForbidden bool
		msg                   string
	}{
		{
			route: v1.Route{
				Path: "",
				Action: &v1.Action{
					Pass: "test",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "empty path",
		},
		{
			route: v1.Route{
				Path: "/test",
				Action: &v1.Action{
					Pass: "-test",
				},
			},
			upstreamNames:         sets.String{},
			isRouteFieldForbidden: false,
			msg:                   "invalid pass action",
		},
		{
			route: v1.Route{
				Path: "/",
				Action: &v1.Action{
					Pass: "test",
				},
			},
			upstreamNames:         sets.String{},
			isRouteFieldForbidden: false,
			msg:                   "non-existing upstream in pass action",
		},
		{
			route: v1.Route{
				Path: "/",
				Action: &v1.Action{
					Pass: "test",
				},
				Splits: []v1.Split{
					{
						Weight: 90,
						Action: &v1.Action{
							Pass: "test-1",
						},
					},
					{
						Weight: 10,
						Action: &v1.Action{
							Pass: "test-2",
						},
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test":   {},
				"test-1": {},
				"test-2": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "both action and splits exist",
		},
		{
			route: v1.Route{
				Path: "/",
				Splits: []v1.Split{
					{
						Weight: 90,
						Action: &v1.Action{
							Pass: "test-1",
						},
					},
					{
						Weight: 10,
						Action: &v1.Action{
							Pass: "test-2",
						},
					},
				},
				Matches: []v1.Match{
					{
						Conditions: []v1.Condition{
							{
								Header: "x-version",
								Value:  "test-1",
							},
						},
						Action: &v1.Action{
							Pass: "test-1",
						},
					},
				},
				Action: &v1.Action{
					Pass: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "both splits and matches exist",
		},
		{
			route: v1.Route{
				Path:  "/",
				Route: "default/test",
			},
			upstreamNames:         map[string]sets.Empty{},
			isRouteFieldForbidden: true,
			msg:                   "route field exists but is forbidden",
		},
	}

	for _, test := range tests {
		allErrs := validateRoute(test.route, field.NewPath("route"), test.upstreamNames, test.isRouteFieldForbidden)
		if len(allErrs) == 0 {
			t.Errorf("validateRoute() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateAction(t *testing.T) {
	upstreamNames := map[string]sets.Empty{
		"test": {},
	}
	tests := []struct {
		action *v1.Action
		msg    string
	}{
		{
			action: &v1.Action{
				Pass: "test",
			},
			msg: "base pass action",
		},
		{
			action: &v1.Action{
				Redirect: &v1.ActionRedirect{
					URL: "http://www.nginx.com",
				},
			},
			msg: "base redirect action",
		},
		{
			action: &v1.Action{
				Redirect: &v1.ActionRedirect{
					URL:  "http://www.nginx.com",
					Code: 302,
				},
			},
			msg: "redirect action with status code set",
		},
		{
			action: &v1.Action{
				Proxy: &v1.ActionProxy{
					Upstream:    "test",
					RewritePath: "/rewrite",
					RequestHeaders: &v1.ProxyRequestHeaders{
						Set: []v1.Header{
							{
								Name:  "Header-Name",
								Value: "value",
							},
						},
					},
					ResponseHeaders: &v1.ProxyResponseHeaders{
						Hide:   []string{"header-name"},
						Pass:   []string{"header-name"},
						Ignore: []string{"Expires"},
						Add: []v1.AddHeader{
							{
								Header: v1.Header{
									Name:  "Header-Name",
									Value: "value",
								},
								Always: true,
							},
						},
					},
				},
			},
			msg: "proxy action with rewritePath, requestHeaders and responseHeaders",
		},
	}

	for _, test := range tests {
		allErrs := validateAction(test.action, field.NewPath("action"), upstreamNames, "", false)
		if len(allErrs) > 0 {
			t.Errorf("validateAction() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidateActionFails(t *testing.T) {
	upstreamNames := map[string]sets.Empty{}

	tests := []struct {
		action *v1.Action
		msg    string
	}{

		{
			action: &v1.Action{},
			msg:    "empty action",
		},
		{
			action: &v1.Action{
				Redirect: &v1.ActionRedirect{},
			},
			msg: "missing required field url",
		},
		{
			action: &v1.Action{
				Pass: "test",
				Redirect: &v1.ActionRedirect{
					URL: "http://www.nginx.com",
				},
			},
			msg: "multiple actions defined",
		},
		{
			action: &v1.Action{
				Redirect: &v1.ActionRedirect{
					URL:  "http://www.nginx.com",
					Code: 305,
				},
			},
			msg: "redirect action with invalid status code set",
		},
		{
			action: &v1.Action{
				Proxy: &v1.ActionProxy{
					Upstream: "",
				},
			},
			msg: "proxy action with missing upstream field",
		},
	}

	for _, test := range tests {
		allErrs := validateAction(test.action, field.NewPath("action"), upstreamNames, "", false)
		if len(allErrs) == 0 {
			t.Errorf("validateAction() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestCaptureVariables(t *testing.T) {
	tests := []struct {
		s        string
		expected []string
	}{
		{
			"${scheme}://${host}",
			[]string{"scheme", "host"},
		},
		{
			"http://www.nginx.org",
			nil,
		},
		{
			"${}",
			[]string{""},
		},
	}
	for _, test := range tests {
		result := captureVariables(test.s)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("captureVariables(%s) returned %v but expected %v", test.s, result, test.expected)
		}
	}
}

func TestValidateRedirectURL(t *testing.T) {
	tests := []struct {
		redirectURL string
		msg         string
	}{
		{
			redirectURL: "http://www.nginx.com",
			msg:         "base redirect url",
		},
		{
			redirectURL: "${scheme}://${host}/sorry",
			msg:         "multi variable redirect url",
		},
		{
			redirectURL: "${http_x_forwarded_proto}://${host}/sorry",
			msg:         "x-forwarded-proto redirect url use case",
		},
		{
			redirectURL: "http://${host}${request_uri}",
			msg:         "use multi variables, no scheme set",
		},
		{
			redirectURL: "${scheme}://www.${host}${request_uri}",
			msg:         "use multi variables",
		},
		{
			redirectURL: "http://example.com/redirect?source=abc",
			msg:         "arg variable use",
		},
		{
			redirectURL: `\"${scheme}://${host}\"`,
			msg:         "url with escaped quotes",
		},
		{
			redirectURL: "http://{abc}",
			msg:         "url with curly braces with no $ prefix",
		},
	}

	for _, test := range tests {
		allErrs := validateRedirectURL(test.redirectURL, field.NewPath("url"), validRedirectVariableNames)
		if len(allErrs) > 0 {
			t.Errorf("validateRedirectURL(%s) returned errors %v for valid input for the case of %s", test.redirectURL, allErrs, test.msg)
		}
	}
}

func TestValidateRedirectURLFails(t *testing.T) {
	tests := []struct {
		redirectURL string
		msg         string
	}{

		{
			redirectURL: "",
			msg:         "url is blank",
		},
		{
			redirectURL: "/uri",
			msg:         "url does not start with http://, https:// or ${scheme}://",
		},
		{
			redirectURL: "$scheme://www.$host.com",
			msg:         "usage of nginx variable in url without ${}",
		},
		{
			redirectURL: "${scheme}://www.${invalid}.com",
			msg:         "invalid nginx variable in url",
		},
		{
			redirectURL: "${scheme}://www.${{host}.com",
			msg:         "leading curly brace",
		},
		{
			redirectURL: "${host.abc}.com",
			msg:         "multi var in curly brace",
		},
		{
			redirectURL: "${scheme}://www.${host{host}}.com",
			msg:         "nested nginx vars",
		},
		{
			redirectURL: `"${scheme}://${host}"`,
			msg:         "url in unescaped quotes",
		},
		{
			redirectURL: `"${scheme}://${host}`,
			msg:         "url with unescaped quote prefix",
		},
		{
			redirectURL: `\\"${scheme}://${host}\\"`,
			msg:         "url with escaped backslash",
		},
		{
			redirectURL: `${scheme}://${host}$`,
			msg:         "url with ending $",
		},
		{
			redirectURL: `http://${}`,
			msg:         "url containing blank var",
		},
		{
			redirectURL: `http://${abca`,
			msg:         "url containing a var without ending }",
		},
	}

	for _, test := range tests {
		allErrs := validateRedirectURL(test.redirectURL, field.NewPath("action"), validRedirectVariableNames)
		if len(allErrs) == 0 {
			t.Errorf("validateRedirectURL(%s) returned no errors for invalid input for the case of %s", test.redirectURL, test.msg)
		}
	}
}

func TestValidateRouteField(t *testing.T) {
	validRouteFields := []string{
		"coffee",
		"default/coffee",
	}

	for _, rf := range validRouteFields {
		allErrs := validateRouteField(rf, field.NewPath("route"))
		if len(allErrs) > 0 {
			t.Errorf("validRouteField(%q) returned errors %v for valid input", rf, allErrs)
		}
	}

	invalidRouteFields := []string{
		"-",
		"/coffee",
		"-/coffee",
	}

	for _, rf := range invalidRouteFields {
		allErrs := validateRouteField(rf, field.NewPath("route"))
		if len(allErrs) == 0 {
			t.Errorf("validRouteField(%q) returned no errors for invalid input", rf)
		}
	}
}

func TestValdateReferencedUpstream(t *testing.T) {
	upstream := "test"
	upstreamNames := map[string]sets.Empty{
		"test": {},
	}

	allErrs := validateReferencedUpstream(upstream, field.NewPath("upstream"), upstreamNames)
	if len(allErrs) > 0 {
		t.Errorf("validateReferencedUpstream() returned errors %v for valid input", allErrs)
	}
}

func TestValdateUpstreamFails(t *testing.T) {
	tests := []struct {
		upstream      string
		upstreamNames sets.String
		msg           string
	}{
		{
			upstream:      "",
			upstreamNames: map[string]sets.Empty{},
			msg:           "empty upstream",
		},
		{
			upstream:      "-test",
			upstreamNames: map[string]sets.Empty{},
			msg:           "invalid upstream",
		},
		{
			upstream:      "test",
			upstreamNames: map[string]sets.Empty{},
			msg:           "non-existing upstream",
		},
	}

	for _, test := range tests {
		allErrs := validateReferencedUpstream(test.upstream, field.NewPath("upstream"), test.upstreamNames)
		if len(allErrs) == 0 {
			t.Errorf("validateReferencedUpstream() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateRegexPath(t *testing.T) {
	tests := []struct {
		regexPath string
		msg       string
	}{
		{
			regexPath: "~ ^/foo.*\\.jpg",
			msg:       "case sensitive regexp",
		},
		{
			regexPath: "~* ^/Bar.*\\.jpg",
			msg:       "case insensitive regexp",
		},
		{
			regexPath: `~ ^/f\"oo.*\\.jpg`,
			msg:       "regexp with escaped double quotes",
		},
	}

	for _, test := range tests {
		allErrs := validateRegexPath(test.regexPath, field.NewPath("path"))
		if len(allErrs) != 0 {
			t.Errorf("validateRegexPath(%v) returned errors for valid input for the case of %v", test.regexPath, test.msg)
		}
	}
}

func TestValidateRegexPathFails(t *testing.T) {
	tests := []struct {
		regexPath string
		msg       string
	}{
		{
			regexPath: "~ [{",
			msg:       "invalid regexp",
		},
		{
			regexPath: `~ /foo"`,
			msg:       "unescaped double quotes",
		},
		{
			regexPath: `~"`,
			msg:       "empty regex",
		},
		{
			regexPath: `~ /foo\`,
			msg:       "ending in backslash",
		},
	}

	for _, test := range tests {
		allErrs := validateRegexPath(test.regexPath, field.NewPath("path"))
		if len(allErrs) == 0 {
			t.Errorf("validateRegexPath(%v) returned no errors for invalid input for the case of %v", test.regexPath, test.msg)
		}
	}
}

func TestValidateRoutePath(t *testing.T) {
	validPaths := []string{
		"/",
		"~ /^foo.*\\.jpg",
		"~* /^Bar.*\\.jpg",
		"=/exact/match",
	}

	for _, path := range validPaths {
		allErrs := validateRoutePath(path, field.NewPath("path"))
		if len(allErrs) != 0 {
			t.Errorf("validateRoutePath(%v) returned errors for valid input", path)
		}
	}

	invalidPaths := []string{
		"",
		"invalid",
	}

	for _, path := range invalidPaths {
		allErrs := validateRoutePath(path, field.NewPath("path"))
		if len(allErrs) == 0 {
			t.Errorf("validateRoutePath(%v) returned no errors for invalid input", path)
		}
	}
}

func TestValidatePath(t *testing.T) {
	validPaths := []string{
		"/",
		"/path",
		"/a-1/_A/",
	}

	for _, path := range validPaths {
		allErrs := validatePath(path, field.NewPath("path"))
		if len(allErrs) > 0 {
			t.Errorf("validatePath(%q) returned errors %v for valid input", path, allErrs)
		}
	}

	invalidPaths := []string{
		"",
		" /",
		"/ ",
		"/{",
		"/}",
		"/abc;",
	}

	for _, path := range invalidPaths {
		allErrs := validatePath(path, field.NewPath("path"))
		if len(allErrs) == 0 {
			t.Errorf("validatePath(%q) returned no errors for invalid input", path)
		}
	}
}

func TestValidateSplits(t *testing.T) {
	splits := []v1.Split{
		{
			Weight: 90,
			Action: &v1.Action{
				Pass: "test-1",
			},
		},
		{
			Weight: 10,
			Action: &v1.Action{
				Proxy: &v1.ActionProxy{
					Upstream: "test-2",
				},
			},
		},
	}
	upstreamNames := map[string]sets.Empty{
		"test-1": {},
		"test-2": {},
	}

	allErrs := validateSplits(splits, field.NewPath("splits"), upstreamNames, "")
	if len(allErrs) > 0 {
		t.Errorf("validateSplits() returned errors %v for valid input", allErrs)
	}
}

func TestValidateSplitsFails(t *testing.T) {
	tests := []struct {
		splits        []v1.Split
		upstreamNames sets.String
		msg           string
	}{
		{
			splits: []v1.Split{
				{
					Weight: 90,
					Action: &v1.Action{
						Pass: "test-1",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
			},
			msg: "only one split",
		},
		{
			splits: []v1.Split{
				{
					Weight: 123,
					Action: &v1.Action{
						Pass: "test-1",
					},
				},
				{
					Weight: 10,
					Action: &v1.Action{
						Pass: "test-2",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "invalid weight",
		},
		{
			splits: []v1.Split{
				{
					Weight: 99,
					Action: &v1.Action{
						Pass: "test-1",
					},
				},
				{
					Weight: 99,
					Action: &v1.Action{
						Pass: "test-2",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "invalid total weight",
		},
		{
			splits: []v1.Split{
				{
					Weight: 90,
					Action: &v1.Action{
						Pass: "",
					},
				},
				{
					Weight: 10,
					Action: &v1.Action{
						Pass: "test-2",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "invalid action",
		},
		{
			splits: []v1.Split{
				{
					Weight: 90,
					Action: &v1.Action{
						Pass: "some-upstream",
					},
				},
				{
					Weight: 10,
					Action: &v1.Action{
						Pass: "test-2",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "invalid action with non-existing upstream",
		},
	}

	for _, test := range tests {
		allErrs := validateSplits(test.splits, field.NewPath("splits"), test.upstreamNames, "")
		if len(allErrs) == 0 {
			t.Errorf("validateSplits() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateCondition(t *testing.T) {
	tests := []struct {
		condition v1.Condition
		msg       string
	}{
		{
			condition: v1.Condition{
				Header: "x-version",
				Value:  "v1",
			},
			msg: "valid header",
		},
		{
			condition: v1.Condition{
				Cookie: "my_cookie",
				Value:  "",
			},
			msg: "valid cookie",
		},
		{
			condition: v1.Condition{
				Argument: "arg",
				Value:    "yes",
			},
			msg: "valid argument",
		},
		{
			condition: v1.Condition{
				Variable: "$request_method",
				Value:    "POST",
			},
			msg: "valid variable",
		},
	}

	for _, test := range tests {
		allErrs := validateCondition(test.condition, field.NewPath("condition"))
		if len(allErrs) > 0 {
			t.Errorf("validateCondition() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidateConditionFails(t *testing.T) {
	tests := []struct {
		condition v1.Condition
		msg       string
	}{
		{
			condition: v1.Condition{},
			msg:       "empty condition",
		},
		{
			condition: v1.Condition{
				Header:   "x-version",
				Cookie:   "user",
				Argument: "answer",
				Variable: "$request_method",
				Value:    "something",
			},
			msg: "invalid condition",
		},
		{
			condition: v1.Condition{
				Header: "x_version",
			},
			msg: "invalid header",
		},
		{
			condition: v1.Condition{
				Cookie: "my-cookie",
			},
			msg: "invalid cookie",
		},
		{
			condition: v1.Condition{
				Argument: "my-arg",
			},
			msg: "invalid argument",
		},
		{
			condition: v1.Condition{
				Variable: "request_method",
			},
			msg: "invalid variable",
		},
	}

	for _, test := range tests {
		allErrs := validateCondition(test.condition, field.NewPath("condition"))
		if len(allErrs) == 0 {
			t.Errorf("validateCondition() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestIsCookieName(t *testing.T) {
	validCookieNames := []string{
		"123",
		"my_cookie",
	}

	for _, name := range validCookieNames {
		errs := isCookieName(name)
		if len(errs) > 0 {
			t.Errorf("isCookieName(%q) returned errors %v for valid input", name, errs)
		}
	}

	invalidCookieNames := []string{
		"",
		"my-cookie",
		"cookie!",
	}

	for _, name := range invalidCookieNames {
		errs := isCookieName(name)
		if len(errs) == 0 {
			t.Errorf("isCookieName(%q) returned no errors for invalid input", name)
		}
	}
}

func TestIsArgumentName(t *testing.T) {
	validArgumentNames := []string{
		"123",
		"my_arg",
	}

	for _, name := range validArgumentNames {
		errs := isArgumentName(name)
		if len(errs) > 0 {
			t.Errorf("isArgumentName(%q) returned errors %v for valid input", name, errs)
		}
	}

	invalidArgumentNames := []string{
		"",
		"my-arg",
		"arg!",
	}

	for _, name := range invalidArgumentNames {
		errs := isArgumentName(name)
		if len(errs) == 0 {
			t.Errorf("isArgumentName(%q) returned no errors for invalid input", name)
		}
	}
}

func TestValidateVariableName(t *testing.T) {
	validNames := []string{
		"$request_method",
	}

	for _, name := range validNames {
		allErrs := validateVariableName(name, field.NewPath("variable"))
		if len(allErrs) > 0 {
			t.Errorf("validateVariableName(%q) returned errors %v for valid input", name, allErrs)
		}
	}

	invalidNames := []string{
		"request_method",
		"$request_id",
	}

	for _, name := range invalidNames {
		allErrs := validateVariableName(name, field.NewPath("variable"))
		if len(allErrs) == 0 {
			t.Errorf("validateVariableName(%q) returned no errors for invalid input", name)
		}
	}
}

func TestValidateMatch(t *testing.T) {
	tests := []struct {
		match         v1.Match
		upstreamNames sets.String
		msg           string
	}{
		{
			match: v1.Match{
				Conditions: []v1.Condition{
					{
						Cookie: "version",
						Value:  "v1",
					},
				},
				Action: &v1.Action{
					Pass: "test",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			msg: "valid match with action",
		},
		{
			match: v1.Match{
				Conditions: []v1.Condition{
					{
						Cookie: "version",
						Value:  "v1",
					},
				},
				Splits: []v1.Split{
					{
						Weight: 90,
						Action: &v1.Action{
							Pass: "test-1",
						},
					},
					{
						Weight: 10,
						Action: &v1.Action{
							Pass: "test-2",
						},
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "valid match with splits",
		},
	}

	for _, test := range tests {
		allErrs := validateMatch(test.match, field.NewPath("match"), test.upstreamNames, "")
		if len(allErrs) > 0 {
			t.Errorf("validateMatch() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidateMatchFails(t *testing.T) {
	tests := []struct {
		match         v1.Match
		upstreamNames sets.String
		msg           string
	}{
		{
			match: v1.Match{
				Conditions: []v1.Condition{},
				Action: &v1.Action{
					Pass: "test",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			msg: "invalid number of conditions",
		},
		{
			match: v1.Match{
				Conditions: []v1.Condition{
					{
						Cookie: "version",
						Value:  `v1"`,
					},
				},
				Action: &v1.Action{
					Pass: "test",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			msg: "invalid condition",
		},
		{
			match: v1.Match{
				Conditions: []v1.Condition{
					{
						Cookie: "version",
						Value:  "v1",
					},
				},
				Action: &v1.Action{},
			},
			upstreamNames: map[string]sets.Empty{},
			msg:           "invalid  action",
		},
		{
			match: v1.Match{
				Conditions: []v1.Condition{
					{
						Cookie: "version",
						Value:  "v1",
					},
				},
				Action: &v1.Action{
					Pass: "test-1",
				},
				Splits: []v1.Split{
					{
						Weight: 90,
						Action: &v1.Action{
							Pass: "test-1",
						},
					},
					{
						Weight: 10,
						Action: &v1.Action{
							Pass: "test-2",
						},
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "both splits and action are set",
		},
	}

	for _, test := range tests {
		allErrs := validateMatch(test.match, field.NewPath("match"), test.upstreamNames, "")
		if len(allErrs) == 0 {
			t.Errorf("validateMatch() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestIsValidMatchValue(t *testing.T) {
	validValues := []string{
		"abc",
		"123",
		`\"
		abc\"`,
		`\"`,
	}

	for _, value := range validValues {
		errs := isValidMatchValue(value)
		if len(errs) > 0 {
			t.Errorf("isValidMatchValue(%q) returned errors %v for valid input", value, errs)
		}
	}

	invalidValues := []string{
		`"`,
		`\`,
		`abc"`,
		`abc\\\`,
		`a"b`,
	}

	for _, value := range invalidValues {
		errs := isValidMatchValue(value)
		if len(errs) == 0 {
			t.Errorf("isValidMatchValue(%q) returned no errors for invalid input", value)
		}
	}
}

func TestValidateVirtualServerRoute(t *testing.T) {
	virtualServerRoute := v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee",
			Namespace: "default",
		},
		Spec: v1.VirtualServerRouteSpec{
			Host: "example.com",
			Upstreams: []v1.Upstream{
				{
					Name:    "first",
					Service: "service-1",
					Port:    80,
				},
				{
					Name:    "second",
					Service: "service-2",
					Port:    80,
				},
			},
			Subroutes: []v1.Route{
				{
					Path: "/test/first",
					Action: &v1.Action{
						Pass: "first",
					},
				},
				{
					Path: "/test/second",
					Action: &v1.Action{
						Pass: "second",
					},
				},
			},
		},
	}
	isPlus := false
	err := ValidateVirtualServerRoute(&virtualServerRoute, isPlus)
	if err != nil {
		t.Errorf("ValidateVirtualServerRoute() returned error %v for valid input %v", err, virtualServerRoute)
	}
}

func TestValidateVirtualServerRouteForVirtualServer(t *testing.T) {
	virtualServerRoute := v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee",
			Namespace: "default",
		},
		Spec: v1.VirtualServerRouteSpec{
			Host: "example.com",
			Upstreams: []v1.Upstream{
				{
					Name:    "first",
					Service: "service-1",
					Port:    80,
				},
				{
					Name:    "second",
					Service: "service-2",
					Port:    80,
				},
			},
			Subroutes: []v1.Route{
				{
					Path: "/test/first",
					Action: &v1.Action{
						Pass: "first",
					},
				},
				{
					Path: "/test/second",
					Action: &v1.Action{
						Pass: "second",
					},
				},
			},
		},
	}
	virtualServerHost := "example.com"
	pathPrefix := "/test"

	isPlus := false
	err := ValidateVirtualServerRouteForVirtualServer(&virtualServerRoute, virtualServerHost, pathPrefix, isPlus)
	if err != nil {
		t.Errorf("ValidateVirtualServerRouteForVirtualServer() returned error %v for valid input %v", err, virtualServerRoute)
	}
}

func TestValidateVirtualServerRouteHost(t *testing.T) {
	virtualServerHost := "example.com"

	validHost := "example.com"

	allErrs := validateVirtualServerRouteHost(validHost, virtualServerHost, field.NewPath("host"))
	if len(allErrs) > 0 {
		t.Errorf("validateVirtualServerRouteHost() returned errors %v for valid input", allErrs)
	}

	invalidHost := "foo.example.com"

	allErrs = validateVirtualServerRouteHost(invalidHost, virtualServerHost, field.NewPath("host"))
	if len(allErrs) == 0 {
		t.Errorf("validateVirtualServerRouteHost() returned no errors for invalid input")
	}
}

func TestValidateVirtualServerRouteSubroutes(t *testing.T) {
	tests := []struct {
		routes        []v1.Route
		upstreamNames sets.String
		pathPrefix    string
		msg           string
	}{
		{
			routes:        []v1.Route{},
			upstreamNames: sets.String{},
			pathPrefix:    "/",
			msg:           "no routes",
		},
		{
			routes: []v1.Route{
				{
					Path: "/",
					Action: &v1.Action{
						Pass: "test",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			pathPrefix: "/",
			msg:        "valid route",
		},
	}

	for _, test := range tests {
		allErrs := validateVirtualServerRouteSubroutes(test.routes, field.NewPath("subroutes"), test.upstreamNames, test.pathPrefix)
		if len(allErrs) > 0 {
			t.Errorf("validateVirtualServerRouteSubroutes() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidateVirtualServerRouteSubroutesFails(t *testing.T) {
	tests := []struct {
		routes        []v1.Route
		upstreamNames sets.String
		pathPrefix    string
		msg           string
	}{
		{
			routes: []v1.Route{
				{
					Path: "/test",
					Action: &v1.Action{
						Pass: "test-1",
					},
				},
				{
					Path: "/test",
					Action: &v1.Action{
						Pass: "test-2",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			pathPrefix: "/",
			msg:        "duplicated paths",
		},
		{
			routes: []v1.Route{
				{
					Path:   "",
					Action: nil,
				},
			},
			upstreamNames: map[string]sets.Empty{},
			pathPrefix:    "",
			msg:           "invalid route",
		},
		{
			routes: []v1.Route{
				{
					Path: "/",
					Action: &v1.Action{
						Pass: "test-1",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
			},
			pathPrefix: "/abc",
			msg:        "invalid prefix",
		},
	}

	for _, test := range tests {
		allErrs := validateVirtualServerRouteSubroutes(test.routes, field.NewPath("subroutes"), test.upstreamNames, test.pathPrefix)
		if len(allErrs) == 0 {
			t.Errorf("validateVirtualServerRouteSubroutes() returned no errors for the case of %s", test.msg)
		}
	}
}

func TestValidateUpstreamLBMethod(t *testing.T) {
	tests := []struct {
		method string
		isPlus bool
	}{
		{
			method: "round_robin",
			isPlus: false,
		},
		{
			method: "",
			isPlus: false,
		},
		{
			method: "ip_hash",
			isPlus: true,
		},
		{
			method: "",
			isPlus: true,
		},
	}

	for _, test := range tests {
		allErrs := validateUpstreamLBMethod(test.method, field.NewPath("lb-method"), test.isPlus)

		if len(allErrs) != 0 {
			t.Errorf("validateUpstreamLBMethod(%q, %v) returned errors for method %s", test.method, test.isPlus, test.method)
		}
	}
}

func TestValidateUpstreamLBMethodFails(t *testing.T) {
	tests := []struct {
		method string
		isPlus bool
	}{
		{
			method: "wrong",
			isPlus: false,
		},
		{
			method: "wrong",
			isPlus: true,
		},
	}

	for _, test := range tests {
		allErrs := validateUpstreamLBMethod(test.method, field.NewPath("lb-method"), test.isPlus)

		if len(allErrs) == 0 {
			t.Errorf("validateUpstreamLBMethod(%q, %v) returned no errors for method %s", test.method, test.isPlus, test.method)
		}
	}
}

func createPointerFromInt(n int) *int {
	return &n
}

func TestValidatePositiveIntOrZeroFromPointer(t *testing.T) {
	tests := []struct {
		number *int
		msg    string
	}{
		{
			number: nil,
			msg:    "valid (nil)",
		},
		{
			number: createPointerFromInt(0),
			msg:    "valid (0)",
		},
		{
			number: createPointerFromInt(1),
			msg:    "valid (1)",
		},
	}

	for _, test := range tests {
		allErrs := validatePositiveIntOrZeroFromPointer(test.number, field.NewPath("int-field"))

		if len(allErrs) != 0 {
			t.Errorf("validatePositiveIntOrZeroFromPointer returned errors for case: %v", test.msg)
		}
	}
}

func TestValidatePositiveIntOrZeroFromPointerFails(t *testing.T) {
	number := createPointerFromInt(-1)
	allErrs := validatePositiveIntOrZeroFromPointer(number, field.NewPath("int-field"))

	if len(allErrs) == 0 {
		t.Error("validatePositiveIntOrZeroFromPointer returned no errors for case: invalid (-1)")
	}
}

func TestValidatePositiveIntOrZero(t *testing.T) {
	tests := []struct {
		number int
		msg    string
	}{
		{
			number: 0,
			msg:    "valid (0)",
		},
		{
			number: 1,
			msg:    "valid (1)",
		},
	}

	for _, test := range tests {
		allErrs := validatePositiveIntOrZero(test.number, field.NewPath("int-field"))

		if len(allErrs) != 0 {
			t.Errorf("validatePositiveIntOrZero returned errors for case: %v", test.msg)
		}
	}
}

func TestValidatePositiveIntOrZeroFails(t *testing.T) {
	allErrs := validatePositiveIntOrZero(-1, field.NewPath("int-field"))

	if len(allErrs) == 0 {
		t.Error("validatePositiveIntOrZero returned no errors for case: invalid (-1)")
	}
}

func TestValidateTime(t *testing.T) {
	time := "1h 2s"
	allErrs := validateTime(time, field.NewPath("time-field"))

	if len(allErrs) != 0 {
		t.Errorf("validateTime returned errors %v valid input %v", allErrs, time)
	}
}

func TestValidateOffset(t *testing.T) {
	var validInput = []string{"", "1", "10k", "11m", "1K", "100M", "5G"}
	for _, test := range validInput {
		allErrs := validateOffset(test, field.NewPath("offset-field"))
		if len(allErrs) != 0 {
			t.Errorf("validateOffset(%q) returned an error for valid input", test)
		}
	}

	var invalidInput = []string{"55mm", "2mG", "6kb", "-5k", "1L", "5Gb"}
	for _, test := range invalidInput {
		allErrs := validateOffset(test, field.NewPath("offset-field"))
		if len(allErrs) == 0 {
			t.Errorf("validateOffset(%q) didn't return error for invalid input.", test)
		}
	}
}

func TestValidateBuffer(t *testing.T) {
	validbuff := &v1.UpstreamBuffers{Number: 8, Size: "8k"}
	allErrs := validateBuffer(validbuff, field.NewPath("buffers-field"))

	if len(allErrs) != 0 {
		t.Errorf("validateBuffer returned errors %v valid input %v", allErrs, validbuff)
	}

	invalidbuff := []*v1.UpstreamBuffers{
		{
			Number: -8,
			Size:   "15m",
		},
		{
			Number: 8,
			Size:   "15G",
		},
		{
			Number: 8,
			Size:   "",
		},
	}
	for _, test := range invalidbuff {
		allErrs = validateBuffer(test, field.NewPath("buffers-field"))
		if len(allErrs) == 0 {
			t.Errorf("validateBuffer didn't return error for invalid input %v.", test)
		}
	}
}

func TestValidateSize(t *testing.T) {
	var validInput = []string{"", "4k", "8K", "16m", "32M"}
	for _, test := range validInput {
		allErrs := validateSize(test, field.NewPath("size-field"))
		if len(allErrs) != 0 {
			t.Errorf("validateSize(%q) returned an error for valid input", test)
		}
	}

	var invalidInput = []string{"55mm", "2mG", "6kb", "-5k", "1L", "5G"}
	for _, test := range invalidInput {
		allErrs := validateSize(test, field.NewPath("size-field"))
		if len(allErrs) == 0 {
			t.Errorf("validateSize(%q) didn't return error for invalid input.", test)
		}
	}
}

func TestValidateTimeFails(t *testing.T) {
	time := "invalid"
	allErrs := validateTime(time, field.NewPath("time-field"))

	if len(allErrs) == 0 {
		t.Errorf("validateTime returned no errors for invalid input %v", time)
	}
}

func TestValidateUpstreamHealthCheck(t *testing.T) {
	hc := &v1.HealthCheck{
		Enable:   true,
		Path:     "/healthz",
		Interval: "4s",
		Jitter:   "2s",
		Fails:    3,
		Passes:   2,
		Port:     8080,
		TLS: &v1.UpstreamTLS{
			Enable: true,
		},
		ConnectTimeout: "1s",
		ReadTimeout:    "1s",
		SendTimeout:    "1s",
		Headers: []v1.Header{
			{
				Name:  "Host",
				Value: "my.service",
			},
		},
		StatusMatch: "! 500",
	}

	allErrs := validateUpstreamHealthCheck(hc, field.NewPath("healthCheck"))

	if len(allErrs) != 0 {
		t.Errorf("validateUpstreamHealthCheck() returned errors for valid input %v", hc)
	}
}

func TestValidateUpstreamHealthCheckFails(t *testing.T) {
	tests := []struct {
		hc *v1.HealthCheck
	}{
		{
			hc: &v1.HealthCheck{
				Enable: true,
				Path:   "/healthz//;",
			},
		},
		{
			hc: &v1.HealthCheck{
				Enable: false,
				Path:   "/healthz//;",
			},
		},
	}

	for _, test := range tests {
		allErrs := validateUpstreamHealthCheck(test.hc, field.NewPath("healthCheck"))

		if len(allErrs) == 0 {
			t.Errorf("validateUpstreamHealthCheck() returned no errors for invalid input %v", test.hc)
		}
	}
}

func TestValidateStatusMatch(t *testing.T) {
	tests := []struct {
		status string
	}{
		{
			status: "200",
		},
		{
			status: "! 500",
		},
		{
			status: "200 204",
		},
		{
			status: "! 301 302",
		},
		{
			status: "200-399",
		},
		{
			status: "! 400-599",
		},
		{
			status: "301-303 307",
		},
	}
	for _, test := range tests {
		allErrs := validateStatusMatch(test.status, field.NewPath("statusMatch"))

		if len(allErrs) != 0 {
			t.Errorf("validateStatusMatch() returned errors %v for valid input %v", allErrs, test.status)
		}
	}
}

func TestValidateStatusMatchFails(t *testing.T) {
	tests := []struct {
		status string
		msg    string
	}{
		{
			status: "qwe",
			msg:    "Invalid: no digits",
		},
		{
			status: "!",
			msg:    "Invalid: `!` character only",
		},
		{
			status: "!500",
			msg:    "Invalid: no space after !",
		},
		{
			status: "0",
			msg:    "Invalid: status out of range (below 100)",
		},
		{
			status: "1000",
			msg:    "Invalid: status out of range (above 999)",
		},
		{
			status: "20-600",
			msg:    "Invalid: code in range is out of range",
		},
		{
			status: "! 200 ! 500",
			msg:    "Invalid: 2 exclamation symbols",
		},
		{
			status: "200 - 500",
			msg:    "Invalid: range with space around `-`",
		},
		{
			status: "500-200",
			msg:    "Invalid: range must be min < max",
		},
		{
			status: "200-200-400",
			msg:    "Invalid: range with more than 2 numbers",
		},
	}
	for _, test := range tests {
		allErrs := validateStatusMatch(test.status, field.NewPath("statusMatch"))

		if len(allErrs) == 0 {
			t.Errorf("validateStatusMatch() returned no errors for case %v", test.msg)
		}
	}
}

func TestValidateHeader(t *testing.T) {
	tests := []struct {
		header v1.Header
	}{
		{
			header: v1.Header{
				Name:  "Host",
				Value: "my.service",
			},
		},
		{
			header: v1.Header{
				Name:  "Host",
				Value: `\"my.service\"`,
			},
		},
	}

	for _, test := range tests {
		allErrs := validateHeader(test.header, field.NewPath("headers"))

		if len(allErrs) != 0 {
			t.Errorf("validateHeader() returned errors %v for valid input %v", allErrs, test.header)
		}
	}
}

func TestValidateHeaderFails(t *testing.T) {
	tests := []struct {
		header v1.Header
		msg    string
	}{
		{
			header: v1.Header{
				Name:  "12378 qwe ",
				Value: "my.service",
			},
			msg: "Invalid name with spaces",
		},
		{
			header: v1.Header{
				Name:  "Host",
				Value: `"my.service`,
			},
			msg: `Invalid value with unescaped '"'`,
		},
		{
			header: v1.Header{
				Name:  "Host",
				Value: `my.service\`,
			},
			msg: "Invalid value with ending '\\'",
		},
		{
			header: v1.Header{
				Name:  "Host",
				Value: "$my.service",
			},
			msg: "Invalid value with '$' character",
		},
		{
			header: v1.Header{
				Name:  "Host",
				Value: "my.\\$service",
			},
			msg: "Invalid value with escaped '$' character",
		},
	}
	for _, test := range tests {
		allErrs := validateHeader(test.header, field.NewPath("headers"))

		if len(allErrs) == 0 {
			t.Errorf("validateHeader() returned no errors for case: %v", test.msg)
		}
	}
}

func TestValidateIntFromString(t *testing.T) {
	input := "404"
	_, errMsg := validateIntFromString(input)

	if errMsg != "" {
		t.Errorf("validateIntFromString() returned errors %v for valid input %v", errMsg, input)
	}
}

func TestValidateIntFromStringFails(t *testing.T) {
	input := "not a number"
	_, errMsg := validateIntFromString(input)

	if errMsg == "" {
		t.Errorf("validateIntFromString() returned no errors for invalid input %v", input)
	}
}

func TestRejectPlusResourcesInOSS(t *testing.T) {
	tests := []struct {
		upstream *v1.Upstream
	}{
		{
			upstream: &v1.Upstream{
				SlowStart: "10s",
			},
		},
		{
			upstream: &v1.Upstream{
				HealthCheck: &v1.HealthCheck{},
			},
		},
		{
			upstream: &v1.Upstream{
				SessionCookie: &v1.SessionCookie{},
			},
		},
		{
			upstream: &v1.Upstream{
				Queue: &v1.UpstreamQueue{},
			},
		},
	}

	for _, test := range tests {
		allErrsOSS := rejectPlusResourcesInOSS(*test.upstream, field.NewPath("upstreams"), false)

		if len(allErrsOSS) == 0 {
			t.Errorf("rejectPlusResourcesInOSS() returned no errors for upstream: %v", test.upstream)
		}

		allErrsPlus := rejectPlusResourcesInOSS(*test.upstream, field.NewPath("upstreams"), true)

		if len(allErrsPlus) != 0 {
			t.Errorf("rejectPlusResourcesInOSS() returned no errors for upstream: %v", test.upstream)
		}
	}
}

func TestValidateQueue(t *testing.T) {
	tests := []struct {
		upstreamQueue *v1.UpstreamQueue
		msg           string
	}{
		{
			upstreamQueue: &v1.UpstreamQueue{Size: 10, Timeout: "10s"},
			msg:           "valid upstream queue with size and timeout",
		},
		{
			upstreamQueue: nil,
			msg:           "upstream queue nil",
		},
		{
			upstreamQueue: nil,
			msg:           "upstream queue nil in OSS",
		},
	}

	for _, test := range tests {
		allErrs := validateQueue(test.upstreamQueue, field.NewPath("queue"))
		if len(allErrs) != 0 {
			t.Errorf("validateQueue() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidateQueueFails(t *testing.T) {
	tests := []struct {
		upstreamQueue *v1.UpstreamQueue
		msg           string
	}{
		{
			upstreamQueue: &v1.UpstreamQueue{Size: -1, Timeout: "10s"},
			msg:           "upstream queue with invalid size",
		},
		{
			upstreamQueue: &v1.UpstreamQueue{Size: 10, Timeout: "-10"},
			msg:           "upstream queue with invalid timeout",
		},
	}

	for _, test := range tests {
		allErrs := validateQueue(test.upstreamQueue, field.NewPath("queue"))
		if len(allErrs) == 0 {
			t.Errorf("validateQueue() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateSessionCookie(t *testing.T) {
	tests := []struct {
		sc  *v1.SessionCookie
		msg string
	}{
		{
			sc:  &v1.SessionCookie{Enable: true, Name: "min"},
			msg: "min valid config",
		},
		{
			sc:  &v1.SessionCookie{Enable: true, Name: "test", Expires: "max"},
			msg: "valid config with expires max",
		},
		{
			sc: &v1.SessionCookie{
				Enable: true, Name: "test", Path: "/tea", Expires: "1", Domain: ".example.com", HTTPOnly: false, Secure: true,
			},

			msg: "max valid config",
		},
	}
	for _, test := range tests {
		allErrs := validateSessionCookie(test.sc, field.NewPath("sessionCookie"))
		if len(allErrs) != 0 {
			t.Errorf("validateSessionCookie() returned errors %v for valid input for the case of: %s", allErrs, test.msg)
		}
	}
}

func TestValidateSessionCookieFails(t *testing.T) {
	tests := []struct {
		sc  *v1.SessionCookie
		msg string
	}{
		{
			sc:  &v1.SessionCookie{Enable: true},
			msg: "missing required field: Name",
		},
		{
			sc:  &v1.SessionCookie{Enable: false},
			msg: "session cookie not enabled",
		},
		{
			sc:  &v1.SessionCookie{Enable: true, Name: "$ecret-Name"},
			msg: "invalid name format",
		},
		{
			sc:  &v1.SessionCookie{Enable: true, Name: "test", Expires: "EGGS"},
			msg: "invalid time format",
		},
		{
			sc:  &v1.SessionCookie{Enable: true, Name: "test", Path: "/ coffee"},
			msg: "invalid path format",
		},
	}
	for _, test := range tests {
		allErrs := validateSessionCookie(test.sc, field.NewPath("sessionCookie"))
		if len(allErrs) == 0 {
			t.Errorf("validateSessionCookie() returned no errors for invalid input for the case of: %v", test.msg)
		}
	}
}

func TestValidateRedirectStatusCode(t *testing.T) {
	tests := []struct {
		code int
	}{
		{code: 301},
		{code: 302},
		{code: 307},
		{code: 308},
	}
	for _, test := range tests {
		allErrs := validateRedirectStatusCode(test.code, field.NewPath("code"))
		if len(allErrs) != 0 {
			t.Errorf("validateRedirectStatusCode(%v) returned errors %v for valid input", test.code, allErrs)
		}
	}
}

func TestValidateRedirectStatusCodeFails(t *testing.T) {
	tests := []struct {
		code int
	}{
		{code: 309},
		{code: 299},
		{code: 305},
	}
	for _, test := range tests {
		allErrs := validateRedirectStatusCode(test.code, field.NewPath("code"))
		if len(allErrs) == 0 {
			t.Errorf("validateRedirectStatusCode(%v) returned no errors for invalid input", test.code)
		}
	}
}

func TestValidateVariable(t *testing.T) {
	var validVars = map[string]bool{
		"scheme":                 true,
		"http_x_forwarded_proto": true,
		"request_uri":            true,
		"host":                   true,
	}

	tests := []struct {
		nVar string
	}{
		{"scheme"},
		{"http_x_forwarded_proto"},
		{"request_uri"},
		{"host"},
	}
	for _, test := range tests {
		allErrs := validateVariable(test.nVar, validVars, field.NewPath("url"))
		if len(allErrs) != 0 {
			t.Errorf("validateVariable(%v) returned errors %v for valid input", test.nVar, allErrs)
		}
	}
}

func TestValidateVariableFails(t *testing.T) {
	var validVars = map[string]bool{
		"host": true,
	}

	tests := []struct {
		nVar string
	}{
		{""},
		{"hostinvalid.com"},
		{"$a"},
		{"host${host}"},
		{"host${host}}"},
		{"host$${host}"},
	}
	for _, test := range tests {
		allErrs := validateVariable(test.nVar, validVars, field.NewPath("url"))
		if len(allErrs) == 0 {
			t.Errorf("validateVariable(%v) returned no errors for invalid input", test.nVar)
		}
	}
}

func TestIsRegexOrExactMatch(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{
			path:     "/path",
			expected: false,
		},
		{
			path:     "~ .*\\.jpg",
			expected: true,
		},
		{
			path:     "=/exact/match",
			expected: true,
		},
	}

	for _, test := range tests {
		result := isRegexOrExactMatch(test.path)
		if result != test.expected {
			t.Errorf("isRegexOrExactMatch(%v) returned %v but expected %v", test.path, result, test.expected)
		}
	}
}

func TestValidateActionReturnBody(t *testing.T) {
	tests := []struct {
		body string
		msg  string
	}{
		{
			body: "Hello World",
			msg:  "single string",
		},
		{
			body: "${host}${request_uri}",
			msg:  "string with variables",
		},
		{
			body: "Could not complete request, please go to ${scheme}://www.${host}${request_uri}-2",
			msg:  "string with url and variables",
		},
		{
			body: "{abc} %&*()!@#",
			msg:  "string with symbols",
		},
		{
			body: "${http_authorization}",
			msg:  "special variable with name",
		},
		{
			body: `
				<html>
				<body>
				<h1>Hello</h1>
				</body>
				</html>`,
			msg: "string with multiple lines",
		},
	}

	for _, test := range tests {
		allErrs := validateActionReturnBody(test.body, field.NewPath("body"), returnBodySpecialVariables, returnBodyVariables)
		if len(allErrs) != 0 {
			t.Errorf("validateActionReturnBody(%v) returned errors %v for valid input for the case of: %v", test.body, allErrs, test.msg)
		}
	}
}

func TestValidateActionReturnBodyFails(t *testing.T) {
	tests := []struct {
		body string
		msg  string
	}{
		{
			body: "Request to $host",
			msg:  "invalid variable format",
		},
		{
			body: `Request to host failed "`,
			msg:  "unescaped double quotes",
		},
		{
			body: "Please access to ${something}.com",
			msg:  "invalid variable",
		},
	}

	for _, test := range tests {
		allErrs := validateActionReturnBody(test.body, field.NewPath("body"), returnBodySpecialVariables, returnBodyVariables)
		if len(allErrs) == 0 {
			t.Errorf("validateActionReturnBody(%v) returned no errors for invalid input for the case of: %v", test.body, test.msg)
		}
	}
}

func TestValidateActionReturnType(t *testing.T) {
	tests := []struct {
		defaultType string
		msg         string
	}{
		{
			defaultType: "application/json",
			msg:         "normal MIME type",
		},
		{
			defaultType: `\"application/json\"`,
			msg:         "double quotes escaped",
		},
	}

	for _, test := range tests {
		allErrs := validateActionReturnType(test.defaultType, field.NewPath("type"))
		if len(allErrs) != 0 {
			t.Errorf("validateActionReturnType(%v) returned errors %v for the case of: %v", test.defaultType, allErrs, test.msg)
		}
	}
}

func TestValidateActionReturnTypeFails(t *testing.T) {
	tests := []struct {
		defaultType string
		msg         string
	}{
		{
			defaultType: "application/{json}",
			msg:         "use of forbidden symbols",
		},
		{
			defaultType: "application;json",
			msg:         "use of forbidden symbols",
		},
		{
			defaultType: `"application/json"`,
			msg:         "double quotes not escaped",
		},
	}

	for _, test := range tests {
		allErrs := validateActionReturnType(test.defaultType, field.NewPath("type"))
		if len(allErrs) == 0 {
			t.Errorf("validateActionReturnType(%v) returned no errors for the case of: %v", test.defaultType, test.msg)
		}
	}
}

func TestValidateActionReturn(t *testing.T) {
	tests := []*v1.ActionReturn{
		{
			Body: "Hello World",
		},
		{
			Type: "application/json",
			Body: "Hello World",
		},
		{
			Code: 200,
			Type: "application/json",
			Body: "Hello World",
		},
	}

	for _, test := range tests {
		allErrs := validateActionReturn(test, field.NewPath("return"), returnBodySpecialVariables, returnBodyVariables)
		if len(allErrs) != 0 {
			t.Errorf("validateActionReturn(%v) returned errors for valid input", test)
		}
	}
}

func TestValidateActionReturnFails(t *testing.T) {
	tests := []*v1.ActionReturn{
		{},
		{
			Code: 301,
			Body: "Hello World",
		},
		{
			Code: 200,
			Type: `application/"json"`,
			Body: "Hello World",
		},
	}

	for _, test := range tests {
		allErrs := validateActionReturn(test, field.NewPath("return"), returnBodySpecialVariables, returnBodyVariables)
		if len(allErrs) == 0 {
			t.Errorf("validateActionReturn(%v) returned no errors for invalid input", test)
		}
	}
}

func TestValidateActionProxy(t *testing.T) {
	upstreamNames := map[string]sets.Empty{
		"upstream1": {},
	}
	path := "/path"
	actionProxy := &v1.ActionProxy{
		Upstream:    "upstream1",
		RewritePath: "/test",
	}

	allErrs := validateActionProxy(actionProxy, field.NewPath("proxy"), upstreamNames, path, false)

	if len(allErrs) != 0 {
		t.Errorf("validateActionProxy(%+v, %v, %v) returned errors for valid input: %v", actionProxy, upstreamNames, path, allErrs)
	}
}

func TestValidateActionProxyFails(t *testing.T) {
	upstreamNames := map[string]sets.Empty{
		"upstream1": {},
	}
	path := "/path"
	actionProxy := &v1.ActionProxy{
		Upstream: "",
	}

	allErrs := validateActionProxy(actionProxy, field.NewPath("proxy"), upstreamNames, path, false)

	if len(allErrs) == 0 {
		t.Errorf("validateActionProxy(%+v, %v, %v) returned no errors for invalid input", actionProxy, upstreamNames, path)
	}
}

func TestValidateActionProxyRewritePath(t *testing.T) {
	tests := []string{"/rewrite", "/rewrite", `/$2`}
	for _, test := range tests {
		allErrs := validateActionProxyRewritePath(test, field.NewPath("rewritePath"))
		if len(allErrs) != 0 {
			t.Errorf("validateActionProxyRewritePath(%v) returned errors for valid input: %v", test, allErrs)
		}
	}
}

func TestValidateActionProxyRewritePathFails(t *testing.T) {
	tests := []string{`/\d{3}`, `(`, "$request_uri"}
	for _, test := range tests {
		allErrs := validateActionProxyRewritePath(test, field.NewPath("rewritePath"))
		if len(allErrs) == 0 {
			t.Errorf("validateActionProxyRewritePath(%v) returned no errors for invalid input", test)
		}
	}
}

func TestValidateActionProxyRewritePathForRegexp(t *testing.T) {
	tests := []string{"/rewrite$1", "test", `/$2`, `\"test\"`}
	for _, test := range tests {
		allErrs := validateActionProxyRewritePathForRegexp(test, field.NewPath("rewritePath"))
		if len(allErrs) != 0 {
			t.Errorf("validateActionProxyRewritePathForRegexp(%v) returned errors for valid input: %v", test, allErrs)
		}
	}
}

func TestValidateActionProxyRewritePathForRegexpFails(t *testing.T) {
	tests := []string{"$request_uri", `"test"`, `test\`}
	for _, test := range tests {
		allErrs := validateActionProxyRewritePathForRegexp(test, field.NewPath("rewritePath"))
		if len(allErrs) == 0 {
			t.Errorf("validateActionProxyRewritePathForRegexp(%v) returned no errors for invalid input", test)
		}
	}
}

func TestValidateActionProxyRequestHeaders(t *testing.T) {
	requestHeaders := &v1.ProxyRequestHeaders{
		Set: []v1.Header{
			{
				Name:  "Host",
				Value: "nginx.org",
			},
		},
	}

	allErrs := validateActionProxyRequestHeaders(requestHeaders, field.NewPath("requestHeaders"))
	if len(allErrs) != 0 {
		t.Errorf("validateActionProxyRequestHeaders(%v) returned errors for valid input: %v", requestHeaders, allErrs)
	}
}

func TestValidateActionProxyRequestHeadersFails(t *testing.T) {
	requestHeaders := &v1.ProxyRequestHeaders{
		Set: []v1.Header{
			{
				Name:  "in va lid",
				Value: "",
			},
			{
				Name:  "Host",
				Value: "$var",
			},
			{
				Name:  "",
				Value: "nginx.org",
			},
		},
	}

	allErrs := validateActionProxyRequestHeaders(requestHeaders, field.NewPath("requestHeaders"))
	if len(allErrs) == 0 {
		t.Errorf("validateActionProxyRequestHeaders(%v) returned no errors for invalid input", requestHeaders)
	}
}

func TestValidateActionProxyResponseHeaders(t *testing.T) {
	tests := []struct {
		responseHeaders *v1.ProxyResponseHeaders
	}{
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Hide:   []string{"Header"},
				Pass:   []string{"Header"},
				Ignore: []string{"Expires"},
				Add: []v1.AddHeader{
					{
						Header: v1.Header{
							Name:  "Host",
							Value: "nginx.org",
						},
						Always: false,
					},
				},
			},
		},
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Hide: []string{"Header"},
			},
		},
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Pass: []string{"Header"},
			},
		},
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Ignore: []string{"Expires"},
			},
		},
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Add: []v1.AddHeader{
					{
						Header: v1.Header{
							Name:  "Host",
							Value: "nginx.org",
						},
						Always: false,
					},
				},
			},
		},
	}

	for _, test := range tests {
		allErrs := validateActionProxyResponseHeaders(test.responseHeaders, field.NewPath("responseHeaders"))
		if len(allErrs) != 0 {
			t.Errorf("validateActionProxyResponseHeaders(%v) returned errors for valid input: %v", test.responseHeaders, allErrs)
		}
	}
}

func TestValidateActionProxyResponseHeadersFails(t *testing.T) {
	tests := []struct {
		responseHeaders *v1.ProxyResponseHeaders
		msg             string
	}{
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Hide:   []string{""},
				Pass:   []string{""},
				Ignore: []string{""},
				Add: []v1.AddHeader{
					{
						Header: v1.Header{
							Name:  "",
							Value: "nginx.org",
						},
					},
				},
			},
			msg: "all fields invalid",
		},
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Hide: []string{"invalid header"},
			},
			msg: "invalid hide headers",
		},
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Pass: []string{"$invalid"},
			},
			msg: "invalid pass headers",
		},
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Ignore: []string{"1234 invalid"},
			},
			msg: "invalid ignore headers",
		},
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Add: []v1.AddHeader{
					{
						Header: v1.Header{
							Name:  "$invalid 123",
							Value: "nginx.org",
						},
						Always: false,
					},
				},
			},
			msg: "invalid Add header name",
		},
		{
			responseHeaders: &v1.ProxyResponseHeaders{
				Add: []v1.AddHeader{
					{
						Header: v1.Header{
							Name:  "Host",
							Value: "$invalid",
						},
						Always: false,
					},
				},
			},
			msg: "invalid Add header value",
		},
	}

	for _, test := range tests {
		allErrs := validateActionProxyResponseHeaders(test.responseHeaders, field.NewPath("responseHeaders"))
		if len(allErrs) == 0 {
			t.Errorf("validateActionProxyResponseHeaders(%v) returned no errors for invalid input for the case of %v", test.responseHeaders, test.msg)
		}
	}
}

func TestValidateIgnoreHeaders(t *testing.T) {
	var ignoreHeaders []string

	for header := range validIgnoreHeaders {
		ignoreHeaders = append(ignoreHeaders, header)
	}

	allErrs := validateIgnoreHeaders(ignoreHeaders, field.NewPath("ignoreHeaders"))
	if len(allErrs) != 0 {
		t.Errorf("validateIgnoreHeaders(%v) returned errors for valid input: %v", ignoreHeaders, allErrs)
	}
}

func TestValidateIgnoreHeadersFails(t *testing.T) {
	ignoreHeaders := []string{
		"Host",
		"Connection",
	}

	allErrs := validateIgnoreHeaders(ignoreHeaders, field.NewPath("ignoreHeaders"))
	if len(allErrs) == 0 {
		t.Errorf("validateIgnoreHeaders(%v) returned no errors for invalid input", ignoreHeaders)
	}
}

func TestValidateStringNoVariables(t *testing.T) {
	tests := []string{
		"string",
		"endWith$",
		"withNumber$1",
		"abcййй",
		"abcййй$1",
		"",
	}

	for _, test := range tests {
		allErrs := validateStringNoVariables(test, field.NewPath("rewritePath"))
		if len(allErrs) != 0 {
			t.Errorf("validateStringNoVariables(%v) returned errors for valid input: %v", test, allErrs)
		}
	}
}

func TestValidateStringNoVariablesFails(t *testing.T) {
	tests := []string{
		"$var",
		"abcйй$й",
		"$$",
	}

	for _, test := range tests {
		allErrs := validateStringNoVariables(test, field.NewPath("rewritePath"))
		if len(allErrs) == 0 {
			t.Errorf("validateStringNoVariables(%v) returned no errors for invalid input", test)
		}
	}
}

func TestValidateStringWithVariables(t *testing.T) {
	testStrings := []string{
		"",
		"${scheme}",
		"${scheme}${host}",
		"foo.bar",
	}
	validVars := map[string]bool{"scheme": true, "host": true}

	for _, test := range testStrings {
		allErrs := validateStringWithVariables(test, field.NewPath("string"), nil, validVars)
		if len(allErrs) != 0 {
			t.Errorf("validateStringWithVariables(%v) returned errors for valid input: %v", test, allErrs)
		}
	}

	specialVars := []string{"arg", "http", "cookie"}
	testStringsSpecial := []string{
		"${arg_username}",
		"${http_header_name}",
		"${cookie_cookie_name}",
	}

	for _, test := range testStringsSpecial {
		allErrs := validateStringWithVariables(test, field.NewPath("string"), specialVars, validVars)
		if len(allErrs) != 0 {
			t.Errorf("validateStringWithVariables(%v) returned errors for valid input: %v", test, allErrs)
		}
	}
}

func TestValidateStringWithVariablesFail(t *testing.T) {
	testStrings := []string{
		"$scheme}",
		"${sch${eme}${host}",
		"host$",
		"${host",
		"${invalid}",
	}
	validVars := map[string]bool{"scheme": true, "host": true}

	for _, test := range testStrings {
		allErrs := validateStringWithVariables(test, field.NewPath("string"), nil, validVars)
		if len(allErrs) == 0 {
			t.Errorf("validateStringWithVariables(%v) returned no errors for invalid input", test)
		}
	}

	specialVars := []string{"arg", "http", "cookie"}
	testStringsSpecial := []string{
		"${arg_username%}",
		"${http_header-name}",
		"${cookie_cookie?name}",
	}

	for _, test := range testStringsSpecial {
		allErrs := validateStringWithVariables(test, field.NewPath("string"), specialVars, validVars)
		if len(allErrs) == 0 {
			t.Errorf("validateStringWithVariables(%v) returned no errors for invalid input", test)
		}
	}
}

func TestValidateActionReturnCode(t *testing.T) {
	codes := []int{200, 201, 400, 404, 500, 502, 599}
	for _, c := range codes {
		allErrs := validateActionReturnCode(c, field.NewPath("code"))
		if len(allErrs) != 0 {
			t.Errorf("validateActionReturnCode(%v) returned errors for valid input: %v", c, allErrs)
		}
	}
}

func TestValidateActionReturnCodeFails(t *testing.T) {
	codes := []int{0, -1, 199, 300, 399, 600, 999}
	for _, c := range codes {
		allErrs := validateActionReturnCode(c, field.NewPath("code"))
		if len(allErrs) == 0 {
			t.Errorf("validateActionReturnCode(%v) returned no errors for invalid input", c)
		}
	}
}

func TestValidateSpecialVariable(t *testing.T) {
	specialVars := []string{"arg_username", "arg_user_name", "http_header_name", "cookie_cookie_name"}
	for _, v := range specialVars {
		allErrs := validateSpecialVariable(v, field.NewPath("variable"))
		if len(allErrs) != 0 {
			t.Errorf("validateSpecialVariable(%v) returned errors for valid case: %v", v, allErrs)
		}
	}
}

func TestValidateSpecialVariableFails(t *testing.T) {
	specialVars := []string{"arg_invalid%", "http_header+invalid", "cookie_cookie_name?invalid"}
	for _, v := range specialVars {
		allErrs := validateSpecialVariable(v, field.NewPath("variable"))
		if len(allErrs) == 0 {
			t.Errorf("validateSpecialVariable(%v) returned no errors for invalid case", v)
		}
	}
}

func TestErrorPageHasRequiredFields(t *testing.T) {
	tests := []struct {
		errorPage v1.ErrorPage
		expected  bool
	}{
		{
			errorPage: v1.ErrorPage{
				Codes:    nil,
				Return:   nil,
				Redirect: nil,
			},
			expected: false,
		},
		{
			errorPage: v1.ErrorPage{
				Codes:    nil,
				Return:   &v1.ErrorPageReturn{},
				Redirect: &v1.ErrorPageRedirect{},
			},
			expected: false,
		},
		{
			errorPage: v1.ErrorPage{
				Codes:    nil,
				Return:   &v1.ErrorPageReturn{},
				Redirect: nil,
			},
			expected: true,
		},
		{
			errorPage: v1.ErrorPage{
				Codes:    nil,
				Return:   nil,
				Redirect: &v1.ErrorPageRedirect{},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		result := errorPageHasRequiredFields(test.errorPage)
		if result != test.expected {
			t.Errorf("errorPageHasRequiredFields(%v) returned %v but expected %v", test.errorPage, result, test.expected)
		}
	}
}

func TestValidateErrorPage(t *testing.T) {
	tests := []v1.ErrorPage{
		{
			Codes: []int{400, 404},
			Return: &v1.ErrorPageReturn{
				ActionReturn: v1.ActionReturn{
					Body: "Hello World",
				},
			},
			Redirect: nil,
		},
		{
			Codes:  []int{400, 404},
			Return: nil,
			Redirect: &v1.ErrorPageRedirect{
				ActionRedirect: v1.ActionRedirect{
					URL: "http://nginx.com",
				},
			},
		},
	}

	for _, ep := range tests {
		allErrs := validateErrorPage(ep, field.NewPath("errorPage"))
		if len(allErrs) != 0 {
			t.Errorf("validateErrorPage(%v) returned errors for valid input: %v", ep, allErrs)
		}
	}
}

func TestValidateErrorPageFails(t *testing.T) {
	tests := []v1.ErrorPage{
		{},
		{
			Codes:    []int{400, 404},
			Return:   &v1.ErrorPageReturn{},
			Redirect: &v1.ErrorPageRedirect{},
		},
		{
			Codes:    []int{100, 700},
			Return:   &v1.ErrorPageReturn{},
			Redirect: nil,
		},
		{
			Codes:    nil,
			Return:   &v1.ErrorPageReturn{},
			Redirect: nil,
		},
	}

	for _, ep := range tests {
		allErrs := validateErrorPage(ep, field.NewPath("errorPage"))
		if len(allErrs) == 0 {
			t.Errorf("validateErrorPage(%v) returned no errors for invalid input", ep)
		}
	}
}

func TestValidateErrorPageReturn(t *testing.T) {
	tests := []v1.ErrorPageReturn{
		{
			ActionReturn: v1.ActionReturn{
				Code: 200,
				Type: "",
				Body: "Could not process request, try again",
			},
			Headers: nil,
		},
		{
			ActionReturn: v1.ActionReturn{
				Code: 0,
				Type: "",
				Body: "Could not process request, try again. Upstream status ${upstream_status}",
			},
			Headers: []v1.Header{
				{
					Name:  "Set-Cookie",
					Value: "mycookie=true",
				},
			},
		},
		{
			ActionReturn: v1.ActionReturn{
				Code: 200,
				Type: "application/json",
				Body: `{\"message\": \"Could not process request, try again\", \"upstream_status\": \"${upstream_status}\"}`,
			},
			Headers: nil,
		},
	}

	for _, epr := range tests {
		allErrs := validateErrorPageReturn(&epr, field.NewPath("return"))
		if len(allErrs) != 0 {
			t.Errorf("validateErrorPageReturn(%v) returned errors for valid input: %v", epr, allErrs)
		}
	}
}

func TestValidateErrorPageReturnFails(t *testing.T) {
	tests := []struct {
		msg string
		epr v1.ErrorPageReturn
	}{
		{
			msg: "empty body",
			epr: v1.ErrorPageReturn{
				ActionReturn: v1.ActionReturn{
					Code: 200,
					Type: "application/json",
					Body: "",
				},
			},
		},
		{
			msg: "unescaped double quotes",
			epr: v1.ErrorPageReturn{
				ActionReturn: v1.ActionReturn{
					Code: 200,
					Type: "",
					Body: ` "Oops, Could not process request"`,
				},
			},
		},
		{
			msg: "invalid variable",
			epr: v1.ErrorPageReturn{
				ActionReturn: v1.ActionReturn{
					Code: 0,
					Type: "",
					Body: "Could not process request, response with invalid var: ${invalid}",
				},
			},
		},
		{
			msg: "invalid cookie name",
			epr: v1.ErrorPageReturn{
				ActionReturn: v1.ActionReturn{
					Code: 200,
					Type: "application/json",
					Body: `{\"message\": \"Could not process request, try again\", \"status\": \"${status}\"}`,
				},
				Headers: []v1.Header{
					{
						Name:  "Set-Cookie$_%^$  -",
						Value: "mycookie=true",
					},
				},
			},
		},
	}

	for _, test := range tests {
		allErrs := validateErrorPageReturn(&test.epr, field.NewPath("return"))
		if len(allErrs) == 0 {
			t.Errorf("validateErrorPageReturn(%v) returned no errors for invalid input for the case of %v", test.epr, test.msg)
		}
	}
}

func TestValidateErrorPageRedirect(t *testing.T) {
	tests := []v1.ErrorPageRedirect{
		{
			ActionRedirect: v1.ActionRedirect{
				URL:  "http://nginx.com",
				Code: 301,
			},
		},
		{
			ActionRedirect: v1.ActionRedirect{
				URL:  "${scheme}://nginx.com",
				Code: 302,
			},
		},
	}

	for _, epr := range tests {
		allErrs := validateErrorPageRedirect(&epr, field.NewPath("redirect"))
		if len(allErrs) != 0 {
			t.Errorf("validateErrorPageRedirect(%v) returned errors for valid input: %v", epr, allErrs)
		}
	}
}

func TestValidateErrorPageRedirectFails(t *testing.T) {
	tests := []v1.ErrorPageRedirect{
		{
			ActionRedirect: v1.ActionRedirect{
				URL:  "",
				Code: 301,
			},
		},
		{
			ActionRedirect: v1.ActionRedirect{
				URL:  `"http://nginx.com"`,
				Code: 301,
			},
		},
		{
			ActionRedirect: v1.ActionRedirect{
				URL:  "http://nginx.com",
				Code: 100,
			},
		},
		{
			ActionRedirect: v1.ActionRedirect{
				URL:  "$scheme://nginx.com",
				Code: 302,
			},
		},
		{
			ActionRedirect: v1.ActionRedirect{
				URL:  "https://${host}.com",
				Code: 302,
			},
		},
	}

	for _, epr := range tests {
		allErrs := validateErrorPageRedirect(&epr, field.NewPath("redirect"))
		if len(allErrs) == 0 {
			t.Errorf("validateErrorPageRedirect(%v) returned no errors for invalid input", epr)
		}
	}
}

func TestValidateErrorPageHeader(t *testing.T) {
	tests := []v1.Header{
		{
			Name:  "Header-Name",
			Value: "",
		},
		{
			Name:  "Header-Name",
			Value: "Value",
		},
		{
			Name:  "Header-Name",
			Value: "${upstream_status}",
		},
	}

	for _, test := range tests {
		allErrs := validateErrorPageHeader(test, field.NewPath("header"))
		if len(allErrs) != 0 {
			t.Errorf("validateErrorPageHeader(%v) returned errors for valid input", test)
		}
	}
}

func TestValidateErrorPageHeaderFails(t *testing.T) {
	tests := []v1.Header{
		{
			Name:  "",
			Value: "",
		},
		{
			Name:  "Header-!!#Name",
			Value: "",
		},
		{
			Name:  "Header-Name",
			Value: "$novar",
		},
		{
			Name:  "Header-Name",
			Value: "${invalid_var}",
		}, {
			Name:  "Header-Name",
			Value: `unescaped "`,
		},
	}

	for _, test := range tests {
		allErrs := validateErrorPageHeader(test, field.NewPath("header"))
		if len(allErrs) == 0 {
			t.Errorf("validateErrorPageHeader(%v) returned no errors for invalid input", test)
		}
	}
}

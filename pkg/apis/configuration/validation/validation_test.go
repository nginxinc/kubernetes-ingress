package validation

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateVirtualServer(t *testing.T) {
	virtualServer := v1alpha1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServerSpec{
			Host: "example.com",
			TLS: &v1alpha1.TLS{
				Secret: "abc",
			},
			Upstreams: []v1alpha1.Upstream{
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
			Routes: []v1alpha1.Route{
				{
					Path:     "/first",
					Upstream: "first",
				},
				{
					Path:     "/second",
					Upstream: "second",
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

func TestValidateTLS(t *testing.T) {
	validTLSes := []*v1alpha1.TLS{
		nil,
		{
			Secret: "my-secret",
		},
	}

	for _, tls := range validTLSes {
		allErrs := validateTLS(tls, field.NewPath("tls"))
		if len(allErrs) > 0 {
			t.Errorf("validateTLS() returned errors %v for valid input %v", allErrs, tls)
		}
	}

	invalidTLSes := []*v1alpha1.TLS{
		{
			Secret: "",
		},
		{
			Secret: "-",
		},
		{
			Secret: "a/b",
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
		upstreams             []v1alpha1.Upstream
		expectedUpstreamNames sets.String
		msg                   string
	}{
		{
			upstreams:             []v1alpha1.Upstream{},
			expectedUpstreamNames: sets.String{},
			msg:                   "no upstreams",
		},
		{
			upstreams: []v1alpha1.Upstream{
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
		upstreams             []v1alpha1.Upstream
		expectedUpstreamNames sets.String
		msg                   string
	}{
		{
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
				{
					Name:    "upstream1",
					Service: "test-1",
					Port:    80,
					ProxyBuffers: &v1alpha1.UpstreamBuffers{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
			upstreams: []v1alpha1.Upstream{
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
		if len(allErrs) < 0 {
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
		routes        []v1alpha1.Route
		upstreamNames sets.String
		msg           string
	}{
		{
			routes:        []v1alpha1.Route{},
			upstreamNames: sets.String{},
			msg:           "no routes",
		},
		{
			routes: []v1alpha1.Route{
				{
					Path:     "/",
					Upstream: "test",
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
		routes        []v1alpha1.Route
		upstreamNames sets.String
		msg           string
	}{
		{
			routes: []v1alpha1.Route{
				{
					Path:     "/test",
					Upstream: "test-1",
				},
				{
					Path:     "/test",
					Upstream: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "duplicated paths",
		},

		{
			routes: []v1alpha1.Route{
				{
					Path:     "",
					Upstream: "",
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
		route                 v1alpha1.Route
		upstreamNames         sets.String
		isRouteFieldForbidden bool
		msg                   string
	}{
		{
			route: v1alpha1.Route{

				Path:     "/",
				Upstream: "test",
			},
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "valid route with upstream",
		},
		{
			route: v1alpha1.Route{
				Path: "/",
				Splits: []v1alpha1.Split{
					{
						Weight:   90,
						Upstream: "test-1",
					},
					{
						Weight:   10,
						Upstream: "test-2",
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
			route: v1alpha1.Route{
				Path: "/",
				Rules: &v1alpha1.Rules{
					Conditions: []v1alpha1.Condition{
						{
							Header: "x-version",
						},
					},
					Matches: []v1alpha1.Match{
						{
							Values: []string{
								"test-1",
							},
							Upstream: "test-1",
						},
					},
					DefaultUpstream: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "valid upstream with rules",
		},
		{
			route: v1alpha1.Route{

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
		route                 v1alpha1.Route
		upstreamNames         sets.String
		isRouteFieldForbidden bool
		msg                   string
	}{
		{
			route: v1alpha1.Route{
				Path:     "",
				Upstream: "test",
			},
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "empty path",
		},
		{
			route: v1alpha1.Route{
				Path:     "/test",
				Upstream: "-test",
			},
			upstreamNames:         sets.String{},
			isRouteFieldForbidden: false,
			msg:                   "invalid upstream",
		},
		{
			route: v1alpha1.Route{
				Path:     "/",
				Upstream: "test",
			},
			upstreamNames:         sets.String{},
			isRouteFieldForbidden: false,
			msg:                   "non-existing upstream",
		},
		{
			route: v1alpha1.Route{
				Path:     "/",
				Upstream: "test",
				Splits: []v1alpha1.Split{
					{
						Weight:   90,
						Upstream: "test-1",
					},
					{
						Weight:   10,
						Upstream: "test-2",
					},
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test":   {},
				"test-1": {},
				"test-2": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "both upstream and splits exist",
		},
		{
			route: v1alpha1.Route{
				Path:     "/",
				Upstream: "test",
				Rules: &v1alpha1.Rules{
					Conditions: []v1alpha1.Condition{
						{
							Header: "x-version",
						},
					},
					Matches: []v1alpha1.Match{
						{
							Values: []string{
								"test-1",
							},
							Upstream: "test-1",
						},
					},
					DefaultUpstream: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test":   {},
				"test-1": {},
				"test-2": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "both upstream and rules exist",
		},
		{
			route: v1alpha1.Route{
				Path: "/",
				Splits: []v1alpha1.Split{
					{
						Weight:   90,
						Upstream: "test-1",
					},
					{
						Weight:   10,
						Upstream: "test-2",
					},
				},
				Rules: &v1alpha1.Rules{
					Conditions: []v1alpha1.Condition{
						{
							Header: "x-version",
						},
					},
					Matches: []v1alpha1.Match{
						{
							Values: []string{
								"test-1",
							},
							Upstream: "test-1",
						},
					},
					DefaultUpstream: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			isRouteFieldForbidden: false,
			msg:                   "both splits and rules exist",
		},
		{
			route: v1alpha1.Route{
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
	splits := []v1alpha1.Split{
		{
			Weight:   90,
			Upstream: "test-1",
		},
		{
			Weight:   10,
			Upstream: "test-2",
		},
	}
	upstreamNames := map[string]sets.Empty{
		"test-1": {},
		"test-2": {},
	}

	allErrs := validateSplits(splits, field.NewPath("splits"), upstreamNames)
	if len(allErrs) > 0 {
		t.Errorf("validateSplits() returned errors %v for valid input", allErrs)
	}
}

func TestValidateSplitsFails(t *testing.T) {
	tests := []struct {
		splits        []v1alpha1.Split
		upstreamNames sets.String
		msg           string
	}{
		{
			splits: []v1alpha1.Split{
				{
					Weight:   90,
					Upstream: "test-1",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
			},
			msg: "only one split",
		},
		{
			splits: []v1alpha1.Split{
				{
					Weight:   123,
					Upstream: "test-1",
				},
				{
					Weight:   10,
					Upstream: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "invalid weight",
		},
		{
			splits: []v1alpha1.Split{
				{
					Weight:   99,
					Upstream: "test-1",
				},
				{
					Weight:   99,
					Upstream: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "invalid total weight",
		},
		{
			splits: []v1alpha1.Split{
				{
					Weight:   90,
					Upstream: "",
				},
				{
					Weight:   10,
					Upstream: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "invalid upstream",
		},
		{
			splits: []v1alpha1.Split{
				{
					Weight:   90,
					Upstream: "some-upstream",
				},
				{
					Weight:   10,
					Upstream: "test-2",
				},
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "non-existing upstream",
		},
	}

	for _, test := range tests {
		allErrs := validateSplits(test.splits, field.NewPath("splits"), test.upstreamNames)
		if len(allErrs) == 0 {
			t.Errorf("validateSplits() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateRules(t *testing.T) {
	rules := v1alpha1.Rules{
		Conditions: []v1alpha1.Condition{
			{
				Header: "x-version",
			},
		},
		Matches: []v1alpha1.Match{
			{
				Values: []string{
					"test-1",
				},
				Upstream: "test-1",
			},
		},
		DefaultUpstream: "test-2",
	}

	upstreamNames := map[string]sets.Empty{
		"test-1": {},
		"test-2": {},
	}

	allErrs := validateRules(&rules, field.NewPath("rules"), upstreamNames)
	if len(allErrs) > 0 {
		t.Errorf("validateRules() returned errors %v for valid input", allErrs)
	}
}

func TestValidateRulesFails(t *testing.T) {
	tests := []struct {
		rules         v1alpha1.Rules
		upstreamNames sets.String
		msg           string
	}{
		{
			rules: v1alpha1.Rules{
				Conditions: []v1alpha1.Condition{},
				Matches: []v1alpha1.Match{
					{
						Values: []string{
							"test-1",
						},
						Upstream: "test-1",
					},
				},
				DefaultUpstream: "test-2",
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test-2": {},
			},
			msg: "no conditions",
		},
		{
			rules: v1alpha1.Rules{
				Conditions: []v1alpha1.Condition{
					{
						Header: "x-version",
					},
				},
				Matches:         []v1alpha1.Match{},
				DefaultUpstream: "test-2",
			},
			upstreamNames: map[string]sets.Empty{
				"test-2": {},
			},
			msg: "no matches",
		},
		{
			rules: v1alpha1.Rules{
				Conditions: []v1alpha1.Condition{
					{
						Header: "x-version",
					},
				},
				Matches: []v1alpha1.Match{
					{
						Values: []string{
							"test-1",
						},
						Upstream: "test-1",
					},
				},
				DefaultUpstream: "",
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
			},
			msg: "no default upstream",
		},
		{
			rules: v1alpha1.Rules{
				Conditions: []v1alpha1.Condition{
					{
						Header: "x-version",
						Cookie: "user",
					},
				},
				Matches: []v1alpha1.Match{
					{
						Values: []string{
							"test-1",
						},
						Upstream: "test-1",
					},
				},
				DefaultUpstream: "test",
			},
			upstreamNames: map[string]sets.Empty{
				"test-1": {},
				"test":   {},
			},
			msg: "invalid values in a match",
		},
	}

	for _, test := range tests {
		allErrs := validateRules(&test.rules, field.NewPath("rules"), test.upstreamNames)
		if len(allErrs) == 0 {
			t.Errorf("validateRules() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateCondition(t *testing.T) {
	tests := []struct {
		condition v1alpha1.Condition
		msg       string
	}{
		{
			condition: v1alpha1.Condition{
				Header: "x-version",
			},
			msg: "valid header",
		},
		{
			condition: v1alpha1.Condition{
				Cookie: "my_cookie",
			},
			msg: "valid cookie",
		},
		{
			condition: v1alpha1.Condition{
				Argument: "arg",
			},
			msg: "valid argument",
		},
		{
			condition: v1alpha1.Condition{
				Variable: "$request_method",
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
		condition v1alpha1.Condition
		msg       string
	}{
		{
			condition: v1alpha1.Condition{},
			msg:       "empty condition",
		},
		{
			condition: v1alpha1.Condition{
				Header:   "x-version",
				Cookie:   "user",
				Argument: "answer",
				Variable: "$request_method",
			},
			msg: "invalid condition",
		},
		{
			condition: v1alpha1.Condition{
				Header: "x_version",
			},
			msg: "invalid header",
		},
		{
			condition: v1alpha1.Condition{
				Cookie: "my-cookie",
			},
			msg: "invalid cookie",
		},
		{
			condition: v1alpha1.Condition{
				Argument: "my-arg",
			},
			msg: "invalid argument",
		},
		{
			condition: v1alpha1.Condition{
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
	match := v1alpha1.Match{
		Values: []string{
			"value1",
			"value2",
		},
		Upstream: "test",
	}
	conditionsCount := 2
	upstreamNames := map[string]sets.Empty{
		"test": {},
	}

	allErrs := validateMatch(match, field.NewPath("match"), conditionsCount, upstreamNames)
	if len(allErrs) > 0 {
		t.Errorf("validateMatch() returned errors %v for valid input", allErrs)
	}
}

func TestValidateMatchFails(t *testing.T) {
	tests := []struct {
		match           v1alpha1.Match
		conditionsCount int
		upstreamNames   sets.String
		msg             string
	}{
		{
			match: v1alpha1.Match{
				Values:   []string{},
				Upstream: "test",
			},
			conditionsCount: 1,
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			msg: "invalid number of values",
		},
		{
			match: v1alpha1.Match{
				Values: []string{
					`abc"`,
				},
				Upstream: "test",
			},
			conditionsCount: 1,
			upstreamNames: map[string]sets.Empty{
				"test": {},
			},
			msg: "invalid value",
		},
		{
			match: v1alpha1.Match{
				Values: []string{
					"value",
				},
				Upstream: "-invalid",
			},
			conditionsCount: 1,
			upstreamNames:   map[string]sets.Empty{},
			msg:             "invalid upstream",
		},
	}

	for _, test := range tests {
		allErrs := validateMatch(test.match, field.NewPath("match"), test.conditionsCount, test.upstreamNames)
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
	virtualServerRoute := v1alpha1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServerRouteSpec{
			Host: "example.com",
			Upstreams: []v1alpha1.Upstream{
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
			Subroutes: []v1alpha1.Route{
				{
					Path:     "/test/first",
					Upstream: "first",
				},
				{
					Path:     "/test/second",
					Upstream: "second",
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
	virtualServerRoute := v1alpha1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee",
			Namespace: "default",
		},
		Spec: v1alpha1.VirtualServerRouteSpec{
			Host: "example.com",
			Upstreams: []v1alpha1.Upstream{
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
			Subroutes: []v1alpha1.Route{
				{
					Path:     "/test/first",
					Upstream: "first",
				},
				{
					Path:     "/test/second",
					Upstream: "second",
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
		routes        []v1alpha1.Route
		upstreamNames sets.String
		pathPrefix    string
		msg           string
	}{
		{
			routes:        []v1alpha1.Route{},
			upstreamNames: sets.String{},
			pathPrefix:    "/",
			msg:           "no routes",
		},
		{
			routes: []v1alpha1.Route{
				{
					Path:     "/",
					Upstream: "test",
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
		routes        []v1alpha1.Route
		upstreamNames sets.String
		pathPrefix    string
		msg           string
	}{
		{
			routes: []v1alpha1.Route{
				{
					Path:     "/test",
					Upstream: "test-1",
				},
				{
					Path:     "/test",
					Upstream: "test-2",
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
			routes: []v1alpha1.Route{
				{
					Path:     "",
					Upstream: "",
				},
			},
			upstreamNames: map[string]sets.Empty{},
			pathPrefix:    "",
			msg:           "invalid route",
		},
		{
			routes: []v1alpha1.Route{
				{
					Path:     "/",
					Upstream: "test-1",
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
	validbuff := &v1alpha1.UpstreamBuffers{Number: 8, Size: "8k"}
	allErrs := validateBuffer(validbuff, field.NewPath("buffers-field"))

	if len(allErrs) != 0 {
		t.Errorf("validateBuffer returned errors %v valid input %v", allErrs, validbuff)
	}

	invalidbuff := []*v1alpha1.UpstreamBuffers{
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
	hc := &v1alpha1.HealthCheck{
		Enable:   true,
		Path:     "/healthz",
		Interval: "4s",
		Jitter:   "2s",
		Fails:    3,
		Passes:   2,
		Port:     8080,
		TLS: &v1alpha1.UpstreamTLS{
			Enable: true,
		},
		ConnectTimeout: "1s",
		ReadTimeout:    "1s",
		SendTimeout:    "1s",
		Headers: []v1alpha1.Header{
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
		hc *v1alpha1.HealthCheck
	}{
		{
			hc: &v1alpha1.HealthCheck{
				Enable: true,
				Path:   "/healthz//;",
			},
		},
		{
			hc: &v1alpha1.HealthCheck{
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
		header v1alpha1.Header
	}{
		{
			header: v1alpha1.Header{
				Name:  "Host",
				Value: "my.service",
			},
		},
		{
			header: v1alpha1.Header{
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
		header v1alpha1.Header
		msg    string
	}{
		{
			header: v1alpha1.Header{
				Name:  "12378 qwe ",
				Value: "my.service",
			},
			msg: "Invalid name with spaces",
		},
		{
			header: v1alpha1.Header{
				Name:  "Host",
				Value: `"my.service`,
			},
			msg: `Invalid value with unescaped '"'`,
		},
		{
			header: v1alpha1.Header{
				Name:  "Host",
				Value: `my.service\`,
			},
			msg: "Invalid value with ending '\\'",
		},
		{
			header: v1alpha1.Header{
				Name:  "Host",
				Value: "$my.service",
			},
			msg: "Invalid value with '$' character",
		},
		{
			header: v1alpha1.Header{
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
	upstream := v1alpha1.Upstream{
		SlowStart:     "10s",
		HealthCheck:   &v1alpha1.HealthCheck{},
		SessionCookie: &v1alpha1.SessionCookie{},
	}

	allErrsPlus := rejectPlusResourcesInOSS(upstream, field.NewPath("upstreams").Index(0), true)
	allErrsOSS := rejectPlusResourcesInOSS(upstream, field.NewPath("upstreams").Index(0), false)

	if len(allErrsPlus) != 0 {
		t.Errorf("rejectPlusResourcesInOSS() returned errors %v for NGINX Plus for upstream %v", allErrsPlus, upstream)
	}

	if len(allErrsOSS) == 0 {
		t.Errorf("rejectPlusResourcesInOSS() returned no errors for NGINX OSS for upstream %v", upstream)
	}

}

func TestValidateQueue(t *testing.T) {
	tests := []struct {
		upstreamQueue *v1alpha1.UpstreamQueue
		fieldPath     *field.Path
		isPlus        bool
		msg           string
	}{
		{
			upstreamQueue: &v1alpha1.UpstreamQueue{Size: 10, Timeout: "10s"},
			fieldPath:     field.NewPath("queue"),
			isPlus:        true,
			msg:           "valid upstream queue with size and timeout",
		},
		{
			upstreamQueue: nil,
			fieldPath:     field.NewPath("queue"),
			isPlus:        true,
			msg:           "upstream queue nil",
		},
		{
			upstreamQueue: nil,
			fieldPath:     field.NewPath("queue"),
			isPlus:        false,
			msg:           "upstream queue nil in OSS",
		},
	}

	for _, test := range tests {
		allErrs := validateQueue(test.upstreamQueue, test.fieldPath, test.isPlus)
		if len(allErrs) != 0 {
			t.Errorf("validateQueue() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidateQueueFails(t *testing.T) {
	tests := []struct {
		upstreamQueue *v1alpha1.UpstreamQueue
		fieldPath     *field.Path
		isPlus        bool
		msg           string
	}{
		{
			upstreamQueue: &v1alpha1.UpstreamQueue{Size: -1, Timeout: "10s"},
			fieldPath:     field.NewPath("queue"),
			isPlus:        true,
			msg:           "upstream queue with invalid size",
		},
		{
			upstreamQueue: &v1alpha1.UpstreamQueue{Size: 10, Timeout: "-10"},
			fieldPath:     field.NewPath("queue"),
			isPlus:        true,
			msg:           "upstream queue with invalid timeout",
		},
		{
			upstreamQueue: &v1alpha1.UpstreamQueue{Size: 10, Timeout: "10s"},
			fieldPath:     field.NewPath("queue"),
			isPlus:        false,
			msg:           "upstream queue with valid size and timeout in OSS",
		},
	}

	for _, test := range tests {
		allErrs := validateQueue(test.upstreamQueue, test.fieldPath, test.isPlus)
		if len(allErrs) == 0 {
			t.Errorf("validateQueue() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateSessionCookie(t *testing.T) {
	tests := []struct {
		sc        *v1alpha1.SessionCookie
		fieldPath *field.Path
		msg       string
	}{
		{
			sc:        &v1alpha1.SessionCookie{Enable: true, Name: "min"},
			fieldPath: field.NewPath("sessionCookie"),
			msg:       "min valid config",
		},
		{
			sc:        &v1alpha1.SessionCookie{Enable: true, Name: "test", Expires: "max"},
			fieldPath: field.NewPath("sessionCookie"),
			msg:       "valid config with expires max",
		},
		{
			sc: &v1alpha1.SessionCookie{
				Enable: true, Name: "test", Path: "/tea", Expires: "1", Domain: ".example.com", HTTPOnly: false, Secure: true,
			},
			fieldPath: field.NewPath("sessionCookie"),
			msg:       "max valid config",
		},
	}
	for _, test := range tests {
		allErrs := validateSessionCookie(test.sc, test.fieldPath)
		if len(allErrs) != 0 {
			t.Errorf("validateSessionCookie() returned errors %v for valid input for the case of: %s", allErrs, test.msg)
		}
	}
}

func TestValidateSessionCookieFails(t *testing.T) {
	tests := []struct {
		sc        *v1alpha1.SessionCookie
		fieldPath *field.Path
		msg       string
	}{
		{
			sc:        &v1alpha1.SessionCookie{Enable: true},
			fieldPath: field.NewPath("sessionCookie"),
			msg:       "missing required field: Name",
		},
		{
			sc:        &v1alpha1.SessionCookie{Enable: false},
			fieldPath: field.NewPath("sessionCookie"),
			msg:       "session cookie not enabled",
		},
		{
			sc:        &v1alpha1.SessionCookie{Enable: true, Name: "$ecret-Name"},
			fieldPath: field.NewPath("sessionCookie"),
			msg:       "invalid name format",
		},
		{
			sc:        &v1alpha1.SessionCookie{Enable: true, Name: "test", Expires: "EGGS"},
			fieldPath: field.NewPath("sessionCookie"),
			msg:       "invalid time format",
		},
		{
			sc:        &v1alpha1.SessionCookie{Enable: true, Name: "test", Path: "/ coffee"},
			fieldPath: field.NewPath("sessionCookie"),
			msg:       "invalid path format",
		},
	}
	for _, test := range tests {
		allErrs := validateSessionCookie(test.sc, test.fieldPath)
		if len(allErrs) == 0 {
			t.Errorf("validateSessionCookie() returned no errors for invalid input for the case of: %v", test.msg)
		}
	}
}

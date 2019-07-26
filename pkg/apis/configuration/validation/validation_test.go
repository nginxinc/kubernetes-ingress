package validation

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateVirtualServer(t *testing.T) {
	var keepalive = 32
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
					Keepalive: &keepalive,
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
				},
				{
					Name:                     "upstream2",
					Service:                  "test-2",
					Port:                     80,
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "10s",
					ProxyNextUpstreamTries:   5,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": sets.Empty{},
				"upstream2": sets.Empty{},
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
				"upstream1": sets.Empty{},
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
				"upstream1": sets.Empty{},
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
				"upstream1": sets.Empty{},
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
				"upstream1": sets.Empty{},
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
				"upstream1": sets.Empty{},
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
				"upstream1": sets.Empty{},
			},
			msg: "invalid upstream tries value",
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
				"test": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test": sets.Empty{},
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
				"test":   sets.Empty{},
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test":   sets.Empty{},
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
		"test": sets.Empty{},
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
		"test-1": sets.Empty{},
		"test-2": sets.Empty{},
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
				"test-1": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
		"test-1": sets.Empty{},
		"test-2": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test-2": sets.Empty{},
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
				"test-1": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test":   sets.Empty{},
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
		"test": sets.Empty{},
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
				"test": sets.Empty{},
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
				"test": sets.Empty{},
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
				"test": sets.Empty{},
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
				"test-1": sets.Empty{},
				"test-2": sets.Empty{},
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
				"test-1": sets.Empty{},
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

func TestValidatePositiveIntOrZero(t *testing.T) {
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
		allErrs := validatePositiveIntOrZero(test.number, field.NewPath("int-field"))

		if len(allErrs) != 0 {
			t.Errorf("validatePositiveInt returned errors for case: %v", test.msg)
		}
	}
}

func TestValidatePositiveIntOrZeroFails(t *testing.T) {
	number := createPointerFromInt(-1)
	allErrs := validatePositiveIntOrZero(number, field.NewPath("int-field"))

	if len(allErrs) == 0 {
		t.Error("validatePositiveInt returned no errors for case: invalid (-1)")
	}
}

func TestValidateTime(t *testing.T) {
	time := "1h 2s"
	allErrs := validateTime(time, field.NewPath("time-field"))

	if len(allErrs) != 0 {
		t.Errorf("validateTime returned errors %v valid input %v", allErrs, time)
	}
}

func TestValidateTimeFails(t *testing.T) {
	time := "invalid"
	allErrs := validateTime(time, field.NewPath("time-field"))

	if len(allErrs) == 0 {
		t.Errorf("validateTime returned no errors for invalid input %v", time)
	}
}

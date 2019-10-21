package configs

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVirtualServerExString(t *testing.T) {
	tests := []struct {
		input    *VirtualServerEx
		expected string
	}{
		{
			input: &VirtualServerEx{
				VirtualServer: &conf_v1alpha1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "cafe",
						Namespace: "default",
					},
				},
			},
			expected: "default/cafe",
		},
		{
			input:    &VirtualServerEx{},
			expected: "VirtualServerEx has no VirtualServer",
		},
		{
			input:    nil,
			expected: "<nil>",
		},
	}

	for _, test := range tests {
		result := test.input.String()
		if result != test.expected {
			t.Errorf("VirtualServerEx.String() returned %v but expected %v", result, test.expected)
		}
	}
}

func TestGenerateEndpointsKey(t *testing.T) {
	serviceNamespace := "default"
	serviceName := "test"
	var port uint16 = 80

	tests := []struct {
		subselector map[string]string
		expected    string
	}{
		{
			subselector: nil,
			expected:    "default/test:80",
		},
		{
			subselector: map[string]string{"version": "v1"},
			expected:    "default/test_version=v1:80",
		},
	}

	for _, test := range tests {
		result := GenerateEndpointsKey(serviceNamespace, serviceName, test.subselector, port)
		if result != test.expected {
			t.Errorf("GenerateEndpointsKey() returned %q but expected %q", result, test.expected)
		}

	}
}

func TestUpstreamNamerForVirtualServer(t *testing.T) {
	virtualServer := conf_v1alpha1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	upstreamNamer := newUpstreamNamerForVirtualServer(&virtualServer)
	upstream := "test"

	expected := "vs_default_cafe_test"

	result := upstreamNamer.GetNameForUpstream(upstream)
	if result != expected {
		t.Errorf("GetNameForUpstream() returned %q but expected %q", result, expected)
	}
}

func TestUpstreamNamerForVirtualServerRoute(t *testing.T) {
	virtualServer := conf_v1alpha1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	virtualServerRoute := conf_v1alpha1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee",
			Namespace: "default",
		},
	}
	upstreamNamer := newUpstreamNamerForVirtualServerRoute(&virtualServer, &virtualServerRoute)
	upstream := "test"

	expected := "vs_default_cafe_vsr_default_coffee_test"

	result := upstreamNamer.GetNameForUpstream(upstream)
	if result != expected {
		t.Errorf("GetNameForUpstream() returned %q but expected %q", result, expected)
	}
}

func TestVariableNamerSafeNsName(t *testing.T) {
	virtualServer := conf_v1alpha1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-test",
			Namespace: "default",
		},
	}

	expected := "default_cafe_test"

	variableNamer := newVariableNamer(&virtualServer)

	if variableNamer.safeNsName != expected {
		t.Errorf("newVariableNamer() returned variableNamer with safeNsName=%q but expected %q", variableNamer.safeNsName, expected)
	}
}

func TestVariableNamer(t *testing.T) {
	virtualServer := conf_v1alpha1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	variableNamer := newVariableNamer(&virtualServer)

	// GetNameForSplitClientVariable()
	index := 0

	expected := "$vs_default_cafe_splits_0"

	result := variableNamer.GetNameForSplitClientVariable(index)
	if result != expected {
		t.Errorf("GetNameForSplitClientVariable() returned %q but expected %q", result, expected)
	}

	// GetNameForVariableForRulesRouteMap()
	rulesIndex := 1
	matchIndex := 2
	conditionIndex := 3

	expected = "$vs_default_cafe_rules_1_match_2_cond_3"

	result = variableNamer.GetNameForVariableForRulesRouteMap(rulesIndex, matchIndex, conditionIndex)
	if result != expected {
		t.Errorf("GetNameForVariableForRulesRouteMap() returned %q but expected %q", result, expected)
	}

	// GetNameForVariableForRulesRouteMainMap()
	rulesIndex = 2

	expected = "$vs_default_cafe_rules_2"

	result = variableNamer.GetNameForVariableForRulesRouteMainMap(rulesIndex)
	if result != expected {
		t.Errorf("GetNameForVariableForRulesRouteMainMap() returned %q but expected %q", result, expected)
	}
}

func TestGenerateVirtualServerConfig(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1alpha1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.VirtualServerSpec{
				Host: "cafe.example.com",
				Upstreams: []conf_v1alpha1.Upstream{
					{
						Name:    "tea",
						Service: "tea-svc",
						Port:    80,
					},
					{
						Name:        "tea-latest",
						Service:     "tea-svc",
						Subselector: map[string]string{"version": "v1"},
						Port:        80,
					},
				},
				Routes: []conf_v1alpha1.Route{
					{
						Path:     "/tea",
						Upstream: "tea",
					},
					{
						Path:     "/tea-latest",
						Upstream: "tea-latest",
					},
					{
						Path:  "/coffee",
						Route: "default/coffee",
					},
					{
						Path:  "/subtea",
						Route: "default/subtea",
					},
				},
			},
		},
		Endpoints: map[string][]string{
			"default/tea-svc:80": {
				"10.0.0.20:80",
			},
			"default/tea-svc_version=v1:80": {
				"10.0.0.30:80",
			},
			"default/coffee-svc:80": {
				"10.0.0.40:80",
			},
			"default/sub-tea-svc_version=v1:80": {
				"10.0.0.50:80",
			},
		},
		VirtualServerRoutes: []*conf_v1alpha1.VirtualServerRoute{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee",
					Namespace: "default",
				},
				Spec: conf_v1alpha1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1alpha1.Upstream{
						{
							Name:    "coffee",
							Service: "coffee-svc",
							Port:    80,
						},
					},
					Subroutes: []conf_v1alpha1.Route{
						{
							Path:     "/coffee",
							Upstream: "coffee",
						},
					},
				},
			},
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "subtea",
					Namespace: "default",
				},
				Spec: conf_v1alpha1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1alpha1.Upstream{
						{
							Name:        "subtea",
							Service:     "sub-tea-svc",
							Port:        80,
							Subselector: map[string]string{"version": "v1"},
						},
					},
					Subroutes: []conf_v1alpha1.Route{
						{
							Path:     "/subtea",
							Upstream: "subtea",
						},
					},
				},
			},
		},
	}

	baseCfgParams := ConfigParams{
		ServerTokens:    "off",
		Keepalive:       16,
		ServerSnippets:  []string{"# server snippet"},
		ProxyProtocol:   true,
		SetRealIPFrom:   []string{"0.0.0.0/0"},
		RealIPHeader:    "X-Real-IP",
		RealIPRecursive: true,
		RedirectToHTTPS: true,
	}

	expected := version2.VirtualServerConfig{
		Upstreams: []version2.Upstream{
			{
				Name: "vs_default_cafe_tea",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.20:80",
					},
				},
				Keepalive: 16,
			},
			{
				Name: "vs_default_cafe_tea-latest",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.30:80",
					},
				},
				Keepalive: 16,
			},
			{
				Name: "vs_default_cafe_vsr_default_coffee_coffee",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.40:80",
					},
				},
				Keepalive: 16,
			},
			{
				Name: "vs_default_cafe_vsr_default_subtea_subtea",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.50:80",
					},
				},
				Keepalive: 16,
			},
		},
		Server: version2.Server{
			ServerName:                            "cafe.example.com",
			StatusZone:                            "cafe.example.com",
			ProxyProtocol:                         true,
			RedirectToHTTPSBasedOnXForwarderProto: true,
			ServerTokens:                          "off",
			SetRealIPFrom:                         []string{"0.0.0.0/0"},
			RealIPHeader:                          "X-Real-IP",
			RealIPRecursive:                       true,
			Snippets:                              []string{"# server snippet"},
			Locations: []version2.Location{
				{
					Path:                     "/tea",
					ProxyPass:                "http://vs_default_cafe_tea",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
				},
				{
					Path:                     "/tea-latest",
					ProxyPass:                "http://vs_default_cafe_tea-latest",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
				},
				{
					Path:                     "/coffee",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
				},
				{
					Path:                     "/subtea",
					ProxyPass:                "http://vs_default_cafe_vsr_default_subtea_subtea",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
				},
			},
		},
	}

	isPlus := false
	isResolverConfigured := false
	tlsPemFileName := ""
	vsc := newVirtualServerConfigurator(&baseCfgParams, isPlus, isResolverConfigured)
	result, warnings := vsc.GenerateVirtualServerConfig(&virtualServerEx, tlsPemFileName)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("GenerateVirtualServerConfig returned \n%v but expected \n%v", result, expected)
	}

	if len(warnings) != 0 {
		t.Errorf("GenerateVirtualServerConfig returned warnings: %v", vsc.warnings)
	}
}
func TestGenerateVirtualServerConfigForVirtualServerWithSplits(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1alpha1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.VirtualServerSpec{
				Host: "cafe.example.com",
				Upstreams: []conf_v1alpha1.Upstream{
					{
						Name:    "tea-v1",
						Service: "tea-svc-v1",
						Port:    80,
					},
					{
						Name:    "tea-v2",
						Service: "tea-svc-v2",
						Port:    80,
					},
				},
				Routes: []conf_v1alpha1.Route{
					{
						Path: "/tea",
						Splits: []conf_v1alpha1.Split{
							{
								Weight:   90,
								Upstream: "tea-v1",
							},
							{
								Weight:   10,
								Upstream: "tea-v2",
							},
						},
					},
					{
						Path:  "/coffee",
						Route: "default/coffee",
					},
				},
			},
		},
		Endpoints: map[string][]string{
			"default/tea-svc-v1:80": {
				"10.0.0.20:80",
			},
			"default/tea-svc-v2:80": {
				"10.0.0.21:80",
			},
			"default/coffee-svc-v1:80": {
				"10.0.0.30:80",
			},
			"default/coffee-svc-v2:80": {
				"10.0.0.31:80",
			},
		},
		VirtualServerRoutes: []*conf_v1alpha1.VirtualServerRoute{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee",
					Namespace: "default",
				},
				Spec: conf_v1alpha1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1alpha1.Upstream{
						{
							Name:    "coffee-v1",
							Service: "coffee-svc-v1",
							Port:    80,
						},
						{
							Name:    "coffee-v2",
							Service: "coffee-svc-v2",
							Port:    80,
						},
					},
					Subroutes: []conf_v1alpha1.Route{
						{
							Path: "/coffee",
							Splits: []conf_v1alpha1.Split{
								{
									Weight:   40,
									Upstream: "coffee-v1",
								},
								{
									Weight:   60,
									Upstream: "coffee-v2",
								},
							},
						},
					},
				},
			},
		},
	}

	baseCfgParams := ConfigParams{}

	expected := version2.VirtualServerConfig{
		Upstreams: []version2.Upstream{
			{
				Name: "vs_default_cafe_tea-v1",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.20:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_tea-v2",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.21:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_vsr_default_coffee_coffee-v1",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.30:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_vsr_default_coffee_coffee-v2",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.31:80",
					},
				},
			},
		},
		SplitClients: []version2.SplitClient{
			{
				Source:   "$request_id",
				Variable: "$vs_default_cafe_splits_0",
				Distributions: []version2.Distribution{
					{
						Weight: "90%",
						Value:  "@splits_0_split_0",
					},
					{
						Weight: "10%",
						Value:  "@splits_0_split_1",
					},
				},
			},
			{
				Source:   "$request_id",
				Variable: "$vs_default_cafe_splits_1",
				Distributions: []version2.Distribution{
					{
						Weight: "40%",
						Value:  "@splits_1_split_0",
					},
					{
						Weight: "60%",
						Value:  "@splits_1_split_1",
					},
				},
			},
		},
		Server: version2.Server{
			ServerName: "cafe.example.com",
			StatusZone: "cafe.example.com",
			InternalRedirectLocations: []version2.InternalRedirectLocation{
				{
					Path:        "/tea",
					Destination: "$vs_default_cafe_splits_0",
				},
				{
					Path:        "/coffee",
					Destination: "$vs_default_cafe_splits_1",
				},
			},
			Locations: []version2.Location{
				{
					Path:                     "@splits_0_split_0",
					ProxyPass:                "http://vs_default_cafe_tea-v1",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
				},
				{
					Path:                     "@splits_0_split_1",
					ProxyPass:                "http://vs_default_cafe_tea-v2",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
				},
				{
					Path:                     "@splits_1_split_0",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee-v1",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
				},
				{
					Path:                     "@splits_1_split_1",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee-v2",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
				},
			},
		},
	}

	isPlus := false
	isResolverConfigured := false
	tlsPemFileName := ""
	vsc := newVirtualServerConfigurator(&baseCfgParams, isPlus, isResolverConfigured)
	result, warnings := vsc.GenerateVirtualServerConfig(&virtualServerEx, tlsPemFileName)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("GenerateVirtualServerConfig returned \n%v but expected \n%v", result, expected)
	}

	if len(warnings) != 0 {
		t.Errorf("GenerateVirtualServerConfig returned warnings: %v", vsc.warnings)
	}
}

func TestGenerateVirtualServerConfigForVirtualServerWithRules(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1alpha1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.VirtualServerSpec{
				Host: "cafe.example.com",
				Upstreams: []conf_v1alpha1.Upstream{
					{
						Name:    "tea-v1",
						Service: "tea-svc-v1",
						Port:    80,
					},
					{
						Name:    "tea-v2",
						Service: "tea-svc-v2",
						Port:    80,
					},
				},
				Routes: []conf_v1alpha1.Route{
					{
						Path: "/tea",
						Rules: &conf_v1alpha1.Rules{
							Conditions: []conf_v1alpha1.Condition{
								{
									Header: "x-version",
								},
							},
							Matches: []conf_v1alpha1.Match{
								{
									Values: []string{
										"v2",
									},
									Upstream: "tea-v2",
								},
							},
							DefaultUpstream: "tea-v1",
						},
					},
					{
						Path:  "/coffee",
						Route: "default/coffee",
					},
				},
			},
		},
		Endpoints: map[string][]string{
			"default/tea-svc-v1:80": {
				"10.0.0.20:80",
			},
			"default/tea-svc-v2:80": {
				"10.0.0.21:80",
			},
			"default/coffee-svc-v1:80": {
				"10.0.0.30:80",
			},
			"default/coffee-svc-v2:80": {
				"10.0.0.31:80",
			},
		},
		VirtualServerRoutes: []*conf_v1alpha1.VirtualServerRoute{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee",
					Namespace: "default",
				},
				Spec: conf_v1alpha1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1alpha1.Upstream{
						{
							Name:    "coffee-v1",
							Service: "coffee-svc-v1",
							Port:    80,
						},
						{
							Name:    "coffee-v2",
							Service: "coffee-svc-v2",
							Port:    80,
						},
					},
					Subroutes: []conf_v1alpha1.Route{
						{
							Path: "/coffee",
							Rules: &conf_v1alpha1.Rules{
								Conditions: []conf_v1alpha1.Condition{
									{
										Argument: "version",
									},
								},
								Matches: []conf_v1alpha1.Match{
									{
										Values: []string{
											"v2",
										},
										Upstream: "coffee-v2",
									},
								},
								DefaultUpstream: "coffee-v1",
							},
						},
					},
				},
			},
		},
	}

	baseCfgParams := ConfigParams{}

	expected := version2.VirtualServerConfig{
		Upstreams: []version2.Upstream{
			{
				Name: "vs_default_cafe_tea-v1",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.20:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_tea-v2",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.21:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_vsr_default_coffee_coffee-v1",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.30:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_vsr_default_coffee_coffee-v2",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.31:80",
					},
				},
			},
		},
		Maps: []version2.Map{
			{
				Source:   "$http_x_version",
				Variable: "$vs_default_cafe_rules_0_match_0_cond_0",
				Parameters: []version2.Parameter{
					{
						Value:  `"v2"`,
						Result: "1",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$vs_default_cafe_rules_0_match_0_cond_0",
				Variable: "$vs_default_cafe_rules_0",
				Parameters: []version2.Parameter{
					{
						Value:  "~^1",
						Result: "@rules_0_match_0",
					},
					{
						Value:  "default",
						Result: "@rules_0_default",
					},
				},
			},
			{
				Source:   "$arg_version",
				Variable: "$vs_default_cafe_rules_1_match_0_cond_0",
				Parameters: []version2.Parameter{
					{
						Value:  `"v2"`,
						Result: "1",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$vs_default_cafe_rules_1_match_0_cond_0",
				Variable: "$vs_default_cafe_rules_1",
				Parameters: []version2.Parameter{
					{
						Value:  "~^1",
						Result: "@rules_1_match_0",
					},
					{
						Value:  "default",
						Result: "@rules_1_default",
					},
				},
			},
		},
		Server: version2.Server{
			ServerName: "cafe.example.com",
			StatusZone: "cafe.example.com",
			InternalRedirectLocations: []version2.InternalRedirectLocation{
				{
					Path:        "/tea",
					Destination: "$vs_default_cafe_rules_0",
				},
				{
					Path:        "/coffee",
					Destination: "$vs_default_cafe_rules_1",
				},
			},
			Locations: []version2.Location{
				{
					Path:                     "@rules_0_match_0",
					ProxyPass:                "http://vs_default_cafe_tea-v2",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
				},
				{
					Path:                     "@rules_0_default",
					ProxyPass:                "http://vs_default_cafe_tea-v1",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
				},
				{
					Path:                     "@rules_1_match_0",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee-v2",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
				},
				{
					Path:                     "@rules_1_default",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee-v1",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
				},
			},
		},
	}

	isPlus := false
	isResolverConfigured := false
	tlsPemFileName := ""
	vsc := newVirtualServerConfigurator(&baseCfgParams, isPlus, isResolverConfigured)
	result, warnings := vsc.GenerateVirtualServerConfig(&virtualServerEx, tlsPemFileName)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("GenerateVirtualServerConfig returned \n%v but expected \n%v", result, expected)
	}

	if len(warnings) != 0 {
		t.Errorf("GenerateVirtualServerConfig returned warnings: %v", vsc.warnings)
	}
}

func TestGenerateUpstream(t *testing.T) {
	name := "test-upstream"
	upstream := conf_v1alpha1.Upstream{Service: name, Port: 80}
	endpoints := []string{
		"192.168.10.10:8080",
	}
	cfgParams := ConfigParams{
		LBMethod:         "random",
		MaxFails:         1,
		MaxConns:         0,
		FailTimeout:      "10s",
		Keepalive:        21,
		UpstreamZoneSize: "256k",
	}

	expected := version2.Upstream{
		Name: "test-upstream",
		Servers: []version2.UpstreamServer{
			{
				Address: "192.168.10.10:8080",
			},
		},
		MaxFails:         1,
		MaxConns:         0,
		FailTimeout:      "10s",
		LBMethod:         "random",
		Keepalive:        21,
		UpstreamZoneSize: "256k",
	}

	vsc := newVirtualServerConfigurator(&cfgParams, false, false)
	result := vsc.generateUpstream(&conf_v1alpha1.VirtualServer{}, name, upstream, false, endpoints)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateUpstream() returned %v but expected %v", result, expected)
	}

	if len(vsc.warnings) != 0 {
		t.Errorf("generateUpstream returned warnings for %v", upstream)
	}
}

func TestGenerateUpstreamWithKeepalive(t *testing.T) {
	name := "test-upstream"
	noKeepalive := 0
	keepalive := 32
	endpoints := []string{
		"192.168.10.10:8080",
	}

	tests := []struct {
		upstream  conf_v1alpha1.Upstream
		cfgParams *ConfigParams
		expected  version2.Upstream
		msg       string
	}{
		{
			conf_v1alpha1.Upstream{Keepalive: &keepalive, Service: name, Port: 80},
			&ConfigParams{Keepalive: 21},
			version2.Upstream{
				Name: "test-upstream",
				Servers: []version2.UpstreamServer{
					{
						Address: "192.168.10.10:8080",
					},
				},
				Keepalive: 32,
			},
			"upstream keepalive set, configparam set",
		},
		{
			conf_v1alpha1.Upstream{Service: name, Port: 80},
			&ConfigParams{Keepalive: 21},
			version2.Upstream{
				Name: "test-upstream",
				Servers: []version2.UpstreamServer{
					{
						Address: "192.168.10.10:8080",
					},
				},
				Keepalive: 21,
			},
			"upstream keepalive not set, configparam set",
		},
		{
			conf_v1alpha1.Upstream{Keepalive: &noKeepalive, Service: name, Port: 80},
			&ConfigParams{Keepalive: 21},
			version2.Upstream{
				Name: "test-upstream",
				Servers: []version2.UpstreamServer{
					{
						Address: "192.168.10.10:8080",
					},
				},
			},
			"upstream keepalive set to 0, configparam set",
		},
	}

	for _, test := range tests {
		vsc := newVirtualServerConfigurator(test.cfgParams, false, false)
		result := vsc.generateUpstream(&conf_v1alpha1.VirtualServer{}, name, test.upstream, false, endpoints)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateUpstream() returned %v but expected %v for the case of %v", result, test.expected, test.msg)
		}

		if len(vsc.warnings) != 0 {
			t.Errorf("generateUpstream() returned warnings for %v", test.upstream)
		}
	}
}

func TestGenerateUpstreamForExternalNameService(t *testing.T) {
	name := "test-upstream"
	endpoints := []string{"example.com"}
	upstream := conf_v1alpha1.Upstream{Service: name}
	cfgParams := ConfigParams{}

	expected := version2.Upstream{
		Name: name,
		Servers: []version2.UpstreamServer{
			{
				Address: "example.com",
			},
		},
		Resolve: true,
	}

	vsc := newVirtualServerConfigurator(&cfgParams, true, true)
	result := vsc.generateUpstream(&conf_v1alpha1.VirtualServer{}, name, upstream, true, endpoints)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateUpstream() returned %v but expected %v", result, expected)
	}

	if len(vsc.warnings) != 0 {
		t.Errorf("generateUpstream() returned warnings for %v", upstream)
	}
}

func TestGenerateProxyPassProtocol(t *testing.T) {
	tests := []struct {
		upstream conf_v1alpha1.Upstream
		expected string
	}{
		{
			upstream: conf_v1alpha1.Upstream{},
			expected: "http",
		},
		{
			upstream: conf_v1alpha1.Upstream{
				TLS: conf_v1alpha1.UpstreamTLS{
					Enable: true,
				},
			},
			expected: "https",
		},
	}

	for _, test := range tests {
		result := generateProxyPassProtocol(test.upstream.TLS.Enable)
		if result != test.expected {
			t.Errorf("generateProxyPassProtocol(%v) returned %v but expected %v", test.upstream.TLS.Enable, result, test.expected)
		}
	}
}

func TestGenerateString(t *testing.T) {
	tests := []struct {
		inputS   string
		expected string
	}{
		{
			inputS:   "http_404",
			expected: "http_404",
		},
		{
			inputS:   "",
			expected: "error timeout",
		},
	}

	for _, test := range tests {
		result := generateString(test.inputS, "error timeout")
		if result != test.expected {
			t.Errorf("generateString() return %v but expected %v", result, test.expected)
		}
	}
}

func TestGenerateBuffer(t *testing.T) {
	tests := []struct {
		inputS   *conf_v1alpha1.UpstreamBuffers
		expected string
	}{
		{
			inputS:   nil,
			expected: "8 4k",
		},
		{
			inputS:   &conf_v1alpha1.UpstreamBuffers{Number: 8, Size: "16K"},
			expected: "8 16K",
		},
	}

	for _, test := range tests {
		result := generateBuffers(test.inputS, "8 4k")
		if result != test.expected {
			t.Errorf("generateBuffer() return %v but expected %v", result, test.expected)
		}
	}
}

func TestGenerateLocation(t *testing.T) {
	cfgParams := ConfigParams{
		ProxyConnectTimeout:  "30s",
		ProxyReadTimeout:     "31s",
		ProxySendTimeout:     "32s",
		ClientMaxBodySize:    "1m",
		ProxyMaxTempFileSize: "1024m",
		ProxyBuffering:       true,
		ProxyBuffers:         "8 4k",
		ProxyBufferSize:      "4k",
		LocationSnippets:     []string{"# location snippet"},
	}
	path := "/"
	upstreamName := "test-upstream"

	expected := version2.Location{
		Path:                     "/",
		Snippets:                 []string{"# location snippet"},
		ProxyConnectTimeout:      "30s",
		ProxyReadTimeout:         "31s",
		ProxySendTimeout:         "32s",
		ClientMaxBodySize:        "1m",
		ProxyMaxTempFileSize:     "1024m",
		ProxyBuffering:           true,
		ProxyBuffers:             "8 4k",
		ProxyBufferSize:          "4k",
		ProxyPass:                "http://test-upstream",
		ProxyNextUpstream:        "error timeout",
		ProxyNextUpstreamTimeout: "0s",
		ProxyNextUpstreamTries:   0,
	}

	result := generateLocation(path, upstreamName, conf_v1alpha1.Upstream{}, &cfgParams)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateLocation() returned %v but expected %v", result, expected)
	}
}

func TestGenerateSSLConfig(t *testing.T) {
	tests := []struct {
		inputTLS            *conf_v1alpha1.TLS
		inputTLSPemFileName string
		inputCfgParams      *ConfigParams
		expected            *version2.SSL
		msg                 string
	}{
		{
			inputTLS:            nil,
			inputTLSPemFileName: "",
			inputCfgParams:      &ConfigParams{},
			expected:            nil,
			msg:                 "no TLS field",
		},
		{
			inputTLS: &conf_v1alpha1.TLS{
				Secret: "",
			},
			inputTLSPemFileName: "",
			inputCfgParams:      &ConfigParams{},
			expected:            nil,
			msg:                 "TLS field with empty secret",
		},
		{
			inputTLS: &conf_v1alpha1.TLS{
				Secret: "secret",
			},
			inputTLSPemFileName: "",
			inputCfgParams:      &ConfigParams{},
			expected: &version2.SSL{
				HTTP2:           false,
				Certificate:     pemFileNameForMissingTLSSecret,
				CertificateKey:  pemFileNameForMissingTLSSecret,
				Ciphers:         "NULL",
				RedirectToHTTPS: false,
			},
			msg: "secret doesn't exist in the cluster with HTTP2 and SSLRedirect disabled",
		},
		{
			inputTLS: &conf_v1alpha1.TLS{
				Secret: "secret",
			},
			inputTLSPemFileName: "secret.pem",
			inputCfgParams:      &ConfigParams{},
			expected: &version2.SSL{
				HTTP2:           false,
				Certificate:     "secret.pem",
				CertificateKey:  "secret.pem",
				Ciphers:         "",
				RedirectToHTTPS: false,
			},
			msg: "normal case with HTTP2 and SSLRedirect disabled",
		},
		{
			inputTLS: &conf_v1alpha1.TLS{
				Secret: "secret",
			},
			inputTLSPemFileName: "secret.pem",
			inputCfgParams: &ConfigParams{
				HTTP2:       true,
				SSLRedirect: true,
			},
			expected: &version2.SSL{
				HTTP2:           true,
				Certificate:     "secret.pem",
				CertificateKey:  "secret.pem",
				Ciphers:         "",
				RedirectToHTTPS: true,
			},
			msg: "normal case with HTTP2 and SSLRedirect enabled",
		},
	}

	for _, test := range tests {
		result := generateSSLConfig(test.inputTLS, test.inputTLSPemFileName, test.inputCfgParams)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateSSLConfig() returned %v but expected %v for the case of %s", result, test.expected, test.msg)
		}
	}
}

func TestCreateUpstreamsForPlus(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1alpha1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.VirtualServerSpec{
				Host: "cafe.example.com",
				Upstreams: []conf_v1alpha1.Upstream{
					{
						Name:    "tea",
						Service: "tea-svc",
						Port:    80,
					},
					{
						Name:    "test",
						Service: "test-svc",
						Port:    80,
					},
					{
						Name:        "subselector-test",
						Service:     "test-svc",
						Subselector: map[string]string{"vs": "works"},
						Port:        80,
					},
					{
						Name:    "external",
						Service: "external-svc",
						Port:    80,
					},
				},
				Routes: []conf_v1alpha1.Route{
					{
						Path:     "/tea",
						Upstream: "tea",
					},
					{
						Path:  "/coffee",
						Route: "default/coffee",
					},
					{
						Path:     "/external",
						Upstream: "external",
					},
				},
			},
		},
		Endpoints: map[string][]string{
			"default/tea-svc:80": {
				"10.0.0.20:80",
			},
			"default/test-svc:80": {},
			"default/test-svc_vs=works:80": {
				"10.0.0.30:80",
			},
			"default/coffee-svc:80": {
				"10.0.0.40:80",
			},
			"default/test-svc_vsr=works:80": {
				"10.0.0.50:80",
			},
			"default/external-svc:80": {
				"example.com:80",
			},
		},
		ExternalNameSvcs: map[string]bool{
			"default/external-svc": true,
		},
		VirtualServerRoutes: []*conf_v1alpha1.VirtualServerRoute{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee",
					Namespace: "default",
				},
				Spec: conf_v1alpha1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1alpha1.Upstream{
						{
							Name:    "coffee",
							Service: "coffee-svc",
							Port:    80,
						},
						{
							Name:        "subselector-test",
							Service:     "test-svc",
							Subselector: map[string]string{"vsr": "works"},
							Port:        80,
						},
					},
					Subroutes: []conf_v1alpha1.Route{
						{
							Path:     "/coffee",
							Upstream: "coffee",
						},
						{
							Path:     "/coffee/sub",
							Upstream: "subselector-test",
						},
					},
				},
			},
		},
	}

	expected := []version2.Upstream{
		{
			Name: "vs_default_cafe_tea",
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.20:80",
				},
			},
		},
		{
			Name:    "vs_default_cafe_test",
			Servers: nil,
		},
		{
			Name: "vs_default_cafe_subselector-test",
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.30:80",
				},
			},
		},
		{
			Name: "vs_default_cafe_vsr_default_coffee_coffee",
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.40:80",
				},
			},
		},
		{
			Name: "vs_default_cafe_vsr_default_coffee_subselector-test",
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.50:80",
				},
			},
		},
	}

	result := createUpstreamsForPlus(&virtualServerEx, &ConfigParams{})
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("createUpstreamsForPlus returned \n%v but expected \n%v", result, expected)
	}
}

func TestCreateUpstreamServersConfigForPlus(t *testing.T) {
	upstream := version2.Upstream{
		Servers: []version2.UpstreamServer{
			{
				Address: "10.0.0.20:80",
			},
		},
		MaxFails:    21,
		MaxConns:    16,
		FailTimeout: "30s",
		SlowStart:   "50s",
	}

	expected := nginx.ServerConfig{
		MaxFails:    21,
		MaxConns:    16,
		FailTimeout: "30s",
		SlowStart:   "50s",
	}

	result := createUpstreamServersConfigForPlus(upstream)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("createUpstreamServersConfigForPlus returned %v but expected %v", result, expected)
	}
}

func TestCreateUpstreamServersConfigForPlusNoUpstreams(t *testing.T) {
	noUpstream := version2.Upstream{}
	expected := nginx.ServerConfig{}

	result := createUpstreamServersConfigForPlus(noUpstream)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("createUpstreamServersConfigForPlus returned %v but expected %v", result, expected)
	}
}

func TestGenerateSplitRouteConfig(t *testing.T) {
	route := conf_v1alpha1.Route{
		Path: "/",
		Splits: []conf_v1alpha1.Split{
			{
				Weight:   90,
				Upstream: "coffee-v1",
			},
			{
				Weight:   10,
				Upstream: "coffee-v2",
			},
		},
	}
	virtualServer := conf_v1alpha1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	upstreamNamer := newUpstreamNamerForVirtualServer(&virtualServer)
	variableNamer := newVariableNamer(&virtualServer)
	index := 1

	expected := splitRouteCfg{
		SplitClient: version2.SplitClient{
			Source:   "$request_id",
			Variable: "$vs_default_cafe_splits_1",
			Distributions: []version2.Distribution{
				{
					Weight: "90%",
					Value:  "@splits_1_split_0",
				},
				{
					Weight: "10%",
					Value:  "@splits_1_split_1",
				},
			},
		},
		Locations: []version2.Location{
			{
				Path:                     "@splits_1_split_0",
				ProxyPass:                "http://vs_default_cafe_coffee-v1",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
			},
			{
				Path:                     "@splits_1_split_1",
				ProxyPass:                "http://vs_default_cafe_coffee-v2",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
			},
		},
		InternalRedirectLocation: version2.InternalRedirectLocation{
			Path:        "/",
			Destination: "$vs_default_cafe_splits_1",
		},
	}

	cfgParams := ConfigParams{}

	result := generateSplitRouteConfig(route, upstreamNamer, map[string]conf_v1alpha1.Upstream{}, variableNamer, index, &cfgParams)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateSplitRouteConfig() returned %v but expected %v", result, expected)
	}
}

func TestGenerateRulesRouteConfig(t *testing.T) {
	route := conf_v1alpha1.Route{
		Path: "/",
		Rules: &conf_v1alpha1.Rules{
			Conditions: []conf_v1alpha1.Condition{
				{
					Header: "x-version",
				},
				{
					Cookie: "user",
				},
				{
					Argument: "answer",
				},
				{
					Variable: "$request_method",
				},
			},
			Matches: []conf_v1alpha1.Match{
				{
					Values: []string{
						"v1",
						"john",
						"yes",
						"GET",
					},
					Upstream: "coffee-v1",
				},
				{
					Values: []string{
						"v2",
						"paul",
						"no",
						"POST",
					},
					Upstream: "coffee-v2",
				},
			},
			DefaultUpstream: "tea",
		},
	}
	virtualServer := conf_v1alpha1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	upstreamNamer := newUpstreamNamerForVirtualServer(&virtualServer)
	variableNamer := newVariableNamer(&virtualServer)
	index := 1

	expected := rulesRouteCfg{
		Maps: []version2.Map{
			{
				Source:   "$http_x_version",
				Variable: "$vs_default_cafe_rules_1_match_0_cond_0",
				Parameters: []version2.Parameter{
					{
						Value:  `"v1"`,
						Result: "$vs_default_cafe_rules_1_match_0_cond_1",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$cookie_user",
				Variable: "$vs_default_cafe_rules_1_match_0_cond_1",
				Parameters: []version2.Parameter{
					{
						Value:  `"john"`,
						Result: "$vs_default_cafe_rules_1_match_0_cond_2",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$arg_answer",
				Variable: "$vs_default_cafe_rules_1_match_0_cond_2",
				Parameters: []version2.Parameter{
					{
						Value:  `"yes"`,
						Result: "$vs_default_cafe_rules_1_match_0_cond_3",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$request_method",
				Variable: "$vs_default_cafe_rules_1_match_0_cond_3",
				Parameters: []version2.Parameter{
					{
						Value:  `"GET"`,
						Result: "1",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$http_x_version",
				Variable: "$vs_default_cafe_rules_1_match_1_cond_0",
				Parameters: []version2.Parameter{
					{
						Value:  `"v2"`,
						Result: "$vs_default_cafe_rules_1_match_1_cond_1",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$cookie_user",
				Variable: "$vs_default_cafe_rules_1_match_1_cond_1",
				Parameters: []version2.Parameter{
					{
						Value:  `"paul"`,
						Result: "$vs_default_cafe_rules_1_match_1_cond_2",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$arg_answer",
				Variable: "$vs_default_cafe_rules_1_match_1_cond_2",
				Parameters: []version2.Parameter{
					{
						Value:  `"no"`,
						Result: "$vs_default_cafe_rules_1_match_1_cond_3",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$request_method",
				Variable: "$vs_default_cafe_rules_1_match_1_cond_3",
				Parameters: []version2.Parameter{
					{
						Value:  `"POST"`,
						Result: "1",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$vs_default_cafe_rules_1_match_0_cond_0$vs_default_cafe_rules_1_match_1_cond_0",
				Variable: "$vs_default_cafe_rules_1",
				Parameters: []version2.Parameter{
					{
						Value:  "~^1",
						Result: "@rules_1_match_0",
					},
					{
						Value:  "~^01",
						Result: "@rules_1_match_1",
					},
					{
						Value:  "default",
						Result: "@rules_1_default",
					},
				},
			},
		},
		Locations: []version2.Location{
			{
				Path:                     "@rules_1_match_0",
				ProxyPass:                "http://vs_default_cafe_coffee-v1",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
			},
			{
				Path:                     "@rules_1_match_1",
				ProxyPass:                "http://vs_default_cafe_coffee-v2",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
			},
			{
				Path:                     "@rules_1_default",
				ProxyPass:                "http://vs_default_cafe_tea",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
			},
		},
		InternalRedirectLocation: version2.InternalRedirectLocation{
			Path:        "/",
			Destination: "$vs_default_cafe_rules_1",
		},
	}

	cfgParams := ConfigParams{}

	result := generateRulesRouteConfig(route, upstreamNamer, map[string]conf_v1alpha1.Upstream{}, variableNamer, index, &cfgParams)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateRulesRouteConfig() returned \n%v but expected \n%v", result, expected)
	}
}

func TestGenerateValueForRulesRouteMap(t *testing.T) {
	tests := []struct {
		input              string
		expectedValue      string
		expectedIsNegative bool
	}{
		{
			input:              "default",
			expectedValue:      `\default`,
			expectedIsNegative: false,
		},
		{
			input:              "!default",
			expectedValue:      `\default`,
			expectedIsNegative: true,
		},
		{
			input:              "hostnames",
			expectedValue:      `\hostnames`,
			expectedIsNegative: false,
		},
		{
			input:              "include",
			expectedValue:      `\include`,
			expectedIsNegative: false,
		},
		{
			input:              "volatile",
			expectedValue:      `\volatile`,
			expectedIsNegative: false,
		},
		{
			input:              "abc",
			expectedValue:      `"abc"`,
			expectedIsNegative: false,
		},
		{
			input:              "!abc",
			expectedValue:      `"abc"`,
			expectedIsNegative: true,
		},
		{
			input:              "",
			expectedValue:      `""`,
			expectedIsNegative: false,
		},
		{
			input:              "!",
			expectedValue:      `""`,
			expectedIsNegative: true,
		},
	}

	for _, test := range tests {
		resultValue, resultIsNegative := generateValueForRulesRouteMap(test.input)
		if resultValue != test.expectedValue {
			t.Errorf("generateValueForRulesRouteMap(%q) returned %q but expected %q as the value", test.input, resultValue, test.expectedValue)
		}
		if resultIsNegative != test.expectedIsNegative {
			t.Errorf("generateValueForRulesRouteMap(%q) returned %v but expected %v as the isNegative", test.input, resultIsNegative, test.expectedIsNegative)
		}
	}
}

func TestGenerateParametersForRulesRouteMap(t *testing.T) {
	tests := []struct {
		inputMatchedValue     string
		inputSuccessfulResult string
		expected              []version2.Parameter
	}{
		{
			inputMatchedValue:     "abc",
			inputSuccessfulResult: "1",
			expected: []version2.Parameter{
				{
					Value:  `"abc"`,
					Result: "1",
				},
				{
					Value:  "default",
					Result: "0",
				},
			},
		},
		{
			inputMatchedValue:     "!abc",
			inputSuccessfulResult: "1",
			expected: []version2.Parameter{
				{
					Value:  `"abc"`,
					Result: "0",
				},
				{
					Value:  "default",
					Result: "1",
				},
			},
		},
	}

	for _, test := range tests {
		result := generateParametersForRulesRouteMap(test.inputMatchedValue, test.inputSuccessfulResult)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateParametersForRulesRouteMap(%q, %q) returned %v but expected %v", test.inputMatchedValue, test.inputSuccessfulResult, result, test.expected)
		}
	}
}

func TestGetNameForSourceForRulesRouteMapFromCondition(t *testing.T) {
	tests := []struct {
		input    conf_v1alpha1.Condition
		expected string
	}{
		{
			input: conf_v1alpha1.Condition{
				Header: "x-version",
			},
			expected: "$http_x_version",
		},
		{
			input: conf_v1alpha1.Condition{
				Cookie: "mycookie",
			},
			expected: "$cookie_mycookie",
		},
		{
			input: conf_v1alpha1.Condition{
				Argument: "arg",
			},
			expected: "$arg_arg",
		},
		{
			input: conf_v1alpha1.Condition{
				Variable: "$request_method",
			},
			expected: "$request_method",
		},
	}

	for _, test := range tests {
		result := getNameForSourceForRulesRouteMapFromCondition(test.input)
		if result != test.expected {
			t.Errorf("getNameForSourceForRulesRouteMapFromCondition() returned %q but expected %q for input %v", result, test.expected, test.input)
		}
	}
}

func TestGenerateLBMethod(t *testing.T) {
	defaultMethod := "random two least_conn"

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: defaultMethod,
		},
		{
			input:    "round_robin",
			expected: "",
		},
		{
			input:    "random",
			expected: "random",
		},
	}
	for _, test := range tests {
		result := generateLBMethod(test.input, defaultMethod)
		if result != test.expected {
			t.Errorf("generateLBMethod() returned %q but expected %q for input '%v'", result, test.expected, test.input)
		}
	}
}

func TestUpstreamHasKeepalive(t *testing.T) {
	noKeepalive := 0
	keepalive := 32

	tests := []struct {
		upstream  conf_v1alpha1.Upstream
		cfgParams *ConfigParams
		expected  bool
		msg       string
	}{
		{
			conf_v1alpha1.Upstream{},
			&ConfigParams{Keepalive: keepalive},
			true,
			"upstream keepalive not set, configparam keepalive set",
		},
		{
			conf_v1alpha1.Upstream{Keepalive: &noKeepalive},
			&ConfigParams{Keepalive: keepalive},
			false,
			"upstream keepalive set to 0, configparam keepive set",
		},
		{
			conf_v1alpha1.Upstream{Keepalive: &keepalive},
			&ConfigParams{Keepalive: noKeepalive},
			true,
			"upstream keepalive set, configparam keepalive set to 0",
		},
	}

	for _, test := range tests {
		result := upstreamHasKeepalive(test.upstream, test.cfgParams)
		if result != test.expected {
			t.Errorf("upstreamHasKeepalive() returned %v, but expected %v for the case of %v", result, test.expected, test.msg)
		}
	}
}

func TestNewHealthCheckWithDefaults(t *testing.T) {
	upstreamName := "test-upstream"
	baseCfgParams := &ConfigParams{
		ProxySendTimeout:    "5s",
		ProxyReadTimeout:    "5s",
		ProxyConnectTimeout: "5s",
	}
	expected := &version2.HealthCheck{
		Name:                upstreamName,
		ProxySendTimeout:    "5s",
		ProxyReadTimeout:    "5s",
		ProxyConnectTimeout: "5s",
		ProxyPass:           fmt.Sprintf("http://%v", upstreamName),
		URI:                 "/",
		Interval:            "5s",
		Jitter:              "0s",
		Fails:               1,
		Passes:              1,
		Headers:             make(map[string]string),
	}

	result := newHealthCheckWithDefaults(conf_v1alpha1.Upstream{}, upstreamName, baseCfgParams)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("newHealthCheckWithDefaults returned \n%v but expected \n%v", result, expected)
	}
}

func TestGenerateHealthCheck(t *testing.T) {
	upstreamName := "test-upstream"
	tests := []struct {
		upstream     conf_v1alpha1.Upstream
		upstreamName string
		expected     *version2.HealthCheck
		msg          string
	}{
		{

			upstream: conf_v1alpha1.Upstream{
				HealthCheck: &conf_v1alpha1.HealthCheck{
					Enable:         true,
					Path:           "/healthz",
					Interval:       "5s",
					Jitter:         "2s",
					Fails:          3,
					Passes:         2,
					Port:           8080,
					ConnectTimeout: "20s",
					SendTimeout:    "20s",
					ReadTimeout:    "20s",
					Headers: []conf_v1alpha1.Header{
						{
							Name:  "Host",
							Value: "my.service",
						},
						{
							Name:  "User-Agent",
							Value: "nginx",
						},
					},
					StatusMatch: "! 500",
				},
			},
			upstreamName: upstreamName,
			expected: &version2.HealthCheck{
				Name:                upstreamName,
				ProxyConnectTimeout: "20s",
				ProxySendTimeout:    "20s",
				ProxyReadTimeout:    "20s",
				ProxyPass:           fmt.Sprintf("http://%v", upstreamName),
				URI:                 "/healthz",
				Interval:            "5s",
				Jitter:              "2s",
				Fails:               3,
				Passes:              2,
				Port:                8080,
				Headers: map[string]string{
					"Host":       "my.service",
					"User-Agent": "nginx",
				},
				Match: fmt.Sprintf("%v_match", upstreamName),
			},
			msg: "HealthCheck with changed parameters",
		},
		{
			upstream: conf_v1alpha1.Upstream{
				HealthCheck: &conf_v1alpha1.HealthCheck{
					Enable: true,
				},
				ProxyConnectTimeout: "30s",
				ProxyReadTimeout:    "30s",
				ProxySendTimeout:    "30s",
			},
			upstreamName: upstreamName,
			expected: &version2.HealthCheck{
				Name:                upstreamName,
				ProxyConnectTimeout: "30s",
				ProxyReadTimeout:    "30s",
				ProxySendTimeout:    "30s",
				ProxyPass:           fmt.Sprintf("http://%v", upstreamName),
				URI:                 "/",
				Interval:            "5s",
				Jitter:              "0s",
				Fails:               1,
				Passes:              1,
				Headers:             make(map[string]string),
			},
			msg: "HealthCheck with default parameters from Upstream",
		},
		{
			upstream: conf_v1alpha1.Upstream{
				HealthCheck: &conf_v1alpha1.HealthCheck{
					Enable: true,
				},
			},
			upstreamName: upstreamName,
			expected: &version2.HealthCheck{
				Name:                upstreamName,
				ProxyConnectTimeout: "5s",
				ProxyReadTimeout:    "5s",
				ProxySendTimeout:    "5s",
				ProxyPass:           fmt.Sprintf("http://%v", upstreamName),
				URI:                 "/",
				Interval:            "5s",
				Jitter:              "0s",
				Fails:               1,
				Passes:              1,
				Headers:             make(map[string]string),
			},
			msg: "HealthCheck with default parameters from ConfigMap (not defined in Upstream)",
		},
		{
			upstream:     conf_v1alpha1.Upstream{},
			upstreamName: upstreamName,
			expected:     nil,
			msg:          "HealthCheck not enabled",
		},
	}

	baseCfgParams := &ConfigParams{
		ProxySendTimeout:    "5s",
		ProxyReadTimeout:    "5s",
		ProxyConnectTimeout: "5s",
	}

	for _, test := range tests {
		result := generateHealthCheck(test.upstream, test.upstreamName, baseCfgParams)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateHealthCheck returned \n%v but expected \n%v \n for case: %v", result, test.expected, test.msg)
		}
	}
}

func TestGenerateEndpointsForUpstream(t *testing.T) {
	name := "test"
	namespace := "test-namespace"

	tests := []struct {
		upstream             conf_v1alpha1.Upstream
		vsEx                 *VirtualServerEx
		isPlus               bool
		isResolverConfigured bool
		expected             []string
		warningsExpected     bool
		msg                  string
	}{
		{
			upstream: conf_v1alpha1.Upstream{
				Service: name,
				Port:    80,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1alpha1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
				},
				Endpoints: map[string][]string{
					"test-namespace/test:80": {"example.com:80"},
				},
				ExternalNameSvcs: map[string]bool{
					"test-namespace/test": true,
				},
			},
			isPlus:               true,
			isResolverConfigured: true,
			expected:             []string{"example.com:80"},
			msg:                  "ExternalName service",
		},
		{
			upstream: conf_v1alpha1.Upstream{
				Service: name,
				Port:    80,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1alpha1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
				},
				Endpoints: map[string][]string{
					"test-namespace/test:80": {"example.com:80"},
				},
				ExternalNameSvcs: map[string]bool{
					"test-namespace/test": true,
				},
			},
			isPlus:               true,
			isResolverConfigured: false,
			warningsExpected:     true,
			expected:             []string{},
			msg:                  "ExternalName service without resolver configured",
		},
		{
			upstream: conf_v1alpha1.Upstream{
				Service: name,
				Port:    8080,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1alpha1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
				},
				Endpoints: map[string][]string{
					"test-namespace/test:8080": {"192.168.10.10:8080"},
				},
			},
			isPlus:               false,
			isResolverConfigured: false,
			expected:             []string{"192.168.10.10:8080"},
			msg:                  "Service with endpoints",
		},
		{
			upstream: conf_v1alpha1.Upstream{
				Service: name,
				Port:    8080,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1alpha1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
				},
				Endpoints: map[string][]string{},
			},
			isPlus:               false,
			isResolverConfigured: false,
			expected:             []string{nginx502Server},
			msg:                  "Service with no endpoints",
		},
		{
			upstream: conf_v1alpha1.Upstream{
				Service: name,
				Port:    8080,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1alpha1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
				},
				Endpoints: map[string][]string{},
			},
			isPlus:               true,
			isResolverConfigured: false,
			expected:             nil,
			msg:                  "Service with no endpoints",
		},
		{
			upstream: conf_v1alpha1.Upstream{
				Service:     name,
				Port:        8080,
				Subselector: map[string]string{"version": "test"},
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1alpha1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
				},
				Endpoints: map[string][]string{
					"test-namespace/test_version=test:8080": {"192.168.10.10:8080"},
				},
			},
			isPlus:               false,
			isResolverConfigured: false,
			expected:             []string{"192.168.10.10:8080"},
			msg:                  "Upstream with subselector, with a matching endpoint",
		},
		{
			upstream: conf_v1alpha1.Upstream{
				Service:     name,
				Port:        8080,
				Subselector: map[string]string{"version": "test"},
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1alpha1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
				},
				Endpoints: map[string][]string{
					"test-namespace/test:8080": {"192.168.10.10:8080"},
				},
			},
			isPlus:               false,
			isResolverConfigured: false,
			expected:             []string{nginx502Server},
			msg:                  "Upstream with subselector, without a matching endpoint",
		},
	}

	for _, test := range tests {
		vsc := newVirtualServerConfigurator(&ConfigParams{}, test.isPlus, test.isResolverConfigured)
		result := vsc.generateEndpointsForUpstream(test.vsEx.VirtualServer, namespace, test.upstream, test.vsEx)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateEndpointsForUpstream(isPlus=%v, isResolverConfigured=%v) returned %v, but expected %v for case: %v",
				test.isPlus, test.isResolverConfigured, result, test.expected, test.msg)
		}

		if len(vsc.warnings) == 0 && test.warningsExpected {
			t.Errorf("generateEndpointsForUpstream(isPlus=%v, isResolverConfigured=%v) didn't return any warnings for %v but warnings expected",
				test.isPlus, test.isResolverConfigured, test.upstream)
		}

		if len(vsc.warnings) != 0 && !test.warningsExpected {
			t.Errorf("generateEndpointsForUpstream(isPlus=%v, isResolverConfigured=%v) returned warnings for %v",
				test.isPlus, test.isResolverConfigured, test.upstream)
		}
	}
}

func TestGenerateSlowStartForPlusWithInCompatibleLBMethods(t *testing.T) {
	serviceName := "test-slowstart-with-incompatible-LBMethods"
	upstream := conf_v1alpha1.Upstream{Service: serviceName, Port: 80, SlowStart: "10s"}
	expected := ""

	var tests = []string{
		"random",
		"ip_hash",
		"hash 123",
		"random two",
		"random two least_conn",
		"random two least_time=header",
		"random two least_time=last_byte",
	}

	for _, lbMethod := range tests {
		vsc := newVirtualServerConfigurator(&ConfigParams{}, true, false)
		result := vsc.generateSlowStartForPlus(&conf_v1alpha1.VirtualServer{}, upstream, lbMethod)

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("generateSlowStartForPlus returned %v, but expected %v for lbMethod %v", result, expected, lbMethod)
		}

		if len(vsc.warnings) == 0 {
			t.Errorf("generateSlowStartForPlus returned no warnings for %v but warnings expected", upstream)
		}
	}

}

func TestGenerateSlowStartForPlus(t *testing.T) {
	serviceName := "test-slowstart"

	tests := []struct {
		upstream conf_v1alpha1.Upstream
		lbMethod string
		expected string
	}{
		{
			upstream: conf_v1alpha1.Upstream{Service: serviceName, Port: 80, SlowStart: "", LBMethod: "least_conn"},
			lbMethod: "least_conn",
			expected: "",
		},
		{
			upstream: conf_v1alpha1.Upstream{Service: serviceName, Port: 80, SlowStart: "10s", LBMethod: "least_conn"},
			lbMethod: "least_conn",
			expected: "10s",
		},
	}

	for _, test := range tests {
		vsc := newVirtualServerConfigurator(&ConfigParams{}, true, false)
		result := vsc.generateSlowStartForPlus(&conf_v1alpha1.VirtualServer{}, test.upstream, test.lbMethod)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateSlowStartForPlus returned %v, but expected %v", result, test.expected)
		}

		if len(vsc.warnings) != 0 {
			t.Errorf("generateSlowStartForPlus returned warnings for %v", test.upstream)
		}
	}
}

func TestCreateEndpointsFromUpstream(t *testing.T) {
	ups := version2.Upstream{
		Servers: []version2.UpstreamServer{
			{
				Address: "10.0.0.20:80",
			},
			{
				Address: "10.0.0.30:80",
			},
		},
	}

	expected := []string{
		"10.0.0.20:80",
		"10.0.0.30:80",
	}

	endpoints := createEndpointsFromUpstream(ups)
	if !reflect.DeepEqual(endpoints, expected) {
		t.Errorf("createEndpointsFromUpstream returned %v, but expected %v", endpoints, expected)
	}
}

func TestGenerateUpstreamWithQueue(t *testing.T) {
	serviceName := "test-queue"

	tests := []struct {
		name     string
		upstream conf_v1alpha1.Upstream
		isPlus   bool
		expected version2.Upstream
		msg      string
	}{
		{
			name: "test-upstream-queue",
			upstream: conf_v1alpha1.Upstream{Service: serviceName, Port: 80, Queue: &conf_v1alpha1.UpstreamQueue{
				Size:    10,
				Timeout: "10s",
			}},
			isPlus: true,
			expected: version2.Upstream{
				Name: "test-upstream-queue",
				Queue: &version2.Queue{
					Size:    10,
					Timeout: "10s",
				},
			},
			msg: "upstream queue with size and timeout",
		},
		{
			name:     "test-upstream-queue-with-default-timeout",
			upstream: conf_v1alpha1.Upstream{Service: serviceName, Port: 80, Queue: &conf_v1alpha1.UpstreamQueue{Size: 10, Timeout: ""}},
			isPlus:   true,
			expected: version2.Upstream{
				Name: "test-upstream-queue-with-default-timeout",
				Queue: &version2.Queue{
					Size:    10,
					Timeout: "60s",
				},
			},
			msg: "upstream queue with only size",
		},
		{
			name:     "test-upstream-queue-nil",
			upstream: conf_v1alpha1.Upstream{Service: serviceName, Port: 80, Queue: nil},
			isPlus:   false,
			expected: version2.Upstream{
				Name: "test-upstream-queue-nil",
			},
			msg: "upstream queue with nil for OSS",
		},
	}

	for _, test := range tests {
		vsc := newVirtualServerConfigurator(&ConfigParams{}, test.isPlus, false)
		result := vsc.generateUpstream(&conf_v1alpha1.VirtualServer{}, test.name, test.upstream, false, []string{})
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateUpstream() returned %v but expected %v for the case of %v", result, test.expected, test.msg)
		}
	}

}

func TestGenerateQueueForPlus(t *testing.T) {
	tests := []struct {
		upstreamQueue *conf_v1alpha1.UpstreamQueue
		expected      *version2.Queue
		msg           string
	}{
		{
			upstreamQueue: &conf_v1alpha1.UpstreamQueue{Size: 10, Timeout: "10s"},
			expected:      &version2.Queue{Size: 10, Timeout: "10s"},
			msg:           "upstream queue with size and timeout",
		},
		{
			upstreamQueue: nil,
			expected:      nil,
			msg:           "upstream queue with nil",
		},
		{
			upstreamQueue: &conf_v1alpha1.UpstreamQueue{Size: 10},
			expected:      &version2.Queue{Size: 10, Timeout: "60s"},
			msg:           "upstream queue with only size",
		},
	}

	for _, test := range tests {
		result := generateQueueForPlus(test.upstreamQueue, "60s")
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateQueueForPlus() returned %v but expected %v for the case of %v", result, test.expected, test.msg)
		}
	}

}

func TestGenerateSessionCookie(t *testing.T) {
	tests := []struct {
		sc       *conf_v1alpha1.SessionCookie
		expected *version2.SessionCookie
		msg      string
	}{
		{
			sc:       &conf_v1alpha1.SessionCookie{Enable: true, Name: "test"},
			expected: &version2.SessionCookie{Enable: true, Name: "test"},
			msg:      "session cookie with name",
		},
		{
			sc:       nil,
			expected: nil,
			msg:      "session cookie with nil",
		},
		{
			sc:       &conf_v1alpha1.SessionCookie{Name: "test"},
			expected: nil,
			msg:      "session cookie not enabled",
		},
	}
	for _, test := range tests {
		result := generateSessionCookie(test.sc)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateSessionCookie() returned %v, but expected %v for the case of: %v", result, test.expected, test.msg)
		}
	}
}

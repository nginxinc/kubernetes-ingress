package configs

import (
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

	expected := "default/test:80"

	result := GenerateEndpointsKey(serviceNamespace, serviceName, port)
	if result != expected {
		t.Errorf("GenerateEndpointsKey() returned %q but expected %q", result, expected)
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
				},
			},
		},
		Endpoints: map[string][]string{
			"default/tea-svc:80": []string{
				"10.0.0.20:80",
			},
			"default/coffee-svc:80": []string{
				"10.0.0.30:80",
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
				Name: "vs_default_cafe_vsr_default_coffee_coffee",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.30:80",
					},
				},
				Keepalive: 16,
			},
		},
		Server: version2.Server{
			ServerName:                            "cafe.example.com",
			ProxyProtocol:                         true,
			RedirectToHTTPSBasedOnXForwarderProto: true,
			ServerTokens:                          "off",
			SetRealIPFrom:                         []string{"0.0.0.0/0"},
			RealIPHeader:                          "X-Real-IP",
			RealIPRecursive:                       true,
			Snippets:                              []string{"# server snippet"},
			Locations: []version2.Location{
				{
					Path:         "/tea",
					ProxyPass:    "http://vs_default_cafe_tea",
					HasKeepalive: true,
				},
				{
					Path:         "/coffee",
					ProxyPass:    "http://vs_default_cafe_vsr_default_coffee_coffee",
					HasKeepalive: true,
				},
			},
		},
	}

	isPlus := false
	tlsPemFileName := ""
	result := generateVirtualServerConfig(&virtualServerEx, tlsPemFileName, &baseCfgParams, isPlus)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateVirtualServerConfig returned \n%v but expected \n%v", result, expected)
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
			"default/tea-svc-v1:80": []string{
				"10.0.0.20:80",
			},
			"default/tea-svc-v2:80": []string{
				"10.0.0.21:80",
			},
			"default/coffee-svc-v1:80": []string{
				"10.0.0.30:80",
			},
			"default/coffee-svc-v2:80": []string{
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
					Path:      "@splits_0_split_0",
					ProxyPass: "http://vs_default_cafe_tea-v1",
				},
				{
					Path:      "@splits_0_split_1",
					ProxyPass: "http://vs_default_cafe_tea-v2",
				},
				{
					Path:      "@splits_1_split_0",
					ProxyPass: "http://vs_default_cafe_vsr_default_coffee_coffee-v1",
				},
				{
					Path:      "@splits_1_split_1",
					ProxyPass: "http://vs_default_cafe_vsr_default_coffee_coffee-v2",
				},
			},
		},
	}

	isPlus := false
	tlsPemFileName := ""
	result := generateVirtualServerConfig(&virtualServerEx, tlsPemFileName, &baseCfgParams, isPlus)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateVirtualServerConfig returned \n%v but expected \n%v", result, expected)
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
			"default/tea-svc-v1:80": []string{
				"10.0.0.20:80",
			},
			"default/tea-svc-v2:80": []string{
				"10.0.0.21:80",
			},
			"default/coffee-svc-v1:80": []string{
				"10.0.0.30:80",
			},
			"default/coffee-svc-v2:80": []string{
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
					Path:      "@rules_0_match_0",
					ProxyPass: "http://vs_default_cafe_tea-v2",
				},
				{
					Path:      "@rules_0_default",
					ProxyPass: "http://vs_default_cafe_tea-v1",
				},
				{
					Path:      "@rules_1_match_0",
					ProxyPass: "http://vs_default_cafe_vsr_default_coffee_coffee-v2",
				},
				{
					Path:      "@rules_1_default",
					ProxyPass: "http://vs_default_cafe_vsr_default_coffee_coffee-v1",
				},
			},
		},
	}

	isPlus := false
	tlsPemFileName := ""
	result := generateVirtualServerConfig(&virtualServerEx, tlsPemFileName, &baseCfgParams, isPlus)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateVirtualServerConfig returned \n%v but expected \n%v", result, expected)
	}
}

func TestGenerateUpstream(t *testing.T) {
	name := "test-upstream"
	upstream := conf_v1alpha1.Upstream{}
	endpoints := []string{
		"192.168.10.10:8080",
	}
	isPlus := false
	cfgParams := ConfigParams{
		LBMethod:    "random",
		MaxFails:    1,
		FailTimeout: "10s",
		Keepalive:   21,
	}

	expected := version2.Upstream{
		Name: "test-upstream",
		Servers: []version2.UpstreamServer{
			{
				Address:     "192.168.10.10:8080",
				MaxFails:    1,
				FailTimeout: "10s",
			},
		},
		LBMethod:  "random",
		Keepalive: 21,
	}

	result := generateUpstream(name, upstream, endpoints, isPlus, &cfgParams)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateUpstream() returned %v but expected %v", result, expected)
	}
}

func TestGenerateUpstreamWithKeepalive(t *testing.T) {
	name := "test-upstream"
	noKeepalive := 0
	keepalive := 32
	endpoints := []string{
		"192.168.10.10:8080",
	}
	isPlus := false

	tests := []struct {
		upstream  conf_v1alpha1.Upstream
		cfgParams *ConfigParams
		expected  version2.Upstream
		msg       string
	}{
		{
			conf_v1alpha1.Upstream{Keepalive: &keepalive},
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
			conf_v1alpha1.Upstream{},
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
			conf_v1alpha1.Upstream{Keepalive: &noKeepalive},
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
		result := generateUpstream(name, test.upstream, endpoints, isPlus, test.cfgParams)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateUpstream() returned %v but expected %v for the case of %v", result, test.expected, test.msg)
		}
	}
}

func TestGenerateUpstreamForZeroEndpoints(t *testing.T) {
	name := "test-upstream"
	upstream := conf_v1alpha1.Upstream{}
	var endpoints []string // nil
	cfgParams := ConfigParams{
		MaxFails:    1,
		FailTimeout: "10s",
	}

	isPlus := false
	expectedForNGINX := version2.Upstream{
		Name: "test-upstream",
		Servers: []version2.UpstreamServer{
			{
				Address:     nginx502Server,
				MaxFails:    1,
				FailTimeout: "10s",
			},
		},
	}

	result := generateUpstream(name, upstream, endpoints, isPlus, &cfgParams)
	if !reflect.DeepEqual(result, expectedForNGINX) {
		t.Errorf("generateUpstream(isPlus=%v) returned %v but expected %v", isPlus, result, expectedForNGINX)
	}

	isPlus = true
	expectedForNGINXPlus := version2.Upstream{
		Name:    "test-upstream",
		Servers: nil,
	}

	result = generateUpstream(name, upstream, endpoints, isPlus, &cfgParams)
	if !reflect.DeepEqual(result, expectedForNGINXPlus) {
		t.Errorf("generateUpstream(isPlus=%v) returned %v but expected %v", isPlus, result, expectedForNGINXPlus)
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
		result := generateProxyPassProtocol(test.upstream)
		if result != test.expected {
			t.Errorf("generateProxyPassProtocol() returned %v but expected %v", result, test.expected)
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
		Path:                 "/",
		Snippets:             []string{"# location snippet"},
		ProxyConnectTimeout:  "30s",
		ProxyReadTimeout:     "31s",
		ProxySendTimeout:     "32s",
		ClientMaxBodySize:    "1m",
		ProxyMaxTempFileSize: "1024m",
		ProxyBuffering:       true,
		ProxyBuffers:         "8 4k",
		ProxyBufferSize:      "4k",
		ProxyPass:            "http://test-upstream",
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

func TestCreateUpstreamServersForPlus(t *testing.T) {
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
				},
			},
		},
		Endpoints: map[string][]string{
			"default/tea-svc:80": []string{
				"10.0.0.20:80",
			},
			"default/test-svc:80": []string{},
			"default/coffee-svc:80": []string{
				"10.0.0.30:80",
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
		},
	}

	expected := map[string][]string{
		"vs_default_cafe_tea": []string{
			"10.0.0.20:80",
		},
		"vs_default_cafe_test": []string{},
		"vs_default_cafe_vsr_default_coffee_coffee": []string{
			"10.0.0.30:80",
		},
	}

	result := createUpstreamServersForPlus(&virtualServerEx)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("createUpstreamServersForPlus returned %v but expected %v", result, expected)
	}
}

func TestCreateUpstreamServersConfig(t *testing.T) {
	baseCfgParams := ConfigParams{
		MaxFails:    5,
		FailTimeout: "30s",
		SlowStart:   "50s",
	}

	expected := nginx.ServerConfig{
		MaxFails:    5,
		FailTimeout: "30s",
		SlowStart:   "50s",
	}

	result := createUpstreamServersConfig(&baseCfgParams)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("createUpstreamServersConfig returned %v but expected %v", result, expected)
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
				Path:      "@splits_1_split_0",
				ProxyPass: "http://vs_default_cafe_coffee-v1",
			},
			{
				Path:      "@splits_1_split_1",
				ProxyPass: "http://vs_default_cafe_coffee-v2",
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
				Path:      "@rules_1_match_0",
				ProxyPass: "http://vs_default_cafe_coffee-v1",
			},
			{
				Path:      "@rules_1_match_1",
				ProxyPass: "http://vs_default_cafe_coffee-v2",
			},
			{
				Path:      "@rules_1_default",
				ProxyPass: "http://vs_default_cafe_tea",
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

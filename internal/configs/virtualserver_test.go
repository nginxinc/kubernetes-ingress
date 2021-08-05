package configs

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func createPointerFromBool(b bool) *bool {
	return &b
}

func TestVirtualServerExString(t *testing.T) {
	tests := []struct {
		input    *VirtualServerEx
		expected string
	}{
		{
			input: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
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
	virtualServer := conf_v1.VirtualServer{
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
	virtualServer := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	virtualServerRoute := conf_v1.VirtualServerRoute{
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
	virtualServer := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-test",
			Namespace: "default",
		},
	}

	expected := "default_cafe_test"

	variableNamer := newVariableNamer(&virtualServer)

	if variableNamer.safeNsName != expected {
		t.Errorf(
			"newVariableNamer() returned variableNamer with safeNsName=%q but expected %q",
			variableNamer.safeNsName,
			expected,
		)
	}
}

func TestVariableNamer(t *testing.T) {
	virtualServer := conf_v1.VirtualServer{
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

	// GetNameForVariableForMatchesRouteMap()
	matchesIndex := 1
	matchIndex := 2
	conditionIndex := 3

	expected = "$vs_default_cafe_matches_1_match_2_cond_3"

	result = variableNamer.GetNameForVariableForMatchesRouteMap(matchesIndex, matchIndex, conditionIndex)
	if result != expected {
		t.Errorf("GetNameForVariableForMatchesRouteMap() returned %q but expected %q", result, expected)
	}

	// GetNameForVariableForMatchesRouteMainMap()
	matchesIndex = 2

	expected = "$vs_default_cafe_matches_2"

	result = variableNamer.GetNameForVariableForMatchesRouteMainMap(matchesIndex)
	if result != expected {
		t.Errorf("GetNameForVariableForMatchesRouteMainMap() returned %q but expected %q", result, expected)
	}
}

func TestGenerateVirtualServerConfig(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe",
				Namespace: "default",
			},
			Spec: conf_v1.VirtualServerSpec{
				Host: "cafe.example.com",
				Upstreams: []conf_v1.Upstream{
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
					{
						Name:    "coffee",
						Service: "coffee-svc",
						Port:    80,
					},
				},
				Routes: []conf_v1.Route{
					{
						Path: "/tea",
						Action: &conf_v1.Action{
							Pass: "tea",
						},
					},
					{
						Path: "/tea-latest",
						Action: &conf_v1.Action{
							Pass: "tea-latest",
						},
					},
					{
						Path:  "/coffee",
						Route: "default/coffee",
					},
					{
						Path:  "/subtea",
						Route: "default/subtea",
					},
					{
						Path: "/coffee-errorpage",
						Action: &conf_v1.Action{
							Pass: "coffee",
						},
						ErrorPages: []conf_v1.ErrorPage{
							{
								Codes: []int{401, 403},
								Redirect: &conf_v1.ErrorPageRedirect{
									ActionRedirect: conf_v1.ActionRedirect{
										URL:  "http://nginx.com",
										Code: 301,
									},
								},
							},
						},
					},
					{
						Path:  "/coffee-errorpage-subroute",
						Route: "default/subcoffee",
						ErrorPages: []conf_v1.ErrorPage{
							{
								Codes: []int{401, 403},
								Redirect: &conf_v1.ErrorPageRedirect{
									ActionRedirect: conf_v1.ActionRedirect{
										URL:  "http://nginx.com",
										Code: 301,
									},
								},
							},
						},
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
		VirtualServerRoutes: []*conf_v1.VirtualServerRoute{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee",
					Namespace: "default",
				},
				Spec: conf_v1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1.Upstream{
						{
							Name:    "coffee",
							Service: "coffee-svc",
							Port:    80,
						},
					},
					Subroutes: []conf_v1.Route{
						{
							Path: "/coffee",
							Action: &conf_v1.Action{
								Pass: "coffee",
							},
						},
					},
				},
			},
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "subtea",
					Namespace: "default",
				},
				Spec: conf_v1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1.Upstream{
						{
							Name:        "subtea",
							Service:     "sub-tea-svc",
							Port:        80,
							Subselector: map[string]string{"version": "v1"},
						},
					},
					Subroutes: []conf_v1.Route{
						{
							Path: "/subtea",
							Action: &conf_v1.Action{
								Pass: "subtea",
							},
						},
					},
				},
			},
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "subcoffee",
					Namespace: "default",
				},
				Spec: conf_v1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1.Upstream{
						{
							Name:    "coffee",
							Service: "coffee-svc",
							Port:    80,
						},
					},
					Subroutes: []conf_v1.Route{
						{
							Path: "/coffee-errorpage-subroute",
							Action: &conf_v1.Action{
								Pass: "coffee",
							},
						},
						{
							Path: "/coffee-errorpage-subroute-defined",
							Action: &conf_v1.Action{
								Pass: "coffee",
							},
							ErrorPages: []conf_v1.ErrorPage{
								{
									Codes: []int{502, 503},
									Return: &conf_v1.ErrorPageReturn{
										ActionReturn: conf_v1.ActionReturn{
											Code: 200,
											Type: "text/plain",
											Body: "All Good",
										},
									},
								},
							},
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
	}

	expected := version2.VirtualServerConfig{
		Upstreams: []version2.Upstream{
			{
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "tea-svc",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_tea",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.20:80",
					},
				},
				Keepalive: 16,
			},
			{
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "tea-svc",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_tea-latest",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.30:80",
					},
				},
				Keepalive: 16,
			},
			{
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "coffee-svc",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_coffee",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.40:80",
					},
				},
				Keepalive: 16,
			},
			{
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "coffee-svc",
					ResourceType:      "virtualserverroute",
					ResourceName:      "coffee",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_vsr_default_coffee_coffee",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.40:80",
					},
				},
				Keepalive: 16,
			},
			{
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "sub-tea-svc",
					ResourceType:      "virtualserverroute",
					ResourceName:      "subtea",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_vsr_default_subtea_subtea",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.50:80",
					},
				},
				Keepalive: 16,
			},
			{
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "coffee-svc",
					ResourceType:      "virtualserverroute",
					ResourceName:      "subcoffee",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_vsr_default_subcoffee_coffee",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.40:80",
					},
				},
				Keepalive: 16,
			},
		},
		HTTPSnippets:  []string{},
		LimitReqZones: []version2.LimitReqZone{},
		Server: version2.Server{
			ServerName:      "cafe.example.com",
			StatusZone:      "cafe.example.com",
			VSNamespace:     "default",
			VSName:          "cafe",
			ProxyProtocol:   true,
			ServerTokens:    "off",
			SetRealIPFrom:   []string{"0.0.0.0/0"},
			RealIPHeader:    "X-Real-IP",
			RealIPRecursive: true,
			Snippets:        []string{"# server snippet"},
			TLSPassthrough:  true,
			Locations: []version2.Location{
				{
					Path:                     "/tea",
					ProxyPass:                "http://vs_default_cafe_tea",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxySSLName:             "tea-svc.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "tea-svc",
				},
				{
					Path:                     "/tea-latest",
					ProxyPass:                "http://vs_default_cafe_tea-latest",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxySSLName:             "tea-svc.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "tea-svc",
				},
				// Order changes here because we generate first all the VS Routes and then all the VSR Subroutes (separated for loops)
				{
					Path:                     "/coffee-errorpage",
					ProxyPass:                "http://vs_default_cafe_coffee",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxyInterceptErrors:     true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "http://nginx.com",
							Codes:        "401 403",
							ResponseCode: 301,
						},
					},
					ProxySSLName:            "coffee-svc.default.svc",
					ProxyPassRequestHeaders: true,
					ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:             "coffee-svc",
				},
				{
					Path:                     "/coffee",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxySSLName:             "coffee-svc.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "coffee-svc",
					IsVSR:                    true,
					VSRName:                  "coffee",
					VSRNamespace:             "default",
				},
				{
					Path:                     "/subtea",
					ProxyPass:                "http://vs_default_cafe_vsr_default_subtea_subtea",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxySSLName:             "sub-tea-svc.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "sub-tea-svc",
					IsVSR:                    true,
					VSRName:                  "subtea",
					VSRNamespace:             "default",
				},

				{
					Path:                     "/coffee-errorpage-subroute",
					ProxyPass:                "http://vs_default_cafe_vsr_default_subcoffee_coffee",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxyInterceptErrors:     true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "http://nginx.com",
							Codes:        "401 403",
							ResponseCode: 301,
						},
					},
					ProxySSLName:            "coffee-svc.default.svc",
					ProxyPassRequestHeaders: true,
					ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:             "coffee-svc",
					IsVSR:                   true,
					VSRName:                 "subcoffee",
					VSRNamespace:            "default",
				},
				{
					Path:                     "/coffee-errorpage-subroute-defined",
					ProxyPass:                "http://vs_default_cafe_vsr_default_subcoffee_coffee",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxyInterceptErrors:     true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@error_page_0_0",
							Codes:        "502 503",
							ResponseCode: 200,
						},
					},
					ProxySSLName:            "coffee-svc.default.svc",
					ProxyPassRequestHeaders: true,
					ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:             "coffee-svc",
					IsVSR:                   true,
					VSRName:                 "subcoffee",
					VSRNamespace:            "default",
				},
			},
			ErrorPageLocations: []version2.ErrorPageLocation{
				{
					Name:        "@error_page_0_0",
					DefaultType: "text/plain",
					Return: &version2.Return{
						Text: "All Good",
					},
				},
			},
		},
	}

	isPlus := false
	isResolverConfigured := false
	isWildcardEnabled := false
	vsc := newVirtualServerConfigurator(
		&baseCfgParams,
		isPlus,
		isResolverConfigured,
		&StaticConfigParams{TLSPassthrough: true},
		isWildcardEnabled,
	)

	result, warnings := vsc.GenerateVirtualServerConfig(&virtualServerEx, nil)
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("GenerateVirtualServerConfig() mismatch (-want +got):\n%s", diff)
	}

	if len(warnings) != 0 {
		t.Errorf("GenerateVirtualServerConfig returned warnings: %v", vsc.warnings)
	}
}

func TestGenerateVirtualServerConfigWithSpiffeCerts(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe",
				Namespace: "default",
			},
			Spec: conf_v1.VirtualServerSpec{
				Host: "cafe.example.com",
				Upstreams: []conf_v1.Upstream{
					{
						Name:    "tea",
						Service: "tea-svc",
						Port:    80,
					},
				},
				Routes: []conf_v1.Route{
					{
						Path: "/tea",
						Action: &conf_v1.Action{
							Pass: "tea",
						},
					},
				},
			},
		},
		Endpoints: map[string][]string{
			"default/tea-svc:80": {
				"10.0.0.20:80",
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
	}

	expected := version2.VirtualServerConfig{
		Upstreams: []version2.Upstream{
			{
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "tea-svc",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_tea",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.20:80",
					},
				},
				Keepalive: 16,
			},
		},
		HTTPSnippets:  []string{},
		LimitReqZones: []version2.LimitReqZone{},
		Server: version2.Server{
			ServerName:      "cafe.example.com",
			StatusZone:      "cafe.example.com",
			VSNamespace:     "default",
			VSName:          "cafe",
			ProxyProtocol:   true,
			ServerTokens:    "off",
			SetRealIPFrom:   []string{"0.0.0.0/0"},
			RealIPHeader:    "X-Real-IP",
			RealIPRecursive: true,
			Snippets:        []string{"# server snippet"},
			TLSPassthrough:  true,
			Locations: []version2.Location{
				{
					Path:                     "/tea",
					ProxyPass:                "https://vs_default_cafe_tea",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxySSLName:             "tea-svc.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "tea-svc",
				},
			},
		},
		SpiffeCerts: true,
	}

	isPlus := false
	isResolverConfigured := false
	staticConfigParams := &StaticConfigParams{TLSPassthrough: true, NginxServiceMesh: true}
	isWildcardEnabled := false
	vsc := newVirtualServerConfigurator(&baseCfgParams, isPlus, isResolverConfigured, staticConfigParams, isWildcardEnabled)

	result, warnings := vsc.GenerateVirtualServerConfig(&virtualServerEx, nil)
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("GenerateVirtualServerConfig() mismatch (-want +got):\n%s", diff)
	}

	if len(warnings) != 0 {
		t.Errorf("GenerateVirtualServerConfig returned warnings: %v", vsc.warnings)
	}
}

func TestGenerateVirtualServerConfigForVirtualServerWithSplits(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe",
				Namespace: "default",
			},
			Spec: conf_v1.VirtualServerSpec{
				Host: "cafe.example.com",
				Upstreams: []conf_v1.Upstream{
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
				Routes: []conf_v1.Route{
					{
						Path: "/tea",
						Splits: []conf_v1.Split{
							{
								Weight: 90,
								Action: &conf_v1.Action{
									Pass: "tea-v1",
								},
							},
							{
								Weight: 10,
								Action: &conf_v1.Action{
									Pass: "tea-v2",
								},
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
		VirtualServerRoutes: []*conf_v1.VirtualServerRoute{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee",
					Namespace: "default",
				},
				Spec: conf_v1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1.Upstream{
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
					Subroutes: []conf_v1.Route{
						{
							Path: "/coffee",
							Splits: []conf_v1.Split{
								{
									Weight: 40,
									Action: &conf_v1.Action{
										Pass: "coffee-v1",
									},
								},
								{
									Weight: 60,
									Action: &conf_v1.Action{
										Pass: "coffee-v2",
									},
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
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "tea-svc-v1",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.20:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_tea-v2",
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "tea-svc-v2",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.21:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_vsr_default_coffee_coffee-v1",
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "coffee-svc-v1",
					ResourceType:      "virtualserverroute",
					ResourceName:      "coffee",
					ResourceNamespace: "default",
				},
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.30:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_vsr_default_coffee_coffee-v2",
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "coffee-svc-v2",
					ResourceType:      "virtualserverroute",
					ResourceName:      "coffee",
					ResourceNamespace: "default",
				},
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
						Value:  "/internal_location_splits_0_split_0",
					},
					{
						Weight: "10%",
						Value:  "/internal_location_splits_0_split_1",
					},
				},
			},
			{
				Source:   "$request_id",
				Variable: "$vs_default_cafe_splits_1",
				Distributions: []version2.Distribution{
					{
						Weight: "40%",
						Value:  "/internal_location_splits_1_split_0",
					},
					{
						Weight: "60%",
						Value:  "/internal_location_splits_1_split_1",
					},
				},
			},
		},
		HTTPSnippets:  []string{},
		LimitReqZones: []version2.LimitReqZone{},
		Server: version2.Server{
			ServerName:  "cafe.example.com",
			StatusZone:  "cafe.example.com",
			VSNamespace: "default",
			VSName:      "cafe",
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
					Path:                     "/internal_location_splits_0_split_0",
					ProxyPass:                "http://vs_default_cafe_tea-v1$request_uri",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					Internal:                 true,
					ProxySSLName:             "tea-svc-v1.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "tea-svc-v1",
				},
				{
					Path:                     "/internal_location_splits_0_split_1",
					ProxyPass:                "http://vs_default_cafe_tea-v2$request_uri",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					Internal:                 true,
					ProxySSLName:             "tea-svc-v2.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "tea-svc-v2",
				},
				{
					Path:                     "/internal_location_splits_1_split_0",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee-v1$request_uri",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					Internal:                 true,
					ProxySSLName:             "coffee-svc-v1.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "coffee-svc-v1",
					IsVSR:                    true,
					VSRName:                  "coffee",
					VSRNamespace:             "default",
				},
				{
					Path:                     "/internal_location_splits_1_split_1",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee-v2$request_uri",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					Internal:                 true,
					ProxySSLName:             "coffee-svc-v2.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "coffee-svc-v2",
					IsVSR:                    true,
					VSRName:                  "coffee",
					VSRNamespace:             "default",
				},
			},
		},
	}

	isPlus := false
	isResolverConfigured := false
	isWildcardEnabled := false
	vsc := newVirtualServerConfigurator(&baseCfgParams, isPlus, isResolverConfigured, &StaticConfigParams{}, isWildcardEnabled)

	result, warnings := vsc.GenerateVirtualServerConfig(&virtualServerEx, nil)
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("GenerateVirtualServerConfig() mismatch (-want +got):\n%s", diff)
	}

	if len(warnings) != 0 {
		t.Errorf("GenerateVirtualServerConfig returned warnings: %v", vsc.warnings)
	}
}

func TestGenerateVirtualServerConfigForVirtualServerWithMatches(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe",
				Namespace: "default",
			},
			Spec: conf_v1.VirtualServerSpec{
				Host: "cafe.example.com",
				Upstreams: []conf_v1.Upstream{
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
				Routes: []conf_v1.Route{
					{
						Path: "/tea",
						Matches: []conf_v1.Match{
							{
								Conditions: []conf_v1.Condition{
									{
										Header: "x-version",
										Value:  "v2",
									},
								},
								Action: &conf_v1.Action{
									Pass: "tea-v2",
								},
							},
						},
						Action: &conf_v1.Action{
							Pass: "tea-v1",
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
		VirtualServerRoutes: []*conf_v1.VirtualServerRoute{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee",
					Namespace: "default",
				},
				Spec: conf_v1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1.Upstream{
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
					Subroutes: []conf_v1.Route{
						{
							Path: "/coffee",
							Matches: []conf_v1.Match{
								{
									Conditions: []conf_v1.Condition{
										{
											Argument: "version",
											Value:    "v2",
										},
									},
									Action: &conf_v1.Action{
										Pass: "coffee-v2",
									},
								},
							},
							Action: &conf_v1.Action{
								Pass: "coffee-v1",
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
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "tea-svc-v1",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_tea-v1",
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.20:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_tea-v2",
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "tea-svc-v2",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.21:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_vsr_default_coffee_coffee-v1",
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "coffee-svc-v1",
					ResourceType:      "virtualserverroute",
					ResourceName:      "coffee",
					ResourceNamespace: "default",
				},
				Servers: []version2.UpstreamServer{
					{
						Address: "10.0.0.30:80",
					},
				},
			},
			{
				Name: "vs_default_cafe_vsr_default_coffee_coffee-v2",
				UpstreamLabels: version2.UpstreamLabels{
					Service:           "coffee-svc-v2",
					ResourceType:      "virtualserverroute",
					ResourceName:      "coffee",
					ResourceNamespace: "default",
				},
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
				Variable: "$vs_default_cafe_matches_0_match_0_cond_0",
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
				Source:   "$vs_default_cafe_matches_0_match_0_cond_0",
				Variable: "$vs_default_cafe_matches_0",
				Parameters: []version2.Parameter{
					{
						Value:  "~^1",
						Result: "/internal_location_matches_0_match_0",
					},
					{
						Value:  "default",
						Result: "/internal_location_matches_0_default",
					},
				},
			},
			{
				Source:   "$arg_version",
				Variable: "$vs_default_cafe_matches_1_match_0_cond_0",
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
				Source:   "$vs_default_cafe_matches_1_match_0_cond_0",
				Variable: "$vs_default_cafe_matches_1",
				Parameters: []version2.Parameter{
					{
						Value:  "~^1",
						Result: "/internal_location_matches_1_match_0",
					},
					{
						Value:  "default",
						Result: "/internal_location_matches_1_default",
					},
				},
			},
		},
		HTTPSnippets:  []string{},
		LimitReqZones: []version2.LimitReqZone{},
		Server: version2.Server{
			ServerName:  "cafe.example.com",
			StatusZone:  "cafe.example.com",
			VSNamespace: "default",
			VSName:      "cafe",
			InternalRedirectLocations: []version2.InternalRedirectLocation{
				{
					Path:        "/tea",
					Destination: "$vs_default_cafe_matches_0",
				},
				{
					Path:        "/coffee",
					Destination: "$vs_default_cafe_matches_1",
				},
			},
			Locations: []version2.Location{
				{
					Path:                     "/internal_location_matches_0_match_0",
					ProxyPass:                "http://vs_default_cafe_tea-v2$request_uri",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					Internal:                 true,
					ProxySSLName:             "tea-svc-v2.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "tea-svc-v2",
				},
				{
					Path:                     "/internal_location_matches_0_default",
					ProxyPass:                "http://vs_default_cafe_tea-v1$request_uri",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					Internal:                 true,
					ProxySSLName:             "tea-svc-v1.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "tea-svc-v1",
				},
				{
					Path:                     "/internal_location_matches_1_match_0",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee-v2$request_uri",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					Internal:                 true,
					ProxySSLName:             "coffee-svc-v2.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "coffee-svc-v2",
					IsVSR:                    true,
					VSRName:                  "coffee",
					VSRNamespace:             "default",
				},
				{
					Path:                     "/internal_location_matches_1_default",
					ProxyPass:                "http://vs_default_cafe_vsr_default_coffee_coffee-v1$request_uri",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					Internal:                 true,
					ProxySSLName:             "coffee-svc-v1.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "coffee-svc-v1",
					IsVSR:                    true,
					VSRName:                  "coffee",
					VSRNamespace:             "default",
				},
			},
		},
	}

	isPlus := false
	isResolverConfigured := false
	isWildcardEnabled := false
	vsc := newVirtualServerConfigurator(&baseCfgParams, isPlus, isResolverConfigured, &StaticConfigParams{}, isWildcardEnabled)

	result, warnings := vsc.GenerateVirtualServerConfig(&virtualServerEx, nil)
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("GenerateVirtualServerConfig() mismatch (-want +got):\n%s", diff)
	}

	if len(warnings) != 0 {
		t.Errorf("GenerateVirtualServerConfig returned warnings: %v", vsc.warnings)
	}
}

func TestGenerateVirtualServerConfigForVirtualServerWithReturns(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "returns",
				Namespace: "default",
			},
			Spec: conf_v1.VirtualServerSpec{
				Host: "example.com",
				Routes: []conf_v1.Route{
					{
						Path: "/return",
						Action: &conf_v1.Action{
							Return: &conf_v1.ActionReturn{
								Body: "hello 0",
							},
						},
					},
					{
						Path: "/splits-with-return",
						Splits: []conf_v1.Split{
							{
								Weight: 90,
								Action: &conf_v1.Action{
									Return: &conf_v1.ActionReturn{
										Body: "hello 1",
									},
								},
							},
							{
								Weight: 10,
								Action: &conf_v1.Action{
									Return: &conf_v1.ActionReturn{
										Body: "hello 2",
									},
								},
							},
						},
					},
					{
						Path: "/matches-with-return",
						Matches: []conf_v1.Match{
							{
								Conditions: []conf_v1.Condition{
									{
										Header: "x-version",
										Value:  "v2",
									},
								},
								Action: &conf_v1.Action{
									Return: &conf_v1.ActionReturn{
										Body: "hello 3",
									},
								},
							},
						},
						Action: &conf_v1.Action{
							Return: &conf_v1.ActionReturn{
								Body: "hello 4",
							},
						},
					},
					{
						Path:  "/more",
						Route: "default/more-returns",
					},
				},
			},
		},
		VirtualServerRoutes: []*conf_v1.VirtualServerRoute{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "more-returns",
					Namespace: "default",
				},
				Spec: conf_v1.VirtualServerRouteSpec{
					Host: "example.com",
					Subroutes: []conf_v1.Route{
						{
							Path: "/more/return",
							Action: &conf_v1.Action{
								Return: &conf_v1.ActionReturn{
									Body: "hello 5",
								},
							},
						},
						{
							Path: "/more/splits-with-return",
							Splits: []conf_v1.Split{
								{
									Weight: 90,
									Action: &conf_v1.Action{
										Return: &conf_v1.ActionReturn{
											Body: "hello 6",
										},
									},
								},
								{
									Weight: 10,
									Action: &conf_v1.Action{
										Return: &conf_v1.ActionReturn{
											Body: "hello 7",
										},
									},
								},
							},
						},
						{
							Path: "/more/matches-with-return",
							Matches: []conf_v1.Match{
								{
									Conditions: []conf_v1.Condition{
										{
											Header: "x-version",
											Value:  "v2",
										},
									},
									Action: &conf_v1.Action{
										Return: &conf_v1.ActionReturn{
											Body: "hello 8",
										},
									},
								},
							},
							Action: &conf_v1.Action{
								Return: &conf_v1.ActionReturn{
									Body: "hello 9",
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
		Maps: []version2.Map{
			{
				Source:   "$http_x_version",
				Variable: "$vs_default_returns_matches_0_match_0_cond_0",
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
				Source:   "$vs_default_returns_matches_0_match_0_cond_0",
				Variable: "$vs_default_returns_matches_0",
				Parameters: []version2.Parameter{
					{
						Value:  "~^1",
						Result: "/internal_location_matches_0_match_0",
					},
					{
						Value:  "default",
						Result: "/internal_location_matches_0_default",
					},
				},
			},
			{
				Source:   "$http_x_version",
				Variable: "$vs_default_returns_matches_1_match_0_cond_0",
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
				Source:   "$vs_default_returns_matches_1_match_0_cond_0",
				Variable: "$vs_default_returns_matches_1",
				Parameters: []version2.Parameter{
					{
						Value:  "~^1",
						Result: "/internal_location_matches_1_match_0",
					},
					{
						Value:  "default",
						Result: "/internal_location_matches_1_default",
					},
				},
			},
		},
		SplitClients: []version2.SplitClient{
			{
				Source:   "$request_id",
				Variable: "$vs_default_returns_splits_0",
				Distributions: []version2.Distribution{
					{
						Weight: "90%",
						Value:  "/internal_location_splits_0_split_0",
					},
					{
						Weight: "10%",
						Value:  "/internal_location_splits_0_split_1",
					},
				},
			},
			{
				Source:   "$request_id",
				Variable: "$vs_default_returns_splits_1",
				Distributions: []version2.Distribution{
					{
						Weight: "90%",
						Value:  "/internal_location_splits_1_split_0",
					},
					{
						Weight: "10%",
						Value:  "/internal_location_splits_1_split_1",
					},
				},
			},
		},
		HTTPSnippets:  []string{},
		LimitReqZones: []version2.LimitReqZone{},
		Server: version2.Server{
			ServerName:  "example.com",
			StatusZone:  "example.com",
			VSNamespace: "default",
			VSName:      "returns",
			InternalRedirectLocations: []version2.InternalRedirectLocation{
				{
					Path:        "/splits-with-return",
					Destination: "$vs_default_returns_splits_0",
				},
				{
					Path:        "/matches-with-return",
					Destination: "$vs_default_returns_matches_0",
				},
				{
					Path:        "/more/splits-with-return",
					Destination: "$vs_default_returns_splits_1",
				},
				{
					Path:        "/more/matches-with-return",
					Destination: "$vs_default_returns_matches_1",
				},
			},
			ReturnLocations: []version2.ReturnLocation{
				{
					Name:        "@return_0",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 0",
					},
				},
				{
					Name:        "@return_1",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 1",
					},
				},
				{
					Name:        "@return_2",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 2",
					},
				},
				{
					Name:        "@return_3",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 3",
					},
				},
				{
					Name:        "@return_4",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 4",
					},
				},
				{
					Name:        "@return_5",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 5",
					},
				},
				{
					Name:        "@return_6",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 6",
					},
				},
				{
					Name:        "@return_7",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 7",
					},
				},
				{
					Name:        "@return_8",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 8",
					},
				},
				{
					Name:        "@return_9",
					DefaultType: "text/plain",
					Return: version2.Return{
						Code: 0,
						Text: "hello 9",
					},
				},
			},
			Locations: []version2.Location{
				{
					Path:                 "/return",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_0",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
				{
					Path:                 "/internal_location_splits_0_split_0",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_1",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
				{
					Path:                 "/internal_location_splits_0_split_1",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_2",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
				{
					Path:                 "/internal_location_matches_0_match_0",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_3",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
				{
					Path:                 "/internal_location_matches_0_default",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_4",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
				{
					Path:                 "/more/return",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_5",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
				{
					Path:                 "/internal_location_splits_1_split_0",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_6",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
				{
					Path:                 "/internal_location_splits_1_split_1",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_7",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
				{
					Path:                 "/internal_location_matches_1_match_0",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_8",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
				{
					Path:                 "/internal_location_matches_1_default",
					ProxyInterceptErrors: true,
					ErrorPages: []version2.ErrorPage{
						{
							Name:         "@return_9",
							Codes:        "418",
							ResponseCode: 200,
						},
					},
					InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
				},
			},
		},
	}

	isPlus := false
	isResolverConfigured := false
	isWildcardEnabled := false
	vsc := newVirtualServerConfigurator(&baseCfgParams, isPlus, isResolverConfigured, &StaticConfigParams{}, isWildcardEnabled)

	result, warnings := vsc.GenerateVirtualServerConfig(&virtualServerEx, nil)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("GenerateVirtualServerConfig returned \n%+v but expected \n%+v", result, expected)
	}

	if len(warnings) != 0 {
		t.Errorf("GenerateVirtualServerConfig returned warnings: %v", vsc.warnings)
	}
}

func TestGeneratePolicies(t *testing.T) {
	ownerDetails := policyOwnerDetails{
		owner:          nil, // nil is OK for the unit test
		ownerNamespace: "default",
		vsNamespace:    "default",
		vsName:         "test",
	}
	ingressMTLSCertPath := "/etc/nginx/secrets/default-ingress-mtls-secret"
	policyOpts := policyOptions{
		tls: true,
		secretRefs: map[string]*secrets.SecretReference{
			"default/ingress-mtls-secret": {
				Secret: &api_v1.Secret{
					Type: secrets.SecretTypeCA,
				},
				Path: ingressMTLSCertPath,
			},
			"default/egress-mtls-secret": {
				Secret: &api_v1.Secret{
					Type: api_v1.SecretTypeTLS,
				},
				Path: "/etc/nginx/secrets/default-egress-mtls-secret",
			},
			"default/egress-trusted-ca-secret": {
				Secret: &api_v1.Secret{
					Type: secrets.SecretTypeCA,
				},
				Path: "/etc/nginx/secrets/default-egress-trusted-ca-secret",
			},
			"default/jwt-secret": {
				Secret: &api_v1.Secret{
					Type: secrets.SecretTypeJWK,
				},
				Path: "/etc/nginx/secrets/default-jwt-secret",
			},
			"default/oidc-secret": {
				Secret: &api_v1.Secret{
					Type: secrets.SecretTypeOIDC,
					Data: map[string][]byte{
						"client-secret": []byte("super_secret_123"),
					},
				},
			},
		},
		apResources: map[string]string{
			"default/logconf":         "/etc/nginx/waf/nac-logconfs/default-logconf",
			"default/dataguard-alarm": "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
		},
	}

	tests := []struct {
		policyRefs []conf_v1.PolicyReference
		policies   map[string]*conf_v1.Policy
		policyOpts policyOptions
		context    string
		expected   policiesCfg
		msg        string
	}{
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "allow-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/allow-policy": {
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expected: policiesCfg{
				Allow: []string{"127.0.0.1"},
			},
			msg: "explicit reference",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name: "allow-policy",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/allow-policy": {
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expected: policiesCfg{
				Allow: []string{"127.0.0.1"},
			},
			msg: "implicit reference",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name: "allow-policy-1",
				},
				{
					Name: "allow-policy-2",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/allow-policy-1": {
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
				"default/allow-policy-2": {
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.2"},
						},
					},
				},
			},
			expected: policiesCfg{
				Allow: []string{"127.0.0.1", "127.0.0.2"},
			},
			msg: "merging",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "rateLimit-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/rateLimit-policy": {
					Spec: conf_v1.PolicySpec{
						RateLimit: &conf_v1.RateLimit{
							Key:      "test",
							ZoneSize: "10M",
							Rate:     "10r/s",
							LogLevel: "notice",
						},
					},
				},
			},
			expected: policiesCfg{
				LimitReqZones: []version2.LimitReqZone{
					{
						Key:      "test",
						ZoneSize: "10M",
						Rate:     "10r/s",
						ZoneName: "pol_rl_default_rateLimit-policy_default_test",
					},
				},
				LimitReqOptions: version2.LimitReqOptions{
					LogLevel:   "notice",
					RejectCode: 503,
				},
				LimitReqs: []version2.LimitReq{
					{
						ZoneName: "pol_rl_default_rateLimit-policy_default_test",
					},
				},
			},
			msg: "rate limit reference",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "rateLimit-policy",
					Namespace: "default",
				},
				{
					Name:      "rateLimit-policy2",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/rateLimit-policy": {
					Spec: conf_v1.PolicySpec{
						RateLimit: &conf_v1.RateLimit{
							Key:      "test",
							ZoneSize: "10M",
							Rate:     "10r/s",
						},
					},
				},
				"default/rateLimit-policy2": {
					Spec: conf_v1.PolicySpec{
						RateLimit: &conf_v1.RateLimit{
							Key:      "test2",
							ZoneSize: "20M",
							Rate:     "20r/s",
						},
					},
				},
			},
			expected: policiesCfg{
				LimitReqZones: []version2.LimitReqZone{
					{
						Key:      "test",
						ZoneSize: "10M",
						Rate:     "10r/s",
						ZoneName: "pol_rl_default_rateLimit-policy_default_test",
					},
					{
						Key:      "test2",
						ZoneSize: "20M",
						Rate:     "20r/s",
						ZoneName: "pol_rl_default_rateLimit-policy2_default_test",
					},
				},
				LimitReqOptions: version2.LimitReqOptions{
					LogLevel:   "error",
					RejectCode: 503,
				},
				LimitReqs: []version2.LimitReq{
					{
						ZoneName: "pol_rl_default_rateLimit-policy_default_test",
					},
					{
						ZoneName: "pol_rl_default_rateLimit-policy2_default_test",
					},
				},
			},
			msg: "multi rate limit reference",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "jwt-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/jwt-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Realm:  "My Test API",
							Secret: "jwt-secret",
						},
					},
				},
			},
			expected: policiesCfg{
				JWTAuth: &version2.JWTAuth{
					Secret: "/etc/nginx/secrets/default-jwt-secret",
					Realm:  "My Test API",
				},
			},
			msg: "jwt reference",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "ingress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/ingress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "ingress-mtls-secret",
							VerifyClient:     "off",
						},
					},
				},
			},
			context: "spec",
			expected: policiesCfg{
				IngressMTLS: &version2.IngressMTLS{
					ClientCert:   ingressMTLSCertPath,
					VerifyClient: "off",
					VerifyDepth:  1,
				},
			},
			msg: "ingressMTLS reference",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "egress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/egress-mtls-policy": {
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret:         "egress-mtls-secret",
							ServerName:        true,
							SessionReuse:      createPointerFromBool(false),
							TrustedCertSecret: "egress-trusted-ca-secret",
						},
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				EgressMTLS: &version2.EgressMTLS{
					Certificate:    "/etc/nginx/secrets/default-egress-mtls-secret",
					CertificateKey: "/etc/nginx/secrets/default-egress-mtls-secret",
					Ciphers:        "DEFAULT",
					Protocols:      "TLSv1 TLSv1.1 TLSv1.2",
					ServerName:     true,
					SessionReuse:   false,
					VerifyDepth:    1,
					VerifyServer:   false,
					TrustedCert:    "/etc/nginx/secrets/default-egress-trusted-ca-secret",
					SSLName:        "$proxy_host",
				},
			},
			msg: "egressMTLS reference",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "oidc-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/oidc-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							AuthEndpoint:  "http://example.com/auth",
							TokenEndpoint: "http://example.com/token",
							JWKSURI:       "http://example.com/jwks",
							ClientID:      "client-id",
							ClientSecret:  "oidc-secret",
							Scope:         "scope",
							RedirectURI:   "/redirect",
						},
					},
				},
			},
			expected: policiesCfg{
				OIDC: true,
			},
			msg: "oidc reference",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "waf-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/waf-policy": {
					Spec: conf_v1.PolicySpec{
						WAF: &conf_v1.WAF{
							Enable:   true,
							ApPolicy: "default/dataguard-alarm",
							SecurityLog: &conf_v1.SecurityLog{
								Enable:    true,
								ApLogConf: "default/logconf",
								LogDest:   "syslog:server=127.0.0.1:514",
							},
						},
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				WAF: &version2.WAF{
					Enable:              "on",
					ApPolicy:            "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
					ApSecurityLogEnable: true,
					ApLogConf:           "/etc/nginx/waf/nac-logconfs/default-logconf syslog:server=127.0.0.1:514",
				},
			},
			msg: "WAF reference",
		},
	}

	vsc := newVirtualServerConfigurator(&ConfigParams{}, false, false, &StaticConfigParams{}, false)

	for _, test := range tests {
		result := vsc.generatePolicies(ownerDetails, test.policyRefs, test.policies, test.context, policyOpts)
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("generatePolicies() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
		if len(vsc.warnings) > 0 {
			t.Errorf("generatePolicies() returned unexpected warnings %v for the case of %s", vsc.warnings, test.msg)
		}
	}
}

func TestGeneratePoliciesFails(t *testing.T) {
	ownerDetails := policyOwnerDetails{
		owner:          nil, // nil is OK for the unit test
		ownerNamespace: "default",
		vsNamespace:    "default",
		vsName:         "test",
	}

	dryRunOverride := true
	rejectCodeOverride := 505

	tests := []struct {
		policyRefs        []conf_v1.PolicyReference
		policies          map[string]*conf_v1.Policy
		policyOpts        policyOptions
		trustedCAFileName string
		context           string
		oidcPolCfg        *oidcPolicyCfg
		expected          policiesCfg
		expectedWarnings  Warnings
		expectedOidc      *oidcPolicyCfg
		msg               string
	}{
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "allow-policy",
					Namespace: "default",
				},
			},
			policies:   map[string]*conf_v1.Policy{},
			policyOpts: policyOptions{},
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					"Policy default/allow-policy is missing or invalid",
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "missing policy",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name: "allow-policy",
				},
				{
					Name: "deny-policy",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/allow-policy": {
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
				"default/deny-policy": {
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Deny: []string{"127.0.0.2"},
						},
					},
				},
			},
			policyOpts: policyOptions{},
			expected: policiesCfg{
				Allow: []string{"127.0.0.1"},
				Deny:  []string{"127.0.0.2"},
			},
			expectedWarnings: Warnings{
				nil: {
					"AccessControl policy (or policies) with deny rules is overridden by policy (or policies) with allow rules",
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "conflicting policies",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "rateLimit-policy",
					Namespace: "default",
				},
				{
					Name:      "rateLimit-policy2",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/rateLimit-policy": {
					Spec: conf_v1.PolicySpec{
						RateLimit: &conf_v1.RateLimit{
							Key:      "test",
							ZoneSize: "10M",
							Rate:     "10r/s",
						},
					},
				},
				"default/rateLimit-policy2": {
					Spec: conf_v1.PolicySpec{
						RateLimit: &conf_v1.RateLimit{
							Key:        "test2",
							ZoneSize:   "20M",
							Rate:       "20r/s",
							DryRun:     &dryRunOverride,
							LogLevel:   "info",
							RejectCode: &rejectCodeOverride,
						},
					},
				},
			},
			policyOpts: policyOptions{},
			expected: policiesCfg{
				LimitReqZones: []version2.LimitReqZone{
					{
						Key:      "test",
						ZoneSize: "10M",
						Rate:     "10r/s",
						ZoneName: "pol_rl_default_rateLimit-policy_default_test",
					},
					{
						Key:      "test2",
						ZoneSize: "20M",
						Rate:     "20r/s",
						ZoneName: "pol_rl_default_rateLimit-policy2_default_test",
					},
				},
				LimitReqOptions: version2.LimitReqOptions{
					LogLevel:   "error",
					RejectCode: 503,
				},
				LimitReqs: []version2.LimitReq{
					{
						ZoneName: "pol_rl_default_rateLimit-policy_default_test",
					},
					{
						ZoneName: "pol_rl_default_rateLimit-policy2_default_test",
					},
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`RateLimit policy default/rateLimit-policy2 with limit request option dryRun='true' is overridden to dryRun='false' by the first policy reference in this context`,
					`RateLimit policy default/rateLimit-policy2 with limit request option logLevel='info' is overridden to logLevel='error' by the first policy reference in this context`,
					`RateLimit policy default/rateLimit-policy2 with limit request option rejectCode='505' is overridden to rejectCode='503' by the first policy reference in this context`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "rate limit policy limit request option override",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "jwt-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/jwt-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Realm:  "test",
							Secret: "jwt-secret",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/jwt-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeJWK,
						},
						Error: errors.New("secret is invalid"),
					},
				},
			},
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`JWT policy default/jwt-policy references an invalid secret default/jwt-secret: secret is invalid`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "jwt reference missing secret",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "jwt-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/jwt-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Realm:  "test",
							Secret: "jwt-secret",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/jwt-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeCA,
						},
					},
				},
			},
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`JWT policy default/jwt-policy references a secret default/jwt-secret of a wrong type 'nginx.org/ca', must be 'nginx.org/jwk'`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "jwt references wrong secret type",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "jwt-policy",
					Namespace: "default",
				},
				{
					Name:      "jwt-policy2",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/jwt-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Realm:  "test",
							Secret: "jwt-secret",
						},
					},
				},
				"default/jwt-policy2": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy2",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Realm:  "test",
							Secret: "jwt-secret2",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/jwt-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeJWK,
						},
						Path: "/etc/nginx/secrets/default-jwt-secret",
					},
					"default/jwt-secret2": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeJWK,
						},
						Path: "/etc/nginx/secrets/default-jwt-secret2",
					},
				},
			},
			expected: policiesCfg{
				JWTAuth: &version2.JWTAuth{
					Secret: "/etc/nginx/secrets/default-jwt-secret",
					Realm:  "test",
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`Multiple jwt policies in the same context is not valid. JWT policy default/jwt-policy2 will be ignored`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "multi jwt reference",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "ingress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/ingress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "ingress-mtls-secret",
						},
					},
				},
			},
			policyOpts: policyOptions{
				tls: true,
				secretRefs: map[string]*secrets.SecretReference{
					"default/ingress-mtls-secret": {
						Error: errors.New("secret is invalid"),
					},
				},
			},
			context: "spec",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`IngressMTLS policy "default/ingress-mtls-policy" references an invalid secret default/ingress-mtls-secret: secret is invalid`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "ingress mtls reference an invalid secret",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "ingress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/ingress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "ingress-mtls-secret",
						},
					},
				},
			},
			policyOpts: policyOptions{
				tls: true,
				secretRefs: map[string]*secrets.SecretReference{
					"default/ingress-mtls-secret": {
						Secret: &api_v1.Secret{
							Type: api_v1.SecretTypeTLS,
						},
					},
				},
			},
			context: "spec",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`IngressMTLS policy default/ingress-mtls-policy references a secret default/ingress-mtls-secret of a wrong type 'kubernetes.io/tls', must be 'nginx.org/ca'`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "ingress mtls references wrong secret type",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "ingress-mtls-policy",
					Namespace: "default",
				},
				{
					Name:      "ingress-mtls-policy2",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/ingress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "ingress-mtls-secret",
						},
					},
				},
				"default/ingress-mtls-policy2": {
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "ingress-mtls-secret2",
						},
					},
				},
			},
			policyOpts: policyOptions{
				tls: true,
				secretRefs: map[string]*secrets.SecretReference{
					"default/ingress-mtls-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeCA,
						},
						Path: "/etc/nginx/secrets/default-ingress-mtls-secret",
					},
				},
			},
			context: "spec",
			expected: policiesCfg{
				IngressMTLS: &version2.IngressMTLS{
					ClientCert:   "/etc/nginx/secrets/default-ingress-mtls-secret",
					VerifyClient: "on",
					VerifyDepth:  1,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`Multiple ingressMTLS policies are not allowed. IngressMTLS policy default/ingress-mtls-policy2 will be ignored`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "multi ingress mtls",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "ingress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/ingress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "ingress-mtls-secret",
						},
					},
				},
			},
			policyOpts: policyOptions{
				tls: true,
				secretRefs: map[string]*secrets.SecretReference{
					"default/ingress-mtls-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeCA,
						},
						Path: "/etc/nginx/secrets/default-ingress-mtls-secret",
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`IngressMTLS policy default/ingress-mtls-policy is not allowed in the route context`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "ingress mtls in the wrong context",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "ingress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/ingress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "ingress-mtls-secret",
						},
					},
				},
			},
			policyOpts: policyOptions{
				tls: false,
				secretRefs: map[string]*secrets.SecretReference{
					"default/ingress-mtls-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeCA,
						},
						Path: "/etc/nginx/secrets/default-ingress-mtls-secret",
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`TLS must be enabled in VirtualServer for IngressMTLS policy default/ingress-mtls-policy`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "ingress mtls missing TLS config",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "egress-mtls-policy",
					Namespace: "default",
				},
				{
					Name:      "egress-mtls-policy2",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/egress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret: "egress-mtls-secret",
						},
					},
				},
				"default/egress-mtls-policy2": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy2",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret: "egress-mtls-secret2",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/egress-mtls-secret": {
						Secret: &api_v1.Secret{
							Type: api_v1.SecretTypeTLS,
						},
						Path: "/etc/nginx/secrets/default-egress-mtls-secret",
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				EgressMTLS: &version2.EgressMTLS{
					Certificate:    "/etc/nginx/secrets/default-egress-mtls-secret",
					CertificateKey: "/etc/nginx/secrets/default-egress-mtls-secret",
					VerifyServer:   false,
					VerifyDepth:    1,
					Ciphers:        "DEFAULT",
					Protocols:      "TLSv1 TLSv1.1 TLSv1.2",
					SessionReuse:   true,
					SSLName:        "$proxy_host",
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`Multiple egressMTLS policies in the same context is not valid. EgressMTLS policy default/egress-mtls-policy2 will be ignored`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "multi egress mtls",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "egress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/egress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TrustedCertSecret: "egress-trusted-secret",
							SSLName:           "foo.com",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/egress-trusted-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeCA,
						},
						Error: errors.New("secret is invalid"),
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`EgressMTLS policy default/egress-mtls-policy references an invalid secret default/egress-trusted-secret: secret is invalid`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "egress mtls referencing an invalid CA secret",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "egress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/egress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret: "egress-mtls-secret",
							SSLName:   "foo.com",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/egress-mtls-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeCA,
						},
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`EgressMTLS policy default/egress-mtls-policy references a secret default/egress-mtls-secret of a wrong type 'nginx.org/ca', must be 'kubernetes.io/tls'`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "egress mtls referencing wrong secret type",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "egress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/egress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TrustedCertSecret: "egress-trusted-secret",
							SSLName:           "foo.com",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/egress-trusted-secret": {
						Secret: &api_v1.Secret{
							Type: api_v1.SecretTypeTLS,
						},
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`EgressMTLS policy default/egress-mtls-policy references a secret default/egress-trusted-secret of a wrong type 'kubernetes.io/tls', must be 'nginx.org/ca'`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "egress trusted secret referencing wrong secret type",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "egress-mtls-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/egress-mtls-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret: "egress-mtls-secret",
							SSLName:   "foo.com",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/egress-mtls-secret": {
						Secret: &api_v1.Secret{
							Type: api_v1.SecretTypeTLS,
						},
						Error: errors.New("secret is invalid"),
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`EgressMTLS policy default/egress-mtls-policy references an invalid secret default/egress-mtls-secret: secret is invalid`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "egress mtls referencing missing tls secret",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "oidc-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/oidc-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientSecret: "oidc-secret",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/oidc-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeOIDC,
						},
						Error: errors.New("secret is invalid"),
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`OIDC policy default/oidc-policy references an invalid secret default/oidc-secret: secret is invalid`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "oidc referencing missing oidc secret",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "oidc-policy",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/oidc-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientSecret:  "oidc-secret",
							AuthEndpoint:  "http://foo.com/bar",
							TokenEndpoint: "http://foo.com/bar",
							JWKSURI:       "http://foo.com/bar",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/oidc-secret": {
						Secret: &api_v1.Secret{
							Type: api_v1.SecretTypeTLS,
						},
					},
				},
			},
			context: "spec",
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`OIDC policy default/oidc-policy references a secret default/oidc-secret of a wrong type 'kubernetes.io/tls', must be 'nginx.org/oidc'`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "oidc secret referencing wrong secret type",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "oidc-policy-2",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/oidc-policy-1": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy-1",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientID:      "foo",
							ClientSecret:  "oidc-secret",
							AuthEndpoint:  "https://foo.com/auth",
							TokenEndpoint: "https://foo.com/token",
							JWKSURI:       "https://foo.com/certs",
						},
					},
				},
				"default/oidc-policy-2": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy-2",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientID:      "foo",
							ClientSecret:  "oidc-secret",
							AuthEndpoint:  "https://bar.com/auth",
							TokenEndpoint: "https://bar.com/token",
							JWKSURI:       "https://bar.com/certs",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/oidc-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeOIDC,
							Data: map[string][]byte{
								"client-secret": []byte("super_secret_123"),
							},
						},
					},
				},
			},
			context: "route",
			oidcPolCfg: &oidcPolicyCfg{
				oidc: &version2.OIDC{
					AuthEndpoint:  "https://foo.com/auth",
					TokenEndpoint: "https://foo.com/token",
					JwksURI:       "https://foo.com/certs",
					ClientID:      "foo",
					ClientSecret:  "super_secret_123",
					RedirectURI:   "/_codexch",
					Scope:         "openid",
				},
				key: "default/oidc-policy-1",
			},
			expected: policiesCfg{
				ErrorReturn: &version2.Return{
					Code: 500,
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`Only one oidc policy is allowed in a VirtualServer and its VirtualServerRoutes. Can't use default/oidc-policy-2. Use default/oidc-policy-1`,
				},
			},
			expectedOidc: &oidcPolicyCfg{
				oidc: &version2.OIDC{
					AuthEndpoint:  "https://foo.com/auth",
					TokenEndpoint: "https://foo.com/token",
					JwksURI:       "https://foo.com/certs",
					ClientID:      "foo",
					ClientSecret:  "super_secret_123",
					RedirectURI:   "/_codexch",
					Scope:         "openid",
				},
				key: "default/oidc-policy-1",
			},
			msg: "multiple oidc policies",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "oidc-policy",
					Namespace: "default",
				},
				{
					Name:      "oidc-policy2",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/oidc-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientSecret:  "oidc-secret",
							AuthEndpoint:  "https://foo.com/auth",
							TokenEndpoint: "https://foo.com/token",
							JWKSURI:       "https://foo.com/certs",
							ClientID:      "foo",
						},
					},
				},
				"default/oidc-policy2": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy2",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientSecret:  "oidc-secret",
							AuthEndpoint:  "https://bar.com/auth",
							TokenEndpoint: "https://bar.com/token",
							JWKSURI:       "https://bar.com/certs",
							ClientID:      "bar",
						},
					},
				},
			},
			policyOpts: policyOptions{
				secretRefs: map[string]*secrets.SecretReference{
					"default/oidc-secret": {
						Secret: &api_v1.Secret{
							Type: secrets.SecretTypeOIDC,
							Data: map[string][]byte{
								"client-secret": []byte("super_secret_123"),
							},
						},
					},
				},
			},
			context: "route",
			expected: policiesCfg{
				OIDC: true,
			},
			expectedWarnings: Warnings{
				nil: {
					`Multiple oidc policies in the same context is not valid. OIDC policy default/oidc-policy2 will be ignored`,
				},
			},
			expectedOidc: &oidcPolicyCfg{
				&version2.OIDC{
					AuthEndpoint:  "https://foo.com/auth",
					TokenEndpoint: "https://foo.com/token",
					JwksURI:       "https://foo.com/certs",
					ClientID:      "foo",
					ClientSecret:  "super_secret_123",
					RedirectURI:   "/_codexch",
					Scope:         "openid",
				},
				"default/oidc-policy",
			},
			msg: "multi oidc",
		},
		{
			policyRefs: []conf_v1.PolicyReference{
				{
					Name:      "waf-policy",
					Namespace: "default",
				},
				{
					Name:      "waf-policy2",
					Namespace: "default",
				},
			},
			policies: map[string]*conf_v1.Policy{
				"default/waf-policy": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "waf-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						WAF: &conf_v1.WAF{
							Enable:   true,
							ApPolicy: "default/dataguard-alarm",
						},
					},
				},
				"default/waf-policy2": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "waf-policy2",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						WAF: &conf_v1.WAF{
							Enable:   true,
							ApPolicy: "default/dataguard-alarm",
						},
					},
				},
			},
			policyOpts: policyOptions{
				apResources: map[string]string{
					"default/logconf":         "/etc/nginx/waf/nac-logconfs/default-logconf",
					"default/dataguard-alarm": "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
				},
			},
			context: "route",
			expected: policiesCfg{
				WAF: &version2.WAF{
					Enable:   "on",
					ApPolicy: "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
				},
			},
			expectedWarnings: Warnings{
				nil: {
					`Multiple WAF policies in the same context is not valid. WAF policy default/waf-policy2 will be ignored`,
				},
			},
			expectedOidc: &oidcPolicyCfg{},
			msg:          "multi waf",
		},
	}

	for _, test := range tests {
		vsc := newVirtualServerConfigurator(&ConfigParams{}, false, false, &StaticConfigParams{}, false)

		if test.oidcPolCfg != nil {
			vsc.oidcPolCfg = test.oidcPolCfg
		}

		result := vsc.generatePolicies(ownerDetails, test.policyRefs, test.policies, test.context, test.policyOpts)
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("generatePolicies() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
		if !reflect.DeepEqual(vsc.warnings, test.expectedWarnings) {
			t.Errorf(
				"generatePolicies() returned warnings of \n%v but expected \n%v for the case of %s",
				vsc.warnings,
				test.expectedWarnings,
				test.msg,
			)
		}
		if diff := cmp.Diff(test.expectedOidc.oidc, vsc.oidcPolCfg.oidc); diff != "" {
			t.Errorf("generatePolicies() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
		if diff := cmp.Diff(test.expectedOidc.key, vsc.oidcPolCfg.key); diff != "" {
			t.Errorf("generatePolicies() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		rlz      []version2.LimitReqZone
		expected []version2.LimitReqZone
	}{
		{
			rlz: []version2.LimitReqZone{
				{ZoneName: "test"},
				{ZoneName: "test"},
				{ZoneName: "test2"},
				{ZoneName: "test3"},
			},
			expected: []version2.LimitReqZone{
				{ZoneName: "test"},
				{ZoneName: "test2"},
				{ZoneName: "test3"},
			},
		},
		{
			rlz: []version2.LimitReqZone{
				{ZoneName: "test"},
				{ZoneName: "test"},
				{ZoneName: "test2"},
				{ZoneName: "test3"},
				{ZoneName: "test3"},
			},
			expected: []version2.LimitReqZone{
				{ZoneName: "test"},
				{ZoneName: "test2"},
				{ZoneName: "test3"},
			},
		},
	}
	for _, test := range tests {
		result := removeDuplicateLimitReqZones(test.rlz)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("removeDuplicateLimitReqZones() returned \n%v, but expected \n%v", result, test.expected)
		}
	}
}

func TestAddPoliciesCfgToLocations(t *testing.T) {
	cfg := policiesCfg{
		Allow: []string{"127.0.0.1"},
		Deny:  []string{"127.0.0.2"},
		ErrorReturn: &version2.Return{
			Code: 400,
		},
	}

	locations := []version2.Location{
		{
			Path: "/",
		},
	}

	expectedLocations := []version2.Location{
		{
			Path:  "/",
			Allow: []string{"127.0.0.1"},
			Deny:  []string{"127.0.0.2"},
			PoliciesErrorReturn: &version2.Return{
				Code: 400,
			},
		},
	}

	addPoliciesCfgToLocations(cfg, locations)
	if !reflect.DeepEqual(locations, expectedLocations) {
		t.Errorf("addPoliciesCfgToLocations() returned \n%+v but expected \n%+v", locations, expectedLocations)
	}
}

func TestGenerateUpstream(t *testing.T) {
	name := "test-upstream"
	upstream := conf_v1.Upstream{Service: name, Port: 80}
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
		UpstreamLabels: version2.UpstreamLabels{
			Service: "test-upstream",
		},
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

	vsc := newVirtualServerConfigurator(&cfgParams, false, false, &StaticConfigParams{}, false)
	result := vsc.generateUpstream(nil, name, upstream, false, endpoints)
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
		upstream  conf_v1.Upstream
		cfgParams *ConfigParams
		expected  version2.Upstream
		msg       string
	}{
		{
			conf_v1.Upstream{Keepalive: &keepalive, Service: name, Port: 80},
			&ConfigParams{Keepalive: 21},
			version2.Upstream{
				Name: "test-upstream",
				UpstreamLabels: version2.UpstreamLabels{
					Service: "test-upstream",
				},
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
			conf_v1.Upstream{Service: name, Port: 80},
			&ConfigParams{Keepalive: 21},
			version2.Upstream{
				Name: "test-upstream",
				UpstreamLabels: version2.UpstreamLabels{
					Service: "test-upstream",
				},
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
			conf_v1.Upstream{Keepalive: &noKeepalive, Service: name, Port: 80},
			&ConfigParams{Keepalive: 21},
			version2.Upstream{
				Name: "test-upstream",
				UpstreamLabels: version2.UpstreamLabels{
					Service: "test-upstream",
				},
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
		vsc := newVirtualServerConfigurator(test.cfgParams, false, false, &StaticConfigParams{}, false)
		result := vsc.generateUpstream(nil, name, test.upstream, false, endpoints)
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
	upstream := conf_v1.Upstream{Service: name}
	cfgParams := ConfigParams{}

	expected := version2.Upstream{
		Name: name,
		UpstreamLabels: version2.UpstreamLabels{
			Service: "test-upstream",
		},
		Servers: []version2.UpstreamServer{
			{
				Address: "example.com",
			},
		},
		Resolve: true,
	}

	vsc := newVirtualServerConfigurator(&cfgParams, true, true, &StaticConfigParams{}, false)
	result := vsc.generateUpstream(nil, name, upstream, true, endpoints)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateUpstream() returned %v but expected %v", result, expected)
	}

	if len(vsc.warnings) != 0 {
		t.Errorf("generateUpstream() returned warnings for %v", upstream)
	}
}

func TestGenerateUpstreamWithNTLM(t *testing.T) {
	name := "test-upstream"
	upstream := conf_v1.Upstream{Service: name, Port: 80, NTLM: true}
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
		UpstreamLabels: version2.UpstreamLabels{
			Service: "test-upstream",
		},
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
		NTLM:             true,
	}

	vsc := newVirtualServerConfigurator(&cfgParams, true, false, &StaticConfigParams{}, false)
	result := vsc.generateUpstream(nil, name, upstream, false, endpoints)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateUpstream() returned %v but expected %v", result, expected)
	}

	if len(vsc.warnings) != 0 {
		t.Errorf("generateUpstream returned warnings for %v", upstream)
	}
}

func TestGenerateProxyPass(t *testing.T) {
	tests := []struct {
		tlsEnabled   bool
		upstreamName string
		internal     bool
		expected     string
	}{
		{
			tlsEnabled:   false,
			upstreamName: "test-upstream",
			internal:     false,
			expected:     "http://test-upstream",
		},
		{
			tlsEnabled:   true,
			upstreamName: "test-upstream",
			internal:     false,
			expected:     "https://test-upstream",
		},
		{
			tlsEnabled:   false,
			upstreamName: "test-upstream",
			internal:     true,
			expected:     "http://test-upstream$request_uri",
		},
		{
			tlsEnabled:   true,
			upstreamName: "test-upstream",
			internal:     true,
			expected:     "https://test-upstream$request_uri",
		},
	}

	for _, test := range tests {
		result := generateProxyPass(test.tlsEnabled, test.upstreamName, test.internal, nil)
		if result != test.expected {
			t.Errorf("generateProxyPass(%v, %v, %v) returned %v but expected %v", test.tlsEnabled, test.upstreamName, test.internal, result, test.expected)
		}
	}
}

func TestGenerateProxyPassProtocol(t *testing.T) {
	tests := []struct {
		upstream conf_v1.Upstream
		expected string
	}{
		{
			upstream: conf_v1.Upstream{},
			expected: "http",
		},
		{
			upstream: conf_v1.Upstream{
				TLS: conf_v1.UpstreamTLS{
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

func TestGenerateSnippets(t *testing.T) {
	tests := []struct {
		enableSnippets bool
		s              string
		defaultS       []string
		expected       []string
	}{
		{
			true,
			"test",
			[]string{},
			[]string{"test"},
		},
		{
			true,
			"",
			[]string{"default"},
			[]string{"default"},
		},
		{
			true,
			"test\none\ntwo",
			[]string{},
			[]string{"test", "one", "two"},
		},
		{
			false,
			"test",
			nil,
			nil,
		},
	}
	for _, test := range tests {
		result := generateSnippets(test.enableSnippets, test.s, test.defaultS)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateSnippets() return %v, but expected %v", result, test.expected)
		}
	}
}

func TestGenerateBuffer(t *testing.T) {
	tests := []struct {
		inputS   *conf_v1.UpstreamBuffers
		expected string
	}{
		{
			inputS:   nil,
			expected: "8 4k",
		},
		{
			inputS:   &conf_v1.UpstreamBuffers{Number: 8, Size: "16K"},
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

func TestGenerateLocationForProxying(t *testing.T) {
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
	vsLocSnippets := []string{"# vs location snippet"}

	expected := version2.Location{
		Path:                     "/",
		Snippets:                 vsLocSnippets,
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
		ProxyPassRequestHeaders:  true,
		ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
		ServiceName:              "",
		IsVSR:                    false,
		VSRName:                  "",
		VSRNamespace:             "",
	}

	result := generateLocationForProxying(path, upstreamName, conf_v1.Upstream{}, &cfgParams, nil, false, 0, "", nil, "", vsLocSnippets, false, "", "")
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateLocationForProxying() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateReturnBlock(t *testing.T) {
	tests := []struct {
		text        string
		code        int
		defaultCode int
		expected    *version2.Return
	}{
		{
			text:        "Hello World!",
			code:        0, // Not set
			defaultCode: 200,
			expected: &version2.Return{
				Code: 200,
				Text: "Hello World!",
			},
		},
		{
			text:        "Hello World!",
			code:        400,
			defaultCode: 200,
			expected: &version2.Return{
				Code: 400,
				Text: "Hello World!",
			},
		},
	}

	for _, test := range tests {
		result := generateReturnBlock(test.text, test.code, test.defaultCode)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateReturnBlock() returned %v but expected %v", result, test.expected)
		}
	}
}

func TestGenerateLocationForReturn(t *testing.T) {
	tests := []struct {
		actionReturn           *conf_v1.ActionReturn
		expectedLocation       version2.Location
		expectedReturnLocation *version2.ReturnLocation
		msg                    string
	}{
		{
			actionReturn: &conf_v1.ActionReturn{
				Body: "hello",
			},

			expectedLocation: version2.Location{
				Path:     "/",
				Snippets: []string{"# location snippet"},
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@return_1",
						Codes:        "418",
						ResponseCode: 200,
					},
				},
				ProxyInterceptErrors: true,
				InternalProxyPass:    "http://unix:/var/lib/nginx/nginx-418-server.sock",
			},
			expectedReturnLocation: &version2.ReturnLocation{
				Name:        "@return_1",
				DefaultType: "text/plain",
				Return: version2.Return{
					Code: 0,
					Text: "hello",
				},
			},
			msg: "return without code and type",
		},
		{
			actionReturn: &conf_v1.ActionReturn{
				Code: 400,
				Type: "text/html",
				Body: "hello",
			},

			expectedLocation: version2.Location{
				Path:     "/",
				Snippets: []string{"# location snippet"},
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@return_1",
						Codes:        "418",
						ResponseCode: 400,
					},
				},
				ProxyInterceptErrors: true,
				InternalProxyPass:    "http://unix:/var/lib/nginx/nginx-418-server.sock",
			},
			expectedReturnLocation: &version2.ReturnLocation{
				Name:        "@return_1",
				DefaultType: "text/html",
				Return: version2.Return{
					Code: 0,
					Text: "hello",
				},
			},
			msg: "return with all fields defined",
		},
	}
	path := "/"
	snippets := []string{"# location snippet"}
	returnLocationIndex := 1

	for _, test := range tests {
		location, returnLocation := generateLocationForReturn(path, snippets, test.actionReturn, returnLocationIndex)
		if !reflect.DeepEqual(location, test.expectedLocation) {
			t.Errorf("generateLocationForReturn() returned  \n%+v but expected \n%+v for the case of %s",
				location, test.expectedLocation, test.msg)
		}
		if !reflect.DeepEqual(returnLocation, test.expectedReturnLocation) {
			t.Errorf("generateLocationForReturn() returned  \n%+v but expected \n%+v for the case of %s",
				returnLocation, test.expectedReturnLocation, test.msg)
		}
	}
}

func TestGenerateLocationForRedirect(t *testing.T) {
	tests := []struct {
		redirect *conf_v1.ActionRedirect
		expected version2.Location
		msg      string
	}{
		{
			redirect: &conf_v1.ActionRedirect{
				URL: "http://nginx.org",
			},

			expected: version2.Location{
				Path:     "/",
				Snippets: []string{"# location snippet"},
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "http://nginx.org",
						Codes:        "418",
						ResponseCode: 301,
					},
				},
				ProxyInterceptErrors: true,
				InternalProxyPass:    "http://unix:/var/lib/nginx/nginx-418-server.sock",
			},
			msg: "redirect without code",
		},
		{
			redirect: &conf_v1.ActionRedirect{
				Code: 302,
				URL:  "http://nginx.org",
			},

			expected: version2.Location{
				Path:     "/",
				Snippets: []string{"# location snippet"},
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "http://nginx.org",
						Codes:        "418",
						ResponseCode: 302,
					},
				},
				ProxyInterceptErrors: true,
				InternalProxyPass:    "http://unix:/var/lib/nginx/nginx-418-server.sock",
			},
			msg: "redirect with all fields defined",
		},
	}

	for _, test := range tests {
		result := generateLocationForRedirect("/", []string{"# location snippet"}, test.redirect)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateLocationForReturn() returned \n%+v but expected \n%+v for the case of %s",
				result, test.expected, test.msg)
		}
	}
}

func TestGenerateSSLConfig(t *testing.T) {
	tests := []struct {
		inputTLS         *conf_v1.TLS
		inputSecretRefs  map[string]*secrets.SecretReference
		inputCfgParams   *ConfigParams
		wildcard         bool
		expectedSSL      *version2.SSL
		expectedWarnings Warnings
		msg              string
	}{
		{
			inputTLS:         nil,
			inputSecretRefs:  map[string]*secrets.SecretReference{},
			inputCfgParams:   &ConfigParams{},
			wildcard:         false,
			expectedSSL:      nil,
			expectedWarnings: Warnings{},
			msg:              "no TLS field",
		},
		{
			inputTLS: &conf_v1.TLS{
				Secret: "",
			},
			inputSecretRefs:  map[string]*secrets.SecretReference{},
			inputCfgParams:   &ConfigParams{},
			wildcard:         false,
			expectedSSL:      nil,
			expectedWarnings: Warnings{},
			msg:              "TLS field with empty secret and wildcard cert disabled",
		},
		{
			inputTLS: &conf_v1.TLS{
				Secret: "",
			},
			inputSecretRefs: map[string]*secrets.SecretReference{},
			inputCfgParams:  &ConfigParams{},
			wildcard:        true,
			expectedSSL: &version2.SSL{
				HTTP2:           false,
				Certificate:     pemFileNameForWildcardTLSSecret,
				CertificateKey:  pemFileNameForWildcardTLSSecret,
				RejectHandshake: false,
			},
			expectedWarnings: Warnings{},
			msg:              "TLS field with empty secret and wildcard cert enabled",
		},
		{
			inputTLS: &conf_v1.TLS{
				Secret: "secret",
			},
			inputCfgParams: &ConfigParams{},
			wildcard:       false,
			inputSecretRefs: map[string]*secrets.SecretReference{
				"default/secret": {
					Error: errors.New("secret doesn't exist"),
				},
			},
			expectedSSL: &version2.SSL{
				HTTP2:           false,
				RejectHandshake: true,
			},
			expectedWarnings: Warnings{
				nil: []string{"TLS secret secret is invalid: secret doesn't exist"},
			},
			msg: "secret doesn't exist in the cluster with HTTPS",
		},
		{
			inputTLS: &conf_v1.TLS{
				Secret: "secret",
			},
			inputCfgParams: &ConfigParams{},
			wildcard:       false,
			inputSecretRefs: map[string]*secrets.SecretReference{
				"default/secret": {
					Secret: &api_v1.Secret{
						Type: secrets.SecretTypeCA,
					},
				},
			},
			expectedSSL: &version2.SSL{
				HTTP2:           false,
				RejectHandshake: true,
			},
			expectedWarnings: Warnings{
				nil: []string{"TLS secret secret is of a wrong type 'nginx.org/ca', must be 'kubernetes.io/tls'"},
			},
			msg: "wrong secret type",
		},
		{
			inputTLS: &conf_v1.TLS{
				Secret: "secret",
			},
			inputSecretRefs: map[string]*secrets.SecretReference{
				"default/secret": {
					Secret: &api_v1.Secret{
						Type: api_v1.SecretTypeTLS,
					},
					Path: "secret.pem",
				},
			},
			inputCfgParams: &ConfigParams{},
			wildcard:       false,
			expectedSSL: &version2.SSL{
				HTTP2:           false,
				Certificate:     "secret.pem",
				CertificateKey:  "secret.pem",
				RejectHandshake: false,
			},
			expectedWarnings: Warnings{},
			msg:              "normal case with HTTPS",
		},
	}

	namespace := "default"

	for _, test := range tests {
		vsc := newVirtualServerConfigurator(&ConfigParams{}, false, false, &StaticConfigParams{}, test.wildcard)

		// it is ok to use nil as the owner
		result := vsc.generateSSLConfig(nil, test.inputTLS, namespace, test.inputSecretRefs, test.inputCfgParams)
		if !reflect.DeepEqual(result, test.expectedSSL) {
			t.Errorf("generateSSLConfig() returned %v but expected %v for the case of %s", result, test.expectedSSL, test.msg)
		}
		if !reflect.DeepEqual(vsc.warnings, test.expectedWarnings) {
			t.Errorf("generateSSLConfig() returned warnings of \n%v but expected \n%v for the case of %s", vsc.warnings, test.expectedWarnings, test.msg)
		}
	}
}

func TestGenerateRedirectConfig(t *testing.T) {
	tests := []struct {
		inputTLS *conf_v1.TLS
		expected *version2.TLSRedirect
		msg      string
	}{
		{
			inputTLS: nil,
			expected: nil,
			msg:      "no TLS field",
		},
		{
			inputTLS: &conf_v1.TLS{
				Secret:   "secret",
				Redirect: nil,
			},
			expected: nil,
			msg:      "no redirect field",
		},
		{
			inputTLS: &conf_v1.TLS{
				Secret:   "secret",
				Redirect: &conf_v1.TLSRedirect{Enable: false},
			},
			expected: nil,
			msg:      "redirect disabled",
		},
		{
			inputTLS: &conf_v1.TLS{
				Secret: "secret",
				Redirect: &conf_v1.TLSRedirect{
					Enable: true,
				},
			},
			expected: &version2.TLSRedirect{
				Code:    301,
				BasedOn: "$scheme",
			},
			msg: "normal case with defaults",
		},
		{
			inputTLS: &conf_v1.TLS{
				Secret: "secret",
				Redirect: &conf_v1.TLSRedirect{
					Enable:  true,
					BasedOn: "x-forwarded-proto",
				},
			},
			expected: &version2.TLSRedirect{
				Code:    301,
				BasedOn: "$http_x_forwarded_proto",
			},
			msg: "normal case with BasedOn set",
		},
	}

	for _, test := range tests {
		result := generateTLSRedirectConfig(test.inputTLS)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateTLSRedirectConfig() returned %v but expected %v for the case of %s", result, test.expected, test.msg)
		}
	}
}

func TestGenerateTLSRedirectBasedOn(t *testing.T) {
	tests := []struct {
		basedOn  string
		expected string
	}{
		{
			basedOn:  "scheme",
			expected: "$scheme",
		},
		{
			basedOn:  "x-forwarded-proto",
			expected: "$http_x_forwarded_proto",
		},
		{
			basedOn:  "",
			expected: "$scheme",
		},
	}
	for _, test := range tests {
		result := generateTLSRedirectBasedOn(test.basedOn)
		if result != test.expected {
			t.Errorf("generateTLSRedirectBasedOn(%v) returned %v but expected %v", test.basedOn, result, test.expected)
		}
	}
}

func TestCreateUpstreamsForPlus(t *testing.T) {
	virtualServerEx := VirtualServerEx{
		VirtualServer: &conf_v1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe",
				Namespace: "default",
			},
			Spec: conf_v1.VirtualServerSpec{
				Host: "cafe.example.com",
				Upstreams: []conf_v1.Upstream{
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
				Routes: []conf_v1.Route{
					{
						Path: "/tea",
						Action: &conf_v1.Action{
							Pass: "tea",
						},
					},
					{
						Path:  "/coffee",
						Route: "default/coffee",
					},
					{
						Path: "/external",
						Action: &conf_v1.Action{
							Pass: "external",
						},
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
		VirtualServerRoutes: []*conf_v1.VirtualServerRoute{
			{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee",
					Namespace: "default",
				},
				Spec: conf_v1.VirtualServerRouteSpec{
					Host: "cafe.example.com",
					Upstreams: []conf_v1.Upstream{
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
					Subroutes: []conf_v1.Route{
						{
							Path: "/coffee",
							Action: &conf_v1.Action{
								Pass: "coffee",
							},
						},
						{
							Path: "/coffee/sub",
							Action: &conf_v1.Action{
								Pass: "subselector-test",
							},
						},
					},
				},
			},
		},
	}

	expected := []version2.Upstream{
		{
			Name: "vs_default_cafe_tea",
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "tea-svc",
				ResourceType:      "virtualserver",
				ResourceNamespace: "default",
				ResourceName:      "cafe",
			},
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.20:80",
				},
			},
		},
		{
			Name: "vs_default_cafe_test",
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "test-svc",
				ResourceType:      "virtualserver",
				ResourceNamespace: "default",
				ResourceName:      "cafe",
			},
			Servers: nil,
		},
		{
			Name: "vs_default_cafe_subselector-test",
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "test-svc",
				ResourceType:      "virtualserver",
				ResourceNamespace: "default",
				ResourceName:      "cafe",
			},
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.30:80",
				},
			},
		},
		{
			Name: "vs_default_cafe_vsr_default_coffee_coffee",
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "coffee-svc",
				ResourceType:      "virtualserverroute",
				ResourceNamespace: "default",
				ResourceName:      "coffee",
			},
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.40:80",
				},
			},
		},
		{
			Name: "vs_default_cafe_vsr_default_coffee_subselector-test",
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "test-svc",
				ResourceType:      "virtualserverroute",
				ResourceNamespace: "default",
				ResourceName:      "coffee",
			},
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.50:80",
				},
			},
		},
	}

	result := createUpstreamsForPlus(&virtualServerEx, &ConfigParams{}, &StaticConfigParams{})
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

func TestGenerateSplits(t *testing.T) {
	originalPath := "/path"
	splits := []conf_v1.Split{
		{
			Weight: 90,
			Action: &conf_v1.Action{
				Proxy: &conf_v1.ActionProxy{
					Upstream:    "coffee-v1",
					RewritePath: "/rewrite",
				},
			},
		},
		{
			Weight: 9,
			Action: &conf_v1.Action{
				Pass: "coffee-v2",
			},
		},
		{
			Weight: 1,
			Action: &conf_v1.Action{
				Return: &conf_v1.ActionReturn{
					Body: "hello",
				},
			},
		},
	}

	virtualServer := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	upstreamNamer := newUpstreamNamerForVirtualServer(&virtualServer)
	variableNamer := newVariableNamer(&virtualServer)
	scIndex := 1
	cfgParams := ConfigParams{}
	crUpstreams := map[string]conf_v1.Upstream{
		"vs_default_cafe_coffee-v1": {
			Service: "coffee-v1",
		},
		"vs_default_cafe_coffee-v2": {
			Service: "coffee-v2",
		},
	}
	locSnippet := "# location snippet"
	enableSnippets := true
	errorPages := []conf_v1.ErrorPage{
		{
			Codes: []int{400, 500},
			Return: &conf_v1.ErrorPageReturn{
				ActionReturn: conf_v1.ActionReturn{
					Code: 200,
					Type: "application/json",
					Body: `{\"message\": \"ok\"}`,
				},
				Headers: []conf_v1.Header{
					{
						Name:  "Set-Cookie",
						Value: "cookie1=value",
					},
				},
			},
			Redirect: nil,
		},
		{
			Codes:  []int{500, 502},
			Return: nil,
			Redirect: &conf_v1.ErrorPageRedirect{
				ActionRedirect: conf_v1.ActionRedirect{
					URL:  "http://nginx.com",
					Code: 301,
				},
			},
		},
	}

	expectedSplitClient := version2.SplitClient{
		Source:   "$request_id",
		Variable: "$vs_default_cafe_splits_1",
		Distributions: []version2.Distribution{
			{
				Weight: "90%",
				Value:  "/internal_location_splits_1_split_0",
			},
			{
				Weight: "9%",
				Value:  "/internal_location_splits_1_split_1",
			},
			{
				Weight: "1%",
				Value:  "/internal_location_splits_1_split_2",
			},
		},
	}
	expectedLocations := []version2.Location{
		{
			Path:      "/internal_location_splits_1_split_0",
			ProxyPass: "http://vs_default_cafe_coffee-v1",
			Rewrites: []string{
				"^ $request_uri",
				fmt.Sprintf(`"^%v(.*)$" "/rewrite$1" break`, originalPath),
			},
			ProxyNextUpstream:        "error timeout",
			ProxyNextUpstreamTimeout: "0s",
			ProxyNextUpstreamTries:   0,
			ProxyInterceptErrors:     true,
			Internal:                 true,
			ErrorPages: []version2.ErrorPage{
				{
					Name:         "@error_page_0_0",
					Codes:        "400 500",
					ResponseCode: 200,
				},
				{
					Name:         "http://nginx.com",
					Codes:        "500 502",
					ResponseCode: 301,
				},
			},
			ProxySSLName:            "coffee-v1.default.svc",
			ProxyPassRequestHeaders: true,
			ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
			Snippets:                []string{locSnippet},
			ServiceName:             "coffee-v1",
			IsVSR:                   true,
			VSRName:                 "coffee",
			VSRNamespace:            "default",
		},
		{
			Path:                     "/internal_location_splits_1_split_1",
			ProxyPass:                "http://vs_default_cafe_coffee-v2$request_uri",
			ProxyNextUpstream:        "error timeout",
			ProxyNextUpstreamTimeout: "0s",
			ProxyNextUpstreamTries:   0,
			ProxyInterceptErrors:     true,
			Internal:                 true,
			ErrorPages: []version2.ErrorPage{
				{
					Name:         "@error_page_0_0",
					Codes:        "400 500",
					ResponseCode: 200,
				},
				{
					Name:         "http://nginx.com",
					Codes:        "500 502",
					ResponseCode: 301,
				},
			},
			ProxySSLName:            "coffee-v2.default.svc",
			ProxyPassRequestHeaders: true,
			ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
			Snippets:                []string{locSnippet},
			ServiceName:             "coffee-v2",
			IsVSR:                   true,
			VSRName:                 "coffee",
			VSRNamespace:            "default",
		},
		{
			Path:                 "/internal_location_splits_1_split_2",
			ProxyInterceptErrors: true,
			ErrorPages: []version2.ErrorPage{
				{
					Name:         "@return_1",
					Codes:        "418",
					ResponseCode: 200,
				},
			},
			InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
		},
	}
	expectedReturnLocations := []version2.ReturnLocation{
		{
			Name:        "@return_1",
			DefaultType: "text/plain",
			Return: version2.Return{
				Code: 0,
				Text: "hello",
			},
		},
	}
	returnLocationIndex := 1

	resultSplitClient, resultLocations, resultReturnLocations := generateSplits(
		splits,
		upstreamNamer,
		crUpstreams,
		variableNamer,
		scIndex,
		&cfgParams,
		errorPages,
		0,
		originalPath,
		locSnippet,
		enableSnippets,
		returnLocationIndex,
		true,
		"coffee",
		"default",
	)
	if !reflect.DeepEqual(resultSplitClient, expectedSplitClient) {
		t.Errorf("generateSplits() returned \n%+v but expected \n%+v", resultSplitClient, expectedSplitClient)
	}
	if !reflect.DeepEqual(resultLocations, expectedLocations) {
		t.Errorf("generateSplits() returned \n%+v but expected \n%+v", resultLocations, expectedLocations)
	}
	if !reflect.DeepEqual(resultReturnLocations, expectedReturnLocations) {
		t.Errorf("generateSplits() returned \n%+v but expected \n%+v", resultReturnLocations, expectedReturnLocations)
	}
}

func TestGenerateDefaultSplitsConfig(t *testing.T) {
	route := conf_v1.Route{
		Path: "/",
		Splits: []conf_v1.Split{
			{
				Weight: 90,
				Action: &conf_v1.Action{
					Pass: "coffee-v1",
				},
			},
			{
				Weight: 10,
				Action: &conf_v1.Action{
					Pass: "coffee-v2",
				},
			},
		},
	}
	virtualServer := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	upstreamNamer := newUpstreamNamerForVirtualServer(&virtualServer)
	variableNamer := newVariableNamer(&virtualServer)
	index := 1

	expected := routingCfg{
		SplitClients: []version2.SplitClient{
			{
				Source:   "$request_id",
				Variable: "$vs_default_cafe_splits_1",
				Distributions: []version2.Distribution{
					{
						Weight: "90%",
						Value:  "/internal_location_splits_1_split_0",
					},
					{
						Weight: "10%",
						Value:  "/internal_location_splits_1_split_1",
					},
				},
			},
		},
		Locations: []version2.Location{
			{
				Path:                     "/internal_location_splits_1_split_0",
				ProxyPass:                "http://vs_default_cafe_coffee-v1$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				Internal:                 true,
				ProxySSLName:             "coffee-v1.default.svc",
				ProxyPassRequestHeaders:  true,
				ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:              "coffee-v1",
				IsVSR:                    true,
				VSRName:                  "coffee",
				VSRNamespace:             "default",
			},
			{
				Path:                     "/internal_location_splits_1_split_1",
				ProxyPass:                "http://vs_default_cafe_coffee-v2$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				Internal:                 true,
				ProxySSLName:             "coffee-v2.default.svc",
				ProxyPassRequestHeaders:  true,
				ProxySetHeaders:          []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:              "coffee-v2",
				IsVSR:                    true,
				VSRName:                  "coffee",
				VSRNamespace:             "default",
			},
		},
		InternalRedirectLocation: version2.InternalRedirectLocation{
			Path:        "/",
			Destination: "$vs_default_cafe_splits_1",
		},
	}

	cfgParams := ConfigParams{}
	locSnippet := ""
	enableSnippets := false
	crUpstreams := map[string]conf_v1.Upstream{
		"vs_default_cafe_coffee-v1": {
			Service: "coffee-v1",
		},
		"vs_default_cafe_coffee-v2": {
			Service: "coffee-v2",
		},
	}

	result := generateDefaultSplitsConfig(route, upstreamNamer, crUpstreams, variableNamer, index, &cfgParams,
		route.ErrorPages, 0, "", locSnippet, enableSnippets, 0, true, "coffee", "default")
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateDefaultSplitsConfig() returned \n%+v but expected \n%+v", result, expected)
	}
}

func TestGenerateMatchesConfig(t *testing.T) {
	route := conf_v1.Route{
		Path: "/",
		Matches: []conf_v1.Match{
			{
				Conditions: []conf_v1.Condition{
					{
						Header: "x-version",
						Value:  "v1",
					},
					{
						Cookie: "user",
						Value:  "john",
					},
					{
						Argument: "answer",
						Value:    "yes",
					},
					{
						Variable: "$request_method",
						Value:    "GET",
					},
				},
				Action: &conf_v1.Action{
					Pass: "coffee-v1",
				},
			},
			{
				Conditions: []conf_v1.Condition{
					{
						Header: "x-version",
						Value:  "v2",
					},
					{
						Cookie: "user",
						Value:  "paul",
					},
					{
						Argument: "answer",
						Value:    "no",
					},
					{
						Variable: "$request_method",
						Value:    "POST",
					},
				},
				Splits: []conf_v1.Split{
					{
						Weight: 90,
						Action: &conf_v1.Action{
							Pass: "coffee-v1",
						},
					},
					{
						Weight: 10,
						Action: &conf_v1.Action{
							Pass: "coffee-v2",
						},
					},
				},
			},
		},
		Action: &conf_v1.Action{
			Pass: "tea",
		},
	}
	virtualServer := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	errorPages := []conf_v1.ErrorPage{
		{
			Codes: []int{400, 500},
			Return: &conf_v1.ErrorPageReturn{
				ActionReturn: conf_v1.ActionReturn{
					Code: 200,
					Type: "application/json",
					Body: `{\"message\": \"ok\"}`,
				},
				Headers: []conf_v1.Header{
					{
						Name:  "Set-Cookie",
						Value: "cookie1=value",
					},
				},
			},
			Redirect: nil,
		},
		{
			Codes:  []int{500, 502},
			Return: nil,
			Redirect: &conf_v1.ErrorPageRedirect{
				ActionRedirect: conf_v1.ActionRedirect{
					URL:  "http://nginx.com",
					Code: 301,
				},
			},
		},
	}
	upstreamNamer := newUpstreamNamerForVirtualServer(&virtualServer)
	variableNamer := newVariableNamer(&virtualServer)
	index := 1
	scIndex := 2

	expected := routingCfg{
		Maps: []version2.Map{
			{
				Source:   "$http_x_version",
				Variable: "$vs_default_cafe_matches_1_match_0_cond_0",
				Parameters: []version2.Parameter{
					{
						Value:  `"v1"`,
						Result: "$vs_default_cafe_matches_1_match_0_cond_1",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$cookie_user",
				Variable: "$vs_default_cafe_matches_1_match_0_cond_1",
				Parameters: []version2.Parameter{
					{
						Value:  `"john"`,
						Result: "$vs_default_cafe_matches_1_match_0_cond_2",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$arg_answer",
				Variable: "$vs_default_cafe_matches_1_match_0_cond_2",
				Parameters: []version2.Parameter{
					{
						Value:  `"yes"`,
						Result: "$vs_default_cafe_matches_1_match_0_cond_3",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$request_method",
				Variable: "$vs_default_cafe_matches_1_match_0_cond_3",
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
				Variable: "$vs_default_cafe_matches_1_match_1_cond_0",
				Parameters: []version2.Parameter{
					{
						Value:  `"v2"`,
						Result: "$vs_default_cafe_matches_1_match_1_cond_1",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$cookie_user",
				Variable: "$vs_default_cafe_matches_1_match_1_cond_1",
				Parameters: []version2.Parameter{
					{
						Value:  `"paul"`,
						Result: "$vs_default_cafe_matches_1_match_1_cond_2",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$arg_answer",
				Variable: "$vs_default_cafe_matches_1_match_1_cond_2",
				Parameters: []version2.Parameter{
					{
						Value:  `"no"`,
						Result: "$vs_default_cafe_matches_1_match_1_cond_3",
					},
					{
						Value:  "default",
						Result: "0",
					},
				},
			},
			{
				Source:   "$request_method",
				Variable: "$vs_default_cafe_matches_1_match_1_cond_3",
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
				Source:   "$vs_default_cafe_matches_1_match_0_cond_0$vs_default_cafe_matches_1_match_1_cond_0",
				Variable: "$vs_default_cafe_matches_1",
				Parameters: []version2.Parameter{
					{
						Value:  "~^1",
						Result: "/internal_location_matches_1_match_0",
					},
					{
						Value:  "~^01",
						Result: "$vs_default_cafe_splits_2",
					},
					{
						Value:  "default",
						Result: "/internal_location_matches_1_default",
					},
				},
			},
		},
		Locations: []version2.Location{
			{
				Path:                     "/internal_location_matches_1_match_0",
				ProxyPass:                "http://vs_default_cafe_coffee-v1$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				ProxyInterceptErrors:     true,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_2_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxySSLName:            "coffee-v1.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "coffee-v1",
				IsVSR:                   false,
				VSRName:                 "",
				VSRNamespace:            "",
			},
			{
				Path:                     "/internal_location_splits_2_split_0",
				ProxyPass:                "http://vs_default_cafe_coffee-v1$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				ProxyInterceptErrors:     true,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_2_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxySSLName:            "coffee-v1.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "coffee-v1",
				IsVSR:                   false,
				VSRName:                 "",
				VSRNamespace:            "",
			},
			{
				Path:                     "/internal_location_splits_2_split_1",
				ProxyPass:                "http://vs_default_cafe_coffee-v2$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				ProxyInterceptErrors:     true,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_2_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxySSLName:            "coffee-v2.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "coffee-v2",
				IsVSR:                   false,
				VSRName:                 "",
				VSRNamespace:            "",
			},
			{
				Path:                     "/internal_location_matches_1_default",
				ProxyPass:                "http://vs_default_cafe_tea$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				ProxyInterceptErrors:     true,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_2_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxySSLName:            "tea.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "tea",
				IsVSR:                   false,
				VSRName:                 "",
				VSRNamespace:            "",
			},
		},
		InternalRedirectLocation: version2.InternalRedirectLocation{
			Path:        "/",
			Destination: "$vs_default_cafe_matches_1",
		},
		SplitClients: []version2.SplitClient{
			{
				Source:   "$request_id",
				Variable: "$vs_default_cafe_splits_2",
				Distributions: []version2.Distribution{
					{
						Weight: "90%",
						Value:  "/internal_location_splits_2_split_0",
					},
					{
						Weight: "10%",
						Value:  "/internal_location_splits_2_split_1",
					},
				},
			},
		},
	}

	cfgParams := ConfigParams{}
	enableSnippets := false
	locSnippets := ""
	crUpstreams := map[string]conf_v1.Upstream{
		"vs_default_cafe_coffee-v1": {Service: "coffee-v1"},
		"vs_default_cafe_coffee-v2": {Service: "coffee-v2"},
		"vs_default_cafe_tea":       {Service: "tea"},
	}

	result := generateMatchesConfig(
		route,
		upstreamNamer,
		crUpstreams,
		variableNamer,
		index,
		scIndex,
		&cfgParams,
		errorPages,
		2,
		locSnippets,
		enableSnippets,
		0,
		false,
		"",
		"",
	)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateMatchesConfig() returned \n%+v but expected \n%+v", result, expected)
	}
}

func TestGenerateMatchesConfigWithMultipleSplits(t *testing.T) {
	route := conf_v1.Route{
		Path: "/",
		Matches: []conf_v1.Match{
			{
				Conditions: []conf_v1.Condition{
					{
						Header: "x-version",
						Value:  "v1",
					},
				},
				Splits: []conf_v1.Split{
					{
						Weight: 30,
						Action: &conf_v1.Action{
							Pass: "coffee-v1",
						},
					},
					{
						Weight: 70,
						Action: &conf_v1.Action{
							Pass: "coffee-v2",
						},
					},
				},
			},
			{
				Conditions: []conf_v1.Condition{
					{
						Header: "x-version",
						Value:  "v2",
					},
				},
				Splits: []conf_v1.Split{
					{
						Weight: 90,
						Action: &conf_v1.Action{
							Pass: "coffee-v2",
						},
					},
					{
						Weight: 10,
						Action: &conf_v1.Action{
							Pass: "coffee-v1",
						},
					},
				},
			},
		},
		Splits: []conf_v1.Split{
			{
				Weight: 99,
				Action: &conf_v1.Action{
					Pass: "coffee-v1",
				},
			},
			{
				Weight: 1,
				Action: &conf_v1.Action{
					Pass: "coffee-v2",
				},
			},
		},
	}
	virtualServer := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe",
			Namespace: "default",
		},
	}
	upstreamNamer := newUpstreamNamerForVirtualServer(&virtualServer)
	variableNamer := newVariableNamer(&virtualServer)
	index := 1
	scIndex := 2
	errorPages := []conf_v1.ErrorPage{
		{
			Codes: []int{400, 500},
			Return: &conf_v1.ErrorPageReturn{
				ActionReturn: conf_v1.ActionReturn{
					Code: 200,
					Type: "application/json",
					Body: `{\"message\": \"ok\"}`,
				},
				Headers: []conf_v1.Header{
					{
						Name:  "Set-Cookie",
						Value: "cookie1=value",
					},
				},
			},
			Redirect: nil,
		},
		{
			Codes:  []int{500, 502},
			Return: nil,
			Redirect: &conf_v1.ErrorPageRedirect{
				ActionRedirect: conf_v1.ActionRedirect{
					URL:  "http://nginx.com",
					Code: 301,
				},
			},
		},
	}

	expected := routingCfg{
		Maps: []version2.Map{
			{
				Source:   "$http_x_version",
				Variable: "$vs_default_cafe_matches_1_match_0_cond_0",
				Parameters: []version2.Parameter{
					{
						Value:  `"v1"`,
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
				Variable: "$vs_default_cafe_matches_1_match_1_cond_0",
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
				Source:   "$vs_default_cafe_matches_1_match_0_cond_0$vs_default_cafe_matches_1_match_1_cond_0",
				Variable: "$vs_default_cafe_matches_1",
				Parameters: []version2.Parameter{
					{
						Value:  "~^1",
						Result: "$vs_default_cafe_splits_2",
					},
					{
						Value:  "~^01",
						Result: "$vs_default_cafe_splits_3",
					},
					{
						Value:  "default",
						Result: "$vs_default_cafe_splits_4",
					},
				},
			},
		},
		Locations: []version2.Location{
			{
				Path:                     "/internal_location_splits_2_split_0",
				ProxyPass:                "http://vs_default_cafe_coffee-v1$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_0_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxyInterceptErrors:    true,
				ProxySSLName:            "coffee-v1.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "coffee-v1",
				IsVSR:                   true,
				VSRName:                 "coffee",
				VSRNamespace:            "default",
			},
			{
				Path:                     "/internal_location_splits_2_split_1",
				ProxyPass:                "http://vs_default_cafe_coffee-v2$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_0_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxyInterceptErrors:    true,
				ProxySSLName:            "coffee-v2.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "coffee-v2",
				IsVSR:                   true,
				VSRName:                 "coffee",
				VSRNamespace:            "default",
			},
			{
				Path:                     "/internal_location_splits_3_split_0",
				ProxyPass:                "http://vs_default_cafe_coffee-v2$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_0_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxyInterceptErrors:    true,
				ProxySSLName:            "coffee-v2.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "coffee-v2",
				IsVSR:                   true,
				VSRName:                 "coffee",
				VSRNamespace:            "default",
			},
			{
				Path:                     "/internal_location_splits_3_split_1",
				ProxyPass:                "http://vs_default_cafe_coffee-v1$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_0_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxyInterceptErrors:    true,
				ProxySSLName:            "coffee-v1.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "coffee-v1",
				IsVSR:                   true,
				VSRName:                 "coffee",
				VSRNamespace:            "default",
			},
			{
				Path:                     "/internal_location_splits_4_split_0",
				ProxyPass:                "http://vs_default_cafe_coffee-v1$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_0_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxyInterceptErrors:    true,
				ProxySSLName:            "coffee-v1.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "coffee-v1",
				IsVSR:                   true,
				VSRName:                 "coffee",
				VSRNamespace:            "default",
			},
			{
				Path:                     "/internal_location_splits_4_split_1",
				ProxyPass:                "http://vs_default_cafe_coffee-v2$request_uri",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "0s",
				ProxyNextUpstreamTries:   0,
				Internal:                 true,
				ErrorPages: []version2.ErrorPage{
					{
						Name:         "@error_page_0_0",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "http://nginx.com",
						Codes:        "500 502",
						ResponseCode: 301,
					},
				},
				ProxyInterceptErrors:    true,
				ProxySSLName:            "coffee-v2.default.svc",
				ProxyPassRequestHeaders: true,
				ProxySetHeaders:         []version2.Header{{Name: "Host", Value: "$host"}},
				ServiceName:             "coffee-v2",
				IsVSR:                   true,
				VSRName:                 "coffee",
				VSRNamespace:            "default",
			},
		},
		InternalRedirectLocation: version2.InternalRedirectLocation{
			Path:        "/",
			Destination: "$vs_default_cafe_matches_1",
		},
		SplitClients: []version2.SplitClient{
			{
				Source:   "$request_id",
				Variable: "$vs_default_cafe_splits_2",
				Distributions: []version2.Distribution{
					{
						Weight: "30%",
						Value:  "/internal_location_splits_2_split_0",
					},
					{
						Weight: "70%",
						Value:  "/internal_location_splits_2_split_1",
					},
				},
			},
			{
				Source:   "$request_id",
				Variable: "$vs_default_cafe_splits_3",
				Distributions: []version2.Distribution{
					{
						Weight: "90%",
						Value:  "/internal_location_splits_3_split_0",
					},
					{
						Weight: "10%",
						Value:  "/internal_location_splits_3_split_1",
					},
				},
			},
			{
				Source:   "$request_id",
				Variable: "$vs_default_cafe_splits_4",
				Distributions: []version2.Distribution{
					{
						Weight: "99%",
						Value:  "/internal_location_splits_4_split_0",
					},
					{
						Weight: "1%",
						Value:  "/internal_location_splits_4_split_1",
					},
				},
			},
		},
	}

	cfgParams := ConfigParams{}
	enableSnippets := false
	locSnippets := ""
	crUpstreams := map[string]conf_v1.Upstream{
		"vs_default_cafe_coffee-v1": {Service: "coffee-v1"},
		"vs_default_cafe_coffee-v2": {Service: "coffee-v2"},
	}
	result := generateMatchesConfig(
		route,
		upstreamNamer,
		crUpstreams,
		variableNamer,
		index,
		scIndex,
		&cfgParams,
		errorPages,
		0,
		locSnippets,
		enableSnippets,
		0,
		true,
		"coffee",
		"default",
	)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateMatchesConfig() returned \n%+v but expected \n%+v", result, expected)
	}
}

func TestGenerateValueForMatchesRouteMap(t *testing.T) {
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
		resultValue, resultIsNegative := generateValueForMatchesRouteMap(test.input)
		if resultValue != test.expectedValue {
			t.Errorf("generateValueForMatchesRouteMap(%q) returned %q but expected %q as the value", test.input, resultValue, test.expectedValue)
		}
		if resultIsNegative != test.expectedIsNegative {
			t.Errorf("generateValueForMatchesRouteMap(%q) returned %v but expected %v as the isNegative", test.input, resultIsNegative, test.expectedIsNegative)
		}
	}
}

func TestGenerateParametersForMatchesRouteMap(t *testing.T) {
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
		result := generateParametersForMatchesRouteMap(test.inputMatchedValue, test.inputSuccessfulResult)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateParametersForMatchesRouteMap(%q, %q) returned %v but expected %v", test.inputMatchedValue, test.inputSuccessfulResult, result, test.expected)
		}
	}
}

func TestGetNameForSourceForMatchesRouteMapFromCondition(t *testing.T) {
	tests := []struct {
		input    conf_v1.Condition
		expected string
	}{
		{
			input: conf_v1.Condition{
				Header: "x-version",
			},
			expected: "$http_x_version",
		},
		{
			input: conf_v1.Condition{
				Cookie: "mycookie",
			},
			expected: "$cookie_mycookie",
		},
		{
			input: conf_v1.Condition{
				Argument: "arg",
			},
			expected: "$arg_arg",
		},
		{
			input: conf_v1.Condition{
				Variable: "$request_method",
			},
			expected: "$request_method",
		},
	}

	for _, test := range tests {
		result := getNameForSourceForMatchesRouteMapFromCondition(test.input)
		if result != test.expected {
			t.Errorf("getNameForSourceForMatchesRouteMapFromCondition() returned %q but expected %q for input %v", result, test.expected, test.input)
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
		upstream  conf_v1.Upstream
		cfgParams *ConfigParams
		expected  bool
		msg       string
	}{
		{
			conf_v1.Upstream{},
			&ConfigParams{Keepalive: keepalive},
			true,
			"upstream keepalive not set, configparam keepalive set",
		},
		{
			conf_v1.Upstream{Keepalive: &noKeepalive},
			&ConfigParams{Keepalive: keepalive},
			false,
			"upstream keepalive set to 0, configparam keepalive set",
		},
		{
			conf_v1.Upstream{Keepalive: &keepalive},
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

	result := newHealthCheckWithDefaults(conf_v1.Upstream{}, upstreamName, baseCfgParams)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("newHealthCheckWithDefaults returned \n%v but expected \n%v", result, expected)
	}
}

func TestGenerateHealthCheck(t *testing.T) {
	upstreamName := "test-upstream"
	tests := []struct {
		upstream     conf_v1.Upstream
		upstreamName string
		expected     *version2.HealthCheck
		msg          string
	}{
		{

			upstream: conf_v1.Upstream{
				HealthCheck: &conf_v1.HealthCheck{
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
					Headers: []conf_v1.Header{
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
			upstream: conf_v1.Upstream{
				HealthCheck: &conf_v1.HealthCheck{
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
			upstream: conf_v1.Upstream{
				HealthCheck: &conf_v1.HealthCheck{
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
			upstream:     conf_v1.Upstream{},
			upstreamName: upstreamName,
			expected:     nil,
			msg:          "HealthCheck not enabled",
		},
		{
			upstream: conf_v1.Upstream{
				HealthCheck: &conf_v1.HealthCheck{
					Enable:         true,
					Interval:       "1m 5s",
					Jitter:         "2m 3s",
					ConnectTimeout: "1m 10s",
					SendTimeout:    "1m 20s",
					ReadTimeout:    "1m 30s",
				},
			},
			upstreamName: upstreamName,
			expected: &version2.HealthCheck{
				Name:                upstreamName,
				ProxyConnectTimeout: "1m10s",
				ProxySendTimeout:    "1m20s",
				ProxyReadTimeout:    "1m30s",
				ProxyPass:           fmt.Sprintf("http://%v", upstreamName),
				URI:                 "/",
				Interval:            "1m5s",
				Jitter:              "2m3s",
				Fails:               1,
				Passes:              1,
				Headers:             make(map[string]string),
			},
			msg: "HealthCheck with time parameters have correct format",
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
		upstream             conf_v1.Upstream
		vsEx                 *VirtualServerEx
		isPlus               bool
		isResolverConfigured bool
		expected             []string
		warningsExpected     bool
		msg                  string
	}{
		{
			upstream: conf_v1.Upstream{
				Service: name,
				Port:    80,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
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
			upstream: conf_v1.Upstream{
				Service: name,
				Port:    80,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
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
			upstream: conf_v1.Upstream{
				Service: name,
				Port:    8080,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
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
			upstream: conf_v1.Upstream{
				Service: name,
				Port:    8080,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
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
			upstream: conf_v1.Upstream{
				Service: name,
				Port:    8080,
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
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
			upstream: conf_v1.Upstream{
				Service:     name,
				Port:        8080,
				Subselector: map[string]string{"version": "test"},
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
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
			upstream: conf_v1.Upstream{
				Service:     name,
				Port:        8080,
				Subselector: map[string]string{"version": "test"},
			},
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
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
		isWildcardEnabled := false
		vsc := newVirtualServerConfigurator(
			&ConfigParams{},
			test.isPlus,
			test.isResolverConfigured,
			&StaticConfigParams{},
			isWildcardEnabled,
		)
		result := vsc.generateEndpointsForUpstream(test.vsEx.VirtualServer, namespace, test.upstream, test.vsEx)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateEndpointsForUpstream(isPlus=%v, isResolverConfigured=%v) returned %v, but expected %v for case: %v",
				test.isPlus, test.isResolverConfigured, result, test.expected, test.msg)
		}

		if len(vsc.warnings) == 0 && test.warningsExpected {
			t.Errorf(
				"generateEndpointsForUpstream(isPlus=%v, isResolverConfigured=%v) didn't return any warnings for %v but warnings expected",
				test.isPlus,
				test.isResolverConfigured,
				test.upstream,
			)
		}

		if len(vsc.warnings) != 0 && !test.warningsExpected {
			t.Errorf("generateEndpointsForUpstream(isPlus=%v, isResolverConfigured=%v) returned warnings for %v",
				test.isPlus, test.isResolverConfigured, test.upstream)
		}
	}
}

func TestGenerateSlowStartForPlusWithInCompatibleLBMethods(t *testing.T) {
	serviceName := "test-slowstart-with-incompatible-LBMethods"
	upstream := conf_v1.Upstream{Service: serviceName, Port: 80, SlowStart: "10s"}
	expected := ""

	tests := []string{
		"random",
		"ip_hash",
		"hash 123",
		"random two",
		"random two least_conn",
		"random two least_time=header",
		"random two least_time=last_byte",
	}

	for _, lbMethod := range tests {
		vsc := newVirtualServerConfigurator(&ConfigParams{}, true, false, &StaticConfigParams{}, false)
		result := vsc.generateSlowStartForPlus(&conf_v1.VirtualServer{}, upstream, lbMethod)

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
		upstream conf_v1.Upstream
		lbMethod string
		expected string
	}{
		{
			upstream: conf_v1.Upstream{Service: serviceName, Port: 80, SlowStart: "", LBMethod: "least_conn"},
			lbMethod: "least_conn",
			expected: "",
		},
		{
			upstream: conf_v1.Upstream{Service: serviceName, Port: 80, SlowStart: "10s", LBMethod: "least_conn"},
			lbMethod: "least_conn",
			expected: "10s",
		},
	}

	for _, test := range tests {
		vsc := newVirtualServerConfigurator(&ConfigParams{}, true, false, &StaticConfigParams{}, false)
		result := vsc.generateSlowStartForPlus(&conf_v1.VirtualServer{}, test.upstream, test.lbMethod)
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
		upstream conf_v1.Upstream
		isPlus   bool
		expected version2.Upstream
		msg      string
	}{
		{
			name: "test-upstream-queue",
			upstream: conf_v1.Upstream{Service: serviceName, Port: 80, Queue: &conf_v1.UpstreamQueue{
				Size:    10,
				Timeout: "10s",
			}},
			isPlus: true,
			expected: version2.Upstream{
				UpstreamLabels: version2.UpstreamLabels{
					Service: "test-queue",
				},
				Name: "test-upstream-queue",
				Queue: &version2.Queue{
					Size:    10,
					Timeout: "10s",
				},
			},
			msg: "upstream queue with size and timeout",
		},
		{
			name: "test-upstream-queue-with-default-timeout",
			upstream: conf_v1.Upstream{
				Service: serviceName,
				Port:    80,
				Queue:   &conf_v1.UpstreamQueue{Size: 10, Timeout: ""},
			},
			isPlus: true,
			expected: version2.Upstream{
				UpstreamLabels: version2.UpstreamLabels{
					Service: "test-queue",
				},
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
			upstream: conf_v1.Upstream{Service: serviceName, Port: 80, Queue: nil},
			isPlus:   false,
			expected: version2.Upstream{
				UpstreamLabels: version2.UpstreamLabels{
					Service: "test-queue",
				},
				Name: "test-upstream-queue-nil",
			},
			msg: "upstream queue with nil for OSS",
		},
	}

	for _, test := range tests {
		vsc := newVirtualServerConfigurator(&ConfigParams{}, test.isPlus, false, &StaticConfigParams{}, false)
		result := vsc.generateUpstream(nil, test.name, test.upstream, false, []string{})
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateUpstream() returned %v but expected %v for the case of %v", result, test.expected, test.msg)
		}
	}
}

func TestGenerateQueueForPlus(t *testing.T) {
	tests := []struct {
		upstreamQueue *conf_v1.UpstreamQueue
		expected      *version2.Queue
		msg           string
	}{
		{
			upstreamQueue: &conf_v1.UpstreamQueue{Size: 10, Timeout: "10s"},
			expected:      &version2.Queue{Size: 10, Timeout: "10s"},
			msg:           "upstream queue with size and timeout",
		},
		{
			upstreamQueue: nil,
			expected:      nil,
			msg:           "upstream queue with nil",
		},
		{
			upstreamQueue: &conf_v1.UpstreamQueue{Size: 10},
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
		sc       *conf_v1.SessionCookie
		expected *version2.SessionCookie
		msg      string
	}{
		{
			sc:       &conf_v1.SessionCookie{Enable: true, Name: "test"},
			expected: &version2.SessionCookie{Enable: true, Name: "test"},
			msg:      "session cookie with name",
		},
		{
			sc:       nil,
			expected: nil,
			msg:      "session cookie with nil",
		},
		{
			sc:       &conf_v1.SessionCookie{Name: "test"},
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

func TestGeneratePath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{
			path:     "/",
			expected: "/",
		},
		{
			path:     "=/exact/match",
			expected: "=/exact/match",
		},
		{
			path:     `~ *\\.jpg`,
			expected: `~ "*\\.jpg"`,
		},
		{
			path:     `~* *\\.PNG`,
			expected: `~* "*\\.PNG"`,
		},
	}

	for _, test := range tests {
		result := generatePath(test.path)
		if result != test.expected {
			t.Errorf("generatePath() returned %v, but expected %v.", result, test.expected)
		}
	}
}

func TestGenerateErrorPageName(t *testing.T) {
	tests := []struct {
		routeIndex int
		index      int
		expected   string
	}{
		{
			0,
			0,
			"@error_page_0_0",
		},
		{
			0,
			1,
			"@error_page_0_1",
		},
		{
			1,
			0,
			"@error_page_1_0",
		},
	}

	for _, test := range tests {
		result := generateErrorPageName(test.routeIndex, test.index)
		if result != test.expected {
			t.Errorf("generateErrorPageName(%v, %v) returned %v but expected %v", test.routeIndex, test.index, result, test.expected)
		}
	}
}

func TestGenerateErrorPageCodes(t *testing.T) {
	tests := []struct {
		codes    []int
		expected string
	}{
		{
			codes:    []int{400},
			expected: "400",
		},
		{
			codes:    []int{404, 405, 502},
			expected: "404 405 502",
		},
	}

	for _, test := range tests {
		result := generateErrorPageCodes(test.codes)
		if result != test.expected {
			t.Errorf("generateErrorPageCodes(%v) returned %v but expected %v", test.codes, result, test.expected)
		}
	}
}

func TestGenerateErrorPages(t *testing.T) {
	tests := []struct {
		upstreamName string
		errorPages   []conf_v1.ErrorPage
		expected     []version2.ErrorPage
	}{
		{}, // empty errorPages
		{
			"vs_test_test",
			[]conf_v1.ErrorPage{
				{
					Codes: []int{404, 405, 500, 502},
					Return: &conf_v1.ErrorPageReturn{
						ActionReturn: conf_v1.ActionReturn{
							Code: 200,
						},
						Headers: nil,
					},
					Redirect: nil,
				},
			},
			[]version2.ErrorPage{
				{
					Name:         "@error_page_1_0",
					Codes:        "404 405 500 502",
					ResponseCode: 200,
				},
			},
		},
		{
			"vs_test_test",
			[]conf_v1.ErrorPage{
				{
					Codes:  []int{404, 405, 500, 502},
					Return: nil,
					Redirect: &conf_v1.ErrorPageRedirect{
						ActionRedirect: conf_v1.ActionRedirect{
							URL:  "http://nginx.org",
							Code: 302,
						},
					},
				},
			},
			[]version2.ErrorPage{
				{
					Name:         "http://nginx.org",
					Codes:        "404 405 500 502",
					ResponseCode: 302,
				},
			},
		},
	}

	for i, test := range tests {
		result := generateErrorPages(i, test.errorPages)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateErrorPages(%v, %v) returned %v but expected %v", test.upstreamName, test.errorPages, result, test.expected)
		}
	}
}

func TestGenerateErrorPageLocations(t *testing.T) {
	tests := []struct {
		upstreamName string
		errorPages   []conf_v1.ErrorPage
		expected     []version2.ErrorPageLocation
	}{
		{},
		{
			"vs_test_test",
			[]conf_v1.ErrorPage{
				{
					Codes:  []int{404, 405, 500, 502},
					Return: nil,
					Redirect: &conf_v1.ErrorPageRedirect{
						ActionRedirect: conf_v1.ActionRedirect{
							URL:  "http://nginx.org",
							Code: 302,
						},
					},
				},
			},
			nil,
		},
		{
			"vs_test_test",
			[]conf_v1.ErrorPage{
				{
					Codes: []int{404, 405, 500, 502},
					Return: &conf_v1.ErrorPageReturn{
						ActionReturn: conf_v1.ActionReturn{
							Code: 200,
							Type: "application/json",
							Body: "Hello World",
						},
						Headers: []conf_v1.Header{
							{
								Name:  "HeaderName",
								Value: "HeaderValue",
							},
						},
					},
					Redirect: nil,
				},
			},
			[]version2.ErrorPageLocation{
				{
					Name:        "@error_page_2_0",
					DefaultType: "application/json",
					Return: &version2.Return{
						Code: 0,
						Text: "Hello World",
					},
					Headers: []version2.Header{
						{
							Name:  "HeaderName",
							Value: "HeaderValue",
						},
					},
				},
			},
		},
	}

	for i, test := range tests {
		result := generateErrorPageLocations(i, test.errorPages)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateErrorPageLocations(%v, %v) returned %v but expected %v", test.upstreamName, test.errorPages, result, test.expected)
		}
	}
}

func TestGenerateProxySSLName(t *testing.T) {
	result := generateProxySSLName("coffee-v1", "default")
	if result != "coffee-v1.default.svc" {
		t.Errorf("generateProxySSLName(coffee-v1, default) returned %v but expected coffee-v1.default.svc", result)
	}
}

func TestIsTLSEnabled(t *testing.T) {
	tests := []struct {
		upstream   conf_v1.Upstream
		spiffeCert bool
		expected   bool
	}{
		{
			upstream: conf_v1.Upstream{
				TLS: conf_v1.UpstreamTLS{
					Enable: false,
				},
			},
			spiffeCert: false,
			expected:   false,
		},
		{
			upstream: conf_v1.Upstream{
				TLS: conf_v1.UpstreamTLS{
					Enable: false,
				},
			},
			spiffeCert: true,
			expected:   true,
		},
		{
			upstream: conf_v1.Upstream{
				TLS: conf_v1.UpstreamTLS{
					Enable: true,
				},
			},
			spiffeCert: true,
			expected:   true,
		},
		{
			upstream: conf_v1.Upstream{
				TLS: conf_v1.UpstreamTLS{
					Enable: true,
				},
			},
			spiffeCert: false,
			expected:   true,
		},
	}

	for _, test := range tests {
		result := isTLSEnabled(test.upstream, test.spiffeCert)
		if result != test.expected {
			t.Errorf("isTLSEnabled(%v, %v) returned %v but expected %v", test.upstream, test.spiffeCert, result, test.expected)
		}
	}
}

func TestGenerateRewrites(t *testing.T) {
	tests := []struct {
		path         string
		proxy        *conf_v1.ActionProxy
		internal     bool
		originalPath string
		expected     []string
	}{
		{
			proxy:    nil,
			expected: nil,
		},
		{
			proxy: &conf_v1.ActionProxy{
				RewritePath: "",
			},
			expected: nil,
		},
		{
			path: "/path",
			proxy: &conf_v1.ActionProxy{
				RewritePath: "/rewrite",
			},
			expected: nil,
		},
		{
			path:     "/_internal_path",
			internal: true,
			proxy: &conf_v1.ActionProxy{
				RewritePath: "/rewrite",
			},
			originalPath: "/path",
			expected:     []string{`^ $request_uri`, `"^/path(.*)$" "/rewrite$1" break`},
		},
		{
			path:     "~/regex",
			internal: true,
			proxy: &conf_v1.ActionProxy{
				RewritePath: "/rewrite",
			},
			originalPath: "/path",
			expected:     []string{`^ $request_uri`, `"^/path(.*)$" "/rewrite$1" break`},
		},
		{
			path:     "~/regex",
			internal: false,
			proxy: &conf_v1.ActionProxy{
				RewritePath: "/rewrite",
			},
			expected: []string{`"^/regex" "/rewrite" break`},
		},
	}

	for _, test := range tests {
		result := generateRewrites(test.path, test.proxy, test.internal, test.originalPath)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateRewrites(%v, %v, %v, %v) returned \n %v but expected \n %v",
				test.path, test.proxy, test.internal, test.originalPath, result, test.expected)
		}
	}
}

func TestGenerateProxyPassRewrite(t *testing.T) {
	tests := []struct {
		path     string
		proxy    *conf_v1.ActionProxy
		internal bool
		expected string
	}{
		{
			expected: "",
		},
		{
			internal: true,
			proxy: &conf_v1.ActionProxy{
				RewritePath: "/rewrite",
			},
			expected: "",
		},
		{
			path: "/path",
			proxy: &conf_v1.ActionProxy{
				RewritePath: "/rewrite",
			},
			expected: "/rewrite",
		},
		{
			path: "=/path",
			proxy: &conf_v1.ActionProxy{
				RewritePath: "/rewrite",
			},
			expected: "/rewrite",
		},
		{
			path: "~/path",
			proxy: &conf_v1.ActionProxy{
				RewritePath: "/rewrite",
			},
			expected: "",
		},
	}

	for _, test := range tests {
		result := generateProxyPassRewrite(test.path, test.proxy, test.internal)
		if result != test.expected {
			t.Errorf("generateProxyPassRewrite(%v, %v, %v) returned %v but expected %v",
				test.path, test.proxy, test.internal, result, test.expected)
		}
	}
}

func TestGenerateProxySetHeaders(t *testing.T) {
	tests := []struct {
		proxy    *conf_v1.ActionProxy
		expected []version2.Header
		msg      string
	}{
		{
			proxy:    nil,
			expected: []version2.Header{{Name: "Host", Value: "$host"}},
			msg:      "no action proxy",
		},
		{
			proxy:    &conf_v1.ActionProxy{},
			expected: []version2.Header{{Name: "Host", Value: "$host"}},
			msg:      "empty action proxy",
		},
		{
			proxy: &conf_v1.ActionProxy{
				RequestHeaders: &conf_v1.ProxyRequestHeaders{
					Set: []conf_v1.Header{
						{
							Name:  "Header-Name",
							Value: "HeaderValue",
						},
					},
				},
			},
			expected: []version2.Header{
				{
					Name:  "Header-Name",
					Value: "HeaderValue",
				},
				{
					Name:  "Host",
					Value: "$host",
				},
			},
			msg: "set headers without host",
		},
		{
			proxy: &conf_v1.ActionProxy{
				RequestHeaders: &conf_v1.ProxyRequestHeaders{
					Set: []conf_v1.Header{
						{
							Name:  "Header-Name",
							Value: "HeaderValue",
						},
						{
							Name:  "Host",
							Value: "example.com",
						},
					},
				},
			},
			expected: []version2.Header{
				{
					Name:  "Header-Name",
					Value: "HeaderValue",
				},
				{
					Name:  "Host",
					Value: "example.com",
				},
			},
			msg: "set headers with host capitalized",
		},
		{
			proxy: &conf_v1.ActionProxy{
				RequestHeaders: &conf_v1.ProxyRequestHeaders{
					Set: []conf_v1.Header{
						{
							Name:  "Header-Name",
							Value: "HeaderValue",
						},
						{
							Name:  "hoST",
							Value: "example.com",
						},
					},
				},
			},
			expected: []version2.Header{
				{
					Name:  "Header-Name",
					Value: "HeaderValue",
				},
				{
					Name:  "hoST",
					Value: "example.com",
				},
			},
			msg: "set headers with host in mixed case",
		},
		{
			proxy: &conf_v1.ActionProxy{
				RequestHeaders: &conf_v1.ProxyRequestHeaders{
					Set: []conf_v1.Header{
						{
							Name:  "Header-Name",
							Value: "HeaderValue",
						},
						{
							Name:  "Host",
							Value: "one.example.com",
						},
						{
							Name:  "Host",
							Value: "two.example.com",
						},
					},
				},
			},
			expected: []version2.Header{
				{
					Name:  "Header-Name",
					Value: "HeaderValue",
				},
				{
					Name:  "Host",
					Value: "one.example.com",
				},
				{
					Name:  "Host",
					Value: "two.example.com",
				},
			},
			msg: "set headers with multiple hosts",
		},
	}

	for _, test := range tests {
		result := generateProxySetHeaders(test.proxy)
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("generateProxySetHeaders() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestGenerateProxyPassRequestHeaders(t *testing.T) {
	passTrue := true
	passFalse := false
	tests := []struct {
		proxy    *conf_v1.ActionProxy
		expected bool
	}{
		{
			proxy:    nil,
			expected: true,
		},
		{
			proxy:    &conf_v1.ActionProxy{},
			expected: true,
		},
		{
			proxy: &conf_v1.ActionProxy{
				RequestHeaders: &conf_v1.ProxyRequestHeaders{
					Pass: nil,
				},
			},
			expected: true,
		},
		{
			proxy: &conf_v1.ActionProxy{
				RequestHeaders: &conf_v1.ProxyRequestHeaders{
					Pass: &passTrue,
				},
			},
			expected: true,
		},
		{
			proxy: &conf_v1.ActionProxy{
				RequestHeaders: &conf_v1.ProxyRequestHeaders{
					Pass: &passFalse,
				},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		result := generateProxyPassRequestHeaders(test.proxy)
		if result != test.expected {
			t.Errorf("generateProxyPassRequestHeaders(%v) returned %v but expected %v", test.proxy, result, test.expected)
		}
	}
}

func TestGenerateProxyHideHeaders(t *testing.T) {
	tests := []struct {
		proxy    *conf_v1.ActionProxy
		expected []string
	}{
		{
			proxy:    nil,
			expected: nil,
		},
		{
			proxy: &conf_v1.ActionProxy{
				ResponseHeaders: nil,
			},
		},
		{
			proxy: &conf_v1.ActionProxy{
				ResponseHeaders: &conf_v1.ProxyResponseHeaders{
					Hide: []string{"Header", "Header-2"},
				},
			},
			expected: []string{"Header", "Header-2"},
		},
	}

	for _, test := range tests {
		result := generateProxyHideHeaders(test.proxy)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateProxyHideHeaders(%v) returned %v but expected %v", test.proxy, result, test.expected)
		}
	}
}

func TestGenerateProxyPassHeaders(t *testing.T) {
	tests := []struct {
		proxy    *conf_v1.ActionProxy
		expected []string
	}{
		{
			proxy:    nil,
			expected: nil,
		},
		{
			proxy: &conf_v1.ActionProxy{
				ResponseHeaders: nil,
			},
		},
		{
			proxy: &conf_v1.ActionProxy{
				ResponseHeaders: &conf_v1.ProxyResponseHeaders{
					Pass: []string{"Header", "Header-2"},
				},
			},
			expected: []string{"Header", "Header-2"},
		},
	}

	for _, test := range tests {
		result := generateProxyPassHeaders(test.proxy)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateProxyPassHeaders(%v) returned %v but expected %v", test.proxy, result, test.expected)
		}
	}
}

func TestGenerateProxyIgnoreHeaders(t *testing.T) {
	tests := []struct {
		proxy    *conf_v1.ActionProxy
		expected string
	}{
		{
			proxy:    nil,
			expected: "",
		},
		{
			proxy: &conf_v1.ActionProxy{
				ResponseHeaders: nil,
			},
			expected: "",
		},
		{
			proxy: &conf_v1.ActionProxy{
				ResponseHeaders: &conf_v1.ProxyResponseHeaders{
					Ignore: []string{"Header", "Header-2"},
				},
			},
			expected: "Header Header-2",
		},
	}

	for _, test := range tests {
		result := generateProxyIgnoreHeaders(test.proxy)
		if result != test.expected {
			t.Errorf("generateProxyIgnoreHeaders(%v) returned %v but expected %v", test.proxy, result, test.expected)
		}
	}
}

func TestGenerateProxyAddHeaders(t *testing.T) {
	tests := []struct {
		proxy    *conf_v1.ActionProxy
		expected []version2.AddHeader
	}{
		{
			proxy:    nil,
			expected: nil,
		},
		{
			proxy:    &conf_v1.ActionProxy{},
			expected: nil,
		},
		{
			proxy: &conf_v1.ActionProxy{
				ResponseHeaders: &conf_v1.ProxyResponseHeaders{
					Add: []conf_v1.AddHeader{
						{
							Header: conf_v1.Header{
								Name:  "Header-Name",
								Value: "HeaderValue",
							},
							Always: true,
						},
						{
							Header: conf_v1.Header{
								Name:  "Server",
								Value: "myServer",
							},
							Always: false,
						},
					},
				},
			},
			expected: []version2.AddHeader{
				{
					Header: version2.Header{
						Name:  "Header-Name",
						Value: "HeaderValue",
					},
					Always: true,
				},
				{
					Header: version2.Header{
						Name:  "Server",
						Value: "myServer",
					},
					Always: false,
				},
			},
		},
	}

	for _, test := range tests {
		result := generateProxyAddHeaders(test.proxy)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("generateProxyAddHeaders(%v) returned %v but expected %v", test.proxy, result, test.expected)
		}
	}
}

func TestGetUpstreamResourceLabels(t *testing.T) {
	tests := []struct {
		owner    runtime.Object
		expected version2.UpstreamLabels
	}{
		{
			owner:    nil,
			expected: version2.UpstreamLabels{},
		},
		{
			owner: &conf_v1.VirtualServer{
				ObjectMeta: meta_v1.ObjectMeta{
					Namespace: "namespace",
					Name:      "name",
				},
			},
			expected: version2.UpstreamLabels{
				ResourceNamespace: "namespace",
				ResourceName:      "name",
				ResourceType:      "virtualserver",
			},
		},
		{
			owner: &conf_v1.VirtualServerRoute{
				ObjectMeta: meta_v1.ObjectMeta{
					Namespace: "namespace",
					Name:      "name",
				},
			},
			expected: version2.UpstreamLabels{
				ResourceNamespace: "namespace",
				ResourceName:      "name",
				ResourceType:      "virtualserverroute",
			},
		},
	}
	for _, test := range tests {
		result := getUpstreamResourceLabels(test.owner)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("getUpstreamResourceLabels(%+v) returned %+v but expected %+v", test.owner, result, test.expected)
		}
	}
}

func TestAddWafConfig(t *testing.T) {
	tests := []struct {
		wafInput     *conf_v1.WAF
		polKey       string
		polNamespace string
		apResources  map[string]string
		wafConfig    *version2.WAF
		expected     *validationResults
		msg          string
	}{
		{

			wafInput: &conf_v1.WAF{
				Enable: true,
			},
			polKey:       "default/waf-policy",
			polNamespace: "default",
			apResources:  map[string]string{},
			wafConfig: &version2.WAF{
				Enable: "on",
			},
			expected: &validationResults{isError: false},
			msg:      "valid waf config, default App Protect config",
		},
		{

			wafInput: &conf_v1.WAF{
				Enable:   true,
				ApPolicy: "dataguard-alarm",
				SecurityLog: &conf_v1.SecurityLog{
					Enable:    true,
					ApLogConf: "logconf",
					LogDest:   "syslog:server=127.0.0.1:514",
				},
			},
			polKey:       "default/waf-policy",
			polNamespace: "default",
			apResources: map[string]string{
				"default/dataguard-alarm": "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
				"default/logconf":         "/etc/nginx/waf/nac-logconfs/default-logconf",
			},
			wafConfig: &version2.WAF{
				ApPolicy:            "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
				ApSecurityLogEnable: true,
				ApLogConf:           "/etc/nginx/waf/nac-logconfs/default-logconf",
			},
			expected: &validationResults{isError: false},
			msg:      "valid waf config",
		},
		{

			wafInput: &conf_v1.WAF{
				Enable:   true,
				ApPolicy: "default/dataguard-alarm",
				SecurityLog: &conf_v1.SecurityLog{
					Enable:    true,
					ApLogConf: "default/logconf",
					LogDest:   "syslog:server=127.0.0.1:514",
				},
			},
			polKey:       "default/waf-policy",
			polNamespace: "",
			apResources: map[string]string{
				"default/dataguard-alarm": "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
			},
			wafConfig: &version2.WAF{
				ApPolicy:            "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
				ApSecurityLogEnable: true,
				ApLogConf:           "/etc/nginx/waf/nac-logconfs/default-logconf",
			},
			expected: &validationResults{
				isError: true,
				warnings: []string{
					`WAF policy default/waf-policy references an invalid or non-existing log config default/logconf`,
				},
			},
			msg: "invalid waf config, apLogConf references non-existing log conf",
		},
		{

			wafInput: &conf_v1.WAF{
				Enable:   true,
				ApPolicy: "default/dataguard-alarm",
				SecurityLog: &conf_v1.SecurityLog{
					Enable:  true,
					LogDest: "syslog:server=127.0.0.1:514",
				},
			},
			polKey:       "default/waf-policy",
			polNamespace: "",
			apResources: map[string]string{
				"default/logconf": "/etc/nginx/waf/nac-logconfs/default-logconf",
			},
			wafConfig: &version2.WAF{
				ApPolicy:            "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
				ApSecurityLogEnable: true,
				ApLogConf:           "/etc/nginx/waf/nac-logconfs/default-logconf",
			},
			expected: &validationResults{
				isError: true,
				warnings: []string{
					`WAF policy default/waf-policy references an invalid or non-existing App Protect policy default/dataguard-alarm`,
				},
			},
			msg: "invalid waf config, apLogConf references non-existing ap conf",
		},
		{

			wafInput: &conf_v1.WAF{
				Enable:   true,
				ApPolicy: "ns1/dataguard-alarm",
				SecurityLog: &conf_v1.SecurityLog{
					Enable:    true,
					ApLogConf: "ns2/logconf",
					LogDest:   "syslog:server=127.0.0.1:514",
				},
			},
			polKey:       "default/waf-policy",
			polNamespace: "",
			apResources: map[string]string{
				"ns1/dataguard-alarm": "/etc/nginx/waf/nac-policies/ns1-dataguard-alarm",
				"ns2/logconf":         "/etc/nginx/waf/nac-logconfs/ns2-logconf",
			},
			wafConfig: &version2.WAF{
				ApPolicy:            "/etc/nginx/waf/nac-policies/ns1-dataguard-alarm",
				ApSecurityLogEnable: true,
				ApLogConf:           "/etc/nginx/waf/nac-logconfs/ns2-logconf",
			},
			expected: &validationResults{},
			msg:      "valid waf config, cross ns reference",
		},
		{

			wafInput: &conf_v1.WAF{
				Enable:   false,
				ApPolicy: "dataguard-alarm",
			},
			polKey:       "default/waf-policy",
			polNamespace: "default",
			apResources: map[string]string{
				"default/dataguard-alarm": "/etc/nginx/waf/nac-policies/ns1-dataguard-alarm",
				"default/logconf":         "/etc/nginx/waf/nac-logconfs/ns2-logconf",
			},
			wafConfig: &version2.WAF{
				Enable:   "off",
				ApPolicy: "/etc/nginx/waf/nac-policies/ns1-dataguard-alarm",
			},
			expected: &validationResults{},
			msg:      "valid waf config, disable waf",
		},
	}

	for _, test := range tests {
		polCfg := newPoliciesConfig()
		result := polCfg.addWAFConfig(test.wafInput, test.polKey, test.polNamespace, test.apResources)
		if diff := cmp.Diff(test.expected.warnings, result.warnings); diff != "" {
			t.Errorf("policiesCfg.addWAFConfig() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestGenerateTime(t *testing.T) {
	tests := []struct {
		value, expected string
	}{
		{
			value:    "0s",
			expected: "0s",
		},
		{
			value:    "0",
			expected: "0s",
		},
		{
			value:    "1h",
			expected: "1h",
		},
		{
			value:    "1h 30m",
			expected: "1h30m",
		},
	}

	for _, test := range tests {
		result := generateTime(test.value)
		if result != test.expected {
			t.Errorf("generateTime(%q) returned %q but expected %q", test.value, result, test.expected)
		}
	}
}

func TestGenerateTimeWithDefault(t *testing.T) {
	tests := []struct {
		value, defaultValue, expected string
	}{
		{
			value:        "1h 30m",
			defaultValue: "",
			expected:     "1h30m",
		},
		{
			value:        "",
			defaultValue: "60s",
			expected:     "60s",
		},
		{
			value:        "",
			defaultValue: "test",
			expected:     "test",
		},
	}

	for _, test := range tests {
		result := generateTimeWithDefault(test.value, test.defaultValue)
		if result != test.expected {
			t.Errorf("generateTimeWithDefault(%q, %q) returned %q but expected %q", test.value, test.defaultValue, result, test.expected)
		}
	}
}

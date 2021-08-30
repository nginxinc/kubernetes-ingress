package version2

import (
	"testing"
)

const (
	nginxPlusVirtualServerTmpl   = "nginx-plus.virtualserver.tmpl"
	nginxVirtualServerTmpl       = "nginx.virtualserver.tmpl"
	nginxPlusTransportServerTmpl = "nginx-plus.transportserver.tmpl"
	nginxTransportServerTmpl     = "nginx.transportserver.tmpl"
)

var virtualServerCfg = VirtualServerConfig{
	LimitReqZones: []LimitReqZone{
		{
			ZoneName: "pol_rl_test_test_test", Rate: "10r/s", ZoneSize: "10m", Key: "$url",
		},
	},
	Upstreams: []Upstream{
		{
			Name: "test-upstream",
			Servers: []UpstreamServer{
				{
					Address: "10.0.0.20:8001",
				},
			},
			LBMethod:         "random",
			Keepalive:        32,
			MaxFails:         4,
			FailTimeout:      "10s",
			MaxConns:         31,
			SlowStart:        "10s",
			UpstreamZoneSize: "256k",
			Queue:            &Queue{Size: 10, Timeout: "60s"},
			SessionCookie:    &SessionCookie{Enable: true, Name: "test", Path: "/tea", Expires: "25s"},
			NTLM:             true,
		},
		{
			Name: "coffee-v1",
			Servers: []UpstreamServer{
				{
					Address: "10.0.0.31:8001",
				},
			},
			MaxFails:         8,
			FailTimeout:      "15s",
			MaxConns:         2,
			UpstreamZoneSize: "256k",
		},
		{
			Name: "coffee-v2",
			Servers: []UpstreamServer{
				{
					Address: "10.0.0.32:8001",
				},
			},
			MaxFails:         12,
			FailTimeout:      "20s",
			MaxConns:         4,
			UpstreamZoneSize: "256k",
		},
	},
	SplitClients: []SplitClient{
		{
			Source:   "$request_id",
			Variable: "$split_0",
			Distributions: []Distribution{
				{
					Weight: "50%",
					Value:  "@loc0",
				},
				{
					Weight: "50%",
					Value:  "@loc1",
				},
			},
		},
	},
	Maps: []Map{
		{
			Source:   "$match_0_0",
			Variable: "$match",
			Parameters: []Parameter{
				{
					Value:  "~^1",
					Result: "@match_loc_0",
				},
				{
					Value:  "default",
					Result: "@match_loc_default",
				},
			},
		},
		{
			Source:   "$http_x_version",
			Variable: "$match_0_0",
			Parameters: []Parameter{
				{
					Value:  "v2",
					Result: "1",
				},
				{
					Value:  "default",
					Result: "0",
				},
			},
		},
	},
	HTTPSnippets: []string{"# HTTP snippet"},
	Server: Server{
		ServerName:    "example.com",
		StatusZone:    "example.com",
		ProxyProtocol: true,
		SSL: &SSL{
			HTTP2:          true,
			Certificate:    "cafe-secret.pem",
			CertificateKey: "cafe-secret.pem",
		},
		TLSRedirect: &TLSRedirect{
			BasedOn: "$scheme",
			Code:    301,
		},
		ServerTokens:    "off",
		SetRealIPFrom:   []string{"0.0.0.0/0"},
		RealIPHeader:    "X-Real-IP",
		RealIPRecursive: true,
		Allow:           []string{"127.0.0.1"},
		Deny:            []string{"127.0.0.1"},
		LimitReqs: []LimitReq{
			{
				ZoneName: "pol_rl_test_test_test",
				Delay:    10,
				Burst:    5,
			},
		},
		LimitReqOptions: LimitReqOptions{
			LogLevel:   "error",
			RejectCode: 503,
		},
		JWTAuth: &JWTAuth{
			Realm:  "My Api",
			Secret: "jwk-secret",
		},
		IngressMTLS: &IngressMTLS{
			ClientCert:   "ingress-mtls-secret",
			VerifyClient: "on",
			VerifyDepth:  2,
		},
		WAF: &WAF{
			ApPolicy:            "/etc/nginx/waf/nac-policies/default-dataguard-alarm",
			ApSecurityLogEnable: true,
			ApLogConf:           "/etc/nginx/waf/nac-logconfs/default-logconf",
		},
		Snippets: []string{"# server snippet"},
		InternalRedirectLocations: []InternalRedirectLocation{
			{
				Path:        "/split",
				Destination: "@split_0",
			},
			{
				Path:        "/coffee",
				Destination: "@match",
			},
		},
		Locations: []Location{
			{
				Path:     "/",
				Snippets: []string{"# location snippet"},
				Allow:    []string{"127.0.0.1"},
				Deny:     []string{"127.0.0.1"},
				LimitReqs: []LimitReq{
					{
						ZoneName: "loc_pol_rl_test_test_test",
					},
				},
				ProxyConnectTimeout:      "30s",
				ProxyReadTimeout:         "31s",
				ProxySendTimeout:         "32s",
				ClientMaxBodySize:        "1m",
				ProxyBuffering:           true,
				ProxyBuffers:             "8 4k",
				ProxyBufferSize:          "4k",
				ProxyMaxTempFileSize:     "1024m",
				ProxyPass:                "http://test-upstream",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "5s",
				Internal:                 true,
				ProxyPassRequestHeaders:  false,
				ProxyPassHeaders:         []string{"Host"},
				ProxyPassRewrite:         "$request_uri",
				ProxyHideHeaders:         []string{"Header"},
				ProxyIgnoreHeaders:       "Cache",
				Rewrites:                 []string{"$request_uri $request_uri", "$request_uri $request_uri"},
				AddHeaders: []AddHeader{
					{
						Header: Header{
							Name:  "Header-Name",
							Value: "Header Value",
						},
						Always: true,
					},
				},
				EgressMTLS: &EgressMTLS{
					Certificate:    "egress-mtls-secret.pem",
					CertificateKey: "egress-mtls-secret.pem",
					VerifyServer:   true,
					VerifyDepth:    1,
					Ciphers:        "DEFAULT",
					Protocols:      "TLSv1.3",
					TrustedCert:    "trusted-cert.pem",
					SessionReuse:   true,
					ServerName:     true,
				},
			},
			{
				Path:                     "@loc0",
				ProxyConnectTimeout:      "30s",
				ProxyReadTimeout:         "31s",
				ProxySendTimeout:         "32s",
				ClientMaxBodySize:        "1m",
				ProxyPass:                "http://coffee-v1",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "5s",
				ProxyInterceptErrors:     true,
				ErrorPages: []ErrorPage{
					{
						Name:         "@error_page_1",
						Codes:        "400 500",
						ResponseCode: 200,
					},
					{
						Name:         "@error_page_2",
						Codes:        "500",
						ResponseCode: 0,
					},
				},
			},
			{
				Path:                     "@loc1",
				ProxyConnectTimeout:      "30s",
				ProxyReadTimeout:         "31s",
				ProxySendTimeout:         "32s",
				ClientMaxBodySize:        "1m",
				ProxyPass:                "http://coffee-v2",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "5s",
			},
			{
				Path:                     "@match_loc_0",
				ProxyConnectTimeout:      "30s",
				ProxyReadTimeout:         "31s",
				ProxySendTimeout:         "32s",
				ClientMaxBodySize:        "1m",
				ProxyPass:                "http://coffee-v2",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "5s",
			},
			{
				Path:                     "@match_loc_default",
				ProxyConnectTimeout:      "30s",
				ProxyReadTimeout:         "31s",
				ProxySendTimeout:         "32s",
				ClientMaxBodySize:        "1m",
				ProxyPass:                "http://coffee-v1",
				ProxyNextUpstream:        "error timeout",
				ProxyNextUpstreamTimeout: "5s",
			},
			{
				Path:                 "/return",
				ProxyInterceptErrors: true,
				ErrorPages: []ErrorPage{
					{
						Name:         "@return_0",
						Codes:        "418",
						ResponseCode: 200,
					},
				},
				InternalProxyPass: "http://unix:/var/lib/nginx/nginx-418-server.sock",
			},
		},
		ErrorPageLocations: []ErrorPageLocation{
			{
				Name:        "@vs_cafe_cafe_vsr_tea_tea_tea__tea_error_page_0",
				DefaultType: "application/json",
				Return: &Return{
					Code: 200,
					Text: "Hello World",
				},
				Headers: nil,
			},
			{
				Name:        "@vs_cafe_cafe_vsr_tea_tea_tea__tea_error_page_1",
				DefaultType: "",
				Return: &Return{
					Code: 200,
					Text: "Hello World",
				},
				Headers: []Header{
					{
						Name:  "Set-Cookie",
						Value: "cookie1=test",
					},
					{
						Name:  "Set-Cookie",
						Value: "cookie2=test; Secure",
					},
				},
			},
		},
		ReturnLocations: []ReturnLocation{
			{
				Name:        "@return_0",
				DefaultType: "text/html",
				Return: Return{
					Code: 200,
					Text: "Hello!",
				},
			},
		},
	},
}

var transportServerCfg = TransportServerConfig{
	Upstreams: []StreamUpstream{
		{
			Name: "udp-upstream",
			Servers: []StreamUpstreamServer{
				{
					Address: "10.0.0.20:5001",
				},
			},
		},
	},
	Match: &Match{
		Name:                "match_udp-upstream",
		Send:                `GET / HTTP/1.0\r\nHost: localhost\r\n\r\n`,
		ExpectRegexModifier: "~*",
		Expect:              "200 OK",
	},
	Server: StreamServer{
		Port:                     1234,
		UDP:                      true,
		StatusZone:               "udp-app",
		ProxyRequests:            createPointerFromInt(1),
		ProxyResponses:           createPointerFromInt(2),
		ProxyPass:                "udp-upstream",
		ProxyTimeout:             "10s",
		ProxyConnectTimeout:      "10s",
		ProxyNextUpstream:        true,
		ProxyNextUpstreamTimeout: "10s",
		ProxyNextUpstreamTries:   5,
		HealthCheck: &StreamHealthCheck{
			Enabled:  false,
			Timeout:  "5s",
			Jitter:   "0",
			Port:     8080,
			Interval: "5s",
			Passes:   1,
			Fails:    1,
			Match:    "match_udp-upstream",
		},
	},
}

func createPointerFromInt(n int) *int {
	return &n
}

func TestVirtualServerForNginxPlus(t *testing.T) {
	executor, err := NewTemplateExecutor(nginxPlusVirtualServerTmpl, nginxPlusTransportServerTmpl)
	if err != nil {
		t.Fatalf("Failed to create template executor: %v", err)
	}

	data, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfg)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	t.Log(string(data))
}

func TestVirtualServerForNginx(t *testing.T) {
	executor, err := NewTemplateExecutor(nginxVirtualServerTmpl, nginxTransportServerTmpl)
	if err != nil {
		t.Fatalf("Failed to create template executor: %v", err)
	}

	data, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfg)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	t.Log(string(data))
}

func TestTransportServerForNginxPlus(t *testing.T) {
	executor, err := NewTemplateExecutor(nginxPlusVirtualServerTmpl, nginxPlusTransportServerTmpl)
	if err != nil {
		t.Fatalf("Failed to create template executor: %v", err)
	}

	data, err := executor.ExecuteTransportServerTemplate(&transportServerCfg)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	t.Log(string(data))
}

func TestTransportServerForNginx(t *testing.T) {
	executor, err := NewTemplateExecutor(nginxVirtualServerTmpl, nginxTransportServerTmpl)
	if err != nil {
		t.Fatalf("Failed to create template executor: %v", err)
	}

	data, err := executor.ExecuteTransportServerTemplate(&transportServerCfg)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	t.Log(string(data))
}

func TestTLSPassthroughHosts(t *testing.T) {
	executor, err := NewTemplateExecutor(nginxVirtualServerTmpl, nginxTransportServerTmpl)
	if err != nil {
		t.Fatalf("Failed to create template executor: %v", err)
	}

	unixSocketsCfg := TLSPassthroughHostsConfig{
		"app.example.com": "unix:/var/lib/nginx/passthrough-default_secure-app.sock",
	}

	data, err := executor.ExecuteTLSPassthroughHostsTemplate(&unixSocketsCfg)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	t.Log(string(data))
}

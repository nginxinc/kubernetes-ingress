package version2

import "testing"

const nginxPlusVirtualServerTmpl = "nginx-plus.virtualserver.tmpl"
const nginxVirtualServerTmpl = "nginx.virtualserver.tmpl"

var virtualServerCfg = VirtualServerConfig{
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
	Server: Server{
		ServerName:    "example.com",
		StatusZone:    "example.com",
		ProxyProtocol: true,
		SSL: &SSL{
			HTTP2:           true,
			Certificate:     "cafe-secret.pem",
			CertificateKey:  "cafe-secret.pem",
			Ciphers:         "NULL",
			RedirectToHTTPS: true,
		},
		RedirectToHTTPSBasedOnXForwarderProto: true,
		ServerTokens:                          "off",
		SetRealIPFrom:                         []string{"0.0.0.0/0"},
		RealIPHeader:                          "X-Real-IP",
		RealIPRecursive:                       true,
		Snippets:                              []string{"# server snippet"},
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
				Path:                 "/",
				Snippets:             []string{"# location snippet"},
				ProxyConnectTimeout:  "30s",
				ProxyReadTimeout:     "31s",
				ProxySendTimeout:     "32s",
				ClientMaxBodySize:    "1m",
				ProxyBuffering:       true,
				ProxyBuffers:         "8 4k",
				ProxyBufferSize:      "4k",
				ProxyMaxTempFileSize: "1024m",
				ProxyPass:            "http://test-upstream",
			},
			{
				Path:                "@loc0",
				ProxyConnectTimeout: "30s",
				ProxyReadTimeout:    "31s",
				ProxySendTimeout:    "32s",
				ClientMaxBodySize:   "1m",
				ProxyPass:           "http://coffee-v1",
			},
			{
				Path:                "@loc1",
				ProxyConnectTimeout: "30s",
				ProxyReadTimeout:    "31s",
				ProxySendTimeout:    "32s",
				ClientMaxBodySize:   "1m",
				ProxyPass:           "http://coffee-v2",
			},
			{
				Path:                "@match_loc_0",
				ProxyConnectTimeout: "30s",
				ProxyReadTimeout:    "31s",
				ProxySendTimeout:    "32s",
				ClientMaxBodySize:   "1m",
				ProxyPass:           "http://coffee-v2",
			},
			{
				Path:                "@match_loc_default",
				ProxyConnectTimeout: "30s",
				ProxyReadTimeout:    "31s",
				ProxySendTimeout:    "32s",
				ClientMaxBodySize:   "1m",
				ProxyPass:           "http://coffee-v1",
			},
		},
	},
}

func TestVirtualServerForNginxPlus(t *testing.T) {
	executor, err := NewTemplateExecutor(nginxPlusVirtualServerTmpl)
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
	executor, err := NewTemplateExecutor(nginxVirtualServerTmpl)
	if err != nil {
		t.Fatalf("Failed to create template executor: %v", err)
	}

	data, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfg)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	t.Log(string(data))
}

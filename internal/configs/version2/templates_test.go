package version2

import (
	"bytes"
	"testing"
)

func createPointerFromInt(n int) *int {
	return &n
}

func newTmplExecutorNGINXPlus(t *testing.T) *TemplateExecutor {
	t.Helper()
	executor, err := NewTemplateExecutor("nginx-plus.virtualserver.tmpl", "nginx-plus.transportserver.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return executor
}

func newTmplExecutorNGINX(t *testing.T) *TemplateExecutor {
	t.Helper()
	executor, err := NewTemplateExecutor("nginx.virtualserver.tmpl", "nginx.transportserver.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return executor
}

func TestVirtualServerForNginxPlus(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	data, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfg)
	if err != nil {
		t.Errorf("Failed to execute template: %v", err)
	}
	t.Log(string(data))
}

func TestExecuteVirtualServerTemplate_RendersTemplateWithServerGunzipOn(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithGunzipOn)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Contains(got, []byte("gunzip on;")) {
		t.Error("want `gunzip on` directive, got no directive")
	}
	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersTemplateWithServerGunzipOff(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithGunzipOff)
	if err != nil {
		t.Error(err)
	}
	if bytes.Contains(got, []byte("gunzip on;")) {
		t.Error("want no directive, got `gunzip on`")
	}
	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersTemplateWithServerGunzipNotSet(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithGunzipNotSet)
	if err != nil {
		t.Error(err)
	}
	if bytes.Contains(got, []byte("gunzip on;")) {
		t.Error("want no directive, got `gunzip on` directive")
	}
	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersTemplateWithSessionCookieSameSite(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithSessionCookieSameSite)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Contains(got, []byte("samesite=strict")) {
		t.Error("want `samesite=strict` in generated template")
	}
	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersTemplateWithCustomListener(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithCustomListener)
	if err != nil {
		t.Error(err)
	}
	wantStrings := []string{
		"listen 8082",
		"listen [::]:8082",
		"listen 8443 ssl",
		"listen [::]:8443 ssl",
	}
	for _, want := range wantStrings {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("want `%s` in generated template", want)
		}
	}
	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersTemplateWithCustomListenerHTTPOnly(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithCustomListenerHTTPOnly)
	if err != nil {
		t.Error(err)
	}
	wantStrings := []string{
		"listen 8082",
		"listen [::]:8082",
	}
	unwantStrings := []string{
		"listen 8443 ssl",
		"listen [::]:8443 ssl",
	}
	for _, want := range wantStrings {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("want `%s` in generated template", want)
		}
	}
	for _, want := range unwantStrings {
		if bytes.Contains(got, []byte(want)) {
			t.Errorf("unwant  `%s` in generated template", want)
		}
	}
	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersTemplateWithCustomListenerHTTPSOnly(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithCustomListenerHTTPSOnly)
	if err != nil {
		t.Error(err)
	}
	wantStrings := []string{
		"listen 8443 ssl",
		"listen [::]:8443 ssl",
	}
	unwantStrings := []string{
		"listen 8082",
		"listen [::]:8082",
	}
	for _, want := range wantStrings {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("want `%s` in generated template", want)
		}
	}
	for _, want := range unwantStrings {
		if bytes.Contains(got, []byte(want)) {
			t.Errorf("want no `%s` in generated template", want)
		}
	}
	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersPlusTemplateWithHTTP2On(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfg)
	if err != nil {
		t.Error(err)
	}
	wantStrings := []string{
		"listen 443 ssl proxy_protocol;",
		"listen [::]:443 ssl proxy_protocol;",
		"http2 on;",
	}
	for _, want := range wantStrings {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("want `%s` in generated template", want)
		}
	}

	unwantStrings := []string{
		"listen 443 ssl http2 proxy_protocol;",
		"listen [::]:443 ssl http2 proxy_protocol;",
	}

	for _, want := range unwantStrings {
		if bytes.Contains(got, []byte(want)) {
			t.Errorf("unwant  `%s` in generated template", want)
		}
	}

	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersPlusTemplateWithHTTP2Off(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithHTTP2Off)
	if err != nil {
		t.Error(err)
	}
	wantStrings := []string{
		"listen 443 ssl proxy_protocol;",
		"listen [::]:443 ssl proxy_protocol;",
	}
	for _, want := range wantStrings {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("want `%s` in generated template", want)
		}
	}

	unwantStrings := []string{
		"http2 on;",
	}

	for _, want := range unwantStrings {
		if bytes.Contains(got, []byte(want)) {
			t.Errorf("unwant  `%s` in generated template", want)
		}
	}

	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersOSSTemplateWithHTTP2On(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINX(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfg)
	if err != nil {
		t.Error(err)
	}
	wantStrings := []string{
		"listen 443 ssl proxy_protocol;",
		"listen [::]:443 ssl proxy_protocol;",
		"http2 on;",
	}
	for _, want := range wantStrings {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("want `%s` in generated template", want)
		}
	}

	unwantStrings := []string{
		"listen 443 ssl http2 proxy_protocol;",
		"listen [::]:443 ssl http2 proxy_protocol;",
	}

	for _, want := range unwantStrings {
		if bytes.Contains(got, []byte(want)) {
			t.Errorf("unwant  `%s` in generated template", want)
		}
	}

	t.Log(string(got))
}

func TestExecuteVirtualServerTemplate_RendersOSSTemplateWithHTTP2Off(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINX(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithHTTP2Off)
	if err != nil {
		t.Error(err)
	}
	wantStrings := []string{
		"listen 443 ssl proxy_protocol;",
		"listen [::]:443 ssl proxy_protocol;",
	}
	for _, want := range wantStrings {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("want `%s` in generated template", want)
		}
	}

	unwantStrings := []string{
		"http2 on;",
	}

	for _, want := range unwantStrings {
		if bytes.Contains(got, []byte(want)) {
			t.Errorf("unwant  `%s` in generated template", want)
		}
	}

	t.Log(string(got))
}

func TestVirtualServerForNginxPlusWithWAFApBundle(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	data, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithWAFApBundle)
	if err != nil {
		t.Errorf("Failed to execute template: %v", err)
	}
	t.Log(string(data))
}

func TestVirtualServerForNginx(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINX(t)
	data, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfg)
	if err != nil {
		t.Errorf("Failed to execute template: %v", err)
	}
	t.Log(string(data))
}

func TestTransportServerForNginxPlus(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	data, err := executor.ExecuteTransportServerTemplate(&transportServerCfg)
	if err != nil {
		t.Errorf("Failed to execute template: %v", err)
	}
	t.Log(string(data))
}

func TestExecuteTemplateForTransportServerWithResolver(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	_, err := executor.ExecuteTransportServerTemplate(&transportServerCfgWithResolver)
	if err != nil {
		t.Errorf("Failed to execute template: %v", err)
	}
}

func TestTransportServerForNginx(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINX(t)
	data, err := executor.ExecuteTransportServerTemplate(&transportServerCfg)
	if err != nil {
		t.Errorf("Failed to execute template: %v", err)
	}
	t.Log(string(data))
}

func TestTLSPassthroughHosts(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINX(t)

	unixSocketsCfg := TLSPassthroughHostsConfig{
		"app.example.com": "unix:/var/lib/nginx/passthrough-default_secure-app.sock",
	}

	data, err := executor.ExecuteTLSPassthroughHostsTemplate(&unixSocketsCfg)
	if err != nil {
		t.Errorf("Failed to execute template: %v", err)
	}
	t.Log(string(data))
}

func TestExecuteVirtualServerTemplateWithJWKSWithToken(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithJWTPolicyJWKSWithToken)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Contains(got, []byte("token=$http_token")) {
		t.Error("want `token=$http_token` in generated template")
	}
	if !bytes.Contains(got, []byte("proxy_cache jwks_uri_")) {
		t.Error("want `proxy_cache` in generated template")
	}
	if !bytes.Contains(got, []byte("proxy_cache_valid 200 12h;")) {
		t.Error("want `proxy_cache_valid 200 12h;` in generated template")
	}
	t.Log(string(got))
}

func TestExecuteVirtualServerTemplateWithJWKSWithoutToken(t *testing.T) {
	t.Parallel()
	executor := newTmplExecutorNGINXPlus(t)
	got, err := executor.ExecuteVirtualServerTemplate(&virtualServerCfgWithJWTPolicyJWKSWithoutToken)
	if err != nil {
		t.Error(err)
	}
	if bytes.Contains(got, []byte("token=$http_token")) {
		t.Error("want no `token=$http_token` string in generated template")
	}
	if !bytes.Contains(got, []byte("proxy_cache jwks_uri_")) {
		t.Error("want `proxy_cache` in generated template")
	}
	if !bytes.Contains(got, []byte("proxy_cache_valid 200 12h;")) {
		t.Error("want `proxy_cache_valid 200 12h;` in generated template")
	}
	t.Log(string(got))
}

var (
	virtualServerCfg = VirtualServerConfig{
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
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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

	virtualServerCfgWithHTTP2Off = VirtualServerConfig{
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
				HTTP2:          false,
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
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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

	virtualServerCfgWithGunzipOn = VirtualServerConfig{
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
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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
			Gunzip: true,
		},
	}

	virtualServerCfgWithGunzipOff = VirtualServerConfig{
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
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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
			Gunzip: false,
		},
	}

	virtualServerCfgWithGunzipNotSet = VirtualServerConfig{
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
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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

	virtualServerCfgWithWAFApBundle = VirtualServerConfig{
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
				ApBundle:            "/etc/nginx/waf/bundles/NginxDefaultPolicy.tgz",
				ApSecurityLogEnable: true,
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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

	virtualServerCfgWithSessionCookieSameSite = VirtualServerConfig{
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
				// SessionCookie set for test:
				SessionCookie: &SessionCookie{
					Enable:   true,
					Name:     "test",
					Path:     "/tea",
					Expires:  "25s",
					SameSite: "STRICT",
				},
				NTLM: true,
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
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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

	// VirtualServer Config data for JWT Policy tests

	virtualServerCfgWithJWTPolicyJWKSWithToken = VirtualServerConfig{
		Upstreams: []Upstream{
			{
				UpstreamLabels: UpstreamLabels{
					Service:           "tea-svc",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_tea",
				Servers: []UpstreamServer{
					{
						Address: "10.0.0.20:80",
					},
				},
				Keepalive: 16,
			},
			{
				UpstreamLabels: UpstreamLabels{
					Service:           "coffee-svc",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_coffee",
				Servers: []UpstreamServer{
					{
						Address: "10.0.0.30:80",
					},
				},
				Keepalive: 16,
			},
		},
		HTTPSnippets:  []string{},
		LimitReqZones: []LimitReqZone{},
		Server: Server{
			JWTAuthList: map[string]*JWTAuth{
				"default/jwt-policy": {
					Key:      "default/jwt-policy",
					Realm:    "Spec Realm API",
					Token:    "$http_token",
					KeyCache: "1h",
					JwksURI: JwksURI{
						JwksScheme: "https",
						JwksHost:   "idp.spec.example.com",
						JwksPort:   "443",
						JwksPath:   "/spec-keys",
					},
				},
				"default/jwt-policy-route": {
					Key:      "default/jwt-policy-route",
					Realm:    "Route Realm API",
					Token:    "$http_token",
					KeyCache: "1h",
					JwksURI: JwksURI{
						JwksScheme: "http",
						JwksHost:   "idp.route.example.com",
						JwksPort:   "80",
						JwksPath:   "/route-keys",
					},
				},
			},
			JWTAuth: &JWTAuth{
				Key:      "default/jwt-policy",
				Realm:    "Spec Realm API",
				Token:    "$http_token",
				KeyCache: "1h",
				JwksURI: JwksURI{
					JwksScheme: "https",
					JwksHost:   "idp.spec.example.com",
					JwksPort:   "443",
					JwksPath:   "/spec-keys",
				},
			},
			JWKSAuthEnabled: true,
			ServerName:      "cafe.example.com",
			StatusZone:      "cafe.example.com",
			ProxyProtocol:   true,
			ServerTokens:    "off",
			RealIPHeader:    "X-Real-IP",
			SetRealIPFrom:   []string{"0.0.0.0/0"},
			RealIPRecursive: true,
			Snippets:        []string{"# server snippet"},
			TLSPassthrough:  true,
			VSNamespace:     "default",
			VSName:          "cafe",
			Locations: []Location{
				{
					Path:                     "/tea",
					ProxyPass:                "http://vs_default_cafe_tea",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxySSLName:             "tea-svc.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "tea-svc",
					JWTAuth: &JWTAuth{
						Key:      "default/jwt-policy-route",
						Realm:    "Route Realm API",
						Token:    "$http_token",
						KeyCache: "1h",
						JwksURI: JwksURI{
							JwksScheme: "http",
							JwksHost:   "idp.route.example.com",
							JwksPort:   "80",
							JwksPath:   "/route-keys",
						},
					},
				},
				{
					Path:                     "/coffee",
					ProxyPass:                "http://vs_default_cafe_coffee",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxySSLName:             "coffee-svc.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "coffee-svc",
					JWTAuth: &JWTAuth{
						Key:      "default/jwt-policy-route",
						Realm:    "Route Realm API",
						Token:    "$http_token",
						KeyCache: "1h",
						JwksURI: JwksURI{
							JwksScheme: "http",
							JwksHost:   "idp.route.example.com",
							JwksPort:   "80",
							JwksPath:   "/route-keys",
						},
					},
				},
			},
		},
	}

	virtualServerCfgWithJWTPolicyJWKSWithoutToken = VirtualServerConfig{
		Upstreams: []Upstream{
			{
				UpstreamLabels: UpstreamLabels{
					Service:           "tea-svc",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_tea",
				Servers: []UpstreamServer{
					{
						Address: "10.0.0.20:80",
					},
				},
				Keepalive: 16,
			},
			{
				UpstreamLabels: UpstreamLabels{
					Service:           "coffee-svc",
					ResourceType:      "virtualserver",
					ResourceName:      "cafe",
					ResourceNamespace: "default",
				},
				Name: "vs_default_cafe_coffee",
				Servers: []UpstreamServer{
					{
						Address: "10.0.0.30:80",
					},
				},
				Keepalive: 16,
			},
		},
		HTTPSnippets:  []string{},
		LimitReqZones: []LimitReqZone{},
		Server: Server{
			JWTAuthList: map[string]*JWTAuth{
				"default/jwt-policy": {
					Key:      "default/jwt-policy",
					Realm:    "Spec Realm API",
					KeyCache: "1h",
					JwksURI: JwksURI{
						JwksScheme: "https",
						JwksHost:   "idp.spec.example.com",
						JwksPort:   "443",
						JwksPath:   "/spec-keys",
					},
				},
				"default/jwt-policy-route": {
					Key:      "default/jwt-policy-route",
					Realm:    "Route Realm API",
					KeyCache: "1h",
					JwksURI: JwksURI{
						JwksScheme: "http",
						JwksHost:   "idp.route.example.com",
						JwksPort:   "80",
						JwksPath:   "/route-keys",
					},
				},
			},
			JWTAuth: &JWTAuth{
				Key:      "default/jwt-policy",
				Realm:    "Spec Realm API",
				KeyCache: "1h",
				JwksURI: JwksURI{
					JwksScheme: "https",
					JwksHost:   "idp.spec.example.com",
					JwksPort:   "443",
					JwksPath:   "/spec-keys",
				},
			},
			JWKSAuthEnabled: true,
			ServerName:      "cafe.example.com",
			StatusZone:      "cafe.example.com",
			ProxyProtocol:   true,
			ServerTokens:    "off",
			RealIPHeader:    "X-Real-IP",
			SetRealIPFrom:   []string{"0.0.0.0/0"},
			RealIPRecursive: true,
			Snippets:        []string{"# server snippet"},
			TLSPassthrough:  true,
			VSNamespace:     "default",
			VSName:          "cafe",
			Locations: []Location{
				{
					Path:                     "/tea",
					ProxyPass:                "http://vs_default_cafe_tea",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxySSLName:             "tea-svc.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "tea-svc",
					JWTAuth: &JWTAuth{
						Key:      "default/jwt-policy-route",
						Realm:    "Route Realm API",
						KeyCache: "1h",
						JwksURI: JwksURI{
							JwksScheme: "http",
							JwksHost:   "idp.route.example.com",
							JwksPort:   "80",
							JwksPath:   "/route-keys",
						},
					},
				},
				{
					Path:                     "/coffee",
					ProxyPass:                "http://vs_default_cafe_coffee",
					ProxyNextUpstream:        "error timeout",
					ProxyNextUpstreamTimeout: "0s",
					ProxyNextUpstreamTries:   0,
					HasKeepalive:             true,
					ProxySSLName:             "coffee-svc.default.svc",
					ProxyPassRequestHeaders:  true,
					ProxySetHeaders:          []Header{{Name: "Host", Value: "$host"}},
					ServiceName:              "coffee-svc",
					JWTAuth: &JWTAuth{
						Key:      "default/jwt-policy-route",
						Realm:    "Route Realm API",
						KeyCache: "1h",
						JwksURI: JwksURI{
							JwksScheme: "http",
							JwksHost:   "idp.route.example.com",
							JwksPort:   "80",
							JwksPath:   "/route-keys",
						},
					},
				},
			},
		},
	}

	virtualServerCfgWithCustomListener = VirtualServerConfig{
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
			CustomListeners: true,
			HTTPPort:        8082,
			HTTPSPort:       8443,
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
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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

	virtualServerCfgWithCustomListenerHTTPOnly = VirtualServerConfig{
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
			CustomListeners: true,
			HTTPPort:        8082,
			HTTPSPort:       0,
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
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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

	virtualServerCfgWithCustomListenerHTTPSOnly = VirtualServerConfig{
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
			CustomListeners: true,
			HTTPPort:        0,
			HTTPSPort:       8443,
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
				ApLogConf:           []string{"/etc/nginx/waf/nac-logconfs/default-logconf"},
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
			HealthChecks: []HealthCheck{
				{
					Name:       "coffee",
					URI:        "/",
					Interval:   "5s",
					Jitter:     "0s",
					Fails:      1,
					Passes:     1,
					Port:       50,
					ProxyPass:  "http://coffee-v2",
					Mandatory:  true,
					Persistent: true,
				},
				{
					Name:        "tea",
					Interval:    "5s",
					Jitter:      "0s",
					Fails:       1,
					Passes:      1,
					Port:        50,
					ProxyPass:   "http://tea-v2",
					GRPCPass:    "grpc://tea-v3",
					GRPCStatus:  createPointerFromInt(12),
					GRPCService: "tea-servicev2",
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
					Path:                "@loc2",
					ProxyConnectTimeout: "30s",
					ProxyReadTimeout:    "31s",
					ProxySendTimeout:    "32s",
					ClientMaxBodySize:   "1m",
					ProxyPass:           "http://coffee-v2",
					GRPCPass:            "grpc://coffee-v3",
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

	transportServerCfg = TransportServerConfig{
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

	transportServerCfgWithResolver = TransportServerConfig{
		Upstreams: []StreamUpstream{
			{
				Name: "udp-upstream",
				Servers: []StreamUpstreamServer{
					{
						Address: "10.0.0.20:5001",
					},
				},
				Resolve: true,
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
)

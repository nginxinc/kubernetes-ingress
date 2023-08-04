package version1

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

func TestIngressForNGINXPlus(t *testing.T) {
	t.Parallel()
	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}
	err := tmpl.Execute(buf, ingressCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func TestIngressForNGINX(t *testing.T) {
	t.Parallel()
	tmpl := newNGINXIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithRegExAnnotationCaseSensitive(t *testing.T) {
	t.Parallel()
	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgWithRegExAnnotationCaseSensitive)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantLocation := "~ \"^/tea/[A-Z0-9]{3}\""
	if !strings.Contains(buf.String(), wantLocation) {
		t.Errorf("want %q in generated config", wantLocation)
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithRegExAnnotationCaseInsensitive(t *testing.T) {
	t.Parallel()
	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgWithRegExAnnotationCaseInsensitive)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantLocation := "~* \"^/tea/[A-Z0-9]{3}\""
	if !strings.Contains(buf.String(), wantLocation) {
		t.Errorf("want %q in generated config", wantLocation)
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithRegExAnnotationExactMatch(t *testing.T) {
	t.Parallel()
	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgWithRegExAnnotationExactMatch)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantLocation := "= \"/tea\""
	if !strings.Contains(buf.String(), wantLocation) {
		t.Errorf("want %q in generated config", wantLocation)
	}
}

func TestExecuteTemplate_ForIngressForNGINXPlusWithRegExAnnotationEmpty(t *testing.T) {
	t.Parallel()
	tmpl := newNGINXPlusIngressTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, ingressCfgWithRegExAnnotationEmptyString)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	wantLocation := "/tea"
	if !strings.Contains(buf.String(), wantLocation) {
		t.Errorf("want %q in generated config", wantLocation)
	}
}

func TestMainForNGINXPlus(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXPlusMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfg)
	if err != nil {
		t.Errorf("Failed to write template %v", err)
	}
	t.Log(buf.String())
}

func TestMainForNGINX(t *testing.T) {
	t.Parallel()

	tmpl := newNGINXMainTmpl(t)
	buf := &bytes.Buffer{}

	err := tmpl.Execute(buf, mainCfg)
	if err != nil {
		t.Errorf("Failed to write template %v", err)
	}
	t.Log(buf.String())
}

func newNGINXPlusIngressTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("nginx-plus.ingress.tmpl").Funcs(helperFunctions).ParseFiles("nginx-plus.ingress.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func newNGINXIngressTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("nginx.ingress.tmpl").Funcs(helperFunctions).ParseFiles("nginx.ingress.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func newNGINXPlusMainTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("nginx-plus.tmpl").ParseFiles("nginx-plus.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func newNGINXMainTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.New("nginx.tmpl").ParseFiles("nginx.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}

var (
	// Ingress Config example without added annotations
	ingressCfg = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:      "cafe-ingress",
			Namespace: "default",
		},
	}

	// Ingress Config example with path-regex annotation value "case_sensitive"
	ingressCfgWithRegExAnnotationCaseSensitive = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea/[A-Z0-9]{3}",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:        "cafe-ingress",
			Namespace:   "default",
			Annotations: map[string]string{"nginx.org/path-regex": "case_sensitive"},
		},
	}

	// Ingress Config example with path-regex annotation value "case_insensitive"
	ingressCfgWithRegExAnnotationCaseInsensitive = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea/[A-Z0-9]{3}",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:        "cafe-ingress",
			Namespace:   "default",
			Annotations: map[string]string{"nginx.org/path-regex": "case_insensitive"},
		},
	}

	// Ingress Config example with path-regex annotation value "exact"
	ingressCfgWithRegExAnnotationExactMatch = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:        "cafe-ingress",
			Namespace:   "default",
			Annotations: map[string]string{"nginx.org/path-regex": "exact"},
		},
	}

	// Ingress Config example with path-regex annotation value of an empty string
	ingressCfgWithRegExAnnotationEmptyString = IngressNginxConfig{
		Servers: []Server{
			{
				Name:         "test.example.com",
				ServerTokens: "off",
				StatusZone:   "test.example.com",
				JWTAuth: &JWTAuth{
					Key:                  "/etc/nginx/secrets/key.jwk",
					Realm:                "closed site",
					Token:                "$cookie_auth_token",
					RedirectLocationName: "@login_url-default-cafe-ingress",
				},
				SSL:               true,
				SSLCertificate:    "secret.pem",
				SSLCertificateKey: "secret.pem",
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				Locations: []Location{
					{
						Path:                "/tea",
						Upstream:            testUpstream,
						ProxyConnectTimeout: "10s",
						ProxyReadTimeout:    "10s",
						ProxySendTimeout:    "10s",
						ClientMaxBodySize:   "2m",
						JWTAuth: &JWTAuth{
							Key:   "/etc/nginx/secrets/location-key.jwk",
							Realm: "closed site",
							Token: "$cookie_auth_token",
						},
						MinionIngress: &Ingress{
							Name:      "tea-minion",
							Namespace: "default",
						},
					},
				},
				HealthChecks: map[string]HealthCheck{"test": healthCheck},
				JWTRedirectLocations: []JWTRedirectLocation{
					{
						Name:     "@login_url-default-cafe-ingress",
						LoginURL: "https://test.example.com/login",
					},
				},
			},
		},
		Upstreams: []Upstream{testUpstream},
		Keepalive: "16",
		Ingress: Ingress{
			Name:        "cafe-ingress",
			Namespace:   "default",
			Annotations: map[string]string{"nginx.org/path-regex": ""},
		},
	}

	mainCfg = MainConfig{
		ServerNamesHashMaxSize:  "512",
		ServerTokens:            "off",
		WorkerProcesses:         "auto",
		WorkerCPUAffinity:       "auto",
		WorkerShutdownTimeout:   "1m",
		WorkerConnections:       "1024",
		WorkerRlimitNofile:      "65536",
		LogFormat:               []string{"$remote_addr", "$remote_user"},
		LogFormatEscaping:       "default",
		StreamSnippets:          []string{"# comment"},
		StreamLogFormat:         []string{"$remote_addr", "$remote_user"},
		StreamLogFormatEscaping: "none",
		ResolverAddresses:       []string{"example.com", "127.0.0.1"},
		ResolverIPV6:            false,
		ResolverValid:           "10s",
		ResolverTimeout:         "15s",
		KeepaliveTimeout:        "65s",
		KeepaliveRequests:       100,
		VariablesHashBucketSize: 256,
		VariablesHashMaxSize:    1024,
		TLSPassthrough:          true,
	}
)

var testUpstream = Upstream{
	Name:             "test",
	UpstreamZoneSize: "256k",
	UpstreamServers: []UpstreamServer{
		{
			Address:     "127.0.0.1:8181",
			MaxFails:    0,
			MaxConns:    0,
			FailTimeout: "1s",
			SlowStart:   "5s",
		},
	},
}

var (
	headers     = map[string]string{"Test-Header": "test-header-value"}
	healthCheck = HealthCheck{
		UpstreamName: "test",
		Fails:        1,
		Interval:     1,
		Passes:       1,
		Headers:      headers,
	}
)

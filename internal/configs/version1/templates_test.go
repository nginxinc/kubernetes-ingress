package version1

import (
	"bytes"
	"testing"
	"text/template"
)

const (
	nginxIngressTmpl     = "nginx.ingress.tmpl"
	nginxMainTmpl        = "nginx.tmpl"
	nginxPlusIngressTmpl = "nginx-plus.ingress.tmpl"
	nginxPlusMainTmpl    = "nginx-plus.tmpl"
)

var testUps = Upstream{
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

var ingCfg = IngressNginxConfig{

	Servers: []Server{
		{
			Name:         "test.example.com",
			ServerTokens: "off",
			StatusZone:   "test.example.com",
			JWTAuth: &JWTAuth{
				Key:                  "/etc/nginx/secrets/key.jwk",
				Realm:                "closed site",
				Token:                "$cookie_auth_token",
				RedirectLocationName: "@login_url-default-cafe-ingres",
			},
			SSL:               true,
			SSLCertificate:    "secret.pem",
			SSLCertificateKey: "secret.pem",
			SSLPorts:          []int{443},
			SSLRedirect:       true,
			Locations: []Location{
				{
					Path:                "/tea",
					Upstream:            testUps,
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
	Upstreams: []Upstream{testUps},
	Keepalive: "16",
	Ingress: Ingress{
		Name:      "cafe-ingress",
		Namespace: "default",
	},
}

var mainCfg = MainConfig{
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

func TestIngressForNGINXPlus(t *testing.T) {
	t.Parallel()
	tmpl, err := template.New(nginxPlusIngressTmpl).Funcs(helperFunctions).ParseFiles(nginxPlusIngressTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, ingCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func TestIngressForNGINX(t *testing.T) {
	t.Parallel()
	tmpl, err := template.New(nginxIngressTmpl).Funcs(helperFunctions).ParseFiles(nginxIngressTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, ingCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func TestMainForNGINXPlus(t *testing.T) {
	t.Parallel()
	tmpl, err := template.New(nginxPlusMainTmpl).ParseFiles(nginxPlusMainTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, mainCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func TestMainForNGINX(t *testing.T) {
	t.Parallel()
	tmpl, err := template.New(nginxMainTmpl).ParseFiles(nginxMainTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, mainCfg)
	t.Log(buf.String())
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func TestSplitHelperFunction(t *testing.T) {
	t.Parallel()
	const tpl = `{{range $n := split . ","}}{{$n}} {{end}}`

	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(tpl)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer

	input := "foo,bar"
	expected := "foo bar "

	err = tmpl.Execute(&buf, input)
	if err != nil {
		t.Fatalf("Failed to execute the template %v", err)
	}

	if buf.String() != expected {
		t.Fatalf("Template generated wrong config, got %v but expected %v.", buf.String(), expected)
	}
}

func TestTrimHelperFunction(t *testing.T) {
	t.Parallel()
	const tpl = `{{trim .}}`

	tmpl, err := template.New("testTemplate").Funcs(helperFunctions).Parse(tpl)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer

	input := "  foobar     "
	expected := "foobar"

	err = tmpl.Execute(&buf, input)
	if err != nil {
		t.Fatalf("Failed to execute the template %v", err)
	}

	if buf.String() != expected {
		t.Fatalf("Template generated wrong config, got %v but expected %v.", buf.String(), expected)
	}
}

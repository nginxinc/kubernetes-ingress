package templates

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
)

const nginxIngressTmpl = "nginx.ingress.tmpl"
const nginxMainTmpl = "nginx.tmpl"
const nginxPlusIngressTmpl = "nginx-plus.ingress.tmpl"
const nginxPlusMainTmpl = "nginx-plus.tmpl"

var testUps = nginx.Upstream{
	Name: "test",
	UpstreamServers: []nginx.UpstreamServer{
		{
			Address:     "127.0.0.1",
			Port:        "8181",
			MaxFails:    0,
			FailTimeout: "1s",
			SlowStart:   "5s",
		},
	},
}

var headers = map[string]string{"Test-Header": "test-header-value"}
var healthCheck = nginx.HealthCheck{
	UpstreamName: "test",
	Fails:        1,
	Interval:     1,
	Passes:       1,
	Headers:      headers,
}

var ingCfg = nginx.IngressNginxConfig{

	Servers: []nginx.Server{
		{
			Name:         "test.example.com",
			ServerTokens: "off",
			StatusZone:   "test.example.com",
			JWTAuth: &nginx.JWTAuth{
				Key:                  "/etc/nginx/secrets/key.jwk",
				Realm:                "closed site",
				Token:                "$cookie_auth_token",
				RedirectLocationName: "@login_url-default-cafe-ingres",
			},
			SSL:               true,
			SSLCertificate:    "secret.pem",
			SSLCertificateKey: "secret.pem",
			SSLCiphers:        "NULL",
			SSLPorts:          []int{443},
			SSLRedirect:       true,
			Locations: []nginx.Location{
				{
					Path:                "/tea",
					Upstream:            testUps,
					ProxyConnectTimeout: "10s",
					ProxyReadTimeout:    "10s",
					ClientMaxBodySize:   "2m",
					JWTAuth: &nginx.JWTAuth{
						Key:   "/etc/nginx/secrets/location-key.jwk",
						Realm: "closed site",
						Token: "$cookie_auth_token",
					},
					MinionIngress: &nginx.Ingress{
						Name:      "tea-minion",
						Namespace: "default",
					},
				},
			},
			HealthChecks: map[string]nginx.HealthCheck{"test": healthCheck},
			JWTRedirectLocations: []nginx.JWTRedirectLocation{
				{
					Name:     "@login_url-default-cafe-ingress",
					LoginURL: "https://test.example.com/login",
				},
			},
		},
	},
	Upstreams: []nginx.Upstream{testUps},
	Keepalive: "16",
	Ingress: nginx.Ingress{
		Name:      "cafe-ingress",
		Namespace: "default",
	},
}

var mainCfg = nginx.MainConfig{
	ServerNamesHashMaxSize: "512",
	ServerTokens:           "off",
	WorkerProcesses:        "auto",
	WorkerCPUAffinity:      "auto",
	WorkerShutdownTimeout:  "1m",
	WorkerConnections:      "1024",
	WorkerRlimitNofile:     "65536",
	StreamSnippets:         []string{"# comment"},
	StreamLogFormat:        "$remote_addr",
}

func TestIngressForNGINXPlus(t *testing.T) {
	tmpl, err := template.New(nginxPlusIngressTmpl).ParseFiles(nginxPlusIngressTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, ingCfg)
	t.Log(string(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func TestIngressForNGINX(t *testing.T) {
	tmpl, err := template.New(nginxIngressTmpl).ParseFiles(nginxIngressTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, ingCfg)
	t.Log(string(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func TestMainForNGINXPlus(t *testing.T) {
	tmpl, err := template.New(nginxPlusMainTmpl).ParseFiles(nginxPlusMainTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, mainCfg)
	t.Log(string(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func TestMainForNGINX(t *testing.T) {
	tmpl, err := template.New(nginxMainTmpl).ParseFiles(nginxMainTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, mainCfg)
	t.Log(string(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

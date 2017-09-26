package templates

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/isserrano/kubernetes-ingress/nginx-controller/nginx"
)

const nginxIngressTmpl = "nginx.ingress.tmpl"
const nginxPlusIngressTmpl = "nginx-plus.ingress.tmpl"

var testUps = nginx.Upstream{
	Name: "test",
	UpstreamServers: []nginx.UpstreamServer{
		{"127.0.0.1", "8181"},
	},
}

var ingCfg = nginx.IngressNginxConfig{

	Servers: []nginx.Server{
		nginx.Server{
			Name:         "test.example.com",
			ServerTokens: "off",
			StatusZone:   "test.example.com",
			JWTKey:       "/etc/nginx/secrets/key.jwk",
			JWTRealm:     "closed site",
			JWTToken:     "$cookie_auth_token",
			JWTLoginURL:  "https://test.example.com/login",
			Locations: []nginx.Location{
				nginx.Location{
					Path:                "/",
					Upstream:            testUps,
					ProxyConnectTimeout: "10s",
					ProxyReadTimeout:    "10s",
					ClientMaxBodySize:   "2m",
				},
			},
		},
	},
	Upstreams: []nginx.Upstream{testUps},
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

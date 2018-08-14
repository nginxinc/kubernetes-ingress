package templates

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/nginxinc/kubernetes-ingress/nginx-controller/nginx"
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
		nginx.Server{
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
			SSLPorts:          []int{443},
			SSLRedirect:       true,
			Locations: []nginx.Location{
				nginx.Location{
					Path:                "/",
					Upstream:            testUps,
					ProxyConnectTimeout: "10s",
					ProxyReadTimeout:    "10s",
					ClientMaxBodySize:   "2m",
					JWTAuth: &nginx.JWTAuth{
						Key:   "/etc/nginx/secrets/location-key.jwk",
						Realm: "closed site",
						Token: "$cookie_auth_token",
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
}

func generateNginxMainConfigVars() nginx.NginxMainConfig {
    mainCfg := nginx.NginxMainConfig{
    	ServerNamesHashMaxSize: "512",
    	ServerTokens:           "off",
    	WorkerProcesses:        "auto",
    	WorkerCPUAffinity:      "auto",
    	WorkerShutdownTimeout:  "1m",
    	WorkerConnections:      "1024",
    	WorkerRlimitNofile:     "65536",
    }
    return mainCfg;
}

func disableHealthStatusAndStubStatus(cfg nginx.NginxMainConfig) nginx.NginxMainConfig {
    cfg.HealthStatus = false
    cfg.StubStatus = false
    return cfg;
}
func enableHealthStatusAndStubStatus(cfg nginx.NginxMainConfig) nginx.NginxMainConfig  {
    cfg.HealthStatus = true
    cfg.StubStatus = true
    return cfg;
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

    mainCfg := generateNginxMainConfigVars()

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

    mainCfg := generateNginxMainConfigVars()

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, mainCfg)
	t.Log(string(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}
}

func loadNginxMainTemplate(t *testing.T) *template.Template {
	tmpl, err := template.New(nginxMainTmpl).ParseFiles(nginxMainTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}
	return tmpl;
}

func loadNginxPlusMainTemplate(t *testing.T) *template.Template {
	tmpl, err := template.New(nginxPlusMainTmpl).ParseFiles(nginxPlusMainTmpl)
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}
	return tmpl;
}

func TestNoStatusAllowIpForNGINXPlus(t *testing.T) {
    tmpl := loadNginxPlusMainTemplate(t);
    mainCfg := generateNginxMainConfigVars()

    mainCfg = enableHealthStatusAndStubStatus(mainCfg);

	var buf bytes.Buffer

	err := tmpl.Execute(&buf, mainCfg)
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	generatedConfig := buf.String()

	if strings.Count(generatedConfig, "location /api") == 0 {
		t.Errorf("Generated Nginx main template returned no location /api directives, expected one.")
	}

	if strings.Count(generatedConfig, "location /api") > 1 {
		t.Errorf("Generated Nginx main template returned more than one location /api directives, expected one.")
	}

	if !strings.Contains(generatedConfig, `location /nginx-health {


            default_type text/plain;
            return 200 "healthy\n";
        }`) {
        t.Log(string(buf.Bytes()))
		t.Errorf("Generated Nginx main template was supposed to contain unrestricted location /nginx-health.")
	}

	if strings.Contains(generatedConfig, `location /nginx-health {
            access_log off;
            allow 1.2.3.4;
            deny all;

            default_type text/plain;
            return 200 "healthy\n";
        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template was not supposed to contain restricted location /nginx-health.")
	}
}

func TestDefaultStatusAllowIpForNGINXPlus(t *testing.T) {
    tmpl := loadNginxPlusMainTemplate(t);
    mainCfg := generateNginxMainConfigVars()

    mainCfg.StatusAllowIp = ""
    mainCfg = enableHealthStatusAndStubStatus(mainCfg);


	var buf bytes.Buffer

	err := tmpl.Execute(&buf, mainCfg)
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	generatedConfig := buf.String()

	if strings.Count(generatedConfig, "location /api") == 0 {
		t.Errorf("Generated Nginx main template returned no location /api directives, expected one.")
	}

	if strings.Count(generatedConfig, "location /api") > 1 {
		t.Errorf("Generated Nginx main template returned more than one location /api directives, expected one.")
	}

	if strings.Contains(generatedConfig, `location /nginx-health {
            access_log off;
            allow 1.2.3.4;
            deny all;

            default_type text/plain;
            return 200 "healthy\n";
        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template was not supposed to contain the restricted location /nginx-health.")
	}
}

func TestGivenStatusAllowIpForNGINXPlus(t *testing.T) {
    tmpl := loadNginxPlusMainTemplate(t);
    mainCfg := generateNginxMainConfigVars()

    mainCfg = enableHealthStatusAndStubStatus(mainCfg);
    mainCfg.StatusAllowIp = "1.2.3.4"

	var buf bytes.Buffer

	err := tmpl.Execute(&buf, mainCfg)
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	generatedConfig := buf.String()

	if !strings.Contains(generatedConfig, "location /nginx-health") {
		t.Errorf("Generated Nginx main template did not contain a location /nginx-health.")
	}

	if !strings.Contains(generatedConfig, `location /nginx-health {
            access_log off;
            allow 1.2.3.4;
            deny all;

            default_type text/plain;
            return 200 "healthy\n";
        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template was supposed to contain the restricted location /nginx-health.")
	}



	if strings.Contains(generatedConfig, "location /stub_status") {
		t.Errorf("Generated Nginx main template should not contain a location /stub_status directive.")
	}

	if strings.Count(generatedConfig, "location /api") == 0 {
		t.Errorf("Generated Nginx main template returned no location /api directives, expected one.")
	}

	if strings.Count(generatedConfig, "location /api") > 1 {
		t.Errorf("Generated Nginx main template returned more than one location /api directives, expected one.")
	}
}


func TestNoStatusAllowIpForNGINX(t *testing.T) {
    tmpl := loadNginxMainTemplate(t);
    mainCfg := generateNginxMainConfigVars()

    mainCfg = enableHealthStatusAndStubStatus(mainCfg);

	var buf bytes.Buffer

	err := tmpl.Execute(&buf, mainCfg)
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	generatedConfig := buf.String()

	if !strings.Contains(generatedConfig, "location /nginx-health") {
		t.Errorf("Generated Nginx main template did not contain a location /nginx-health.")
	}
	if !strings.Contains(generatedConfig, "location /stub_status") {
		t.Errorf("Generated Nginx main template did not contain a location /stub_status directive.")
	}

	if strings.Contains(generatedConfig, `location /stub_status {
            stub_status;

            access_log off;
            allow 1.2.3.4;
            deny all;
        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template was not supposed to contain the restricted location /stub_status directive.")
	}

	if !strings.Contains(generatedConfig, `location /stub_status {
            stub_status;


        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template did not contain an opened location /stub_status directive.")
	}
}

func TestDefaultStatusAllowIpForNGINX(t *testing.T) {
    tmpl := loadNginxMainTemplate(t);
    mainCfg := generateNginxMainConfigVars()

    mainCfg = enableHealthStatusAndStubStatus(mainCfg);
    mainCfg.StatusAllowIp = ""

	var buf bytes.Buffer

	err := tmpl.Execute(&buf, mainCfg)
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	generatedConfig := buf.String()

	if !strings.Contains(generatedConfig, "location /nginx-health") {
		t.Errorf("Generated Nginx main template did not contain a location /nginx-health.")
	}

	if !strings.Contains(generatedConfig, "location /stub_status") {
		t.Errorf("Generated Nginx main template did not contain a location /stub_status directive.")
	}

	if strings.Contains(generatedConfig, `location /stub_status {
            stub_status;

            access_log off;
            allow 1.2.3.4;
            deny all;
        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template was not supposed to contain the restricted location /stub_status directive.")
	}

	if !strings.Contains(generatedConfig, `location /stub_status {
            stub_status;


        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template did not contain an opened location /stub_status directive.")
	}
}

func TestGivenStatusWithoutAllowIpForNGINX(t *testing.T) {
    tmpl := loadNginxMainTemplate(t);
    mainCfg := generateNginxMainConfigVars()

    mainCfg = enableHealthStatusAndStubStatus(mainCfg);
    mainCfg.StatusAllowIp = ""

	var buf bytes.Buffer

	err := tmpl.Execute(&buf, mainCfg)
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	generatedConfig := buf.String()

	if !strings.Contains(generatedConfig, "location /nginx-health") {
		t.Errorf("Generated Nginx main template did not contain a location /nginx-health.")
	}
	if !strings.Contains(generatedConfig, "location /stub_status") {
		t.Errorf("Generated Nginx main template did not contain a location /stub_status directive.")
	}

	if strings.Contains(generatedConfig, `location /stub_status {
            stub_status;

            access_log off;
            allow 1.2.3.4;
            deny all;
        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template was not supposed to contain the restricted location /stub_status directive.")
	}

	if !strings.Contains(generatedConfig, `location /stub_status {
            stub_status;


        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template did not contain an opened location /stub_status directive.")
	}
}

func TestGivenStatusAllowIpForNGINX(t *testing.T) {
    tmpl := loadNginxMainTemplate(t);
    mainCfg := generateNginxMainConfigVars()

    mainCfg = enableHealthStatusAndStubStatus(mainCfg);
    mainCfg.StatusAllowIp = "1.2.3.4"

	var buf bytes.Buffer

	err := tmpl.Execute(&buf, mainCfg)
	if err != nil {
		t.Fatalf("Failed to write template %v", err)
	}

	generatedConfig := buf.String()

	if !strings.Contains(generatedConfig, "location /nginx-health") {
		t.Errorf("Generated Nginx main template did not contain a location /nginx-health.")
	}
	if !strings.Contains(generatedConfig, "location /stub_status") {
		t.Errorf("Generated Nginx main template did not contain a location /stub_status directive.")
	}

	if !strings.Contains(generatedConfig, `location /stub_status {
            stub_status;

            access_log off;
            allow 1.2.3.4;
            deny all;
        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template did not contain the restricted location /stub_status directive.2")
	}

	if strings.Contains(generatedConfig, `location /stub_status {
            stub_status;


        }`) {
        t.Log(generatedConfig)
		t.Errorf("Generated Nginx main template was not supposed to contain an opened location /stub_status directive.")
	}
}

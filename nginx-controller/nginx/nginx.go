package nginx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"text/template"

	"github.com/golang/glog"
)

const dhparamFilename = "dhparam.pem"

// NginxController Updates NGINX configuration, starts and reloads NGINX
type NginxController struct {
	nginxConfdPath string
	nginxCertsPath string
	local          bool
}

// IngressNginxConfig describes an NGINX configuration
type IngressNginxConfig struct {
	Upstreams []Upstream
	Servers   []Server
}

// Upstream describes an NGINX upstream
type Upstream struct {
	Name            string
	UpstreamServers []UpstreamServer
}

// UpstreamServer describes a server in an NGINX upstream
type UpstreamServer struct {
	Address string
	Port    string
}

// Server describes an NGINX server
type Server struct {
	Name                  string
	Locations             []Location
	SSL                   bool
	SSLCertificate        string
	SSLCertificateKey     string
	HTTP2                 bool
	ProxyProtocol         bool
	HSTS                  bool
	HSTSMaxAge            int64
	HSTSIncludeSubdomains bool
	ProxyIgnoreHeaders    []string
	ProxyHideHeaders      []string
	ProxyPassHeaders      []string

	// http://nginx.org/en/docs/http/ngx_http_realip_module.html
	RealIPHeader    string
	SetRealIPFrom   []string
	RealIPRecursive bool
}

// Location describes an NGINX location
type Location struct {
	Path                 string
	Upstream             Upstream
	ProxyConnectTimeout  string
	ProxyReadTimeout     string
	ClientMaxBodySize    string
	Websocket            bool
	Rewrite              string
	SSL                  bool
	ProxyBuffering       bool
	ProxyBuffers         string
	ProxyBufferSize      string
	ProxyMaxTempFileSize string
}

// NginxMainConfig describe the main NGINX configuration file
type NginxMainConfig struct {
	ServerNamesHashBucketSize string
	ServerNamesHashMaxSize    string
	LogFormat                 string
	HealthStatus              bool
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html
	SSLProtocols           string
	SSLPreferServerCiphers bool
	SSLCiphers             string
	SSLDHParam             string
}

// NewUpstreamWithDefaultServer creates an upstream with the default server.
// proxy_pass to an upstream with the default server returns 502.
// We use it for services that have no endpoints
func NewUpstreamWithDefaultServer(name string) Upstream {
	return Upstream{
		Name:            name,
		UpstreamServers: []UpstreamServer{UpstreamServer{Address: "127.0.0.1", Port: "8181"}},
	}
}

// NewNginxController creates a NGINX controller
func NewNginxController(nginxConfPath string, local bool, healthStatus bool) (*NginxController, error) {
	ngxc := NginxController{
		nginxConfdPath: path.Join(nginxConfPath, "conf.d"),
		nginxCertsPath: path.Join(nginxConfPath, "ssl"),
		local:          local,
	}

	if !local {
		createDir(ngxc.nginxCertsPath)
	}

	cfg := &NginxMainConfig{ServerNamesHashMaxSize: NewDefaultConfig().MainServerNamesHashMaxSize, HealthStatus: healthStatus}
	ngxc.UpdateMainConfigFile(cfg)

	return &ngxc, nil
}

// DeleteIngress deletes the configuration file, which corresponds for the
// specified ingress from NGINX conf directory
func (nginx *NginxController) DeleteIngress(name string) {
	filename := nginx.getIngressNginxConfigFileName(name)
	glog.V(3).Infof("deleting %v", filename)

	if !nginx.local {
		if err := os.Remove(filename); err != nil {
			glog.Warningf("Failed to delete %v: %v", filename, err)
		}
	}
}

// AddOrUpdateIngress creates or updates a file with
// the specified configuration for the specified ingress
func (nginx *NginxController) AddOrUpdateIngress(name string, config IngressNginxConfig) {
	glog.V(3).Infof("Updating NGINX configuration")
	filename := nginx.getIngressNginxConfigFileName(name)
	nginx.templateIt(config, filename)
}

// AddOrUpdateDHParam creates the servers dhparam.pem file
func (nginx *NginxController) AddOrUpdateDHParam(dhparam string) (string, error) {
	fileName := nginx.nginxCertsPath + "/" + dhparamFilename
	if !nginx.local {
		pem, err := os.Create(fileName)
		if err != nil {
			return fileName, fmt.Errorf("Couldn't create file %v: %v", fileName, err)
		}
		defer pem.Close()

		_, err = pem.WriteString(dhparam)
		if err != nil {
			return fileName, fmt.Errorf("Couldn't write to pem file %v: %v", fileName, err)
		}
	}
	return fileName, nil
}

// AddOrUpdateCertAndKey creates a .pem file wth the cert and the key with the
// specified name
func (nginx *NginxController) AddOrUpdateCertAndKey(name string, cert string, key string) string {
	pemFileName := nginx.nginxCertsPath + "/" + name + ".pem"

	if !nginx.local {
		pem, err := os.Create(pemFileName)
		if err != nil {
			glog.Fatalf("Couldn't create pem file %v: %v", pemFileName, err)
		}
		defer pem.Close()

		_, err = pem.WriteString(key)
		if err != nil {
			glog.Fatalf("Couldn't write to pem file %v: %v", pemFileName, err)
		}

		_, err = pem.WriteString("\n")
		if err != nil {
			glog.Fatalf("Couldn't write to pem file %v: %v", pemFileName, err)
		}

		_, err = pem.WriteString(cert)
		if err != nil {
			glog.Fatalf("Couldn't write to pem file %v: %v", pemFileName, err)
		}
	}

	return pemFileName
}

func (nginx *NginxController) getIngressNginxConfigFileName(name string) string {
	return path.Join(nginx.nginxConfdPath, name+".conf")
}

func (nginx *NginxController) templateIt(config IngressNginxConfig, filename string) {
	tmpl, err := template.New("ingress.tmpl").ParseFiles("ingress.tmpl")
	if err != nil {
		glog.Fatal("Failed to parse template file")
	}

	glog.V(3).Infof("Writing NGINX conf to %v", filename)

	if glog.V(3) {
		tmpl.Execute(os.Stdout, config)
	}

	if !nginx.local {
		w, err := os.Create(filename)
		if err != nil {
			glog.Fatalf("Failed to open %v: %v", filename, err)
		}
		defer w.Close()

		if err := tmpl.Execute(w, config); err != nil {
			glog.Fatalf("Failed to write template %v", err)
		}
	} else {
		// print conf to stdout here
	}

	glog.V(3).Infof("NGINX configuration file had been updated")
}

// Reload reloads NGINX
func (nginx *NginxController) Reload() error {
	if !nginx.local {
		if err := shellOut("nginx -t"); err != nil {
			return fmt.Errorf("Invalid nginx configuration detected, not reloading: %s", err)
		}
		if err := shellOut("nginx -s reload"); err != nil {
			return fmt.Errorf("Reloading NGINX failed: %s", err)
		}
	} else {
		glog.V(3).Info("Reloading nginx")
	}
	return nil
}

// Start starts NGINX
func (nginx *NginxController) Start() {
	if !nginx.local {
		if err := shellOut("nginx"); err != nil {
			glog.Fatalf("Failed to start nginx: %v", err)
		}
	} else {
		glog.V(3).Info("Starting nginx")
	}
}

func createDir(path string) {
	if err := os.Mkdir(path, os.ModeDir); err != nil {
		glog.Fatalf("Couldn't create directory %v: %v", path, err)
	}
}

func shellOut(cmd string) (err error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	glog.V(3).Infof("executing %s", cmd)

	command := exec.Command("sh", "-c", cmd)
	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Start()
	if err != nil {
		return fmt.Errorf("Failed to execute %v, err: %v", cmd, err)
	}

	err = command.Wait()
	if err != nil {
		return fmt.Errorf("Command %v stdout: %q\nstderr: %q\nfinished with error: %v", cmd,
			stdout.String(), stderr.String(), err)
	}
	return nil
}

// UpdateMainConfigFile update the main NGINX configuration file
func (nginx *NginxController) UpdateMainConfigFile(cfg *NginxMainConfig) {
	tmpl, err := template.New("nginx.conf.tmpl").ParseFiles("nginx.conf.tmpl")
	if err != nil {
		glog.Fatalf("Failed to parse the main config template file: %v", err)
	}

	filename := "/etc/nginx/nginx.conf"
	glog.V(3).Infof("Writing NGINX conf to %v", filename)

	if glog.V(3) {
		tmpl.Execute(os.Stdout, cfg)
	}

	if !nginx.local {
		w, err := os.Create(filename)
		if err != nil {
			glog.Fatalf("Failed to open %v: %v", filename, err)
		}
		defer w.Close()

		if err := tmpl.Execute(w, cfg); err != nil {
			glog.Fatalf("Failed to write template %v", err)
		}
	}

	glog.V(3).Infof("The main NGINX configuration file had been updated")
}

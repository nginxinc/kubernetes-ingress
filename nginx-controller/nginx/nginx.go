package nginx

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/golang/glog"
)

const dhparamFilename = "dhparam.pem"

// TLSSecretFileMode defines the default filemode for files with TLS Secrets
const TLSSecretFileMode = 0600
const jwkSecretFileMode = 0644

// NginxController updates NGINX configuration, starts and reloads NGINX
type NginxController struct {
	nginxConfdPath   string
	nginxSecretsPath string
	local            bool
}

// IngressNginxConfig describes an NGINX configuration
type IngressNginxConfig struct {
	Upstreams []Upstream
	Servers   []Server
	Keepalive string
}

// Upstream describes an NGINX upstream
type Upstream struct {
	Name            string
	UpstreamServers []UpstreamServer
	StickyCookie    string
	LBMethod        string
	Queue           int64
	QueueTimeout    int64
}

// UpstreamServer describes a server in an NGINX upstream
type UpstreamServer struct {
	Address     string
	Port        string
	MaxFails    int64
	FailTimeout string
	SlowStart   string
}

// HealthCheck describes an active HTTP health check
type HealthCheck struct {
	UpstreamName   string
	URI            string
	Interval       int32
	Fails          int32
	Passes         int32
	Scheme         string
	Mandatory      bool
	Headers        map[string]string
	TimeoutSeconds int64
}

// Server describes an NGINX server
type Server struct {
	ServerSnippets        []string
	Name                  string
	ServerTokens          string
	Locations             []Location
	SSL                   bool
	SSLCertificate        string
	SSLCertificateKey     string
	GRPCOnly              bool
	StatusZone            string
	HTTP2                 bool
	RedirectToHTTPS       bool
	SSLRedirect           bool
	ProxyProtocol         bool
	HSTS                  bool
	HSTSMaxAge            int64
	HSTSIncludeSubdomains bool
	ProxyHideHeaders      []string
	ProxyPassHeaders      []string

	HealthChecks map[string]HealthCheck

	// http://nginx.org/en/docs/http/ngx_http_realip_module.html
	RealIPHeader    string
	SetRealIPFrom   []string
	RealIPRecursive bool

	JWTAuth              *JWTAuth
	JWTRedirectLocations []JWTRedirectLocation

	Ports    []int
	SSLPorts []int

	// Used for mergeable types
	IngressResource string
}

// JWTRedirectLocation describes a location for redirecting client requests to a login URL for JWT Authentication
type JWTRedirectLocation struct {
	Name     string
	LoginURL string
}

// JWTAuth holds JWT authentication configuration
type JWTAuth struct {
	Key                  string
	Realm                string
	Token                string
	RedirectLocationName string
}

// Location describes an NGINX location
type Location struct {
	LocationSnippets     []string
	Path                 string
	Upstream             Upstream
	ProxyConnectTimeout  string
	ProxyReadTimeout     string
	ClientMaxBodySize    string
	Websocket            bool
	Rewrite              string
	SSL                  bool
	GRPC                 bool
	ProxyBuffering       bool
	ProxyBuffers         string
	ProxyBufferSize      string
	ProxyMaxTempFileSize string
	JWTAuth              *JWTAuth

	// Used for mergeable types
	IngressResource string
}

// NginxMainConfig describe the main NGINX configuration file
type NginxMainConfig struct {
	ServerNamesHashBucketSize string
	ServerNamesHashMaxSize    string
	LogFormat                 string
	HealthStatus              bool
	MainSnippets              []string
	HTTPSnippets              []string
	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html
	SSLProtocols           string
	SSLPreferServerCiphers bool
	SSLCiphers             string
	SSLDHParam             string
	HTTP2                  bool
	ServerTokens           string
	ProxyProtocol          bool
	WorkerProcesses        string
	WorkerCPUAffinity      string
	WorkerShutdownTimeout  string
	WorkerConnections      string
	WorkerRlimitNofile     string
}

// NewUpstreamWithDefaultServer creates an upstream with the default server.
// proxy_pass to an upstream with the default server returns 502.
// We use it for services that have no endpoints
func NewUpstreamWithDefaultServer(name string) Upstream {
	return Upstream{
		Name: name,
		UpstreamServers: []UpstreamServer{
			UpstreamServer{
				Address:     "127.0.0.1",
				Port:        "8181",
				MaxFails:    1,
				FailTimeout: "10s",
			}},
	}
}

// NewNginxController creates a NGINX controller
func NewNginxController(nginxConfPath string, local bool) *NginxController {
	ngxc := NginxController{
		nginxConfdPath:   path.Join(nginxConfPath, "conf.d"),
		nginxSecretsPath: path.Join(nginxConfPath, "secrets"),
		local:            local,
	}

	return &ngxc
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

// AddOrUpdateDHParam creates the servers dhparam.pem file
func (nginx *NginxController) AddOrUpdateDHParam(dhparam string) (string, error) {
	fileName := nginx.nginxSecretsPath + "/" + dhparamFilename
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

// AddOrUpdateSecretFile creates a file with the specified name, content and mode.
func (nginx *NginxController) AddOrUpdateSecretFile(name string, content []byte, mode os.FileMode) string {
	filename := nginx.getSecretFileName(name)

	if !nginx.local {
		file, err := ioutil.TempFile(nginx.nginxSecretsPath, name)
		if err != nil {
			glog.Fatalf("Couldn't create a temp file for the secret file %v: %v", name, err)
		}

		err = file.Chmod(mode)
		if err != nil {
			glog.Fatalf("Couldn't change the mode of the temp secret file %v: %v", file.Name(), err)
		}

		_, err = file.Write(content)
		if err != nil {
			glog.Fatalf("Couldn't write to the temp secret file %v: %v", file.Name(), err)
		}

		err = file.Close()
		if err != nil {
			glog.Fatalf("Couldn't close the temp secret file %v: %v", file.Name(), err)
		}

		err = os.Rename(file.Name(), filename)
		if err != nil {
			glog.Fatalf("Fail to rename the temp secret file %v to %v: %v", file.Name(), filename, err)
		}
	}

	return filename
}

// DeleteSecretFile the file with a Secret
func (nginx *NginxController) DeleteSecretFile(name string) {
	filename := nginx.getSecretFileName(name)
	glog.V(3).Infof("deleting %v", filename)

	if !nginx.local {
		if err := os.Remove(filename); err != nil {
			glog.Warningf("Failed to delete %v: %v", filename, err)
		}
	}

}

func (nginx *NginxController) getIngressNginxConfigFileName(name string) string {
	return path.Join(nginx.nginxConfdPath, name+".conf")
}

func (nginx *NginxController) getSecretFileName(name string) string {
	return path.Join(nginx.nginxSecretsPath, name)
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
func (nginx *NginxController) Start(done chan error) {
	if !nginx.local {
		cmd := exec.Command("nginx")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			glog.Fatalf("Failed to start nginx: %v", err)
		}
		go func() {
			done <- cmd.Wait()
		}()
	} else {
		glog.V(3).Info("Starting nginx")
	}
}

// Quit shutdowns NGINX gracefully
func (nginx *NginxController) Quit() {
	if !nginx.local {
		if err := shellOut("nginx -s quit"); err != nil {
			glog.Fatalf("Failed to quit nginx: %v", err)
		}
	} else {
		glog.V(3).Info("Quitting nginx")
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

// UpdateMainConfigFile writes the main NGINX configuration file to the filesystem
func (nginx *NginxController) UpdateMainConfigFile(cfg []byte) {
	filename := "/etc/nginx/nginx.conf"
	glog.V(3).Infof("Writing NGINX conf to %v", filename)

	if bool(glog.V(3)) || nginx.local {
		glog.Info(string(cfg))
	}

	if !nginx.local {
		w, err := os.Create(filename)
		if err != nil {
			glog.Fatalf("Failed to open %v: %v", filename, err)
		}
		_, err = w.Write(cfg)
		if err != nil {
			glog.Fatalf("Failed to write to %v: %v", filename, err)
		}
		defer w.Close()
	}
	glog.V(3).Infof("The main NGINX config file has been updated")
}

// UpdateIngressConfigFile writes the Ingress configuration file to the filesystem
func (nginx *NginxController) UpdateIngressConfigFile(name string, cfg []byte) {
	filename := nginx.getIngressNginxConfigFileName(name)
	glog.V(3).Infof("Writing Ingress conf to %v", filename)

	if bool(glog.V(3)) || nginx.local {
		glog.Info(string(cfg))
	}

	if !nginx.local {
		w, err := os.Create(filename)
		if err != nil {
			glog.Fatalf("Failed to open %v: %v", filename, err)
		}
		_, err = w.Write(cfg)
		if err != nil {
			glog.Fatalf("Failed to write to %v: %v", filename, err)
		}
		defer w.Close()
	}
	glog.V(3).Infof("The Ingress config file has been updated")
}

package nginx

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"

	"github.com/golang/glog"
	"github.com/nginxinc/nginx-plus-go-client/client"
)

// TLSSecretFileMode defines the default filemode for files with TLS Secrets.
const TLSSecretFileMode = 0600

// JWKSecretFileMode defines the default filemode for files with JWK Secrets.
const JWKSecretFileMode = 0644

const configFileMode = 0644
const jsonFileForOpenTracingTracer = "/var/lib/nginx/tracer-config.json"

// ServerConfig holds the config data for an upstream server in NGINX Plus.
type ServerConfig struct {
	MaxFails    int
	MaxConns    int
	FailTimeout string
	SlowStart   string
}

// The Manager interface updates NGINX configuration, starts, reloads and quits NGINX,
// updates NGINX Plus upstream servers.
type Manager interface {
	CreateMainConfig(content []byte)
	CreateConfig(name string, content []byte)
	DeleteConfig(name string)
	CreateSecret(name string, content []byte, mode os.FileMode) string
	DeleteSecret(name string)
	GetFilenameForSecret(name string) string
	CreateDHParam(content string) (string, error)
	CreateOpenTracingTracerConfig(content string) error
	Start(done chan error)
	Reload() error
	Quit()
	UpdateConfigVersionFile(openTracing bool)
	SetPlusClients(plusClient *client.NginxClient, plusConfigVersionCheckClient *http.Client)
	UpdateServersInPlus(upstream string, servers []string, config ServerConfig) error
	SetOpenTracing(openTracing bool)
}

// LocalManager updates NGINX configuration, starts, reloads and quits NGINX,
// updates NGINX Plus upstream servers. It assumes that NGINX is running in the same container.
type LocalManager struct {
	confdPath                    string
	secretsPath                  string
	mainConfFilename             string
	configVersionFilename        string
	binaryFilename               string
	dhparamFilename              string
	verifyConfigGenerator        *verifyConfigGenerator
	verifyClient                 *verifyClient
	configVersion                int
	reloadCmd                    string
	quitCmd                      string
	plusClient                   *client.NginxClient
	plusConfigVersionCheckClient *http.Client
	metricsCollector             collectors.ManagerCollector
	OpenTracing                  bool
}

// NewLocalManager creates a LocalManager.
func NewLocalManager(confPath string, binaryFilename string, mc collectors.ManagerCollector) *LocalManager {
	verifyConfigGenerator, err := newVerifyConfigGenerator()
	if err != nil {
		glog.Fatalf("error instantiating a verifyConfigGenerator: %v", err)
	}

	manager := LocalManager{
		confdPath:             path.Join(confPath, "conf.d"),
		secretsPath:           path.Join(confPath, "secrets"),
		dhparamFilename:       path.Join(confPath, "secrets", "dhparam.pem"),
		mainConfFilename:      path.Join(confPath, "nginx.conf"),
		configVersionFilename: path.Join(confPath, "config-version.conf"),
		binaryFilename:        binaryFilename,
		verifyConfigGenerator: verifyConfigGenerator,
		configVersion:         0,
		verifyClient:          newVerifyClient(),
		reloadCmd:             fmt.Sprintf("%v -s %v", binaryFilename, "reload"),
		quitCmd:               fmt.Sprintf("%v -s %v", binaryFilename, "quit"),
		metricsCollector:      mc,
	}

	return &manager
}

// CreateMainConfig creates the main NGINX configuration file. If the file already exists, it will be overridden.
func (lm *LocalManager) CreateMainConfig(content []byte) {
	glog.V(3).Infof("Writing main config to %v", lm.mainConfFilename)
	glog.V(3).Infof(string(content))

	err := createFileAndWrite(lm.mainConfFilename, content)
	if err != nil {
		glog.Fatalf("Failed to write main config: %v", err)
	}
}

// CreateConfig creates a configuration file. If the file already exists, it will be overridden.
func (lm *LocalManager) CreateConfig(name string, content []byte) {
	filename := lm.getFilenameForConfig(name)

	glog.V(3).Infof("Writing config to %v", filename)
	glog.V(3).Info(string(content))

	err := createFileAndWrite(filename, content)
	if err != nil {
		glog.Fatalf("Failed to write config to %v: %v", filename, err)
	}
}

// DeleteConfig deletes the configuration file from the conf.d folder.
func (lm *LocalManager) DeleteConfig(name string) {
	filename := lm.getFilenameForConfig(name)

	glog.V(3).Infof("Deleting config from %v", filename)

	if err := os.Remove(filename); err != nil {
		glog.Warningf("Failed to delete config from %v: %v", filename, err)
	}
}

func (lm *LocalManager) getFilenameForConfig(name string) string {
	return path.Join(lm.confdPath, name+".conf")
}

// CreateSecret creates a secret file with the specified name, content and mode. If the file already exists,
// it will be overridden.
func (lm *LocalManager) CreateSecret(name string, content []byte, mode os.FileMode) string {
	filename := lm.GetFilenameForSecret(name)

	glog.V(3).Infof("Writing secret to %v", filename)

	createFileAndWriteAtomically(filename, lm.secretsPath, mode, content)

	return filename
}

// DeleteSecret the file with the secret.
func (lm *LocalManager) DeleteSecret(name string) {
	filename := lm.GetFilenameForSecret(name)

	glog.V(3).Infof("Deleting secret from %v", filename)

	if err := os.Remove(filename); err != nil {
		glog.Warningf("Failed to delete secret from %v: %v", filename, err)
	}
}

// GetFilenameForSecret constructs the filename for the secret.
func (lm *LocalManager) GetFilenameForSecret(name string) string {
	return path.Join(lm.secretsPath, name)
}

// CreateDHParam creates the servers dhparam.pem file. If the file already exists, it will be overridden.
func (lm *LocalManager) CreateDHParam(content string) (string, error) {
	glog.V(3).Infof("Writing dhparam file to %v", lm.dhparamFilename)

	err := createFileAndWrite(lm.dhparamFilename, []byte(content))
	if err != nil {
		return lm.dhparamFilename, fmt.Errorf("Failed to write dhparam file from %v: %v", lm.dhparamFilename, err)
	}

	return lm.dhparamFilename, nil
}

// Start starts NGINX.
func (lm *LocalManager) Start(done chan error) {
	glog.V(3).Info("Starting nginx")

	cmd := exec.Command(lm.binaryFilename)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		glog.Fatalf("Failed to start nginx: %v", err)
	}

	go func() {
		done <- cmd.Wait()
	}()

	err := lm.verifyClient.WaitForCorrectVersion(lm.configVersion)
	if err != nil {
		glog.Fatalf("Could not get newest config version: %v", err)
	}
}

// Reload reloads NGINX.
func (lm *LocalManager) Reload() error {
	// write a new config version
	lm.configVersion++
	lm.UpdateConfigVersionFile(lm.OpenTracing)

	glog.V(3).Infof("Reloading nginx with configVersion: %v", lm.configVersion)

	t1 := time.Now()

	if err := shellOut(lm.reloadCmd); err != nil {
		lm.metricsCollector.IncNginxReloadErrors()
		return fmt.Errorf("nginx reload failed: %v", err)
	}
	err := lm.verifyClient.WaitForCorrectVersion(lm.configVersion)
	if err != nil {
		lm.metricsCollector.IncNginxReloadErrors()
		return fmt.Errorf("could not get newest config version: %v", err)
	}

	lm.metricsCollector.IncNginxReloadCount()

	t2 := time.Now()
	lm.metricsCollector.UpdateLastReloadTime(t2.Sub(t1))
	return nil
}

// Quit shutdowns NGINX gracefully.
func (lm *LocalManager) Quit() {
	glog.V(3).Info("Quitting nginx")

	if err := shellOut(lm.quitCmd); err != nil {
		glog.Fatalf("Failed to quit nginx: %v", err)
	}
}

// UpdateConfigVersionFile writes the config version file.
func (lm *LocalManager) UpdateConfigVersionFile(openTracing bool) {
	cfg, err := lm.verifyConfigGenerator.GenerateVersionConfig(lm.configVersion, openTracing)
	if err != nil {
		glog.Fatalf("Error generating config version content: %v", err)
	}

	glog.V(3).Infof("Writing config version to %v", lm.configVersionFilename)
	glog.V(3).Info(string(cfg))

	createFileAndWriteAtomically(lm.configVersionFilename, path.Dir(lm.configVersionFilename), configFileMode, cfg)
}

// SetPlusClients sets the necessary clients to work with NGINX Plus API. If not set, invoking the UpdateServersInPlus
// will fail.
func (lm *LocalManager) SetPlusClients(plusClient *client.NginxClient, plusConfigVersionCheckClient *http.Client) {
	lm.plusClient = plusClient
	lm.plusConfigVersionCheckClient = plusConfigVersionCheckClient
}

// UpdateServersInPlus updates NGINX Plus servers of the given upstream.
func (lm *LocalManager) UpdateServersInPlus(upstream string, servers []string, config ServerConfig) error {
	err := verifyConfigVersion(lm.plusConfigVersionCheckClient, lm.configVersion)
	if err != nil {
		return fmt.Errorf("error verifying config version: %v", err)
	}

	glog.V(3).Infof("API has the correct config version: %v.", lm.configVersion)

	var upsServers []client.UpstreamServer
	for _, s := range servers {
		upsServers = append(upsServers, client.UpstreamServer{
			Server:      s,
			MaxFails:    config.MaxFails,
			MaxConns:    config.MaxConns,
			FailTimeout: config.FailTimeout,
			SlowStart:   config.SlowStart,
		})
	}

	added, removed, err := lm.plusClient.UpdateHTTPServers(upstream, upsServers)
	if err != nil {
		glog.V(3).Infof("Couldn't update servers of %v upstream: %v", upstream, err)
		return fmt.Errorf("error updating servers of %v upstream: %v", upstream, err)
	}

	glog.V(3).Infof("Updated servers of %v; Added: %v, Removed: %v", upstream, added, removed)

	return nil
}

// CreateOpenTracingTracerConfig creates a json configuration file for the OpenTracing tracer with the content of the string.
func (lm *LocalManager) CreateOpenTracingTracerConfig(content string) error {
	glog.V(3).Infof("Writing OpenTracing tracer config file to %v", jsonFileForOpenTracingTracer)
	err := createFileAndWrite(jsonFileForOpenTracingTracer, []byte(content))
	if err != nil {
		return fmt.Errorf("Failed to write config file: %v", err)
	}

	return nil
}

// verifyConfigVersion is used to check if the worker process that the API client is connected
// to is using the latest version of nginx config. This way we avoid making changes on
// a worker processes that is being shut down.
func verifyConfigVersion(httpClient *http.Client, configVersion int) error {
	req, err := http.NewRequest("GET", "http://nginx-plus-api/configVersionCheck", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("x-expected-config-version", fmt.Sprintf("%v", configVersion))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error doing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned non-success status: %v", resp.StatusCode)
	}

	return nil
}

// SetOpenTracing sets the value of OpenTracing for the Manager
func (lm *LocalManager) SetOpenTracing(openTracing bool) {
	lm.OpenTracing = openTracing
}

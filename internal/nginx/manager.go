package nginx

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"

	"github.com/golang/glog"
	"github.com/nginxinc/nginx-plus-go-client/client"
)

const (
	// ReloadForEndpointsUpdate means that is caused by an endpoints update.
	ReloadForEndpointsUpdate = true
	// ReloadForOtherUpdate means that a reload is caused by an update for a resource(s) other than endpoints.
	ReloadForOtherUpdate = false
	// TLSSecretFileMode defines the default filemode for files with TLS Secrets.
	TLSSecretFileMode = 0o600
	// JWKSecretFileMode defines the default filemode for files with JWK Secrets.
	JWKSecretFileMode = 0o644
	// HtpasswdSecretFileMode defines the default filemode for HTTP basic auth user files.
	HtpasswdSecretFileMode = 0o644

	configFileMode               = 0o644
	jsonFileForOpenTracingTracer = "/var/lib/nginx/tracer-config.json"
	nginxBinaryPath              = "/usr/sbin/nginx"
	nginxBinaryPathDebug         = "/usr/sbin/nginx-debug"

	appProtectPluginStartCmd = "/usr/share/ts/bin/bd-socket-plugin"
	appProtectAgentStartCmd  = "/opt/app_protect/bin/bd_agent"
	appProtectLogLevelCmd    = "/opt/app_protect/bin/set_log_level"

	// appPluginParams is the configuration of App-Protect plugin
	appPluginParams = "tmm_count 4 proc_cpuinfo_cpu_mhz 2000000 total_xml_memory 471859200 total_umu_max_size 3129344 sys_max_account_id 1024 no_static_config"

	appProtectDosAgentInstallCmd    = "/usr/bin/adminstall"
	appProtectDosAgentStartCmd      = "/usr/bin/admd -d --standalone"
	appProtectDosAgentStartDebugCmd = "/usr/bin/admd -d --standalone --log debug"
)

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
	CreateStreamConfig(name string, content []byte)
	DeleteStreamConfig(name string)
	CreateTLSPassthroughHostsConfig(content []byte)
	CreateSecret(name string, content []byte, mode os.FileMode) string
	DeleteSecret(name string)
	CreateAppProtectResourceFile(name string, content []byte)
	DeleteAppProtectResourceFile(name string)
	ClearAppProtectFolder(name string)
	GetFilenameForSecret(name string) string
	CreateDHParam(content string) (string, error)
	CreateOpenTracingTracerConfig(content string) error
	Start(done chan error)
	Version() string
	Reload(isEndpointsUpdate bool) error
	Quit()
	UpdateConfigVersionFile(openTracing bool)
	SetPlusClients(plusClient *client.NginxClient, plusConfigVersionCheckClient *http.Client)
	UpdateServersInPlus(upstream string, servers []string, config ServerConfig) error
	UpdateStreamServersInPlus(upstream string, servers []string) error
	SetOpenTracing(openTracing bool)
	AppProtectAgentStart(apaDone chan error, logLevel string)
	AppProtectAgentQuit()
	AppProtectPluginStart(appDone chan error)
	AppProtectPluginQuit()
	AppProtectDosAgentStart(apdaDone chan error, debug bool, maxDaemon int, maxWorkers int, memory int)
	AppProtectDosAgentQuit()
}

// LocalManager updates NGINX configuration, starts, reloads and quits NGINX,
// updates NGINX Plus upstream servers. It assumes that NGINX is running in the same container.
type LocalManager struct {
	confdPath                    string
	streamConfdPath              string
	secretsPath                  string
	mainConfFilename             string
	configVersionFilename        string
	debug                        bool
	dhparamFilename              string
	tlsPassthroughHostsFilename  string
	verifyConfigGenerator        *verifyConfigGenerator
	verifyClient                 *verifyClient
	configVersion                int
	plusClient                   *client.NginxClient
	plusConfigVersionCheckClient *http.Client
	metricsCollector             collectors.ManagerCollector
	OpenTracing                  bool
	appProtectPluginPid          int
	appProtectAgentPid           int
	appProtectDosAgentPid        int
}

// NewLocalManager creates a LocalManager.
func NewLocalManager(confPath string, debug bool, mc collectors.ManagerCollector, timeout time.Duration) *LocalManager {
	verifyConfigGenerator, err := newVerifyConfigGenerator()
	if err != nil {
		glog.Fatalf("error instantiating a verifyConfigGenerator: %v", err)
	}

	manager := LocalManager{
		confdPath:                   path.Join(confPath, "conf.d"),
		streamConfdPath:             path.Join(confPath, "stream-conf.d"),
		secretsPath:                 path.Join(confPath, "secrets"),
		dhparamFilename:             path.Join(confPath, "secrets", "dhparam.pem"),
		mainConfFilename:            path.Join(confPath, "nginx.conf"),
		configVersionFilename:       path.Join(confPath, "config-version.conf"),
		tlsPassthroughHostsFilename: path.Join(confPath, "tls-passthrough-hosts.conf"),
		debug:                       debug,
		verifyConfigGenerator:       verifyConfigGenerator,
		configVersion:               0,
		verifyClient:                newVerifyClient(timeout),
		metricsCollector:            mc,
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
	createConfig(lm.getFilenameForConfig(name), content)
}

func createConfig(filename string, content []byte) {
	glog.V(3).Infof("Writing config to %v", filename)
	glog.V(3).Info(string(content))

	err := createFileAndWrite(filename, content)
	if err != nil {
		glog.Fatalf("Failed to write config to %v: %v", filename, err)
	}
}

// DeleteConfig deletes the configuration file from the conf.d folder.
func (lm *LocalManager) DeleteConfig(name string) {
	deleteConfig(lm.getFilenameForConfig(name))
}

func deleteConfig(filename string) {
	glog.V(3).Infof("Deleting config from %v", filename)

	if err := os.Remove(filename); err != nil {
		glog.Warningf("Failed to delete config from %v: %v", filename, err)
	}
}

func (lm *LocalManager) getFilenameForConfig(name string) string {
	return path.Join(lm.confdPath, name+".conf")
}

// CreateStreamConfig creates a configuration file for stream module.
// If the file already exists, it will be overridden.
func (lm *LocalManager) CreateStreamConfig(name string, content []byte) {
	createConfig(lm.getFilenameForStreamConfig(name), content)
}

// DeleteStreamConfig deletes the configuration file from the stream-conf.d folder.
func (lm *LocalManager) DeleteStreamConfig(name string) {
	deleteConfig(lm.getFilenameForStreamConfig(name))
}

func (lm *LocalManager) getFilenameForStreamConfig(name string) string {
	return path.Join(lm.streamConfdPath, name+".conf")
}

// CreateTLSPassthroughHostsConfig creates a configuration file with mapping between TLS Passthrough hosts and
// the corresponding unix sockets.
// If the file already exists, it will be overridden.
func (lm *LocalManager) CreateTLSPassthroughHostsConfig(content []byte) {
	glog.V(3).Infof("Writing TLS Passthrough Hosts config file to %v", lm.tlsPassthroughHostsFilename)
	createConfig(lm.tlsPassthroughHostsFilename, content)
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
		return lm.dhparamFilename, fmt.Errorf("failed to write dhparam file from %v: %w", lm.dhparamFilename, err)
	}

	return lm.dhparamFilename, nil
}

// CreateAppProtectResourceFile writes contents of An App Protect resource to a file
func (lm *LocalManager) CreateAppProtectResourceFile(name string, content []byte) {
	glog.V(3).Infof("Writing App Protect Resource to %v", name)
	err := createFileAndWrite(name, content)
	if err != nil {
		glog.Fatalf("Failed to write App Protect Resource to %v: %v", name, err)
	}
}

// DeleteAppProtectResourceFile removes an App Protect resource file from storage
func (lm *LocalManager) DeleteAppProtectResourceFile(name string) {
	// This check is done to avoid errors in case eg. a policy is referenced, but it never became valid.
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		if err := os.Remove(name); err != nil {
			glog.Fatalf("Failed to delete App Protect Resource from %v: %v", name, err)
		}
	}
}

// ClearAppProtectFolder clears contents of a config folder
func (lm *LocalManager) ClearAppProtectFolder(name string) {
	files, err := os.ReadDir(name)
	if err != nil {
		glog.Fatalf("Failed to read the App Protect folder %s: %v", name, err)
	}
	for _, file := range files {
		lm.DeleteAppProtectResourceFile(fmt.Sprintf("%s/%s", name, file.Name()))
	}
}

// Start starts NGINX.
func (lm *LocalManager) Start(done chan error) {
	glog.V(3).Info("Starting nginx")

	binaryFilename := getBinaryFileName(lm.debug)
	cmd := exec.Command(binaryFilename, "-e", "stderr") // #nosec G204
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
func (lm *LocalManager) Reload(isEndpointsUpdate bool) error {
	// write a new config version
	lm.configVersion++
	lm.UpdateConfigVersionFile(lm.OpenTracing)

	glog.V(3).Infof("Reloading nginx with configVersion: %v", lm.configVersion)

	t1 := time.Now()

	binaryFilename := getBinaryFileName(lm.debug)
	if err := shellOut(fmt.Sprintf("%v -s %v -e stderr", binaryFilename, "reload")); err != nil {
		lm.metricsCollector.IncNginxReloadErrors()
		return fmt.Errorf("nginx reload failed: %w", err)
	}
	err := lm.verifyClient.WaitForCorrectVersion(lm.configVersion)
	if err != nil {
		lm.metricsCollector.IncNginxReloadErrors()
		return fmt.Errorf("could not get newest config version: %w", err)
	}

	lm.metricsCollector.IncNginxReloadCount(isEndpointsUpdate)

	t2 := time.Now()
	lm.metricsCollector.UpdateLastReloadTime(t2.Sub(t1))
	return nil
}

// Quit shutdowns NGINX gracefully.
func (lm *LocalManager) Quit() {
	glog.V(3).Info("Quitting nginx")

	binaryFilename := getBinaryFileName(lm.debug)
	if err := shellOut(fmt.Sprintf("%v -s %v", binaryFilename, "quit")); err != nil {
		glog.Fatalf("Failed to quit nginx: %v", err)
	}
}

// Version returns NGINX version
func (lm *LocalManager) Version() string {
	binaryFilename := getBinaryFileName(lm.debug)
	out, err := exec.Command(binaryFilename, "-v").CombinedOutput()
	if err != nil {
		glog.Fatalf("Failed to get nginx version: %v", err)
	}
	return string(out)
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
		return fmt.Errorf("error verifying config version: %w", err)
	}

	glog.V(3).Infof("API has the correct config version: %v.", lm.configVersion)

	var upsServers []client.UpstreamServer
	for _, s := range servers {
		upsServers = append(upsServers, client.UpstreamServer{
			Server:      s,
			MaxFails:    &config.MaxFails,
			MaxConns:    &config.MaxConns,
			FailTimeout: config.FailTimeout,
			SlowStart:   config.SlowStart,
		})
	}

	added, removed, updated, err := lm.plusClient.UpdateHTTPServers(upstream, upsServers)
	if err != nil {
		glog.V(3).Infof("Couldn't update servers of %v upstream: %v", upstream, err)
		return fmt.Errorf("error updating servers of %v upstream: %w", upstream, err)
	}

	glog.V(3).Infof("Updated servers of %v; Added: %v, Removed: %v, Updated: %v", upstream, added, removed, updated)

	return nil
}

// UpdateStreamServersInPlus updates NGINX Plus stream servers of the given upstream.
func (lm *LocalManager) UpdateStreamServersInPlus(upstream string, servers []string) error {
	err := verifyConfigVersion(lm.plusConfigVersionCheckClient, lm.configVersion)
	if err != nil {
		return fmt.Errorf("error verifying config version: %w", err)
	}

	glog.V(3).Infof("API has the correct config version: %v.", lm.configVersion)

	var upsServers []client.StreamUpstreamServer
	for _, s := range servers {
		upsServers = append(upsServers, client.StreamUpstreamServer{
			Server: s,
		})
	}

	added, removed, updated, err := lm.plusClient.UpdateStreamServers(upstream, upsServers)
	if err != nil {
		glog.V(3).Infof("Couldn't update stream servers of %v upstream: %v", upstream, err)
		return fmt.Errorf("error updating stream servers of %v upstream: %w", upstream, err)
	}

	glog.V(3).Infof("Updated stream servers of %v; Added: %v, Removed: %v, Updated: %v", upstream, added, removed, updated)

	return nil
}

// CreateOpenTracingTracerConfig creates a json configuration file for the OpenTracing tracer with the content of the string.
func (lm *LocalManager) CreateOpenTracingTracerConfig(content string) error {
	glog.V(3).Infof("Writing OpenTracing tracer config file to %v", jsonFileForOpenTracingTracer)
	err := createFileAndWrite(jsonFileForOpenTracingTracer, []byte(content))
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// verifyConfigVersion is used to check if the worker process that the API client is connected
// to is using the latest version of nginx config. This way we avoid making changes on
// a worker processes that is being shut down.
func verifyConfigVersion(httpClient *http.Client, configVersion int) error {
	req, err := http.NewRequest("GET", "http://nginx-plus-api/configVersionCheck", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("x-expected-config-version", fmt.Sprintf("%v", configVersion))

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error doing request: %w", err)
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

// AppProtectAgentStart starts the AppProtect agent
func (lm *LocalManager) AppProtectAgentStart(apaDone chan error, logLevel string) {
	glog.V(3).Info("Setting log level for App Protect - ", logLevel)
	appProtectLogLevelCmdfull := fmt.Sprintf("%v %v", appProtectLogLevelCmd, logLevel)
	logLevelCmd := exec.Command("sh", "-c", appProtectLogLevelCmdfull) // #nosec G204
	if err := logLevelCmd.Run(); err != nil {
		glog.Fatalf("Failed to set log level for AppProtect: %v", err)
	}

	glog.V(3).Info("Starting AppProtect Agent")
	cmd := exec.Command(appProtectAgentStartCmd)
	if err := cmd.Start(); err != nil {
		glog.Fatalf("Failed to start AppProtect Agent: %v", err)
	}
	lm.appProtectAgentPid = cmd.Process.Pid
	go func() {
		apaDone <- cmd.Wait()
	}()
}

// AppProtectAgentQuit gracefully ends AppProtect Agent.
func (lm *LocalManager) AppProtectAgentQuit() {
	glog.V(3).Info("Quitting AppProtect Agent")
	killcmd := fmt.Sprintf("kill %d", lm.appProtectAgentPid)
	if err := shellOut(killcmd); err != nil {
		glog.Fatalf("Failed to quit AppProtect Agent: %v", err)
	}
}

// AppProtectPluginStart starts the AppProtect plugin.
func (lm *LocalManager) AppProtectPluginStart(appDone chan error) {
	glog.V(3).Info("Starting AppProtect Plugin")
	startupParams := strings.Fields(appPluginParams)
	cmd := exec.Command(appProtectPluginStartCmd, startupParams...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "LD_LIBRARY_PATH=/usr/lib64/bd")

	if err := cmd.Start(); err != nil {
		glog.Fatalf("Failed to start AppProtect Plugin: %v", err)
	}
	lm.appProtectPluginPid = cmd.Process.Pid
	go func() {
		appDone <- cmd.Wait()
	}()
}

// AppProtectPluginQuit gracefully ends AppProtect Agent.
func (lm *LocalManager) AppProtectPluginQuit() {
	glog.V(3).Info("Quitting AppProtect Plugin")
	killcmd := fmt.Sprintf("kill %d", lm.appProtectPluginPid)
	if err := shellOut(killcmd); err != nil {
		glog.Fatalf("Failed to quit AppProtect Plugin: %v", err)
	}
}

// AppProtectDosAgentQuit gracefully ends AppProtect Agent.
func (lm *LocalManager) AppProtectDosAgentQuit() {
	glog.V(3).Info("Quitting AppProtectDos Agent")
	killcmd := fmt.Sprintf("kill %d", lm.appProtectDosAgentPid)
	if err := shellOut(killcmd); err != nil {
		glog.Fatalf("Failed to quit AppProtect Agent: %v", err)
	}
}

// AppProtectDosAgentStart starts the AppProtectDos agent
func (lm *LocalManager) AppProtectDosAgentStart(apdaDone chan error, debug bool, maxDaemon int, maxWorkers int, memory int) {
	glog.V(3).Info("Starting AppProtectDos Agent")

	// Perform installation by adminstall
	appProtectDosAgentInstallCmdFull := appProtectDosAgentInstallCmd

	if maxDaemon != 0 {
		appProtectDosAgentInstallCmdFull += " -d " + strconv.Itoa(maxDaemon)
	}

	if maxWorkers != 0 {
		appProtectDosAgentInstallCmdFull += " -w " + strconv.Itoa(maxWorkers)
	}

	if memory != 0 {
		appProtectDosAgentInstallCmdFull += " -m " + strconv.Itoa(memory)
	}

	cmdInstall := exec.Command("sh", "-c", appProtectDosAgentInstallCmdFull)

	if err := cmdInstall.Run(); err != nil {
		glog.Fatalf("Failed to install AppProtectDos: %v", err)
	}

	// case debug add debug flag to admd
	appProtectDosAgentCmd := appProtectDosAgentStartCmd
	if debug {
		appProtectDosAgentCmd = appProtectDosAgentStartDebugCmd
	}

	cmd := exec.Command("sh", "-c", appProtectDosAgentCmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	if err := cmd.Start(); err != nil {
		glog.Fatalf("Failed to start AppProtectDos Agent: %v", err)
	}
	lm.appProtectDosAgentPid = cmd.Process.Pid
	go func() {
		apdaDone <- cmd.Wait()
	}()
}

func getBinaryFileName(debug bool) string {
	if debug {
		return nginxBinaryPathDebug
	}
	return nginxBinaryPath
}

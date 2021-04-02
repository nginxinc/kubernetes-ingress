package nginx

import (
	"net/http"
	"os"
	"path"

	"github.com/golang/glog"
	"github.com/nginxinc/nginx-plus-go-client/client"
)

// FakeManager provides a fake implementation of the Manager interface.
type FakeManager struct {
	confdPath       string
	secretsPath     string
	dhparamFilename string
}

// NewFakeManager creates a FakeMananger.
func NewFakeManager(confPath string) *FakeManager {
	return &FakeManager{
		confdPath:       path.Join(confPath, "conf.d"),
		secretsPath:     path.Join(confPath, "secrets"),
		dhparamFilename: path.Join(confPath, "secrets", "dhparam.pem"),
	}
}

// CreateMainConfig provides a fake implementation of CreateMainConfig.
func (*FakeManager) CreateMainConfig(content []byte) {
	glog.V(3).Info("Writing main config")
	glog.V(3).Info(string(content))
}

// CreateConfig provides a fake implementation of CreateConfig.
func (*FakeManager) CreateConfig(name string, content []byte) {
	glog.V(3).Infof("Writing config %v", name)
	glog.V(3).Info(string(content))
}

// CreateAppProtectResourceFile provides a fake implementation of CreateAppProtectResourceFile
func (*FakeManager) CreateAppProtectResourceFile(name string, content []byte) {
	glog.V(3).Infof("Writing Ap Resource File %v", name)
	glog.V(3).Info(string(content))
}

// DeleteAppProtectResourceFile provides a fake implementation of DeleteAppProtectResourceFile
func (*FakeManager) DeleteAppProtectResourceFile(name string) {
	glog.V(3).Infof("Deleting Ap Resource File %v", name)
}

// ClearAppProtectFolder provides a fake implementation of ClearAppProtectFolder
func (*FakeManager) ClearAppProtectFolder(name string) {
	glog.V(3).Infof("Deleting Ap Resource folder %v", name)
}

// DeleteConfig provides a fake implementation of DeleteConfig.
func (*FakeManager) DeleteConfig(name string) {
	glog.V(3).Infof("Deleting config %v", name)
}

// CreateStreamConfig provides a fake implementation of CreateStreamConfig.
func (*FakeManager) CreateStreamConfig(name string, content []byte) {
	glog.V(3).Infof("Writing stream config %v", name)
	glog.V(3).Info(string(content))
}

// DeleteStreamConfig provides a fake implementation of DeleteStreamConfig.
func (*FakeManager) DeleteStreamConfig(name string) {
	glog.V(3).Infof("Deleting stream config %v", name)
}

// CreateTLSPassthroughHostsConfig provides a fake implementation of CreateTLSPassthroughHostsConfig.
func (*FakeManager) CreateTLSPassthroughHostsConfig(content []byte) {
	glog.V(3).Infof("Writing TLS Passthrough Hosts config file")
}

// CreateSecret provides a fake implementation of CreateSecret.
func (fm *FakeManager) CreateSecret(name string, content []byte, mode os.FileMode) string {
	glog.V(3).Infof("Writing secret %v", name)
	return fm.GetFilenameForSecret(name)
}

// DeleteSecret provides a fake implementation of DeleteSecret.
func (*FakeManager) DeleteSecret(name string) {
	glog.V(3).Infof("Deleting secret %v", name)
}

// GetFilenameForSecret provides a fake implementation of GetFilenameForSecret.
func (fm *FakeManager) GetFilenameForSecret(name string) string {
	return path.Join(fm.secretsPath, name)
}

// CreateDHParam provides a fake implementation of CreateDHParam.
func (fm *FakeManager) CreateDHParam(content string) (string, error) {
	glog.V(3).Infof("Writing dhparam file")
	return fm.dhparamFilename, nil
}

// Version provides a fake implementation of Version.
func (*FakeManager) Version() string {
	glog.V(3).Info("Printing nginx version")
	return "fake version"
}

// Start provides a fake implementation of Start.
func (*FakeManager) Start(done chan error) {
	glog.V(3).Info("Starting nginx")
}

// Reload provides a fake implementation of Reload.
func (*FakeManager) Reload(isEndpointsUpdate bool) error {
	glog.V(3).Infof("Reloading nginx")
	return nil
}

// Quit provides a fake implementation of Quit.
func (*FakeManager) Quit() {
	glog.V(3).Info("Quitting nginx")
}

// UpdateConfigVersionFile provides a fake implementation of UpdateConfigVersionFile.
func (*FakeManager) UpdateConfigVersionFile(openTracing bool) {
	glog.V(3).Infof("Writing config version")
}

// SetPlusClients provides a fake implementation of SetPlusClients.
func (*FakeManager) SetPlusClients(plusClient *client.NginxClient, plusConfigVersionCheckClient *http.Client) {
}

// UpdateServersInPlus provides a fake implementation of UpdateServersInPlus.
func (*FakeManager) UpdateServersInPlus(upstream string, servers []string, config ServerConfig) error {
	glog.V(3).Infof("Updating servers of %v: %v", upstream, servers)
	return nil
}

// UpdateStreamServersInPlus provides a fake implementation of UpdateStreamServersInPlus.
func (*FakeManager) UpdateStreamServersInPlus(upstream string, servers []string) error {
	glog.V(3).Infof("Updating stream servers of %v: %v", upstream, servers)
	return nil
}

// CreateOpenTracingTracerConfig creates a fake implementation of CreateOpenTracingTracerConfig.
func (*FakeManager) CreateOpenTracingTracerConfig(content string) error {
	glog.V(3).Infof("Writing OpenTracing tracer config file")

	return nil
}

// SetOpenTracing creates a fake implementation of SetOpenTracing.
func (*FakeManager) SetOpenTracing(openTracing bool) {
}

// AppProtectAgentStart is a fake implementation of AppProtectAgentStart
func (*FakeManager) AppProtectAgentStart(apaDone chan error, debug bool) {
	glog.V(3).Infof("Starting FakeAppProtectAgent")
}

// AppProtectAgentQuit is a fake implementtion AppProtectAgentQuit
func (*FakeManager) AppProtectAgentQuit() {
	glog.V(3).Infof("Quitting FakeAppProtectAgent")
}

// AppProtectPluginStart is a fake implementtion AppProtectPluginStart
func (*FakeManager) AppProtectPluginStart(appDone chan error) {
	glog.V(3).Infof("Starting FakeAppProtectPlugin")
}

// AppProtectPluginQuit is a fake implementtion AppProtectPluginQuit
func (*FakeManager) AppProtectPluginQuit() {
	glog.V(3).Infof("Quitting FakeAppProtectPlugin")
}

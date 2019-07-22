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

// DeleteConfig provides a fake implementation of DeleteConfig.
func (*FakeManager) DeleteConfig(name string) {
	glog.V(3).Infof("Deleting config %v", name)
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

// Start provides a fake implementation of Start.
func (*FakeManager) Start(done chan error) {
	glog.V(3).Info("Starting nginx")
}

// Reload provides a fake implementation of Reload.
func (*FakeManager) Reload() error {
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

// CreateOpenTracingTracerConfig creates a fake implementation of CreateOpenTracingTracerConfig.
func (*FakeManager) CreateOpenTracingTracerConfig(content string) error {
	glog.V(3).Infof("Writing OpenTracing tracer config file")

	return nil
}

// SetOpenTracing creates a fake implementation of SetOpenTracing.
func (*FakeManager) SetOpenTracing(openTracing bool) {
}

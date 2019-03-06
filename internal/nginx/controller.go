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

// JWKSecretFileMode defines the default filemode for files with JWK Secrets
const JWKSecretFileMode = 0644

// Controller updates NGINX configuration, starts and reloads NGINX
type Controller struct {
	nginxConfdPath        string
	nginxSecretsPath      string
	local                 bool
	nginxBinaryPath       string
	verifyConfigGenerator *verifyConfigGenerator
	verifyClient          *verifyClient
	ConfigVersion         int
}

// NewNginxController creates a NGINX controller
func NewNginxController(nginxConfPath string, nginxBinaryPath string, local bool) *Controller {
	verifyConfigGenerator, err := newVerifyConfigGenerator()
	if err != nil {
		glog.Fatalf("error instantiating a verifyConfigGenerator: %v", err)
	}

	ngxc := Controller{
		nginxConfdPath:        path.Join(nginxConfPath, "conf.d"),
		nginxSecretsPath:      path.Join(nginxConfPath, "secrets"),
		local:                 local,
		nginxBinaryPath:       nginxBinaryPath,
		verifyConfigGenerator: verifyConfigGenerator,
		ConfigVersion:         0,
		verifyClient:          newVerifyClient(),
	}

	return &ngxc
}

// DeleteIngress deletes the configuration file, which corresponds for the
// specified ingress from NGINX conf directory
func (nginx *Controller) DeleteIngress(name string) {
	filename := nginx.getIngressNginxConfigFileName(name)
	glog.V(3).Infof("deleting %v", filename)

	if !nginx.local {
		if err := os.Remove(filename); err != nil {
			glog.Warningf("Failed to delete %v: %v", filename, err)
		}
	}
}

// AddOrUpdateDHParam creates the servers dhparam.pem file
func (nginx *Controller) AddOrUpdateDHParam(dhparam string) (string, error) {
	fileName := nginx.nginxSecretsPath + "/" + dhparamFilename
	if !nginx.local {
		err := createFileAndWrite(fileName, []byte(dhparam))
		if err != nil {
			return fileName, fmt.Errorf("Failed to write pem file: %v", err)
		}
	}
	return fileName, nil
}

// AddOrUpdateSecretFile creates a file with the specified name, content and mode.
func (nginx *Controller) AddOrUpdateSecretFile(name string, content []byte, mode os.FileMode) string {
	filename := nginx.GetSecretFileName(name)

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
func (nginx *Controller) DeleteSecretFile(name string) {
	filename := nginx.GetSecretFileName(name)
	glog.V(3).Infof("deleting %v", filename)

	if !nginx.local {
		if err := os.Remove(filename); err != nil {
			glog.Warningf("Failed to delete %v: %v", filename, err)
		}
	}
}

func (nginx *Controller) getIngressNginxConfigFileName(name string) string {
	return path.Join(nginx.nginxConfdPath, name+".conf")
}

// GetSecretFileName constructs the filename for a Secret name
func (nginx *Controller) GetSecretFileName(name string) string {
	return path.Join(nginx.nginxSecretsPath, name)
}

func (nginx *Controller) getNginxCommand(cmd string) string {
	return fmt.Sprint(nginx.nginxBinaryPath, " -s ", cmd)
}

// Reload reloads NGINX
func (nginx *Controller) Reload() error {
	if nginx.local {
		glog.V(3).Info("local - skipping nginx reload")
		return nil
	}
	// write a new config version
	nginx.ConfigVersion++
	nginx.UpdateConfigVersionFile()

	glog.V(3).Infof("Reloading nginx. configVersion: %v", nginx.ConfigVersion)

	reloadCmd := nginx.getNginxCommand("reload")
	if err := shellOut(reloadCmd); err != nil {
		return fmt.Errorf("nginx reload failed: %v", err)
	}
	err := nginx.verifyClient.WaitForCorrectVersion(nginx.ConfigVersion)
	if err != nil {
		return fmt.Errorf("could not get newest config version: %v", err)
	}

	return nil
}

// Start starts NGINX
func (nginx *Controller) Start(done chan error) {
	glog.V(3).Info("Starting nginx")

	if nginx.local {
		return
	}

	cmd := exec.Command(nginx.nginxBinaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		glog.Fatalf("Failed to start nginx: %v", err)
	}

	go func() {
		done <- cmd.Wait()
	}()

	err := nginx.verifyClient.WaitForCorrectVersion(nginx.ConfigVersion)
	if err != nil {
		glog.Fatalf("Could not get newest config version: %v", err)
	}
}

// Quit shutdowns NGINX gracefully
func (nginx *Controller) Quit() {
	if !nginx.local {
		quitCmd := nginx.getNginxCommand("quit")
		if err := shellOut(quitCmd); err != nil {
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
func (nginx *Controller) UpdateMainConfigFile(cfg []byte) {
	filename := "/etc/nginx/nginx.conf"
	glog.V(3).Infof("Writing NGINX conf to %v", filename)

	if bool(glog.V(3)) || nginx.local {
		glog.Info(string(cfg))
	}

	if !nginx.local {
		err := createFileAndWrite(filename, cfg)
		if err != nil {
			glog.Fatalf("Failed to write NGINX conf: %v", err)
		}
	}
	glog.V(3).Infof("The main NGINX config file has been updated")
}

// UpdateIngressConfigFile writes the Ingress configuration file to the filesystem
func (nginx *Controller) UpdateIngressConfigFile(name string, cfg []byte) {
	filename := nginx.getIngressNginxConfigFileName(name)
	glog.V(3).Infof("Writing Ingress conf to %v", filename)

	if bool(glog.V(3)) || nginx.local {
		glog.Info(string(cfg))
	}

	if !nginx.local {
		err := createFileAndWrite(filename, cfg)
		if err != nil {
			glog.Fatalf("Failed to write Ingress conf: %v", err)
		}
	}
	glog.V(3).Infof("The Ingress config file has been updated")
}

// UpdateConfigVersionFile writes the config version file.
func (nginx *Controller) UpdateConfigVersionFile() {
	cfg, err := nginx.verifyConfigGenerator.GenerateVersionConfig(nginx.ConfigVersion)
	if err != nil {
		glog.Fatalf("Error generating config version content: %v", err)
	}

	filename := "/etc/nginx/config-version.conf"
	tempname := "/etc/nginx/config-version.conf.tmp"
	glog.V(3).Infof("Writing config version to %v", filename)

	if bool(glog.V(3)) || nginx.local {
		glog.Info(string(cfg))
	}

	if !nginx.local {
		err := createFileAndWrite(tempname, cfg)
		if err != nil {
			glog.Fatalf("Failed to write version config file: %v", err)
		}

		err = os.Rename(tempname, filename)
		if err != nil {
			glog.Fatalf("failed to rename version config file: %v", err)
		}
	}
	glog.V(3).Infof("The config version file has been updated.")
}

func createFileAndWrite(name string, b []byte) error {
	w, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("Failed to open %v: %v", name, err)
	}

	defer w.Close()

	_, err = w.Write(b)
	if err != nil {
		return fmt.Errorf("Failed to write to %v: %v", name, err)
	}

	return nil
}

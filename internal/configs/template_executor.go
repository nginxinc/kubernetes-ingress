package configs

import (
	"bytes"
	"path"
	"text/template"
)

// TemplateExecutor executes NGINX configuration templates
type TemplateExecutor struct {
	HealthStatus                   bool
	NginxStatus                    bool
	NginxStatusAllowCIDRs          []string
	NginxStatusPort                int
	StubStatusOverUnixSocketForOSS bool
	mainTemplate                   *template.Template
	ingressTemplate                *template.Template
}

// NewTemplateExecutor creates a TemplateExecutor
func NewTemplateExecutor(mainTemplatePath string, ingressTemplatePath string, healthStatus bool, nginxStatus bool, nginxStatusAllowCIDRs []string, nginxStatusPort int, stubStatusOverUnixSocketForOSS bool) (*TemplateExecutor, error) {
	// template name must be the base name of the template file https://golang.org/pkg/text/template/#Template.ParseFiles
	nginxTemplate, err := template.New(path.Base(mainTemplatePath)).ParseFiles(mainTemplatePath)
	if err != nil {
		return nil, err
	}

	ingressTemplate, err := template.New(path.Base(ingressTemplatePath)).ParseFiles(ingressTemplatePath)
	if err != nil {
		return nil, err
	}

	return &TemplateExecutor{
		mainTemplate:                   nginxTemplate,
		ingressTemplate:                ingressTemplate,
		HealthStatus:                   healthStatus,
		NginxStatus:                    nginxStatus,
		NginxStatusAllowCIDRs:          nginxStatusAllowCIDRs,
		NginxStatusPort:                nginxStatusPort,
		StubStatusOverUnixSocketForOSS: stubStatusOverUnixSocketForOSS,
	}, nil
}

// UpdateMainTemplate updates the main NGINX template
func (te *TemplateExecutor) UpdateMainTemplate(templateString *string) error {
	newTemplate, err := template.New("nginxTemplate").Parse(*templateString)
	if err != nil {
		return err
	}
	te.mainTemplate = newTemplate

	return nil
}

// UpdateIngressTemplate updates the ingress template
func (te *TemplateExecutor) UpdateIngressTemplate(templateString *string) error {
	newTemplate, err := template.New("ingressTemplate").Parse(*templateString)
	if err != nil {
		return err
	}
	te.ingressTemplate = newTemplate

	return nil
}

// ExecuteMainConfigTemplate generates the content of the main NGINX configuration file
func (te *TemplateExecutor) ExecuteMainConfigTemplate(cfg *MainConfig) ([]byte, error) {
	cfg.HealthStatus = te.HealthStatus
	cfg.NginxStatus = te.NginxStatus
	cfg.NginxStatusAllowCIDRs = te.NginxStatusAllowCIDRs
	cfg.NginxStatusPort = te.NginxStatusPort
	cfg.StubStatusOverUnixSocketForOSS = te.StubStatusOverUnixSocketForOSS

	var configBuffer bytes.Buffer
	err := te.mainTemplate.Execute(&configBuffer, cfg)

	return configBuffer.Bytes(), err
}

// ExecuteIngressConfigTemplate generates the content of a NGINX configuration file for an Ingress resource
func (te *TemplateExecutor) ExecuteIngressConfigTemplate(cfg *IngressNginxConfig) ([]byte, error) {
	var configBuffer bytes.Buffer
	err := te.ingressTemplate.Execute(&configBuffer, cfg)

	return configBuffer.Bytes(), err
}

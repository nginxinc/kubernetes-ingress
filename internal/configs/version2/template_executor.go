package version2

import (
	"bytes"
	"path"
	"text/template"
)

// TemplateExecutor executes NGINX configuration templates.
type TemplateExecutor struct {
	virtualServerTemplate   *template.Template
	transportServerTemplate *template.Template
}

// NewTemplateExecutor creates a TemplateExecutor.
func NewTemplateExecutor(virtualServerTemplatePath string, transportServerTemplatePath string) (*TemplateExecutor, error) {
	// template names  must be the base name of the template file https://golang.org/pkg/text/template/#Template.ParseFiles

	vsTemplate, err := template.New(path.Base(virtualServerTemplatePath)).ParseFiles(virtualServerTemplatePath)
	if err != nil {
		return nil, err
	}

	tsTemplate, err := template.New(path.Base(transportServerTemplatePath)).ParseFiles(transportServerTemplatePath)
	if err != nil {
		return nil, err
	}

	return &TemplateExecutor{
		virtualServerTemplate:   vsTemplate,
		transportServerTemplate: tsTemplate,
	}, nil
}

// ExecuteVirtualServerTemplate generates the content of an NGINX configuration file for a VirtualServer resource.
func (te *TemplateExecutor) ExecuteVirtualServerTemplate(cfg *VirtualServerConfig) ([]byte, error) {
	var configBuffer bytes.Buffer
	err := te.virtualServerTemplate.Execute(&configBuffer, cfg)

	return configBuffer.Bytes(), err
}

// ExecuteTransportServerTemplate generates the content of an NGINX configuration file for a TransportServer resource.
func (te *TemplateExecutor) ExecuteTransportServerTemplate(cfg *TransportServerConfig) ([]byte, error) {
	var configBuffer bytes.Buffer
	err := te.transportServerTemplate.Execute(&configBuffer, cfg)

	return configBuffer.Bytes(), err
}

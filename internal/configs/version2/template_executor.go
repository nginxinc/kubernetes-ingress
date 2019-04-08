package version2

import (
	"bytes"
	"path"
	"text/template"
)

// TemplateExecutor executes NGINX configuration templates.
type TemplateExecutor struct {
	virtualServerTemplate *template.Template
}

// NewTemplateExecutor creates a TemplateExecutor.
func NewTemplateExecutor(virtualServerTemplatePath string) (*TemplateExecutor, error) {
	// template name must be the base name of the template file https://golang.org/pkg/text/template/#Template.ParseFiles
	vsTemplate, err := template.New(path.Base(virtualServerTemplatePath)).ParseFiles(virtualServerTemplatePath)
	if err != nil {
		return nil, err
	}

	return &TemplateExecutor{
		virtualServerTemplate: vsTemplate,
	}, nil
}

// ExecuteVirtualServerTemplate generates the content of an NGINX configuration file for a VirtualServer resource.
func (te *TemplateExecutor) ExecuteVirtualServerTemplate(cfg *VirtualServerConfig) ([]byte, error) {
	var configBuffer bytes.Buffer
	err := te.virtualServerTemplate.Execute(&configBuffer, cfg)

	return configBuffer.Bytes(), err
}

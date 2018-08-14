package verify

import (
	"bytes"
	"html/template"
)

const configVersionTemplateString = `server {
    listen unix:/var/run/nginx-config-version.sock;
    access_log off;

    location /configVersion {
        return 200 {{.ConfigVersion}};
    }
}
map $http_x_expected_config_version $config_version_mismatch {
	"{{.ConfigVersion}}" "";
	default "mismatch";
}`

// ConfigGenerator handles generating and writing the config version file.
type ConfigGenerator struct {
	configVersionTemplate *template.Template
}

// NewConfigGenerator builds a new ConfigWriter - primarily parsing the config version template.
func NewConfigGenerator() (*ConfigGenerator, error) {
	configVersionTemplate, err := template.New("configVersionTemplate").Parse(configVersionTemplateString)
	if err != nil {
		return nil, err
	}
	return &ConfigGenerator{
		configVersionTemplate: configVersionTemplate,
	}, nil
}

// GenerateVersionConfig generates the config version file.
func (c *ConfigGenerator) GenerateVersionConfig(configVersion int) ([]byte, error) {
	var configBuffer bytes.Buffer
	templateValues := struct {
		ConfigVersion int
	}{
		configVersion,
	}
	err := c.configVersionTemplate.Execute(&configBuffer, templateValues)
	if err != nil {
		return nil, err
	}

	return configBuffer.Bytes(), nil
}

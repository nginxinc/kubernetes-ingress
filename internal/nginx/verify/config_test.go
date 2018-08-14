package verify

import (
	"strings"
	"testing"
)

func TestConfigWriter(t *testing.T) {
	cw, err := NewConfigGenerator()
	if err != nil {
		t.Fatalf("error instantiating ConfigWriter: %v", err)
	}
	config, err := cw.GenerateVersionConfig(1)
	if err != nil {
		t.Errorf("error generating version config: %v", err)
	}
	if !strings.Contains(string(config), "configVersion") {
		t.Errorf("configVersion endpoint not set. config contents: %v", string(config))
	}
}

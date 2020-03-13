package validation

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func createGlobalConfigurationValidator() *GlobalConfigurationValidator {
	return &GlobalConfigurationValidator{}
}

func TestValidateGlobalConfiguration(t *testing.T) {
	globalConfiguration := v1alpha1.GlobalConfiguration{
		Spec: v1alpha1.GlobalConfigurationSpec{
			Listeners: []v1alpha1.Listener{
				{
					Name:     "tcp-listener",
					Port:     53,
					Protocol: "TCP",
				},
				{
					Name:     "udp-listener",
					Port:     53,
					Protocol: "UDP",
				},
			},
		},
	}

	gcv := createGlobalConfigurationValidator()

	err := gcv.ValidateGlobalConfiguration(&globalConfiguration)
	if err != nil {
		t.Errorf("ValidateGlobalConfiguration() returned error %v for valid input", err)
	}
}

func TestValidateListenerPort(t *testing.T) {
	forbiddenListenerPorts := map[int]bool{
		1234: true,
	}

	gcv := &GlobalConfigurationValidator{
		forbiddenListenerPorts: forbiddenListenerPorts,
	}

	allErrs := gcv.validateListenerPort(5555, field.NewPath("port"))
	if len(allErrs) > 0 {
		t.Errorf("validateListenerPort() returned errors %v for valid input", allErrs)
	}

	allErrs = gcv.validateListenerPort(1234, field.NewPath("port"))
	if len(allErrs) == 0 {
		t.Errorf("validateListenerPort() returned no errors for invalid input")
	}
}

func TestValidateListeners(t *testing.T) {
	listeners := []v1alpha1.Listener{
		{
			Name:     "tcp-listener",
			Port:     53,
			Protocol: "TCP",
		},
		{
			Name:     "udp-listener",
			Port:     53,
			Protocol: "UDP",
		},
	}

	gcv := createGlobalConfigurationValidator()

	allErrs := gcv.validateListeners(listeners, field.NewPath("listeners"))
	if len(allErrs) > 0 {
		t.Errorf("validateListeners() returned errors %v for valid intput", allErrs)
	}
}

func TestValidateListenersFails(t *testing.T) {
	tests := []struct {
		listeners []v1alpha1.Listener
		msg       string
	}{
		{
			listeners: []v1alpha1.Listener{
				{
					Name:     "tcp-listener",
					Port:     2201,
					Protocol: "TCP",
				},
				{
					Name:     "tcp-listener",
					Port:     2202,
					Protocol: "TCP",
				},
			},
			msg: "duplicated name",
		},
		{
			listeners: []v1alpha1.Listener{
				{
					Name:     "tcp-listener-1",
					Port:     2201,
					Protocol: "TCP",
				},
				{
					Name:     "tcp-listener-2",
					Port:     2201,
					Protocol: "TCP",
				},
			},
			msg: "duplicated port/protocol combination",
		},
	}

	gcv := createGlobalConfigurationValidator()

	for _, test := range tests {
		allErrs := gcv.validateListeners(test.listeners, field.NewPath("listeners"))
		if len(allErrs) == 0 {
			t.Errorf("validateListeners() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateListener(t *testing.T) {
	listener := v1alpha1.Listener{
		Name:     "tcp-listener",
		Port:     53,
		Protocol: "TCP",
	}

	gcv := createGlobalConfigurationValidator()

	allErrs := gcv.validateListener(listener, field.NewPath("listener"))
	if len(allErrs) > 0 {
		t.Errorf("validateListener() returned errors %v for valid intput", allErrs)
	}
}

func TestValidateListenerFails(t *testing.T) {
	tests := []struct {
		Listener v1alpha1.Listener
		msg      string
	}{
		{
			Listener: v1alpha1.Listener{
				Name:     "@",
				Port:     2201,
				Protocol: "TCP",
			},
			msg: "invalid name",
		},
		{
			Listener: v1alpha1.Listener{
				Name:     "tcp-listener",
				Port:     -1,
				Protocol: "TCP",
			},
			msg: "invalid port",
		},
		{
			Listener: v1alpha1.Listener{
				Name:     "name",
				Port:     2201,
				Protocol: "IP",
			},
			msg: "invalid protocol",
		},
		{
			Listener: v1alpha1.Listener{
				Name:     "tls-passthrough",
				Port:     2201,
				Protocol: "TCP",
			},
			msg: "name of a built-in listener",
		},
	}

	gcv := createGlobalConfigurationValidator()

	for _, test := range tests {
		allErrs := gcv.validateListener(test.Listener, field.NewPath("listener"))
		if len(allErrs) == 0 {
			t.Errorf("validateListener() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestGeneratePortProtocolKey(t *testing.T) {
	port := 53
	protocol := "UDP"

	expected := "53/UDP"

	result := generatePortProtocolKey(port, protocol)

	if result != expected {
		t.Errorf("generatePortProtocolKey(%d, %q) returned %q but expected %q", port, protocol, result, expected)
	}
}

package validation

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func createGlobalConfigurationValidator() *GlobalConfigurationValidator {
	return &GlobalConfigurationValidator{}
}

func TestValidateGlobalConfiguration(t *testing.T) {
	t.Parallel()
	globalConfiguration := conf_v1.GlobalConfiguration{
		Spec: conf_v1.GlobalConfigurationSpec{
			Listeners: []conf_v1.Listener{
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
	t.Parallel()
	forbiddenListenerPorts := map[int]bool{
		1234: true,
	}

	gcv := &GlobalConfigurationValidator{
		forbiddenListenerPorts: forbiddenListenerPorts,
	}

	allErrs := gcv.validateListenerPort("sample-listener", 5555, field.NewPath("port"))
	if len(allErrs) > 0 {
		t.Errorf("validateListenerPort() returned errors %v for valid input", allErrs)
	}

	allErrs = gcv.validateListenerPort("sample-listener", 1234, field.NewPath("port"))
	if len(allErrs) == 0 {
		t.Errorf("validateListenerPort() returned no errors for invalid input")
	}
}

func TestValidateListeners(t *testing.T) {
	t.Parallel()
	listeners := []conf_v1.Listener{
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

	_, allErrs := gcv.getValidListeners(listeners, field.NewPath("listeners"))
	if len(allErrs) > 0 {
		t.Errorf("validateListeners() returned errors %v for valid input", allErrs)
	}
}

func TestValidateListenersFails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		listeners     []conf_v1.Listener
		wantListeners []conf_v1.Listener
		msg           string
	}{
		{
			listeners: []conf_v1.Listener{
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
			wantListeners: []conf_v1.Listener{
				{
					Name:     "tcp-listener",
					Port:     2201,
					Protocol: "TCP",
				},
			},
			msg: "duplicated name",
		},
		{
			listeners: []conf_v1.Listener{
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
			wantListeners: []conf_v1.Listener{
				{
					Name:     "tcp-listener-1",
					Port:     2201,
					Protocol: "TCP",
				},
			},
			msg: "duplicated port/protocol combination",
		},
		{
			listeners: []conf_v1.Listener{
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
				{
					Name:     "udp-listener-3",
					Port:     2201,
					Protocol: "UDP",
				},
			},
			wantListeners: []conf_v1.Listener{
				{
					Name:     "tcp-listener-1",
					Port:     2201,
					Protocol: "TCP",
				},
				{
					Name:     "udp-listener-3",
					Port:     2201,
					Protocol: "UDP",
				},
			},
			msg: "duplicated port/protocol combination",
		},
	}

	gcv := createGlobalConfigurationValidator()

	for _, test := range tests {
		listeners, allErrs := gcv.getValidListeners(test.listeners, field.NewPath("listeners"))
		if diff := cmp.Diff(listeners, test.wantListeners); diff != "" {
			t.Errorf("getValidListeners() returned unexpected result for the case of %s:(-want +got), %s", test.msg, diff)
		}

		if len(allErrs) == 0 {
			t.Errorf("validateListeners() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateListener(t *testing.T) {
	t.Parallel()
	listener := conf_v1.Listener{
		Name:     "tcp-listener",
		Port:     53,
		Protocol: "TCP",
	}

	gcv := createGlobalConfigurationValidator()

	allErrs := gcv.validateListener(listener, field.NewPath("listener"))
	if len(allErrs) > 0 {
		t.Errorf("validateListener() returned errors %v for valid input", allErrs)
	}
}

func TestValidateListenerFails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		Listener conf_v1.Listener
		msg      string
	}{
		{
			Listener: conf_v1.Listener{
				Name:     "@",
				Port:     2201,
				Protocol: "TCP",
			},
			msg: "invalid name",
		},
		{
			Listener: conf_v1.Listener{
				Name:     "tcp-listener",
				Port:     -1,
				Protocol: "TCP",
			},
			msg: "invalid port",
		},
		{
			Listener: conf_v1.Listener{
				Name:     "name",
				Port:     2201,
				Protocol: "IP",
			},
			msg: "invalid protocol",
		},
		{
			Listener: conf_v1.Listener{
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
	t.Parallel()
	port := 53
	protocol := "UDP"

	expected := "53/UDP"

	result := generatePortProtocolKey(port, protocol)

	if result != expected {
		t.Errorf("generatePortProtocolKey(%d, %q) returned %q but expected %q", port, protocol, result, expected)
	}
}

func TestValidateListenerProtocol_FailsOnInvalidInput(t *testing.T) {
	t.Parallel()
	invalidProtocols := []string{
		"",
		"udp",
		"UDP ",
	}

	for _, p := range invalidProtocols {
		allErrs := validateListenerProtocol(p, field.NewPath("protocol"))
		if len(allErrs) == 0 {
			t.Errorf("validateListenerProtocol(%q) returned no errors for invalid input", p)
		}
	}
}

func TestValidateListenerProtocol_PassesOnValidInput(t *testing.T) {
	t.Parallel()
	validProtocols := []string{
		"TCP",
		"HTTP",
		"UDP",
	}

	for _, p := range validProtocols {
		allErrs := validateListenerProtocol(p, field.NewPath("protocol"))
		if len(allErrs) != 0 {
			t.Errorf("validateListenerProtocol(%q) returned errors for valid input", p)
		}
	}
}

func TestValidateListenerProtocol_PassesOnHttpListenerUsingDiffPortToTCPAndUDPListenerWithTCPAndUDPDefinedFirst(t *testing.T) {
	t.Parallel()
	listeners := []conf_v1.Listener{
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
		{
			Name:     "http-listener",
			Port:     63,
			Protocol: "HTTP",
		},
	}

	gcv := createGlobalConfigurationValidator()

	_, allErrs := gcv.getValidListeners(listeners, field.NewPath("listeners"))
	if len(allErrs) > 0 {
		t.Errorf("validateListeners() returned errors %v for valid input", allErrs)
	}
}

func TestValidateListenerProtocol_PassesOnHttpListenerUsingDiffPortToTCPAndUDPListenerWithHTTPDefinedFirst(t *testing.T) {
	t.Parallel()
	listeners := []conf_v1.Listener{
		{
			Name:     "http-listener",
			Port:     63,
			Protocol: "HTTP",
		},
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

	_, allErrs := gcv.getValidListeners(listeners, field.NewPath("listeners"))
	if len(allErrs) > 0 {
		t.Errorf("validateListeners() returned errors %v for valid input", allErrs)
	}
}

func TestValidateListenerProtocol_FailsOnHttpListenerUsingSamePortAsTCPListener(t *testing.T) {
	t.Parallel()
	listeners := []conf_v1.Listener{
		{
			Name:     "tcp-listener",
			Port:     53,
			Protocol: "TCP",
		},
		{
			Name:     "http-listener",
			Port:     53,
			Protocol: "HTTP",
		},
	}
	wantListeners := []conf_v1.Listener{
		{
			Name:     "tcp-listener",
			Port:     53,
			Protocol: "TCP",
		},
	}

	gcv := createGlobalConfigurationValidator()

	listeners, allErrs := gcv.getValidListeners(listeners, field.NewPath("listeners"))
	if diff := cmp.Diff(listeners, wantListeners); diff != "" {
		t.Errorf("getValidListeners() returned unexpected result: (-want +got):\n%s", diff)
	}
	if len(allErrs) == 0 {
		t.Errorf("validateListeners() returned no errors %v for invalid input", allErrs)
	}
}

func TestValidateListenerProtocol_FailsOnHttpListenerUsingSamePortAsUDPListener(t *testing.T) {
	t.Parallel()
	listeners := []conf_v1.Listener{
		{
			Name:     "udp-listener",
			Port:     53,
			Protocol: "UDP",
		},
		{
			Name:     "http-listener",
			Port:     53,
			Protocol: "HTTP",
		},
	}
	wantListeners := []conf_v1.Listener{
		{
			Name:     "udp-listener",
			Port:     53,
			Protocol: "UDP",
		},
	}
	gcv := createGlobalConfigurationValidator()
	listeners, allErrs := gcv.getValidListeners(listeners, field.NewPath("listeners"))
	if diff := cmp.Diff(listeners, wantListeners); diff != "" {
		t.Errorf("getValidListeners() returned unexpected result: (-want +got):\n%s", diff)
	}
	if len(allErrs) == 0 {
		t.Errorf("validateListeners() returned no errors %v for invalid input", allErrs)
	}
}

func TestValidateListenerProtocol_FailsOnHttpListenerUsingSamePortAsTCPAndUDPListener(t *testing.T) {
	t.Parallel()
	listeners := []conf_v1.Listener{
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
		{
			Name:     "http-listener",
			Port:     53,
			Protocol: "HTTP",
		},
	}
	wantListeners := []conf_v1.Listener{
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

	listeners, allErrs := gcv.getValidListeners(listeners, field.NewPath("listeners"))
	if diff := cmp.Diff(listeners, wantListeners); diff != "" {
		t.Errorf("getValidListeners() returned unexpected result: (-want +got):\n%s", diff)
	}
	if len(allErrs) == 0 {
		t.Errorf("validateListeners() returned no errors %v for invalid input", allErrs)
	}
}

func TestValidateListenerProtocol_FailsOnTCPListenerUsingSamePortAsHTTPListener(t *testing.T) {
	t.Parallel()
	listeners := []conf_v1.Listener{
		{
			Name:     "http-listener",
			Port:     53,
			Protocol: "HTTP",
		},
		{
			Name:     "tcp-listener",
			Port:     53,
			Protocol: "TCP",
		},
	}
	wantListeners := []conf_v1.Listener{
		{
			Name:     "http-listener",
			Port:     53,
			Protocol: "HTTP",
		},
	}

	gcv := createGlobalConfigurationValidator()

	listeners, allErrs := gcv.getValidListeners(listeners, field.NewPath("listeners"))
	if diff := cmp.Diff(listeners, wantListeners); diff != "" {
		t.Errorf("getValidListeners() returned unexpected result: (-want +got):\n%s", diff)
	}
	if len(allErrs) == 0 {
		t.Errorf("validateListeners() returned no errors %v for invalid input", allErrs)
	}
}

func TestValidateListenerProtocol_FailsOnUDPListenerUsingSamePortAsHTTPListener(t *testing.T) {
	t.Parallel()
	listeners := []conf_v1.Listener{
		{
			Name:     "http-listener",
			Port:     53,
			Protocol: "HTTP",
		},
		{
			Name:     "udp-listener",
			Port:     53,
			Protocol: "UDP",
		},
	}
	wantListeners := []conf_v1.Listener{
		{
			Name:     "http-listener",
			Port:     53,
			Protocol: "HTTP",
		},
	}

	gcv := createGlobalConfigurationValidator()

	listeners, allErrs := gcv.getValidListeners(listeners, field.NewPath("listeners"))
	if diff := cmp.Diff(listeners, wantListeners); diff != "" {
		t.Errorf("getValidListeners() returned unexpected result: (-want +got):\n%s", diff)
	}
	if len(allErrs) == 0 {
		t.Errorf("validateListeners() returned no errors %v for invalid input", allErrs)
	}
}

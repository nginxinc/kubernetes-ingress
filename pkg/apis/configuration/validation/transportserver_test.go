package validation

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateTransportServer(t *testing.T) {
	ts := v1alpha1.TransportServer{
		Spec: v1alpha1.TransportServerSpec{
			Listener: v1alpha1.TransportServerListener{
				Name:     "tcp-listener",
				Protocol: "TCP",
			},
			Upstreams: []v1alpha1.Upstream{
				{
					Name:    "upstream1",
					Service: "test-1",
					Port:    5501,
				},
			},
			Action: &v1alpha1.Action{
				Pass: "upstream1",
			},
		},
	}

	err := ValidateTransportServer(&ts)
	if err != nil {
		t.Errorf("ValidateTransportServer() returned error %v for valid input", err)
	}
}

func TestValidateTransportServerFails(t *testing.T) {
	ts := v1alpha1.TransportServer{
		Spec: v1alpha1.TransportServerSpec{
			Listener: v1alpha1.TransportServerListener{
				Name:     "tcp-listener",
				Protocol: "TCP",
			},
			Upstreams: []v1alpha1.Upstream{
				{
					Name:    "upstream1",
					Service: "test-1",
					Port:    5501,
				},
			},
			Action: nil,
		},
	}

	err := ValidateTransportServer(&ts)
	if err == nil {
		t.Errorf("ValidateTransportServer() returned no error for invalid input")
	}
}

func TestValidateTransportServerUpstreams(t *testing.T) {
	tests := []struct {
		upstreams             []v1alpha1.Upstream
		expectedUpstreamNames sets.String
		msg                   string
	}{
		{
			upstreams:             []v1alpha1.Upstream{},
			expectedUpstreamNames: sets.String{},
			msg:                   "no upstreams",
		},
		{
			upstreams: []v1alpha1.Upstream{
				{
					Name:    "upstream1",
					Service: "test-1",
					Port:    80,
				},
				{
					Name:    "upstream2",
					Service: "test-2",
					Port:    80,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
				"upstream2": {},
			},
			msg: "2 valid upstreams",
		},
	}

	for _, test := range tests {
		allErrs, resultUpstreamNames := validateTransportServerUpstreams(test.upstreams, field.NewPath("upstreams"))
		if len(allErrs) > 0 {
			t.Errorf("validateTransportServerUpstreams() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
		if !resultUpstreamNames.Equal(test.expectedUpstreamNames) {
			t.Errorf("validateTransportServerUpstreams() returned %v expected %v for the case of %s", resultUpstreamNames, test.expectedUpstreamNames, test.msg)
		}
	}
}

func TestValidateTransportServerUpstreamsFails(t *testing.T) {
	tests := []struct {
		upstreams             []v1alpha1.Upstream
		expectedUpstreamNames sets.String
		msg                   string
	}{
		{
			upstreams: []v1alpha1.Upstream{
				{
					Name:    "@upstream1",
					Service: "test-1",
					Port:    80,
				},
			},
			expectedUpstreamNames: sets.String{},
			msg:                   "invalid upstream name",
		},
		{
			upstreams: []v1alpha1.Upstream{
				{
					Name:    "upstream1",
					Service: "@test-1",
					Port:    80,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid service",
		},
		{
			upstreams: []v1alpha1.Upstream{
				{
					Name:    "upstream1",
					Service: "test-1",
					Port:    -80,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "invalid port",
		},
		{
			upstreams: []v1alpha1.Upstream{
				{
					Name:    "upstream1",
					Service: "test-1",
					Port:    80,
				},
				{
					Name:    "upstream1",
					Service: "test-2",
					Port:    80,
				},
			},
			expectedUpstreamNames: map[string]sets.Empty{
				"upstream1": {},
			},
			msg: "duplicated upstreams",
		},
	}

	for _, test := range tests {
		allErrs, resultUpstreamNames := validateTransportServerUpstreams(test.upstreams, field.NewPath("upstreams"))
		if len(allErrs) == 0 {
			t.Errorf("validateTransportServerUpstreams() returned errors no errors for the case of %s", test.msg)
		}
		if !resultUpstreamNames.Equal(test.expectedUpstreamNames) {
			t.Errorf("validateTransportServerUpstreams() returned %v expected %v for the case of %s", resultUpstreamNames, test.expectedUpstreamNames, test.msg)
		}
	}
}

func TestValidateListenerProtocol(t *testing.T) {
	validProtocols := []string{
		"TCP",
		"UDP",
	}

	for _, p := range validProtocols {
		allErrs := validateListenerProtocol(p, field.NewPath("protocol"))
		if len(allErrs) > 0 {
			t.Errorf("validateListenerProtocol(%q) returned errors %v for valid input", p, allErrs)
		}
	}

	invalidProtocols := []string{
		"",
		"HTTP",
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

func TestValidateUpstreamParameters(t *testing.T) {
	tests := []struct {
		parameters *v1alpha1.UpstreamParameters
		msg        string
	}{
		{
			parameters: nil,
			msg:        "nil parameters",
		},
		{
			parameters: &v1alpha1.UpstreamParameters{},
			msg:        "Non-nil parameters",
		},
	}

	for _, test := range tests {
		allErrs := validateTransportServerUpstreamParameters(test.parameters, field.NewPath("upstreamParameters"), "UDP")
		if len(allErrs) > 0 {
			t.Errorf("validateTransportServerUpstreamParameters() returned errors %v for valid input for the case of %s", allErrs, test.msg)
		}
	}
}

func TestValidateUDPUpstreamParameter(t *testing.T) {
	validInput := []struct {
		parameter *int
		protocol  string
	}{
		{
			parameter: nil,
			protocol:  "TCP",
		},
		{
			parameter: nil,
			protocol:  "UDP",
		},
		{
			parameter: createPointerFromInt(0),
			protocol:  "UDP",
		},
		{
			parameter: createPointerFromInt(1),
			protocol:  "UDP",
		},
	}

	for _, input := range validInput {
		allErrs := validateUDPUpstreamParameter(input.parameter, field.NewPath("parameter"), input.protocol)
		if len(allErrs) > 0 {
			t.Errorf("validateUDPUpstreamParameter(%v, %q) returned errors %v for valid input", input.parameter, input.protocol, allErrs)
		}
	}
}

func TestValidateUDPUpstreamParameterFails(t *testing.T) {
	invalidInput := []struct {
		parameter *int
		protocol  string
	}{
		{
			parameter: createPointerFromInt(0),
			protocol:  "TCP",
		},
		{
			parameter: createPointerFromInt(-1),
			protocol:  "UDP",
		},
	}

	for _, input := range invalidInput {
		allErrs := validateUDPUpstreamParameter(input.parameter, field.NewPath("parameter"), input.protocol)
		if len(allErrs) == 0 {
			t.Errorf("validateUDPUpstreamParameter(%v, %q) returned no errors for invalid input", input.parameter, input.protocol)
		}
	}
}

func TestValidateTransportServerAction(t *testing.T) {
	upstreamNames := map[string]sets.Empty{
		"test": {},
	}

	action := &v1alpha1.Action{
		Pass: "test",
	}

	allErrs := validateTransportServerAction(action, field.NewPath("action"), upstreamNames)
	if len(allErrs) > 0 {
		t.Errorf("validateTransportServerAction() returned errors %v for valid input", allErrs)
	}
}

func TestValidateTransportServerActionFails(t *testing.T) {
	upstreamNames := map[string]sets.Empty{}

	tests := []struct {
		action *v1alpha1.Action
		msg    string
	}{
		{
			action: &v1alpha1.Action{
				Pass: "",
			},
			msg: "missing pass field",
		},
		{
			action: &v1alpha1.Action{
				Pass: "non-existing",
			},
			msg: "pass references a non-existing upstream",
		},
	}

	for _, test := range tests {
		allErrs := validateTransportServerAction(test.action, field.NewPath("action"), upstreamNames)
		if len(allErrs) == 0 {
			t.Errorf("validateTransportServerAction() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

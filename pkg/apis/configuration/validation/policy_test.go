package validation

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidatePolicy(t *testing.T) {
	policy := &v1alpha1.Policy{
		Spec: v1alpha1.PolicySpec{
			AccessControl: &v1alpha1.AccessControl{
				Allow: []string{"127.0.0.1"},
			},
		},
	}

	err := ValidatePolicy(policy)
	if err != nil {
		t.Errorf("ValidatePolicy() returned error %v for valid input", err)
	}
}

func TestValidatePolicyFails(t *testing.T) {
	policy := &v1alpha1.Policy{
		Spec: v1alpha1.PolicySpec{},
	}

	err := ValidatePolicy(policy)
	if err == nil {
		t.Errorf("ValidatePolicy() returned no error for invalid input")
	}
}

func TestValidateAccessControl(t *testing.T) {
	validInput := []*v1alpha1.AccessControl{
		{
			Allow: []string{},
		},
		{
			Allow: []string{"127.0.0.1"},
		},
		{
			Deny: []string{},
		},
		{
			Deny: []string{"127.0.0.1"},
		},
	}

	for _, input := range validInput {
		allErrs := validateAccessControl(input, field.NewPath("accessControl"))
		if len(allErrs) > 0 {
			t.Errorf("validateAccessControl(%+v) returned errors %v for valid input", input, allErrs)
		}
	}
}

func TestValidateAccessControlFails(t *testing.T) {
	tests := []struct {
		accessControl *v1alpha1.AccessControl
		msg           string
	}{
		{
			accessControl: &v1alpha1.AccessControl{
				Allow: nil,
				Deny:  nil,
			},
			msg: "neither allow nor deny is defined",
		},
		{
			accessControl: &v1alpha1.AccessControl{
				Allow: []string{},
				Deny:  []string{},
			},
			msg: "both allow and deny are defined",
		},
		{
			accessControl: &v1alpha1.AccessControl{
				Allow: []string{"invalid"},
			},
			msg: "invalid allow",
		},
		{
			accessControl: &v1alpha1.AccessControl{
				Deny: []string{"invalid"},
			},
			msg: "invalid deny",
		},
	}

	for _, test := range tests {
		allErrs := validateAccessControl(test.accessControl, field.NewPath("accessControl"))
		if len(allErrs) == 0 {
			t.Errorf("validateAccessControl() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateIPorCIDR(t *testing.T) {
	validInput := []string{
		"192.168.1.1",
		"192.168.1.0/24",
		"2001:0db8::1",
		"2001:0db8::/32",
	}

	for _, input := range validInput {
		allErrs := validateIPorCIDR(input, field.NewPath("ipOrCIDR"))
		if len(allErrs) > 0 {
			t.Errorf("validateIPorCIDR(%q) returned errors %v for valid input", input, allErrs)
		}
	}

	invalidInput := []string{
		"localhost",
		"192.168.1.0/",
		"2001:0db8:::1",
		"2001:0db8::/",
	}

	for _, input := range invalidInput {
		allErrs := validateIPorCIDR(input, field.NewPath("ipOrCIDR"))
		if len(allErrs) == 0 {
			t.Errorf("validateIPorCIDR(%q) returned no errors for invalid input", input)
		}
	}
}

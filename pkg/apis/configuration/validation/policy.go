package validation

import (
	"net"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidatePolicy validates a Policy.
func ValidatePolicy(policy *v1alpha1.Policy) error {
	allErrs := validatePolicySpec(&policy.Spec, field.NewPath("spec"))
	return allErrs.ToAggregate()
}

func validatePolicySpec(spec *v1alpha1.PolicySpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	fieldCount := 0

	if spec.AccessControl != nil {
		allErrs = append(allErrs, validateAccessControl(spec.AccessControl, fieldPath.Child("accessControl"))...)
		fieldCount++
	}

	if fieldCount != 1 {
		allErrs = append(allErrs, field.Invalid(fieldPath, "", "must specify exactly one of: `accessControl`"))
	}

	return allErrs
}

func validateAccessControl(accessControl *v1alpha1.AccessControl, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	fieldCount := 0

	if accessControl.Allow != nil {
		for i, ipOrCIDR := range accessControl.Allow {
			allErrs = append(allErrs, validateIPorCIDR(ipOrCIDR, fieldPath.Child("allow").Index(i))...)
		}
		fieldCount++
	}

	if accessControl.Deny != nil {
		for i, ipOrCIDR := range accessControl.Deny {
			allErrs = append(allErrs, validateIPorCIDR(ipOrCIDR, fieldPath.Child("deny").Index(i))...)
		}
		fieldCount++
	}

	if fieldCount != 1 {
		allErrs = append(allErrs, field.Invalid(fieldPath, "", "must specify exactly one of: `allow` or `deny`"))
	}

	return allErrs
}

func validateIPorCIDR(ipOrCIDR string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	_, _, err := net.ParseCIDR(ipOrCIDR)
	if err == nil {
		// valid CIDR
		return allErrs
	}

	ip := net.ParseIP(ipOrCIDR)
	if ip != nil {
		// valid IP
		return allErrs
	}

	return append(allErrs, field.Invalid(fieldPath, ipOrCIDR, "must be a CIDR or IP"))
}

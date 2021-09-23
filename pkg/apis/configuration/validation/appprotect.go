package validation

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var appProtectPolicyRequiredFields = [][]string{
	{"spec", "policy"},
}

var appProtectLogConfRequiredFields = [][]string{
	{"spec", "content"},
	{"spec", "filter"},
}

var appProtectUserSigRequiredSlices = [][]string{
	{"spec", "signatures"},
}

// ValidateAppProtectPolicy validates Policy resource
func ValidateAppProtectPolicy(policy *unstructured.Unstructured) error {
	polName := policy.GetName()

	err := ValidateRequiredFields(policy, appProtectPolicyRequiredFields)
	if err != nil {
		return fmt.Errorf("Error validating App Protect Policy %v: %w", polName, err)
	}

	return nil
}

// ValidateAppProtectLogConf validates LogConfiguration resource
func ValidateAppProtectLogConf(logConf *unstructured.Unstructured) error {
	lcName := logConf.GetName()
	err := ValidateRequiredFields(logConf, appProtectLogConfRequiredFields)
	if err != nil {
		return fmt.Errorf("Error validating App Protect Log Configuration %v: %w", lcName, err)
	}

	return nil
}

func ValidateAppProtectUserSig(userSig *unstructured.Unstructured) error {
	sigName := userSig.GetName()
	err := ValidateRequiredSlices(userSig, appProtectUserSigRequiredSlices)
	if err != nil {
		return fmt.Errorf("Error validating App Protect User Signature %v: %w", sigName, err)
	}

	return nil
}

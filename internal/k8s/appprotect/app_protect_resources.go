package appprotect

import (
	"fmt"

	"github.com/nginxinc/kubernetes-ingress/internal/k8s/appprotect_common"

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

// validateAppProtectPolicy validates Policy resource
func validateAppProtectPolicy(policy *unstructured.Unstructured) error {
	polName := policy.GetName()

	err := appprotect_common.ValidateRequiredFields(policy, appProtectPolicyRequiredFields)
	if err != nil {
		return fmt.Errorf("Error validating App Protect Policy %v: %w", polName, err)
	}

	return nil
}

// validateAppProtectLogConf validates LogConfiguration resource
func validateAppProtectLogConf(logConf *unstructured.Unstructured) error {
	lcName := logConf.GetName()
	err := appprotect_common.ValidateRequiredFields(logConf, appProtectLogConfRequiredFields)
	if err != nil {
		return fmt.Errorf("Error validating App Protect Log Configuration %v: %w", lcName, err)
	}

	return nil
}

func validateAppProtectUserSig(userSig *unstructured.Unstructured) error {
	sigName := userSig.GetName()
	err := appprotect_common.ValidateRequiredSlices(userSig, appProtectUserSigRequiredSlices)
	if err != nil {
		return fmt.Errorf("Error validating App Protect User Signature %v: %w", sigName, err)
	}

	return nil
}
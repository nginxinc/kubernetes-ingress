package validation

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestValidateAppProtectPolicy(t *testing.T) {
	tests := []struct {
		policy    *unstructured.Unstructured
		expectErr bool
		msg       string
	}{
		{
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"policy": map[string]interface{}{},
					},
				},
			},
			expectErr: false,
			msg:       "valid policy",
		},
		{
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"something": map[string]interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid policy with no policy field",
		},
		{
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"something": map[string]interface{}{
						"policy": map[string]interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid policy with no spec field",
		},
	}

	for _, test := range tests {
		err := ValidateAppProtectPolicy(test.policy)
		if test.expectErr && err == nil {
			t.Errorf("validateAppProtectPolicy() returned no error for the case of %s", test.msg)
		}
		if !test.expectErr && err != nil {
			t.Errorf("validateAppProtectPolicy() returned unexpected error %v for the case of %s", err, test.msg)
		}
	}
}

func TestValidateAppProtectLogConf(t *testing.T) {
	tests := []struct {
		logConf   *unstructured.Unstructured
		expectErr bool
		msg       string
	}{
		{
			logConf: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"content": map[string]interface{}{},
						"filter":  map[string]interface{}{},
					},
				},
			},
			expectErr: false,
			msg:       "valid log conf",
		},
		{
			logConf: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"filter": map[string]interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid log conf with no content field",
		},
		{
			logConf: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"content": map[string]interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid log conf with no filter field",
		},
		{
			logConf: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"something": map[string]interface{}{
						"content": map[string]interface{}{},
						"filter":  map[string]interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid log conf with no spec field",
		},
	}

	for _, test := range tests {
		err := ValidateAppProtectLogConf(test.logConf)
		if test.expectErr && err == nil {
			t.Errorf("validateAppProtectLogConf() returned no error for the case of %s", test.msg)
		}
		if !test.expectErr && err != nil {
			t.Errorf("validateAppProtectLogConf() returned unexpected error %v for the case of %s", err, test.msg)
		}
	}
}

func TestValidateAppProtectUserSig(t *testing.T) {
	tests := []struct {
		userSig   *unstructured.Unstructured
		expectErr bool
		msg       string
	}{
		{
			userSig: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"signatures": []interface{}{},
					},
				},
			},
			expectErr: false,
			msg:       "valid user sig",
		},
		{
			userSig: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"something": []interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid user sig with no signatures",
		},
		{
			userSig: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"something": map[string]interface{}{
						"signatures": []interface{}{},
					},
				},
			},
			expectErr: true,
			msg:       "invalid user sign with no spec field",
		},
	}

	for _, test := range tests {
		err := ValidateAppProtectUserSig(test.userSig)
		if test.expectErr && err == nil {
			t.Errorf("validateAppProtectUserSig() returned no error for the case of %s", test.msg)
		}
		if !test.expectErr && err != nil {
			t.Errorf("validateAppProtectUserSig() returned unexpected error %v for the case of %s", err, test.msg)
		}
	}
}

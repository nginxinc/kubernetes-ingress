package appprotectdos

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestCreateAppProtectDosPolicyEx(t *testing.T) {
	tests := []struct {
		policy           *unstructured.Unstructured
		expectedPolicyEx *DosPolicyEx
		wantErr          bool
		msg              string
	}{
		{
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
				},
			},
			expectedPolicyEx: &DosPolicyEx{
				IsValid:  false,
				ErrorMsg: "Validation Failed",
			},
			wantErr: true,
			msg:     "dos policy no spec",
		},
		{
			policy: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
					},
				},
			},
			expectedPolicyEx: &DosPolicyEx{
				IsValid:  true,
				ErrorMsg: "",
			},
			wantErr: false,
			msg:     "dos policy is valid",
		},
	}

	for _, test := range tests {
		test.expectedPolicyEx.Obj = test.policy

		policyEx, err := createAppProtectDosPolicyEx(test.policy)
		if (err != nil) != test.wantErr {
			t.Errorf("createAppProtectDosPolicyEx() returned %v, for the case of %s", err, test.msg)
		}
		if diff := cmp.Diff(test.expectedPolicyEx, policyEx); diff != "" {
			t.Errorf("createAppProtectDosPolicyEx() %q returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestCreateAppProtectDosLogConfEx(t *testing.T) {
	tests := []struct {
		logConf           *unstructured.Unstructured
		expectedLogConfEx *DosLogConfEx
		wantErr           bool
		msg               string
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
			expectedLogConfEx: &DosLogConfEx{
				IsValid:  true,
				ErrorMsg: "",
			},
			wantErr: false,
			msg:     "Valid DosLogConf",
		},
		{
			logConf: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						"content": map[string]interface{}{},
					},
				},
			},
			expectedLogConfEx: &DosLogConfEx{
				IsValid:  false,
				ErrorMsg: failedValidationErrorMsg,
			},
			wantErr: true,
			msg:     "Invalid DosLogConf",
		},
	}

	for _, test := range tests {
		test.expectedLogConfEx.Obj = test.logConf

		policyEx, err := createAppProtectDosLogConfEx(test.logConf)
		if (err != nil) != test.wantErr {
			t.Errorf("createAppProtectDosLogConfEx() returned %v, for the case of %s", err, test.msg)
		}
		if diff := cmp.Diff(test.expectedLogConfEx, policyEx); diff != "" {
			t.Errorf("createAppProtectDosLogConfEx() %q returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddOrUpdateDosPolicy(t *testing.T) {
	basicTestPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "testing",
			},
			"spec": map[string]interface{}{
				"mitigation_mode":            "standard",
				"automation_tools_detection": "on",
				"tls_fingerprint":            "on",
				"signatures":                 "on",
				"bad_actors":                 "on",
			},
		},
	}
	invalidTestPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "testing",
			},
		},
	}
	apc := newConfigurationImpl()
	tests := []struct {
		policy           *unstructured.Unstructured
		expectedChanges  []Change
		expectedProblems []Problem
		msg              string
	}{
		{
			policy: basicTestPolicy,
			expectedChanges: []Change{
				{
					Resource: &DosPolicyEx{
						Obj:     basicTestPolicy,
						IsValid: true,
					},
					Op: AddOrUpdate,
				},
			},
			expectedProblems: nil,
			msg:              "Basic Case",
		},
		{
			policy: invalidTestPolicy,
			expectedChanges: []Change{
				{
					Resource: &DosPolicyEx{
						Obj:      invalidTestPolicy,
						IsValid:  false,
						ErrorMsg: "Validation Failed",
					},
					Op: Delete,
				},
			},
			expectedProblems: []Problem{
				{
					Object:  invalidTestPolicy,
					Reason:  "Rejected",
					Message: "Error validating dos policy : Error validating App Protect Dos Policy : Required field map[] not found",
				},
			},
			msg: "validation failed",
		},
	}
	for _, test := range tests {
		aPChans, aPProbs := apc.AddOrUpdatePolicy(test.policy)
		if diff := cmp.Diff(test.expectedChanges, aPChans); diff != "" {
			t.Errorf("test.expectedChanges: %q", test.expectedChanges)
			t.Errorf("aPChans              : %q", aPChans)
			t.Errorf("AddOrUpdatePolicy() %q changes returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
		if diff := cmp.Diff(test.expectedProblems, aPProbs); diff != "" {
			t.Errorf("test.expectedProblems: %v", test.expectedProblems)
			t.Errorf("aPProbs              : %v", aPProbs)
			t.Errorf("AddOrUpdatePolicy() %q problems returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddOrUpdateDosLogConf(t *testing.T) {
	validLogConf := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "testing",
				"name":      "testlogconf",
			},
			"spec": map[string]interface{}{
				"content": map[string]interface{}{},
				"filter":  map[string]interface{}{},
			},
		},
	}
	invalidLogConf := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "testing",
				"name":      "testlogconf",
			},
			"spec": map[string]interface{}{
				"content": map[string]interface{}{},
			},
		},
	}
	apc := NewConfiguration()
	tests := []struct {
		logconf          *unstructured.Unstructured
		expectedChanges  []Change
		expectedProblems []Problem
		msg              string
	}{
		{
			logconf: validLogConf,
			expectedChanges: []Change{
				{
					Resource: &DosLogConfEx{
						Obj:     validLogConf,
						IsValid: true,
					},
					Op: AddOrUpdate,
				},
			},
			expectedProblems: nil,
			msg:              "Basic Case",
		},
		{
			logconf: invalidLogConf,
			expectedChanges: []Change{
				{
					Resource: &DosLogConfEx{
						Obj:      invalidLogConf,
						IsValid:  false,
						ErrorMsg: "Validation Failed",
					},
					Op: Delete,
				},
			},
			expectedProblems: []Problem{
				{
					Object:  invalidLogConf,
					Reason:  "Rejected",
					Message: "Error validating App Protect Dos Log Configuration testlogconf: Required field map[] not found",
				},
			},
			msg: "validation failed",
		},
	}
	for _, test := range tests {
		aPChans, aPProbs := apc.AddOrUpdateLogConf(test.logconf)
		if diff := cmp.Diff(test.expectedChanges, aPChans); diff != "" {
			t.Errorf("AddOrUpdateLogConf() %q changes returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
		if diff := cmp.Diff(test.expectedProblems, aPProbs); diff != "" {
			t.Errorf("AddOrUpdateLogConf() %q problems returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestDeletePolicy(t *testing.T) {
	appProtectConfiguration := newConfigurationImpl()
	appProtectConfiguration.DosPolicies["testing/test"] = &DosPolicyEx{}
	tests := []struct {
		key              string
		expectedChanges  []Change
		expectedProblems []Problem
		msg              string
	}{
		{
			key: "testing/test",
			expectedChanges: []Change{
				{
					Op:       Delete,
					Resource: &DosPolicyEx{},
				},
			},
			expectedProblems: nil,
			msg:              "Positive",
		},
		{
			key:              "testing/notpresent",
			expectedChanges:  nil,
			expectedProblems: nil,
			msg:              "Negative",
		},
	}
	for _, test := range tests {
		apChan, apProbs := appProtectConfiguration.DeletePolicy(test.key)
		if diff := cmp.Diff(test.expectedChanges, apChan); diff != "" {
			t.Errorf("DeletePolicy() %q changes returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
		if diff := cmp.Diff(test.expectedProblems, apProbs); diff != "" {
			t.Errorf("DeletePolicy() %q problems returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestDeleteDosLogConf(t *testing.T) {
	appProtectConfiguration := newConfigurationImpl()
	appProtectConfiguration.DosLogConfs["testing/test"] = &DosLogConfEx{}
	tests := []struct {
		key              string
		expectedChanges  []Change
		expectedProblems []Problem
		msg              string
	}{
		{
			key: "testing/test",
			expectedChanges: []Change{
				{
					Op:       Delete,
					Resource: &DosLogConfEx{},
				},
			},
			expectedProblems: nil,
			msg:              "Positive",
		},
		{
			key:              "testing/notpresent",
			expectedChanges:  nil,
			expectedProblems: nil,
			msg:              "Negative",
		},
	}
	for _, test := range tests {
		apChan, apProbs := appProtectConfiguration.DeleteLogConf(test.key)
		if diff := cmp.Diff(test.expectedChanges, apChan); diff != "" {
			t.Errorf("DeleteLogConf() %q changes returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
		if diff := cmp.Diff(test.expectedProblems, apProbs); diff != "" {
			t.Errorf("DeleteLogConf() %q problems returned unexpected result (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestGetAppProtectDosResource(t *testing.T) {
	tests := []struct {
		kind    string
		key     string
		wantErr bool
		errMsg  string
		msg     string
	}{
		{
			kind:    "APDosPolicy",
			key:     "testing/test1",
			wantErr: false,
			msg:     "Policy, positive",
		},
		{
			kind:    "APDosPolicy",
			key:     "testing/test2",
			wantErr: true,
			errMsg:  "Validation Failed",
			msg:     "Policy, Negative, invalid object",
		},
		{
			kind:    "APDosPolicy",
			key:     "testing/test3",
			wantErr: true,
			errMsg:  "App Protect Dos Policy testing/test3 not found",
			msg:     "Policy, Negative, Object Does not exist",
		},
		{
			kind:    "APDosLogConf",
			key:     "testing/test1",
			wantErr: false,
			msg:     "LogConf, positive",
		},
		{
			kind:    "APDosLogConf",
			key:     "testing/test2",
			wantErr: true,
			errMsg:  "Validation Failed",
			msg:     "LogConf, Negative, invalid object",
		},
		{
			kind:    "APDosLogConf",
			key:     "testing/test3",
			wantErr: true,
			errMsg:  "App Protect DosLogConf testing/test3 not found",
			msg:     "LogConf, Negative, Object Does not exist",
		},
		{
			kind:    "Notreal",
			key:     "testing/test3",
			wantErr: true,
			errMsg:  "Unknown App Protect Dos resource kind Notreal",
			msg:     "Ivalid kind, Negative",
		},
	}
	appProtectConfiguration := newConfigurationImpl()
	appProtectConfiguration.DosPolicies["testing/test1"] = &DosPolicyEx{IsValid: true, Obj: &unstructured.Unstructured{}}
	appProtectConfiguration.DosPolicies["testing/test2"] = &DosPolicyEx{IsValid: false, Obj: &unstructured.Unstructured{}, ErrorMsg: "Validation Failed"}
	appProtectConfiguration.DosLogConfs["testing/test1"] = &DosLogConfEx{IsValid: true, Obj: &unstructured.Unstructured{}}
	appProtectConfiguration.DosLogConfs["testing/test2"] = &DosLogConfEx{IsValid: false, Obj: &unstructured.Unstructured{}, ErrorMsg: "Validation Failed"}

	for _, test := range tests {
		_, err := appProtectConfiguration.GetAppResource(test.kind, test.key)
		if (err != nil) != test.wantErr {
			t.Errorf("GetAppResource() returned %v on case %s", err, test.msg)
		}
		if test.wantErr || err != nil {
			if test.errMsg != err.Error() {
				t.Errorf("GetAppResource() returned error message %s on case %s (expected %s)", err.Error(), test.msg, test.errMsg)
			}
		}
	}
}

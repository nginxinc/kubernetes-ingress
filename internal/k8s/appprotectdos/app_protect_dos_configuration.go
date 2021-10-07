package appprotectdos

import (
	"fmt"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/validation"

	"github.com/nginxinc/kubernetes-ingress/internal/k8s/appprotect_common"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// reasons for invalidity
const (
	failedValidationErrorMsg = "Validation Failed"
)

var (
	// DosPolicyGVR is the group version resource of the appprotectdos policy
	DosPolicyGVR = schema.GroupVersionResource{
		Group:    "appprotectdos.f5.com",
		Version:  "v1beta1",
		Resource: "apdospolicies",
	}

	// DosPolicyGVK is the group version kind of the appprotectdos policy
	DosPolicyGVK = schema.GroupVersionKind{
		Group:   "appprotectdos.f5.com",
		Version: "v1beta1",
		Kind:    "APDosPolicy",
	}

	// DosLogConfGVR is the group version resource of the appprotectdos policy
	DosLogConfGVR = schema.GroupVersionResource{
		Group:    "appprotectdos.f5.com",
		Version:  "v1beta1",
		Resource: "apdoslogconfs",
	}
	// DosLogConfGVK is the group version kind of the appprotectdos policy
	DosLogConfGVK = schema.GroupVersionKind{
		Group:   "appprotectdos.f5.com",
		Version: "v1beta1",
		Kind:    "APDosLogConf",
	}
)

// Operation defines an operation to perform for an App Protect Dos resource.
type Operation int

const (
	// Delete the config of the resource
	Delete Operation = iota
	// AddOrUpdate the config of the resource
	AddOrUpdate
)

// Change represents a change in an App Protect Dos resource
type Change struct {
	// Op is an operation that needs be performed on the resource.
	Op Operation
	// Resource is the target resource.
	Resource interface{}
}

// Problem represents a problem with an App Protect Dos resource
type Problem struct {
	// Object is a configuration object.
	Object *unstructured.Unstructured
	// Reason tells the reason. It matches the reason in the events of our configuration objects.
	Reason string
	// Message gives the details about the problem. It matches the message in the events of our configuration objects.
	Message string
}

// Configuration configures App Protect Dos resources that the Ingress Controller uses.
type Configuration interface {
	AddOrUpdatePolicy(policyObj *unstructured.Unstructured) (changes []Change, problems []Problem)
	AddOrUpdateLogConf(logConfObj *unstructured.Unstructured) (changes []Change, problems []Problem)
	GetAppResource(kind, key string) (*unstructured.Unstructured, error)
	DeletePolicy(key string) (changes []Change, problems []Problem)
	DeleteLogConf(key string) (changes []Change, problems []Problem)
}

// ConfigurationImpl holds representations of App Protect Dos cluster resources
type ConfigurationImpl struct {
	DosPolicies map[string]*DosPolicyEx
	DosLogConfs map[string]*DosLogConfEx
}

// NewConfiguration creates a new App Protect Dos Configuration
func NewConfiguration() Configuration {
	return newConfigurationImpl()
}

// newConfigurationImpl creates a new App Protect Dos ConfigurationImpl
func newConfigurationImpl() *ConfigurationImpl {
	return &ConfigurationImpl{
		DosPolicies: make(map[string]*DosPolicyEx),
		DosLogConfs: make(map[string]*DosLogConfEx),
	}
}

// DosPolicyEx represents an App Protect Dos policy cluster resource
type DosPolicyEx struct {
	Obj      *unstructured.Unstructured
	IsValid  bool
	ErrorMsg string
}

func createAppProtectDosPolicyEx(policyObj *unstructured.Unstructured) (*DosPolicyEx, error) {
	err := validation.ValidateAppProtectDosPolicy(policyObj)
	if err != nil {
		errMsg := fmt.Sprintf("Error validating dos policy %s: %v", policyObj.GetName(), err)
		return &DosPolicyEx{Obj: policyObj, IsValid: false, ErrorMsg: failedValidationErrorMsg}, fmt.Errorf(errMsg)
	}

	return &DosPolicyEx{
		Obj:     policyObj,
		IsValid: true,
	}, nil
}

// DosLogConfEx represents an App Protect Dos Log Configuration cluster resource
type DosLogConfEx struct {
	Obj      *unstructured.Unstructured
	IsValid  bool
	ErrorMsg string
}

func createAppProtectDosLogConfEx(dosLogConfObj *unstructured.Unstructured) (*DosLogConfEx, error) {
	err := validation.ValidateAppProtectDosLogConf(dosLogConfObj)
	if err != nil {
		return &DosLogConfEx{
			Obj:      dosLogConfObj,
			IsValid:  false,
			ErrorMsg: failedValidationErrorMsg,
		}, err
	}
	return &DosLogConfEx{
		Obj:     dosLogConfObj,
		IsValid: true,
	}, nil
}

// AddOrUpdatePolicy adds or updates an App Protect Dos Policy to App Protect Dos Configuration
func (ci *ConfigurationImpl) AddOrUpdatePolicy(policyObj *unstructured.Unstructured) (changes []Change, problems []Problem) {
	resNsName := appprotect_common.GetNsName(policyObj)
	policy, err := createAppProtectDosPolicyEx(policyObj)
	if err != nil {
		ci.DosPolicies[resNsName] = policy
		return append(changes, Change{Op: Delete, Resource: policy}),
			append(problems, Problem{Object: policyObj, Reason: "Rejected", Message: err.Error()})
	}
	ci.DosPolicies[resNsName] = policy
	return append(changes, Change{Op: AddOrUpdate, Resource: policy}), problems
}

// AddOrUpdateLogConf adds or updates App Protect Dos Log Configuration to App Protect Dos Configuration
func (ci *ConfigurationImpl) AddOrUpdateLogConf(logconfObj *unstructured.Unstructured) (changes []Change, problems []Problem) {
	resNsName := appprotect_common.GetNsName(logconfObj)
	logConf, err := createAppProtectDosLogConfEx(logconfObj)
	ci.DosLogConfs[resNsName] = logConf
	if err != nil {
		return append(changes, Change{Op: Delete, Resource: logConf}),
			append(problems, Problem{Object: logconfObj, Reason: "Rejected", Message: err.Error()})
	}
	return append(changes, Change{Op: AddOrUpdate, Resource: logConf}), problems
}

// GetAppResource returns a pointer to an App Protect Dos resource
func (ci *ConfigurationImpl) GetAppResource(kind, key string) (*unstructured.Unstructured, error) {
	switch kind {
	case DosPolicyGVK.Kind:
		if obj, ok := ci.DosPolicies[key]; ok {
			if obj.IsValid {
				return obj.Obj, nil
			}
			return nil, fmt.Errorf(obj.ErrorMsg)
		}
		return nil, fmt.Errorf("App Protect Dos Policy %s not found", key)
	case DosLogConfGVK.Kind:
		if obj, ok := ci.DosLogConfs[key]; ok {
			if obj.IsValid {
				return obj.Obj, nil
			}
			return nil, fmt.Errorf(obj.ErrorMsg)
		}
		return nil, fmt.Errorf("App Protect DosLogConf %s not found", key)
	}
	return nil, fmt.Errorf("Unknown App Protect Dos resource kind %s", kind)
}

// DeletePolicy deletes an App Protect Policy from App Protect Dos Configuration
func (ci *ConfigurationImpl) DeletePolicy(key string) (changes []Change, problems []Problem) {
	if _, has := ci.DosPolicies[key]; has {
		change := Change{Op: Delete, Resource: ci.DosPolicies[key]}
		delete(ci.DosPolicies, key)
		return append(changes, change), problems
	}
	return changes, problems
}

// DeleteLogConf deletes an App Protect Dos Log Configuration from App Protect Dos Configuration
func (ci *ConfigurationImpl) DeleteLogConf(key string) (changes []Change, problems []Problem) {
	if _, has := ci.DosLogConfs[key]; has {
		change := Change{Op: Delete, Resource: ci.DosLogConfs[key]}
		delete(ci.DosLogConfs, key)
		return append(changes, change), problems
	}
	return changes, problems
}

// FakeConfiguration holds representations of fake App Protect Dos cluster resources
type FakeConfiguration struct {
	DosPolicies map[string]*DosPolicyEx
	DosLogConfs map[string]*DosLogConfEx
}

// NewFakeConfiguration creates a new App Protect Dos Configuration
func NewFakeConfiguration() Configuration {
	return &FakeConfiguration{
		DosPolicies: make(map[string]*DosPolicyEx),
		DosLogConfs: make(map[string]*DosLogConfEx),
	}
}

// AddOrUpdatePolicy adds or updates an App Protect Policy to App Protect Dos Configuration
func (fc *FakeConfiguration) AddOrUpdatePolicy(policyObj *unstructured.Unstructured) (changes []Change, problems []Problem) {
	resNsName := appprotect_common.GetNsName(policyObj)
	policy := &DosPolicyEx{
		Obj:     policyObj,
		IsValid: true,
	}
	fc.DosPolicies[resNsName] = policy
	return changes, problems
}

// AddOrUpdateLogConf adds or updates App Protect Dos Log Configuration to App Protect Dos Configuration
func (fc *FakeConfiguration) AddOrUpdateLogConf(logConfObj *unstructured.Unstructured) (changes []Change, problems []Problem) {
	resNsName := appprotect_common.GetNsName(logConfObj)
	logConf := &DosLogConfEx{
		Obj:     logConfObj,
		IsValid: true,
	}
	fc.DosLogConfs[resNsName] = logConf
	return changes, problems
}

// GetAppResource returns a pointer to an App Protect Dos resource
func (fc *FakeConfiguration) GetAppResource(kind, key string) (*unstructured.Unstructured, error) {
	switch kind {
	case DosPolicyGVK.Kind:
		if obj, ok := fc.DosPolicies[key]; ok {
			return obj.Obj, nil
		}
		return nil, fmt.Errorf("App Protect Dos Policy %s not found", key)
	case DosLogConfGVK.Kind:
		if obj, ok := fc.DosLogConfs[key]; ok {
			return obj.Obj, nil
		}
		return nil, fmt.Errorf("App Protect Dos LogConf %s not found", key)
	}
	return nil, fmt.Errorf("Unknown App Protect Dos resource kind %s", kind)
}

// DeletePolicy deletes an App Protect Dos Policy from App Protect Dos Configuration
func (fc *FakeConfiguration) DeletePolicy(key string) (changes []Change, problems []Problem) {
	return changes, problems
}

// DeleteLogConf deletes an App Protect Dos Log Configuration from App Protect Dos Configuration
func (fc *FakeConfiguration) DeleteLogConf(key string) (changes []Change, problems []Problem) {
	return changes, problems
}

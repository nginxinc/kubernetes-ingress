package validation

import (
	"fmt"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// GlobalConfigurationValidator validates a GlobalConfiguration resource.
type GlobalConfigurationValidator struct {
	forbiddenListenerPorts map[int]bool
}

// NewGlobalConfigurationValidator creates a new GlobalConfigurationValidator.
func NewGlobalConfigurationValidator(forbiddenListenerPorts map[int]bool) *GlobalConfigurationValidator {
	return &GlobalConfigurationValidator{
		forbiddenListenerPorts: forbiddenListenerPorts,
	}
}

// ValidateGlobalConfiguration validates a GlobalConfiguration.
func (gcv *GlobalConfigurationValidator) ValidateGlobalConfiguration(globalConfiguration *v1alpha1.GlobalConfiguration) error {
	allErrs := gcv.validateGlobalConfigurationSpec(&globalConfiguration.Spec, field.NewPath("spec"))
	return allErrs.ToAggregate()
}

func (gcv *GlobalConfigurationValidator) validateGlobalConfigurationSpec(spec *v1alpha1.GlobalConfigurationSpec, fieldPath *field.Path) field.ErrorList {
	return gcv.validateListeners(spec.Listeners, fieldPath.Child("listeners"))
}

func (gcv *GlobalConfigurationValidator) validateListeners(listeners []v1alpha1.Listener, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	listenerNames := sets.String{}
	portProtocolCombinations := sets.String{}

	for i, l := range listeners {
		idxPath := fieldPath.Index(i)
		portProtocolKey := generatePortProtocolKey(l.Port, l.Protocol)

		listenerErrs := gcv.validateListener(l, idxPath)
		if len(listenerErrs) > 0 {
			allErrs = append(allErrs, listenerErrs...)
		} else if listenerNames.Has(l.Name) {
			allErrs = append(allErrs, field.Duplicate(idxPath.Child("name"), l.Name))
		} else if portProtocolCombinations.Has(portProtocolKey) {
			msg := fmt.Sprintf("Duplicated port/protocol combination %s", portProtocolKey)
			allErrs = append(allErrs, field.Duplicate(fieldPath, msg))
		} else {
			listenerNames.Insert(l.Name)
			portProtocolCombinations.Insert(portProtocolKey)
		}
	}

	return allErrs
}

func generatePortProtocolKey(port int, protocol string) string {
	return fmt.Sprintf("%d/%s", port, protocol)
}

func (gcv *GlobalConfigurationValidator) validateListener(listener v1alpha1.Listener, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateGlobalConfigurationListenerName(listener.Name, fieldPath.Child("name"))...)
	allErrs = append(allErrs, gcv.validateListenerPort(listener.Port, fieldPath.Child("port"))...)
	allErrs = append(allErrs, validateListenerProtocol(listener.Protocol, fieldPath.Child("protocol"))...)

	return allErrs
}

func validateGlobalConfigurationListenerName(name string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if name == v1alpha1.TLSPassthroughListenerName {
		return append(allErrs, field.Forbidden(fieldPath, "is the name of a built-in listener"))
	}

	return validateListenerName(name, fieldPath)
}

func (gcv *GlobalConfigurationValidator) validateListenerPort(port int, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if gcv.forbiddenListenerPorts[port] {
		msg := fmt.Sprintf("port %v is forbidden", port)
		return append(allErrs, field.Forbidden(fieldPath, msg))
	}

	for _, msg := range validation.IsValidPortNum(port) {
		allErrs = append(allErrs, field.Invalid(fieldPath, port, msg))
	}

	return allErrs
}

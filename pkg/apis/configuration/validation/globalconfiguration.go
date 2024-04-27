package validation

import (
	"fmt"
	"sort"
	"strings"

	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var allowedProtocols = map[string]bool{
	"TCP":  true,
	"UDP":  true,
	"HTTP": true,
}

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
func (gcv *GlobalConfigurationValidator) ValidateGlobalConfiguration(globalConfiguration *conf_v1.GlobalConfiguration) error {
	allErrs := gcv.validateGlobalConfigurationSpec(&globalConfiguration.Spec, field.NewPath("spec"))
	return allErrs.ToAggregate()
}

func (gcv *GlobalConfigurationValidator) validateGlobalConfigurationSpec(spec *conf_v1.GlobalConfigurationSpec, fieldPath *field.Path) field.ErrorList {
	validListeners, err := gcv.getValidListeners(spec.Listeners, fieldPath.Child("listeners"))
	spec.Listeners = validListeners
	return err
}

func (gcv *GlobalConfigurationValidator) getValidListeners(listeners []conf_v1.Listener, fieldPath *field.Path) ([]conf_v1.Listener, field.ErrorList) {
	allErrs := field.ErrorList{}

	listenerNames := sets.Set[string]{}
	portProtocolCombinations := sets.Set[string]{}

	portProtocolMap := make(map[int]string)
	var validListeners []conf_v1.Listener

	for i, l := range listeners {
		idxPath := fieldPath.Index(i)
		portProtocolKey := generatePortProtocolKey(l.Port, l.Protocol)
		listenerErrs := gcv.validateListener(l, idxPath)
		if len(listenerErrs) > 0 {
			allErrs = append(allErrs, listenerErrs...)
		} else if listenerNames.Has(l.Name) {
			allErrs = append(allErrs, field.Duplicate(idxPath.Child("name"), l.Name))
		} else if portProtocolCombinations.Has(portProtocolKey) {
			msg := fmt.Sprintf("Listener %s: Duplicated port/protocol combination %s", l.Name, portProtocolKey)
			allErrs = append(allErrs, field.Duplicate(fieldPath, msg))
		} else if protocol, ok := portProtocolMap[l.Port]; ok {
			var msg string
			switch protocol {
			case "HTTP":
				if l.Protocol == "TCP" || l.Protocol == "UDP" {
					msg = fmt.Sprintf(
						"Listener %s with protocol %s can't use port %d. Port is taken by an HTTP listener",
						l.Name, l.Protocol, l.Port)
					allErrs = append(allErrs, field.Forbidden(fieldPath, msg))
				} else {
					validListeners = append(validListeners, l)
				}
			case "TCP", "UDP":
				if l.Protocol == "HTTP" {
					msg = fmt.Sprintf(
						"Listener %s with protocol %s can't use port %d. Port is taken by TCP or UDP listener",
						l.Name, l.Protocol, l.Port)
					allErrs = append(allErrs, field.Forbidden(fieldPath, msg))
				} else {
					validListeners = append(validListeners, l)
				}
			}
		} else {
			listenerNames.Insert(l.Name)
			portProtocolCombinations.Insert(portProtocolKey)
			portProtocolMap[l.Port] = l.Protocol
			validListeners = append(validListeners, l)
		}
	}
	return validListeners, allErrs
}

func generatePortProtocolKey(port int, protocol string) string {
	return fmt.Sprintf("%d/%s", port, protocol)
}

func (gcv *GlobalConfigurationValidator) validateListener(listener conf_v1.Listener, fieldPath *field.Path) field.ErrorList {
	allErrs := validateGlobalConfigurationListenerName(listener.Name, fieldPath.Child("name"))
	allErrs = append(allErrs, gcv.validateListenerPort(listener.Name, listener.Port, fieldPath.Child("port"))...)
	allErrs = append(allErrs, validateListenerProtocol(listener.Protocol, fieldPath.Child("protocol"))...)

	return allErrs
}

func validateGlobalConfigurationListenerName(name string, fieldPath *field.Path) field.ErrorList {
	if name == conf_v1.TLSPassthroughListenerName {
		return field.ErrorList{field.Forbidden(fieldPath, "is the name of a built-in listener")}
	}
	return validateListenerName(name, fieldPath)
}

func (gcv *GlobalConfigurationValidator) validateListenerPort(name string, port int, fieldPath *field.Path) field.ErrorList {
	if gcv.forbiddenListenerPorts[port] {
		msg := fmt.Sprintf("Listener %v: port %v is forbidden", name, port)
		return field.ErrorList{field.Forbidden(fieldPath, msg)}
	}

	allErrs := field.ErrorList{}
	for _, msg := range validation.IsValidPortNum(port) {
		allErrs = append(allErrs, field.Invalid(fieldPath, port, msg))
	}
	return allErrs
}

func validateListenerProtocol(protocol string, fieldPath *field.Path) field.ErrorList {
	switch {
	case allowedProtocols[protocol]:
		return nil
	default:
		msg := fmt.Sprintf("must specify a valid protocol. Accepted values: %s",
			strings.Join(getProtocolsFromMap(allowedProtocols), ","))
		return field.ErrorList{field.Invalid(fieldPath, protocol, msg)}
	}
}

func getProtocolsFromMap(p map[string]bool) []string {
	var keys []string

	for k := range p {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

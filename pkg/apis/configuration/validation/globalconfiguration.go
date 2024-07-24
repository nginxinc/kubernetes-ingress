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
	ipPortProtocolCombinations := make(map[string]map[int]string) // map[IP]map[Port]Protocol
	var validListeners []conf_v1.Listener

	for i, l := range listeners {
		idxPath := fieldPath.Index(i)
		listenerErrs := gcv.validateListener(l, idxPath)
		if len(listenerErrs) > 0 {
			allErrs = append(allErrs, listenerErrs...)
			continue
		}

		if err := gcv.validateIP(l, idxPath); err != nil {
			allErrs = append(allErrs, err)
			continue
		}

		if err := gcv.checkForDuplicateName(listenerNames, l, idxPath); err != nil {
			allErrs = append(allErrs, err)
			continue
		}

		if err := gcv.checkPortProtocolConflicts(ipPortProtocolCombinations, l, fieldPath); err != nil {
			allErrs = append(allErrs, err)
			continue
		}

		gcv.updatePortProtocolCombinations(ipPortProtocolCombinations, l)
		validListeners = append(validListeners, l)
	}
	return validListeners, allErrs
}

// validateIP checks if the provided IP address of the listener is valid.
func (gcv *GlobalConfigurationValidator) validateIP(listener conf_v1.Listener, idxPath *field.Path) *field.Error {
	if listener.IP != "" {
		ipPath := idxPath.Child("IP")
		if len(validation.IsValidIP(ipPath, listener.IP)) > 0 {
			return field.Invalid(ipPath, listener.IP, "invalid IP address")
		}
	}
	return nil
}

// checkForDuplicateName checks if the listener name is unique.
func (gcv *GlobalConfigurationValidator) checkForDuplicateName(listenerNames sets.Set[string], listener conf_v1.Listener, idxPath *field.Path) *field.Error {
	if listenerNames.Has(listener.Name) {
		return field.Duplicate(idxPath.Child("name"), listener.Name)
	}
	listenerNames.Insert(listener.Name)
	return nil
}

// checkPortProtocolConflicts ensures no duplicate or conflicting port/protocol combinations exist.
func (gcv *GlobalConfigurationValidator) checkPortProtocolConflicts(combinations map[string]map[int]string, listener conf_v1.Listener, fieldPath *field.Path) *field.Error {
	ip := listener.IP

	if combinations[ip] == nil {
		combinations[ip] = make(map[int]string)
	}

	existingProtocol, exists := combinations[ip][listener.Port]
	if exists {
		if existingProtocol == listener.Protocol {
			return field.Duplicate(fieldPath, fmt.Sprintf("Listener %s: Duplicated port/protocol combination %d/%s", listener.Name, listener.Port, listener.Protocol))
		} else if listener.Protocol == "HTTP" || existingProtocol == "HTTP" {
			return field.Invalid(fieldPath.Child("port"), listener.Port, fmt.Sprintf("Listener %s: Port %d is used with a different protocol (current: %s, new: %s)", listener.Name, listener.Port, existingProtocol, listener.Protocol))
		}
	}

	return nil
}

// updatePortProtocolCombinations updates the port/protocol combinations map with the given listener's details.
func (gcv *GlobalConfigurationValidator) updatePortProtocolCombinations(combinations map[string]map[int]string, listener conf_v1.Listener) {
	ip := listener.IP

	if combinations[ip] == nil {
		combinations[ip] = make(map[int]string)
	}

	combinations[ip][listener.Port] = listener.Protocol
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

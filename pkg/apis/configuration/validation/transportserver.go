package validation

import (
	"fmt"

	v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// TransportServerValidator validates a TransportServer resource.
type TransportServerValidator struct {
	tlsPassthrough bool
}

// NewTransportServerValidator creates a new TransportServerValidator.
func NewTransportServerValidator(tlsPassthrough bool) *TransportServerValidator {
	return &TransportServerValidator{
		tlsPassthrough: tlsPassthrough,
	}
}

// ValidateTransportServer validates a TransportServer.
func (tsv *TransportServerValidator) ValidateTransportServer(transportServer *v1alpha1.TransportServer) error {
	allErrs := tsv.validateTransportServerSpec(&transportServer.Spec, field.NewPath("spec"))
	return allErrs.ToAggregate()
}

func (tsv *TransportServerValidator) validateTransportServerSpec(spec *v1alpha1.TransportServerSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, tsv.validateTransportListener(&spec.Listener, fieldPath.Child("listener"))...)

	isTLSPassthroughListener := isPotentialTLSPassthroughListener(&spec.Listener)
	allErrs = append(allErrs, validateTransportServerHost(spec.Host, fieldPath.Child("host"), isTLSPassthroughListener)...)

	upstreamErrs, upstreamNames := validateTransportServerUpstreams(spec.Upstreams, fieldPath.Child("upstreams"))
	allErrs = append(allErrs, upstreamErrs...)

	allErrs = append(allErrs, validateTransportServerUpstreamParameters(spec.UpstreamParameters, fieldPath.Child("upstreamParameters"), spec.Listener.Protocol)...)

	if spec.Action == nil {
		allErrs = append(allErrs, field.Required(fieldPath.Child("action"), "must specify action"))
	} else {
		allErrs = append(allErrs, validateTransportServerAction(spec.Action, fieldPath.Child("action"), upstreamNames)...)
	}

	return allErrs
}

func validateTransportServerHost(host string, fieldPath *field.Path, isTLSPassthroughListener bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if !isTLSPassthroughListener {
		if host != "" {
			return append(allErrs, field.Forbidden(fieldPath, "host field is allowed only for TLS Passthrough TransportServers"))
		}
		return allErrs
	}

	return validateHost(host, fieldPath)
}

func (tsv *TransportServerValidator) validateTransportListener(listener *v1alpha1.TransportServerListener, fieldPath *field.Path) field.ErrorList {
	if isPotentialTLSPassthroughListener(listener) {
		return tsv.validateTLSPassthroughListener(listener, fieldPath)
	}

	return validateRegularListener(listener, fieldPath)
}

func validateRegularListener(listener *v1alpha1.TransportServerListener, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateListenerName(listener.Name, fieldPath.Child("name"))...)
	allErrs = append(allErrs, validateListenerProtocol(listener.Protocol, fieldPath.Child("protocol"))...)

	return allErrs
}

func isPotentialTLSPassthroughListener(listener *v1alpha1.TransportServerListener) bool {
	return listener.Name == v1alpha1.TLSPassthroughListenerName || listener.Protocol == v1alpha1.TLSPassthroughListenerProtocol
}

func (tsv *TransportServerValidator) validateTLSPassthroughListener(listener *v1alpha1.TransportServerListener, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !tsv.tlsPassthrough {
		return append(allErrs, field.Forbidden(fieldPath, "TLS Passthrough is not enabled"))
	}

	if listener.Name == v1alpha1.TLSPassthroughListenerName && listener.Protocol != v1alpha1.TLSPassthroughListenerProtocol {
		msg := fmt.Sprintf("must be '%s' for the built-in %s listener", v1alpha1.TLSPassthroughListenerProtocol, v1alpha1.TLSPassthroughListenerName)
		return append(allErrs, field.Invalid(fieldPath.Child("protocol"), listener.Protocol, msg))
	}

	if listener.Protocol == v1alpha1.TLSPassthroughListenerProtocol && listener.Name != v1alpha1.TLSPassthroughListenerName {
		msg := fmt.Sprintf("must be '%s' for a listener with the protocol %s", v1alpha1.TLSPassthroughListenerName, v1alpha1.TLSPassthroughListenerProtocol)
		return append(allErrs, field.Invalid(fieldPath.Child("name"), listener.Name, msg))
	}

	return allErrs
}

func validateListenerName(name string, fieldPath *field.Path) field.ErrorList {
	return validateDNS1035Label(name, fieldPath)
}

// listenerProtocols defines the protocols supported by a listener.
var listenerProtocols = map[string]bool{
	"TCP": true,
	"UDP": true,
}

func validateListenerProtocol(protocol string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if protocol == "" {
		msg := fmt.Sprintf("must specify protocol. Accepted values: %s", mapToPrettyString(listenerProtocols))
		return append(allErrs, field.Required(fieldPath, msg))
	}

	if !listenerProtocols[protocol] {
		msg := fmt.Sprintf("invalid protocol. Accepted values: %s", mapToPrettyString(listenerProtocols))
		allErrs = append(allErrs, field.Invalid(fieldPath, protocol, msg))
	}

	return allErrs
}

func validateTransportServerUpstreams(upstreams []v1alpha1.Upstream, fieldPath *field.Path) (allErrs field.ErrorList, upstreamNames sets.String) {
	allErrs = field.ErrorList{}
	upstreamNames = sets.String{}

	for i, u := range upstreams {
		idxPath := fieldPath.Index(i)

		upstreamErrors := validateUpstreamName(u.Name, idxPath.Child("name"))
		if len(upstreamErrors) > 0 {
			allErrs = append(allErrs, upstreamErrors...)
		} else if upstreamNames.Has(u.Name) {
			allErrs = append(allErrs, field.Duplicate(idxPath.Child("name"), u.Name))
		} else {
			upstreamNames.Insert(u.Name)
		}

		allErrs = append(allErrs, validateServiceName(u.Service, idxPath.Child("service"))...)

		for _, msg := range validation.IsValidPortNum(u.Port) {
			allErrs = append(allErrs, field.Invalid(idxPath.Child("port"), u.Port, msg))
		}
	}

	return allErrs, upstreamNames
}

func validateTransportServerUpstreamParameters(upstreamParameters *v1alpha1.UpstreamParameters, fieldPath *field.Path, protocol string) field.ErrorList {
	allErrs := field.ErrorList{}

	if upstreamParameters == nil {
		return allErrs
	}

	allErrs = append(allErrs, validateUDPUpstreamParameter(upstreamParameters.UDPRequests, fieldPath.Child("udpRequests"), protocol)...)
	allErrs = append(allErrs, validateUDPUpstreamParameter(upstreamParameters.UDPResponses, fieldPath.Child("udpResponses"), protocol)...)

	return allErrs
}

func validateUDPUpstreamParameter(parameter *int, fieldPath *field.Path, protocol string) field.ErrorList {
	allErrs := field.ErrorList{}

	if parameter != nil && protocol != "UDP" {
		return append(allErrs, field.Forbidden(fieldPath, "is not allowed for non-UDP TransportServers"))
	}

	return validatePositiveIntOrZeroFromPointer(parameter, fieldPath)
}

func validateTransportServerAction(action *v1alpha1.Action, fieldPath *field.Path, upstreamNames sets.String) field.ErrorList {
	allErrs := field.ErrorList{}

	if action.Pass == "" {
		return append(allErrs, field.Required(fieldPath, "must specify pass"))
	}

	return validateReferencedUpstream(action.Pass, fieldPath.Child("pass"), upstreamNames)
}

package k8s

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	mergeableIngressTypeAnnotation        = "nginx.org/mergeable-ingress-type"
	lbMethodAnnotation                    = "nginx.org/lb-method"
	healthChecksAnnotation                = "nginx.com/health-checks"
	healthChecksMandatoryAnnotation       = "nginx.com/health-checks-mandatory"
	healthChecksMandatoryQueueAnnotation  = "nginx.com/health-checks-mandatory-queue"
	slowStartAnnotation                   = "nginx.com/slow-start"
	serverTokensAnnotation                = "nginx.org/server-tokens" // #nosec G101
	serverSnippetsAnnotation              = "nginx.org/server-snippets"
	locationSnippetsAnnotation            = "nginx.org/location-snippets"
	proxyConnectTimeoutAnnotation         = "nginx.org/proxy-connect-timeout"
	proxyReadTimeoutAnnotation            = "nginx.org/proxy-read-timeout"
	proxySendTimeoutAnnotation            = "nginx.org/proxy-send-timeout"
	proxyHideHeadersAnnotation            = "nginx.org/proxy-hide-headers"
	proxyPassHeadersAnnotation            = "nginx.org/proxy-pass-headers" // #nosec G101
	clientMaxBodySizeAnnotation           = "nginx.org/client-max-body-size"
	redirectToHTTPSAnnotation             = "nginx.org/redirect-to-https"
	sslRedirectAnnotation                 = "ingress.kubernetes.io/ssl-redirect"
	proxyBufferingAnnotation              = "nginx.org/proxy-buffering"
	hstsAnnotation                        = "nginx.org/hsts"
	hstsMaxAgeAnnotation                  = "nginx.org/hsts-max-age"
	hstsIncludeSubdomainsAnnotation       = "nginx.org/hsts-include-subdomains"
	hstsBehindProxyAnnotation             = "nginx.org/hsts-behind-proxy"
	proxyBuffersAnnotation                = "nginx.org/proxy-buffers"
	proxyBufferSizeAnnotation             = "nginx.org/proxy-buffer-size"
	proxyMaxTempFileSizeAnnotation        = "nginx.org/proxy-max-temp-file-size"
	upstreamZoneSizeAnnotation            = "nginx.org/upstream-zone-size"
	jwtRealmAnnotation                    = "nginx.com/jwt-realm"
	jwtKeyAnnotation                      = "nginx.com/jwt-key"
	jwtTokenAnnotation                    = "nginx.com/jwt-token" // #nosec G101
	jwtLoginURLAnnotation                 = "nginx.com/jwt-login-url"
	listenPortsAnnotation                 = "nginx.org/listen-ports"
	listenPortsSSLAnnotation              = "nginx.org/listen-ports-ssl"
	keepaliveAnnotation                   = "nginx.org/keepalive"
	maxFailsAnnotation                    = "nginx.org/max-fails"
	maxConnsAnnotation                    = "nginx.org/max-conns"
	failTimeoutAnnotation                 = "nginx.org/fail-timeout"
	appProtectEnableAnnotation            = "appprotect.f5.com/app-protect-enable"
	appProtectSecurityLogEnableAnnotation = "appprotect.f5.com/app-protect-security-log-enable"
	internalRouteAnnotation               = "nsm.nginx.com/internal-route"
	websocketServicesAnnotation           = "nginx.org/websocket-services"
	sslServicesAnnotation                 = "nginx.org/ssl-services"
	grpcServicesAnnotation                = "nginx.org/grpc-services"
	rewritesAnnotation                    = "nginx.org/rewrites"
	stickyCookieServicesAnnotation        = "nginx.com/sticky-cookie-services"
)

type annotationValidationContext struct {
	annotations           map[string]string
	specServices          map[string]bool
	name                  string
	value                 string
	isPlus                bool
	appProtectEnabled     bool
	internalRoutesEnabled bool
	fieldPath             *field.Path
}

type (
	annotationValidationFunc   func(context *annotationValidationContext) field.ErrorList
	annotationValidationConfig map[string][]annotationValidationFunc
	validatorFunc              func(val string) error
)

var (
	// annotationValidations defines the various validations which will be applied in order to each ingress annotation.
	// If any specified validation fails, the remaining validations for that annotation will not be run.
	annotationValidations = annotationValidationConfig{
		mergeableIngressTypeAnnotation: {
			validateRequiredAnnotation,
			validateMergeableIngressTypeAnnotation,
		},
		lbMethodAnnotation: {
			validateRequiredAnnotation,
			validateLBMethodAnnotation,
		},
		healthChecksAnnotation: {
			validatePlusOnlyAnnotation,
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		healthChecksMandatoryAnnotation: {
			validatePlusOnlyAnnotation,
			validateRelatedAnnotation(healthChecksAnnotation, validateIsTrue),
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		healthChecksMandatoryQueueAnnotation: {
			validatePlusOnlyAnnotation,
			validateRelatedAnnotation(healthChecksMandatoryAnnotation, validateIsTrue),
			validateRequiredAnnotation,
			validateUint64Annotation,
		},
		slowStartAnnotation: {
			validatePlusOnlyAnnotation,
			validateRequiredAnnotation,
			validateTimeAnnotation,
		},
		serverTokensAnnotation: {
			validateServerTokensAnnotation,
		},
		serverSnippetsAnnotation:   {},
		locationSnippetsAnnotation: {},
		proxyConnectTimeoutAnnotation: {
			validateRequiredAnnotation,
			validateTimeAnnotation,
		},
		proxyReadTimeoutAnnotation: {
			validateRequiredAnnotation,
			validateTimeAnnotation,
		},
		proxySendTimeoutAnnotation: {
			validateRequiredAnnotation,
			validateTimeAnnotation,
		},
		proxyHideHeadersAnnotation: {},
		proxyPassHeadersAnnotation: {},
		clientMaxBodySizeAnnotation: {
			validateRequiredAnnotation,
			validateOffsetAnnotation,
		},
		redirectToHTTPSAnnotation: {
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		sslRedirectAnnotation: {
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		proxyBufferingAnnotation: {
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		hstsAnnotation: {
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		hstsMaxAgeAnnotation: {
			validateRelatedAnnotation(hstsAnnotation, validateIsBool),
			validateRequiredAnnotation,
			validateInt64Annotation,
		},
		hstsIncludeSubdomainsAnnotation: {
			validateRelatedAnnotation(hstsAnnotation, validateIsBool),
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		hstsBehindProxyAnnotation: {
			validateRelatedAnnotation(hstsAnnotation, validateIsBool),
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		proxyBuffersAnnotation: {
			validateRequiredAnnotation,
			validateProxyBuffersAnnotation,
		},
		proxyBufferSizeAnnotation: {
			validateRequiredAnnotation,
			validateSizeAnnotation,
		},
		proxyMaxTempFileSizeAnnotation: {
			validateRequiredAnnotation,
			validateSizeAnnotation,
		},
		upstreamZoneSizeAnnotation: {
			validateRequiredAnnotation,
			validateSizeAnnotation,
		},
		jwtRealmAnnotation: {
			validatePlusOnlyAnnotation,
		},
		jwtKeyAnnotation: {
			validatePlusOnlyAnnotation,
		},
		jwtTokenAnnotation: {
			validatePlusOnlyAnnotation,
		},
		jwtLoginURLAnnotation: {
			validatePlusOnlyAnnotation,
		},
		listenPortsAnnotation: {
			validateRequiredAnnotation,
			validatePortListAnnotation,
		},
		listenPortsSSLAnnotation: {
			validateRequiredAnnotation,
			validatePortListAnnotation,
		},
		keepaliveAnnotation: {
			validateRequiredAnnotation,
			validateIntAnnotation,
		},
		maxFailsAnnotation: {
			validateRequiredAnnotation,
			validateUint64Annotation,
		},
		maxConnsAnnotation: {
			validateRequiredAnnotation,
			validateUint64Annotation,
		},
		failTimeoutAnnotation: {
			validateRequiredAnnotation,
			validateTimeAnnotation,
		},
		appProtectEnableAnnotation: {
			validateAppProtectOnlyAnnotation,
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		appProtectSecurityLogEnableAnnotation: {
			validateAppProtectOnlyAnnotation,
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		internalRouteAnnotation: {
			validateInternalRoutesOnlyAnnotation,
			validateRequiredAnnotation,
			validateBoolAnnotation,
		},
		websocketServicesAnnotation: {
			validateRequiredAnnotation,
			validateServiceListAnnotation,
		},
		sslServicesAnnotation: {
			validateRequiredAnnotation,
			validateServiceListAnnotation,
		},
		grpcServicesAnnotation: {
			validateRequiredAnnotation,
			validateServiceListAnnotation,
		},
		rewritesAnnotation: {
			validateRequiredAnnotation,
			validateRewriteListAnnotation,
		},
		stickyCookieServicesAnnotation: {
			validatePlusOnlyAnnotation,
			validateRequiredAnnotation,
			validateStickyServiceListAnnotation,
		},
	}
	annotationNames = sortedAnnotationNames(annotationValidations)
)

func sortedAnnotationNames(annotationValidations annotationValidationConfig) []string {
	sortedNames := make([]string, 0)
	for annotationName := range annotationValidations {
		sortedNames = append(sortedNames, annotationName)
	}
	sort.Strings(sortedNames)
	return sortedNames
}

// validateIngress validate an Ingress resource with rules that our Ingress Controller enforces.
// Note that the full validation of Ingress resources is done by Kubernetes.
func validateIngress(
	ing *networking.Ingress,
	isPlus bool,
	appProtectEnabled bool,
	internalRoutesEnabled bool,
) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateIngressAnnotations(
		ing.Annotations,
		getSpecServices(ing.Spec),
		isPlus,
		appProtectEnabled,
		internalRoutesEnabled,
		field.NewPath("annotations"),
	)...)

	allErrs = append(allErrs, validateIngressSpec(&ing.Spec, field.NewPath("spec"))...)

	if isMaster(ing) {
		allErrs = append(allErrs, validateMasterSpec(&ing.Spec, field.NewPath("spec"))...)
	} else if isMinion(ing) {
		allErrs = append(allErrs, validateMinionSpec(&ing.Spec, field.NewPath("spec"))...)
	}

	return allErrs
}

func validateIngressAnnotations(
	annotations map[string]string,
	specServices map[string]bool,
	isPlus bool,
	appProtectEnabled bool,
	internalRoutesEnabled bool,
	fieldPath *field.Path,
) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, name := range annotationNames {
		if value, exists := annotations[name]; exists {
			context := &annotationValidationContext{
				annotations:           annotations,
				specServices:          specServices,
				name:                  name,
				value:                 value,
				isPlus:                isPlus,
				appProtectEnabled:     appProtectEnabled,
				internalRoutesEnabled: internalRoutesEnabled,
				fieldPath:             fieldPath.Child(name),
			}
			allErrs = append(allErrs, validateIngressAnnotation(context)...)
		}
	}

	return allErrs
}

func validateIngressAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if validationFuncs, exists := annotationValidations[context.name]; exists {
		for _, validationFunc := range validationFuncs {
			valErrors := validationFunc(context)
			if len(valErrors) > 0 {
				allErrs = append(allErrs, valErrors...)
				break
			}
		}
	}
	return allErrs
}

func validateRelatedAnnotation(name string, validator validatorFunc) annotationValidationFunc {
	return func(context *annotationValidationContext) field.ErrorList {
		allErrs := field.ErrorList{}
		val, exists := context.annotations[name]
		if !exists {
			return append(allErrs, field.Forbidden(context.fieldPath, fmt.Sprintf("related annotation %s: must be set", name)))
		}

		if err := validator(val); err != nil {
			return append(allErrs, field.Forbidden(context.fieldPath, fmt.Sprintf("related annotation %s: %s", name, err.Error())))
		}
		return allErrs
	}
}

func validateMergeableIngressTypeAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if context.value != "master" && context.value != "minion" {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be one of: 'master' or 'minion'"))
	}
	return allErrs
}

func validateLBMethodAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}

	parseFunc := configs.ParseLBMethod
	if context.isPlus {
		parseFunc = configs.ParseLBMethodForPlus
	}

	if _, err := parseFunc(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, err.Error()))
	}
	return allErrs
}

func validateServerTokensAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if !context.isPlus {
		if _, err := configs.ParseBool(context.value); err != nil {
			return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a boolean"))
		}
	}
	return allErrs
}

func validateRequiredAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if context.value == "" {
		return append(allErrs, field.Required(context.fieldPath, ""))
	}
	return allErrs
}

func validatePlusOnlyAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if !context.isPlus {
		return append(allErrs, field.Forbidden(context.fieldPath, "annotation requires NGINX Plus"))
	}
	return allErrs
}

func validateAppProtectOnlyAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if !context.appProtectEnabled {
		return append(allErrs, field.Forbidden(context.fieldPath, "annotation requires AppProtect"))
	}
	return allErrs
}

func validateInternalRoutesOnlyAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if !context.internalRoutesEnabled {
		return append(allErrs, field.Forbidden(context.fieldPath, "annotation requires Internal Routes enabled"))
	}
	return allErrs
}

func validateBoolAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseBool(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a boolean"))
	}
	return allErrs
}

func validateTimeAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseTime(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a time"))
	}
	return allErrs
}

func validateOffsetAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseOffset(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be an offset"))
	}
	return allErrs
}

func validateSizeAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseSize(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a size"))
	}
	return allErrs
}

func validateProxyBuffersAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseProxyBuffersSpec(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a proxy buffer spec"))
	}
	return allErrs
}

func validateUint64Annotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseUint64(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a non-negative integer"))
	}
	return allErrs
}

func validateInt64Annotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseInt64(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be an integer"))
	}
	return allErrs
}

func validateIntAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseInt(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be an integer"))
	}
	return allErrs
}

func validatePortListAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParsePortList(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a comma-separated list of port numbers"))
	}
	return allErrs
}

func validateServiceListAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	var unknownServices []string
	annotationServices := configs.ParseServiceList(context.value)
	for svc := range annotationServices {
		if _, exists := context.specServices[svc]; !exists {
			unknownServices = append(unknownServices, svc)
		}
	}
	if len(unknownServices) > 0 {
		errorMsg := fmt.Sprintf(
			"must be a comma-separated list of services. The following services were not found: %s",
			strings.Join(unknownServices, ","),
		)
		return append(allErrs, field.Invalid(context.fieldPath, context.value, errorMsg))
	}
	return allErrs
}

func validateStickyServiceListAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseStickyServiceList(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a semicolon-separated list of sticky services"))
	}
	return allErrs
}

func validateRewriteListAnnotation(context *annotationValidationContext) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, err := configs.ParseRewriteList(context.value); err != nil {
		return append(allErrs, field.Invalid(context.fieldPath, context.value, "must be a semicolon-separated list of rewrites"))
	}
	return allErrs
}

func validateIsBool(v string) error {
	_, err := configs.ParseBool(v)
	return err
}

func validateIsTrue(v string) error {
	b, err := configs.ParseBool(v)
	if err != nil {
		return err
	}
	if !b {
		return errors.New("must be true")
	}
	return nil
}

func validateIngressSpec(spec *networking.IngressSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.DefaultBackend != nil {
		allErrs = append(allErrs, validateBackend(spec.DefaultBackend, fieldPath.Child("defaultBackend"))...)
	}

	allHosts := sets.String{}

	if len(spec.Rules) == 0 {
		return append(allErrs, field.Required(fieldPath.Child("rules"), ""))
	}

	for i, r := range spec.Rules {
		idxRule := fieldPath.Child("rules").Index(i)

		if r.Host == "" {
			allErrs = append(allErrs, field.Required(idxRule.Child("host"), ""))
		} else if allHosts.Has(r.Host) {
			allErrs = append(allErrs, field.Duplicate(idxRule.Child("host"), r.Host))
		} else {
			allHosts.Insert(r.Host)
		}

		if r.HTTP == nil {
			continue
		}

		for _, path := range r.HTTP.Paths {
			idxPath := idxRule.Child("http").Child("path").Index(i)

			allErrs = append(allErrs, validateBackend(&path.Backend, idxPath.Child("backend"))...)
		}
	}

	return allErrs
}

func validateBackend(backend *networking.IngressBackend, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if backend.Resource != nil {
		return append(allErrs, field.Forbidden(fieldPath.Child("resource"), "resource backends are not supported"))
	}

	return allErrs
}

func validateMasterSpec(spec *networking.IngressSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(spec.Rules) != 1 {
		return append(allErrs, field.TooMany(fieldPath.Child("rules"), len(spec.Rules), 1))
	}

	// the number of paths of the first rule of the spec must be 0
	if spec.Rules[0].HTTP != nil && len(spec.Rules[0].HTTP.Paths) > 0 {
		pathsField := fieldPath.Child("rules").Index(0).Child("http").Child("paths")
		return append(allErrs, field.TooMany(pathsField, len(spec.Rules[0].HTTP.Paths), 0))
	}

	return allErrs
}

func validateMinionSpec(spec *networking.IngressSpec, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(spec.TLS) > 0 {
		allErrs = append(allErrs, field.TooMany(fieldPath.Child("tls"), len(spec.TLS), 0))
	}

	if len(spec.Rules) != 1 {
		return append(allErrs, field.TooMany(fieldPath.Child("rules"), len(spec.Rules), 1))
	}

	// the number of paths of the first rule of the spec must be greater than 0
	if spec.Rules[0].HTTP == nil || len(spec.Rules[0].HTTP.Paths) == 0 {
		pathsField := fieldPath.Child("rules").Index(0).Child("http").Child("paths")
		return append(allErrs, field.Required(pathsField, "must include at least one path"))
	}

	return allErrs
}

func getSpecServices(ingressSpec networking.IngressSpec) map[string]bool {
	services := make(map[string]bool)
	if ingressSpec.DefaultBackend != nil && ingressSpec.DefaultBackend.Service != nil {
		services[ingressSpec.DefaultBackend.Service.Name] = true
	}
	for _, rule := range ingressSpec.Rules {
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil {
					services[path.Backend.Service.Name] = true
				}
			}
		}
	}
	return services
}

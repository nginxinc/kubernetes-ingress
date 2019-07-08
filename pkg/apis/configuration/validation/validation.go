package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateVirtualServer validates a VirtualServer.
func ValidateVirtualServer(virtualServer *v1alpha1.VirtualServer, isPlus bool) error {
	allErrs := validateVirtualServerSpec(&virtualServer.Spec, field.NewPath("spec"), isPlus)
	return allErrs.ToAggregate()
}

// validateVirtualServerSpec validates a VirtualServerSpec.
func validateVirtualServerSpec(spec *v1alpha1.VirtualServerSpec, fieldPath *field.Path, isPlus bool) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateHost(spec.Host, fieldPath.Child("host"))...)
	allErrs = append(allErrs, validateTLS(spec.TLS, fieldPath.Child("tls"))...)

	upstreamErrs, upstreamNames := validateUpstreams(spec.Upstreams, fieldPath.Child("upstreams"), isPlus)
	allErrs = append(allErrs, upstreamErrs...)

	allErrs = append(allErrs, validateVirtualServerRoutes(spec.Routes, fieldPath.Child("routes"), upstreamNames)...)

	return allErrs
}

func validateHost(host string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if host == "" {
		return append(allErrs, field.Required(fieldPath, ""))
	}

	for _, msg := range validation.IsDNS1123Subdomain(host) {
		allErrs = append(allErrs, field.Invalid(fieldPath, host, msg))
	}

	return allErrs
}

func validateTLS(tls *v1alpha1.TLS, fieldPath *field.Path) field.ErrorList {
	if tls == nil {
		// valid case - tls is not defined
		return field.ErrorList{}
	}

	return validateSecretName(tls.Secret, fieldPath.Child("secret"))
}

func validatePositiveIntOrZero(n *int, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if n == nil {
		return allErrs
	}

	if *n < 0 {
		return append(allErrs, field.Invalid(fieldPath, n, "must be positive or zero"))
	}

	return allErrs
}

func validateTime(time string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if time == "" {
		return allErrs
	}

	if _, err := configs.ParseTime(time); err != nil {
		return append(allErrs, field.Invalid(fieldPath, time, err.Error()))
	}

	return allErrs
}

func validateUpstreamLBMethod(lBMethod string, fieldPath *field.Path, isPlus bool) field.ErrorList {
	allErrs := field.ErrorList{}
	if lBMethod == "" {
		return allErrs
	}

	if isPlus {
		_, err := configs.ParseLBMethodForPlus(lBMethod)
		if err != nil {
			return append(allErrs, field.Invalid(fieldPath, lBMethod, err.Error()))
		}
	} else {
		_, err := configs.ParseLBMethod(lBMethod)
		if err != nil {
			return append(allErrs, field.Invalid(fieldPath, lBMethod, err.Error()))
		}
	}

	return allErrs
}

// validateSecretName checks if a secret name is valid.
// It performs the same validation as ValidateSecretName from k8s.io/kubernetes/pkg/apis/core/validation/validation.go.
func validateSecretName(name string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if name == "" {
		return append(allErrs, field.Required(fieldPath, ""))
	}

	for _, msg := range validation.IsDNS1123Subdomain(name) {
		allErrs = append(allErrs, field.Invalid(fieldPath, name, msg))
	}

	return allErrs
}

func validateUpstreams(upstreams []v1alpha1.Upstream, fieldPath *field.Path, isPlus bool) (allErrs field.ErrorList, upstreamNames sets.String) {
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

		allErrs = append(allErrs, validateTime(u.ProxyConnectTimeout, idxPath.Child("connect-timeout"))...)
		allErrs = append(allErrs, validateTime(u.ProxyReadTimeout, idxPath.Child("read-timeout"))...)
		allErrs = append(allErrs, validateTime(u.ProxySendTimeout, idxPath.Child("send-timeout"))...)

		allErrs = append(allErrs, validateUpstreamLBMethod(u.LBMethod, idxPath.Child("lb-method"), isPlus)...)
		allErrs = append(allErrs, validateTime(u.FailTimeout, idxPath.Child("fail-timeout"))...)
		allErrs = append(allErrs, validatePositiveIntOrZero(u.MaxFails, idxPath.Child("max-fails"))...)
		allErrs = append(allErrs, validatePositiveIntOrZero(u.Keepalive, idxPath.Child("keepalive"))...)

		for _, msg := range validation.IsValidPortNum(int(u.Port)) {
			allErrs = append(allErrs, field.Invalid(idxPath.Child("port"), u.Port, msg))
		}
	}

	return allErrs, upstreamNames
}

// validateUpstreamName checks is an upstream name is valid.
// The rules for NGINX upstream names are less strict than IsDNS1035Label.
// However, it is convenient to enforce IsDNS1035Label in the yaml for
// the names of upstreams.
func validateUpstreamName(name string, fieldPath *field.Path) field.ErrorList {
	return validateDNS1035Label(name, fieldPath)
}

// validateServiceName checks if a service name is valid.
// It performs the same validation as ValidateServiceName from k8s.io/kubernetes/pkg/apis/core/validation/validation.go.
func validateServiceName(name string, fieldPath *field.Path) field.ErrorList {
	return validateDNS1035Label(name, fieldPath)
}

func validateDNS1035Label(name string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if name == "" {
		return append(allErrs, field.Required(fieldPath, ""))
	}

	for _, msg := range validation.IsDNS1035Label(name) {
		allErrs = append(allErrs, field.Invalid(fieldPath, name, msg))
	}

	return allErrs
}

func validateVirtualServerRoutes(routes []v1alpha1.Route, fieldPath *field.Path, upstreamNames sets.String) field.ErrorList {
	allErrs := field.ErrorList{}

	allPaths := sets.String{}

	for i, r := range routes {
		idxPath := fieldPath.Index(i)

		isRouteFieldForbidden := false
		routeErrs := validateRoute(r, idxPath, upstreamNames, isRouteFieldForbidden)
		if len(routeErrs) > 0 {
			allErrs = append(allErrs, routeErrs...)
		} else if allPaths.Has(r.Path) {
			allErrs = append(allErrs, field.Duplicate(idxPath.Child("path"), r.Path))
		} else {
			allPaths.Insert(r.Path)
		}
	}

	return allErrs
}

func validateRoute(route v1alpha1.Route, fieldPath *field.Path, upstreamNames sets.String, isRouteFieldForbidden bool) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validatePath(route.Path, fieldPath.Child("path"))...)

	fieldCount := 0

	if route.Upstream != "" {
		allErrs = append(allErrs, validateReferencedUpstream(route.Upstream, fieldPath.Child("upstream"), upstreamNames)...)
		fieldCount++
	}

	if len(route.Splits) > 0 {
		allErrs = append(allErrs, validateSplits(route.Splits, fieldPath.Child("splits"), upstreamNames)...)
		fieldCount++
	}

	if route.Rules != nil {
		allErrs = append(allErrs, validateRules(route.Rules, fieldPath.Child("rules"), upstreamNames)...)
		fieldCount++
	}

	if route.Route != "" {
		if isRouteFieldForbidden {
			allErrs = append(allErrs, field.Forbidden(fieldPath.Child("route"), "is not allowed"))
		} else {
			allErrs = append(allErrs, validateRouteField(route.Route, fieldPath.Child("route"))...)
			fieldCount++
		}
	}

	if fieldCount != 1 {
		msg := "must specify exactly one of: `upstream`, `splits`, `rules` or `route`"
		if isRouteFieldForbidden {
			msg = "must specify exactly one of: `upstream`, `splits` or `rules`"
		}

		allErrs = append(allErrs, field.Invalid(fieldPath, "", msg))
	}

	return allErrs
}

func validateRouteField(value string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, msg := range validation.IsQualifiedName(value) {
		allErrs = append(allErrs, field.Invalid(fieldPath, value, msg))
	}

	return allErrs
}

func validateReferencedUpstream(name string, fieldPath *field.Path, upstreamNames sets.String) field.ErrorList {
	allErrs := field.ErrorList{}

	upstreamErrs := validateUpstreamName(name, fieldPath)
	if len(upstreamErrs) > 0 {
		allErrs = append(allErrs, upstreamErrs...)
	} else if !upstreamNames.Has(name) {
		allErrs = append(allErrs, field.NotFound(fieldPath, name))
	}

	return allErrs
}

func validateSplits(splits []v1alpha1.Split, fieldPath *field.Path, upstreamNames sets.String) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(splits) < 2 {
		return append(allErrs, field.Invalid(fieldPath, "", "must include at least 2 splits"))
	}

	totalWeight := 0

	for i, s := range splits {
		idxPath := fieldPath.Index(i)

		for _, msg := range validation.IsInRange(s.Weight, 1, 99) {
			allErrs = append(allErrs, field.Invalid(idxPath.Child("weight"), s.Weight, msg))
		}

		allErrs = append(allErrs, validateReferencedUpstream(s.Upstream, idxPath.Child("upstream"), upstreamNames)...)

		totalWeight += s.Weight
	}

	if totalWeight != 100 {
		allErrs = append(allErrs, field.Invalid(fieldPath, "", "the sum of the weights of all splits must be equal to 100"))
	}

	return allErrs
}

// For now, we only support prefix-based NGINX locations. For example, location /abc { ... }.
const pathFmt = `/[^\s{};]*`
const pathErrMsg = "must start with / and must not include any whitespace character, `{`, `}` or `;`"

var pathRegexp = regexp.MustCompile("^" + pathFmt + "$")

func validatePath(path string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if path == "" {
		return append(allErrs, field.Required(fieldPath, ""))
	}

	if !pathRegexp.MatchString(path) {
		msg := validation.RegexError(pathErrMsg, pathFmt, "/", "/path", "/path/subpath-123")
		return append(allErrs, field.Invalid(fieldPath, path, msg))
	}

	return allErrs
}

func validateRules(rules *v1alpha1.Rules, fieldPath *field.Path, upstreamNames sets.String) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(rules.Conditions) == 0 {
		allErrs = append(allErrs, field.Required(fieldPath.Child("conditions"), "must specify at least one condition"))
	} else {
		for i, c := range rules.Conditions {
			allErrs = append(allErrs, validateCondition(c, fieldPath.Child("conditions").Index(i))...)
		}
	}

	if len(rules.Matches) == 0 {
		allErrs = append(allErrs, field.Required(fieldPath.Child("matches"), "must specify at least one match"))
	} else {
		for i, m := range rules.Matches {
			allErrs = append(allErrs, validateMatch(m, fieldPath.Child("matches").Index(i), len(rules.Conditions), upstreamNames)...)
		}
	}

	allErrs = append(allErrs, validateReferencedUpstream(rules.DefaultUpstream, fieldPath.Child("defaultUpstream"), upstreamNames)...)

	return allErrs
}

func validateCondition(condition v1alpha1.Condition, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	fieldCount := 0

	if condition.Header != "" {
		for _, msg := range validation.IsHTTPHeaderName(condition.Header) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("header"), condition.Header, msg))
		}
		fieldCount++
	}

	if condition.Cookie != "" {
		for _, msg := range isCookieName(condition.Cookie) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("cookie"), condition.Cookie, msg))
		}
		fieldCount++
	}

	if condition.Argument != "" {
		for _, msg := range isArgumentName(condition.Argument) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("argument"), condition.Argument, msg))
		}
		fieldCount++
	}

	if condition.Variable != "" {
		allErrs = append(allErrs, validateVariableName(condition.Variable, fieldPath.Child("variable"))...)
		fieldCount++
	}

	if fieldCount != 1 {
		allErrs = append(allErrs, field.Invalid(fieldPath, "", "must specify exactly one of: `header`, `cookie`, `argument` or `variable`"))
	}

	return allErrs
}

const cookieNameFmt string = "[_A-Za-z0-9]+"
const cookieNameErrMsg string = "a valid cookie name must consist of alphanumeric characters or '_'"

var cookieNameRegexp = regexp.MustCompile("^" + cookieNameFmt + "$")

func isCookieName(value string) []string {
	if !cookieNameRegexp.MatchString(value) {
		return []string{validation.RegexError(cookieNameErrMsg, cookieNameFmt, "my_cookie_123")}
	}
	return nil
}

const argumentNameFmt string = "[_A-Za-z0-9]+"
const argumentNameErrMsg string = "a valid argument name must consist of alphanumeric characters or '_'"

var argumentNameRegexp = regexp.MustCompile("^" + argumentNameFmt + "$")

func isArgumentName(value string) []string {
	if !argumentNameRegexp.MatchString(value) {
		return []string{validation.RegexError(argumentNameErrMsg, argumentNameFmt, "argument_123")}
	}
	return nil
}

// validVariableNames includes NGINX variables allowed to be used in conditions.
// Not all NGINX variables are allowed. The full list of NGINX variables is at https://nginx.org/en/docs/varindex.html
var validVariableNames = map[string]bool{
	"$args":           true,
	"$http2":          true,
	"$https":          true,
	"$remote_addr":    true,
	"$remote_port":    true,
	"$query_string":   true,
	"$request":        true,
	"$request_body":   true,
	"$request_uri":    true,
	"$request_method": true,
	"$scheme":         true,
}

func validateVariableName(name string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !strings.HasPrefix(name, "$") {
		return append(allErrs, field.Invalid(fieldPath, name, "must start with `$`"))
	}

	if _, exists := validVariableNames[name]; !exists {
		return append(allErrs, field.Invalid(fieldPath, name, "is not allowed or is not an NGINX variable"))
	}

	return allErrs
}

func validateMatch(match v1alpha1.Match, fieldPath *field.Path, conditionsCount int, upstreamNames sets.String) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(match.Values) != conditionsCount {
		msg := fmt.Sprintf("must specify %d values (same as the number of conditions)", conditionsCount)
		allErrs = append(allErrs, field.Invalid(fieldPath.Child("values"), "", msg))
	}

	for i, v := range match.Values {
		for _, msg := range isValidMatchValue(v) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("values").Index(i), v, msg))
		}
	}

	allErrs = append(allErrs, validateReferencedUpstream(match.Upstream, fieldPath.Child("upstream"), upstreamNames)...)

	return allErrs
}

const matchValueFmt string = `([^"\\]|\\.)*`
const matchValueErrMsg string = `a valid value must have all '"' (double quotes) escaped and must not end with an unescaped '\' (backslash)`

var matchValueRegexp = regexp.MustCompile("^" + matchValueFmt + "$")

func isValidMatchValue(value string) []string {
	if !matchValueRegexp.MatchString(value) {
		return []string{validation.RegexError(matchValueErrMsg, matchValueFmt, "value-123")}
	}
	return nil
}

// ValidateVirtualServerRoute validates a VirtualServerRoute.
func ValidateVirtualServerRoute(virtualServerRoute *v1alpha1.VirtualServerRoute, isPlus bool) error {
	allErrs := validateVirtualServerRouteSpec(&virtualServerRoute.Spec, field.NewPath("spec"), "", "/", isPlus)
	return allErrs.ToAggregate()
}

// ValidateVirtualServerRouteForVirtualServer validates a VirtualServerRoute for a VirtualServer represented by its host and path prefix.
func ValidateVirtualServerRouteForVirtualServer(virtualServerRoute *v1alpha1.VirtualServerRoute, virtualServerHost string, pathPrefix string, isPlus bool) error {
	allErrs := validateVirtualServerRouteSpec(&virtualServerRoute.Spec, field.NewPath("spec"), virtualServerHost, pathPrefix, isPlus)
	return allErrs.ToAggregate()
}

func validateVirtualServerRouteSpec(spec *v1alpha1.VirtualServerRouteSpec, fieldPath *field.Path, virtualServerHost string, pathPrefix string, isPlus bool) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateVirtualServerRouteHost(spec.Host, virtualServerHost, fieldPath.Child("host"))...)

	upstreamErrs, upstreamNames := validateUpstreams(spec.Upstreams, fieldPath.Child("upstreams"), isPlus)
	allErrs = append(allErrs, upstreamErrs...)

	allErrs = append(allErrs, validateVirtualServerRouteSubroutes(spec.Subroutes, fieldPath.Child("subroutes"), upstreamNames, pathPrefix)...)

	return allErrs
}

func validateVirtualServerRouteHost(host string, virtualServerHost string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateHost(host, fieldPath)...)

	if virtualServerHost != "" && host != virtualServerHost {
		msg := fmt.Sprintf("must be equal to '%s'", virtualServerHost)
		allErrs = append(allErrs, field.Invalid(fieldPath, host, msg))
	}

	return allErrs
}

func validateVirtualServerRouteSubroutes(routes []v1alpha1.Route, fieldPath *field.Path, upstreamNames sets.String, pathPrefix string) field.ErrorList {
	allErrs := field.ErrorList{}

	allPaths := sets.String{}

	for i, r := range routes {
		idxPath := fieldPath.Index(i)

		isRouteFieldForbidden := true
		routeErrs := validateRoute(r, idxPath, upstreamNames, isRouteFieldForbidden)

		if pathPrefix != "" && !strings.HasPrefix(r.Path, pathPrefix) {
			msg := fmt.Sprintf("must start with '%s'", pathPrefix)
			routeErrs = append(routeErrs, field.Invalid(idxPath, r.Path, msg))
		}

		if len(routeErrs) > 0 {
			allErrs = append(allErrs, routeErrs...)
		} else if allPaths.Has(r.Path) {
			allErrs = append(allErrs, field.Duplicate(idxPath.Child("path"), r.Path))
		} else {
			allPaths.Insert(r.Path)
		}
	}

	return allErrs
}

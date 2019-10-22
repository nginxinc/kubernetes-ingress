package validation

import (
	"fmt"
	"regexp"
	"strconv"
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

func validatePositiveIntOrZero(n int, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if n < 0 {
		return append(allErrs, field.Invalid(fieldPath, n, "must be positive"))
	}

	return allErrs
}

func validatePositiveIntOrZeroFromPointer(n *int, fieldPath *field.Path) field.ErrorList {
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

// http://nginx.org/en/docs/syntax.html
const offsetFmt = `\d+[kKmMgG]?`
const offsetErrMsg = "must consist of numeric characters followed by a valid size suffix. 'k|K|m|M|g|G"

var offsetRegexp = regexp.MustCompile("^" + offsetFmt + "$")

func validateOffset(offset string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if offset == "" {
		return allErrs
	}

	if !offsetRegexp.MatchString(offset) {
		msg := validation.RegexError(offsetErrMsg, offsetFmt, "16", "32k", "64M")
		return append(allErrs, field.Invalid(fieldPath, offset, msg))
	}

	return allErrs
}

const sizeFmt = `\d+[kKmM]?`
const sizeErrMsg = "must consist of numeric characters followed by a valid size suffix. 'k|K|m|M"

var sizeRegexp = regexp.MustCompile("^" + sizeFmt + "$")

func validateSize(size string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if size == "" {
		return allErrs
	}

	if !sizeRegexp.MatchString(size) {
		msg := validation.RegexError(sizeErrMsg, sizeFmt, "16", "32k", "64M")
		return append(allErrs, field.Invalid(fieldPath, size, msg))
	}
	return allErrs
}

func validateBuffer(buff *v1alpha1.UpstreamBuffers, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if buff == nil {
		return allErrs
	}

	if buff.Number <= 0 {
		allErrs = append(allErrs, field.Invalid(fieldPath.Child("number"), buff.Number, "must be positive"))
	}

	if buff.Size == "" {
		allErrs = append(allErrs, field.Required(fieldPath.Child("size"), "cannot be empty"))
	} else {
		allErrs = append(allErrs, validateSize(buff.Size, fieldPath.Child("size"))...)
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

func validateUpstreamHealthCheck(hc *v1alpha1.HealthCheck, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if hc == nil {
		return allErrs
	}

	if hc.Path != "" {
		allErrs = append(allErrs, validatePath(hc.Path, fieldPath.Child("path"))...)
	}

	allErrs = append(allErrs, validateTime(hc.Interval, fieldPath.Child("interval"))...)
	allErrs = append(allErrs, validateTime(hc.Jitter, fieldPath.Child("jitter"))...)
	allErrs = append(allErrs, validatePositiveIntOrZero(hc.Fails, fieldPath.Child("fails"))...)
	allErrs = append(allErrs, validatePositiveIntOrZero(hc.Passes, fieldPath.Child("passes"))...)
	allErrs = append(allErrs, validateTime(hc.ConnectTimeout, fieldPath.Child("connect-timeout"))...)
	allErrs = append(allErrs, validateTime(hc.ReadTimeout, fieldPath.Child("read-timeout"))...)
	allErrs = append(allErrs, validateTime(hc.SendTimeout, fieldPath.Child("send-timeout"))...)
	allErrs = append(allErrs, validateStatusMatch(hc.StatusMatch, fieldPath.Child("statusMatch"))...)

	for i, header := range hc.Headers {
		idxPath := fieldPath.Child("headers").Index(i)
		allErrs = append(allErrs, validateHeader(header, idxPath)...)
	}

	if hc.Port > 0 {
		for _, msg := range validation.IsValidPortNum(hc.Port) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("port"), hc.Port, msg))
		}
	}

	return allErrs
}

func validateSessionCookie(sc *v1alpha1.SessionCookie, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if sc == nil {
		return allErrs
	}

	if sc.Name == "" {
		allErrs = append(allErrs, field.Required(fieldPath.Child("name"), ""))
	} else {
		for _, msg := range isCookieName(sc.Name) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("name"), sc.Name, msg))
		}
	}

	if sc.Path != "" {
		allErrs = append(allErrs, validatePath(sc.Path, fieldPath.Child("path"))...)
	}

	if sc.Expires != "max" {
		allErrs = append(allErrs, validateTime(sc.Expires, fieldPath.Child("expires"))...)
	}

	if sc.Domain != "" {
		// A Domain prefix of "." is allowed.
		domain := strings.TrimPrefix(sc.Domain, ".")
		for _, msg := range validation.IsDNS1123Subdomain(domain) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("domain"), sc.Domain, msg))
		}
	}

	return allErrs
}

func validateStatusMatch(s string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if s == "" {
		return allErrs
	}

	if strings.HasPrefix(s, "!") {
		if !strings.HasPrefix(s, "! ") {
			allErrs = append(allErrs, field.Invalid(fieldPath, s, "must have an space character after the `!`"))
		}
	}

	statuses := strings.Split(s, " ")
	for i, value := range statuses {

		if value == "!" {
			if i != 0 {
				allErrs = append(allErrs, field.Invalid(fieldPath, s, "`!` can only appear once at the beginning"))
			}
		} else if strings.Contains(value, "-") {
			if msg := validateStatusCodeRange(value); msg != "" {
				allErrs = append(allErrs, field.Invalid(fieldPath, s, msg))
			}
		} else if msg := validateStatusCode(value); msg != "" {
			allErrs = append(allErrs, field.Invalid(fieldPath, s, msg))
		}
	}

	return allErrs
}

func validateStatusCodeRange(statusRangeStr string) string {
	statusRange := strings.Split(statusRangeStr, "-")
	if len(statusRange) != 2 {
		return "ranges must only have 2 numbers"
	}

	min, msg := validateIntFromString(statusRange[0])
	if msg != "" {
		return msg
	}

	max, msg := validateIntFromString(statusRange[1])
	if msg != "" {
		return msg
	}

	for _, code := range statusRange {
		if msg := validateStatusCode(code); msg != "" {
			return msg
		}
	}

	if max <= min {
		return fmt.Sprintf("range limits must be %v < %v", min, max)
	}

	return ""
}

func validateIntFromString(number string) (int, string) {
	numberInt, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		return 0, fmt.Sprintf("%v must be a valid integer", number)
	}

	return int(numberInt), ""
}

func validateStatusCode(status string) string {
	code, errMsg := validateIntFromString(status)
	if errMsg != "" {
		return errMsg
	}

	if code < 100 || code > 999 {
		return validation.InclusiveRangeError(100, 999)
	}

	return ""
}

func validateHeader(h v1alpha1.Header, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if h.Name == "" {
		allErrs = append(allErrs, field.Required(fieldPath.Child("name"), ""))
	}

	for _, msg := range validation.IsHTTPHeaderName(h.Name) {
		allErrs = append(allErrs, field.Invalid(fieldPath.Child("name"), h.Name, msg))
	}

	for _, msg := range isValidHeaderValue(h.Value) {
		allErrs = append(allErrs, field.Invalid(fieldPath.Child("value"), h.Value, msg))
	}

	return allErrs
}

const headerValueFmt = `([^"$\\]|\\[^$])*`
const headerValueFmtErrMsg string = `a valid header must have all '"' escaped and must not contain any '$' or end with an unescaped '\'`

var headerValueFmtRegexp = regexp.MustCompile("^" + headerValueFmt + "$")

func isValidHeaderValue(s string) []string {
	if !headerValueFmtRegexp.MatchString(s) {
		return []string{validation.RegexError(headerValueFmtErrMsg, headerValueFmt, "my.service", "foo")}
	}
	return nil
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
		allErrs = append(allErrs, validateLabels(u.Subselector, idxPath.Child("subselector"))...)
		allErrs = append(allErrs, validateTime(u.ProxyConnectTimeout, idxPath.Child("connect-timeout"))...)
		allErrs = append(allErrs, validateTime(u.ProxyReadTimeout, idxPath.Child("read-timeout"))...)
		allErrs = append(allErrs, validateTime(u.ProxySendTimeout, idxPath.Child("send-timeout"))...)
		allErrs = append(allErrs, validateNextUpstream(u.ProxyNextUpstream, idxPath.Child("next-upstream"))...)
		allErrs = append(allErrs, validateTime(u.ProxyNextUpstreamTimeout, idxPath.Child("next-upstream-timeout"))...)
		allErrs = append(allErrs, validatePositiveIntOrZeroFromPointer(&u.ProxyNextUpstreamTries, idxPath.Child("next-upstream-tries"))...)
		allErrs = append(allErrs, validateUpstreamLBMethod(u.LBMethod, idxPath.Child("lb-method"), isPlus)...)
		allErrs = append(allErrs, validateTime(u.FailTimeout, idxPath.Child("fail-timeout"))...)
		allErrs = append(allErrs, validatePositiveIntOrZeroFromPointer(u.MaxFails, idxPath.Child("max-fails"))...)
		allErrs = append(allErrs, validatePositiveIntOrZeroFromPointer(u.Keepalive, idxPath.Child("keepalive"))...)
		allErrs = append(allErrs, validatePositiveIntOrZeroFromPointer(u.MaxConns, idxPath.Child("max-conns"))...)
		allErrs = append(allErrs, validateOffset(u.ClientMaxBodySize, idxPath.Child("client-max-body-size"))...)
		allErrs = append(allErrs, validateUpstreamHealthCheck(u.HealthCheck, idxPath.Child("healthCheck"))...)
		allErrs = append(allErrs, validateTime(u.SlowStart, idxPath.Child("slow-start"))...)
		allErrs = append(allErrs, validateBuffer(u.ProxyBuffers, idxPath.Child("buffers"))...)
		allErrs = append(allErrs, validateSize(u.ProxyBufferSize, idxPath.Child("buffer-size"))...)
		allErrs = append(allErrs, rejectPlusResourcesInOSS(u, idxPath, isPlus)...)
		allErrs = append(allErrs, validateQueue(u.Queue, idxPath.Child("queue"), isPlus)...)
		allErrs = append(allErrs, validateSessionCookie(u.SessionCookie, idxPath.Child("sessionCookie"))...)

		for _, msg := range validation.IsValidPortNum(int(u.Port)) {
			allErrs = append(allErrs, field.Invalid(idxPath.Child("port"), u.Port, msg))
		}
	}

	return allErrs, upstreamNames
}

var validNextUpstreamParams = map[string]bool{
	"error":          true,
	"timeout":        true,
	"invalid_header": true,
	"http_500":       true,
	"http_502":       true,
	"http_503":       true,
	"http_504":       true,
	"http_403":       true,
	"http_404":       true,
	"http_429":       true,
	"non_idempotent": true,
	"off":            true,
	"":               true,
}

// validateNextUpstream checks the values given for passing queries to a upstream
func validateNextUpstream(nextUpstream string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allParams := sets.String{}
	if nextUpstream == "" {
		return allErrs
	}
	params := strings.Fields(nextUpstream)
	for _, para := range params {
		if !validNextUpstreamParams[para] {
			allErrs = append(allErrs, field.Invalid(fieldPath, para, "not a valid parameter"))
		}
		if allParams.Has(para) {
			allErrs = append(allErrs, field.Invalid(fieldPath, para, "can not have duplicate parameters"))
		} else {
			allParams.Insert(para)
		}
	}
	return allErrs
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

func rejectPlusResourcesInOSS(upstream v1alpha1.Upstream, idxPath *field.Path, isPlus bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if isPlus {
		return allErrs
	}

	if upstream.HealthCheck != nil {
		allErrs = append(allErrs, field.Forbidden(idxPath.Child("healthCheck"), "active health checks are only supported in NGINX Plus"))
	}

	if upstream.SlowStart != "" {
		allErrs = append(allErrs, field.Forbidden(idxPath.Child("slow-start"), "slow start is only supported in NGINX Plus"))
	}

	if upstream.SessionCookie != nil {
		allErrs = append(allErrs, field.Forbidden(idxPath.Child("sessionCookie"), "sticky cookies are only supported in NGINX Plus"))
	}

	return allErrs
}

func validateQueue(queue *v1alpha1.UpstreamQueue, fieldPath *field.Path, isPlus bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if queue == nil {
		return allErrs
	}

	if !isPlus {
		allErrs = append(allErrs, field.Forbidden(fieldPath, "queue is only supported in NGINX Plus"))
		return allErrs
	}

	allErrs = append(allErrs, validateTime(queue.Timeout, fieldPath.Child("timeout"))...)
	if queue.Size <= 0 {
		allErrs = append(allErrs, field.Required(fieldPath.Child("size"), "must be positive"))
	}

	return allErrs
}

// isValidLabelName checks if a label name is valid.
// It performs the same validation as ValidateLabelName from k8s.io/apimachinery/pkg/apis/meta/v1/validation/validation.go.
func isValidLabelName(labelName string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, msg := range validation.IsQualifiedName(labelName) {
		allErrs = append(allErrs, field.Invalid(fieldPath, labelName, msg))
	}

	return allErrs
}

// validateLabels validates that a set of labels are correctly defined.
// It performs the same validation as ValidateLabels from k8s.io/apimachinery/pkg/apis/meta/v1/validation/validation.go.
func validateLabels(labels map[string]string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for labelName, labelValue := range labels {
		allErrs = append(allErrs, isValidLabelName(labelName, fieldPath)...)
		for _, msg := range validation.IsValidLabelValue(labelValue) {
			allErrs = append(allErrs, field.Invalid(fieldPath, labelValue, msg))
		}
	}

	return allErrs
}

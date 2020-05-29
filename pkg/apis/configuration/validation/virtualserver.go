package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	escapedStringsFmt    = `([^"\\]|\\.)*`
	escapedStringsErrMsg = `must have all '"' (double quotes) escaped and must not end with an unescaped '\' (backslash)`
)

var escapedStringsFmtRegexp = regexp.MustCompile("^" + escapedStringsFmt + "$")

// ValidateVirtualServer validates a VirtualServer.
func ValidateVirtualServer(virtualServer *v1.VirtualServer, isPlus bool) error {
	allErrs := validateVirtualServerSpec(&virtualServer.Spec, field.NewPath("spec"), isPlus, virtualServer.Namespace)
	return allErrs.ToAggregate()
}

// validateVirtualServerSpec validates a VirtualServerSpec.
func validateVirtualServerSpec(spec *v1.VirtualServerSpec, fieldPath *field.Path, isPlus bool, namespace string) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateHost(spec.Host, fieldPath.Child("host"))...)
	allErrs = append(allErrs, validateTLS(spec.TLS, fieldPath.Child("tls"))...)
	allErrs = append(allErrs, validatePolicies(spec.Policies, fieldPath.Child("policies"), namespace)...)

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

func validatePolicies(policies []v1.PolicyReference, fieldPath *field.Path, namespace string) field.ErrorList {
	allErrs := field.ErrorList{}
	policyKeys := sets.String{}

	for i, p := range policies {
		idxPath := fieldPath.Index(i)

		polNamespace := p.Namespace
		if polNamespace == "" {
			polNamespace = namespace
		}

		key := fmt.Sprintf("%s/%s", polNamespace, p.Name)

		if policyKeys.Has(key) {
			allErrs = append(allErrs, field.Duplicate(idxPath, key))
		} else {
			policyKeys.Insert(key)
		}

		if p.Name == "" {
			allErrs = append(allErrs, field.Required(idxPath.Child("name"), ""))
		} else {
			for _, msg := range validation.IsDNS1123Subdomain(p.Name) {
				allErrs = append(allErrs, field.Invalid(idxPath.Child("name"), p.Name, msg))
			}
		}

		if p.Namespace != "" {
			for _, msg := range validation.IsDNS1123Label(p.Namespace) {
				allErrs = append(allErrs, field.Invalid(idxPath.Child("namespace"), p.Namespace, msg))
			}
		}
	}

	return allErrs
}

func validateTLS(tls *v1.TLS, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if tls == nil {
		// valid case - tls is not defined
		return allErrs
	}

	allErrs = append(allErrs, validateSecretName(tls.Secret, fieldPath.Child("secret"))...)

	allErrs = append(allErrs, validateTLSRedirect(tls.Redirect, fieldPath.Child("redirect"))...)

	return allErrs
}

func validateTLSRedirect(redirect *v1.TLSRedirect, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if redirect == nil {
		return allErrs
	}

	if redirect.Code != nil {
		allErrs = append(allErrs, validateRedirectStatusCode(*redirect.Code, fieldPath.Child("code"))...)
	}

	if redirect.BasedOn != "" && redirect.BasedOn != "scheme" && redirect.BasedOn != "x-forwarded-proto" {
		allErrs = append(allErrs, field.Invalid(fieldPath.Child("basedOn"), redirect.BasedOn, "accepted values are 'scheme', 'x-forwarded-proto'"))
	}

	return allErrs
}

var validRedirectStatusCodes = map[int]bool{
	301: true,
	302: true,
	307: true,
	308: true,
}

func validateRedirectStatusCode(code int, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if _, ok := validRedirectStatusCodes[code]; !ok {
		allErrs = append(allErrs, field.Invalid(fieldPath, code, "status code out of accepted range. accepted values are '301', '302', '307', '308'"))
	}

	return allErrs
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

func validateBuffer(buff *v1.UpstreamBuffers, fieldPath *field.Path) field.ErrorList {
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

func validateUpstreamHealthCheck(hc *v1.HealthCheck, fieldPath *field.Path) field.ErrorList {
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

func validateSessionCookie(sc *v1.SessionCookie, fieldPath *field.Path) field.ErrorList {
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

func validateHeader(h v1.Header, fieldPath *field.Path) field.ErrorList {
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
		return allErrs
	}

	for _, msg := range validation.IsDNS1123Subdomain(name) {
		allErrs = append(allErrs, field.Invalid(fieldPath, name, msg))
	}

	return allErrs
}

func validateUpstreams(upstreams []v1.Upstream, fieldPath *field.Path, isPlus bool) (allErrs field.ErrorList, upstreamNames sets.String) {
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
		allErrs = append(allErrs, validateQueue(u.Queue, idxPath.Child("queue"))...)
		allErrs = append(allErrs, validateSessionCookie(u.SessionCookie, idxPath.Child("sessionCookie"))...)

		for _, msg := range validation.IsValidPortNum(int(u.Port)) {
			allErrs = append(allErrs, field.Invalid(idxPath.Child("port"), u.Port, msg))
		}

		allErrs = append(allErrs, rejectPlusResourcesInOSS(u, idxPath, isPlus)...)
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

func validateVirtualServerRoutes(routes []v1.Route, fieldPath *field.Path, upstreamNames sets.String) field.ErrorList {
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

func validateRoute(route v1.Route, fieldPath *field.Path, upstreamNames sets.String, isRouteFieldForbidden bool) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateRoutePath(route.Path, fieldPath.Child("path"))...)

	fieldCount := 0

	if route.Action != nil {
		allErrs = append(allErrs, validateAction(route.Action, fieldPath.Child("action"), upstreamNames, route.Path, false)...)
		fieldCount++
	}

	if len(route.Splits) > 0 {
		allErrs = append(allErrs, validateSplits(route.Splits, fieldPath.Child("splits"), upstreamNames, route.Path)...)
		fieldCount++
	}

	// Matches are optional. that's why we don't do fieldCount++
	if len(route.Matches) > 0 {
		for i, m := range route.Matches {
			allErrs = append(allErrs, validateMatch(m, fieldPath.Child("matches").Index(i), upstreamNames, route.Path)...)
		}
	}

	for i, e := range route.ErrorPages {
		allErrs = append(allErrs, validateErrorPage(e, fieldPath.Child("errorPages").Index(i))...)
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
		msg := "must specify exactly one of `action`, `splits` or `route`"
		if isRouteFieldForbidden || len(route.Matches) > 0 {
			msg = "must specify exactly one of `action` or `splits`"
		}

		allErrs = append(allErrs, field.Invalid(fieldPath, "", msg))
	}

	return allErrs
}

func errorPageHasRequiredFields(errorPage v1.ErrorPage) bool {
	var count int

	if errorPage.Return != nil {
		count++
	}

	if errorPage.Redirect != nil {
		count++
	}

	return count == 1
}

func validateErrorPage(errorPage v1.ErrorPage, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !errorPageHasRequiredFields(errorPage) {
		return append(allErrs, field.Required(fieldPath, "must specify exactly one of `redirect` or `return`"))
	}

	if len(errorPage.Codes) == 0 {
		return append(allErrs, field.Required(fieldPath.Child("codes"), "must include at least 1 status code in `codes`"))
	}

	for i, c := range errorPage.Codes {
		for _, msg := range validation.IsInRange(c, 300, 599) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("codes").Index(i), c, msg))
		}
	}

	if errorPage.Return != nil {
		allErrs = append(allErrs, validateErrorPageReturn(errorPage.Return, fieldPath.Child("return"))...)
	}

	if errorPage.Redirect != nil {
		allErrs = append(allErrs, validateErrorPageRedirect(errorPage.Redirect, fieldPath.Child("redirect"))...)
	}

	return allErrs
}

var errorPageReturnBodyVariable = map[string]bool{"upstream_status": true}

func validateErrorPageReturn(r *v1.ErrorPageReturn, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateActionReturn(&r.ActionReturn, fieldPath, nil, errorPageReturnBodyVariable)...)

	for i, header := range r.Headers {
		allErrs = append(allErrs, validateErrorPageHeader(header, fieldPath.Child("headers").Index(i))...)
	}

	return allErrs
}

var errorPageHeaderValueVariables = map[string]bool{"upstream_status": true}

func validateErrorPageHeader(h v1.Header, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if h.Name == "" {
		allErrs = append(allErrs, field.Required(fieldPath.Child("name"), ""))
	}

	for _, msg := range validation.IsHTTPHeaderName(h.Name) {
		allErrs = append(allErrs, field.Invalid(fieldPath.Child("name"), h.Name, msg))
	}

	if !escapedStringsFmtRegexp.MatchString(h.Value) {
		msg := validation.RegexError(escapedStringsErrMsg, escapedStringsFmt, "value", `\"${status}\"`)
		allErrs = append(allErrs, field.Invalid(fieldPath.Child("value"), h.Value, msg))
	}

	allErrs = append(allErrs, validateStringWithVariables(h.Value, fieldPath.Child("value"), nil, errorPageHeaderValueVariables)...)

	return allErrs
}

var validErrorPageRedirectVariables = map[string]bool{"scheme": true, "http_x_forwarded_proto": true}

func validateErrorPageRedirect(r *v1.ErrorPageRedirect, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateActionRedirect(&r.ActionRedirect, fieldPath, validErrorPageRedirectVariables)...)

	return allErrs
}

func countActions(action *v1.Action) int {
	var count int
	if action.Pass != "" {
		count++
	}

	if action.Redirect != nil {
		count++
	}

	if action.Return != nil {
		count++
	}

	if action.Proxy != nil {
		count++
	}

	return count
}

// returnBodyVariables includes NGINX variables allowed to be used in a return body.
var returnBodyVariables = map[string]bool{
	"request_uri":         true,
	"request_method":      true,
	"request_body":        true,
	"scheme":              true,
	"args":                true,
	"host":                true,
	"request_time":        true,
	"request_length":      true,
	"nginx_version":       true,
	"pid":                 true,
	"connection":          true,
	"remote_addr":         true,
	"remote_port":         true,
	"time_iso8601":        true,
	"time_local":          true,
	"server_addr":         true,
	"server_port":         true,
	"server_name":         true,
	"server_protocol":     true,
	"connections_active":  true,
	"connections_reading": true,
	"connections_writing": true,
	"connections_waiting": true,
}

var returnBodySpecialVariables = []string{"arg_", "http_", "cookie_"}

// validRedirectVariableNames includes NGINX variables allowed to be used in redirects.
var validRedirectVariableNames = map[string]bool{
	"scheme":                 true,
	"http_x_forwarded_proto": true,
	"request_uri":            true,
	"host":                   true,
}

func validateAction(action *v1.Action, fieldPath *field.Path, upstreamNames sets.String, path string, internal bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if countActions(action) != 1 {
		return append(allErrs, field.Required(fieldPath, "action must specify exactly one of `pass`, `redirect`, `return` or `proxy`"))
	}

	if action.Pass != "" {
		allErrs = append(allErrs, validateReferencedUpstream(action.Pass, fieldPath.Child("pass"), upstreamNames)...)
	}

	if action.Redirect != nil {
		allErrs = append(allErrs, validateActionRedirect(action.Redirect, fieldPath.Child("redirect"), validRedirectVariableNames)...)
	}

	if action.Return != nil {
		allErrs = append(allErrs, validateActionReturn(action.Return, fieldPath.Child("return"), returnBodySpecialVariables, returnBodyVariables)...)
	}

	if action.Proxy != nil {
		allErrs = append(allErrs, validateActionProxy(action.Proxy, fieldPath.Child("proxy"), upstreamNames, path, internal)...)
	}

	return allErrs
}

func validateActionRedirect(redirect *v1.ActionRedirect, fieldPath *field.Path, validVars map[string]bool) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateRedirectURL(redirect.URL, fieldPath.Child("url"), validVars)...)

	if redirect.Code != 0 {
		allErrs = append(allErrs, validateRedirectStatusCode(redirect.Code, fieldPath.Child("code"))...)
	}

	return allErrs
}

var nginxVariableRegexp = regexp.MustCompile(`\$\{([^}]*)\}`)

// captureVariables returns a slice of vars enclosed in ${}. For example "${a} ${b}" would return ["a", "b"].
func captureVariables(s string) []string {
	var nVars []string

	res := nginxVariableRegexp.FindAllStringSubmatch(s, -1)
	for _, n := range res {
		nVars = append(nVars, n[1])
	}

	return nVars
}

func validateRedirectURL(redirectURL string, fieldPath *field.Path, validVars map[string]bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if redirectURL == "" {
		return append(allErrs, field.Required(fieldPath, "must specify a url"))
	}

	if !strings.Contains(redirectURL, "://") {
		return append(allErrs, field.Invalid(fieldPath, redirectURL, "must contain the protocol with '://', for example http://, https:// or ${scheme}://"))
	}

	if !escapedStringsFmtRegexp.MatchString(redirectURL) {
		msg := validation.RegexError(escapedStringsErrMsg, escapedStringsFmt, "http://www.nginx.com", "${scheme}://${host}/green/", `\"http://www.nginx.com\"`)
		return append(allErrs, field.Invalid(fieldPath, redirectURL, msg))
	}

	allErrs = append(allErrs, validateStringWithVariables(redirectURL, fieldPath, nil, validVars)...)

	return allErrs
}

func validateStringWithVariables(str string, fieldPath *field.Path, specialVars []string, validVars map[string]bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if strings.HasSuffix(str, "$") {
		return append(allErrs, field.Invalid(fieldPath, str, "must not end with $"))
	}

	for i, c := range str {
		if c == '$' {
			msg := "variables must be enclosed in curly braces, for example ${host}"

			if str[i+1] != '{' {
				return append(allErrs, field.Invalid(fieldPath, str, msg))
			}

			if !strings.Contains(str[i+1:], "}") {
				return append(allErrs, field.Invalid(fieldPath, str, msg))
			}
		}
	}

	nginxVars := captureVariables(str)
	for _, nVar := range nginxVars {
		special := false
		for _, specialVar := range specialVars {
			if strings.HasPrefix(nVar, specialVar) {
				special = true
				break
			}
		}

		if special {
			allErrs = append(allErrs, validateSpecialVariable(nVar, fieldPath)...)
		} else {
			allErrs = append(allErrs, validateVariable(nVar, validVars, fieldPath)...)
		}
	}

	return allErrs
}

func validateVariable(nVar string, validVars map[string]bool, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !validVars[nVar] {
		msg := fmt.Sprintf("'%v' contains an invalid NGINX variable. Accepted variables are: %v", nVar, mapToPrettyString(validVars))
		allErrs = append(allErrs, field.Invalid(fieldPath, nVar, msg))
	}
	return allErrs
}

func isValidSpecialVariableHeader(header string) []string {
	// underscores in $http_ variable represent '-'.
	errMsgs := validation.IsHTTPHeaderName(strings.Replace(header, "_", "-", -1))
	if len(errMsgs) >= 1 || strings.Contains(header, "-") {
		return []string{"a valid HTTP header must consist of alphanumeric characters or '_'"}
	}
	return nil
}

func validateSpecialVariable(nVar string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	value := strings.SplitN(nVar, "_", 2)

	switch value[0] {
	case "arg":
		for _, msg := range isArgumentName(value[1]) {
			allErrs = append(allErrs, field.Invalid(fieldPath, nVar, msg))
		}
	case "http":
		for _, msg := range isValidSpecialVariableHeader(value[1]) {
			allErrs = append(allErrs, field.Invalid(fieldPath, nVar, msg))
		}
	case "cookie":
		for _, msg := range isCookieName(value[1]) {
			allErrs = append(allErrs, field.Invalid(fieldPath, nVar, msg))
		}
	}

	return allErrs
}

func validateActionReturnCode(code int, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if (code >= 200 && code <= 299) || (code >= 400 && code <= 599) {
		return allErrs
	}

	msg := fmt.Sprintf("must be a valid status code either 2XX, 4XX or 5XX, for example, 200 or 402.")
	return append(allErrs, field.Invalid(fieldPath, code, msg))
}

func validateActionReturn(r *v1.ActionReturn, fieldPath *field.Path, specialValidVars []string, validVars map[string]bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if r.Body == "" {
		return append(allErrs, field.Required(fieldPath.Child("body"), ""))
	}

	allErrs = append(allErrs, validateActionReturnBody(r.Body, fieldPath.Child("body"), specialValidVars, validVars)...)

	if r.Type != "" {
		allErrs = append(allErrs, validateActionReturnType(r.Type, fieldPath.Child("type"))...)
	}

	if r.Code != 0 {
		allErrs = append(allErrs, validateActionReturnCode(r.Code, fieldPath.Child("code"))...)
	}

	return allErrs
}

func validateActionReturnBody(body string, fieldPath *field.Path, specialValidVars []string, validVars map[string]bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if !escapedStringsFmtRegexp.MatchString(body) {
		msg := validation.RegexError(escapedStringsErrMsg, escapedStringsFmt, `Hello World! \n`, `\"${request_uri}\" is unavailable. \n`)
		allErrs = append(allErrs, field.Invalid(fieldPath, body, msg))
	}

	allErrs = append(allErrs, validateStringWithVariables(body, fieldPath, specialValidVars, validVars)...)

	return allErrs
}

var actionReturnTypeFmt = `([^;\{\}"\\]|\\.)*`
var actionReturnTypeErr = `must have all '"' (double quotes), '{', '}' or ';' escaped and must not end with an unescaped '\' (backslash)`

var actionReturnTypeRegexp = regexp.MustCompile("^" + actionReturnTypeFmt + "$")

func validateActionReturnType(returnType string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !actionReturnTypeRegexp.MatchString(returnType) {
		msg := validation.RegexError(actionReturnTypeErr, actionReturnTypeFmt, "type/subtype", "application/json")
		allErrs = append(allErrs, field.Invalid(fieldPath, returnType, msg))
	}

	return allErrs
}

func mapToPrettyString(m map[string]bool) string {
	var out []string

	for k := range m {
		out = append(out, k)
	}

	return strings.Join(out, ", ")
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

func validateActionProxy(p *v1.ActionProxy, fieldPath *field.Path, upstreamNames sets.String, path string, internal bool) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateReferencedUpstream(p.Upstream, fieldPath.Child("upstream"), upstreamNames)...)
	allErrs = append(allErrs, validateActionProxyRequestHeaders(p.RequestHeaders, fieldPath.Child("requestHeaders"))...)
	allErrs = append(allErrs, validateActionProxyResponseHeaders(p.ResponseHeaders, fieldPath.Child("responseHeaders"))...)

	if strings.HasPrefix(path, "~") || internal {
		allErrs = append(allErrs, validateActionProxyRewritePathForRegexp(p.RewritePath, fieldPath.Child("rewritePath"))...)
	} else {
		allErrs = append(allErrs, validateActionProxyRewritePath(p.RewritePath, fieldPath.Child("rewritePath"))...)
	}

	return allErrs
}

func validateStringNoVariables(s string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, char := range s {
		charLen := len(string(char))
		if string(char) == "$" && i+charLen < len(s) {
			if _, err := strconv.Atoi(string(s[i+charLen])); err != nil {
				return append(allErrs, field.Invalid(fieldPath, s, "`$` character can be only followed by a number"))
			}
		}
	}

	return allErrs
}

func validateActionProxyRewritePath(rewritePath string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if rewritePath == "" {
		return allErrs
	}

	allErrs = append(allErrs, validateStringNoVariables(rewritePath, fieldPath)...)

	return append(allErrs, validatePath(rewritePath, fieldPath)...)
}

func validateActionProxyRewritePathForRegexp(rewritePath string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if rewritePath == "" {
		return allErrs
	}

	allErrs = append(allErrs, validateStringNoVariables(rewritePath, fieldPath)...)

	if !escapedStringsFmtRegexp.MatchString(rewritePath) {
		msg := validation.RegexError(escapedStringsErrMsg, escapedStringsFmt, "/rewrite$1", "/images")
		allErrs = append(allErrs, field.Invalid(fieldPath, rewritePath, msg))
	}

	return allErrs
}

func validateActionProxyRequestHeaders(requestHeaders *v1.ProxyRequestHeaders, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if requestHeaders == nil {
		return allErrs
	}

	for i, header := range requestHeaders.Set {
		allErrs = append(allErrs, validateHeader(header, fieldPath.Index(i))...)
	}

	return allErrs
}

func validateActionProxyResponseHeaders(responseHeaders *v1.ProxyResponseHeaders, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if responseHeaders == nil {
		return allErrs
	}

	for i, header := range responseHeaders.Hide {
		for _, msg := range validation.IsHTTPHeaderName(header) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("hide").Index(i), header, msg))
		}
	}

	for i, header := range responseHeaders.Pass {
		for _, msg := range validation.IsHTTPHeaderName(header) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("pass").Index(i), header, msg))
		}
	}

	for i, header := range responseHeaders.Add {
		allErrs = append(allErrs, validateHeader(header.Header, fieldPath.Child("add").Index(i))...)
	}

	allErrs = append(allErrs, validateIgnoreHeaders(responseHeaders.Ignore, fieldPath.Child("ignore"))...)

	return allErrs
}

var validIgnoreHeaders = map[string]bool{
	"X-Accel-Redirect":   true,
	"X-Accel-Expires":    true,
	"X-Accel-Limit-Rate": true,
	"X-Accel-Buffering":  true,
	"X-Accel-Charset":    true,
	"Expires":            true,
	"Cache-Control":      true,
	"Set-Cookie":         true,
	"Vary":               true,
}

func validateIgnoreHeaders(ignoreHeaders []string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(ignoreHeaders) == 0 {
		return allErrs
	}

	for i, h := range ignoreHeaders {
		if !validIgnoreHeaders[h] {
			msg := fmt.Sprintf("not a valid ignore header name. Accepted headers are : %v", mapToPrettyString(validIgnoreHeaders))
			allErrs = append(allErrs, field.Invalid(fieldPath.Index(i), h, msg))
		}
	}

	return allErrs
}

func validateSplits(splits []v1.Split, fieldPath *field.Path, upstreamNames sets.String, path string) field.ErrorList {
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

		if s.Action == nil {
			allErrs = append(allErrs, field.Required(idxPath.Child("action"), ""))
		} else {
			allErrs = append(allErrs, validateAction(s.Action, idxPath.Child("action"), upstreamNames, path, true)...)
		}

		totalWeight += s.Weight
	}

	if totalWeight != 100 {
		allErrs = append(allErrs, field.Invalid(fieldPath, "", "the sum of the weights of all splits must be equal to 100"))
	}

	return allErrs
}

// We support prefix-based NGINX locations, positive case-sensitive/insensitive regular expressions matches and exact matches.
// More info http://nginx.org/en/docs/http/ngx_http_core_module.html#location
func validateRoutePath(path string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if path == "" {
		return append(allErrs, field.Required(fieldPath, ""))
	}

	if strings.HasPrefix(path, "~") {
		allErrs = append(allErrs, validateRegexPath(path, fieldPath)...)
	} else if strings.HasPrefix(path, "/") {
		allErrs = append(allErrs, validatePath(path, fieldPath)...)
	} else if strings.HasPrefix(path, "=") {
		allErrs = append(allErrs, validatePath(strings.TrimPrefix(path, "="), fieldPath)...)
	} else {
		allErrs = append(allErrs, field.Invalid(fieldPath, path, "must start with /, ~ or ="))
	}

	return allErrs
}

func validateRegexPath(path string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if _, err := regexp.Compile(path); err != nil {
		return append(allErrs, field.Invalid(fieldPath, path, fmt.Sprintf("must be a valid regular expression: %v", err)))
	}

	if !escapedStringsFmtRegexp.MatchString(path) {
		msg := validation.RegexError(escapedStringsErrMsg, escapedStringsFmt, "*.jpg", "^/images/image_*.png$")
		return append(allErrs, field.Invalid(fieldPath, path, msg))
	}

	return allErrs
}

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

func validateMatch(match v1.Match, fieldPath *field.Path, upstreamNames sets.String, path string) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(match.Conditions) == 0 {
		allErrs = append(allErrs, field.Required(fieldPath.Child("conditions"), "must specify at least one condition"))
	} else {
		for i, c := range match.Conditions {
			allErrs = append(allErrs, validateCondition(c, fieldPath.Child("conditions").Index(i))...)
		}
	}

	fieldCount := 0

	if match.Action != nil {
		allErrs = append(allErrs, validateAction(match.Action, fieldPath.Child("action"), upstreamNames, path, true)...)
		fieldCount++
	}

	if len(match.Splits) > 0 {
		allErrs = append(allErrs, validateSplits(match.Splits, fieldPath.Child("splits"), upstreamNames, path)...)
		fieldCount++
	}

	if fieldCount != 1 {
		allErrs = append(allErrs, field.Invalid(fieldPath, "", "must specify exactly one of `action` or `splits`"))
	}

	return allErrs
}

func validateCondition(condition v1.Condition, fieldPath *field.Path) field.ErrorList {
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

	for _, msg := range isValidMatchValue(condition.Value) {
		allErrs = append(allErrs, field.Invalid(fieldPath.Child("value"), condition.Value, msg))
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

func isValidMatchValue(value string) []string {
	if !escapedStringsFmtRegexp.MatchString(value) {
		return []string{validation.RegexError(escapedStringsErrMsg, escapedStringsFmt, "value-123")}
	}
	return nil
}

// ValidateVirtualServerRoute validates a VirtualServerRoute.
func ValidateVirtualServerRoute(virtualServerRoute *v1.VirtualServerRoute, isPlus bool) error {
	allErrs := validateVirtualServerRouteSpec(&virtualServerRoute.Spec, field.NewPath("spec"), "", "/", isPlus)
	return allErrs.ToAggregate()
}

// ValidateVirtualServerRouteForVirtualServer validates a VirtualServerRoute for a VirtualServer represented by its host and path prefix.
func ValidateVirtualServerRouteForVirtualServer(virtualServerRoute *v1.VirtualServerRoute, virtualServerHost string, vsPath string, isPlus bool) error {
	allErrs := validateVirtualServerRouteSpec(&virtualServerRoute.Spec, field.NewPath("spec"), virtualServerHost, vsPath, isPlus)
	return allErrs.ToAggregate()
}

func validateVirtualServerRouteSpec(spec *v1.VirtualServerRouteSpec, fieldPath *field.Path, virtualServerHost string, vsPath string, isPlus bool) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateVirtualServerRouteHost(spec.Host, virtualServerHost, fieldPath.Child("host"))...)

	upstreamErrs, upstreamNames := validateUpstreams(spec.Upstreams, fieldPath.Child("upstreams"), isPlus)
	allErrs = append(allErrs, upstreamErrs...)

	allErrs = append(allErrs, validateVirtualServerRouteSubroutes(spec.Subroutes, fieldPath.Child("subroutes"), upstreamNames, vsPath)...)

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

func isRegexOrExactMatch(path string) bool {
	return strings.HasPrefix(path, "~") || strings.HasPrefix(path, "=")
}

func validateVirtualServerRouteSubroutes(routes []v1.Route, fieldPath *field.Path, upstreamNames sets.String, vsPath string) field.ErrorList {
	allErrs := field.ErrorList{}

	allPaths := sets.String{}

	if isRegexOrExactMatch(vsPath) {
		if len(routes) != 1 {
			return append(allErrs, field.Invalid(fieldPath, "subroutes", "must have only one subroute if regex match or exact match are being used"))
		}

		idxPath := fieldPath.Index(0)
		if routes[0].Path != vsPath {
			return append(allErrs, field.Invalid(idxPath.Child("path"), routes[0].Path, "must have the same path as the referenced VirtualServer route path"))
		}

		return validateRoute(routes[0], idxPath, upstreamNames, true)
	}

	for i, r := range routes {
		idxPath := fieldPath.Index(i)

		isRouteFieldForbidden := true
		routeErrs := validateRoute(r, idxPath, upstreamNames, isRouteFieldForbidden)

		if vsPath != "" && !strings.HasPrefix(r.Path, vsPath) && !isRegexOrExactMatch(r.Path) {
			msg := fmt.Sprintf("must start with '%s'", vsPath)
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

func rejectPlusResourcesInOSS(upstream v1.Upstream, idxPath *field.Path, isPlus bool) field.ErrorList {
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

	if upstream.Queue != nil {
		allErrs = append(allErrs, field.Forbidden(idxPath.Child("queue"), "queue is only supported in NGINX Plus"))
	}

	return allErrs
}

func validateQueue(queue *v1.UpstreamQueue, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if queue == nil {
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

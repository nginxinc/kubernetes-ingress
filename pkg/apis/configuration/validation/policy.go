package validation

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidatePolicy validates a Policy.
func ValidatePolicy(policy *v1.Policy, isPlus, enableOIDC, enableAppProtect bool) error {
	allErrs := validatePolicySpec(&policy.Spec, field.NewPath("spec"), isPlus, enableOIDC, enableAppProtect)
	return allErrs.ToAggregate()
}

func validatePolicySpec(spec *v1.PolicySpec, fieldPath *field.Path, isPlus, enableOIDC, enableAppProtect bool) field.ErrorList {
	allErrs := field.ErrorList{}

	fieldCount := 0

	if spec.AccessControl != nil {
		allErrs = append(allErrs, validateAccessControl(spec.AccessControl, fieldPath.Child("accessControl"))...)
		fieldCount++
	}

	if spec.RateLimit != nil {
		allErrs = append(allErrs, validateRateLimit(spec.RateLimit, fieldPath.Child("rateLimit"), isPlus)...)
		fieldCount++
	}

	if spec.JWTAuth != nil {
		if !isPlus {
			return append(allErrs, field.Forbidden(fieldPath.Child("jwt"), "jwt secrets are only supported in NGINX Plus"))
		}

		allErrs = append(allErrs, validateJWT(spec.JWTAuth, fieldPath.Child("jwt"))...)
		fieldCount++
	}

	if spec.IngressMTLS != nil {
		allErrs = append(allErrs, validateIngressMTLS(spec.IngressMTLS, fieldPath.Child("ingressMTLS"))...)
		fieldCount++
	}

	if spec.EgressMTLS != nil {
		allErrs = append(allErrs, validateEgressMTLS(spec.EgressMTLS, fieldPath.Child("egressMTLS"))...)
		fieldCount++
	}

	if spec.OIDC != nil {
		if !enableOIDC {
			allErrs = append(allErrs, field.Forbidden(fieldPath.Child("oidc"),
				"OIDC must be enabled via cli argument -enable-oidc to use OIDC policy"))
		}
		if !isPlus {
			return append(allErrs, field.Forbidden(fieldPath.Child("oidc"), "OIDC is only supported in NGINX Plus"))
		}

		allErrs = append(allErrs, validateOIDC(spec.OIDC, fieldPath.Child("oidc"))...)
		fieldCount++
	}

	if spec.WAF != nil {
		if !isPlus {
			allErrs = append(allErrs, field.Forbidden(fieldPath.Child("waf"), "WAF is only supported in NGINX Plus"))
		}
		if !enableAppProtect {
			allErrs = append(allErrs, field.Forbidden(fieldPath.Child("waf"),
				"App Protect must be enabled via cli argument -enable-appprotect to use WAF policy"))
		}

		allErrs = append(allErrs, validateWAF(spec.WAF, fieldPath.Child("waf"))...)
		fieldCount++
	}

	if fieldCount != 1 {
		msg := "must specify exactly one of: `accessControl`, `rateLimit`, `ingressMTLS`, `egressMTLS`"
		if isPlus {
			msg = fmt.Sprint(msg, ", `jwt`, `oidc`, `waf`")
		}
		allErrs = append(allErrs, field.Invalid(fieldPath, "", msg))
	}

	return allErrs
}

func validateAccessControl(accessControl *v1.AccessControl, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	fieldCount := 0

	if accessControl.Allow != nil {
		for i, ipOrCIDR := range accessControl.Allow {
			allErrs = append(allErrs, validateIPorCIDR(ipOrCIDR, fieldPath.Child("allow").Index(i))...)
		}
		fieldCount++
	}

	if accessControl.Deny != nil {
		for i, ipOrCIDR := range accessControl.Deny {
			allErrs = append(allErrs, validateIPorCIDR(ipOrCIDR, fieldPath.Child("deny").Index(i))...)
		}
		fieldCount++
	}

	if fieldCount != 1 {
		allErrs = append(allErrs, field.Invalid(fieldPath, "", "must specify exactly one of: `allow` or `deny`"))
	}

	return allErrs
}

func validateRateLimit(rateLimit *v1.RateLimit, fieldPath *field.Path, isPlus bool) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateRateLimitZoneSize(rateLimit.ZoneSize, fieldPath.Child("zoneSize"))...)
	allErrs = append(allErrs, validateRate(rateLimit.Rate, fieldPath.Child("rate"))...)
	allErrs = append(allErrs, validateRateLimitKey(rateLimit.Key, fieldPath.Child("key"), isPlus)...)

	if rateLimit.Delay != nil {
		allErrs = append(allErrs, validatePositiveInt(*rateLimit.Delay, fieldPath.Child("delay"))...)
	}

	if rateLimit.Burst != nil {
		allErrs = append(allErrs, validatePositiveInt(*rateLimit.Burst, fieldPath.Child("burst"))...)
	}

	if rateLimit.LogLevel != "" {
		allErrs = append(allErrs, validateRateLimitLogLevel(rateLimit.LogLevel, fieldPath.Child("logLevel"))...)
	}

	if rateLimit.RejectCode != nil {
		if *rateLimit.RejectCode < 400 || *rateLimit.RejectCode > 599 {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("rejectCode"), rateLimit.RejectCode,
				"must be within the range [400-599]"))
		}
	}

	return allErrs
}

func validateJWT(jwt *v1.JWTAuth, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateJWTRealm(jwt.Realm, fieldPath.Child("realm"))...)

	if jwt.Secret == "" {
		return append(allErrs, field.Required(fieldPath.Child("secret"), ""))
	}
	allErrs = append(allErrs, validateSecretName(jwt.Secret, fieldPath.Child("secret"))...)

	allErrs = append(allErrs, validateJWTToken(jwt.Token, fieldPath.Child("token"))...)

	return allErrs
}

func validateIngressMTLS(ingressMTLS *v1.IngressMTLS, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if ingressMTLS.ClientCertSecret == "" {
		return append(allErrs, field.Required(fieldPath.Child("clientCertSecret"), ""))
	}
	allErrs = append(allErrs, validateSecretName(ingressMTLS.ClientCertSecret, fieldPath.Child("clientCertSecret"))...)

	allErrs = append(allErrs, validateIngressMTLSVerifyClient(ingressMTLS.VerifyClient, fieldPath.Child("verifyClient"))...)

	if ingressMTLS.VerifyDepth != nil {
		allErrs = append(allErrs, validatePositiveIntOrZero(*ingressMTLS.VerifyDepth, fieldPath.Child("verifyDepth"))...)
	}
	return allErrs
}

func validateEgressMTLS(egressMTLS *v1.EgressMTLS, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateSecretName(egressMTLS.TLSSecret, fieldPath.Child("tlsSecret"))...)

	if egressMTLS.VerifyServer && egressMTLS.TrustedCertSecret == "" {
		return append(allErrs, field.Required(fieldPath.Child("trustedCertSecret"), "must be set when verifyServer is 'true'"))
	}
	allErrs = append(allErrs, validateSecretName(egressMTLS.TrustedCertSecret, fieldPath.Child("trustedCertSecret"))...)

	if egressMTLS.VerifyDepth != nil {
		allErrs = append(allErrs, validatePositiveIntOrZero(*egressMTLS.VerifyDepth, fieldPath.Child("verifyDepth"))...)
	}

	allErrs = append(allErrs, validateSSLName(egressMTLS.SSLName, fieldPath.Child("sslName"))...)

	return allErrs
}

func validateOIDC(oidc *v1.OIDC, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if oidc.AuthEndpoint == "" {
		return append(allErrs, field.Required(fieldPath.Child("authEndpoint"), ""))
	}
	if oidc.TokenEndpoint == "" {
		return append(allErrs, field.Required(fieldPath.Child("tokenEndpoint"), ""))
	}
	if oidc.JWKSURI == "" {
		return append(allErrs, field.Required(fieldPath.Child("jwksURI"), ""))
	}
	if oidc.ClientID == "" {
		return append(allErrs, field.Required(fieldPath.Child("clientID"), ""))
	}
	if oidc.ClientSecret == "" {
		return append(allErrs, field.Required(fieldPath.Child("clientSecret"), ""))
	}

	if oidc.Scope != "" {
		allErrs = append(allErrs, validateOIDCScope(oidc.Scope, fieldPath.Child("scope"))...)
	}

	if oidc.RedirectURI != "" {
		allErrs = append(allErrs, validatePath(oidc.RedirectURI, fieldPath.Child("redirectURI"))...)
	}

	allErrs = append(allErrs, validateURL(oidc.AuthEndpoint, fieldPath.Child("authEndpoint"))...)
	allErrs = append(allErrs, validateURL(oidc.TokenEndpoint, fieldPath.Child("tokenEndpoint"))...)
	allErrs = append(allErrs, validateURL(oidc.JWKSURI, fieldPath.Child("jwksURI"))...)
	allErrs = append(allErrs, validateSecretName(oidc.ClientSecret, fieldPath.Child("clientSecret"))...)
	allErrs = append(allErrs, validateClientID(oidc.ClientID, fieldPath.Child("clientID"))...)

	return allErrs
}

func validateWAF(waf *v1.WAF, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if waf.ApPolicy != "" {
		for _, msg := range validation.IsQualifiedName(waf.ApPolicy) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("apPolicy"), waf.ApPolicy, msg))
		}
	}

	if waf.SecurityLog != nil {
		allErrs = append(allErrs, validateLogConf(waf.SecurityLog.ApLogConf, waf.SecurityLog.LogDest, fieldPath.Child("securityLog"))...)
	}

	return allErrs
}

func validateLogConf(logConf, logDest string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if logConf != "" {
		for _, msg := range validation.IsQualifiedName(logConf) {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child("apLogConf"), logConf, msg))
		}
	}

	err := ValidateAppProtectLogDestination(logDest)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath.Child("logDest"), logDest, err.Error()))
	}
	return allErrs
}

func validateClientID(client string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// isValidHeaderValue checks for $ and " in the string
	if isValidHeaderValue(client) != nil {
		allErrs = append(allErrs, field.Invalid(
			fieldPath,
			client,
			`invalid string. String must contain valid ASCII characters, must have all '"' escaped and must not contain any '$' or end with an unescaped '\'
		`))
	}

	return allErrs
}

var validScopes = map[string]bool{
	"openid":  true,
	"profile": true,
	"email":   true,
	"address": true,
	"phone":   true,
}

// https://openid.net/specs/openid-connect-core-1_0.html#ScopeClaims
func validateOIDCScope(scope string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !strings.Contains(scope, "openid") {
		return append(allErrs, field.Required(fieldPath, "openid scope"))
	}

	s := strings.Split(scope, "+")
	for _, v := range s {
		if !validScopes[v] {
			msg := fmt.Sprintf("invalid Scope. Accepted scopes are: %v", mapToPrettyString(validScopes))
			allErrs = append(allErrs, field.Invalid(fieldPath, v, msg))
		}
	}

	return allErrs
}

func validateURL(name string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	u, err := url.Parse(name)
	if err != nil {
		return append(allErrs, field.Invalid(fieldPath, name, err.Error()))
	}
	var msg string
	if u.Scheme == "" {
		msg = "scheme required, please use the prefix http(s)://"
		return append(allErrs, field.Invalid(fieldPath, name, msg))
	}
	if u.Host == "" {
		msg = "hostname required"
		return append(allErrs, field.Invalid(fieldPath, name, msg))
	}
	if u.Path == "" {
		msg = "path required"
		return append(allErrs, field.Invalid(fieldPath, name, msg))
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}

	allErrs = append(allErrs, validateSSLName(host, fieldPath)...)
	if port != "" {
		allErrs = append(allErrs, validatePortNumber(port, fieldPath)...)
	}

	return allErrs
}

func validatePortNumber(port string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	portInt, _ := strconv.Atoi(port)
	msg := validation.IsValidPortNum(portInt)
	if msg != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, port, msg[0]))
	}
	return allErrs
}

func validateSSLName(name string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if name != "" {
		for _, msg := range validation.IsDNS1123Subdomain(name) {
			allErrs = append(allErrs, field.Invalid(fieldPath, name, msg))
		}
	}
	return allErrs
}

var validateVerifyClientKeyParameters = map[string]bool{
	"on":             true,
	"off":            true,
	"optional":       true,
	"optional_no_ca": true,
}

func validateIngressMTLSVerifyClient(verifyClient string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if verifyClient != "" {
		allErrs = append(allErrs, ValidateParameter(verifyClient, validateVerifyClientKeyParameters, fieldPath)...)
	}
	return allErrs
}

const (
	rateFmt    = `[1-9]\d*r/[sSmM]`
	rateErrMsg = "must consist of numeric characters followed by a valid rate suffix. 'r/s|r/m"
)

var rateRegexp = regexp.MustCompile("^" + rateFmt + "$")

func validateRate(rate string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if rate == "" {
		return append(allErrs, field.Required(fieldPath, ""))
	}

	if !rateRegexp.MatchString(rate) {
		msg := validation.RegexError(rateErrMsg, rateFmt, "16r/s", "32r/m", "64r/s")
		return append(allErrs, field.Invalid(fieldPath, rate, msg))
	}
	return allErrs
}

func validateRateLimitZoneSize(zoneSize string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if zoneSize == "" {
		return append(allErrs, field.Required(fieldPath, ""))
	}

	allErrs = append(allErrs, validateSize(zoneSize, fieldPath)...)

	kbZoneSize := strings.TrimSuffix(strings.ToLower(zoneSize), "k")
	kbZoneSizeNum, err := strconv.Atoi(kbZoneSize)

	mbZoneSize := strings.TrimSuffix(strings.ToLower(zoneSize), "m")
	mbZoneSizeNum, mbErr := strconv.Atoi(mbZoneSize)

	if err == nil && kbZoneSizeNum < 32 || mbErr == nil && mbZoneSizeNum == 0 {
		allErrs = append(allErrs, field.Invalid(fieldPath, zoneSize, "must be greater than 31k"))
	}

	return allErrs
}

var rateLimitKeySpecialVariables = []string{"arg_", "http_", "cookie_"}

// rateLimitKeyVariables includes NGINX variables allowed to be used in a rateLimit policy key.
var rateLimitKeyVariables = map[string]bool{
	"binary_remote_addr": true,
	"request_uri":        true,
	"uri":                true,
	"args":               true,
}

func validateRateLimitKey(key string, fieldPath *field.Path, isPlus bool) field.ErrorList {
	allErrs := field.ErrorList{}

	if key == "" {
		return append(allErrs, field.Required(fieldPath, ""))
	}

	if err := ValidateEscapedString(key, `Hello World! \n`, `\"${request_uri}\" is unavailable. \n`); err != nil {
		allErrs = append(allErrs, field.Invalid(fieldPath, key, err.Error()))
	}

	allErrs = append(allErrs, validateStringWithVariables(key, fieldPath, rateLimitKeySpecialVariables, rateLimitKeyVariables, isPlus)...)

	return allErrs
}

var jwtTokenSpecialVariables = []string{"arg_", "http_", "cookie_"}

func validateJWTToken(token string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if token == "" {
		return allErrs
	}

	nginxVars := strings.Split(token, "$")
	if len(nginxVars) != 2 {
		return append(allErrs, field.Invalid(fieldPath, token, "must have 1 var"))
	}
	nVar := token[1:]

	special := false
	for _, specialVar := range jwtTokenSpecialVariables {
		if strings.HasPrefix(nVar, specialVar) {
			special = true
			break
		}
	}

	if special {
		// validateJWTToken is called only when NGINX Plus is running
		isPlus := true
		allErrs = append(allErrs, validateSpecialVariable(nVar, fieldPath, isPlus)...)
	} else {
		return append(allErrs, field.Invalid(fieldPath, token, "must only have special vars"))
	}

	return allErrs
}

var validLogLevels = map[string]bool{
	"info":   true,
	"notice": true,
	"warn":   true,
	"error":  true,
}

func validateRateLimitLogLevel(logLevel string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if !validLogLevels[logLevel] {
		allErrs = append(allErrs, field.Invalid(fieldPath, logLevel, fmt.Sprintf("Accepted values: %s",
			mapToPrettyString(validLogLevels))))
	}

	return allErrs
}

const (
	jwtRealmFmt              = `([^"$\\]|\\[^$])*`
	jwtRealmFmtErrMsg string = `a valid realm must have all '"' escaped and must not contain any '$' or end with an unescaped '\'`
)

var jwtRealmFmtRegexp = regexp.MustCompile("^" + jwtRealmFmt + "$")

func validateJWTRealm(realm string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if realm == "" {
		return append(allErrs, field.Required(fieldPath, ""))
	}

	if !jwtRealmFmtRegexp.MatchString(realm) {
		msg := validation.RegexError(jwtRealmFmtErrMsg, jwtRealmFmt, "MyAPI", "My Product API")
		allErrs = append(allErrs, field.Invalid(fieldPath, realm, msg))
	}

	return allErrs
}

func validateIPorCIDR(ipOrCIDR string, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	_, _, err := net.ParseCIDR(ipOrCIDR)
	if err == nil {
		// valid CIDR
		return allErrs
	}

	ip := net.ParseIP(ipOrCIDR)
	if ip != nil {
		// valid IP
		return allErrs
	}

	return append(allErrs, field.Invalid(fieldPath, ipOrCIDR, "must be a CIDR or IP"))
}

func validatePositiveInt(n int, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if n <= 0 {
		return append(allErrs, field.Invalid(fieldPath, n, "must be positive"))
	}

	return allErrs
}

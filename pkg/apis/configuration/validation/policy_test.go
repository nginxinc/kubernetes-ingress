package validation

import (
	"testing"

	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidatePolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		policy           *v1.Policy
		isPlus           bool
		enableOIDC       bool
		enableAppProtect bool
		msg              string
	}{
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					AccessControl: &v1.AccessControl{
						Allow: []string{"127.0.0.1"},
					},
				},
			},
			isPlus:           false,
			enableOIDC:       false,
			enableAppProtect: false,
		},
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					JWTAuth: &v1.JWTAuth{
						Realm:  "My Product API",
						Secret: "my-jwk",
					},
				},
			},
			isPlus:           true,
			enableOIDC:       false,
			enableAppProtect: false,
			msg:              "use jwt(plus only) policy",
		},
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					OIDC: &v1.OIDC{
						AuthEndpoint:  "https://foo.bar/auth",
						TokenEndpoint: "https://foo.bar/token",
						JWKSURI:       "https://foo.bar/certs",
						ClientID:      "random-string",
						ClientSecret:  "random-secret",
						Scope:         "openid",
					},
				},
			},
			isPlus:     true,
			enableOIDC: true,
			msg:        "use OIDC (plus only)",
		},
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					WAF: &v1.WAF{
						Enable: true,
					},
				},
			},
			isPlus:           true,
			enableOIDC:       false,
			enableAppProtect: true,
			msg:              "use WAF(plus only) policy",
		},
	}
	for _, test := range tests {
		err := ValidatePolicy(test.policy, test.isPlus, test.enableOIDC, test.enableAppProtect)
		if err != nil {
			t.Errorf("ValidatePolicy() returned error %v for valid input for the case of %v", err, test.msg)
		}
	}
}

func TestValidatePolicyFails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		policy           *v1.Policy
		isPlus           bool
		enableOIDC       bool
		enableAppProtect bool
		msg              string
	}{
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{},
			},
			isPlus:           false,
			enableOIDC:       false,
			enableAppProtect: false,
			msg:              "empty policy spec",
		},
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					AccessControl: &v1.AccessControl{
						Allow: []string{"127.0.0.1"},
					},
					RateLimit: &v1.RateLimit{
						Key:      "${uri}",
						ZoneSize: "10M",
						Rate:     "10r/s",
					},
				},
			},
			isPlus:           true,
			enableOIDC:       false,
			enableAppProtect: false,
			msg:              "multiple policies in spec",
		},
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					JWTAuth: &v1.JWTAuth{
						Realm:  "My Product API",
						Secret: "my-jwk",
					},
				},
			},
			isPlus:           false,
			enableOIDC:       false,
			enableAppProtect: false,
			msg:              "jwt(plus only) policy on OSS",
		},
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					WAF: &v1.WAF{
						Enable: true,
					},
				},
			},
			isPlus:           false,
			enableOIDC:       false,
			enableAppProtect: false,
			msg:              "WAF(plus only) policy on OSS",
		},
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					OIDC: &v1.OIDC{
						AuthEndpoint:  "https://foo.bar/auth",
						TokenEndpoint: "https://foo.bar/token",
						JWKSURI:       "https://foo.bar/certs",
						ClientID:      "random-string",
						ClientSecret:  "random-secret",
						Scope:         "openid",
					},
				},
			},
			isPlus:     true,
			enableOIDC: false,
			msg:        "OIDC policy with enable OIDC flag disabled",
		},
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					OIDC: &v1.OIDC{
						AuthEndpoint:  "https://foo.bar/auth",
						TokenEndpoint: "https://foo.bar/token",
						JWKSURI:       "https://foo.bar/certs",
						ClientID:      "random-string",
						ClientSecret:  "random-secret",
						Scope:         "openid",
					},
				},
			},
			isPlus:     false,
			enableOIDC: true,
			msg:        "OIDC policy in OSS",
		},
		{
			policy: &v1.Policy{
				Spec: v1.PolicySpec{
					WAF: &v1.WAF{
						Enable: true,
					},
				},
			},
			isPlus:           true,
			enableOIDC:       false,
			enableAppProtect: false,
			msg:              "WAF policy with AP disabled",
		},
	}
	for _, test := range tests {
		err := ValidatePolicy(test.policy, test.isPlus, test.enableOIDC, test.enableAppProtect)
		if err == nil {
			t.Errorf("ValidatePolicy() returned no error for invalid input")
		}
	}
}

func TestValidateAccessControl(t *testing.T) {
	t.Parallel()
	validInput := []*v1.AccessControl{
		{
			Allow: []string{},
		},
		{
			Allow: []string{"127.0.0.1"},
		},
		{
			Deny: []string{},
		},
		{
			Deny: []string{"127.0.0.1"},
		},
	}

	for _, input := range validInput {
		allErrs := validateAccessControl(input, field.NewPath("accessControl"))
		if len(allErrs) > 0 {
			t.Errorf("validateAccessControl(%+v) returned errors %v for valid input", input, allErrs)
		}
	}
}

func TestValidateAccessControlFails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		accessControl *v1.AccessControl
		msg           string
	}{
		{
			accessControl: &v1.AccessControl{
				Allow: nil,
				Deny:  nil,
			},
			msg: "neither allow nor deny is defined",
		},
		{
			accessControl: &v1.AccessControl{
				Allow: []string{},
				Deny:  []string{},
			},
			msg: "both allow and deny are defined",
		},
		{
			accessControl: &v1.AccessControl{
				Allow: []string{"invalid"},
			},
			msg: "invalid allow",
		},
		{
			accessControl: &v1.AccessControl{
				Deny: []string{"invalid"},
			},
			msg: "invalid deny",
		},
	}

	for _, test := range tests {
		allErrs := validateAccessControl(test.accessControl, field.NewPath("accessControl"))
		if len(allErrs) == 0 {
			t.Errorf("validateAccessControl() returned no errors for invalid input for the case of %s", test.msg)
		}
	}
}

func TestValidateRateLimit(t *testing.T) {
	t.Parallel()
	dryRun := true
	noDelay := false

	tests := []struct {
		rateLimit *v1.RateLimit
		msg       string
	}{
		{
			rateLimit: &v1.RateLimit{
				Rate:     "10r/s",
				ZoneSize: "10M",
				Key:      "${request_uri}",
			},
			msg: "only required fields are set",
		},
		{
			rateLimit: &v1.RateLimit{
				Rate:       "30r/m",
				Key:        "${request_uri}",
				Delay:      createPointerFromInt(5),
				NoDelay:    &noDelay,
				Burst:      createPointerFromInt(10),
				ZoneSize:   "10M",
				DryRun:     &dryRun,
				LogLevel:   "info",
				RejectCode: createPointerFromInt(505),
			},
			msg: "ratelimit all fields set",
		},
	}

	isPlus := false

	for _, test := range tests {
		allErrs := validateRateLimit(test.rateLimit, field.NewPath("rateLimit"), isPlus)
		if len(allErrs) > 0 {
			t.Errorf("validateRateLimit() returned errors %v for valid input for the case of %v", allErrs, test.msg)
		}
	}
}

func createInvalidRateLimit(f func(r *v1.RateLimit)) *v1.RateLimit {
	validRateLimit := &v1.RateLimit{
		Rate:     "10r/s",
		ZoneSize: "10M",
		Key:      "${request_uri}",
	}
	f(validRateLimit)
	return validRateLimit
}

func TestValidateRateLimitFails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rateLimit *v1.RateLimit
		msg       string
	}{
		{
			rateLimit: createInvalidRateLimit(func(r *v1.RateLimit) {
				r.Rate = "0r/s"
			}),
			msg: "invalid rateLimit rate",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1.RateLimit) {
				r.Key = "${fail}"
			}),
			msg: "invalid rateLimit key variable use",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1.RateLimit) {
				r.Delay = createPointerFromInt(0)
			}),
			msg: "invalid rateLimit delay",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1.RateLimit) {
				r.Burst = createPointerFromInt(0)
			}),
			msg: "invalid rateLimit burst",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1.RateLimit) {
				r.ZoneSize = "31k"
			}),
			msg: "invalid rateLimit zoneSize",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1.RateLimit) {
				r.RejectCode = createPointerFromInt(600)
			}),
			msg: "invalid rateLimit rejectCode",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1.RateLimit) {
				r.LogLevel = "invalid"
			}),
			msg: "invalid rateLimit logLevel",
		},
	}

	isPlus := false

	for _, test := range tests {
		allErrs := validateRateLimit(test.rateLimit, field.NewPath("rateLimit"), isPlus)
		if len(allErrs) == 0 {
			t.Errorf("validateRateLimit() returned no errors for invalid input for the case of %v", test.msg)
		}
	}
}

func TestValidateJWT(t *testing.T) {
	t.Parallel()
	tests := []struct {
		jwt *v1.JWTAuth
		msg string
	}{
		{
			jwt: &v1.JWTAuth{
				Realm:  "My Product API",
				Secret: "my-jwk",
			},
			msg: "basic",
		},
		{
			jwt: &v1.JWTAuth{
				Realm:  "My Product API",
				Secret: "my-jwk",
				Token:  "$cookie_auth_token",
			},
			msg: "jwt with token",
		},
	}
	for _, test := range tests {
		allErrs := validateJWT(test.jwt, field.NewPath("jwt"))
		if len(allErrs) != 0 {
			t.Errorf("validateJWT() returned errors %v for valid input for the case of %v", allErrs, test.msg)
		}
	}
}

func TestValidateJWTFails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		msg string
		jwt *v1.JWTAuth
	}{
		{
			jwt: &v1.JWTAuth{
				Realm: "My Product API",
			},
			msg: "missing secret",
		},
		{
			jwt: &v1.JWTAuth{
				Secret: "my-jwk",
			},
			msg: "missing realm",
		},
		{
			jwt: &v1.JWTAuth{
				Realm:  "My Product API",
				Secret: "my-jwk",
				Token:  "$uri",
			},
			msg: "invalid variable use in token",
		},
		{
			jwt: &v1.JWTAuth{
				Realm:  "My Product API",
				Secret: "my-\"jwk",
			},
			msg: "invalid secret name",
		},
		{
			jwt: &v1.JWTAuth{
				Realm:  "My \"Product API",
				Secret: "my-jwk",
			},
			msg: "invalid realm due to escaped string",
		},
		{
			jwt: &v1.JWTAuth{
				Realm:  "My Product ${api}",
				Secret: "my-jwk",
			},
			msg: "invalid variable use in realm with curly braces",
		},
		{
			jwt: &v1.JWTAuth{
				Realm:  "My Product $api",
				Secret: "my-jwk",
			},
			msg: "invalid variable use in realm without curly braces",
		},
	}
	for _, test := range tests {
		allErrs := validateJWT(test.jwt, field.NewPath("jwt"))
		if len(allErrs) == 0 {
			t.Errorf("validateJWT() returned no errors for invalid input for the case of %v", test.msg)
		}
	}
}

func TestValidateIPorCIDR(t *testing.T) {
	t.Parallel()
	validInput := []string{
		"192.168.1.1",
		"192.168.1.0/24",
		"2001:0db8::1",
		"2001:0db8::/32",
	}

	for _, input := range validInput {
		allErrs := validateIPorCIDR(input, field.NewPath("ipOrCIDR"))
		if len(allErrs) > 0 {
			t.Errorf("validateIPorCIDR(%q) returned errors %v for valid input", input, allErrs)
		}
	}

	invalidInput := []string{
		"localhost",
		"192.168.1.0/",
		"2001:0db8:::1",
		"2001:0db8::/",
	}

	for _, input := range invalidInput {
		allErrs := validateIPorCIDR(input, field.NewPath("ipOrCIDR"))
		if len(allErrs) == 0 {
			t.Errorf("validateIPorCIDR(%q) returned no errors for invalid input", input)
		}
	}
}

func TestValidateRate(t *testing.T) {
	t.Parallel()
	validInput := []string{
		"10r/s",
		"100r/m",
		"1r/s",
	}

	for _, input := range validInput {
		allErrs := validateRate(input, field.NewPath("rate"))
		if len(allErrs) > 0 {
			t.Errorf("validateRate(%q) returned errors %v for valid input", input, allErrs)
		}
	}

	invalidInput := []string{
		"10s",
		"10r/",
		"10r/ms",
		"0r/s",
	}

	for _, input := range invalidInput {
		allErrs := validateRate(input, field.NewPath("rate"))
		if len(allErrs) == 0 {
			t.Errorf("validateRate(%q) returned no errors for invalid input", input)
		}
	}
}

func TestValidatePositiveInt(t *testing.T) {
	t.Parallel()
	validInput := []int{1, 2}

	for _, input := range validInput {
		allErrs := validatePositiveInt(input, field.NewPath("int"))
		if len(allErrs) > 0 {
			t.Errorf("validatePositiveInt(%q) returned errors %v for valid input", input, allErrs)
		}
	}

	invalidInput := []int{-1, 0}

	for _, input := range invalidInput {
		allErrs := validatePositiveInt(input, field.NewPath("int"))
		if len(allErrs) == 0 {
			t.Errorf("validatePositiveInt(%q) returned no errors for invalid input", input)
		}
	}
}

func TestValidateRateLimitZoneSize(t *testing.T) {
	t.Parallel()
	validInput := []string{"32", "32k", "32K", "10m"}

	for _, test := range validInput {
		allErrs := validateRateLimitZoneSize(test, field.NewPath("size"))
		if len(allErrs) != 0 {
			t.Errorf("validateRateLimitZoneSize(%q) returned an error for valid input", test)
		}
	}

	invalidInput := []string{"", "31", "31k", "0", "0M"}

	for _, test := range invalidInput {
		allErrs := validateRateLimitZoneSize(test, field.NewPath("size"))
		if len(allErrs) == 0 {
			t.Errorf("validateRateLimitZoneSize(%q) didn't return error for invalid input", test)
		}
	}
}

func TestValidateRateLimitLogLevel(t *testing.T) {
	t.Parallel()
	validInput := []string{"error", "info", "warn", "notice"}

	for _, test := range validInput {
		allErrs := validateRateLimitLogLevel(test, field.NewPath("logLevel"))
		if len(allErrs) != 0 {
			t.Errorf("validateRateLimitLogLevel(%q) returned an error for valid input", test)
		}
	}

	invalidInput := []string{"warn ", "info error", ""}

	for _, test := range invalidInput {
		allErrs := validateRateLimitLogLevel(test, field.NewPath("logLevel"))
		if len(allErrs) == 0 {
			t.Errorf("validateRateLimitLogLevel(%q) didn't return error for invalid input", test)
		}
	}
}

func TestValidateJWTToken(t *testing.T) {
	t.Parallel()
	validTests := []struct {
		token string
		msg   string
	}{
		{
			token: "",
			msg:   "no token set",
		},
		{
			token: "$http_token",
			msg:   "http special variable usage",
		},
		{
			token: "$arg_token",
			msg:   "arg special variable usage",
		},
		{
			token: "$cookie_token",
			msg:   "cookie special variable usage",
		},
	}
	for _, test := range validTests {
		allErrs := validateJWTToken(test.token, field.NewPath("token"))
		if len(allErrs) != 0 {
			t.Errorf("validateJWTToken(%v) returned an error for valid input for the case of %v", test.token, test.msg)
		}
	}

	invalidTests := []struct {
		token string
		msg   string
	}{
		{
			token: "http_token",
			msg:   "missing $ prefix",
		},
		{
			token: "${http_token}",
			msg:   "usage of $ and curly braces",
		},
		{
			token: "$http_token$http_token",
			msg:   "multi variable usage",
		},
		{
			token: "something$http_token",
			msg:   "non variable usage",
		},
		{
			token: "$uri",
			msg:   "non special variable usage",
		},
	}
	for _, test := range invalidTests {
		allErrs := validateJWTToken(test.token, field.NewPath("token"))
		if len(allErrs) == 0 {
			t.Errorf("validateJWTToken(%v) didn't return error for invalid input for the case of %v", test.token, test.msg)
		}
	}
}

func TestValidateIngressMTLS(t *testing.T) {
	t.Parallel()
	tests := []struct {
		ing *v1.IngressMTLS
		msg string
	}{
		{
			ing: &v1.IngressMTLS{
				ClientCertSecret: "mtls-secret",
			},
			msg: "default",
		},
		{
			ing: &v1.IngressMTLS{
				ClientCertSecret: "mtls-secret",
				VerifyClient:     "on",
				VerifyDepth:      createPointerFromInt(1),
			},
			msg: "all parameters with default value",
		},
		{
			ing: &v1.IngressMTLS{
				ClientCertSecret: "ingress-mtls-secret",
				VerifyClient:     "optional",
				VerifyDepth:      createPointerFromInt(2),
			},
			msg: "optional parameters",
		},
	}
	for _, test := range tests {
		allErrs := validateIngressMTLS(test.ing, field.NewPath("ingressMTLS"))
		if len(allErrs) != 0 {
			t.Errorf("validateIngressMTLS() returned errors %v for valid input for the case of %v", allErrs, test.msg)
		}
	}
}

func TestValidateIngressMTLSInvalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		ing *v1.IngressMTLS
		msg string
	}{
		{
			ing: &v1.IngressMTLS{
				VerifyClient: "on",
			},
			msg: "no secret",
		},
		{
			ing: &v1.IngressMTLS{
				ClientCertSecret: "-foo-",
			},
			msg: "invalid secret name",
		},
		{
			ing: &v1.IngressMTLS{
				ClientCertSecret: "mtls-secret",
				VerifyClient:     "foo",
			},
			msg: "invalid verify client",
		},
		{
			ing: &v1.IngressMTLS{
				ClientCertSecret: "ingress-mtls-secret",
				VerifyClient:     "on",
				VerifyDepth:      createPointerFromInt(-1),
			},
			msg: "invalid depth",
		},
	}
	for _, test := range tests {
		allErrs := validateIngressMTLS(test.ing, field.NewPath("ingressMTLS"))
		if len(allErrs) == 0 {
			t.Errorf("validateIngressMTLS() returned no errors for invalid input for the case of %v", test.msg)
		}
	}
}

func TestValidateIngressMTLSVerifyClient(t *testing.T) {
	t.Parallel()
	validInput := []string{"on", "off", "optional", "optional_no_ca"}

	for _, test := range validInput {
		allErrs := validateIngressMTLSVerifyClient(test, field.NewPath("verifyClient"))
		if len(allErrs) != 0 {
			t.Errorf("validateIngressMTLSVerifyClient(%q) returned errors %v for valid input", allErrs, test)
		}
	}

	invalidInput := []string{"true", "false"}

	for _, test := range invalidInput {
		allErrs := validateIngressMTLSVerifyClient(test, field.NewPath("verifyClient"))
		if len(allErrs) == 0 {
			t.Errorf("validateIngressMTLSVerifyClient(%q) didn't return error for invalid input", test)
		}
	}
}

func TestValidateEgressMTLS(t *testing.T) {
	t.Parallel()
	tests := []struct {
		eg  *v1.EgressMTLS
		msg string
	}{
		{
			eg: &v1.EgressMTLS{
				TLSSecret: "mtls-secret",
			},
			msg: "tls secret",
		},
		{
			eg: &v1.EgressMTLS{
				TrustedCertSecret: "tls-secret",
				VerifyServer:      true,
				VerifyDepth:       createPointerFromInt(2),
				ServerName:        false,
			},
			msg: "verify server set to true",
		},
		{
			eg: &v1.EgressMTLS{
				VerifyServer: false,
			},
			msg: "verify server set to false",
		},
		{
			eg: &v1.EgressMTLS{
				SSLName: "foo.com",
			},
			msg: "ssl name",
		},
	}
	for _, test := range tests {
		allErrs := validateEgressMTLS(test.eg, field.NewPath("egressMTLS"))
		if len(allErrs) != 0 {
			t.Errorf("validateEgressMTLS() returned errors %v for valid input for the case of %v", allErrs, test.msg)
		}
	}
}

func TestValidateEgressMTLSInvalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		eg  *v1.EgressMTLS
		msg string
	}{
		{
			eg: &v1.EgressMTLS{
				VerifyServer: true,
			},
			msg: "verify server set to true",
		},
		{
			eg: &v1.EgressMTLS{
				TrustedCertSecret: "-foo-",
			},
			msg: "invalid secret name",
		},
		{
			eg: &v1.EgressMTLS{
				TrustedCertSecret: "ingress-mtls-secret",
				VerifyServer:      true,
				VerifyDepth:       createPointerFromInt(-1),
			},
			msg: "invalid depth",
		},
		{
			eg: &v1.EgressMTLS{
				SSLName: "foo.com;",
			},
			msg: "invalid name",
		},
	}

	for _, test := range tests {
		allErrs := validateEgressMTLS(test.eg, field.NewPath("egressMTLS"))
		if len(allErrs) == 0 {
			t.Errorf("validateEgressMTLS() returned no errors for invalid input for the case of %v", test.msg)
		}
	}
}

func TestValidateOIDCValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		oidc *v1.OIDC
		msg  string
	}{
		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenEndpoint: "https://oauth2.googleapis.com/token",
				JWKSURI:       "https://www.googleapis.com/oauth2/v3/certs",
				ClientID:      "random-string",
				ClientSecret:  "random-secret",
				Scope:         "openid",
				RedirectURI:   "/foo",
			},
			msg: "verify full oidc",
		},
		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/authorize",
				TokenEndpoint: "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/token",
				JWKSURI:       "https://login.microsoftonline.com/dd-fff-eee-1234-9be/discovery/v2.0/keys",
				ClientID:      "ff",
				ClientSecret:  "ff",
				Scope:         "openid+profile",
				RedirectURI:   "/_codexe",
			},
			msg: "verify azure endpoint",
		},
		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "http://keycloak.default.svc.cluster.local:8080/auth/realms/master/protocol/openid-connect/auth",
				TokenEndpoint: "http://keycloak.default.svc.cluster.local:8080/auth/realms/master/protocol/openid-connect/token",
				JWKSURI:       "http://keycloak.default.svc.cluster.local:8080/auth/realms/master/protocol/openid-connect/certs",
				ClientID:      "bar",
				ClientSecret:  "foo",
				Scope:         "openid",
			},
			msg: "domain with port number",
		},
		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "http://127.0.0.1:8080/auth/realms/master/protocol/openid-connect/auth",
				TokenEndpoint: "http://127.0.0.1:8080/auth/realms/master/protocol/openid-connect/token",
				JWKSURI:       "http://127.0.0.1:8080/auth/realms/master/protocol/openid-connect/certs",
				ClientID:      "client",
				ClientSecret:  "secret",
				Scope:         "openid",
			},
			msg: "ip address",
		},
	}

	for _, test := range tests {
		allErrs := validateOIDC(test.oidc, field.NewPath("oidc"))
		if len(allErrs) != 0 {
			t.Errorf("validateOIDC() returned errors %v for valid input for the case of %v", allErrs, test.msg)
		}
	}
}

func TestValidateOIDCInvalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		oidc *v1.OIDC
		msg  string
	}{
		{
			oidc: &v1.OIDC{
				RedirectURI: "/foo",
			},
			msg: "missing required field auth",
		},
		{
			oidc: &v1.OIDC{
				AuthEndpoint: "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/authorize",
				JWKSURI:      "https://login.microsoftonline.com/dd-fff-eee-1234-9be/discovery/v2.0/keys",
				ClientID:     "ff",
				ClientSecret: "ff",
				Scope:        "openid+profile",
			},
			msg: "missing required field token",
		},
		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/authorize",
				TokenEndpoint: "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/token",
				ClientID:      "ff",
				ClientSecret:  "ff",
				Scope:         "openid+profile",
			},
			msg: "missing required field jwk",
		},
		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/authorize",
				TokenEndpoint: "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/token",
				JWKSURI:       "https://login.microsoftonline.com/dd-fff-eee-1234-9be/discovery/v2.0/keys",
				ClientSecret:  "ff",
				Scope:         "openid+profile",
			},
			msg: "missing required field clientid",
		},
		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/authorize",
				TokenEndpoint: "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/token",
				JWKSURI:       "https://login.microsoftonline.com/dd-fff-eee-1234-9be/discovery/v2.0/keys",
				ClientID:      "ff",
				Scope:         "openid+profile",
			},
			msg: "missing required field client secret",
		},

		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/authorize",
				TokenEndpoint: "https://login.microsoftonline.com/dd-fff-eee-1234-9be/oauth2/v2.0/token",
				JWKSURI:       "https://login.microsoftonline.com/dd-fff-eee-1234-9be/discovery/v2.0/keys",
				ClientID:      "ff",
				ClientSecret:  "-ff-",
				Scope:         "openid+profile",
			},
			msg: "invalid secret name",
		},
		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "http://foo.\bar.com",
				TokenEndpoint: "http://keycloak.default",
				JWKSURI:       "http://keycloak.default",
				ClientID:      "bar",
				ClientSecret:  "foo",
				Scope:         "openid",
			},
			msg: "invalid URL",
		},
		{
			oidc: &v1.OIDC{
				AuthEndpoint:  "http://127.0.0.1:8080/auth/realms/master/protocol/openid-connect/auth",
				TokenEndpoint: "http://127.0.0.1:8080/auth/realms/master/protocol/openid-connect/token",
				JWKSURI:       "http://127.0.0.1:8080/auth/realms/master/protocol/openid-connect/certs",
				ClientID:      "$foo$bar",
				ClientSecret:  "secret",
				Scope:         "openid",
			},
			msg: "invalid chars in clientID",
		},
	}

	for _, test := range tests {
		allErrs := validateOIDC(test.oidc, field.NewPath("oidc"))
		if len(allErrs) == 0 {
			t.Errorf("validateOIDC() returned no errors for invalid input for the case of %v", test.msg)
		}
	}
}

func TestValidateClientID(t *testing.T) {
	t.Parallel()
	validInput := []string{"myid", "your.id", "id-sf-sjfdj.com", "foo_bar~vni"}

	for _, test := range validInput {
		allErrs := validateClientID(test, field.NewPath("clientID"))
		if len(allErrs) != 0 {
			t.Errorf("validateClientID(%q) returned errors %v for valid input", allErrs, test)
		}
	}

	invalidInput := []string{"$boo", "foo$bar", `foo_bar"vni`, `client\`}

	for _, test := range invalidInput {
		allErrs := validateClientID(test, field.NewPath("clientID"))
		if len(allErrs) == 0 {
			t.Errorf("validateClientID(%q) didn't return error for invalid input", test)
		}
	}
}

func TestValidateOIDCScope(t *testing.T) {
	t.Parallel()
	validInput := []string{"openid", "openid+profile", "openid+email", "openid+phone"}

	for _, test := range validInput {
		allErrs := validateOIDCScope(test, field.NewPath("scope"))
		if len(allErrs) != 0 {
			t.Errorf("validateOIDCScope(%q) returned errors %v for valid input", allErrs, test)
		}
	}

	invalidInput := []string{"profile", "openid+web", `openid+foobar.com`}

	for _, test := range invalidInput {
		allErrs := validateOIDCScope(test, field.NewPath("scope"))
		if len(allErrs) == 0 {
			t.Errorf("validateOIDCScope(%q) didn't return error for invalid input", test)
		}
	}
}

func TestValidateURL(t *testing.T) {
	t.Parallel()
	validInput := []string{"http://google.com/auth", "https://foo.bar/baz", "http://127.0.0.1/bar", "http://openid.connect.com:8080/foo"}

	for _, test := range validInput {
		allErrs := validateURL(test, field.NewPath("authEndpoint"))
		if len(allErrs) != 0 {
			t.Errorf("validateURL(%q) returned errors %v for valid input", allErrs, test)
		}
	}

	invalidInput := []string{"www.google..foo.com", "http://{foo.bar", `https://google.foo\bar`, "http://foo.bar:8080", "http://foo.bar:812345/fooo"}

	for _, test := range invalidInput {
		allErrs := validateURL(test, field.NewPath("authEndpoint"))
		if len(allErrs) == 0 {
			t.Errorf("validateURL(%q) didn't return error for invalid input", test)
		}
	}
}

func TestValidateWAF(t *testing.T) {
	t.Parallel()
	tests := []struct {
		waf *v1.WAF
		msg string
	}{
		{
			waf: &v1.WAF{
				Enable: true,
			},
			msg: "waf enabled",
		},
		{
			waf: &v1.WAF{
				Enable:   true,
				ApPolicy: "ns1/waf-pol",
			},
			msg: "cross ns reference",
		},
		{
			waf: &v1.WAF{
				Enable: true,
				SecurityLog: &v1.SecurityLog{
					Enable:  true,
					LogDest: "syslog:server=8.7.7.7:517",
				},
			},
			msg: "custom logdest",
		},
	}

	for _, test := range tests {
		allErrs := validateWAF(test.waf, field.NewPath("waf"))
		if len(allErrs) != 0 {
			t.Errorf("validateWAF() returned errors %v for valid input for the case of %v", allErrs, test.msg)
		}
	}
}

func TestValidateWAFInvalid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		waf *v1.WAF
		msg string
	}{
		{
			waf: &v1.WAF{
				Enable:   true,
				ApPolicy: "ns1/ap-pol/ns2",
			},
			msg: "invalid apPolicy format",
		},
		{
			waf: &v1.WAF{
				Enable: true,
				SecurityLog: &v1.SecurityLog{
					Enable:  true,
					LogDest: "stdout",
				},
			},
			msg: "invalid logdest",
		},
		{
			waf: &v1.WAF{
				Enable: true,
				SecurityLog: &v1.SecurityLog{
					Enable:    true,
					ApLogConf: "ns1/log-conf/ns2",
				},
			},
			msg: "invalid logConf format",
		},
	}

	for _, test := range tests {
		allErrs := validateWAF(test.waf, field.NewPath("waf"))
		if len(allErrs) == 0 {
			t.Errorf("validateWAF() returned no errors for invalid input for the case of %v", test.msg)
		}
	}
}

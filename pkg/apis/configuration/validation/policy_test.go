package validation

import (
	"testing"

	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidatePolicy(t *testing.T) {
	policy := &v1.Policy{
		Spec: v1.PolicySpec{
			AccessControl: &v1.AccessControl{
				Allow: []string{"127.0.0.1"},
			},
		},
	}
	isPlus := false

	err := ValidatePolicy(policy, isPlus)
	if err != nil {
		t.Errorf("ValidatePolicy() returned error %v for valid input", err)
	}
}

func TestValidatePolicyFails(t *testing.T) {
	policy := &v1.Policy{
		Spec: v1.PolicySpec{},
	}
	isPlus := false

	err := ValidatePolicy(policy, isPlus)
	if err == nil {
		t.Errorf("ValidatePolicy() returned no error for invalid input")
	}

	multiPolicy := &v1.Policy{
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
	}

	err = ValidatePolicy(multiPolicy, isPlus)
	if err == nil {
		t.Errorf("ValidatePolicy() returned no error for invalid input")
	}
}

func TestValidateAccessControl(t *testing.T) {
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
	var validInput = []string{"32", "32k", "32K", "10m"}

	for _, test := range validInput {
		allErrs := validateRateLimitZoneSize(test, field.NewPath("size"))
		if len(allErrs) != 0 {
			t.Errorf("validateRateLimitZoneSize(%q) returned an error for valid input", test)
		}
	}

	var invalidInput = []string{"", "31", "31k", "0", "0M"}

	for _, test := range invalidInput {
		allErrs := validateRateLimitZoneSize(test, field.NewPath("size"))
		if len(allErrs) == 0 {
			t.Errorf("validateRateLimitZoneSize(%q) didn't return error for invalid input", test)
		}
	}
}

func TestValidateRateLimitLogLevel(t *testing.T) {
	var validInput = []string{"error", "info", "warn", "notice"}

	for _, test := range validInput {
		allErrs := validateRateLimitLogLevel(test, field.NewPath("logLevel"))
		if len(allErrs) != 0 {
			t.Errorf("validateRateLimitLogLevel(%q) returned an error for valid input", test)
		}
	}

	var invalidInput = []string{"warn ", "info error", ""}

	for _, test := range invalidInput {
		allErrs := validateRateLimitLogLevel(test, field.NewPath("logLevel"))
		if len(allErrs) == 0 {
			t.Errorf("validateRateLimitLogLevel(%q) didn't return error for invalid input", test)
		}
	}
}

func TestValidateJWTToken(t *testing.T) {
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
	var validInput = []string{"on", "off", "optional", "optional_no_ca"}

	for _, test := range validInput {
		allErrs := validateIngressMTLSVerifyClient(test, field.NewPath("verifyClient"))
		if len(allErrs) != 0 {
			t.Errorf("validateIngressMTLSVerifyClient(%q) returned errors %v for valid input", allErrs, test)
		}
	}

	var invalidInput = []string{"true", "false"}

	for _, test := range invalidInput {
		allErrs := validateIngressMTLSVerifyClient(test, field.NewPath("verifyClient"))
		if len(allErrs) == 0 {
			t.Errorf("validateIngressMTLSVerifyClient(%q) didn't return error for invalid input", test)
		}
	}
}

func TestValidateEgressMTLS(t *testing.T) {
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

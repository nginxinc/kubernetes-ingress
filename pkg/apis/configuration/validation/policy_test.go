package validation

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidatePolicy(t *testing.T) {
	policy := &v1alpha1.Policy{
		Spec: v1alpha1.PolicySpec{
			AccessControl: &v1alpha1.AccessControl{
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
	policy := &v1alpha1.Policy{
		Spec: v1alpha1.PolicySpec{},
	}
	isPlus := false

	err := ValidatePolicy(policy, isPlus)
	if err == nil {
		t.Errorf("ValidatePolicy() returned no error for invalid input")
	}

	multiPolicy := &v1alpha1.Policy{
		Spec: v1alpha1.PolicySpec{
			AccessControl: &v1alpha1.AccessControl{
				Allow: []string{"127.0.0.1"},
			},
			RateLimit: &v1alpha1.RateLimit{
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
	validInput := []*v1alpha1.AccessControl{
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
		accessControl *v1alpha1.AccessControl
		msg           string
	}{
		{
			accessControl: &v1alpha1.AccessControl{
				Allow: nil,
				Deny:  nil,
			},
			msg: "neither allow nor deny is defined",
		},
		{
			accessControl: &v1alpha1.AccessControl{
				Allow: []string{},
				Deny:  []string{},
			},
			msg: "both allow and deny are defined",
		},
		{
			accessControl: &v1alpha1.AccessControl{
				Allow: []string{"invalid"},
			},
			msg: "invalid allow",
		},
		{
			accessControl: &v1alpha1.AccessControl{
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
		rateLimit *v1alpha1.RateLimit
		msg       string
	}{
		{
			rateLimit: &v1alpha1.RateLimit{
				Rate:     "10r/s",
				ZoneSize: "10M",
				Key:      "${request_uri}",
			},
			msg: "only required fields are set",
		},
		{
			rateLimit: &v1alpha1.RateLimit{
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
	for _, test := range tests {
		allErrs := validateRateLimit(test.rateLimit, field.NewPath("rateLimit"))
		if len(allErrs) > 0 {
			t.Errorf("validateRateLimit() returned errors %v for valid input for the case of %v", allErrs, test.msg)
		}
	}
}

func createInvalidRateLimit(f func(r *v1alpha1.RateLimit)) *v1alpha1.RateLimit {
	validRateLimit := &v1alpha1.RateLimit{
		Rate:     "10r/s",
		ZoneSize: "10M",
		Key:      "${request_uri}",
	}
	f(validRateLimit)
	return validRateLimit
}

func TestValidateRateLimitFails(t *testing.T) {
	tests := []struct {
		rateLimit *v1alpha1.RateLimit
		msg       string
	}{
		{
			rateLimit: createInvalidRateLimit(func(r *v1alpha1.RateLimit) {
				r.Rate = "0r/s"
			}),
			msg: "invalid rateLimit rate",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1alpha1.RateLimit) {
				r.Key = "${fail}"
			}),
			msg: "invalid rateLimit key variable use",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1alpha1.RateLimit) {
				r.Delay = createPointerFromInt(0)
			}),
			msg: "invalid rateLimit delay",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1alpha1.RateLimit) {
				r.Burst = createPointerFromInt(0)
			}),
			msg: "invalid rateLimit burst",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1alpha1.RateLimit) {
				r.ZoneSize = "31k"
			}),
			msg: "invalid rateLimit zoneSize",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1alpha1.RateLimit) {
				r.RejectCode = createPointerFromInt(600)
			}),
			msg: "invalid rateLimit rejectCode",
		},
		{
			rateLimit: createInvalidRateLimit(func(r *v1alpha1.RateLimit) {
				r.LogLevel = "invalid"
			}),
			msg: "invalid rateLimit logLevel",
		},
	}
	for _, test := range tests {
		allErrs := validateRateLimit(test.rateLimit, field.NewPath("rateLimit"))
		if len(allErrs) == 0 {
			t.Errorf("validateRateLimit() returned no errors for invalid input for the case of %v", test.msg)
		}
	}
}

func TestValidateJWT(t *testing.T) {
	tests := []struct {
		jwt *v1alpha1.JWTAuth
		msg string
	}{
		{
			jwt: &v1alpha1.JWTAuth{
				Realm:  "My Product API",
				Secret: "my-jwk",
			},
			msg: "basic",
		},
		{
			jwt: &v1alpha1.JWTAuth{
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
		jwt *v1alpha1.JWTAuth
	}{
		{
			jwt: &v1alpha1.JWTAuth{
				Realm: "My Product API",
			},
			msg: "missing secret",
		},
		{
			jwt: &v1alpha1.JWTAuth{
				Secret: "my-jwk",
			},
			msg: "missing realm",
		},
		{
			jwt: &v1alpha1.JWTAuth{
				Realm:  "My Product API",
				Secret: "my-jwk",
				Token:  "$uri",
			},
			msg: "invalid variable use in token",
		},
		{
			jwt: &v1alpha1.JWTAuth{
				Realm:  "My Product API",
				Secret: "my-\"jwk",
			},
			msg: "invalid secret name",
		},
		{
			jwt: &v1alpha1.JWTAuth{
				Realm:  "My \"Product API",
				Secret: "my-jwk",
			},
			msg: "invalid realm due to escaped string",
		},
		{
			jwt: &v1alpha1.JWTAuth{
				Realm:  "My Product ${api}",
				Secret: "my-jwk",
			},
			msg: "invalid variable use in realm with curly braces",
		},
		{
			jwt: &v1alpha1.JWTAuth{
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

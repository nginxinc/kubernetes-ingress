package validation

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func createPointerFromInt(n int) *int {
	return &n
}

func TestValidateVariable(t *testing.T) {
	var validVars = map[string]bool{
		"scheme":                 true,
		"http_x_forwarded_proto": true,
		"request_uri":            true,
		"host":                   true,
	}

	validTests := []string{
		"scheme",
		"http_x_forwarded_proto",
		"request_uri",
		"host",
	}
	for _, nVar := range validTests {
		allErrs := validateVariable(nVar, validVars, field.NewPath("url"))
		if len(allErrs) != 0 {
			t.Errorf("validateVariable(%v) returned errors %v for valid input", nVar, allErrs)
		}
	}
}

func TestValidateVariableFails(t *testing.T) {
	var validVars = map[string]bool{
		"host": true,
	}
	invalidVars := []string{
		"",
		"hostinvalid.com",
		"$a",
		"host${host}",
		"host${host}}",
		"host$${host}",
	}
	for _, nVar := range invalidVars {
		allErrs := validateVariable(nVar, validVars, field.NewPath("url"))
		if len(allErrs) == 0 {
			t.Errorf("validateVariable(%v) returned no errors for invalid input", nVar)
		}
	}
}

func TestValidateSpecialVariable(t *testing.T) {
	specialVars := []string{"arg_username", "arg_user_name", "http_header_name", "cookie_cookie_name"}
	for _, v := range specialVars {
		allErrs := validateSpecialVariable(v, field.NewPath("variable"))
		if len(allErrs) != 0 {
			t.Errorf("validateSpecialVariable(%v) returned errors for valid case: %v", v, allErrs)
		}
	}
}

func TestValidateSpecialVariableFails(t *testing.T) {
	specialVars := []string{"arg_invalid%", "http_header+invalid", "cookie_cookie_name?invalid"}
	for _, v := range specialVars {
		allErrs := validateSpecialVariable(v, field.NewPath("variable"))
		if len(allErrs) == 0 {
			t.Errorf("validateSpecialVariable(%v) returned no errors for invalid case", v)
		}
	}
}

func TestValidateStringWithVariables(t *testing.T) {
	testStrings := []string{
		"",
		"${scheme}",
		"${scheme}${host}",
		"foo.bar",
	}
	validVars := map[string]bool{"scheme": true, "host": true}

	for _, test := range testStrings {
		allErrs := validateStringWithVariables(test, field.NewPath("string"), nil, validVars)
		if len(allErrs) != 0 {
			t.Errorf("validateStringWithVariables(%v) returned errors for valid input: %v", test, allErrs)
		}
	}

	specialVars := []string{"arg", "http", "cookie"}
	testStringsSpecial := []string{
		"${arg_username}",
		"${http_header_name}",
		"${cookie_cookie_name}",
	}

	for _, test := range testStringsSpecial {
		allErrs := validateStringWithVariables(test, field.NewPath("string"), specialVars, validVars)
		if len(allErrs) != 0 {
			t.Errorf("validateStringWithVariables(%v) returned errors for valid input: %v", test, allErrs)
		}
	}
}

func TestValidateStringWithVariablesFail(t *testing.T) {
	testStrings := []string{
		"$scheme}",
		"${sch${eme}${host}",
		"host$",
		"${host",
		"${invalid}",
	}
	validVars := map[string]bool{"scheme": true, "host": true}

	for _, test := range testStrings {
		allErrs := validateStringWithVariables(test, field.NewPath("string"), nil, validVars)
		if len(allErrs) == 0 {
			t.Errorf("validateStringWithVariables(%v) returned no errors for invalid input", test)
		}
	}

	specialVars := []string{"arg", "http", "cookie"}
	testStringsSpecial := []string{
		"${arg_username%}",
		"${http_header-name}",
		"${cookie_cookie?name}",
	}

	for _, test := range testStringsSpecial {
		allErrs := validateStringWithVariables(test, field.NewPath("string"), specialVars, validVars)
		if len(allErrs) == 0 {
			t.Errorf("validateStringWithVariables(%v) returned no errors for invalid input", test)
		}
	}
}

func TestValidateSize(t *testing.T) {
	var validInput = []string{"", "4k", "8K", "16m", "32M"}
	for _, test := range validInput {
		allErrs := validateSize(test, field.NewPath("size-field"))
		if len(allErrs) != 0 {
			t.Errorf("validateSize(%q) returned an error for valid input", test)
		}
	}

	var invalidInput = []string{"55mm", "2mG", "6kb", "-5k", "1L", "5G"}
	for _, test := range invalidInput {
		allErrs := validateSize(test, field.NewPath("size-field"))
		if len(allErrs) == 0 {
			t.Errorf("validateSize(%q) didn't return error for invalid input.", test)
		}
	}
}

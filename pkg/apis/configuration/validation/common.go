package validation

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	escapedStringsFmt    = `([^"\\]|\\.)*`
	escapedStringsErrMsg = `must have all '"' (double quotes) escaped and must not end with an unescaped '\' (backslash)`
)

var escapedStringsFmtRegexp = regexp.MustCompile("^" + escapedStringsFmt + "$")

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

func mapToPrettyString(m map[string]bool) string {
	var out []string

	for k := range m {
		out = append(out, k)
	}

	return strings.Join(out, ", ")
}

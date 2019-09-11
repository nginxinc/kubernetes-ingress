package version1

import (
	"strings"
	"text/template"
)

// SplitInput splits the input from "," and returns an array of strings
func SplitInput(s string, delim string) []string {
	return strings.Split(s, delim)
}

// TrimInput trims the leading and trailing spaces in the string
func TrimInput(s string) string {
	return strings.TrimSpace(s)
}

// HelperFunctions to parse the annotations
var helperFunctions = template.FuncMap{
	"splitinput": SplitInput, //returns array of strings
	"triminput":  TrimInput,  //returns string with trimmed leading and trailing spaces
}

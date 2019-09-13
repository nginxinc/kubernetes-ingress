package version1

import (
	"strings"
	"text/template"
)

// splitinput splits the input from "," and returns an array of strings
func splitinput(s string, delim string) []string {
	return strings.Split(s, delim)
}

// triminput trims the leading and trailing spaces in the string
func triminput(s string) string {
	return strings.TrimSpace(s)
}

var helperFunctions = template.FuncMap{
	"split": splitinput, //returns array of strings
	"trim":  triminput,  //returns string with trimmed leading and trailing spaces
}

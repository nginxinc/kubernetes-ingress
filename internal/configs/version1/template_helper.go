package version1

import (
	"strings"
	"text/template"
)

func split(s string, delim string) []string {
	return strings.Split(s, delim)
}

func trim(s string) string {
	return strings.TrimSpace(s)
}

var helperFunctions = template.FuncMap{
	"split": split,
	"trim":  trim,
}

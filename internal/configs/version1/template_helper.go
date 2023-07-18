package version1

import (
	"fmt"
	"strings"
	"text/template"
)

func split(s string, delim string) []string {
	return strings.Split(s, delim)
}

func trim(s string) string {
	return strings.TrimSpace(s)
}

// makePathRegex takes a string representing a location path
// and a map representing Ingress annotations.
// It returns a location path with added regular expression modifier.
// See [Location Directive].
//
// [Location Directive]: https://nginx.org/en/docs/http/ngx_http_core_module.html#location
func makePathRegex(path string, annotations map[string]string) string {
	p, ok := annotations["nginx.org/path-regex"]
	if !ok {
		return path
	}
	switch p {
	case "case_sensitive":
		return fmt.Sprintf("~ \"^%s\"", path)
	case "case_insensitive":
		return fmt.Sprintf("~* \"^%s\"", path)
	case "exact":
		return fmt.Sprintf("= \"%s\"", path)
	default:
		return path
	}
}

var helperFunctions = template.FuncMap{
	"split":         split,
	"trim":          trim,
	"makePathRegex": makePathRegex,
}

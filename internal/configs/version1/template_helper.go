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

// makeLocationPath takes location and Ingress annotations and returns
// modified location path with added regex modifier or the original path
// if no path-regex annotation is present in ingressAnnotations
// or in Location's Ingress.
//
// Annotations 'path-regex' are set only on Minions. If set on Master Ingress,
// they are ignored and have no effect.
func makeLocationPath(loc *Location, ingressAnnotations map[string]string) string {
	if loc.MinionIngress != nil {
		// Case when annotation 'path-regex' set on Location's Minion.
		ingressType, isMergeable := loc.MinionIngress.Annotations["nginx.org/mergeable-ingress-type"]
		regexType, hasRegex := loc.MinionIngress.Annotations["nginx.org/path-regex"]

		if isMergeable && ingressType == "minion" && hasRegex {
			return makePathWithRegex(loc.Path, regexType)
		}
		if isMergeable && ingressType == "minion" && !hasRegex {
			return loc.Path
		}
	}

	// Case when annotation 'path-regex' set on Ingress (including Master).
	regexType, ok := ingressAnnotations["nginx.org/path-regex"]
	if !ok {
		return loc.Path
	}
	return makePathWithRegex(loc.Path, regexType)
}

// makePathWithRegex takes a path representing a location and a regexType
// (one of `case_sensitive`, `case_insensitive` or `exact`).
// It returns a location path with added regular expression modifier.
// See [Location Directive].
//
// [Location Directive]: https://nginx.org/en/docs/http/ngx_http_core_module.html#location
func makePathWithRegex(path, regexType string) string {
	switch regexType {
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
	"split":            split,
	"trim":             trim,
	"contains":         strings.Contains,
	"hasPrefix":        strings.HasPrefix,
	"hasSuffix":        strings.HasSuffix,
	"toLower":          strings.ToLower,
	"toUpper":          strings.ToUpper,
	"makeLocationPath": makeLocationPath,
}

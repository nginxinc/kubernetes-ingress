package version1

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"regexp"
	"strings"
	"text/template"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/commonhelpers"
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

var setHeader = regexp.MustCompile("[a-zA-Z]+$")

func validateProxySetHeader(header string) error {
	header = strings.TrimSpace(header)

	if !setHeader.MatchString(header) {
		return errors.New("invalid header syntax")
	}
	return nil
}

func printDefaultHeaderValues(headerParts []string, headerName string) string {
	headerValue := strings.TrimSpace(headerParts[0])
	headerValue = strings.ReplaceAll(headerValue, "-", "_")
	headerValue = strings.ToLower(headerValue)
	return fmt.Sprintf("\n\t\tproxy_set_header %s $http_%s;", headerName, headerValue)
}

func printHeadersGreaterThanOne(headerParts []string, header string, headerName string) (string, error) {
	headerValue := strings.TrimSpace(headerParts[1])
	if strings.Contains(headerValue, " ") {
		return "", errors.New("multiple values found in header: " + header)
	}
	return fmt.Sprintf("\n\t\tproxy_set_header %s %q;", headerName, headerValue), nil
}

func splittingHeaders(header string) (string, []string, string) {
	header = strings.TrimSpace(header)
	headerParts := strings.SplitN(header, " ", 2)
	headerName := strings.TrimSpace(headerParts[0])
	return header, headerParts, headerName
}

func printMinionProySetHeaders(loc *Location, result *strings.Builder, minionHeaders map[string]bool) (*strings.Builder, map[string]bool, error) {
	proxySetHeaders, ok := loc.MinionIngress.Annotations["nginx.org/proxy-set-headers"]
	if ok {
		headers := strings.Split(proxySetHeaders, ",")
		for _, header := range headers {
			header, headerParts, headerName := splittingHeaders(header)
			err := validateProxySetHeader(headerName)
			if err != nil {
				return nil, nil, err
			}
			if len(headerParts) > 1 {
				output, err := printHeadersGreaterThanOne(headerParts, header, headerName)
				minionHeaders[headerName] = true
				if err != nil {
					return nil, nil, err
				}
				result.WriteString(output)
			} else {
				output := printDefaultHeaderValues(headerParts, headerName)
				result.WriteString(output)
			}
		}
	}
	return result, minionHeaders, nil
}

func printMasterProySetHeaders(ingressAnnotations map[string]string, result *strings.Builder, minionHeaders map[string]bool) (*strings.Builder, error) {
	proxySetHeaders, ok := ingressAnnotations["nginx.org/proxy-set-headers"]
	if ok {
		headers := strings.Split(proxySetHeaders, ",")
		for _, header := range headers {
			header, headerParts, headerName := splittingHeaders(header)
			if _, ok := minionHeaders[headerName]; !ok {
				if err := validateProxySetHeader(headerName); err != nil {
					return nil, err
				}
				if len(headerParts) > 1 {
					output, err := printHeadersGreaterThanOne(headerParts, header, headerName)
					if err != nil {
						return nil, err
					}
					result.WriteString(output)
				} else {
					output := printDefaultHeaderValues(headerParts, headerName)
					result.WriteString(output)
				}
			}
		}
	}
	return result, nil
}

func printNotMergableProxySetHeaders(ingressAnnotations map[string]string) (string, error) {
	var result string
	proxySetHeaders, ok := ingressAnnotations["nginx.org/proxy-set-headers"]
	if ok {
		headers := strings.Split(proxySetHeaders, ",")
		for _, header := range headers {
			header, headerParts, headerName := splittingHeaders(header)
			err := validateProxySetHeader(headerName)
			if err != nil {
				return "", err
			}
			if len(headerParts) > 1 {
				output, err := printHeadersGreaterThanOne(headerParts, header, headerName)
				if err != nil {
					return "nil", err
				}
				result += output
			} else {
				output := printDefaultHeaderValues(headerParts, headerName)
				result += output
			}
		}
	}
	return result, nil
}

func generateProxySetHeaders(loc *Location, ingressAnnotations map[string]string) (string, error) {
	var result strings.Builder
	isMergable := loc.MinionIngress != nil
	if !isMergable {
		result, err := printNotMergableProxySetHeaders(ingressAnnotations)
		if err != nil {
			return "", err
		}
		return result, nil
	}
	glog.Infof("Proxy Set Header for %s - %s", loc.Path, loc.MinionIngress.Annotations["nginx.org/proxy-set-headers"])
	minionHeaders := make(map[string]bool)
	_, minionHeaders, err := printMinionProySetHeaders(loc, &result, minionHeaders)
	if err != nil {
		return "", err
	}
	_, err = printMasterProySetHeaders(ingressAnnotations, &result, minionHeaders)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

var helperFunctions = template.FuncMap{
	"split":                   split,
	"trim":                    trim,
	"contains":                strings.Contains,
	"hasPrefix":               strings.HasPrefix,
	"hasSuffix":               strings.HasSuffix,
	"toLower":                 strings.ToLower,
	"toUpper":                 strings.ToUpper,
	"makeLocationPath":        makeLocationPath,
	"makeSecretPath":          commonhelpers.MakeSecretPath,
	"generateProxySetHeaders": generateProxySetHeaders,
}

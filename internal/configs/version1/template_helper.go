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

func replace(s string, old, new string) string { return strings.ReplaceAll(s, old, new) }

func concat(s1, s2 string) string { return s1 + s2 }

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

//func printDefaultHeaderValues(headerParts []string, headerName string) string {
//	var result strings.Builder
//	headerValue := strings.TrimSpace(headerParts[0])
//	headerValue = strings.ReplaceAll(headerValue, "-", "_")
//	headerValue = strings.ToLower(headerValue)
//	result.WriteString(fmt.Sprintf("\n\t\tproxy_set_header %s $http_%s;", headerName, headerValue))
//	return result.String()
//}
//
//func printHeadersLessThanOne(headerParts []string, header string, headerName string) (string, error) {
//	var result strings.Builder
//	headerValue := strings.TrimSpace(headerParts[1])
//	if strings.Contains(headerValue, " ") {
//		return "", errors.New("multiple values found in header: " + header)
//	}
//	result.WriteString(fmt.Sprintf("\n\t\tproxy_set_header %s %q;", headerName, headerValue))
//	return result.String(), nil
//}
//
//func splittingHeaders(header string) (string, []string, string) {
//	header = strings.TrimSpace(header)
//	headerParts := strings.SplitN(header, " ", 2)
//	headerName := strings.TrimSpace(headerParts[0])
//	return header, headerParts, headerName
//}

// generateProxySetHeaders takes a location and an ingress annotations map
// and generates proxy_set_header directives based on the nginx.org/proxy-set-headers annotation.
// It returns a string containing the generated Nginx configuration.
func generateProxySetHeaders(loc *Location, ingressAnnotations map[string]string) (string, error) {
	var result strings.Builder
	var proxySetHeaders, ingressType string
	var headers []string
	isMergable := loc.MinionIngress != nil
	var ok bool
	if !isMergable {
		glog.Info("In Ingress")
		proxySetHeaders, ok = ingressAnnotations["nginx.org/proxy-set-headers"]
		if ok {
			headers = strings.Split(proxySetHeaders, ",")
			for _, header := range headers {
				header = strings.TrimSpace(header)
				headerParts := strings.SplitN(header, " ", 2)
				headerName := strings.TrimSpace(headerParts[0])
				err := validateProxySetHeader(headerName)
				if err != nil {
					return "", err
				}
				if len(headerParts) > 1 {
					headerValue := strings.TrimSpace(headerParts[1])
					if strings.Contains(headerValue, " ") {
						return "", errors.New("multiple values found in header: " + header)
					}
					result.WriteString(fmt.Sprintf("\n\t\tproxy_set_header %s %q;", headerName, headerValue))
				} else {
					headerValue := strings.TrimSpace(headerParts[0])
					headerValue = strings.ReplaceAll(headerValue, "-", "_")
					headerValue = strings.ToLower(headerValue)
					result.WriteString(fmt.Sprintf("\n\t\tproxy_set_header %s $http_%s;", headerName, headerValue))
				}
			}
		}
		return result.String(), nil
	}
	if loc.MinionIngress != nil {
		ingressType, isMergable = loc.MinionIngress.Annotations["nginx.org/mergeable-ingress-type"]
		if ingressType == "minion" {
			glog.Infof("Proxy Set Header for %s - %s", loc.Path, loc.MinionIngress.Annotations["nginx.org/proxy-set-headers"])

			minionHeaders := make(map[string]bool)
			proxySetHeaders, ok = loc.MinionIngress.Annotations["nginx.org/proxy-set-headers"]
			if ok {
				headers = strings.Split(proxySetHeaders, ",")
				for _, header := range headers {
					header = strings.TrimSpace(header)
					headerParts := strings.SplitN(header, " ", 2)
					headerName := strings.TrimSpace(headerParts[0])
					minionHeaders[headerName] = true
					err := validateProxySetHeader(headerName)
					if err != nil {
						return "", err
					}
					if len(headerParts) > 1 {
						headerValue := strings.TrimSpace(headerParts[1])
						if strings.Contains(headerValue, " ") {
							return "", errors.New("multiple values found in header: " + header)
						}
						result.WriteString(fmt.Sprintf("\n\t\tproxy_set_header %s %q;", headerName, headerValue))
					} else {
						headerValue := strings.TrimSpace(headerParts[0])
						headerValue = strings.ReplaceAll(headerValue, "-", "_")
						headerValue = strings.ToLower(headerValue)
						result.WriteString(fmt.Sprintf("\n\t\tproxy_set_header %s $http_%s;", headerName, headerValue))
					}
				}
			}
			proxySetHeaders, ok = ingressAnnotations["nginx.org/proxy-set-headers"]
			if ok {
				headers = strings.Split(proxySetHeaders, ",")
				for _, header := range headers {
					header = strings.TrimSpace(header)
					headerParts := strings.SplitN(header, " ", 2)
					headerName := strings.TrimSpace(headerParts[0])
					if _, ok := minionHeaders[headerName]; !ok {
						if err := validateProxySetHeader(headerName); err != nil {
							return "", err
						}
						if len(headerParts) > 1 {
							headerValue := strings.TrimSpace(headerParts[1])
							if strings.Contains(headerValue, " ") {
								return "", errors.New("multiple values found in header: " + header)
							}
							result.WriteString(fmt.Sprintf("\n\t\tproxy_set_header %s %q;", headerName, headerValue))
						} else {
							headerValue := strings.TrimSpace(headerParts[0])
							headerValue = strings.ReplaceAll(headerValue, "-", "_")
							headerValue = strings.ToLower(headerValue)
							result.WriteString(fmt.Sprintf("\n\t\tproxy_set_header %s $http_%s;", headerName, headerValue))
						}
					}
				}
			}
		}
	}
	return result.String(), nil
}

var helperFunctions = template.FuncMap{
	"split":                   split,
	"trim":                    trim,
	"replace":                 replace,
	"concat":                  concat,
	"contains":                strings.Contains,
	"hasPrefix":               strings.HasPrefix,
	"hasSuffix":               strings.HasSuffix,
	"toLower":                 strings.ToLower,
	"toUpper":                 strings.ToUpper,
	"makeLocationPath":        makeLocationPath,
	"makeSecretPath":          commonhelpers.MakeSecretPath,
	"generateProxySetHeaders": generateProxySetHeaders,
}

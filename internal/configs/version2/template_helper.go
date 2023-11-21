package version2

import (
	"strconv"
	"strings"
	"text/template"
)

type ListenerType int

const (
	http ListenerType = iota
	https
)

func headerListToCIMap(headers []Header) map[string]string {
	ret := make(map[string]string)

	for _, header := range headers {
		ret[strings.ToLower(header.Name)] = header.Value
	}

	return ret
}

func hasCIKey(key string, d map[string]string) bool {
	_, ok := d[strings.ToLower(key)]
	return ok
}

func makeListener(listenerType ListenerType, s Server) string {
	var directives string

	if !s.CustomListeners {
		directives += "listen"
		if listenerType == http {
			directives += " 80"
		} else if listenerType == https {
			directives += " 443 ssl"
		}
		if s.ProxyProtocol {
			directives += " proxy_protocol"
		}
		directives += ";\n"

		if !s.DisableIPV6 {
			directives += "listen [::]:"
			if listenerType == http {
				directives += "80"
			} else if listenerType == https {
				directives += "443 ssl"
			}
			if s.ProxyProtocol {
				directives += " proxy_protocol"
			}
			directives += ";\n"
		}
	} else {
		if listenerType == http && s.HTTPPort > 0 || listenerType == https && s.HTTPSPort > 0 {
			directives += "listen"
			if listenerType == http {
				directives += " " + strconv.Itoa(s.HTTPPort)
			} else if listenerType == https {
				directives += " " + strconv.Itoa(s.HTTPSPort) + " ssl"
			}

			if s.ProxyProtocol {
				directives += " proxy_protocol"
			}
			directives += ";\n"

			if !s.DisableIPV6 {
				directives += "listen [::]:"
				if listenerType == http {
					directives += strconv.Itoa(s.HTTPPort)
				} else if listenerType == https {
					directives += strconv.Itoa(s.HTTPSPort) + " ssl"
				}
				if s.ProxyProtocol {
					directives += " proxy_protocol"
				}
				directives += ";\n"
			}
		}
	}

	return directives
}

func makeHTTPListener(s Server) string {
	return makeListener(http, s)
}

func makeHTTPSListener(s Server) string {
	return makeListener(https, s)
}

var helperFunctions = template.FuncMap{
	"headerListToCIMap": headerListToCIMap,
	"hasCIKey":          hasCIKey,
	"contains":          strings.Contains,
	"hasPrefix":         strings.HasPrefix,
	"hasSuffix":         strings.HasSuffix,
	"toLower":           strings.ToLower,
	"toUpper":           strings.ToUpper,
	"makeHTTPListener":  makeHTTPListener,
	"makeHTTPSListener": makeHTTPSListener,
}

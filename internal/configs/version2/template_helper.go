package version2

import (
	"strconv"
	"strings"
	"text/template"
)

type Protocol int

const (
	http Protocol = iota
	https
)

type ListenType int

const (
	ipv4 ListenType = iota
	ipv6
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

func makeListener(listenerType Protocol, s Server) string {
	var directives string

	if !s.CustomListeners {
		directives += buildDefaultListenerDirectives(listenerType, s)
	} else {
		directives += buildCustomListenerDirectives(listenerType, s)
	}

	return directives
}

func buildDefaultListenerDirectives(listenerType Protocol, s Server) string {
	var directives string
	port := getDefaultPort(listenerType)

	directives += buildListenDirective(port, s.ProxyProtocol, ipv4)

	if !s.DisableIPV6 {
		directives += buildListenDirective(port, s.ProxyProtocol, ipv6)
	}

	return directives
}

func buildCustomListenerDirectives(listenerType Protocol, s Server) string {
	var directives string

	if (listenerType == http && s.HTTPPort > 0) || (listenerType == https && s.HTTPSPort > 0) {
		port := getCustomPort(listenerType, s)
		directives += buildListenDirective(port, s.ProxyProtocol, ipv4)

		if !s.DisableIPV6 {
			directives += buildListenDirective(port, s.ProxyProtocol, ipv6)
		}
	}

	return directives
}

func getDefaultPort(listenerType Protocol) string {
	if listenerType == http {
		return "80"
	} else if listenerType == https {
		return "443 ssl"
	}
	return ""
}

func getCustomPort(listenerType Protocol, s Server) string {
	if listenerType == http {
		return strconv.Itoa(s.HTTPPort)
	} else if listenerType == https {
		return strconv.Itoa(s.HTTPSPort) + " ssl"
	}
	return ""
}

func buildListenDirective(port string, proxyProtocol bool, listenType ListenType) string {
	base := "listen"
	var directive string

	if listenType == ipv6 {
		directive = base + " [::]:" + port
	} else {
		directive = base + " " + port
	}

	if proxyProtocol {
		directive += " proxy_protocol"
	}

	directive += ";\n"
	return directive
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

package version2

import (
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/commonhelpers"
)

type protocol int

const (
	http protocol = iota
	https
)

type listenerType int

const (
	ipv4 listenerType = iota
	ipv6
)

const spacing = "    "

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

func makeListener(listenerType protocol, s Server) string {
	var directives string

	if !s.CustomListeners {
		directives += buildDefaultListenerDirectives(listenerType, s)
	} else {
		directives += buildCustomListenerDirectives(listenerType, s)
	}

	return directives
}

func buildDefaultListenerDirectives(listenerType protocol, s Server) string {
	var directives string
	port := getDefaultPort(listenerType)

	directives += buildListenDirective(port, s.ProxyProtocol, ipv4)

	if !s.DisableIPV6 {
		directives += spacing
		directives += buildListenDirective(port, s.ProxyProtocol, ipv6)
	}

	return directives
}

func buildCustomListenerDirectives(listenerType protocol, s Server) string {
	var directives string

	if (listenerType == http && s.HTTPPort > 0) || (listenerType == https && s.HTTPSPort > 0) {
		port := getCustomPort(listenerType, s)
		directives += buildListenDirective(port, s.ProxyProtocol, ipv4)

		if !s.DisableIPV6 {
			directives += spacing
			directives += buildListenDirective(port, s.ProxyProtocol, ipv6)
		}
	}

	return directives
}

func getDefaultPort(listenerType protocol) string {
	if listenerType == http {
		return "80"
	}
	return "443 ssl"
}

func getCustomPort(listenerType protocol, s Server) string {
	if listenerType == http {
		return strconv.Itoa(s.HTTPPort)
	}
	return strconv.Itoa(s.HTTPSPort) + " ssl"
}

func buildListenDirective(port string, proxyProtocol bool, listenType listenerType) string {
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

func makeHeaderQueryValue(apiKey APIKey) string {
	var parts []string

	for _, header := range apiKey.Header {
		nginxHeader := strings.ReplaceAll(header, "-", "_")
		nginxHeader = strings.ToLower(nginxHeader)

		parts = append(parts, fmt.Sprintf("${http_%s}", nginxHeader))
	}

	for _, query := range apiKey.Query {
		parts = append(parts, fmt.Sprintf("${arg_%s}", query))
	}

	return fmt.Sprintf("\"%s\"", strings.Join(parts, ""))
}

var helperFunctions = template.FuncMap{
	"headerListToCIMap":    headerListToCIMap,
	"hasCIKey":             hasCIKey,
	"contains":             strings.Contains,
	"hasPrefix":            strings.HasPrefix,
	"hasSuffix":            strings.HasSuffix,
	"toLower":              strings.ToLower,
	"toUpper":              strings.ToUpper,
	"replaceAll":           strings.ReplaceAll,
	"makeHTTPListener":     makeHTTPListener,
	"makeHTTPSListener":    makeHTTPSListener,
	"makeSecretPath":       commonhelpers.MakeSecretPath,
	"makeHeaderQueryValue": makeHeaderQueryValue,
}

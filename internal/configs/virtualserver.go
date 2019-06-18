package configs

import (
	"fmt"
	"strings"

	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	api_v1 "k8s.io/api/core/v1"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
)

const nginx502Server = "unix:/var/run/nginx-502-server.sock"

// VirtualServerEx holds a VirtualServer along with the resources that are referenced in this VirtualServer.
type VirtualServerEx struct {
	VirtualServer       *conf_v1alpha1.VirtualServer
	Endpoints           map[string][]string
	TLSSecret           *api_v1.Secret
	VirtualServerRoutes []*conf_v1alpha1.VirtualServerRoute
}

func (vsx *VirtualServerEx) String() string {
	if vsx == nil {
		return "<nil>"
	}

	if vsx.VirtualServer == nil {
		return "VirtualServerEx has no VirtualServer"
	}

	return fmt.Sprintf("%s/%s", vsx.VirtualServer.Namespace, vsx.VirtualServer.Name)
}

// GenerateEndpointsKey generates a key for the Endpoints map in VirtualServerEx.
func GenerateEndpointsKey(serviceNamespace string, serviceName string, port uint16) string {
	return fmt.Sprintf("%s/%s:%d", serviceNamespace, serviceName, port)
}

type upstreamNamer struct {
	prefix string
}

func newUpstreamNamerForVirtualServer(virtualServer *conf_v1alpha1.VirtualServer) *upstreamNamer {
	return &upstreamNamer{
		prefix: fmt.Sprintf("vs_%s_%s", virtualServer.Namespace, virtualServer.Name),
	}
}

func newUpstreamNamerForVirtualServerRoute(virtualServer *conf_v1alpha1.VirtualServer, virtualServerRoute *conf_v1alpha1.VirtualServerRoute) *upstreamNamer {
	return &upstreamNamer{
		prefix: fmt.Sprintf("vs_%s_%s_vsr_%s_%s", virtualServer.Namespace, virtualServer.Name, virtualServerRoute.Namespace, virtualServerRoute.Name),
	}
}

func (namer *upstreamNamer) GetNameForUpstream(upstream string) string {
	return fmt.Sprintf("%s_%s", namer.prefix, upstream)
}

type variableNamer struct {
	safeNsName string
}

func newVariableNamer(virtualServer *conf_v1alpha1.VirtualServer) *variableNamer {
	safeNsName := strings.ReplaceAll(fmt.Sprintf("%s_%s", virtualServer.Namespace, virtualServer.Name), "-", "_")
	return &variableNamer{
		safeNsName: safeNsName,
	}
}

func (namer *variableNamer) GetNameForSplitClientVariable(index int) string {
	return fmt.Sprintf("$vs_%s_splits_%d", namer.safeNsName, index)
}

func (namer *variableNamer) GetNameForVariableForRulesRouteMap(rulesIndex int, matchIndex int, conditionIndex int) string {
	return fmt.Sprintf("$vs_%s_rules_%d_match_%d_cond_%d", namer.safeNsName, rulesIndex, matchIndex, conditionIndex)
}

func (namer *variableNamer) GetNameForVariableForRulesRouteMainMap(rulesIndex int) string {
	return fmt.Sprintf("$vs_%s_rules_%d", namer.safeNsName, rulesIndex)
}

func generateVirtualServerConfig(virtualServerEx *VirtualServerEx, tlsPemFileName string, baseCfgParams *ConfigParams, isPlus bool) version2.VirtualServerConfig {
	ssl := generateSSLConfig(virtualServerEx.VirtualServer.Spec.TLS, tlsPemFileName, baseCfgParams)

	virtualServerUpstreamNamer := newUpstreamNamerForVirtualServer(virtualServerEx.VirtualServer)

	var upstreams []version2.Upstream

	// generate upstreams for VirtualServer
	for _, u := range virtualServerEx.VirtualServer.Spec.Upstreams {
		upstreamName := virtualServerUpstreamNamer.GetNameForUpstream(u.Name)
		endpointsKey := GenerateEndpointsKey(virtualServerEx.VirtualServer.Namespace, u.Service, u.Port)
		ups := generateUpstream(upstreamName, u.LBMethod, virtualServerEx.Endpoints[endpointsKey], isPlus, baseCfgParams)
		upstreams = append(upstreams, ups)
	}
	// generate upstreams for each VirtualServerRoute
	for _, vsr := range virtualServerEx.VirtualServerRoutes {
		upstreamNamer := newUpstreamNamerForVirtualServerRoute(virtualServerEx.VirtualServer, vsr)
		for _, u := range vsr.Spec.Upstreams {
			upstreamName := upstreamNamer.GetNameForUpstream(u.Name)
			endpointsKey := GenerateEndpointsKey(vsr.Namespace, u.Service, u.Port)
			ups := generateUpstream(upstreamName, u.LBMethod, virtualServerEx.Endpoints[endpointsKey], isPlus, baseCfgParams)
			upstreams = append(upstreams, ups)
		}
	}

	var locations []version2.Location
	var internalRedirectLocations []version2.InternalRedirectLocation
	var splitClients []version2.SplitClient
	var maps []version2.Map

	rulesRoutes := 0

	variableNamer := newVariableNamer(virtualServerEx.VirtualServer)

	// generates config for VirtualServer routes
	for _, r := range virtualServerEx.VirtualServer.Spec.Routes {
		// ignore routes that reference VirtualServerRoute
		if r.Route != "" {
			continue
		}

		if len(r.Splits) > 0 {
			splitCfg := generateSplitRouteConfig(r, virtualServerUpstreamNamer, variableNamer, len(splitClients), baseCfgParams)

			splitClients = append(splitClients, splitCfg.SplitClient)
			locations = append(locations, splitCfg.Locations...)
			internalRedirectLocations = append(internalRedirectLocations, splitCfg.InternalRedirectLocation)
		} else if r.Rules != nil {
			rulesRouteCfg := generateRulesRouteConfig(r, virtualServerUpstreamNamer, variableNamer, rulesRoutes, baseCfgParams)

			maps = append(maps, rulesRouteCfg.Maps...)
			locations = append(locations, rulesRouteCfg.Locations...)
			internalRedirectLocations = append(internalRedirectLocations, rulesRouteCfg.InternalRedirectLocation)

			rulesRoutes++
		} else {
			upstreamName := virtualServerUpstreamNamer.GetNameForUpstream(r.Upstream)
			loc := generateLocation(r.Path, upstreamName, baseCfgParams)
			locations = append(locations, loc)
		}
	}

	// generate config for subroutes of each VirtualServerRoute
	for _, vsr := range virtualServerEx.VirtualServerRoutes {
		upstreamNamer := newUpstreamNamerForVirtualServerRoute(virtualServerEx.VirtualServer, vsr)
		for _, r := range vsr.Spec.Subroutes {
			if len(r.Splits) > 0 {
				splitCfg := generateSplitRouteConfig(r, upstreamNamer, variableNamer, len(splitClients), baseCfgParams)

				splitClients = append(splitClients, splitCfg.SplitClient)
				locations = append(locations, splitCfg.Locations...)
				internalRedirectLocations = append(internalRedirectLocations, splitCfg.InternalRedirectLocation)
			} else if r.Rules != nil {
				rulesRouteCfg := generateRulesRouteConfig(r, upstreamNamer, variableNamer, rulesRoutes, baseCfgParams)

				maps = append(maps, rulesRouteCfg.Maps...)
				locations = append(locations, rulesRouteCfg.Locations...)
				internalRedirectLocations = append(internalRedirectLocations, rulesRouteCfg.InternalRedirectLocation)

				rulesRoutes++
			} else {
				upstreamName := upstreamNamer.GetNameForUpstream(r.Upstream)
				loc := generateLocation(r.Path, upstreamName, baseCfgParams)
				locations = append(locations, loc)
			}
		}
	}

	keepalive := ""
	if baseCfgParams.Keepalive > 0 {
		keepalive = fmt.Sprint(baseCfgParams.Keepalive)
	}

	return version2.VirtualServerConfig{
		Upstreams:    upstreams,
		SplitClients: splitClients,
		Maps:         maps,
		Server: version2.Server{
			ServerName:                            virtualServerEx.VirtualServer.Spec.Host,
			ProxyProtocol:                         baseCfgParams.ProxyProtocol,
			SSL:                                   ssl,
			RedirectToHTTPSBasedOnXForwarderProto: baseCfgParams.RedirectToHTTPS,
			ServerTokens:                          baseCfgParams.ServerTokens,
			SetRealIPFrom:                         baseCfgParams.SetRealIPFrom,
			RealIPHeader:                          baseCfgParams.RealIPHeader,
			RealIPRecursive:                       baseCfgParams.RealIPRecursive,
			Snippets:                              baseCfgParams.ServerSnippets,
			InternalRedirectLocations:             internalRedirectLocations,
			Locations:                             locations,
		},
		Keepalive: keepalive,
	}
}

func generateUpstream(upstreamName string, lBMethod string, endpoints []string, isPlus bool, cfgParams *ConfigParams) version2.Upstream {
	var upsServers []version2.UpstreamServer

	for _, e := range endpoints {
		s := version2.UpstreamServer{
			Address:     e,
			MaxFails:    cfgParams.MaxFails,
			FailTimeout: cfgParams.FailTimeout,
		}
		upsServers = append(upsServers, s)
	}

	if !isPlus && len(upsServers) == 0 {
		s := version2.UpstreamServer{
			Address:     nginx502Server,
			MaxFails:    cfgParams.MaxFails,
			FailTimeout: cfgParams.FailTimeout,
		}
		upsServers = append(upsServers, s)
	}

	ups := version2.Upstream{
		Name:     upstreamName,
		Servers:  upsServers,
		LBMethod: generateLBMethod(lBMethod, cfgParams.LBMethod),
	}

	return ups
}

func generateLBMethod(method string, defaultMethod string) string {
	if method == "" {
		return defaultMethod
	} else if method == "round_robin" {
		return ""
	}

	return method
}

func generateLocation(path string, upstreamName string, cfgParams *ConfigParams) version2.Location {
	loc := version2.Location{
		Path:                 path,
		Snippets:             cfgParams.LocationSnippets,
		ProxyConnectTimeout:  cfgParams.ProxyConnectTimeout,
		ProxyReadTimeout:     cfgParams.ProxyReadTimeout,
		ClientMaxBodySize:    cfgParams.ClientMaxBodySize,
		ProxyMaxTempFileSize: cfgParams.ProxyMaxTempFileSize,
		ProxyBuffering:       cfgParams.ProxyBuffering,
		ProxyBuffers:         cfgParams.ProxyBuffers,
		ProxyBufferSize:      cfgParams.ProxyBufferSize,
		ProxyPass:            fmt.Sprintf("http://%v", upstreamName),
	}
	return loc
}

type splitRouteCfg struct {
	SplitClient              version2.SplitClient
	Locations                []version2.Location
	InternalRedirectLocation version2.InternalRedirectLocation
}

func generateSplitRouteConfig(route conf_v1alpha1.Route, upstreamNamer *upstreamNamer, variableNamer *variableNamer, index int, cfgParams *ConfigParams) splitRouteCfg {
	splitClientVarName := variableNamer.GetNameForSplitClientVariable(index)

	// Generate a SplitClient
	var distributions []version2.Distribution

	for i, s := range route.Splits {
		d := version2.Distribution{
			Weight: fmt.Sprintf("%d%%", s.Weight),
			Value:  fmt.Sprintf("@splits_%d_split_%d", index, i),
		}
		distributions = append(distributions, d)
	}

	splitClient := version2.SplitClient{
		Source:        "$request_id",
		Variable:      splitClientVarName,
		Distributions: distributions,
	}

	// Generate locations
	var locations []version2.Location

	for i, s := range route.Splits {
		path := fmt.Sprintf("@splits_%d_split_%d", index, i)
		upstreamName := upstreamNamer.GetNameForUpstream(s.Upstream)
		loc := generateLocation(path, upstreamName, cfgParams)
		locations = append(locations, loc)
	}

	// Generate an InternalRedirectLocation
	irl := version2.InternalRedirectLocation{
		Path:        route.Path,
		Destination: splitClientVarName,
	}

	return splitRouteCfg{
		SplitClient:              splitClient,
		Locations:                locations,
		InternalRedirectLocation: irl,
	}
}

type rulesRouteCfg struct {
	Maps                     []version2.Map
	Locations                []version2.Location
	InternalRedirectLocation version2.InternalRedirectLocation
}

func generateRulesRouteConfig(route conf_v1alpha1.Route, upstreamNamer *upstreamNamer, variableNamer *variableNamer, index int, cfgParams *ConfigParams) rulesRouteCfg {
	// Generate maps
	var maps []version2.Map

	for i, m := range route.Rules.Matches {
		for j, c := range route.Rules.Conditions {
			source := getNameForSourceForRulesRouteMapFromCondition(c)
			variable := variableNamer.GetNameForVariableForRulesRouteMap(index, i, j)
			successfulResult := "1"
			if j < len(m.Values)-1 {
				successfulResult = variableNamer.GetNameForVariableForRulesRouteMap(index, i, j+1)
			}

			params := generateParametersForRulesRouteMap(m.Values[j], successfulResult)

			matchMap := version2.Map{
				Source:     source,
				Variable:   variable,
				Parameters: params,
			}
			maps = append(maps, matchMap)
		}
	}

	// Generate the main map
	source := ""
	var params []version2.Parameter
	for i := range route.Rules.Matches {
		source += variableNamer.GetNameForVariableForRulesRouteMap(index, i, 0)

		p := version2.Parameter{
			Value:  fmt.Sprintf("~^%s1", strings.Repeat("0", i)),
			Result: fmt.Sprintf("@rules_%d_match_%d", index, i),
		}
		params = append(params, p)
	}

	defaultParam := version2.Parameter{
		Value:  "default",
		Result: fmt.Sprintf("@rules_%d_default", index),
	}
	params = append(params, defaultParam)

	variable := variableNamer.GetNameForVariableForRulesRouteMainMap(index)

	mainMap := version2.Map{
		Source:     source,
		Variable:   variable,
		Parameters: params,
	}
	maps = append(maps, mainMap)

	// Generate locations for each match
	var locations []version2.Location

	for i, m := range route.Rules.Matches {
		path := fmt.Sprintf("@rules_%d_match_%d", index, i)
		upstreamName := upstreamNamer.GetNameForUpstream(m.Upstream)
		loc := generateLocation(path, upstreamName, cfgParams)
		locations = append(locations, loc)
	}

	// Generate defaultUpsteam location
	path := fmt.Sprintf("@rules_%d_default", index)
	upstreamName := upstreamNamer.GetNameForUpstream(route.Rules.DefaultUpstream)
	loc := generateLocation(path, upstreamName, cfgParams)
	locations = append(locations, loc)

	// Generate an InternalRedirectLocation to the location defined by the main map variable
	irl := version2.InternalRedirectLocation{
		Path:        route.Path,
		Destination: variable,
	}

	return rulesRouteCfg{
		Maps:                     maps,
		Locations:                locations,
		InternalRedirectLocation: irl,
	}
}

var specialMapParameters = map[string]bool{
	"default":   true,
	"hostnames": true,
	"include":   true,
	"volatile":  true,
}

func generateValueForRulesRouteMap(matchedValue string) (value string, isNegative bool) {
	if len(matchedValue) == 0 {
		return `""`, false
	}

	if matchedValue[0] == '!' {
		isNegative = true
		matchedValue = matchedValue[1:]
	}

	if _, exists := specialMapParameters[matchedValue]; exists {
		return `\` + matchedValue, isNegative
	}

	return fmt.Sprintf(`"%s"`, matchedValue), isNegative
}

func generateParametersForRulesRouteMap(matchedValue string, successfulResult string) []version2.Parameter {
	value, isNegative := generateValueForRulesRouteMap(matchedValue)

	valueResult := successfulResult
	defaultResult := "0"
	if isNegative {
		valueResult = "0"
		defaultResult = successfulResult
	}

	params := []version2.Parameter{
		{
			Value:  value,
			Result: valueResult,
		},
		{
			Value:  "default",
			Result: defaultResult,
		},
	}

	return params
}

func getNameForSourceForRulesRouteMapFromCondition(condition conf_v1alpha1.Condition) string {
	if condition.Header != "" {
		return fmt.Sprintf("$http_%s", strings.ReplaceAll(condition.Header, "-", "_"))
	}

	if condition.Cookie != "" {
		return fmt.Sprintf("$cookie_%s", condition.Cookie)
	}

	if condition.Argument != "" {
		return fmt.Sprintf("$arg_%s", condition.Argument)
	}

	return condition.Variable
}

func generateSSLConfig(tls *conf_v1alpha1.TLS, tlsPemFileName string, cfgParams *ConfigParams) *version2.SSL {
	if tls == nil {
		return nil
	}

	if tls.Secret == "" {
		return nil
	}

	var name string
	var ciphers string

	if tlsPemFileName != "" {
		name = tlsPemFileName
	} else {
		name = pemFileNameForMissingTLSSecret
		ciphers = "NULL"
	}

	ssl := version2.SSL{
		HTTP2:           cfgParams.HTTP2,
		Certificate:     name,
		CertificateKey:  name,
		Ciphers:         ciphers,
		RedirectToHTTPS: cfgParams.SSLRedirect,
	}

	return &ssl
}

func createUpstreamServersForPlus(virtualServerEx *VirtualServerEx) map[string][]string {
	upstreamEndpoints := make(map[string][]string)

	virtualServerUpstreamNamer := newUpstreamNamerForVirtualServer(virtualServerEx.VirtualServer)

	for _, u := range virtualServerEx.VirtualServer.Spec.Upstreams {
		endpointsKey := GenerateEndpointsKey(virtualServerEx.VirtualServer.Namespace, u.Service, u.Port)
		name := virtualServerUpstreamNamer.GetNameForUpstream(u.Name)

		upstreamEndpoints[name] = virtualServerEx.Endpoints[endpointsKey]
	}

	for _, vsr := range virtualServerEx.VirtualServerRoutes {
		upstreamNamer := newUpstreamNamerForVirtualServerRoute(virtualServerEx.VirtualServer, vsr)
		for _, u := range vsr.Spec.Upstreams {
			endpointsKey := GenerateEndpointsKey(vsr.Namespace, u.Service, u.Port)
			name := upstreamNamer.GetNameForUpstream(u.Name)

			upstreamEndpoints[name] = virtualServerEx.Endpoints[endpointsKey]
		}
	}

	return upstreamEndpoints
}

func createUpstreamServersConfig(baseCfgParams *ConfigParams) nginx.ServerConfig {
	cfg := nginx.ServerConfig{
		MaxFails:    baseCfgParams.MaxFails,
		FailTimeout: baseCfgParams.FailTimeout,
		SlowStart:   baseCfgParams.SlowStart,
	}

	return cfg
}

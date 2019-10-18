package configs

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
)

const nginx502Server = "unix:/var/lib/nginx-502-server.sock"

var incompatibleLBMethodsForSlowStart = map[string]bool{
	"random":                          true,
	"ip_hash":                         true,
	"random two":                      true,
	"random two least_conn":           true,
	"random two least_time=header":    true,
	"random two least_time=last_byte": true,
}

// VirtualServerEx holds a VirtualServer along with the resources that are referenced in this VirtualServer.
type VirtualServerEx struct {
	VirtualServer       *conf_v1alpha1.VirtualServer
	Endpoints           map[string][]string
	TLSSecret           *api_v1.Secret
	VirtualServerRoutes []*conf_v1alpha1.VirtualServerRoute
	ExternalNameSvcs    map[string]bool
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
func GenerateEndpointsKey(serviceNamespace string, serviceName string, subselector map[string]string, port uint16) string {
	if len(subselector) > 0 {
		return fmt.Sprintf("%s/%s_%s:%d", serviceNamespace, serviceName, labels.Set(subselector).String(), port)
	}
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

func newHealthCheckWithDefaults(upstream conf_v1alpha1.Upstream, upstreamName string, cfgParams *ConfigParams) *version2.HealthCheck {
	return &version2.HealthCheck{
		Name:                upstreamName,
		URI:                 "/",
		Interval:            "5s",
		Jitter:              "0s",
		Fails:               1,
		Passes:              1,
		Port:                int(upstream.Port),
		ProxyPass:           fmt.Sprintf("%v://%v", generateProxyPassProtocol(upstream.TLS.Enable), upstreamName),
		ProxyConnectTimeout: generateString(upstream.ProxyConnectTimeout, cfgParams.ProxyConnectTimeout),
		ProxyReadTimeout:    generateString(upstream.ProxyReadTimeout, cfgParams.ProxyReadTimeout),
		ProxySendTimeout:    generateString(upstream.ProxySendTimeout, cfgParams.ProxySendTimeout),
		Headers:             make(map[string]string),
	}
}

// VirtualServerConfigurator generates a VirtualServer configuration
type virtualServerConfigurator struct {
	cfgParams            *ConfigParams
	isPlus               bool
	isResolverConfigured bool
	warnings             Warnings
}

func (vsc *virtualServerConfigurator) addWarningf(obj runtime.Object, msgFmt string, args ...interface{}) {
	vsc.warnings[obj] = append(vsc.warnings[obj], fmt.Sprintf(msgFmt, args...))
}

func (vsc *virtualServerConfigurator) clearWarnings() {
	vsc.warnings = make(map[runtime.Object][]string)
}

// newVirtualServerConfigurator creates a new VirtualServerConfigurator
func newVirtualServerConfigurator(cfgParams *ConfigParams, isPlus bool, isResolverConfigured bool) *virtualServerConfigurator {
	return &virtualServerConfigurator{
		cfgParams:            cfgParams,
		isPlus:               isPlus,
		isResolverConfigured: isResolverConfigured,
		warnings:             make(map[runtime.Object][]string),
	}
}

func (vsc *virtualServerConfigurator) generateEndpointsForUpstream(owner runtime.Object, namespace string, upstream conf_v1alpha1.Upstream, virtualServerEx *VirtualServerEx) []string {
	endpointsKey := GenerateEndpointsKey(namespace, upstream.Service, upstream.Subselector, upstream.Port)
	externalNameSvcKey := GenerateExternalNameSvcKey(namespace, upstream.Service)
	endpoints := virtualServerEx.Endpoints[endpointsKey]
	if !vsc.isPlus && len(endpoints) == 0 {
		return []string{nginx502Server}
	}

	_, isExternalNameSvc := virtualServerEx.ExternalNameSvcs[externalNameSvcKey]
	if isExternalNameSvc && !vsc.isResolverConfigured {
		msgFmt := "Type ExternalName service %v in upstream %v will be ignored. To use ExternaName services, a resolver must be configured in the ConfigMap"
		vsc.addWarningf(owner, msgFmt, upstream.Service, upstream.Name)
		endpoints = []string{}
	}

	return endpoints
}

// GenerateVirtualServerConfig generates a full configuration for a VirtualServer
func (vsc *virtualServerConfigurator) GenerateVirtualServerConfig(virtualServerEx *VirtualServerEx, tlsPemFileName string) (version2.VirtualServerConfig, Warnings) {
	vsc.clearWarnings()
	ssl := generateSSLConfig(virtualServerEx.VirtualServer.Spec.TLS, tlsPemFileName, vsc.cfgParams)

	// crUpstreams maps an UpstreamName to its conf_v1alpha1.Upstream as they are generated
	// necessary for generateLocation to know what Upstream each Location references
	crUpstreams := make(map[string]conf_v1alpha1.Upstream)

	virtualServerUpstreamNamer := newUpstreamNamerForVirtualServer(virtualServerEx.VirtualServer)

	var upstreams []version2.Upstream
	var statusMatches []version2.StatusMatch
	var healthChecks []version2.HealthCheck

	// generate upstreams for VirtualServer
	for _, u := range virtualServerEx.VirtualServer.Spec.Upstreams {
		upstreamName := virtualServerUpstreamNamer.GetNameForUpstream(u.Name)
		upstreamNamespace := virtualServerEx.VirtualServer.Namespace
		endpoints := vsc.generateEndpointsForUpstream(virtualServerEx.VirtualServer, upstreamNamespace, u, virtualServerEx)

		// isExternalNameSvc is always false for OSS
		_, isExternalNameSvc := virtualServerEx.ExternalNameSvcs[GenerateExternalNameSvcKey(upstreamNamespace, u.Service)]
		ups := vsc.generateUpstream(virtualServerEx.VirtualServer, upstreamName, u, isExternalNameSvc, endpoints)
		upstreams = append(upstreams, ups)
		crUpstreams[upstreamName] = u

		if hc := generateHealthCheck(u, upstreamName, vsc.cfgParams); hc != nil {
			healthChecks = append(healthChecks, *hc)
			if u.HealthCheck.StatusMatch != "" {
				statusMatches = append(statusMatches, generateUpstreamStatusMatch(upstreamName, u.HealthCheck.StatusMatch))
			}
		}
	}
	// generate upstreams for each VirtualServerRoute
	for _, vsr := range virtualServerEx.VirtualServerRoutes {
		upstreamNamer := newUpstreamNamerForVirtualServerRoute(virtualServerEx.VirtualServer, vsr)
		for _, u := range vsr.Spec.Upstreams {
			upstreamName := upstreamNamer.GetNameForUpstream(u.Name)
			upstreamNamespace := vsr.Namespace
			endpoints := vsc.generateEndpointsForUpstream(vsr, upstreamNamespace, u, virtualServerEx)

			// isExternalNameSvc is always false for OSS
			_, isExternalNameSvc := virtualServerEx.ExternalNameSvcs[GenerateExternalNameSvcKey(upstreamNamespace, u.Service)]
			ups := vsc.generateUpstream(vsr, upstreamName, u, isExternalNameSvc, endpoints)
			upstreams = append(upstreams, ups)
			crUpstreams[upstreamName] = u

			if hc := generateHealthCheck(u, upstreamName, vsc.cfgParams); hc != nil {
				healthChecks = append(healthChecks, *hc)
				if u.HealthCheck.StatusMatch != "" {
					statusMatches = append(statusMatches, generateUpstreamStatusMatch(upstreamName, u.HealthCheck.StatusMatch))
				}
			}
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
			splitCfg := generateSplitRouteConfig(r, virtualServerUpstreamNamer, crUpstreams, variableNamer, len(splitClients), vsc.cfgParams)

			splitClients = append(splitClients, splitCfg.SplitClient)
			locations = append(locations, splitCfg.Locations...)
			internalRedirectLocations = append(internalRedirectLocations, splitCfg.InternalRedirectLocation)
		} else if r.Rules != nil {
			rulesRouteCfg := generateRulesRouteConfig(r, virtualServerUpstreamNamer, crUpstreams, variableNamer, rulesRoutes, vsc.cfgParams)

			maps = append(maps, rulesRouteCfg.Maps...)
			locations = append(locations, rulesRouteCfg.Locations...)
			internalRedirectLocations = append(internalRedirectLocations, rulesRouteCfg.InternalRedirectLocation)

			rulesRoutes++
		} else {
			upstreamName := virtualServerUpstreamNamer.GetNameForUpstream(r.Upstream)
			upstream := crUpstreams[upstreamName]
			loc := generateLocation(r.Path, upstreamName, upstream, vsc.cfgParams)
			locations = append(locations, loc)
		}

	}

	// generate config for subroutes of each VirtualServerRoute
	for _, vsr := range virtualServerEx.VirtualServerRoutes {
		upstreamNamer := newUpstreamNamerForVirtualServerRoute(virtualServerEx.VirtualServer, vsr)
		for _, r := range vsr.Spec.Subroutes {
			if len(r.Splits) > 0 {
				splitCfg := generateSplitRouteConfig(r, upstreamNamer, crUpstreams, variableNamer, len(splitClients), vsc.cfgParams)

				splitClients = append(splitClients, splitCfg.SplitClient)
				locations = append(locations, splitCfg.Locations...)
				internalRedirectLocations = append(internalRedirectLocations, splitCfg.InternalRedirectLocation)
			} else if r.Rules != nil {
				rulesRouteCfg := generateRulesRouteConfig(r, upstreamNamer, crUpstreams, variableNamer, rulesRoutes, vsc.cfgParams)

				maps = append(maps, rulesRouteCfg.Maps...)
				locations = append(locations, rulesRouteCfg.Locations...)
				internalRedirectLocations = append(internalRedirectLocations, rulesRouteCfg.InternalRedirectLocation)

				rulesRoutes++
			} else {
				upstreamName := upstreamNamer.GetNameForUpstream(r.Upstream)
				upstream := crUpstreams[upstreamName]
				loc := generateLocation(r.Path, upstreamName, upstream, vsc.cfgParams)
				locations = append(locations, loc)
			}
		}
	}

	vscfg := version2.VirtualServerConfig{
		Upstreams:     upstreams,
		SplitClients:  splitClients,
		Maps:          maps,
		StatusMatches: statusMatches,
		Server: version2.Server{
			ServerName:                            virtualServerEx.VirtualServer.Spec.Host,
			StatusZone:                            virtualServerEx.VirtualServer.Spec.Host,
			ProxyProtocol:                         vsc.cfgParams.ProxyProtocol,
			SSL:                                   ssl,
			RedirectToHTTPSBasedOnXForwarderProto: vsc.cfgParams.RedirectToHTTPS,
			ServerTokens:                          vsc.cfgParams.ServerTokens,
			SetRealIPFrom:                         vsc.cfgParams.SetRealIPFrom,
			RealIPHeader:                          vsc.cfgParams.RealIPHeader,
			RealIPRecursive:                       vsc.cfgParams.RealIPRecursive,
			Snippets:                              vsc.cfgParams.ServerSnippets,
			InternalRedirectLocations:             internalRedirectLocations,
			Locations:                             locations,
			HealthChecks:                          healthChecks,
		},
	}

	return vscfg, vsc.warnings
}

func (vsc *virtualServerConfigurator) generateUpstream(owner runtime.Object, upstreamName string, upstream conf_v1alpha1.Upstream, isExternalNameSvc bool, endpoints []string) version2.Upstream {
	var upsServers []version2.UpstreamServer
	for _, e := range endpoints {
		s := version2.UpstreamServer{
			Address: e,
		}

		upsServers = append(upsServers, s)
	}

	lbMethod := generateLBMethod(upstream.LBMethod, vsc.cfgParams.LBMethod)

	ups := version2.Upstream{
		Name:             upstreamName,
		Servers:          upsServers,
		Resolve:          isExternalNameSvc,
		LBMethod:         lbMethod,
		Keepalive:        generateIntFromPointer(upstream.Keepalive, vsc.cfgParams.Keepalive),
		MaxFails:         generateIntFromPointer(upstream.MaxFails, vsc.cfgParams.MaxFails),
		FailTimeout:      generateString(upstream.FailTimeout, vsc.cfgParams.FailTimeout),
		MaxConns:         generateIntFromPointer(upstream.MaxConns, vsc.cfgParams.MaxConns),
		UpstreamZoneSize: vsc.cfgParams.UpstreamZoneSize,
	}

	if vsc.isPlus {
		ups.SlowStart = vsc.generateSlowStartForPlus(owner, upstream, lbMethod)
		ups.Queue = generateQueueForPlus(upstream.Queue, "60s")
		ups.SessionCookie = generateSessionCookie(upstream.SessionCookie)
	}

	return ups
}

func (vsc *virtualServerConfigurator) generateSlowStartForPlus(owner runtime.Object, upstream conf_v1alpha1.Upstream, lbMethod string) string {
	if upstream.SlowStart == "" {
		return ""
	}

	_, isIncompatible := incompatibleLBMethodsForSlowStart[lbMethod]
	isHash := strings.HasPrefix(lbMethod, "hash")
	if isIncompatible || isHash {
		msgFmt := "Slow start will be disabled for upstream %v because lb method '%v' is incompatible with slow start"
		vsc.addWarningf(owner, msgFmt, upstream.Name, lbMethod)
		return ""
	}

	return upstream.SlowStart
}

func generateHealthCheck(upstream conf_v1alpha1.Upstream, upstreamName string, cfgParams *ConfigParams) *version2.HealthCheck {
	if upstream.HealthCheck == nil || !upstream.HealthCheck.Enable {
		return nil
	}

	hc := newHealthCheckWithDefaults(upstream, upstreamName, cfgParams)

	if upstream.HealthCheck.Path != "" {
		hc.URI = upstream.HealthCheck.Path
	}

	if upstream.HealthCheck.Interval != "" {
		hc.Interval = upstream.HealthCheck.Interval
	}

	if upstream.HealthCheck.Jitter != "" {
		hc.Jitter = upstream.HealthCheck.Jitter
	}

	if upstream.HealthCheck.Fails > 0 {
		hc.Fails = upstream.HealthCheck.Fails
	}

	if upstream.HealthCheck.Passes > 0 {
		hc.Passes = upstream.HealthCheck.Passes
	}

	if upstream.HealthCheck.Port > 0 {
		hc.Port = upstream.HealthCheck.Port
	}

	if upstream.HealthCheck.ConnectTimeout != "" {
		hc.ProxyConnectTimeout = upstream.HealthCheck.ConnectTimeout
	}

	if upstream.HealthCheck.ReadTimeout != "" {
		hc.ProxyReadTimeout = upstream.HealthCheck.ReadTimeout
	}

	if upstream.HealthCheck.SendTimeout != "" {
		hc.ProxySendTimeout = upstream.HealthCheck.SendTimeout
	}

	for _, h := range upstream.HealthCheck.Headers {
		hc.Headers[h.Name] = h.Value
	}

	if upstream.HealthCheck.TLS != nil {
		hc.ProxyPass = fmt.Sprintf("%v://%v", generateProxyPassProtocol(upstream.HealthCheck.TLS.Enable), upstreamName)
	}

	if upstream.HealthCheck.StatusMatch != "" {
		hc.Match = generateStatusMatchName(upstreamName)
	}

	return hc
}

func generateSessionCookie(sc *conf_v1alpha1.SessionCookie) *version2.SessionCookie {
	if sc == nil || !sc.Enable {
		return nil
	}

	return &version2.SessionCookie{
		Enable:   true,
		Name:     sc.Name,
		Path:     sc.Path,
		Expires:  sc.Expires,
		Domain:   sc.Domain,
		HTTPOnly: sc.HTTPOnly,
		Secure:   sc.Secure,
	}
}

func generateStatusMatchName(upstreamName string) string {
	return fmt.Sprintf("%s_match", upstreamName)
}

func generateUpstreamStatusMatch(upstreamName string, status string) version2.StatusMatch {
	return version2.StatusMatch{
		Name: generateStatusMatchName(upstreamName),
		Code: status,
	}
}

// GenerateExternalNameSvcKey returns the key to identify an ExternalName service.
func GenerateExternalNameSvcKey(namespace string, service string) string {
	return fmt.Sprintf("%v/%v", namespace, service)
}

func generateLBMethod(method string, defaultMethod string) string {
	if method == "" {
		return defaultMethod
	} else if method == "round_robin" {
		return ""
	}
	return method
}

func generateIntFromPointer(n *int, defaultN int) int {
	if n == nil {
		return defaultN
	}
	return *n
}

func upstreamHasKeepalive(upstream conf_v1alpha1.Upstream, cfgParams *ConfigParams) bool {
	if upstream.Keepalive != nil {
		return *upstream.Keepalive != 0
	}
	return cfgParams.Keepalive != 0
}

func generateProxyPassProtocol(enableTLS bool) string {
	if enableTLS {
		return "https"
	}
	return "http"
}

func generateString(s string, defaultS string) string {
	if s == "" {
		return defaultS
	}
	return s
}

func generateBuffers(s *conf_v1alpha1.UpstreamBuffers, defaultS string) string {
	if s == nil {
		return defaultS
	}
	return fmt.Sprintf("%v %v", s.Number, s.Size)
}

func generateBool(s *bool, defaultS bool) bool {
	if s != nil {
		return *s
	}
	return defaultS
}

func generateLocation(path string, upstreamName string, upstream conf_v1alpha1.Upstream, cfgParams *ConfigParams) version2.Location {
	return version2.Location{
		Path:                     path,
		Snippets:                 cfgParams.LocationSnippets,
		ProxyConnectTimeout:      generateString(upstream.ProxyConnectTimeout, cfgParams.ProxyConnectTimeout),
		ProxyReadTimeout:         generateString(upstream.ProxyReadTimeout, cfgParams.ProxyReadTimeout),
		ProxySendTimeout:         generateString(upstream.ProxySendTimeout, cfgParams.ProxySendTimeout),
		ClientMaxBodySize:        generateString(upstream.ClientMaxBodySize, cfgParams.ClientMaxBodySize),
		ProxyMaxTempFileSize:     cfgParams.ProxyMaxTempFileSize,
		ProxyBuffering:           generateBool(upstream.ProxyBuffering, cfgParams.ProxyBuffering),
		ProxyBuffers:             generateBuffers(upstream.ProxyBuffers, cfgParams.ProxyBuffers),
		ProxyBufferSize:          generateString(upstream.ProxyBufferSize, cfgParams.ProxyBufferSize),
		ProxyPass:                fmt.Sprintf("%v://%v", generateProxyPassProtocol(upstream.TLS.Enable), upstreamName),
		ProxyNextUpstream:        generateString(upstream.ProxyNextUpstream, "error timeout"),
		ProxyNextUpstreamTimeout: generateString(upstream.ProxyNextUpstreamTimeout, "0s"),
		ProxyNextUpstreamTries:   upstream.ProxyNextUpstreamTries,
		HasKeepalive:             upstreamHasKeepalive(upstream, cfgParams),
	}
}

type splitRouteCfg struct {
	SplitClient              version2.SplitClient
	Locations                []version2.Location
	InternalRedirectLocation version2.InternalRedirectLocation
}

func generateSplitRouteConfig(route conf_v1alpha1.Route, upstreamNamer *upstreamNamer, crUpstreams map[string]conf_v1alpha1.Upstream, variableNamer *variableNamer, index int, cfgParams *ConfigParams) splitRouteCfg {
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
		upstream := crUpstreams[upstreamName]
		loc := generateLocation(path, upstreamName, upstream, cfgParams)
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

func generateRulesRouteConfig(route conf_v1alpha1.Route, upstreamNamer *upstreamNamer, crUpstreams map[string]conf_v1alpha1.Upstream,
	variableNamer *variableNamer, index int, cfgParams *ConfigParams) rulesRouteCfg {
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
		upstream := crUpstreams[upstreamName]
		loc := generateLocation(path, upstreamName, upstream, cfgParams)
		locations = append(locations, loc)
	}

	// Generate defaultUpstream location
	path := fmt.Sprintf("@rules_%d_default", index)
	upstreamName := upstreamNamer.GetNameForUpstream(route.Rules.DefaultUpstream)
	upstream := crUpstreams[upstreamName]
	loc := generateLocation(path, upstreamName, upstream, cfgParams)
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

func createEndpointsFromUpstream(upstream version2.Upstream) []string {
	var endpoints []string

	for _, server := range upstream.Servers {
		endpoints = append(endpoints, server.Address)
	}

	return endpoints
}

func createUpstreamsForPlus(virtualServerEx *VirtualServerEx, baseCfgParams *ConfigParams) []version2.Upstream {
	var upstreams []version2.Upstream

	isPlus := true
	upstreamNamer := newUpstreamNamerForVirtualServer(virtualServerEx.VirtualServer)
	vsc := newVirtualServerConfigurator(baseCfgParams, isPlus, false)

	for _, u := range virtualServerEx.VirtualServer.Spec.Upstreams {
		isExternalNameSvc := virtualServerEx.ExternalNameSvcs[GenerateExternalNameSvcKey(virtualServerEx.VirtualServer.Namespace, u.Service)]
		if isExternalNameSvc {
			glog.V(3).Infof("Service %s is Type ExternalName, skipping NGINX Plus endpoints update via API", u.Service)
			continue
		}

		upstreamName := upstreamNamer.GetNameForUpstream(u.Name)
		upstreamNamespace := virtualServerEx.VirtualServer.Namespace

		endpointsKey := GenerateEndpointsKey(upstreamNamespace, u.Service, u.Subselector, u.Port)
		endpoints := virtualServerEx.Endpoints[endpointsKey]

		ups := vsc.generateUpstream(virtualServerEx.VirtualServer, upstreamName, u, isExternalNameSvc, endpoints)
		upstreams = append(upstreams, ups)
	}

	for _, vsr := range virtualServerEx.VirtualServerRoutes {
		upstreamNamer = newUpstreamNamerForVirtualServerRoute(virtualServerEx.VirtualServer, vsr)
		for _, u := range vsr.Spec.Upstreams {
			isExternalNameSvc := virtualServerEx.ExternalNameSvcs[GenerateExternalNameSvcKey(vsr.Namespace, u.Service)]
			if isExternalNameSvc {
				glog.V(3).Infof("Service %s is Type ExternalName, skipping NGINX Plus endpoints update via API", u.Service)
				continue
			}

			upstreamName := upstreamNamer.GetNameForUpstream(u.Name)
			upstreamNamespace := vsr.Namespace

			endpointsKey := GenerateEndpointsKey(upstreamNamespace, u.Service, u.Subselector, u.Port)
			endpoints := virtualServerEx.Endpoints[endpointsKey]

			ups := vsc.generateUpstream(vsr, upstreamName, u, isExternalNameSvc, endpoints)
			upstreams = append(upstreams, ups)
		}
	}

	return upstreams
}

func createUpstreamServersConfigForPlus(upstream version2.Upstream) nginx.ServerConfig {
	if len(upstream.Servers) == 0 {
		return nginx.ServerConfig{}
	}
	return nginx.ServerConfig{
		MaxFails:    upstream.MaxFails,
		FailTimeout: upstream.FailTimeout,
		MaxConns:    upstream.MaxConns,
		SlowStart:   upstream.SlowStart,
	}
}

func generateQueueForPlus(upstreamQueue *conf_v1alpha1.UpstreamQueue, defaultTimeout string) *version2.Queue {
	if upstreamQueue == nil {
		return nil
	}

	return &version2.Queue{
		Size:    upstreamQueue.Size,
		Timeout: generateString(upstreamQueue.Timeout, defaultTimeout),
	}
}

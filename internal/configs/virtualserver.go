package configs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
)

const nginx502Server = "unix:/var/lib/nginx/nginx-502-server.sock"
const internalLocationPrefix = "internal_location_"
const nginx418Server = "unix:/var/lib/nginx/nginx-418-server.sock"

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
	VirtualServer       *conf_v1.VirtualServer
	Endpoints           map[string][]string
	TLSSecret           *api_v1.Secret
	VirtualServerRoutes []*conf_v1.VirtualServerRoute
	ExternalNameSvcs    map[string]bool
	Policies            map[string]*conf_v1alpha1.Policy
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
	prefix    string
	namespace string
}

func newUpstreamNamerForVirtualServer(virtualServer *conf_v1.VirtualServer) *upstreamNamer {
	return &upstreamNamer{
		prefix:    fmt.Sprintf("vs_%s_%s", virtualServer.Namespace, virtualServer.Name),
		namespace: virtualServer.Namespace,
	}
}

func newUpstreamNamerForVirtualServerRoute(virtualServer *conf_v1.VirtualServer, virtualServerRoute *conf_v1.VirtualServerRoute) *upstreamNamer {
	return &upstreamNamer{
		prefix:    fmt.Sprintf("vs_%s_%s_vsr_%s_%s", virtualServer.Namespace, virtualServer.Name, virtualServerRoute.Namespace, virtualServerRoute.Name),
		namespace: virtualServerRoute.Namespace,
	}
}

func newUpstreamNamerForTransportServer(transportServer *conf_v1alpha1.TransportServer) *upstreamNamer {
	return &upstreamNamer{
		prefix: fmt.Sprintf("ts_%s_%s", transportServer.Namespace, transportServer.Name),
	}
}

func (namer *upstreamNamer) GetNameForUpstreamFromAction(action *conf_v1.Action) string {
	var upstream string
	if action.Proxy != nil && action.Proxy.Upstream != "" {
		upstream = action.Proxy.Upstream
	} else {
		upstream = action.Pass
	}

	return fmt.Sprintf("%s_%s", namer.prefix, upstream)
}

func (namer *upstreamNamer) GetNameForUpstream(upstream string) string {
	return fmt.Sprintf("%s_%s", namer.prefix, upstream)
}

type variableNamer struct {
	safeNsName string
}

func newVariableNamer(virtualServer *conf_v1.VirtualServer) *variableNamer {
	safeNsName := strings.ReplaceAll(fmt.Sprintf("%s_%s", virtualServer.Namespace, virtualServer.Name), "-", "_")
	return &variableNamer{
		safeNsName: safeNsName,
	}
}

func (namer *variableNamer) GetNameForSplitClientVariable(index int) string {
	return fmt.Sprintf("$vs_%s_splits_%d", namer.safeNsName, index)
}

func (namer *variableNamer) GetNameForVariableForMatchesRouteMap(matchesIndex int, matchIndex int, conditionIndex int) string {
	return fmt.Sprintf("$vs_%s_matches_%d_match_%d_cond_%d", namer.safeNsName, matchesIndex, matchIndex, conditionIndex)
}

func (namer *variableNamer) GetNameForVariableForMatchesRouteMainMap(matchesIndex int) string {
	return fmt.Sprintf("$vs_%s_matches_%d", namer.safeNsName, matchesIndex)
}

func newHealthCheckWithDefaults(upstream conf_v1.Upstream, upstreamName string, cfgParams *ConfigParams) *version2.HealthCheck {
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
	isTLSPassthrough     bool
	enableSnippets       bool
	warnings             Warnings
	spiffeCerts          bool
}

func (vsc *virtualServerConfigurator) addWarningf(obj runtime.Object, msgFmt string, args ...interface{}) {
	vsc.warnings[obj] = append(vsc.warnings[obj], fmt.Sprintf(msgFmt, args...))
}

func (vsc *virtualServerConfigurator) clearWarnings() {
	vsc.warnings = make(map[runtime.Object][]string)
}

// newVirtualServerConfigurator creates a new VirtualServerConfigurator
func newVirtualServerConfigurator(cfgParams *ConfigParams, isPlus bool, isResolverConfigured bool, staticParams *StaticConfigParams) *virtualServerConfigurator {
	return &virtualServerConfigurator{
		cfgParams:            cfgParams,
		isPlus:               isPlus,
		isResolverConfigured: isResolverConfigured,
		isTLSPassthrough:     staticParams.TLSPassthrough,
		enableSnippets:       staticParams.EnableSnippets,
		warnings:             make(map[runtime.Object][]string),
		spiffeCerts:          staticParams.SpiffeCerts,
	}
}

func (vsc *virtualServerConfigurator) generateEndpointsForUpstream(owner runtime.Object, namespace string, upstream conf_v1.Upstream, virtualServerEx *VirtualServerEx) []string {
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

	policiesCfg := vsc.generatePolicies(virtualServerEx.VirtualServer, virtualServerEx.VirtualServer.Namespace,
		virtualServerEx.VirtualServer.Spec.Policies, virtualServerEx.Policies)

	// crUpstreams maps an UpstreamName to its conf_v1.Upstream as they are generated
	// necessary for generateLocation to know what Upstream each Location references
	crUpstreams := make(map[string]conf_v1.Upstream)

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

		u.TLS.Enable = isTLSEnabled(u, vsc.spiffeCerts)
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
			u.TLS.Enable = isTLSEnabled(u, vsc.spiffeCerts)
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
	var returnLocations []version2.ReturnLocation
	var splitClients []version2.SplitClient
	var maps []version2.Map
	var errorPageLocations []version2.ErrorPageLocation
	var vsrErrorPagesFromVs = make(map[string][]conf_v1.ErrorPage)
	var vsrLocationSnippetsFromVs = make(map[string]string)
	var vsrErrorPagesRouteIndex = make(map[string]int)
	matchesRoutes := 0

	variableNamer := newVariableNamer(virtualServerEx.VirtualServer)

	// generates config for VirtualServer routes
	for _, r := range virtualServerEx.VirtualServer.Spec.Routes {
		errorPageIndex := len(errorPageLocations)
		errorPageLocations = append(errorPageLocations, generateErrorPageLocations(errorPageIndex, r.ErrorPages)...)

		// ignore routes that reference VirtualServerRoute
		if r.Route != "" {
			// store route location snippet for the referenced VirtualServerRoute in case they don't define their own
			if r.LocationSnippets != "" {
				name := r.Route
				if !strings.Contains(name, "/") {
					name = fmt.Sprintf("%v/%v", virtualServerEx.VirtualServer.Namespace, r.Route)
				}
				vsrLocationSnippetsFromVs[name] = r.LocationSnippets
			}

			// store route error pages and route index for the referenced VirtualServerRoute in case they don't define their own
			if len(r.ErrorPages) > 0 {
				name := r.Route
				if !strings.Contains(name, "/") {
					name = fmt.Sprintf("%v/%v", virtualServerEx.VirtualServer.Namespace, r.Route)
				}
				vsrErrorPagesFromVs[name] = r.ErrorPages
				vsrErrorPagesRouteIndex[name] = errorPageIndex
			}
			continue
		}

		vsLocSnippets := r.LocationSnippets

		if len(r.Matches) > 0 {
			cfg := generateMatchesConfig(r, virtualServerUpstreamNamer, crUpstreams, variableNamer, matchesRoutes, len(splitClients),
				vsc.cfgParams, r.ErrorPages, errorPageIndex, vsLocSnippets, vsc.enableSnippets, len(returnLocations))
			maps = append(maps, cfg.Maps...)
			locations = append(locations, cfg.Locations...)
			internalRedirectLocations = append(internalRedirectLocations, cfg.InternalRedirectLocation)
			returnLocations = append(returnLocations, cfg.ReturnLocations...)
			splitClients = append(splitClients, cfg.SplitClients...)
			matchesRoutes++
		} else if len(r.Splits) > 0 {
			cfg := generateDefaultSplitsConfig(r, virtualServerUpstreamNamer, crUpstreams, variableNamer, len(splitClients),
				vsc.cfgParams, r.ErrorPages, errorPageIndex, r.Path, vsLocSnippets, vsc.enableSnippets, len(returnLocations))

			splitClients = append(splitClients, cfg.SplitClients...)
			locations = append(locations, cfg.Locations...)
			internalRedirectLocations = append(internalRedirectLocations, cfg.InternalRedirectLocation)
			returnLocations = append(returnLocations, cfg.ReturnLocations...)
		} else {
			upstreamName := virtualServerUpstreamNamer.GetNameForUpstreamFromAction(r.Action)
			upstream := crUpstreams[upstreamName]
			proxySSLName := generateProxySSLName(upstream.Service, virtualServerEx.VirtualServer.Namespace)
			loc, returnLoc := generateLocation(r.Path, upstreamName, upstream, r.Action, vsc.cfgParams, r.ErrorPages, false,
				errorPageIndex, proxySSLName, r.Path, vsLocSnippets, vsc.enableSnippets, len(returnLocations))
			locations = append(locations, loc)
			if returnLoc != nil {
				returnLocations = append(returnLocations, *returnLoc)
			}
		}
	}

	// generate config for subroutes of each VirtualServerRoute
	for _, vsr := range virtualServerEx.VirtualServerRoutes {
		upstreamNamer := newUpstreamNamerForVirtualServerRoute(virtualServerEx.VirtualServer, vsr)
		for _, r := range vsr.Spec.Subroutes {
			errorPageIndex := len(errorPageLocations)
			errorPageLocations = append(errorPageLocations, generateErrorPageLocations(errorPageIndex, r.ErrorPages)...)
			errorPages := r.ErrorPages
			vsrNamespaceName := fmt.Sprintf("%v/%v", vsr.Namespace, vsr.Name)
			// use referenced VirtualServer error pages if the route does not define any
			if r.ErrorPages == nil {
				if vsErrorPages, ok := vsrErrorPagesFromVs[vsrNamespaceName]; ok {
					errorPages = vsErrorPages
					errorPageIndex = vsrErrorPagesRouteIndex[vsrNamespaceName]
				}
			}

			locSnippets := r.LocationSnippets
			// use referenced VirtualServer location snippet if the route does not define any
			if r.LocationSnippets == "" {
				locSnippets = vsrLocationSnippetsFromVs[r.Route]
			}

			if len(r.Matches) > 0 {
				cfg := generateMatchesConfig(r, upstreamNamer, crUpstreams, variableNamer, matchesRoutes, len(splitClients),
					vsc.cfgParams, errorPages, errorPageIndex, locSnippets, vsc.enableSnippets, len(returnLocations))
				maps = append(maps, cfg.Maps...)
				locations = append(locations, cfg.Locations...)
				internalRedirectLocations = append(internalRedirectLocations, cfg.InternalRedirectLocation)
				returnLocations = append(returnLocations, cfg.ReturnLocations...)
				splitClients = append(splitClients, cfg.SplitClients...)
				matchesRoutes++
			} else if len(r.Splits) > 0 {
				cfg := generateDefaultSplitsConfig(r, upstreamNamer, crUpstreams, variableNamer, len(splitClients), vsc.cfgParams,
					errorPages, errorPageIndex, r.Path, locSnippets, vsc.enableSnippets, len(returnLocations))
				splitClients = append(splitClients, cfg.SplitClients...)
				locations = append(locations, cfg.Locations...)
				internalRedirectLocations = append(internalRedirectLocations, cfg.InternalRedirectLocation)
				returnLocations = append(returnLocations, cfg.ReturnLocations...)
			} else {
				upstreamName := upstreamNamer.GetNameForUpstreamFromAction(r.Action)
				upstream := crUpstreams[upstreamName]
				proxySSLName := generateProxySSLName(upstream.Service, vsr.Namespace)
				loc, returnLoc := generateLocation(r.Path, upstreamName, upstream, r.Action, vsc.cfgParams, errorPages, false,
					errorPageIndex, proxySSLName, r.Path, locSnippets, vsc.enableSnippets, len(returnLocations))
				locations = append(locations, loc)
				if returnLoc != nil {
					returnLocations = append(returnLocations, *returnLoc)
				}
			}
		}
	}

	ssl := generateSSLConfig(virtualServerEx.VirtualServer.Spec.TLS, tlsPemFileName, vsc.cfgParams)
	tlsRedirectConfig := generateTLSRedirectConfig(virtualServerEx.VirtualServer.Spec.TLS)
	httpSnippets := generateSnippets(vsc.enableSnippets, virtualServerEx.VirtualServer.Spec.HTTPSnippets, []string{""})
	serverSnippets := generateSnippets(vsc.enableSnippets, virtualServerEx.VirtualServer.Spec.ServerSnippets, vsc.cfgParams.ServerSnippets)

	vsCfg := version2.VirtualServerConfig{
		Upstreams:     upstreams,
		SplitClients:  splitClients,
		Maps:          maps,
		StatusMatches: statusMatches,
		HTTPSnippets:  httpSnippets,
		Server: version2.Server{
			ServerName:                virtualServerEx.VirtualServer.Spec.Host,
			StatusZone:                virtualServerEx.VirtualServer.Spec.Host,
			ProxyProtocol:             vsc.cfgParams.ProxyProtocol,
			SSL:                       ssl,
			ServerTokens:              vsc.cfgParams.ServerTokens,
			SetRealIPFrom:             vsc.cfgParams.SetRealIPFrom,
			RealIPHeader:              vsc.cfgParams.RealIPHeader,
			RealIPRecursive:           vsc.cfgParams.RealIPRecursive,
			Snippets:                  serverSnippets,
			InternalRedirectLocations: internalRedirectLocations,
			Locations:                 locations,
			ReturnLocations:           returnLocations,
			HealthChecks:              healthChecks,
			TLSRedirect:               tlsRedirectConfig,
			ErrorPageLocations:        errorPageLocations,
			TLSPassthrough:            vsc.isTLSPassthrough,
			Allow:                     policiesCfg.Allow,
			Deny:                      policiesCfg.Deny,
			PoliciesErrorReturn:       policiesCfg.ErrorReturn,
		},
		SpiffeCerts: vsc.spiffeCerts,
	}

	return vsCfg, vsc.warnings
}

type policiesCfg struct {
	Allow       []string
	Deny        []string
	ErrorReturn *version2.Return
}

func (vsc *virtualServerConfigurator) generatePolicies(owner runtime.Object, ownerNamespace string, policyRefs []conf_v1.PolicyReference,
	policies map[string]*conf_v1alpha1.Policy) policiesCfg {
	var policyErrorReturn *version2.Return
	var allow, deny []string
	var policyError bool

	for _, p := range policyRefs {
		polNamespace := p.Namespace
		if polNamespace == "" {
			polNamespace = ownerNamespace
		}

		key := fmt.Sprintf("%s/%s", polNamespace, p.Name)

		if pol, exists := policies[key]; exists {
			if pol.Spec.AccessControl != nil {
				allow = append(allow, pol.Spec.AccessControl.Allow...)
				deny = append(deny, pol.Spec.AccessControl.Deny...)
			}
		} else {
			vsc.addWarningf(owner, "Policy %s is missing or invalid", key)
			policyError = true
			break
		}
	}
	if policyError {
		allow = []string{}
		deny = []string{}
		policyErrorReturn = &version2.Return{Code: 500}
	} else if len(allow) > 0 && len(deny) > 0 {
		vsc.addWarningf(owner, "AccessControl policy (or policies) with deny rules is overridden by policy (or policies) with allow rules")
	}

	return policiesCfg{
		Allow:       allow,
		Deny:        deny,
		ErrorReturn: policyErrorReturn,
	}
}

func (vsc *virtualServerConfigurator) generateUpstream(owner runtime.Object, upstreamName string, upstream conf_v1.Upstream, isExternalNameSvc bool, endpoints []string) version2.Upstream {
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

func (vsc *virtualServerConfigurator) generateSlowStartForPlus(owner runtime.Object, upstream conf_v1.Upstream, lbMethod string) string {
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

func generateHealthCheck(upstream conf_v1.Upstream, upstreamName string, cfgParams *ConfigParams) *version2.HealthCheck {
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

func generateSessionCookie(sc *conf_v1.SessionCookie) *version2.SessionCookie {
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

func upstreamHasKeepalive(upstream conf_v1.Upstream, cfgParams *ConfigParams) bool {
	if upstream.Keepalive != nil {
		return *upstream.Keepalive != 0
	}
	return cfgParams.Keepalive != 0
}

func generateRewrites(path string, proxy *conf_v1.ActionProxy, internal bool, originalPath string) []string {
	if proxy == nil || proxy.RewritePath == "" {
		return nil
	}

	if originalPath != "" {
		path = originalPath
	}

	isRegex := false
	if strings.HasPrefix(path, "~") {
		isRegex = true
	}

	trimmedPath := strings.TrimPrefix(strings.TrimPrefix(path, "~"), "*")
	trimmedPath = strings.TrimSpace(trimmedPath)

	var rewrites []string

	if internal {
		// For internal locations (splits locations) only, recover the original request_uri.
		rewrites = append(rewrites, "^ $request_uri")
	}

	if isRegex {
		rewrites = append(rewrites, fmt.Sprintf(`"^%v" "%v" break`, trimmedPath, proxy.RewritePath))
	} else if internal {
		rewrites = append(rewrites, fmt.Sprintf(`"^%v(.*)$" "%v$1" break`, trimmedPath, proxy.RewritePath))
	}

	return rewrites
}

func generateProxyPassRewrite(path string, proxy *conf_v1.ActionProxy, internal bool) string {
	if proxy == nil || internal {
		return ""
	}

	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "=") {
		return proxy.RewritePath
	}

	return ""
}

func generateProxyPass(tlsEnabled bool, upstreamName string, internal bool, proxy *conf_v1.ActionProxy) string {
	proxyPass := fmt.Sprintf("%v://%v", generateProxyPassProtocol(tlsEnabled), upstreamName)

	if internal && (proxy == nil || proxy.RewritePath == "") {
		return fmt.Sprintf("%v$request_uri", proxyPass)
	}

	return proxyPass
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

func generateSnippets(enableSnippets bool, s string, defaultS []string) []string {
	if !enableSnippets || s == "" {
		return defaultS
	}
	return strings.Split(s, "\n")
}

func generateBuffers(s *conf_v1.UpstreamBuffers, defaultS string) string {
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

func generatePath(path string) string {
	// Wrap the regular expression (if present) inside double quotes (") to avoid NGINX parsing errors
	if strings.HasPrefix(path, "~*") {
		return fmt.Sprintf(`~* "%v"`, strings.TrimPrefix(strings.TrimPrefix(path, "~*"), " "))
	}
	if strings.HasPrefix(path, "~") {
		return fmt.Sprintf(`~ "%v"`, strings.TrimPrefix(strings.TrimPrefix(path, "~"), " "))
	}

	return path
}

func generateReturnBlock(text string, code int, defaultCode int) *version2.Return {
	returnBlock := &version2.Return{
		Code: defaultCode,
		Text: text,
	}

	if code != 0 {
		returnBlock.Code = code
	}

	return returnBlock
}

func generateLocation(path string, upstreamName string, upstream conf_v1.Upstream, action *conf_v1.Action,
	cfgParams *ConfigParams, errorPages []conf_v1.ErrorPage, internal bool, errPageIndex int, proxySSLName string,
	originalPath string, locSnippets string, enableSnippets bool, retLocIndex int) (version2.Location, *version2.ReturnLocation) {

	locationSnippets := generateSnippets(enableSnippets, locSnippets, cfgParams.LocationSnippets)

	if action.Redirect != nil {
		return generateLocationForRedirect(path, locationSnippets, action.Redirect), nil
	}

	if action.Return != nil {
		return generateLocationForReturn(path, cfgParams.LocationSnippets, action.Return, retLocIndex)
	}

	return generateLocationForProxying(path, upstreamName, upstream, cfgParams, errorPages, internal, errPageIndex, proxySSLName, action.Proxy, originalPath, locationSnippets), nil
}

func generateProxySetHeaders(proxy *conf_v1.ActionProxy) []version2.Header {
	if proxy == nil || proxy.RequestHeaders == nil {
		return nil
	}

	var headers []version2.Header
	for _, h := range proxy.RequestHeaders.Set {
		headers = append(headers, version2.Header{
			Name:  h.Name,
			Value: h.Value,
		})
	}

	return headers
}

func generateProxyPassRequestHeaders(proxy *conf_v1.ActionProxy) bool {
	if proxy == nil || proxy.RequestHeaders == nil {
		return true
	}

	if proxy.RequestHeaders.Pass != nil {
		return *proxy.RequestHeaders.Pass
	}

	return true
}

func generateProxyHideHeaders(proxy *conf_v1.ActionProxy) []string {
	if proxy == nil || proxy.ResponseHeaders == nil {
		return nil
	}

	return proxy.ResponseHeaders.Hide
}

func generateProxyPassHeaders(proxy *conf_v1.ActionProxy) []string {
	if proxy == nil || proxy.ResponseHeaders == nil {
		return nil
	}

	return proxy.ResponseHeaders.Pass
}

func generateProxyIgnoreHeaders(proxy *conf_v1.ActionProxy) string {
	if proxy == nil || proxy.ResponseHeaders == nil {
		return ""
	}

	return strings.Join(proxy.ResponseHeaders.Ignore, " ")
}

func generateProxyAddHeaders(proxy *conf_v1.ActionProxy) []version2.AddHeader {
	if proxy == nil || proxy.ResponseHeaders == nil {
		return nil
	}

	var addHeaders []version2.AddHeader
	for _, h := range proxy.ResponseHeaders.Add {
		addHeaders = append(addHeaders, version2.AddHeader{
			Header: version2.Header{
				Name:  h.Name,
				Value: h.Value,
			},
			Always: h.Always,
		})
	}

	return addHeaders
}

func generateLocationForProxying(path string, upstreamName string, upstream conf_v1.Upstream,
	cfgParams *ConfigParams, errorPages []conf_v1.ErrorPage, internal bool, errPageIndex int,
	proxySSLName string, proxy *conf_v1.ActionProxy, originalPath string, locationSnippets []string) version2.Location {
	return version2.Location{
		Path:                     generatePath(path),
		Internal:                 internal,
		Snippets:                 locationSnippets,
		ProxyConnectTimeout:      generateString(upstream.ProxyConnectTimeout, cfgParams.ProxyConnectTimeout),
		ProxyReadTimeout:         generateString(upstream.ProxyReadTimeout, cfgParams.ProxyReadTimeout),
		ProxySendTimeout:         generateString(upstream.ProxySendTimeout, cfgParams.ProxySendTimeout),
		ClientMaxBodySize:        generateString(upstream.ClientMaxBodySize, cfgParams.ClientMaxBodySize),
		ProxyMaxTempFileSize:     cfgParams.ProxyMaxTempFileSize,
		ProxyBuffering:           generateBool(upstream.ProxyBuffering, cfgParams.ProxyBuffering),
		ProxyBuffers:             generateBuffers(upstream.ProxyBuffers, cfgParams.ProxyBuffers),
		ProxyBufferSize:          generateString(upstream.ProxyBufferSize, cfgParams.ProxyBufferSize),
		ProxyPass:                generateProxyPass(upstream.TLS.Enable, upstreamName, internal, proxy),
		ProxyNextUpstream:        generateString(upstream.ProxyNextUpstream, "error timeout"),
		ProxyNextUpstreamTimeout: generateString(upstream.ProxyNextUpstreamTimeout, "0s"),
		ProxyNextUpstreamTries:   upstream.ProxyNextUpstreamTries,
		ProxyInterceptErrors:     generateProxyInterceptErrors(errorPages),
		ProxyPassRequestHeaders:  generateProxyPassRequestHeaders(proxy),
		ProxySetHeaders:          generateProxySetHeaders(proxy),
		ProxyHideHeaders:         generateProxyHideHeaders(proxy),
		ProxyPassHeaders:         generateProxyPassHeaders(proxy),
		ProxyIgnoreHeaders:       generateProxyIgnoreHeaders(proxy),
		AddHeaders:               generateProxyAddHeaders(proxy),
		ProxyPassRewrite:         generateProxyPassRewrite(path, proxy, internal),
		Rewrites:                 generateRewrites(path, proxy, internal, originalPath),
		HasKeepalive:             upstreamHasKeepalive(upstream, cfgParams),
		ErrorPages:               generateErrorPages(errPageIndex, errorPages),
		ProxySSLName:             proxySSLName,
	}
}

func generateProxyInterceptErrors(errorPages []conf_v1.ErrorPage) bool {
	return len(errorPages) > 0
}

func generateLocationForRedirect(path string, locationSnippets []string, redirect *conf_v1.ActionRedirect) version2.Location {
	code := redirect.Code
	if code == 0 {
		code = 301
	}

	return version2.Location{
		Path:                 path,
		Snippets:             locationSnippets,
		ProxyInterceptErrors: true,
		InternalProxyPass:    fmt.Sprintf("http://%s", nginx418Server),
		ErrorPages: []version2.ErrorPage{
			{
				Name:         redirect.URL,
				Codes:        "418",
				ResponseCode: code,
			},
		},
	}
}

func generateLocationForReturn(path string, locationSnippets []string, actionReturn *conf_v1.ActionReturn,
	retLocIndex int) (version2.Location, *version2.ReturnLocation) {
	defaultType := actionReturn.Type
	if defaultType == "" {
		defaultType = "text/plain"
	}
	code := actionReturn.Code
	if code == 0 {
		code = 200
	}

	retLocName := fmt.Sprintf("@return_%d", retLocIndex)

	return version2.Location{
			Path:                 path,
			Snippets:             locationSnippets,
			ProxyInterceptErrors: true,
			InternalProxyPass:    fmt.Sprintf("http://%s", nginx418Server),
			ErrorPages: []version2.ErrorPage{
				{
					Name:         retLocName,
					Codes:        "418",
					ResponseCode: code,
				},
			},
		},
		&version2.ReturnLocation{
			Name:        retLocName,
			DefaultType: defaultType,
			Return: version2.Return{
				Text: actionReturn.Body,
			},
		}
}

type routingCfg struct {
	Maps                     []version2.Map
	SplitClients             []version2.SplitClient
	Locations                []version2.Location
	InternalRedirectLocation version2.InternalRedirectLocation
	ReturnLocations          []version2.ReturnLocation
}

func generateSplits(splits []conf_v1.Split, upstreamNamer *upstreamNamer, crUpstreams map[string]conf_v1.Upstream,
	variableNamer *variableNamer, scIndex int, cfgParams *ConfigParams, errorPages []conf_v1.ErrorPage, errPageIndex int,
	originalPath string, locSnippets string, enableSnippets bool,
	retLocIndex int) (version2.SplitClient, []version2.Location, []version2.ReturnLocation) {

	var distributions []version2.Distribution

	for i, s := range splits {
		d := version2.Distribution{
			Weight: fmt.Sprintf("%d%%", s.Weight),
			Value:  fmt.Sprintf("/%vsplits_%d_split_%d", internalLocationPrefix, scIndex, i),
		}
		distributions = append(distributions, d)
	}

	splitClient := version2.SplitClient{
		Source:        "$request_id",
		Variable:      variableNamer.GetNameForSplitClientVariable(scIndex),
		Distributions: distributions,
	}

	var locations []version2.Location
	var returnLocations []version2.ReturnLocation

	for i, s := range splits {
		path := fmt.Sprintf("/%vsplits_%d_split_%d", internalLocationPrefix, scIndex, i)
		upstreamName := upstreamNamer.GetNameForUpstreamFromAction(s.Action)
		upstream := crUpstreams[upstreamName]
		proxySSLName := generateProxySSLName(upstream.Service, upstreamNamer.namespace)
		newRetLocIndex := retLocIndex + len(returnLocations)
		loc, returnLoc := generateLocation(path, upstreamName, upstream, s.Action, cfgParams, errorPages, true,
			errPageIndex, proxySSLName, originalPath, locSnippets, enableSnippets, newRetLocIndex)
		locations = append(locations, loc)
		if returnLoc != nil {
			returnLocations = append(returnLocations, *returnLoc)
		}
	}

	return splitClient, locations, returnLocations
}

func generateDefaultSplitsConfig(route conf_v1.Route, upstreamNamer *upstreamNamer, crUpstreams map[string]conf_v1.Upstream,
	variableNamer *variableNamer, scIndex int, cfgParams *ConfigParams, errorPages []conf_v1.ErrorPage, errPageIndex int,
	originalPath string, locSnippets string, enableSnippets bool, retLocIndex int) routingCfg {

	sc, locs, returnLocs := generateSplits(route.Splits, upstreamNamer, crUpstreams, variableNamer, scIndex, cfgParams,
		errorPages, errPageIndex, originalPath, locSnippets, enableSnippets, retLocIndex)

	splitClientVarName := variableNamer.GetNameForSplitClientVariable(scIndex)

	irl := version2.InternalRedirectLocation{
		Path:        route.Path,
		Destination: splitClientVarName,
	}

	return routingCfg{
		SplitClients:             []version2.SplitClient{sc},
		Locations:                locs,
		InternalRedirectLocation: irl,
		ReturnLocations:          returnLocs,
	}
}

func generateMatchesConfig(route conf_v1.Route, upstreamNamer *upstreamNamer, crUpstreams map[string]conf_v1.Upstream,
	variableNamer *variableNamer, index int, scIndex int, cfgParams *ConfigParams, errorPages []conf_v1.ErrorPage,
	errPageIndex int, locSnippets string, enableSnippets bool, retLocIndex int) routingCfg {
	// Generate maps
	var maps []version2.Map

	for i, m := range route.Matches {
		for j, c := range m.Conditions {
			source := getNameForSourceForMatchesRouteMapFromCondition(c)
			variable := variableNamer.GetNameForVariableForMatchesRouteMap(index, i, j)
			successfulResult := "1"
			if j < len(m.Conditions)-1 {
				successfulResult = variableNamer.GetNameForVariableForMatchesRouteMap(index, i, j+1)
			}

			params := generateParametersForMatchesRouteMap(c.Value, successfulResult)

			matchMap := version2.Map{
				Source:     source,
				Variable:   variable,
				Parameters: params,
			}
			maps = append(maps, matchMap)
		}
	}

	scLocalIndex := 0

	// Generate the main map
	source := ""
	var params []version2.Parameter
	for i, m := range route.Matches {
		source += variableNamer.GetNameForVariableForMatchesRouteMap(index, i, 0)

		v := fmt.Sprintf("~^%s1", strings.Repeat("0", i))
		r := fmt.Sprintf("/%vmatches_%d_match_%d", internalLocationPrefix, index, i)
		if len(m.Splits) > 0 {
			r = variableNamer.GetNameForSplitClientVariable(scIndex + scLocalIndex)
			scLocalIndex++
		}

		p := version2.Parameter{
			Value:  v,
			Result: r,
		}
		params = append(params, p)
	}

	defaultResult := fmt.Sprintf("/%vmatches_%d_default", internalLocationPrefix, index)
	if len(route.Splits) > 0 {
		defaultResult = variableNamer.GetNameForSplitClientVariable(scIndex + scLocalIndex)
	}

	defaultParam := version2.Parameter{
		Value:  "default",
		Result: defaultResult,
	}
	params = append(params, defaultParam)

	variable := variableNamer.GetNameForVariableForMatchesRouteMainMap(index)

	mainMap := version2.Map{
		Source:     source,
		Variable:   variable,
		Parameters: params,
	}
	maps = append(maps, mainMap)

	// Generate locations for each match and split client
	var locations []version2.Location
	var returnLocations []version2.ReturnLocation
	var splitClients []version2.SplitClient
	scLocalIndex = 0

	for i, m := range route.Matches {
		if len(m.Splits) > 0 {
			newRetLocIndex := retLocIndex + len(returnLocations)
			sc, locs, returnLocs := generateSplits(m.Splits, upstreamNamer, crUpstreams, variableNamer, scIndex+scLocalIndex,
				cfgParams, errorPages, errPageIndex, route.Path, locSnippets, enableSnippets, newRetLocIndex)
			scLocalIndex++
			splitClients = append(splitClients, sc)
			locations = append(locations, locs...)
			returnLocations = append(returnLocations, returnLocs...)
		} else {
			path := fmt.Sprintf("/%vmatches_%d_match_%d", internalLocationPrefix, index, i)
			upstreamName := upstreamNamer.GetNameForUpstreamFromAction(m.Action)
			upstream := crUpstreams[upstreamName]
			proxySSLName := generateProxySSLName(upstream.Service, upstreamNamer.namespace)
			newRetLocIndex := retLocIndex + len(returnLocations)
			loc, returnLoc := generateLocation(path, upstreamName, upstream, m.Action, cfgParams, errorPages, true,
				errPageIndex, proxySSLName, route.Path, locSnippets, enableSnippets, newRetLocIndex)
			locations = append(locations, loc)
			if returnLoc != nil {
				returnLocations = append(returnLocations, *returnLoc)
			}
		}
	}

	// Generate default splits or default action
	if len(route.Splits) > 0 {
		newRetLocIndex := retLocIndex + len(returnLocations)
		sc, locs, returnLocs := generateSplits(route.Splits, upstreamNamer, crUpstreams, variableNamer, scIndex+scLocalIndex,
			cfgParams, errorPages, errPageIndex, route.Path, locSnippets, enableSnippets, newRetLocIndex)
		splitClients = append(splitClients, sc)
		locations = append(locations, locs...)
		returnLocations = append(returnLocations, returnLocs...)
	} else {
		path := fmt.Sprintf("/%vmatches_%d_default", internalLocationPrefix, index)
		upstreamName := upstreamNamer.GetNameForUpstreamFromAction(route.Action)
		upstream := crUpstreams[upstreamName]
		proxySSLName := generateProxySSLName(upstream.Service, upstreamNamer.namespace)
		newRetLocIndex := retLocIndex + len(returnLocations)
		loc, returnLoc := generateLocation(path, upstreamName, upstream, route.Action, cfgParams, errorPages, true,
			errPageIndex, proxySSLName, route.Path, locSnippets, enableSnippets, newRetLocIndex)
		locations = append(locations, loc)
		if returnLoc != nil {
			returnLocations = append(returnLocations, *returnLoc)
		}
	}

	// Generate an InternalRedirectLocation to the location defined by the main map variable
	irl := version2.InternalRedirectLocation{
		Path:        route.Path,
		Destination: variable,
	}

	return routingCfg{
		Maps:                     maps,
		Locations:                locations,
		InternalRedirectLocation: irl,
		SplitClients:             splitClients,
		ReturnLocations:          returnLocations,
	}
}

var specialMapParameters = map[string]bool{
	"default":   true,
	"hostnames": true,
	"include":   true,
	"volatile":  true,
}

func generateValueForMatchesRouteMap(matchedValue string) (value string, isNegative bool) {
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

func generateParametersForMatchesRouteMap(matchedValue string, successfulResult string) []version2.Parameter {
	value, isNegative := generateValueForMatchesRouteMap(matchedValue)

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

func getNameForSourceForMatchesRouteMapFromCondition(condition conf_v1.Condition) string {
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

func generateSSLConfig(tls *conf_v1.TLS, tlsPemFileName string, cfgParams *ConfigParams) *version2.SSL {
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
		HTTP2:          cfgParams.HTTP2,
		Certificate:    name,
		CertificateKey: name,
		Ciphers:        ciphers,
	}

	return &ssl
}

func generateTLSRedirectConfig(tls *conf_v1.TLS) *version2.TLSRedirect {
	if tls == nil || tls.Redirect == nil || !tls.Redirect.Enable {
		return nil
	}

	redirect := &version2.TLSRedirect{
		Code:    generateIntFromPointer(tls.Redirect.Code, 301),
		BasedOn: generateTLSRedirectBasedOn(tls.Redirect.BasedOn),
	}

	return redirect
}

func generateTLSRedirectBasedOn(basedOn string) string {
	if basedOn == "x-forwarded-proto" {
		return "$http_x_forwarded_proto"
	}
	return "$scheme"
}

func createEndpointsFromUpstream(upstream version2.Upstream) []string {
	var endpoints []string

	for _, server := range upstream.Servers {
		endpoints = append(endpoints, server.Address)
	}

	return endpoints
}

func createUpstreamsForPlus(virtualServerEx *VirtualServerEx, baseCfgParams *ConfigParams, staticParams *StaticConfigParams) []version2.Upstream {
	var upstreams []version2.Upstream

	isPlus := true
	upstreamNamer := newUpstreamNamerForVirtualServer(virtualServerEx.VirtualServer)
	vsc := newVirtualServerConfigurator(baseCfgParams, isPlus, false, staticParams)

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

func generateQueueForPlus(upstreamQueue *conf_v1.UpstreamQueue, defaultTimeout string) *version2.Queue {
	if upstreamQueue == nil {
		return nil
	}

	return &version2.Queue{
		Size:    upstreamQueue.Size,
		Timeout: generateString(upstreamQueue.Timeout, defaultTimeout),
	}
}

func generateErrorPageName(errPageIndex int, index int) string {
	return fmt.Sprintf("@error_page_%v_%v", errPageIndex, index)
}

func generateErrorPageCodes(codes []int) string {
	var c []string
	for _, code := range codes {
		c = append(c, strconv.Itoa(code))
	}
	return strings.Join(c, " ")
}

func generateErrorPages(errPageIndex int, errorPages []conf_v1.ErrorPage) []version2.ErrorPage {
	var ePages []version2.ErrorPage

	for i, e := range errorPages {
		var code int
		var name string

		if e.Redirect != nil {
			code = 301
			if e.Redirect.Code != 0 {
				code = e.Redirect.Code
			}
			name = e.Redirect.URL
		} else {
			code = e.Return.Code
			name = generateErrorPageName(errPageIndex, i)
		}

		ep := version2.ErrorPage{
			Name:         name,
			Codes:        generateErrorPageCodes(e.Codes),
			ResponseCode: code,
		}

		ePages = append(ePages, ep)
	}

	return ePages
}

func generateErrorPageLocations(errPageIndex int, errorPages []conf_v1.ErrorPage) []version2.ErrorPageLocation {
	var errorPageLocations []version2.ErrorPageLocation
	for i, e := range errorPages {
		if e.Redirect != nil {
			// Redirects are handled in the error_page of the location directly, no need for a named location.
			continue
		}

		var headers []version2.Header

		for _, h := range e.Return.Headers {
			headers = append(headers, version2.Header{
				Name:  h.Name,
				Value: h.Value,
			})
		}

		defaultType := "text/html"
		if e.Return.Type != "" {
			defaultType = e.Return.Type
		}

		epl := version2.ErrorPageLocation{
			Name:        generateErrorPageName(errPageIndex, i),
			DefaultType: defaultType,
			Return:      generateReturnBlock(e.Return.Body, 0, 0),
			Headers:     headers,
		}

		errorPageLocations = append(errorPageLocations, epl)
	}

	return errorPageLocations
}

func generateProxySSLName(svcName, ns string) string {
	return fmt.Sprintf("%s.%s.svc", svcName, ns)
}

func isTLSEnabled(u conf_v1.Upstream, spiffeCerts bool) bool {
	return u.TLS.Enable || spiffeCerts
}

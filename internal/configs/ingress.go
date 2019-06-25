package configs

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
)

const emptyHost = ""

// IngressEx holds an Ingress along with the resources that are referenced in this Ingress.
type IngressEx struct {
	Ingress          *extensions.Ingress
	TLSSecrets       map[string]*api_v1.Secret
	JWTKey           JWTKey
	Endpoints        map[string][]string
	HealthChecks     map[string]*api_v1.Probe
	ExternalNameSvcs map[string]bool
}

// JWTKey represents a secret that holds JSON Web Key.
type JWTKey struct {
	Name   string
	Secret *api_v1.Secret
}

func (ingEx *IngressEx) String() string {
	if ingEx.Ingress == nil {
		return "IngressEx has no Ingress"
	}

	return fmt.Sprintf("%v/%v", ingEx.Ingress.Namespace, ingEx.Ingress.Name)
}

// MergeableIngresses is a mergeable ingress of a master and minions.
type MergeableIngresses struct {
	Master  *IngressEx
	Minions []*IngressEx
}

func generateNginxCfg(ingEx *IngressEx, pems map[string]string, isMinion bool, baseCfgParams *ConfigParams, isPlus bool, isResolverConfigured bool, jwtKeyFileName string) version1.IngressNginxConfig {
	cfgParams := parseAnnotations(ingEx, baseCfgParams, isPlus)
	wsServices := getWebsocketServices(ingEx)
	spServices := getSessionPersistenceServices(ingEx)
	rewrites := getRewrites(ingEx)
	sslServices := getSSLServices(ingEx)
	grpcServices := getGrpcServices(ingEx)

	upstreams := make(map[string]version1.Upstream)
	healthChecks := make(map[string]version1.HealthCheck)

	// HTTP2 is required for gRPC to function
	if len(grpcServices) > 0 && !cfgParams.HTTP2 {
		glog.Errorf("Ingress %s/%s: annotation nginx.org/grpc-services requires HTTP2, ignoring", ingEx.Ingress.Namespace, ingEx.Ingress.Name)
		grpcServices = make(map[string]bool)
	}

	if ingEx.Ingress.Spec.Backend != nil {
		name := getNameForUpstream(ingEx.Ingress, emptyHost, ingEx.Ingress.Spec.Backend)
		upstream := createUpstream(ingEx, name, ingEx.Ingress.Spec.Backend, spServices[ingEx.Ingress.Spec.Backend.ServiceName], &cfgParams,
			isPlus, isResolverConfigured)
		upstreams[name] = upstream

		if cfgParams.HealthCheckEnabled {
			if hc, exists := ingEx.HealthChecks[ingEx.Ingress.Spec.Backend.ServiceName+ingEx.Ingress.Spec.Backend.ServicePort.String()]; exists {
				healthChecks[name] = createHealthCheck(hc, name, &cfgParams)
			}
		}
	}

	var servers []version1.Server

	for _, rule := range ingEx.Ingress.Spec.Rules {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		serverName := rule.Host

		statusZone := rule.Host

		server := version1.Server{
			Name:                  serverName,
			ServerTokens:          cfgParams.ServerTokens,
			HTTP2:                 cfgParams.HTTP2,
			RedirectToHTTPS:       cfgParams.RedirectToHTTPS,
			SSLRedirect:           cfgParams.SSLRedirect,
			ProxyProtocol:         cfgParams.ProxyProtocol,
			HSTS:                  cfgParams.HSTS,
			HSTSMaxAge:            cfgParams.HSTSMaxAge,
			HSTSIncludeSubdomains: cfgParams.HSTSIncludeSubdomains,
			HSTSBehindProxy:       cfgParams.HSTSBehindProxy,
			StatusZone:            statusZone,
			RealIPHeader:          cfgParams.RealIPHeader,
			SetRealIPFrom:         cfgParams.SetRealIPFrom,
			RealIPRecursive:       cfgParams.RealIPRecursive,
			ProxyHideHeaders:      cfgParams.ProxyHideHeaders,
			ProxyPassHeaders:      cfgParams.ProxyPassHeaders,
			ServerSnippets:        cfgParams.ServerSnippets,
			Ports:                 cfgParams.Ports,
			SSLPorts:              cfgParams.SSLPorts,
		}

		if pemFile, ok := pems[serverName]; ok {
			server.SSL = true
			server.SSLCertificate = pemFile
			server.SSLCertificateKey = pemFile
			if pemFile == pemFileNameForMissingTLSSecret {
				server.SSLCiphers = "NULL"
			}
		}

		if !isMinion && ingEx.JWTKey.Name != "" {
			server.JWTAuth = &version1.JWTAuth{
				Key:   jwtKeyFileName,
				Realm: cfgParams.JWTRealm,
				Token: cfgParams.JWTToken,
			}

			if cfgParams.JWTLoginURL != "" {
				server.JWTAuth.RedirectLocationName = getNameForRedirectLocation(ingEx.Ingress)
				server.JWTRedirectLocations = append(server.JWTRedirectLocations, version1.JWTRedirectLocation{
					Name:     server.JWTAuth.RedirectLocationName,
					LoginURL: cfgParams.JWTLoginURL,
				})
			}
		}

		var locations []version1.Location
		healthChecks := make(map[string]version1.HealthCheck)

		rootLocation := false

		grpcOnly := true
		if len(grpcServices) > 0 {
			for _, path := range rule.HTTP.Paths {
				if _, exists := grpcServices[path.Backend.ServiceName]; !exists {
					grpcOnly = false
					break
				}
			}
		} else {
			grpcOnly = false
		}

		for _, path := range rule.HTTP.Paths {
			upsName := getNameForUpstream(ingEx.Ingress, rule.Host, &path.Backend)

			if cfgParams.HealthCheckEnabled {
				if hc, exists := ingEx.HealthChecks[path.Backend.ServiceName+path.Backend.ServicePort.String()]; exists {
					healthChecks[upsName] = createHealthCheck(hc, upsName, &cfgParams)
				}
			}

			if _, exists := upstreams[upsName]; !exists {
				upstream := createUpstream(ingEx, upsName, &path.Backend, spServices[path.Backend.ServiceName], &cfgParams, isPlus, isResolverConfigured)
				upstreams[upsName] = upstream
			}

			loc := createLocation(pathOrDefault(path.Path), upstreams[upsName], &cfgParams, wsServices[path.Backend.ServiceName], rewrites[path.Backend.ServiceName],
				sslServices[path.Backend.ServiceName], grpcServices[path.Backend.ServiceName])
			if isMinion && ingEx.JWTKey.Name != "" {
				loc.JWTAuth = &version1.JWTAuth{
					Key:   jwtKeyFileName,
					Realm: cfgParams.JWTRealm,
					Token: cfgParams.JWTToken,
				}

				if cfgParams.JWTLoginURL != "" {
					loc.JWTAuth.RedirectLocationName = getNameForRedirectLocation(ingEx.Ingress)
					server.JWTRedirectLocations = append(server.JWTRedirectLocations, version1.JWTRedirectLocation{
						Name:     loc.JWTAuth.RedirectLocationName,
						LoginURL: cfgParams.JWTLoginURL,
					})
				}
			}
			locations = append(locations, loc)

			if loc.Path == "/" {
				rootLocation = true
			}
		}

		if !rootLocation && ingEx.Ingress.Spec.Backend != nil {
			upsName := getNameForUpstream(ingEx.Ingress, emptyHost, ingEx.Ingress.Spec.Backend)

			loc := createLocation(pathOrDefault("/"), upstreams[upsName], &cfgParams, wsServices[ingEx.Ingress.Spec.Backend.ServiceName], rewrites[ingEx.Ingress.Spec.Backend.ServiceName],
				sslServices[ingEx.Ingress.Spec.Backend.ServiceName], grpcServices[ingEx.Ingress.Spec.Backend.ServiceName])
			locations = append(locations, loc)

			if cfgParams.HealthCheckEnabled {
				if hc, exists := ingEx.HealthChecks[ingEx.Ingress.Spec.Backend.ServiceName+ingEx.Ingress.Spec.Backend.ServicePort.String()]; exists {
					healthChecks[upsName] = createHealthCheck(hc, upsName, &cfgParams)
				}
			}

			if _, exists := grpcServices[ingEx.Ingress.Spec.Backend.ServiceName]; !exists {
				grpcOnly = false
			}
		}

		server.Locations = locations
		server.HealthChecks = healthChecks
		server.GRPCOnly = grpcOnly

		servers = append(servers, server)
	}

	var keepalive string
	if cfgParams.Keepalive > 0 {
		keepalive = strconv.FormatInt(cfgParams.Keepalive, 10)
	}

	return version1.IngressNginxConfig{
		Upstreams: upstreamMapToSlice(upstreams),
		Servers:   servers,
		Keepalive: keepalive,
		Ingress: version1.Ingress{
			Name:        ingEx.Ingress.Name,
			Namespace:   ingEx.Ingress.Namespace,
			Annotations: ingEx.Ingress.Annotations,
		},
	}
}

func createLocation(path string, upstream version1.Upstream, cfg *ConfigParams, websocket bool, rewrite string, ssl bool, grpc bool) version1.Location {
	loc := version1.Location{
		Path:                 path,
		Upstream:             upstream,
		ProxyConnectTimeout:  cfg.ProxyConnectTimeout,
		ProxyReadTimeout:     cfg.ProxyReadTimeout,
		ProxySendTimeout:     cfg.ProxySendTimeout,
		ClientMaxBodySize:    cfg.ClientMaxBodySize,
		Websocket:            websocket,
		Rewrite:              rewrite,
		SSL:                  ssl,
		GRPC:                 grpc,
		ProxyBuffering:       cfg.ProxyBuffering,
		ProxyBuffers:         cfg.ProxyBuffers,
		ProxyBufferSize:      cfg.ProxyBufferSize,
		ProxyMaxTempFileSize: cfg.ProxyMaxTempFileSize,
		LocationSnippets:     cfg.LocationSnippets,
	}

	return loc
}

// upstreamRequiresQueue checks if the upstream requires a queue.
// Mandatory Health Checks can cause nginx to return errors on reload, since all Upstreams start
// Unhealthy. By adding a queue to the Upstream we can avoid returning errors, at the cost of a short delay.
func upstreamRequiresQueue(name string, ingEx *IngressEx, cfg *ConfigParams) (n int64, timeout int64) {
	if cfg.HealthCheckEnabled && cfg.HealthCheckMandatory && cfg.HealthCheckMandatoryQueue > 0 {
		if hc, exists := ingEx.HealthChecks[name]; exists {
			return cfg.HealthCheckMandatoryQueue, int64(hc.TimeoutSeconds)
		}
	}
	return 0, 0
}

func createUpstream(ingEx *IngressEx, name string, backend *extensions.IngressBackend, stickyCookie string, cfg *ConfigParams,
	isPlus bool, isResolverConfigured bool) version1.Upstream {
	var ups version1.Upstream

	if isPlus {
		queue, timeout := upstreamRequiresQueue(backend.ServiceName+backend.ServicePort.String(), ingEx, cfg)
		ups = version1.Upstream{Name: name, StickyCookie: stickyCookie, Queue: queue, QueueTimeout: timeout}
	} else {
		ups = version1.NewUpstreamWithDefaultServer(name)
	}

	endps, exists := ingEx.Endpoints[backend.ServiceName+backend.ServicePort.String()]
	if exists {
		var upsServers []version1.UpstreamServer
		// Always false for NGINX OSS
		_, isExternalNameSvc := ingEx.ExternalNameSvcs[backend.ServiceName]
		if isExternalNameSvc && !isResolverConfigured {
			glog.Warningf("A resolver must be configured for Type ExternalName service %s, no upstream servers will be created", backend.ServiceName)
			endps = []string{}
		}

		for _, endp := range endps {
			addressport := strings.Split(endp, ":")
			upsServers = append(upsServers, version1.UpstreamServer{
				Address:     addressport[0],
				Port:        addressport[1],
				MaxFails:    cfg.MaxFails,
				FailTimeout: cfg.FailTimeout,
				SlowStart:   cfg.SlowStart,
				Resolve:     isExternalNameSvc,
			})
		}
		if len(upsServers) > 0 {
			ups.UpstreamServers = upsServers
		}
	}

	ups.LBMethod = cfg.LBMethod
	return ups
}

func createHealthCheck(hc *api_v1.Probe, upstreamName string, cfg *ConfigParams) version1.HealthCheck {
	return version1.HealthCheck{
		UpstreamName:   upstreamName,
		Fails:          hc.FailureThreshold,
		Interval:       hc.PeriodSeconds,
		Passes:         hc.SuccessThreshold,
		URI:            hc.HTTPGet.Path,
		Scheme:         strings.ToLower(string(hc.HTTPGet.Scheme)),
		Mandatory:      cfg.HealthCheckMandatory,
		Headers:        headersToString(hc.HTTPGet.HTTPHeaders),
		TimeoutSeconds: int64(hc.TimeoutSeconds),
	}
}

func headersToString(headers []api_v1.HTTPHeader) map[string]string {
	m := make(map[string]string)
	for _, header := range headers {
		m[header.Name] = header.Value
	}
	return m
}

func pathOrDefault(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

func getNameForUpstream(ing *extensions.Ingress, host string, backend *extensions.IngressBackend) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", ing.Namespace, ing.Name, host, backend.ServiceName, backend.ServicePort.String())
}

func getNameForRedirectLocation(ing *extensions.Ingress) string {
	return fmt.Sprintf("@login_url_%v-%v", ing.Namespace, ing.Name)
}

func upstreamMapToSlice(upstreams map[string]version1.Upstream) []version1.Upstream {
	keys := make([]string, 0, len(upstreams))
	for k := range upstreams {
		keys = append(keys, k)
	}

	// this ensures that the slice 'result' is sorted, which preserves the order of upstream servers
	// in the generated configuration file from one version to another and is also required for repeatable
	// Unit test results
	sort.Strings(keys)

	result := make([]version1.Upstream, 0, len(upstreams))

	for _, k := range keys {
		result = append(result, upstreams[k])
	}

	return result
}

func generateNginxCfgForMergeableIngresses(mergeableIngs *MergeableIngresses, masterPems map[string]string, masterJwtKeyFileName string,
	minionJwtKeyFileNames map[string]string, baseCfgParams *ConfigParams, isPlus bool, isResolverConfigured bool) version1.IngressNginxConfig {
	var masterServer version1.Server
	var locations []version1.Location
	var upstreams []version1.Upstream
	healthChecks := make(map[string]version1.HealthCheck)
	var keepalive string

	removedAnnotations := filterMasterAnnotations(mergeableIngs.Master.Ingress.Annotations)
	if len(removedAnnotations) != 0 {
		glog.Errorf("Ingress Resource %v/%v with the annotation 'nginx.org/mergeable-ingress-type' set to 'master' cannot contain the '%v' annotation(s). They will be ignored",
			mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, strings.Join(removedAnnotations, ","))
	}

	isMinion := false
	masterNginxCfg := generateNginxCfg(mergeableIngs.Master, masterPems, isMinion, baseCfgParams, isPlus, isResolverConfigured, masterJwtKeyFileName)

	masterServer = masterNginxCfg.Servers[0]
	masterServer.Locations = []version1.Location{}

	upstreams = append(upstreams, masterNginxCfg.Upstreams...)

	if masterNginxCfg.Keepalive != "" {
		keepalive = masterNginxCfg.Keepalive
	}

	minions := mergeableIngs.Minions
	for _, minion := range minions {
		// Remove the default backend so that "/" will not be generated
		minion.Ingress.Spec.Backend = nil

		// Add acceptable master annotations to minion
		mergeMasterAnnotationsIntoMinion(minion.Ingress.Annotations, mergeableIngs.Master.Ingress.Annotations)

		removedAnnotations = filterMinionAnnotations(minion.Ingress.Annotations)
		if len(removedAnnotations) != 0 {
			glog.Errorf("Ingress Resource %v/%v with the annotation 'nginx.org/mergeable-ingress-type' set to 'minion' cannot contain the %v annotation(s). They will be ignored",
				minion.Ingress.Namespace, minion.Ingress.Name, strings.Join(removedAnnotations, ","))
		}

		pems := make(map[string]string)
		jwtKeyFileName := minionJwtKeyFileNames[objectMetaToFileName(&minion.Ingress.ObjectMeta)]
		isMinion := true
		nginxCfg := generateNginxCfg(minion, pems, isMinion, baseCfgParams, isPlus, isResolverConfigured, jwtKeyFileName)

		for _, server := range nginxCfg.Servers {
			for _, loc := range server.Locations {
				loc.MinionIngress = &nginxCfg.Ingress
				locations = append(locations, loc)
			}
			for hcName, healthCheck := range server.HealthChecks {
				healthChecks[hcName] = healthCheck
			}
			masterServer.JWTRedirectLocations = append(masterServer.JWTRedirectLocations, server.JWTRedirectLocations...)
		}

		upstreams = append(upstreams, nginxCfg.Upstreams...)
	}

	masterServer.HealthChecks = healthChecks
	masterServer.Locations = locations

	return version1.IngressNginxConfig{
		Servers:   []version1.Server{masterServer},
		Upstreams: upstreams,
		Keepalive: keepalive,
		Ingress:   masterNginxCfg.Ingress,
	}
}

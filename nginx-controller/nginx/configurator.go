package nginx

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/nginx-controller/nginx/plus"
	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const emptyHost = ""

// DefaultServerSecretName is the filename of the Secret with a TLS cert and a key for the default server
const DefaultServerSecretName = "default"

// JWTKey is the key of the data field of a Secret where the JWK must be stored.
const JWTKey = "jwk"

// JWTKeyAnnotation is the annotation where the Secret with a JWK is specified.
const JWTKeyAnnotation = "nginx.com/jwt-key"

// Configurator transforms an Ingress resource into NGINX Configuration
type Configurator struct {
	nginx            *NginxController
	config           *Config
	nginxAPI         *plus.NginxAPIController
	templateExecutor *TemplateExecutor
	ingresses        map[string]*IngressEx
	minions          map[string]map[string]bool
}

// NewConfigurator creates a new Configurator
func NewConfigurator(nginx *NginxController, config *Config, nginxAPI *plus.NginxAPIController, templateExecutor *TemplateExecutor) *Configurator {
	cnf := Configurator{
		nginx:            nginx,
		config:           config,
		nginxAPI:         nginxAPI,
		ingresses:        make(map[string]*IngressEx),
		templateExecutor: templateExecutor,
		minions:          make(map[string]map[string]bool),
	}

	return &cnf
}

// AddOrUpdateDHParam creates a dhparam file with the content of the string.
func (cnf *Configurator) AddOrUpdateDHParam(content string) (string, error) {
	return cnf.nginx.AddOrUpdateDHParam(content)
}

// AddOrUpdateIngress adds or updates NGINX configuration for the Ingress resource
func (cnf *Configurator) AddOrUpdateIngress(ingEx *IngressEx) error {
	if err := cnf.addOrUpdateIngress(ingEx); err != nil {
		return fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
	}
	if err := cnf.nginx.Reload(); err != nil {
		return fmt.Errorf("Error reloading NGINX for %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
	}
	return nil
}

func (cnf *Configurator) addOrUpdateIngress(ingEx *IngressEx) error {
	pems, jwtKeyFileName := cnf.updateSecrets(ingEx)
	isMinion := false
	nginxCfg := cnf.generateNginxCfg(ingEx, pems, jwtKeyFileName, isMinion)
	name := objectMetaToFileName(&ingEx.Ingress.ObjectMeta)
	content, err := cnf.templateExecutor.ExecuteIngressConfigTemplate(&nginxCfg)
	if err != nil {
		return fmt.Errorf("Error generating Ingress Config %v: %v", name, err)
	}
	cnf.nginx.UpdateIngressConfigFile(name, content)
	cnf.ingresses[name] = ingEx
	return nil
}

// AddOrUpdateMergeableIngress adds or updates NGINX configuration for the Ingress resources with Mergeable Types
func (cnf *Configurator) AddOrUpdateMergeableIngress(mergeableIngs *MergeableIngresses) error {
	if err := cnf.addOrUpdateMergeableIngress(mergeableIngs); err != nil {
		return fmt.Errorf("Error when adding or updating ingress %v/%v: %v", mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, err)
	}
	if err := cnf.nginx.Reload(); err != nil {
		return fmt.Errorf("Error reloading NGINX for %v/%v: %v", mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, err)
	}
	return nil
}

func (cnf *Configurator) addOrUpdateMergeableIngress(mergeableIngs *MergeableIngresses) error {
	nginxCfg := cnf.generateNginxCfgForMergeableIngresses(mergeableIngs)
	name := objectMetaToFileName(&mergeableIngs.Master.Ingress.ObjectMeta)
	content, err := cnf.templateExecutor.ExecuteIngressConfigTemplate(&nginxCfg)
	if err != nil {
		return fmt.Errorf("Error generating Ingress Config %v: %v", name, err)
	}
	cnf.nginx.UpdateIngressConfigFile(name, content)
	cnf.ingresses[name] = mergeableIngs.Master
	cnf.minions[name] = make(map[string]bool)
	for _, minion := range mergeableIngs.Minions {
		minionName := objectMetaToFileName(&minion.Ingress.ObjectMeta)
		cnf.minions[name][minionName] = true
	}
	return nil
}

func (cnf *Configurator) generateNginxCfgForMergeableIngresses(mergeableIngs *MergeableIngresses) IngressNginxConfig {
	var masterServer Server
	var locations []Location
	var upstreams []Upstream
	healthChecks := make(map[string]HealthCheck)
	var keepalive string
	var removedAnnotations []string

	removedAnnotations = filterMasterAnnotations(mergeableIngs.Master.Ingress.Annotations)
	if len(removedAnnotations) != 0 {
		glog.Errorf("Ingress Resource %v/%v with the annotation 'nginx.org/mergeable-ingress-type' set to 'master' cannot contain the '%v' annotation(s). They will be ignored",
			mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, strings.Join(removedAnnotations, ","))
	}

	pems, jwtKeyFileName := cnf.updateSecrets(mergeableIngs.Master)
	isMinion := false
	masterNginxCfg := cnf.generateNginxCfg(mergeableIngs.Master, pems, jwtKeyFileName, isMinion)

	masterServer = masterNginxCfg.Servers[0]
	masterServer.IngressResource = objectMetaToFileName(&mergeableIngs.Master.Ingress.ObjectMeta)
	masterServer.Locations = []Location{}

	for _, val := range masterNginxCfg.Upstreams {
		upstreams = append(upstreams, val)
	}
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

		pems, jwtKeyFileName := cnf.updateSecrets(minion)
		isMinion := true
		nginxCfg := cnf.generateNginxCfg(minion, pems, jwtKeyFileName, isMinion)

		for _, server := range nginxCfg.Servers {
			for _, loc := range server.Locations {
				loc.IngressResource = objectMetaToFileName(&minion.Ingress.ObjectMeta)
				locations = append(locations, loc)
			}
			for hcName, healthCheck := range server.HealthChecks {
				healthChecks[hcName] = healthCheck
			}
			masterServer.JWTRedirectLocations = append(masterServer.JWTRedirectLocations, server.JWTRedirectLocations...)
		}

		for _, val := range nginxCfg.Upstreams {
			upstreams = append(upstreams, val)
		}
	}

	masterServer.HealthChecks = healthChecks
	masterServer.Locations = locations

	return IngressNginxConfig{
		Servers:   []Server{masterServer},
		Upstreams: upstreams,
		Keepalive: keepalive,
	}
}

func (cnf *Configurator) updateSecrets(ingEx *IngressEx) (map[string]string, string) {
	pems := make(map[string]string)

	for _, tls := range ingEx.Ingress.Spec.TLS {
		secretName := tls.SecretName

		pemFileName := cnf.addOrUpdateSecret(ingEx.TLSSecrets[secretName])

		for _, host := range tls.Hosts {
			pems[host] = pemFileName
		}
		if len(tls.Hosts) == 0 {
			pems[emptyHost] = pemFileName
		}
	}

	jwtKeyFileName := ""

	if cnf.isPlus() && ingEx.JWTKey != nil {
		jwtKeyFileName = cnf.addOrUpdateSecret(ingEx.JWTKey)
	}

	return pems, jwtKeyFileName
}

func (cnf *Configurator) generateNginxCfg(ingEx *IngressEx, pems map[string]string, jwtKeyFileName string, isMinion bool) IngressNginxConfig {
	ingCfg := cnf.createConfig(ingEx)

	upstreams := make(map[string]Upstream)
	healthChecks := make(map[string]HealthCheck)

	wsServices := getWebsocketServices(ingEx)
	spServices := getSessionPersistenceServices(ingEx)
	rewrites := getRewrites(ingEx)
	sslServices := getSSLServices(ingEx)
	grpcServices := getGrpcServices(ingEx)

	// HTTP2 is required for gRPC to function
	if len(grpcServices) > 0 && !ingCfg.HTTP2 {
		glog.Errorf("Ingress %s/%s: annotation nginx.org/grpc-services requires HTTP2, ignoring", ingEx.Ingress.Namespace, ingEx.Ingress.Name)
		grpcServices = make(map[string]bool)
	}

	if ingEx.Ingress.Spec.Backend != nil {
		name := getNameForUpstream(ingEx.Ingress, emptyHost, ingEx.Ingress.Spec.Backend)
		upstream := cnf.createUpstream(ingEx, name, ingEx.Ingress.Spec.Backend, ingEx.Ingress.Namespace, spServices[ingEx.Ingress.Spec.Backend.ServiceName], &ingCfg)
		upstreams[name] = upstream

		if ingCfg.HealthCheckEnabled {
			if hc, exists := ingEx.HealthChecks[ingEx.Ingress.Spec.Backend.ServiceName+ingEx.Ingress.Spec.Backend.ServicePort.String()]; exists {
				healthChecks[name] = cnf.createHealthCheck(hc, name, &ingCfg)
			}
		}
	}

	var servers []Server

	for _, rule := range ingEx.Ingress.Spec.Rules {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		serverName := rule.Host

		statuzZone := rule.Host

		server := Server{
			Name:                  serverName,
			ServerTokens:          ingCfg.ServerTokens,
			HTTP2:                 ingCfg.HTTP2,
			RedirectToHTTPS:       ingCfg.RedirectToHTTPS,
			SSLRedirect:           ingCfg.SSLRedirect,
			ProxyProtocol:         ingCfg.ProxyProtocol,
			HSTS:                  ingCfg.HSTS,
			HSTSMaxAge:            ingCfg.HSTSMaxAge,
			HSTSIncludeSubdomains: ingCfg.HSTSIncludeSubdomains,
			StatusZone:            statuzZone,
			RealIPHeader:          ingCfg.RealIPHeader,
			SetRealIPFrom:         ingCfg.SetRealIPFrom,
			RealIPRecursive:       ingCfg.RealIPRecursive,
			ProxyHideHeaders:      ingCfg.ProxyHideHeaders,
			ProxyPassHeaders:      ingCfg.ProxyPassHeaders,
			ServerSnippets:        ingCfg.ServerSnippets,
			Ports:                 ingCfg.Ports,
			SSLPorts:              ingCfg.SSLPorts,
		}

		if pemFile, ok := pems[serverName]; ok {
			server.SSL = true
			server.SSLCertificate = pemFile
			server.SSLCertificateKey = pemFile
		}

		if !isMinion && jwtKeyFileName != "" {
			server.JWTAuth = &JWTAuth{
				Key:   jwtKeyFileName,
				Realm: ingCfg.JWTRealm,
				Token: ingCfg.JWTToken,
			}

			if ingCfg.JWTLoginURL != "" {
				server.JWTAuth.RedirectLocationName = getNameForRedirectLocation(ingEx.Ingress)
				server.JWTRedirectLocations = append(server.JWTRedirectLocations, JWTRedirectLocation{
					Name:     server.JWTAuth.RedirectLocationName,
					LoginURL: ingCfg.JWTLoginURL,
				})
			}
		}

		var locations []Location
		healthChecks := make(map[string]HealthCheck)

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

			if ingCfg.HealthCheckEnabled {
				if hc, exists := ingEx.HealthChecks[path.Backend.ServiceName+path.Backend.ServicePort.String()]; exists {
					healthChecks[upsName] = cnf.createHealthCheck(hc, upsName, &ingCfg)
				}
			}

			if _, exists := upstreams[upsName]; !exists {
				upstream := cnf.createUpstream(ingEx, upsName, &path.Backend, ingEx.Ingress.Namespace, spServices[path.Backend.ServiceName], &ingCfg)
				upstreams[upsName] = upstream
			}

			loc := createLocation(pathOrDefault(path.Path), upstreams[upsName], &ingCfg, wsServices[path.Backend.ServiceName], rewrites[path.Backend.ServiceName],
				sslServices[path.Backend.ServiceName], grpcServices[path.Backend.ServiceName])
			if isMinion && jwtKeyFileName != "" {
				loc.JWTAuth = &JWTAuth{
					Key:   jwtKeyFileName,
					Realm: ingCfg.JWTRealm,
					Token: ingCfg.JWTToken,
				}

				if ingCfg.JWTLoginURL != "" {
					loc.JWTAuth.RedirectLocationName = getNameForRedirectLocation(ingEx.Ingress)
					server.JWTRedirectLocations = append(server.JWTRedirectLocations, JWTRedirectLocation{
						Name:     loc.JWTAuth.RedirectLocationName,
						LoginURL: ingCfg.JWTLoginURL,
					})
				}
			}
			locations = append(locations, loc)

			if loc.Path == "/" {
				rootLocation = true
			}
		}

		if rootLocation == false && ingEx.Ingress.Spec.Backend != nil {
			upsName := getNameForUpstream(ingEx.Ingress, emptyHost, ingEx.Ingress.Spec.Backend)

			loc := createLocation(pathOrDefault("/"), upstreams[upsName], &ingCfg, wsServices[ingEx.Ingress.Spec.Backend.ServiceName], rewrites[ingEx.Ingress.Spec.Backend.ServiceName],
				sslServices[ingEx.Ingress.Spec.Backend.ServiceName], grpcServices[ingEx.Ingress.Spec.Backend.ServiceName])
			locations = append(locations, loc)

			if ingCfg.HealthCheckEnabled {
				if hc, exists := ingEx.HealthChecks[ingEx.Ingress.Spec.Backend.ServiceName+ingEx.Ingress.Spec.Backend.ServicePort.String()]; exists {
					healthChecks[upsName] = cnf.createHealthCheck(hc, upsName, &ingCfg)
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
	if ingCfg.Keepalive > 0 {
		keepalive = strconv.FormatInt(ingCfg.Keepalive, 10)
	}

	return IngressNginxConfig{
		Upstreams: upstreamMapToSlice(upstreams),
		Servers:   servers,
		Keepalive: keepalive,
	}
}

func (cnf *Configurator) createConfig(ingEx *IngressEx) Config {
	ingCfg := *cnf.config

	//Override from annotation
	if lbMethod, exists := ingEx.Ingress.Annotations["nginx.org/lb-method"]; exists {
		if cnf.isPlus() {
			if parsedMethod, err := ParseLBMethodForPlus(lbMethod); err != nil {
				glog.Errorf("Ingress %s/%s: Invalid value for the nginx.org/lb-method: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), lbMethod, err)
			} else {
				ingCfg.LBMethod = parsedMethod
			}
		} else {
			if parsedMethod, err := ParseLBMethod(lbMethod); err != nil {
				glog.Errorf("Ingress %s/%s: Invalid value for the nginx.org/lb-method: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), lbMethod, err)
			} else {
				ingCfg.LBMethod = parsedMethod
			}
		}
	}

	if healthCheckEnabled, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.com/health-checks", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		}
		if cnf.isPlus() {
			ingCfg.HealthCheckEnabled = healthCheckEnabled
		} else {
			glog.Warning("Annotation 'nginx.com/health-checks' requires NGINX Plus")
		}
	}
	if ingCfg.HealthCheckEnabled {
		if healthCheckMandatory, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.com/health-checks-mandatory", ingEx.Ingress); exists {
			if err != nil {
				glog.Error(err)
			}
			ingCfg.HealthCheckMandatory = healthCheckMandatory
		}
	}
	if ingCfg.HealthCheckMandatory {
		if healthCheckQueue, exists, err := GetMapKeyAsInt(ingEx.Ingress.Annotations, "nginx.com/health-checks-mandatory-queue", ingEx.Ingress); exists {
			if err != nil {
				glog.Error(err)
			}
			ingCfg.HealthCheckMandatoryQueue = healthCheckQueue
		}
	}

	if slowStart, exists := ingEx.Ingress.Annotations["nginx.com/slow-start"]; exists {
		if parsedSlowStart, err := ParseSlowStart(slowStart); err != nil {
			glog.Errorf("Ingress %s/%s: Invalid value nginx.org/slow-start: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), slowStart, err)
		} else {
			if cnf.isPlus() {
				ingCfg.SlowStart = parsedSlowStart
			} else {
				glog.Warning("Annotation 'nginx.com/slow-start' requires NGINX Plus")
			}
		}
	}

	if serverTokens, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/server-tokens", ingEx.Ingress); exists {
		if err != nil {
			if cnf.isPlus() {
				ingCfg.ServerTokens = ingEx.Ingress.Annotations["nginx.org/server-tokens"]
			} else {
				glog.Error(err)
			}
		} else {
			ingCfg.ServerTokens = "off"
			if serverTokens {
				ingCfg.ServerTokens = "on"
			}
		}
	}

	if serverSnippets, exists, err := GetMapKeyAsStringSlice(ingEx.Ingress.Annotations, "nginx.org/server-snippets", ingEx.Ingress, "\n"); exists {
		if err != nil {
			glog.Error(err)
		} else {
			ingCfg.ServerSnippets = serverSnippets
		}
	}
	if locationSnippets, exists, err := GetMapKeyAsStringSlice(ingEx.Ingress.Annotations, "nginx.org/location-snippets", ingEx.Ingress, "\n"); exists {
		if err != nil {
			glog.Error(err)
		} else {
			ingCfg.LocationSnippets = locationSnippets
		}
	}

	if proxyConnectTimeout, exists := ingEx.Ingress.Annotations["nginx.org/proxy-connect-timeout"]; exists {
		ingCfg.ProxyConnectTimeout = proxyConnectTimeout
	}
	if proxyReadTimeout, exists := ingEx.Ingress.Annotations["nginx.org/proxy-read-timeout"]; exists {
		ingCfg.ProxyReadTimeout = proxyReadTimeout
	}
	if proxyHideHeaders, exists, err := GetMapKeyAsStringSlice(ingEx.Ingress.Annotations, "nginx.org/proxy-hide-headers", ingEx.Ingress, ","); exists {
		if err != nil {
			glog.Error(err)
		} else {
			ingCfg.ProxyHideHeaders = proxyHideHeaders
		}
	}
	if proxyPassHeaders, exists, err := GetMapKeyAsStringSlice(ingEx.Ingress.Annotations, "nginx.org/proxy-pass-headers", ingEx.Ingress, ","); exists {
		if err != nil {
			glog.Error(err)
		} else {
			ingCfg.ProxyPassHeaders = proxyPassHeaders
		}
	}
	if clientMaxBodySize, exists := ingEx.Ingress.Annotations["nginx.org/client-max-body-size"]; exists {
		ingCfg.ClientMaxBodySize = clientMaxBodySize
	}
	if redirectToHTTPS, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/redirect-to-https", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			ingCfg.RedirectToHTTPS = redirectToHTTPS
		}
	}
	if sslRedirect, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "ingress.kubernetes.io/ssl-redirect", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			ingCfg.SSLRedirect = sslRedirect
		}
	}
	if proxyBuffering, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/proxy-buffering", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			ingCfg.ProxyBuffering = proxyBuffering
		}
	}

	if hsts, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/hsts", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			parsingErrors := false

			hstsMaxAge, existsMA, err := GetMapKeyAsInt(ingEx.Ingress.Annotations, "nginx.org/hsts-max-age", ingEx.Ingress)
			if existsMA && err != nil {
				glog.Error(err)
				parsingErrors = true
			}
			hstsIncludeSubdomains, existsIS, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/hsts-include-subdomains", ingEx.Ingress)
			if existsIS && err != nil {
				glog.Error(err)
				parsingErrors = true
			}

			if parsingErrors {
				glog.Errorf("Ingress %s/%s: There are configuration issues with hsts annotations, skipping annotions for all hsts settings", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName())
			} else {
				ingCfg.HSTS = hsts
				if existsMA {
					ingCfg.HSTSMaxAge = hstsMaxAge
				}
				if existsIS {
					ingCfg.HSTSIncludeSubdomains = hstsIncludeSubdomains
				}
			}
		}
	}

	if proxyBuffers, exists := ingEx.Ingress.Annotations["nginx.org/proxy-buffers"]; exists {
		ingCfg.ProxyBuffers = proxyBuffers
	}
	if proxyBufferSize, exists := ingEx.Ingress.Annotations["nginx.org/proxy-buffer-size"]; exists {
		ingCfg.ProxyBufferSize = proxyBufferSize
	}
	if proxyMaxTempFileSize, exists := ingEx.Ingress.Annotations["nginx.org/proxy-max-temp-file-size"]; exists {
		ingCfg.ProxyMaxTempFileSize = proxyMaxTempFileSize
	}

	if cnf.isPlus() {
		if jwtRealm, exists := ingEx.Ingress.Annotations["nginx.com/jwt-realm"]; exists {
			ingCfg.JWTRealm = jwtRealm
		}
		if jwtKey, exists := ingEx.Ingress.Annotations[JWTKeyAnnotation]; exists {
			ingCfg.JWTKey = fmt.Sprintf("%v/%v", ingEx.Ingress.Namespace, jwtKey)
		}
		if jwtToken, exists := ingEx.Ingress.Annotations["nginx.com/jwt-token"]; exists {
			ingCfg.JWTToken = jwtToken
		}
		if jwtLoginURL, exists := ingEx.Ingress.Annotations["nginx.com/jwt-login-url"]; exists {
			ingCfg.JWTLoginURL = jwtLoginURL
		}
	}

	ports, sslPorts := getServicesPorts(ingEx)
	if len(ports) > 0 {
		ingCfg.Ports = ports
	}

	if len(sslPorts) > 0 {
		ingCfg.SSLPorts = sslPorts
	}

	if keepalive, exists, err := GetMapKeyAsInt(ingEx.Ingress.Annotations, "nginx.org/keepalive", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			ingCfg.Keepalive = keepalive
		}
	}

	if maxFails, exists, err := GetMapKeyAsInt(ingEx.Ingress.Annotations, "nginx.org/max-fails", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			ingCfg.MaxFails = maxFails
		}
	}
	if failTimeout, exists := ingEx.Ingress.Annotations["nginx.org/fail-timeout"]; exists {
		ingCfg.FailTimeout = failTimeout
	}

	return ingCfg
}

func getWebsocketServices(ingEx *IngressEx) map[string]bool {
	wsServices := make(map[string]bool)

	if services, exists := ingEx.Ingress.Annotations["nginx.org/websocket-services"]; exists {
		for _, svc := range strings.Split(services, ",") {
			wsServices[svc] = true
		}
	}

	return wsServices
}

func getRewrites(ingEx *IngressEx) map[string]string {
	rewrites := make(map[string]string)

	if services, exists := ingEx.Ingress.Annotations["nginx.org/rewrites"]; exists {
		for _, svc := range strings.Split(services, ";") {
			if serviceName, rewrite, err := parseRewrites(svc); err != nil {
				glog.Errorf("In %v nginx.org/rewrites contains invalid declaration: %v, ignoring", ingEx.Ingress.Name, err)
			} else {
				rewrites[serviceName] = rewrite
			}
		}
	}

	return rewrites
}

func parseRewrites(service string) (serviceName string, rewrite string, err error) {
	parts := strings.SplitN(service, " ", 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Invalid rewrite format: %s", service)
	}

	svcNameParts := strings.Split(parts[0], "=")
	if len(svcNameParts) != 2 {
		return "", "", fmt.Errorf("Invalid rewrite format: %s", svcNameParts)
	}

	rwPathParts := strings.Split(parts[1], "=")
	if len(rwPathParts) != 2 {
		return "", "", fmt.Errorf("Invalid rewrite format: %s", rwPathParts)
	}

	return svcNameParts[1], rwPathParts[1], nil
}

func getSSLServices(ingEx *IngressEx) map[string]bool {
	sslServices := make(map[string]bool)

	if services, exists := ingEx.Ingress.Annotations["nginx.org/ssl-services"]; exists {
		for _, svc := range strings.Split(services, ",") {
			sslServices[svc] = true
		}
	}

	return sslServices
}

func getGrpcServices(ingEx *IngressEx) map[string]bool {
	grpcServices := make(map[string]bool)

	if services, exists := ingEx.Ingress.Annotations["nginx.org/grpc-services"]; exists {
		for _, svc := range strings.Split(services, ",") {
			grpcServices[svc] = true
		}
	}

	return grpcServices
}

func getSessionPersistenceServices(ingEx *IngressEx) map[string]string {
	spServices := make(map[string]string)

	if services, exists := ingEx.Ingress.Annotations["nginx.com/sticky-cookie-services"]; exists {
		for _, svc := range strings.Split(services, ";") {
			if serviceName, sticky, err := parseStickyService(svc); err != nil {
				glog.Errorf("In %v nginx.com/sticky-cookie-services contains invalid declaration: %v, ignoring", ingEx.Ingress.Name, err)
			} else {
				spServices[serviceName] = sticky
			}
		}
	}

	return spServices
}

func parseStickyService(service string) (serviceName string, stickyCookie string, err error) {
	parts := strings.SplitN(service, " ", 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Invalid sticky-cookie service format: %s", service)
	}

	svcNameParts := strings.Split(parts[0], "=")
	if len(svcNameParts) != 2 {
		return "", "", fmt.Errorf("Invalid sticky-cookie service format: %s", svcNameParts)
	}

	return svcNameParts[1], parts[1], nil
}

func getServicesPorts(ingEx *IngressEx) ([]int, []int) {
	ports := map[string][]int{}

	annotations := []string{
		"nginx.org/listen-ports",
		"nginx.org/listen-ports-ssl",
	}

	for _, annotation := range annotations {
		if values, exists := ingEx.Ingress.Annotations[annotation]; exists {
			for _, value := range strings.Split(values, ",") {
				if port, err := parsePort(value); err != nil {
					glog.Errorf(
						"In %v %s contains invalid declaration: %v, ignoring",
						ingEx.Ingress.Name,
						annotation,
						err,
					)
				} else {
					ports[annotation] = append(ports[annotation], port)
				}
			}
		}
	}

	return ports[annotations[0]], ports[annotations[1]]
}

func parsePort(value string) (int, error) {
	port, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return 0, fmt.Errorf(
			"Unable to parse port as integer: %s\n",
			err,
		)
	}

	if port <= 0 {
		return 0, fmt.Errorf(
			"Port number should be greater than zero: %q",
			port,
		)
	}

	return int(port), nil
}

func createLocation(path string, upstream Upstream, cfg *Config, websocket bool, rewrite string, ssl bool, grpc bool) Location {
	loc := Location{
		Path:                 path,
		Upstream:             upstream,
		ProxyConnectTimeout:  cfg.ProxyConnectTimeout,
		ProxyReadTimeout:     cfg.ProxyReadTimeout,
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
func (cnf *Configurator) upstreamRequiresQueue(name string, ingEx *IngressEx, cfg *Config) (n int64, timeout int64) {
	if cnf.isPlus() && cfg.HealthCheckEnabled && cfg.HealthCheckMandatory && cfg.HealthCheckMandatoryQueue > 0 {
		if hc, exists := ingEx.HealthChecks[name]; exists {
			return cfg.HealthCheckMandatoryQueue, int64(hc.TimeoutSeconds)
		}
	}
	return 0, 0
}

func (cnf *Configurator) createUpstream(ingEx *IngressEx, name string, backend *extensions.IngressBackend, namespace string, stickyCookie string, cfg *Config) Upstream {
	var ups Upstream

	queue, timeout := cnf.upstreamRequiresQueue(backend.ServiceName+backend.ServicePort.String(), ingEx, cfg)

	if cnf.isPlus() {
		ups = Upstream{Name: name, StickyCookie: stickyCookie, Queue: queue, QueueTimeout: timeout}
	} else {
		ups = NewUpstreamWithDefaultServer(name)
	}

	endps, exists := ingEx.Endpoints[backend.ServiceName+backend.ServicePort.String()]
	if exists {
		var upsServers []UpstreamServer
		for _, endp := range endps {
			addressport := strings.Split(endp, ":")
			upsServers = append(upsServers, UpstreamServer{
				Address:     addressport[0],
				Port:        addressport[1],
				MaxFails:    cfg.MaxFails,
				FailTimeout: cfg.FailTimeout,
				SlowStart:   cfg.SlowStart,
			})
		}
		if len(upsServers) > 0 {
			ups.UpstreamServers = upsServers
		}
	}
	ups.LBMethod = cfg.LBMethod
	return ups
}

func (cnf *Configurator) createHealthCheck(hc *api_v1.Probe, upstreamName string, cfg *Config) HealthCheck {
	return HealthCheck{
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

func upstreamMapToSlice(upstreams map[string]Upstream) []Upstream {
	keys := make([]string, 0, len(upstreams))
	for k := range upstreams {
		keys = append(keys, k)
	}

	// this ensures that the slice 'result' is sorted, which preserves the order of upstream servers
	// in the generated configuration file from one version to another and is also required for repeatable
	// Unit test results
	sort.Strings(keys)

	result := make([]Upstream, 0, len(upstreams))

	for _, k := range keys {
		result = append(result, upstreams[k])
	}

	return result
}

// AddOrUpdateSecret creates or updates a file with the content of the secret
func (cnf *Configurator) AddOrUpdateSecret(secret *api_v1.Secret) error {
	cnf.addOrUpdateSecret(secret)

	kind, _ := GetSecretKind(secret)
	if cnf.isPlus() && kind == JWK {
		return nil
	}

	if err := cnf.nginx.Reload(); err != nil {
		return fmt.Errorf("Error when reloading NGINX when updating Secret: %v", err)
	}
	return nil
}

func (cnf *Configurator) addOrUpdateSecret(secret *api_v1.Secret) string {
	name := objectMetaToFileName(&secret.ObjectMeta)

	var data []byte
	var mode os.FileMode

	kind, _ := GetSecretKind(secret)
	if cnf.isPlus() && kind == JWK {
		mode = jwkSecretFileMode
		data = []byte(secret.Data[JWTKey])
	} else {
		mode = TLSSecretFileMode
		data = GenerateCertAndKeyFileContent(secret)
	}
	return cnf.nginx.AddOrUpdateSecretFile(name, data, mode)
}

// AddOrUpdateDefaultServerTLSSecret creates or updates a file with a TLS cert and a key from the secret for the default server.
func (cnf *Configurator) AddOrUpdateDefaultServerTLSSecret(secret *api_v1.Secret) error {
	data := GenerateCertAndKeyFileContent(secret)
	cnf.nginx.AddOrUpdateSecretFile(DefaultServerSecretName, data, TLSSecretFileMode)

	if err := cnf.nginx.Reload(); err != nil {
		return fmt.Errorf("Error when reloading NGINX when updating the default server Secret: %v", err)
	}
	return nil
}

// GenerateCertAndKeyFileContent generates a pem file content from the secret
func GenerateCertAndKeyFileContent(secret *api_v1.Secret) []byte {
	var res bytes.Buffer

	res.Write(secret.Data[api_v1.TLSCertKey])
	res.WriteString("\n")
	res.Write(secret.Data[api_v1.TLSPrivateKeyKey])

	return res.Bytes()
}

// DeleteSecret deletes the file associated with the secret and the configuration files for the Ingress resources. NGINX is reloaded only when len(ings) > 0
func (cnf *Configurator) DeleteSecret(key string, ings []extensions.Ingress) error {
	for _, ing := range ings {
		name := objectMetaToFileName(&ing.ObjectMeta)
		cnf.nginx.DeleteIngress(name)
		delete(cnf.ingresses, name)
		delete(cnf.minions, name)
	}

	cnf.nginx.DeleteSecretFile(keyToFileName(key))

	if len(ings) > 0 {
		if err := cnf.nginx.Reload(); err != nil {
			return fmt.Errorf("Error when reloading NGINX when deleting Secret %v: %v", key, err)
		}
	}

	return nil
}

// DeleteIngress deletes NGINX configuration for the Ingress resource
func (cnf *Configurator) DeleteIngress(key string) error {
	name := keyToFileName(key)
	cnf.nginx.DeleteIngress(name)
	delete(cnf.ingresses, name)
	delete(cnf.minions, name)

	if err := cnf.nginx.Reload(); err != nil {
		return fmt.Errorf("Error when removing ingress %v: %v", key, err)
	}
	return nil
}

// UpdateEndpoints updates endpoints in NGINX configuration for the Ingress resource
func (cnf *Configurator) UpdateEndpoints(ingEx *IngressEx) error {
	err := cnf.addOrUpdateIngress(ingEx)
	if err != nil {
		return fmt.Errorf("Error adding or updating ingress %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
	}

	if cnf.isPlus() {
		err = cnf.updatePlusEndpoints(ingEx)
		if err == nil {
			return nil
		}
		glog.Warningf("Couldn't update the endpoints via the API: %v; reloading configuration instead", err)
	}
	if err := cnf.nginx.Reload(); err != nil {
		return fmt.Errorf("Error reloading NGINX when updating endpoints for %v/%v: %v", ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
	}

	return nil
}

// UpdateEndpointsMergeableIngress updates endpoints in NGINX configuration for a mergeable Ingress resource
func (cnf *Configurator) UpdateEndpointsMergeableIngress(mergeableIngs *MergeableIngresses) error {
	err := cnf.addOrUpdateMergeableIngress(mergeableIngs)
	if err != nil {
		return fmt.Errorf("Error adding or updating ingress %v/%v: %v", mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, err)
	}

	if cnf.isPlus() {
		for _, ing := range mergeableIngs.Minions {
			err = cnf.updatePlusEndpoints(ing)
			if err != nil {
				glog.Warningf("Couldn't update the endpoints via the API: %v; reloading configuration instead", err)
				break
			}
		}
		if err == nil {
			return nil
		}
	}

	if err := cnf.nginx.Reload(); err != nil {
		return fmt.Errorf("Error reloading NGINX when updating endpoints for %v/%v: %v", mergeableIngs.Master.Ingress.Namespace, mergeableIngs.Master.Ingress.Name, err)
	}

	return nil
}

func (cnf *Configurator) updatePlusEndpoints(ingEx *IngressEx) error {
	ingCfg := cnf.createConfig(ingEx)

	cfg := plus.ServerConfig{
		MaxFails:    ingCfg.MaxFails,
		FailTimeout: ingCfg.FailTimeout,
		SlowStart:   ingCfg.SlowStart,
	}

	if ingEx.Ingress.Spec.Backend != nil {
		name := getNameForUpstream(ingEx.Ingress, emptyHost, ingEx.Ingress.Spec.Backend)
		endps, exists := ingEx.Endpoints[ingEx.Ingress.Spec.Backend.ServiceName+ingEx.Ingress.Spec.Backend.ServicePort.String()]
		if exists {
			err := cnf.nginxAPI.UpdateServers(name, endps, cfg)
			if err != nil {
				return fmt.Errorf("Couldn't update the endpoints for %v: %v", name, err)
			}
		}
	}
	for _, rule := range ingEx.Ingress.Spec.Rules {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			name := getNameForUpstream(ingEx.Ingress, rule.Host, &path.Backend)
			endps, exists := ingEx.Endpoints[path.Backend.ServiceName+path.Backend.ServicePort.String()]
			if exists {
				err := cnf.nginxAPI.UpdateServers(name, endps, cfg)
				if err != nil {
					return fmt.Errorf("Couldn't update the endpoints for %v: %v", name, err)
				}
			}
		}
	}

	return nil
}

func filterMasterAnnotations(annotations map[string]string) []string {
	var removedAnnotations []string

	for key, _ := range annotations {
		if _, notAllowed := masterBlacklist[key]; notAllowed {
			removedAnnotations = append(removedAnnotations, key)
			delete(annotations, key)
		}
	}

	return removedAnnotations
}

func filterMinionAnnotations(annotations map[string]string) []string {
	var removedAnnotations []string

	for key, _ := range annotations {
		if _, notAllowed := minionBlacklist[key]; notAllowed {
			removedAnnotations = append(removedAnnotations, key)
			delete(annotations, key)
		}
	}

	return removedAnnotations
}

func mergeMasterAnnotationsIntoMinion(minionAnnotations map[string]string, masterAnnotations map[string]string) {
	for key, val := range masterAnnotations {
		if _, exists := minionAnnotations[key]; !exists {
			if _, allowed := minionInheritanceList[key]; allowed {
				minionAnnotations[key] = val
			}
		}
	}
}

// GenerateNginxMainConfig generate NginxMainConfig from Config
func GenerateNginxMainConfig(config *Config) *NginxMainConfig {
	nginxCfg := &NginxMainConfig{
		MainSnippets:              config.MainMainSnippets,
		HTTPSnippets:              config.MainHTTPSnippets,
		StreamSnippets:            config.MainStreamSnippets,
		ServerNamesHashBucketSize: config.MainServerNamesHashBucketSize,
		ServerNamesHashMaxSize:    config.MainServerNamesHashMaxSize,
		LogFormat:                 config.MainLogFormat,
		ErrorLogLevel:             config.MainErrorLogLevel,
		StreamLogFormat:           config.MainStreamLogFormat,
		SSLProtocols:              config.MainServerSSLProtocols,
		SSLCiphers:                config.MainServerSSLCiphers,
		SSLDHParam:                config.MainServerSSLDHParam,
		SSLPreferServerCiphers:    config.MainServerSSLPreferServerCiphers,
		HTTP2:                 config.HTTP2,
		ServerTokens:          config.ServerTokens,
		ProxyProtocol:         config.ProxyProtocol,
		WorkerProcesses:       config.MainWorkerProcesses,
		WorkerCPUAffinity:     config.MainWorkerCPUAffinity,
		WorkerShutdownTimeout: config.MainWorkerShutdownTimeout,
		WorkerConnections:     config.MainWorkerConnections,
		WorkerRlimitNofile:    config.MainWorkerRlimitNofile,
	}
	return nginxCfg
}

// UpdateConfig updates NGINX Configuration parameters
func (cnf *Configurator) UpdateConfig(config *Config, ingExes []*IngressEx, mergeableIngs map[string]*MergeableIngresses) error {
	cnf.config = config
	if cnf.config.MainServerSSLDHParamFileContent != nil {
		fileName, err := cnf.nginx.AddOrUpdateDHParam(*cnf.config.MainServerSSLDHParamFileContent)
		if err != nil {
			return fmt.Errorf("Error when updating dhparams: %v", err)
		}
		config.MainServerSSLDHParam = fileName
	}

	if config.MainTemplate != nil {
		err := cnf.templateExecutor.UpdateMainTemplate(config.MainTemplate)
		if err != nil {
			return fmt.Errorf("Error when parsing the main template: %v", err)
		}
	}
	if config.IngressTemplate != nil {
		err := cnf.templateExecutor.UpdateIngressTemplate(config.IngressTemplate)
		if err != nil {
			return fmt.Errorf("Error when parsing the ingress template: %v", err)
		}
	}

	mainCfg := GenerateNginxMainConfig(config)

	mainCfgContent, err := cnf.templateExecutor.ExecuteMainConfigTemplate(mainCfg)
	if err != nil {
		return fmt.Errorf("Error when writing main Config")
	}
	cnf.nginx.UpdateMainConfigFile(mainCfgContent)

	for _, ingEx := range ingExes {
		if err := cnf.addOrUpdateIngress(ingEx); err != nil {
			return err
		}
	}
	for _, mergeableIng := range mergeableIngs {
		if err := cnf.addOrUpdateMergeableIngress(mergeableIng); err != nil {
			return err
		}
	}

	if err := cnf.nginx.Reload(); err != nil {
		return fmt.Errorf("Error when updating config from ConfigMap: %v", err)
	}

	return nil
}

func (cnf *Configurator) isPlus() bool {
	return cnf.nginxAPI != nil
}

func keyToFileName(key string) string {
	return strings.Replace(key, "/", "-", -1)
}

func objectMetaToFileName(meta *meta_v1.ObjectMeta) string {
	return meta.Namespace + "-" + meta.Name
}

// HasIngress checks if the Ingress resource is present in NGINX configuration
func (cnf *Configurator) HasIngress(ing *extensions.Ingress) bool {
	name := objectMetaToFileName(&ing.ObjectMeta)
	_, exists := cnf.ingresses[name]
	return exists
}

// HasMinion checks if the minion Ingress resource of the master is present in NGINX configuration
func (cnf *Configurator) HasMinion(master *extensions.Ingress, minion *extensions.Ingress) bool {
	masterName := objectMetaToFileName(&master.ObjectMeta)
	if _, exists := cnf.minions[masterName]; !exists {
		return false
	}
	return cnf.minions[masterName][objectMetaToFileName(&minion.ObjectMeta)]
}

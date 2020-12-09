package configs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

// JWTKeyAnnotation is the annotation where the Secret with a JWK is specified.
const JWTKeyAnnotation = "nginx.com/jwt-key"

// AppProtectPolicyAnnotation is where the NGINX App Protect policy is specified
const AppProtectPolicyAnnotation = "appprotect.f5.com/app-protect-policy"

// AppProtectLogConfAnnotation is where the NGINX AppProtect Log Configuration is specified
const AppProtectLogConfAnnotation = "appprotect.f5.com/app-protect-security-log"

// AppProtectLogConfDstAnnotation is where the NGINX AppProtect Log Configuration is specified
const AppProtectLogConfDstAnnotation = "appprotect.f5.com/app-protect-security-log-destination"

// nginxMeshInternalRoute specifies if the ingress resource is an internal route.
const nginxMeshInternalRouteAnnotation = "nsm.nginx.com/internal-route"

var masterBlacklist = map[string]bool{
	"nginx.org/rewrites":                      true,
	"nginx.org/ssl-services":                  true,
	"nginx.org/grpc-services":                 true,
	"nginx.org/websocket-services":            true,
	"nginx.com/sticky-cookie-services":        true,
	"nginx.com/health-checks":                 true,
	"nginx.com/health-checks-mandatory":       true,
	"nginx.com/health-checks-mandatory-queue": true,
}

var minionBlacklist = map[string]bool{
	"nginx.org/proxy-hide-headers":                      true,
	"nginx.org/proxy-pass-headers":                      true,
	"nginx.org/redirect-to-https":                       true,
	"ingress.kubernetes.io/ssl-redirect":                true,
	"nginx.org/hsts":                                    true,
	"nginx.org/hsts-max-age":                            true,
	"nginx.org/hsts-include-subdomains":                 true,
	"nginx.org/server-tokens":                           true,
	"nginx.org/listen-ports":                            true,
	"nginx.org/listen-ports-ssl":                        true,
	"nginx.org/server-snippets":                         true,
	"appprotect.f5.com/app_protect_enable":              true,
	"appprotect.f5.com/app_protect_policy":              true,
	"appprotect.f5.com/app_protect_security_log_enable": true,
	"appprotect.f5.com/app_protect_security_log":        true,
}

var minionInheritanceList = map[string]bool{
	"nginx.org/proxy-connect-timeout":    true,
	"nginx.org/proxy-read-timeout":       true,
	"nginx.org/proxy-send-timeout":       true,
	"nginx.org/client-max-body-size":     true,
	"nginx.org/proxy-buffering":          true,
	"nginx.org/proxy-buffers":            true,
	"nginx.org/proxy-buffer-size":        true,
	"nginx.org/proxy-max-temp-file-size": true,
	"nginx.org/upstream-zone-size":       true,
	"nginx.org/location-snippets":        true,
	"nginx.org/lb-method":                true,
	"nginx.org/keepalive":                true,
	"nginx.org/max-fails":                true,
	"nginx.org/max-conns":                true,
	"nginx.org/fail-timeout":             true,
}

func parseAnnotations(ingEx *IngressEx, baseCfgParams *ConfigParams, isPlus bool, hasAppProtect bool, enableInternalRoutes bool) ConfigParams {
	cfgParams := *baseCfgParams

	if lbMethod, exists := ingEx.Ingress.Annotations["nginx.org/lb-method"]; exists {
		if isPlus {
			if parsedMethod, err := ParseLBMethodForPlus(lbMethod); err != nil {
				glog.Errorf("Ingress %s/%s: Invalid value for the nginx.org/lb-method: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), lbMethod, err)
			} else {
				cfgParams.LBMethod = parsedMethod
			}
		} else {
			if parsedMethod, err := ParseLBMethod(lbMethod); err != nil {
				glog.Errorf("Ingress %s/%s: Invalid value for the nginx.org/lb-method: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), lbMethod, err)
			} else {
				cfgParams.LBMethod = parsedMethod
			}
		}
	}

	if healthCheckEnabled, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.com/health-checks", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		}
		if isPlus {
			cfgParams.HealthCheckEnabled = healthCheckEnabled
		} else {
			glog.Warning("Annotation 'nginx.com/health-checks' requires NGINX Plus")
		}
	}

	if cfgParams.HealthCheckEnabled {
		if healthCheckMandatory, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.com/health-checks-mandatory", ingEx.Ingress); exists {
			if err != nil {
				glog.Error(err)
			}
			cfgParams.HealthCheckMandatory = healthCheckMandatory
		}
	}

	if cfgParams.HealthCheckMandatory {
		if healthCheckQueue, exists, err := GetMapKeyAsInt64(ingEx.Ingress.Annotations, "nginx.com/health-checks-mandatory-queue", ingEx.Ingress); exists {
			if err != nil {
				glog.Error(err)
			}
			cfgParams.HealthCheckMandatoryQueue = healthCheckQueue
		}
	}

	if slowStart, exists := ingEx.Ingress.Annotations["nginx.com/slow-start"]; exists {
		if parsedSlowStart, err := ParseTime(slowStart); err != nil {
			glog.Errorf("Ingress %s/%s: Invalid value nginx.org/slow-start: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), slowStart, err)
		} else {
			if isPlus {
				cfgParams.SlowStart = parsedSlowStart
			} else {
				glog.Warning("Annotation 'nginx.com/slow-start' requires NGINX Plus")
			}
		}
	}

	if serverTokens, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/server-tokens", ingEx.Ingress); exists {
		if err != nil {
			if isPlus {
				cfgParams.ServerTokens = ingEx.Ingress.Annotations["nginx.org/server-tokens"]
			} else {
				glog.Error(err)
			}
		} else {
			cfgParams.ServerTokens = "off"
			if serverTokens {
				cfgParams.ServerTokens = "on"
			}
		}
	}

	if serverSnippets, exists, err := GetMapKeyAsStringSlice(ingEx.Ingress.Annotations, "nginx.org/server-snippets", ingEx.Ingress, "\n"); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.ServerSnippets = serverSnippets
		}
	}

	if locationSnippets, exists, err := GetMapKeyAsStringSlice(ingEx.Ingress.Annotations, "nginx.org/location-snippets", ingEx.Ingress, "\n"); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.LocationSnippets = locationSnippets
		}
	}

	if proxyConnectTimeout, exists := ingEx.Ingress.Annotations["nginx.org/proxy-connect-timeout"]; exists {
		cfgParams.ProxyConnectTimeout = proxyConnectTimeout
	}

	if proxyReadTimeout, exists := ingEx.Ingress.Annotations["nginx.org/proxy-read-timeout"]; exists {
		cfgParams.ProxyReadTimeout = proxyReadTimeout
	}

	if proxySendTimeout, exists := ingEx.Ingress.Annotations["nginx.org/proxy-send-timeout"]; exists {
		cfgParams.ProxySendTimeout = proxySendTimeout
	}

	if proxyHideHeaders, exists, err := GetMapKeyAsStringSlice(ingEx.Ingress.Annotations, "nginx.org/proxy-hide-headers", ingEx.Ingress, ","); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.ProxyHideHeaders = proxyHideHeaders
		}
	}

	if proxyPassHeaders, exists, err := GetMapKeyAsStringSlice(ingEx.Ingress.Annotations, "nginx.org/proxy-pass-headers", ingEx.Ingress, ","); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.ProxyPassHeaders = proxyPassHeaders
		}
	}

	if clientMaxBodySize, exists := ingEx.Ingress.Annotations["nginx.org/client-max-body-size"]; exists {
		cfgParams.ClientMaxBodySize = clientMaxBodySize
	}

	if redirectToHTTPS, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/redirect-to-https", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.RedirectToHTTPS = redirectToHTTPS
		}
	}

	if sslRedirect, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "ingress.kubernetes.io/ssl-redirect", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.SSLRedirect = sslRedirect
		}
	}

	if proxyBuffering, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/proxy-buffering", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.ProxyBuffering = proxyBuffering
		}
	}

	if hsts, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/hsts", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			parsingErrors := false

			hstsMaxAge, existsMA, err := GetMapKeyAsInt64(ingEx.Ingress.Annotations, "nginx.org/hsts-max-age", ingEx.Ingress)
			if existsMA && err != nil {
				glog.Error(err)
				parsingErrors = true
			}
			hstsIncludeSubdomains, existsIS, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/hsts-include-subdomains", ingEx.Ingress)
			if existsIS && err != nil {
				glog.Error(err)
				parsingErrors = true
			}
			hstsBehindProxy, existsBP, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "nginx.org/hsts-behind-proxy", ingEx.Ingress)
			if existsBP && err != nil {
				glog.Error(err)
				parsingErrors = true
			}

			if parsingErrors {
				glog.Errorf("Ingress %s/%s: There are configuration issues with hsts annotations, skipping annotions for all hsts settings", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName())
			} else {
				cfgParams.HSTS = hsts
				if existsMA {
					cfgParams.HSTSMaxAge = hstsMaxAge
				}
				if existsIS {
					cfgParams.HSTSIncludeSubdomains = hstsIncludeSubdomains
				}
				if existsBP {
					cfgParams.HSTSBehindProxy = hstsBehindProxy
				}
			}
		}
	}

	if proxyBuffers, exists := ingEx.Ingress.Annotations["nginx.org/proxy-buffers"]; exists {
		cfgParams.ProxyBuffers = proxyBuffers
	}

	if proxyBufferSize, exists := ingEx.Ingress.Annotations["nginx.org/proxy-buffer-size"]; exists {
		cfgParams.ProxyBufferSize = proxyBufferSize
	}

	if upstreamZoneSize, exists := ingEx.Ingress.Annotations["nginx.org/upstream-zone-size"]; exists {
		cfgParams.UpstreamZoneSize = upstreamZoneSize
	}

	if proxyMaxTempFileSize, exists := ingEx.Ingress.Annotations["nginx.org/proxy-max-temp-file-size"]; exists {
		cfgParams.ProxyMaxTempFileSize = proxyMaxTempFileSize
	}

	if isPlus {
		if jwtRealm, exists := ingEx.Ingress.Annotations["nginx.com/jwt-realm"]; exists {
			cfgParams.JWTRealm = jwtRealm
		}
		if jwtKey, exists := ingEx.Ingress.Annotations[JWTKeyAnnotation]; exists {
			cfgParams.JWTKey = jwtKey
		}
		if jwtToken, exists := ingEx.Ingress.Annotations["nginx.com/jwt-token"]; exists {
			cfgParams.JWTToken = jwtToken
		}
		if jwtLoginURL, exists := ingEx.Ingress.Annotations["nginx.com/jwt-login-url"]; exists {
			cfgParams.JWTLoginURL = jwtLoginURL
		}
	}

	ports, sslPorts := getServicesPorts(ingEx)
	if len(ports) > 0 {
		cfgParams.Ports = ports
	}

	if len(sslPorts) > 0 {
		cfgParams.SSLPorts = sslPorts
	}

	if keepalive, exists, err := GetMapKeyAsInt(ingEx.Ingress.Annotations, "nginx.org/keepalive", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.Keepalive = keepalive
		}
	}

	if maxFails, exists, err := GetMapKeyAsInt(ingEx.Ingress.Annotations, "nginx.org/max-fails", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.MaxFails = maxFails
		}
	}

	if maxConns, exists, err := GetMapKeyAsInt(ingEx.Ingress.Annotations, "nginx.org/max-conns", ingEx.Ingress); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.MaxConns = maxConns
		}
	}

	if failTimeout, exists := ingEx.Ingress.Annotations["nginx.org/fail-timeout"]; exists {
		cfgParams.FailTimeout = failTimeout
	}

	if hasAppProtect {
		if appProtectEnable, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "appprotect.f5.com/app-protect-enable", ingEx.Ingress); exists {
			if err != nil {
				glog.Error(err)
			} else {
				if appProtectEnable {
					cfgParams.AppProtectEnable = "on"
				} else {
					cfgParams.AppProtectEnable = "off"
				}
			}
		}

		if appProtectLogEnable, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, "appprotect.f5.com/app-protect-security-log-enable", ingEx.Ingress); exists {
			if err != nil {
				glog.Error(err)
			} else {
				if appProtectLogEnable {
					cfgParams.AppProtectLogEnable = "on"
				} else {
					cfgParams.AppProtectLogEnable = "off"
				}
			}
		}

	}
	if enableInternalRoutes {
		if spiffeServerCerts, exists, err := GetMapKeyAsBool(ingEx.Ingress.Annotations, nginxMeshInternalRouteAnnotation, ingEx.Ingress); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfgParams.SpiffeServerCerts = spiffeServerCerts
			}
		}
	}
	return cfgParams
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

func filterMasterAnnotations(annotations map[string]string) []string {
	var removedAnnotations []string

	for key := range annotations {
		if _, notAllowed := masterBlacklist[key]; notAllowed {
			removedAnnotations = append(removedAnnotations, key)
			delete(annotations, key)
		}
	}

	return removedAnnotations
}

func filterMinionAnnotations(annotations map[string]string) []string {
	var removedAnnotations []string

	for key := range annotations {
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

func parsePort(value string) (int, error) {
	port, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return 0, fmt.Errorf(
			"Unable to parse port as integer: %s",
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

func parseRewrites(service string) (serviceName string, rewrite string, err error) {
	parts := strings.SplitN(strings.TrimSpace(service), " ", 2)

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

package configs

import (
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
		if parsedProxyConnectTimeout, err := ParseTime(proxyConnectTimeout); err != nil {
			glog.Errorf("Ingress %s/%s: Invalid value nginx.org/proxy-connect-timeout: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), proxyConnectTimeout, err)
		} else {
			cfgParams.ProxyConnectTimeout = parsedProxyConnectTimeout
		}
	}

	if proxyReadTimeout, exists := ingEx.Ingress.Annotations["nginx.org/proxy-read-timeout"]; exists {
		if parsedProxyReadTimeout, err := ParseTime(proxyReadTimeout); err != nil {
			glog.Errorf("Ingress %s/%s: Invalid value nginx.org/proxy-read-timeout: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), proxyReadTimeout, err)
		} else {
			cfgParams.ProxyReadTimeout = parsedProxyReadTimeout
		}
	}

	if proxySendTimeout, exists := ingEx.Ingress.Annotations["nginx.org/proxy-send-timeout"]; exists {
		if parsedProxySendTimeout, err := ParseTime(proxySendTimeout); err != nil {
			glog.Errorf("Ingress %s/%s: Invalid value nginx.org/proxy-send-timeout: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), proxySendTimeout, err)
		} else {
			cfgParams.ProxySendTimeout = parsedProxySendTimeout
		}
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

	if values, exists := ingEx.Ingress.Annotations["nginx.org/listen-ports"]; exists {
		ports, err := ParsePortList(values)
		if err != nil {
			glog.Errorf("In %v nginx.org/listen-ports contains invalid declaration: %v, ignoring", ingEx.Ingress.Name, err)
		}
		if len(ports) > 0 {
			cfgParams.Ports = ports
		}
	}

	if values, exists := ingEx.Ingress.Annotations["nginx.org/listen-ports-ssl"]; exists {
		sslPorts, err := ParsePortList(values)
		if err != nil {
			glog.Errorf("In %v nginx.org/listen-ports-ssl contains invalid declaration: %v, ignoring", ingEx.Ingress.Name, err)
		}
		if len(sslPorts) > 0 {
			cfgParams.SSLPorts = sslPorts
		}
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
		if parsedFailTimeout, err := ParseTime(failTimeout); err != nil {
			glog.Errorf("Ingress %s/%s: Invalid value nginx.org/fail-timeout: got %q: %v", ingEx.Ingress.GetNamespace(), ingEx.Ingress.GetName(), failTimeout, err)
		} else {
			cfgParams.FailTimeout = parsedFailTimeout
		}
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
	if value, exists := ingEx.Ingress.Annotations["nginx.org/websocket-services"]; exists {
		return ParseServiceList(value)
	}
	return nil
}

func getRewrites(ingEx *IngressEx) map[string]string {
	if value, exists := ingEx.Ingress.Annotations["nginx.org/rewrites"]; exists {
		rewrites, err := ParseRewriteList(value)
		if err != nil {
			glog.Error(err)
		}
		return rewrites
	}
	return nil
}

func getSSLServices(ingEx *IngressEx) map[string]bool {
	if value, exists := ingEx.Ingress.Annotations["nginx.org/ssl-services"]; exists {
		return ParseServiceList(value)
	}
	return nil
}

func getGrpcServices(ingEx *IngressEx) map[string]bool {
	if value, exists := ingEx.Ingress.Annotations["nginx.org/grpc-services"]; exists {
		return ParseServiceList(value)
	}
	return nil
}

func getSessionPersistenceServices(ingEx *IngressEx) map[string]string {
	if value, exists := ingEx.Ingress.Annotations["nginx.com/sticky-cookie-services"]; exists {
		services, err := ParseStickyServiceList(value)
		if err != nil {
			glog.Error(err)
		}
		return services
	}
	return nil
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

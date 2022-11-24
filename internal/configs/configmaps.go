package configs

import (
	"strings"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
)

// ParseConfigMap parses ConfigMap into ConfigParams.
//
//nolint:gocyclo
func ParseConfigMap(cfgm *v1.ConfigMap, nginxPlus bool, hasAppProtect bool, hasAppProtectDos bool, hasTLSPassthrough bool) *ConfigParams {
	cfgParams := NewDefaultConfigParams(nginxPlus)

	if serverTokens, exists, err := GetMapKeyAsBool(cfgm.Data, "server-tokens", cfgm); exists {
		if err != nil {
			if nginxPlus {
				cfgParams.ServerTokens = cfgm.Data["server-tokens"]
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

	if lbMethod, exists := cfgm.Data["lb-method"]; exists {
		if nginxPlus {
			if parsedMethod, err := ParseLBMethodForPlus(lbMethod); err != nil {
				glog.Errorf("Configmap %s/%s: Invalid value for the lb-method key: got %q: %v", cfgm.GetNamespace(), cfgm.GetName(), lbMethod, err)
			} else {
				cfgParams.LBMethod = parsedMethod
			}
		} else {
			if parsedMethod, err := ParseLBMethod(lbMethod); err != nil {
				glog.Errorf("Configmap %s/%s: Invalid value for the lb-method key: got %q: %v", cfgm.GetNamespace(), cfgm.GetName(), lbMethod, err)
			} else {
				cfgParams.LBMethod = parsedMethod
			}
		}
	}

	if proxyConnectTimeout, exists := cfgm.Data["proxy-connect-timeout"]; exists {
		cfgParams.ProxyConnectTimeout = proxyConnectTimeout
	}

	if proxyReadTimeout, exists := cfgm.Data["proxy-read-timeout"]; exists {
		cfgParams.ProxyReadTimeout = proxyReadTimeout
	}

	if proxySendTimeout, exists := cfgm.Data["proxy-send-timeout"]; exists {
		cfgParams.ProxySendTimeout = proxySendTimeout
	}

	if proxyHideHeaders, exists := GetMapKeyAsStringSlice(cfgm.Data, "proxy-hide-headers", cfgm, ","); exists {
		cfgParams.ProxyHideHeaders = proxyHideHeaders
	}

	if proxyPassHeaders, exists := GetMapKeyAsStringSlice(cfgm.Data, "proxy-pass-headers", cfgm, ","); exists {
		cfgParams.ProxyPassHeaders = proxyPassHeaders
	}

	if clientMaxBodySize, exists := cfgm.Data["client-max-body-size"]; exists {
		cfgParams.ClientMaxBodySize = clientMaxBodySize
	}

	if serverNamesHashBucketSize, exists := cfgm.Data["server-names-hash-bucket-size"]; exists {
		cfgParams.MainServerNamesHashBucketSize = serverNamesHashBucketSize
	}

	if serverNamesHashMaxSize, exists := cfgm.Data["server-names-hash-max-size"]; exists {
		cfgParams.MainServerNamesHashMaxSize = serverNamesHashMaxSize
	}

	if mapHashBucketSize, exists := cfgm.Data["map-hash-bucket-size"]; exists {
		cfgParams.MainMapHashBucketSize = mapHashBucketSize
	}

	if mapHashMaxSize, exists := cfgm.Data["map-hash-max-size"]; exists {
		cfgParams.MainMapHashMaxSize = mapHashMaxSize
	}

	if HTTP2, exists, err := GetMapKeyAsBool(cfgm.Data, "http2", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.HTTP2 = HTTP2
		}
	}

	if redirectToHTTPS, exists, err := GetMapKeyAsBool(cfgm.Data, "redirect-to-https", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.RedirectToHTTPS = redirectToHTTPS
		}
	}

	if sslRedirect, exists, err := GetMapKeyAsBool(cfgm.Data, "ssl-redirect", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.SSLRedirect = sslRedirect
		}
	}

	if hsts, exists, err := GetMapKeyAsBool(cfgm.Data, "hsts", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			parsingErrors := false

			hstsMaxAge, existsMA, err := GetMapKeyAsInt64(cfgm.Data, "hsts-max-age", cfgm)
			if existsMA && err != nil {
				glog.Error(err)
				parsingErrors = true
			}
			hstsIncludeSubdomains, existsIS, err := GetMapKeyAsBool(cfgm.Data, "hsts-include-subdomains", cfgm)
			if existsIS && err != nil {
				glog.Error(err)
				parsingErrors = true
			}
			hstsBehindProxy, existsBP, err := GetMapKeyAsBool(cfgm.Data, "hsts-behind-proxy", cfgm)
			if existsBP && err != nil {
				glog.Error(err)
				parsingErrors = true
			}

			if parsingErrors {
				glog.Errorf("Configmap %s/%s: There are configuration issues with hsts annotations, skipping options for all hsts settings", cfgm.GetNamespace(), cfgm.GetName())
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

	if proxyProtocol, exists, err := GetMapKeyAsBool(cfgm.Data, "proxy-protocol", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.ProxyProtocol = proxyProtocol
		}
	}

	if realIPHeader, exists := cfgm.Data["real-ip-header"]; exists {
		if hasTLSPassthrough {
			msg := "Configmap %s/%s: key real-ip-header is ignored, directive real_ip_header is automatically set to 'proxy_protocol' when TLS passthrough is enabled."
			if realIPHeader == "proxy_protocol" {
				glog.Infof(msg, cfgm.GetNamespace(), cfgm.GetName())
			} else {
				glog.Errorf(msg, cfgm.GetNamespace(), cfgm.GetName())
			}
		} else {
			cfgParams.RealIPHeader = realIPHeader
		}
	}

	if setRealIPFrom, exists := GetMapKeyAsStringSlice(cfgm.Data, "set-real-ip-from", cfgm, ","); exists {
		cfgParams.SetRealIPFrom = setRealIPFrom
	}

	if realIPRecursive, exists, err := GetMapKeyAsBool(cfgm.Data, "real-ip-recursive", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.RealIPRecursive = realIPRecursive
		}
	}

	if sslProtocols, exists := cfgm.Data["ssl-protocols"]; exists {
		cfgParams.MainServerSSLProtocols = sslProtocols
	}

	if sslPreferServerCiphers, exists, err := GetMapKeyAsBool(cfgm.Data, "ssl-prefer-server-ciphers", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.MainServerSSLPreferServerCiphers = sslPreferServerCiphers
		}
	}

	if sslCiphers, exists := cfgm.Data["ssl-ciphers"]; exists {
		cfgParams.MainServerSSLCiphers = strings.Trim(sslCiphers, "\n")
	}

	if sslDHParamFile, exists := cfgm.Data["ssl-dhparam-file"]; exists {
		sslDHParamFile = strings.Trim(sslDHParamFile, "\n")
		cfgParams.MainServerSSLDHParamFileContent = &sslDHParamFile
	}

	if errorLogLevel, exists := cfgm.Data["error-log-level"]; exists {
		cfgParams.MainErrorLogLevel = errorLogLevel
	}

	if accessLogOff, exists, err := GetMapKeyAsBool(cfgm.Data, "access-log-off", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.MainAccessLogOff = accessLogOff
		}
	}

	if logFormat, exists := GetMapKeyAsStringSlice(cfgm.Data, "log-format", cfgm, "\n"); exists {
		cfgParams.MainLogFormat = logFormat
	}

	if logFormatEscaping, exists := cfgm.Data["log-format-escaping"]; exists {
		logFormatEscaping = strings.TrimSpace(logFormatEscaping)
		if logFormatEscaping != "" {
			cfgParams.MainLogFormatEscaping = logFormatEscaping
		}
	}

	if streamLogFormat, exists := GetMapKeyAsStringSlice(cfgm.Data, "stream-log-format", cfgm, "\n"); exists {
		cfgParams.MainStreamLogFormat = streamLogFormat
	}

	if streamLogFormatEscaping, exists := cfgm.Data["stream-log-format-escaping"]; exists {
		streamLogFormatEscaping = strings.TrimSpace(streamLogFormatEscaping)
		if streamLogFormatEscaping != "" {
			cfgParams.MainStreamLogFormatEscaping = streamLogFormatEscaping
		}
	}

	if defaultServerAccessLogOff, exists, err := GetMapKeyAsBool(cfgm.Data, "default-server-access-log-off", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.DefaultServerAccessLogOff = defaultServerAccessLogOff
		}
	}

	if defaultServerReturn, exists := cfgm.Data["default-server-return"]; exists {
		cfgParams.DefaultServerReturn = defaultServerReturn
	}

	if proxyBuffering, exists, err := GetMapKeyAsBool(cfgm.Data, "proxy-buffering", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.ProxyBuffering = proxyBuffering
		}
	}

	if proxyBuffers, exists := cfgm.Data["proxy-buffers"]; exists {
		cfgParams.ProxyBuffers = proxyBuffers
	}

	if proxyBufferSize, exists := cfgm.Data["proxy-buffer-size"]; exists {
		cfgParams.ProxyBufferSize = proxyBufferSize
	}

	if proxyMaxTempFileSize, exists := cfgm.Data["proxy-max-temp-file-size"]; exists {
		cfgParams.ProxyMaxTempFileSize = proxyMaxTempFileSize
	}

	if mainMainSnippets, exists := GetMapKeyAsStringSlice(cfgm.Data, "main-snippets", cfgm, "\n"); exists {
		cfgParams.MainMainSnippets = mainMainSnippets
	}

	if mainHTTPSnippets, exists := GetMapKeyAsStringSlice(cfgm.Data, "http-snippets", cfgm, "\n"); exists {
		cfgParams.MainHTTPSnippets = mainHTTPSnippets
	}

	if locationSnippets, exists := GetMapKeyAsStringSlice(cfgm.Data, "location-snippets", cfgm, "\n"); exists {
		cfgParams.LocationSnippets = locationSnippets
	}

	if serverSnippets, exists := GetMapKeyAsStringSlice(cfgm.Data, "server-snippets", cfgm, "\n"); exists {
		cfgParams.ServerSnippets = serverSnippets
	}

	if _, exists, err := GetMapKeyAsInt(cfgm.Data, "worker-processes", cfgm); exists {
		if err != nil && cfgm.Data["worker-processes"] != "auto" {
			glog.Errorf("Configmap %s/%s: Invalid value for worker-processes key: must be an integer or the string 'auto', got %q", cfgm.GetNamespace(), cfgm.GetName(), cfgm.Data["worker-processes"])
		} else {
			cfgParams.MainWorkerProcesses = cfgm.Data["worker-processes"]
		}
	}

	if workerCPUAffinity, exists := cfgm.Data["worker-cpu-affinity"]; exists {
		cfgParams.MainWorkerCPUAffinity = workerCPUAffinity
	}

	if workerShutdownTimeout, exists := cfgm.Data["worker-shutdown-timeout"]; exists {
		cfgParams.MainWorkerShutdownTimeout = workerShutdownTimeout
	}

	if workerConnections, exists := cfgm.Data["worker-connections"]; exists {
		cfgParams.MainWorkerConnections = workerConnections
	}

	if workerRlimitNofile, exists := cfgm.Data["worker-rlimit-nofile"]; exists {
		cfgParams.MainWorkerRlimitNofile = workerRlimitNofile
	}

	if keepalive, exists, err := GetMapKeyAsInt(cfgm.Data, "keepalive", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.Keepalive = keepalive
		}
	}

	if maxFails, exists, err := GetMapKeyAsInt(cfgm.Data, "max-fails", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.MaxFails = maxFails
		}
	}

	if upstreamZoneSize, exists := cfgm.Data["upstream-zone-size"]; exists {
		cfgParams.UpstreamZoneSize = upstreamZoneSize
	}

	if failTimeout, exists := cfgm.Data["fail-timeout"]; exists {
		cfgParams.FailTimeout = failTimeout
	}

	if mainTemplate, exists := cfgm.Data["main-template"]; exists {
		cfgParams.MainTemplate = &mainTemplate
	}

	if ingressTemplate, exists := cfgm.Data["ingress-template"]; exists {
		cfgParams.IngressTemplate = &ingressTemplate
	}

	if virtualServerTemplate, exists := cfgm.Data["virtualserver-template"]; exists {
		cfgParams.VirtualServerTemplate = &virtualServerTemplate
	}

	if mainStreamSnippets, exists := GetMapKeyAsStringSlice(cfgm.Data, "stream-snippets", cfgm, "\n"); exists {
		cfgParams.MainStreamSnippets = mainStreamSnippets
	}

	if resolverAddresses, exists := GetMapKeyAsStringSlice(cfgm.Data, "resolver-addresses", cfgm, ","); exists {
		if nginxPlus {
			cfgParams.ResolverAddresses = resolverAddresses
		} else {
			glog.Warning("ConfigMap key 'resolver-addresses' requires NGINX Plus")
		}
	}

	if resolverIpv6, exists, err := GetMapKeyAsBool(cfgm.Data, "resolver-ipv6", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			if nginxPlus {
				cfgParams.ResolverIPV6 = resolverIpv6
			} else {
				glog.Warning("ConfigMap key 'resolver-ipv6' requires NGINX Plus")
			}
		}
	}

	if resolverValid, exists := cfgm.Data["resolver-valid"]; exists {
		if nginxPlus {
			cfgParams.ResolverValid = resolverValid
		} else {
			glog.Warning("ConfigMap key 'resolver-valid' requires NGINX Plus")
		}
	}

	if resolverTimeout, exists := cfgm.Data["resolver-timeout"]; exists {
		if nginxPlus {
			cfgParams.ResolverTimeout = resolverTimeout
		} else {
			glog.Warning("ConfigMap key 'resolver-timeout' requires NGINX Plus")
		}
	}

	if keepaliveTimeout, exists := cfgm.Data["keepalive-timeout"]; exists {
		cfgParams.MainKeepaliveTimeout = keepaliveTimeout
	}

	if keepaliveRequests, exists, err := GetMapKeyAsInt64(cfgm.Data, "keepalive-requests", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.MainKeepaliveRequests = keepaliveRequests
		}
	}

	if varHashBucketSize, exists, err := GetMapKeyAsUint64(cfgm.Data, "variables-hash-bucket-size", cfgm, true); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.VariablesHashBucketSize = varHashBucketSize
		}
	}

	if varHashMaxSize, exists, err := GetMapKeyAsUint64(cfgm.Data, "variables-hash-max-size", cfgm, false); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfgParams.VariablesHashMaxSize = varHashMaxSize
		}
	}

	if openTracingTracer, exists := cfgm.Data["opentracing-tracer"]; exists {
		cfgParams.MainOpenTracingTracer = openTracingTracer
	}

	if openTracingTracerConfig, exists := cfgm.Data["opentracing-tracer-config"]; exists {
		cfgParams.MainOpenTracingTracerConfig = openTracingTracerConfig
	}

	if cfgParams.MainOpenTracingTracer != "" || cfgParams.MainOpenTracingTracerConfig != "" {
		cfgParams.MainOpenTracingLoadModule = true
	}

	if openTracing, exists, err := GetMapKeyAsBool(cfgm.Data, "opentracing", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			if cfgParams.MainOpenTracingLoadModule {
				cfgParams.MainOpenTracingEnabled = openTracing
			} else {
				glog.Error("ConfigMap Key 'opentracing' requires both 'opentracing-tracer' and 'opentracing-tracer-config' Keys configured, Opentracing will be disabled")
			}
		}
	}

	if hasAppProtect {
		if appProtectFailureModeAction, exists := cfgm.Data["app-protect-failure-mode-action"]; exists {
			if appProtectFailureModeAction == "pass" || appProtectFailureModeAction == "drop" {
				cfgParams.MainAppProtectFailureModeAction = appProtectFailureModeAction
			} else {
				glog.Error("ConfigMap Key 'app-protect-failure-mode-action' must have value 'pass' or 'drop'. Ignoring.")
			}
		}

		if appProtectCompressedRequestsAction, exists := cfgm.Data["app-protect-compressed-requests-action"]; exists {
			if appProtectCompressedRequestsAction == "pass" || appProtectCompressedRequestsAction == "drop" {
				cfgParams.MainAppProtectCompressedRequestsAction = appProtectCompressedRequestsAction
			} else {
				glog.Error("ConfigMap Key 'app-protect-compressed-requests-action' must have value 'pass' or 'drop'. Ignoring.")
			}
		}

		if appProtectCookieSeed, exists := cfgm.Data["app-protect-cookie-seed"]; exists {
			cfgParams.MainAppProtectCookieSeed = appProtectCookieSeed
		}

		if appProtectCPUThresholds, exists := cfgm.Data["app-protect-cpu-thresholds"]; exists {
			if VerifyAppProtectThresholds(appProtectCPUThresholds) {
				cfgParams.MainAppProtectCPUThresholds = appProtectCPUThresholds
			} else {
				glog.Error("ConfigMap Key 'app-protect-cpu-thresholds' must follow pattern: 'high=<0 - 100> low=<0 - 100>'. Ignoring.")
			}
		}

		if appProtectPhysicalMemoryThresholds, exists := cfgm.Data["app-protect-physical-memory-util-thresholds"]; exists {
			cfgParams.MainAppProtectPhysicalMemoryThresholds = appProtectPhysicalMemoryThresholds
			if VerifyAppProtectThresholds(appProtectPhysicalMemoryThresholds) {
				cfgParams.MainAppProtectPhysicalMemoryThresholds = appProtectPhysicalMemoryThresholds
			} else {
				glog.Error("ConfigMap Key 'app-protect-physical-memory-thresholds' must follow pattern: 'high=<0 - 100> low=<0 - 100>'. Ignoring.")
			}
		}
		if appProtectReconnectPeriod, exists := cfgm.Data["app-protect-reconnect-period-seconds"]; exists {
			period, err := ParseFloat64(appProtectReconnectPeriod)
			if err == nil && period > 0 && period <= 60 {
				cfgParams.MainAppProtectReconnectPeriod = appProtectReconnectPeriod
			} else {
				glog.Error("ConfigMap Key 'app-protect-reconnect-period-second' must have value between '0' and '60'. '0' is illegal. Ignoring.")
			}
		}
	}

	if hasAppProtectDos {
		if appProtectDosLogFormat, exists := GetMapKeyAsStringSlice(cfgm.Data, "app-protect-dos-log-format", cfgm, "\n"); exists {
			cfgParams.MainAppProtectDosLogFormat = appProtectDosLogFormat
		}

		if appProtectDosLogFormatEscaping, exists := cfgm.Data["app-protect-dos-log-format-escaping"]; exists {
			appProtectDosLogFormatEscaping = strings.TrimSpace(appProtectDosLogFormatEscaping)
			if appProtectDosLogFormatEscaping != "" {
				cfgParams.MainAppProtectDosLogFormatEscaping = appProtectDosLogFormatEscaping
			}
		}

		if appProtectDosArbFqdn, exists := cfgm.Data["app-protect-dos-arb-fqdn"]; exists {
			appProtectDosArbFqdn = strings.TrimSpace(appProtectDosArbFqdn)
			if appProtectDosArbFqdn != "" {
				cfgParams.MainAppProtectDosArbFqdn = appProtectDosArbFqdn
			}
		}
	}

	return cfgParams
}

// GenerateNginxMainConfig generates MainConfig.
func GenerateNginxMainConfig(staticCfgParams *StaticConfigParams, config *ConfigParams) *version1.MainConfig {
	nginxCfg := &version1.MainConfig{
		AccessLogOff:                       config.MainAccessLogOff,
		DefaultServerAccessLogOff:          config.DefaultServerAccessLogOff,
		DefaultServerReturn:                config.DefaultServerReturn,
		DisableIPV6:                        staticCfgParams.DisableIPV6,
		ErrorLogLevel:                      config.MainErrorLogLevel,
		HealthStatus:                       staticCfgParams.HealthStatus,
		HealthStatusURI:                    staticCfgParams.HealthStatusURI,
		HTTP2:                              config.HTTP2,
		HTTPSnippets:                       config.MainHTTPSnippets,
		KeepaliveRequests:                  config.MainKeepaliveRequests,
		KeepaliveTimeout:                   config.MainKeepaliveTimeout,
		LogFormat:                          config.MainLogFormat,
		LogFormatEscaping:                  config.MainLogFormatEscaping,
		MainSnippets:                       config.MainMainSnippets,
		NginxStatus:                        staticCfgParams.NginxStatus,
		NginxStatusAllowCIDRs:              staticCfgParams.NginxStatusAllowCIDRs,
		NginxStatusPort:                    staticCfgParams.NginxStatusPort,
		OpenTracingEnabled:                 config.MainOpenTracingEnabled,
		OpenTracingLoadModule:              config.MainOpenTracingLoadModule,
		OpenTracingTracer:                  config.MainOpenTracingTracer,
		OpenTracingTracerConfig:            config.MainOpenTracingTracerConfig,
		ProxyProtocol:                      config.ProxyProtocol,
		ResolverAddresses:                  config.ResolverAddresses,
		ResolverIPV6:                       config.ResolverIPV6,
		ResolverTimeout:                    config.ResolverTimeout,
		ResolverValid:                      config.ResolverValid,
		RealIPHeader:                       config.RealIPHeader,
		RealIPRecursive:                    config.RealIPRecursive,
		SetRealIPFrom:                      config.SetRealIPFrom,
		ServerNamesHashBucketSize:          config.MainServerNamesHashBucketSize,
		ServerNamesHashMaxSize:             config.MainServerNamesHashMaxSize,
		MapHashBucketSize:                  config.MainMapHashBucketSize,
		MapHashMaxSize:                     config.MainMapHashMaxSize,
		ServerTokens:                       config.ServerTokens,
		SSLCiphers:                         config.MainServerSSLCiphers,
		SSLDHParam:                         config.MainServerSSLDHParam,
		SSLPreferServerCiphers:             config.MainServerSSLPreferServerCiphers,
		SSLProtocols:                       config.MainServerSSLProtocols,
		SSLRejectHandshake:                 staticCfgParams.SSLRejectHandshake,
		TLSPassthrough:                     staticCfgParams.TLSPassthrough,
		StreamLogFormat:                    config.MainStreamLogFormat,
		StreamLogFormatEscaping:            config.MainStreamLogFormatEscaping,
		StreamSnippets:                     config.MainStreamSnippets,
		StubStatusOverUnixSocketForOSS:     staticCfgParams.StubStatusOverUnixSocketForOSS,
		WorkerCPUAffinity:                  config.MainWorkerCPUAffinity,
		WorkerProcesses:                    config.MainWorkerProcesses,
		WorkerShutdownTimeout:              config.MainWorkerShutdownTimeout,
		WorkerConnections:                  config.MainWorkerConnections,
		WorkerRlimitNofile:                 config.MainWorkerRlimitNofile,
		VariablesHashBucketSize:            config.VariablesHashBucketSize,
		VariablesHashMaxSize:               config.VariablesHashMaxSize,
		AppProtectLoadModule:               staticCfgParams.MainAppProtectLoadModule,
		AppProtectDosLoadModule:            staticCfgParams.MainAppProtectDosLoadModule,
		AppProtectFailureModeAction:        config.MainAppProtectFailureModeAction,
		AppProtectCompressedRequestsAction: config.MainAppProtectCompressedRequestsAction,
		AppProtectCookieSeed:               config.MainAppProtectCookieSeed,
		AppProtectCPUThresholds:            config.MainAppProtectCPUThresholds,
		AppProtectPhysicalMemoryThresholds: config.MainAppProtectPhysicalMemoryThresholds,
		AppProtectReconnectPeriod:          config.MainAppProtectReconnectPeriod,
		AppProtectDosLogFormat:             config.MainAppProtectDosLogFormat,
		AppProtectDosLogFormatEscaping:     config.MainAppProtectDosLogFormatEscaping,
		AppProtectDosArbFqdn:               config.MainAppProtectDosArbFqdn,
		InternalRouteServer:                staticCfgParams.EnableInternalRoutes,
		InternalRouteServerName:            staticCfgParams.InternalRouteServerName,
		LatencyMetrics:                     staticCfgParams.EnableLatencyMetrics,
		OIDC:                               staticCfgParams.EnableOIDC,
	}
	return nginxCfg
}

package nginx

import (
	"strings"

	"github.com/golang/glog"

	api_v1 "k8s.io/api/core/v1"
)

// Config holds NGINX configuration parameters
type Config struct {
	LocationSnippets              []string
	ServerSnippets                []string
	ServerTokens                  string
	ProxyConnectTimeout           string
	ProxyReadTimeout              string
	ClientMaxBodySize             string
	HTTP2                         bool
	RedirectToHTTPS               bool
	SSLRedirect                   bool
	MainMainSnippets              []string
	MainHTTPSnippets              []string
	MainStreamSnippets            []string
	MainServerNamesHashBucketSize string
	MainServerNamesHashMaxSize    string
	MainLogFormat                 string
	MainErrorLogLevel             string
	MainStreamLogFormat           string
	ProxyBuffering                bool
	ProxyBuffers                  string
	ProxyBufferSize               string
	ProxyMaxTempFileSize          string
	ProxyProtocol                 bool
	ProxyHideHeaders              []string
	ProxyPassHeaders              []string
	HSTS                          bool
	HSTSMaxAge                    int64
	HSTSIncludeSubdomains         bool
	LBMethod                      string
	MainWorkerProcesses           string
	MainWorkerCPUAffinity         string
	MainWorkerShutdownTimeout     string
	MainWorkerConnections         string
	MainWorkerRlimitNofile        string
	Keepalive                     int64
	MaxFails                      int64
	FailTimeout                   string
	HealthCheckEnabled            bool
	HealthCheckMandatory          bool
	HealthCheckMandatoryQueue     int64
	SlowStart                     string

	// http://nginx.org/en/docs/http/ngx_http_realip_module.html
	RealIPHeader    string
	SetRealIPFrom   []string
	RealIPRecursive bool

	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html
	MainServerSSLProtocols           string
	MainServerSSLPreferServerCiphers bool
	MainServerSSLCiphers             string
	MainServerSSLDHParam             string
	MainServerSSLDHParamFileContent  *string

	MainTemplate    *string
	IngressTemplate *string

	JWTRealm    string
	JWTKey      string
	JWTToken    string
	JWTLoginURL string

	Ports    []int
	SSLPorts []int
}

// NewDefaultConfig creates a Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		ServerTokens:               "on",
		ProxyConnectTimeout:        "60s",
		ProxyReadTimeout:           "60s",
		ClientMaxBodySize:          "1m",
		SSLRedirect:                true,
		MainServerNamesHashMaxSize: "512",
		ProxyBuffering:             true,
		MainWorkerProcesses:        "auto",
		MainWorkerConnections:      "1024",
		HSTSMaxAge:                 2592000,
		Ports:                      []int{80},
		SSLPorts:                   []int{443},
		MaxFails:                   1,
		FailTimeout:                "10s",
		LBMethod:                   "least_conn",
		MainErrorLogLevel:          "notice",
	}
}

// ParseConfigMap Parse ConfigMap to Config
func ParseConfigMap(cfgm *api_v1.ConfigMap, nginxPlus bool) *Config {
	cfg := NewDefaultConfig()
	if serverTokens, exists, err := GetMapKeyAsBool(cfgm.Data, "server-tokens", cfgm); exists {
		if err != nil {
			if nginxPlus {
				cfg.ServerTokens = cfgm.Data["server-tokens"]
			} else {
				glog.Error(err)
			}
		} else {
			cfg.ServerTokens = "off"
			if serverTokens {
				cfg.ServerTokens = "on"
			}
		}
	}

	if lbMethod, exists := cfgm.Data["lb-method"]; exists {
		if nginxPlus {
			if parsedMethod, err := ParseLBMethodForPlus(lbMethod); err != nil {
				glog.Errorf("Configmap %s/%s: Invalid value for the lb-method key: got %q: %v", cfgm.GetNamespace(), cfgm.GetName(), lbMethod, err)
			} else {
				cfg.LBMethod = parsedMethod
			}
		} else {
			if parsedMethod, err := ParseLBMethod(lbMethod); err != nil {
				glog.Errorf("Configmap %s/%s: Invalid value for the lb-method key: got %q: %v", cfgm.GetNamespace(), cfgm.GetName(), lbMethod, err)
			} else {
				cfg.LBMethod = parsedMethod
			}
		}
	}

	if proxyConnectTimeout, exists := cfgm.Data["proxy-connect-timeout"]; exists {
		cfg.ProxyConnectTimeout = proxyConnectTimeout
	}
	if proxyReadTimeout, exists := cfgm.Data["proxy-read-timeout"]; exists {
		cfg.ProxyReadTimeout = proxyReadTimeout
	}
	if proxyHideHeaders, exists, err := GetMapKeyAsStringSlice(cfgm.Data, "proxy-hide-headers", cfgm, ","); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.ProxyHideHeaders = proxyHideHeaders
		}
	}
	if proxyPassHeaders, exists, err := GetMapKeyAsStringSlice(cfgm.Data, "proxy-pass-headers", cfgm, ","); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.ProxyPassHeaders = proxyPassHeaders
		}
	}
	if clientMaxBodySize, exists := cfgm.Data["client-max-body-size"]; exists {
		cfg.ClientMaxBodySize = clientMaxBodySize
	}
	if serverNamesHashBucketSize, exists := cfgm.Data["server-names-hash-bucket-size"]; exists {
		cfg.MainServerNamesHashBucketSize = serverNamesHashBucketSize
	}
	if serverNamesHashMaxSize, exists := cfgm.Data["server-names-hash-max-size"]; exists {
		cfg.MainServerNamesHashMaxSize = serverNamesHashMaxSize
	}
	if HTTP2, exists, err := GetMapKeyAsBool(cfgm.Data, "http2", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.HTTP2 = HTTP2
		}
	}
	if redirectToHTTPS, exists, err := GetMapKeyAsBool(cfgm.Data, "redirect-to-https", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.RedirectToHTTPS = redirectToHTTPS
		}
	}
	if sslRedirect, exists, err := GetMapKeyAsBool(cfgm.Data, "ssl-redirect", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.SSLRedirect = sslRedirect
		}
	}

	// HSTS block
	if hsts, exists, err := GetMapKeyAsBool(cfgm.Data, "hsts", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			parsingErrors := false

			hstsMaxAge, existsMA, err := GetMapKeyAsInt(cfgm.Data, "hsts-max-age", cfgm)
			if existsMA && err != nil {
				glog.Error(err)
				parsingErrors = true
			}
			hstsIncludeSubdomains, existsIS, err := GetMapKeyAsBool(cfgm.Data, "hsts-include-subdomains", cfgm)
			if existsIS && err != nil {
				glog.Error(err)
				parsingErrors = true
			}

			if parsingErrors {
				glog.Errorf("Configmap %s/%s: There are configuration issues with hsts annotations, skipping options for all hsts settings", cfgm.GetNamespace(), cfgm.GetName())
			} else {
				cfg.HSTS = hsts
				if existsMA {
					cfg.HSTSMaxAge = hstsMaxAge
				}
				if existsIS {
					cfg.HSTSIncludeSubdomains = hstsIncludeSubdomains
				}
			}
		}
	}

	if proxyProtocol, exists, err := GetMapKeyAsBool(cfgm.Data, "proxy-protocol", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.ProxyProtocol = proxyProtocol
		}
	}

	// ngx_http_realip_module
	if realIPHeader, exists := cfgm.Data["real-ip-header"]; exists {
		cfg.RealIPHeader = realIPHeader
	}
	if setRealIPFrom, exists, err := GetMapKeyAsStringSlice(cfgm.Data, "set-real-ip-from", cfgm, ","); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.SetRealIPFrom = setRealIPFrom
		}
	}
	if realIPRecursive, exists, err := GetMapKeyAsBool(cfgm.Data, "real-ip-recursive", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.RealIPRecursive = realIPRecursive
		}
	}

	// SSL block
	if sslProtocols, exists := cfgm.Data["ssl-protocols"]; exists {
		cfg.MainServerSSLProtocols = sslProtocols
	}
	if sslPreferServerCiphers, exists, err := GetMapKeyAsBool(cfgm.Data, "ssl-prefer-server-ciphers", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.MainServerSSLPreferServerCiphers = sslPreferServerCiphers
		}
	}
	if sslCiphers, exists := cfgm.Data["ssl-ciphers"]; exists {
		cfg.MainServerSSLCiphers = strings.Trim(sslCiphers, "\n")
	}
	if sslDHParamFile, exists := cfgm.Data["ssl-dhparam-file"]; exists {
		sslDHParamFile = strings.Trim(sslDHParamFile, "\n")
		cfg.MainServerSSLDHParamFileContent = &sslDHParamFile
	}
	if errorLogLevel, exists := cfgm.Data["error-log-level"]; exists {
		cfg.MainErrorLogLevel = errorLogLevel
	}
	if logFormat, exists := cfgm.Data["log-format"]; exists {
		cfg.MainLogFormat = logFormat
	}
	if streamLogFormat, exists := cfgm.Data["stream-log-format"]; exists {
		cfg.MainStreamLogFormat = streamLogFormat
	}
	if proxyBuffering, exists, err := GetMapKeyAsBool(cfgm.Data, "proxy-buffering", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.ProxyBuffering = proxyBuffering
		}
	}
	if proxyBuffers, exists := cfgm.Data["proxy-buffers"]; exists {
		cfg.ProxyBuffers = proxyBuffers
	}
	if proxyBufferSize, exists := cfgm.Data["proxy-buffer-size"]; exists {
		cfg.ProxyBufferSize = proxyBufferSize
	}
	if proxyMaxTempFileSize, exists := cfgm.Data["proxy-max-temp-file-size"]; exists {
		cfg.ProxyMaxTempFileSize = proxyMaxTempFileSize
	}

	if mainMainSnippets, exists, err := GetMapKeyAsStringSlice(cfgm.Data, "main-snippets", cfgm, "\n"); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.MainMainSnippets = mainMainSnippets
		}
	}
	if mainHTTPSnippets, exists, err := GetMapKeyAsStringSlice(cfgm.Data, "http-snippets", cfgm, "\n"); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.MainHTTPSnippets = mainHTTPSnippets
		}
	}
	if locationSnippets, exists, err := GetMapKeyAsStringSlice(cfgm.Data, "location-snippets", cfgm, "\n"); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.LocationSnippets = locationSnippets
		}
	}
	if serverSnippets, exists, err := GetMapKeyAsStringSlice(cfgm.Data, "server-snippets", cfgm, "\n"); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.ServerSnippets = serverSnippets
		}
	}
	if _, exists, err := GetMapKeyAsInt(cfgm.Data, "worker-processes", cfgm); exists {
		if err != nil && cfgm.Data["worker-processes"] != "auto" {
			glog.Errorf("Configmap %s/%s: Invalid value for worker-processes key: must be an integer or the string 'auto', got %q", cfgm.GetNamespace(), cfgm.GetName(), cfgm.Data["worker-processes"])
		} else {
			cfg.MainWorkerProcesses = cfgm.Data["worker-processes"]
		}
	}
	if workerCPUAffinity, exists := cfgm.Data["worker-cpu-affinity"]; exists {
		cfg.MainWorkerCPUAffinity = workerCPUAffinity
	}
	if workerShutdownTimeout, exists := cfgm.Data["worker-shutdown-timeout"]; exists {
		cfg.MainWorkerShutdownTimeout = workerShutdownTimeout
	}
	if workerConnections, exists := cfgm.Data["worker-connections"]; exists {
		cfg.MainWorkerConnections = workerConnections
	}
	if workerRlimitNofile, exists := cfgm.Data["worker-rlimit-nofile"]; exists {
		cfg.MainWorkerRlimitNofile = workerRlimitNofile
	}
	if keepalive, exists, err := GetMapKeyAsInt(cfgm.Data, "keepalive", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.Keepalive = keepalive
		}
	}
	if maxFails, exists, err := GetMapKeyAsInt(cfgm.Data, "max-fails", cfgm); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.MaxFails = maxFails
		}
	}
	if failTimeout, exists := cfgm.Data["fail-timeout"]; exists {
		cfg.FailTimeout = failTimeout
	}
	if mainTemplate, exists := cfgm.Data["main-template"]; exists {
		cfg.MainTemplate = &mainTemplate
	}
	if ingressTemplate, exists := cfgm.Data["ingress-template"]; exists {
		cfg.IngressTemplate = &ingressTemplate
	}
	if mainStreamSnippets, exists, err := GetMapKeyAsStringSlice(cfgm.Data, "stream-snippets", cfgm, "\n"); exists {
		if err != nil {
			glog.Error(err)
		} else {
			cfg.MainStreamSnippets = mainStreamSnippets
		}
	}
	return cfg
}

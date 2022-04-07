package configs

import conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"

// ConfigParams holds NGINX configuration parameters that affect the main NGINX config
// as well as configs for Ingress resources.
type ConfigParams struct {
	ClientMaxBodySize                      string
	DefaultServerAccessLogOff              bool
	DefaultServerReturn                    string
	FailTimeout                            string
	HealthCheckEnabled                     bool
	HealthCheckMandatory                   bool
	HealthCheckMandatoryQueue              int64
	HSTS                                   bool
	HSTSBehindProxy                        bool
	HSTSIncludeSubdomains                  bool
	HSTSMaxAge                             int64
	HTTP2                                  bool
	Keepalive                              int
	LBMethod                               string
	LocationSnippets                       []string
	MainAccessLogOff                       bool
	MainErrorLogLevel                      string
	MainHTTPSnippets                       []string
	MainKeepaliveRequests                  int64
	MainKeepaliveTimeout                   string
	MainLogFormat                          []string
	MainLogFormatEscaping                  string
	MainMainSnippets                       []string
	MainOpenTracingEnabled                 bool
	MainOpenTracingLoadModule              bool
	MainOpenTracingTracer                  string
	MainOpenTracingTracerConfig            string
	MainServerNamesHashBucketSize          string
	MainServerNamesHashMaxSize             string
	MainStreamLogFormat                    []string
	MainStreamLogFormatEscaping            string
	MainStreamSnippets                     []string
	MainWorkerConnections                  string
	MainWorkerCPUAffinity                  string
	MainWorkerProcesses                    string
	MainWorkerRlimitNofile                 string
	MainWorkerShutdownTimeout              string
	MaxConns                               int
	MaxFails                               int
	AppProtectEnable                       string
	AppProtectPolicy                       string
	AppProtectLogConf                      string
	AppProtectLogEnable                    string
	MainAppProtectFailureModeAction        string
	MainAppProtectCompressedRequestsAction string
	MainAppProtectCookieSeed               string
	MainAppProtectCPUThresholds            string
	MainAppProtectPhysicalMemoryThresholds string
	MainAppProtectReconnectPeriod          string
	AppProtectDosResource                  string
	MainAppProtectDosLogFormat             []string
	MainAppProtectDosLogFormatEscaping     string
	ProxyBuffering                         bool
	ProxyBuffers                           string
	ProxyBufferSize                        string
	ProxyConnectTimeout                    string
	ProxyHideHeaders                       []string
	ProxyMaxTempFileSize                   string
	ProxyPassHeaders                       []string
	ProxyProtocol                          bool
	ProxyReadTimeout                       string
	ProxySendTimeout                       string
	RedirectToHTTPS                        bool
	ResolverAddresses                      []string
	ResolverIPV6                           bool
	ResolverTimeout                        string
	ResolverValid                          string
	ServerSnippets                         []string
	ServerTokens                           string
	SlowStart                              string
	SSLRedirect                            bool
	UpstreamZoneSize                       string
	VariablesHashBucketSize                uint64
	VariablesHashMaxSize                   uint64

	RealIPHeader    string
	RealIPRecursive bool
	SetRealIPFrom   []string

	MainServerSSLCiphers             string
	MainServerSSLDHParam             string
	MainServerSSLDHParamFileContent  *string
	MainServerSSLPreferServerCiphers bool
	MainServerSSLProtocols           string

	IngressTemplate       *string
	VirtualServerTemplate *string
	MainTemplate          *string

	JWTKey      string
	JWTLoginURL string
	JWTRealm    string
	JWTToken    string

	Ports    []int
	SSLPorts []int

	SpiffeServerCerts bool
}

// StaticConfigParams holds immutable NGINX configuration parameters that affect the main NGINX config.
type StaticConfigParams struct {
	HealthStatus                   bool
	HealthStatusURI                string
	NginxStatus                    bool
	NginxStatusAllowCIDRs          []string
	NginxStatusPort                int
	StubStatusOverUnixSocketForOSS bool
	TLSPassthrough                 bool
	EnableSnippets                 bool
	NginxServiceMesh               bool
	EnableInternalRoutes           bool
	MainAppProtectLoadModule       bool
	MainAppProtectDosLoadModule    bool
	PodName                        string
	EnableLatencyMetrics           bool
	EnableOIDC                     bool
	SSLRejectHandshake             bool
	EnableCertManager              bool
}

// GlobalConfigParams holds global configuration parameters. For now, it only holds listeners.
// GlobalConfigParams should replace ConfigParams in the future.
type GlobalConfigParams struct {
	Listeners map[string]Listener
}

// Listener represents a listener that can be used in a TransportServer resource.
type Listener struct {
	Port     int
	Protocol string
}

// NewDefaultConfigParams creates a ConfigParams with default values.
func NewDefaultConfigParams(isPlus bool) *ConfigParams {
	upstreamZoneSize := "256k"
	if isPlus {
		upstreamZoneSize = "512k"
	}

	return &ConfigParams{
		DefaultServerReturn:           "404",
		ServerTokens:                  "on",
		ProxyConnectTimeout:           "60s",
		ProxyReadTimeout:              "60s",
		ProxySendTimeout:              "60s",
		ClientMaxBodySize:             "1m",
		SSLRedirect:                   true,
		MainServerNamesHashBucketSize: "256",
		MainServerNamesHashMaxSize:    "1024",
		ProxyBuffering:                true,
		MainWorkerProcesses:           "auto",
		MainWorkerConnections:         "1024",
		HSTSMaxAge:                    2592000,
		Ports:                         []int{80},
		SSLPorts:                      []int{443},
		MaxFails:                      1,
		MaxConns:                      0,
		UpstreamZoneSize:              upstreamZoneSize,
		FailTimeout:                   "10s",
		LBMethod:                      "random two least_conn",
		MainErrorLogLevel:             "notice",
		ResolverIPV6:                  true,
		MainKeepaliveTimeout:          "65s",
		MainKeepaliveRequests:         100,
		VariablesHashBucketSize:       256,
		VariablesHashMaxSize:          1024,
	}
}

// NewDefaultGlobalConfigParams creates a GlobalConfigParams with default values.
func NewDefaultGlobalConfigParams() *GlobalConfigParams {
	return &GlobalConfigParams{Listeners: map[string]Listener{}}
}

// NewGlobalConfigParamsWithTLSPassthrough creates new GlobalConfigParams with enabled TLS Passthrough listener.
func NewGlobalConfigParamsWithTLSPassthrough() *GlobalConfigParams {
	return &GlobalConfigParams{
		Listeners: map[string]Listener{
			conf_v1alpha1.TLSPassthroughListenerName: {
				Protocol: conf_v1alpha1.TLSPassthroughListenerProtocol,
			},
		},
	}
}

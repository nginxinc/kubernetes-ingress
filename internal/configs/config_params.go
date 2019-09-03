package configs

// ConfigParams holds NGINX configuration parameters that affect the main NGINX config
// as well as configs for Ingress resources.
type ConfigParams struct {
	LocationSnippets              []string
	ServerSnippets                []string
	ServerTokens                  string
	ProxyConnectTimeout           string
	ProxyReadTimeout              string
	ProxySendTimeout              string
	ClientMaxBodySize             string
	HTTP2                         bool
	RedirectToHTTPS               bool
	SSLRedirect                   bool
	MainMainSnippets              []string
	MainHTTPSnippets              []string
	MainStreamSnippets            []string
	MainServerNamesHashBucketSize string
	MainServerNamesHashMaxSize    string
	MainAccessLogOff              bool
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
	UpstreamZoneSize              string
	HSTS                          bool
	HSTSBehindProxy               bool
	HSTSMaxAge                    int64
	HSTSIncludeSubdomains         bool
	LBMethod                      string
	MainWorkerProcesses           string
	MainWorkerCPUAffinity         string
	MainWorkerShutdownTimeout     string
	MainWorkerConnections         string
	MainWorkerRlimitNofile        string
	Keepalive                     int
	MaxFails                      int
	MaxConns                      int
	FailTimeout                   string
	HealthCheckEnabled            bool
	HealthCheckMandatory          bool
	HealthCheckMandatoryQueue     int64
	SlowStart                     string
	ResolverAddresses             []string
	ResolverIPV6                  bool
	ResolverValid                 string
	ResolverTimeout               string
	MainKeepaliveTimeout          string
	MainKeepaliveRequests         int64
	VariablesHashBucketSize       uint64
	VariablesHashMaxSize          uint64
	MainOpenTracingLoadModule     bool
	MainOpenTracingEnabled        bool
	MainOpenTracingTracer         string
	MainOpenTracingTracerConfig   string

	RealIPHeader    string
	SetRealIPFrom   []string
	RealIPRecursive bool

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

// StaticConfigParams holds immutable NGINX configuration parameters that affect the main NGINX config.
type StaticConfigParams struct {
	HealthStatus                   bool
	NginxStatus                    bool
	NginxStatusAllowCIDRs          []string
	NginxStatusPort                int
	StubStatusOverUnixSocketForOSS bool
}

// NewDefaultConfigParams creates a ConfigParams with default values.
func NewDefaultConfigParams() *ConfigParams {
	return &ConfigParams{
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
		UpstreamZoneSize:              "256k",
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

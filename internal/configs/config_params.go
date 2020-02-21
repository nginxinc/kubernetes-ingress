package configs

// ConfigParams holds NGINX configuration parameters that affect the main NGINX config
// as well as configs for Ingress resources.
type ConfigParams struct {
	ClientMaxBodySize             string
	DefaultServerAccessLogOff     bool
	FailTimeout                   string
	HealthCheckEnabled            bool
	HealthCheckMandatory          bool
	HealthCheckMandatoryQueue     int64
	HSTS                          bool
	HSTSBehindProxy               bool
	HSTSIncludeSubdomains         bool
	HSTSMaxAge                    int64
	HTTP2                         bool
	Keepalive                     int
	LBMethod                      string
	LocationSnippets              []string
	MainAccessLogOff              bool
	MainErrorLogLevel             string
	MainHTTPSnippets              []string
	MainKeepaliveRequests         int64
	MainKeepaliveTimeout          string
	MainLogFormat                 []string
	MainLogFormatEscaping         string
	MainMainSnippets              []string
	MainOpenTracingEnabled        bool
	MainOpenTracingLoadModule     bool
	MainOpenTracingTracer         string
	MainOpenTracingTracerConfig   string
	MainServerNamesHashBucketSize string
	MainServerNamesHashMaxSize    string
	MainStreamLogFormat           []string
	MainStreamLogFormatEscaping   string
	MainStreamSnippets            []string
	MainWorkerConnections         string
	MainWorkerCPUAffinity         string
	MainWorkerProcesses           string
	MainWorkerRlimitNofile        string
	MainWorkerShutdownTimeout     string
	MaxConns                      int
	MaxFails                      int
	ProxyBuffering                bool
	ProxyBuffers                  string
	ProxyBufferSize               string
	ProxyConnectTimeout           string
	ProxyHideHeaders              []string
	ProxyMaxTempFileSize          string
	ProxyPassHeaders              []string
	ProxyProtocol                 bool
	ProxyReadTimeout              string
	ProxySendTimeout              string
	RedirectToHTTPS               bool
	ResolverAddresses             []string
	ResolverIPV6                  bool
	ResolverTimeout               string
	ResolverValid                 string
	ServerSnippets                []string
	ServerTokens                  string
	SlowStart                     string
	SSLRedirect                   bool
	UpstreamZoneSize              string
	VariablesHashBucketSize       uint64
	VariablesHashMaxSize          uint64

	RealIPHeader    string
	RealIPRecursive bool
	SetRealIPFrom   []string

	MainServerSSLCiphers             string
	MainServerSSLDHParam             string
	MainServerSSLDHParamFileContent  *string
	MainServerSSLPreferServerCiphers bool
	MainServerSSLProtocols           string

	IngressTemplate *string
	MainTemplate    *string

	JWTKey      string
	JWTLoginURL string
	JWTRealm    string
	JWTToken    string

	Ports    []int
	SSLPorts []int
}

// StaticConfigParams holds immutable NGINX configuration parameters that affect the main NGINX config.
type StaticConfigParams struct {
	HealthStatus                   bool
	HealthStatusURI                string
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

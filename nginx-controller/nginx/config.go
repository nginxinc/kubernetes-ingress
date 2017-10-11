package nginx

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
	MainHTTPSnippets              []string
	MainServerNamesHashBucketSize string
	MainServerNamesHashMaxSize    string
	MainLogFormat                 string
	ProxyBuffering                bool
	ProxyBuffers                  string
	ProxyBufferSize               string
	ProxyMaxTempFileSize          string
	ProxyProtocol                 bool
	ProxyHideHeaders              []string
	ProxyPassHeaders              []string
	ProxyTerminateAuthHeaders     bool
	HSTS                          bool
	HSTSMaxAge                    int64
	HSTSIncludeSubdomains         bool
	LBMethod                      string
	MainWorkerProcesses           string
	MainWorkerCPUAffinity         string
	Keepalive                     int64

	// http://nginx.org/en/docs/http/ngx_http_realip_module.html
	RealIPHeader    string
	SetRealIPFrom   []string
	RealIPRecursive bool

	// http://nginx.org/en/docs/http/ngx_http_ssl_module.html
	MainServerSSLProtocols           string
	MainServerSSLPreferServerCiphers bool
	MainServerSSLCiphers             string
	MainServerSSLDHParam             string

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
		HSTSMaxAge:                 2592000,
		Ports:                      []int{80},
		SSLPorts:                   []int{443},
	}
}

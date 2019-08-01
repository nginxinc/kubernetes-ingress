package version2

// VirtualServerConfig holds NGINX configuration for a VirtualServer.
type VirtualServerConfig struct {
	Server        Server
	Upstreams     []Upstream
	SplitClients  []SplitClient
	Maps          []Map
	StatusMatches []StatusMatch
}

// Upstream defines an upstream.
type Upstream struct {
	Name      string
	Servers   []UpstreamServer
	LBMethod  string
	Keepalive int
}

// UpstreamServer defines an upstream server.
type UpstreamServer struct {
	Address     string
	MaxFails    int
	MaxConns    int
	FailTimeout string
	Resolve     bool
}

// Server defines a server.
type Server struct {
	ServerName                            string
	ProxyProtocol                         bool
	SSL                                   *SSL
	RedirectToHTTPSBasedOnXForwarderProto bool
	ServerTokens                          string
	RealIPHeader                          string
	SetRealIPFrom                         []string
	RealIPRecursive                       bool
	Snippets                              []string
	InternalRedirectLocations             []InternalRedirectLocation
	Locations                             []Location
	HealthChecks                          []HealthCheck
}

// SSL defines SSL configuration for a server.
type SSL struct {
	HTTP2           bool
	Certificate     string
	CertificateKey  string
	Ciphers         string
	RedirectToHTTPS bool
}

// Location defines a location.
type Location struct {
	Path                     string
	Snippets                 []string
	ProxyConnectTimeout      string
	ProxyReadTimeout         string
	ProxySendTimeout         string
	ClientMaxBodySize        string
	ProxyMaxTempFileSize     string
	ProxyBuffering           bool
	ProxyBuffers             string
	ProxyBufferSize          string
	ProxyPass                string
	ProxyNextUpstream        string
	ProxyNextUpstreamTimeout string
	ProxyNextUpstreamTries   int
	HasKeepalive             bool
}

// SplitClient defines a split_clients.
type SplitClient struct {
	Source        string
	Variable      string
	Distributions []Distribution
}

// HealthCheck defines a HealthCheck for an upstream in a Server.
type HealthCheck struct {
	Name                string
	URI                 string
	Interval            string
	Jitter              string
	Fails               int
	Passes              int
	Port                int
	ProxyPass           string
	ProxyConnectTimeout string
	ProxyReadTimeout    string
	ProxySendTimeout    string
	Headers             map[string]string
	Match               string
}

// Distribution maps weight to a value in a SplitClient.
type Distribution struct {
	Weight string
	Value  string
}

// InternalRedirectLocation defines a location for internally redirecting requests to named locations.
type InternalRedirectLocation struct {
	Path        string
	Destination string
}

// Map defines a map.
type Map struct {
	Source     string
	Variable   string
	Parameters []Parameter
}

// Parameter defines a Parameter in a Map.
type Parameter struct {
	Value  string
	Result string
}

// StatusMatch defines a Match block for status codes.
type StatusMatch struct {
	Name string
	Code string
}

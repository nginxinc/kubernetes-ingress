package version2

// VirtualServerConfig holds NGINX configuration for a VirtualServer.
type VirtualServerConfig struct {
	Server       Server
	Upstreams    []Upstream
	SplitClients []SplitClient
	Maps         []Map
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

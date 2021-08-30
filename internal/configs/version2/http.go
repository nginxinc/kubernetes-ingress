package version2

import "fmt"

// UpstreamLabels describes the Prometheus labels for an NGINX upstream.
type UpstreamLabels struct {
	Service           string
	ResourceType      string
	ResourceName      string
	ResourceNamespace string
}

// VirtualServerConfig holds NGINX configuration for a VirtualServer.
type VirtualServerConfig struct {
	HTTPSnippets  []string
	LimitReqZones []LimitReqZone
	Maps          []Map
	Server        Server
	SpiffeCerts   bool
	SplitClients  []SplitClient
	StatusMatches []StatusMatch
	Upstreams     []Upstream
}

// Upstream defines an upstream.
type Upstream struct {
	Name             string
	Servers          []UpstreamServer
	LBMethod         string
	Resolve          bool
	Keepalive        int
	MaxFails         int
	MaxConns         int
	SlowStart        string
	FailTimeout      string
	UpstreamZoneSize string
	Queue            *Queue
	SessionCookie    *SessionCookie
	UpstreamLabels   UpstreamLabels
	NTLM             bool
}

// UpstreamServer defines an upstream server.
type UpstreamServer struct {
	Address string
}

// Server defines a server.
type Server struct {
	ServerName                string
	StatusZone                string
	ProxyProtocol             bool
	SSL                       *SSL
	ServerTokens              string
	RealIPHeader              string
	SetRealIPFrom             []string
	RealIPRecursive           bool
	Snippets                  []string
	InternalRedirectLocations []InternalRedirectLocation
	Locations                 []Location
	ErrorPageLocations        []ErrorPageLocation
	ReturnLocations           []ReturnLocation
	HealthChecks              []HealthCheck
	TLSRedirect               *TLSRedirect
	TLSPassthrough            bool
	Allow                     []string
	Deny                      []string
	LimitReqOptions           LimitReqOptions
	LimitReqs                 []LimitReq
	JWTAuth                   *JWTAuth
	IngressMTLS               *IngressMTLS
	EgressMTLS                *EgressMTLS
	OIDC                      *OIDC
	WAF                       *WAF
	PoliciesErrorReturn       *Return
	VSNamespace               string
	VSName                    string
}

// SSL defines SSL configuration for a server.
type SSL struct {
	HTTP2           bool
	Certificate     string
	CertificateKey  string
	RejectHandshake bool
}

// IngressMTLS defines TLS configuration for a server. This is a subset of TLS specifically for clients auth.
type IngressMTLS struct {
	ClientCert   string
	VerifyClient string
	VerifyDepth  int
}

// EgressMTLS defines TLS configuration for a location.
type EgressMTLS struct {
	Certificate    string
	CertificateKey string
	VerifyServer   bool
	VerifyDepth    int
	Ciphers        string
	Protocols      string
	TrustedCert    string
	SessionReuse   bool
	ServerName     bool
	SSLName        string
}

type OIDC struct {
	AuthEndpoint  string
	ClientID      string
	ClientSecret  string
	JwksURI       string
	Scope         string
	TokenEndpoint string
	RedirectURI   string
}

// WAF defines WAF configuration.
type WAF struct {
	Enable              string
	ApPolicy            string
	ApSecurityLogEnable bool
	ApLogConf           string
}

// Location defines a location.
type Location struct {
	Path                     string
	Internal                 bool
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
	ProxyInterceptErrors     bool
	ProxyPassRequestHeaders  bool
	ProxySetHeaders          []Header
	ProxyHideHeaders         []string
	ProxyPassHeaders         []string
	ProxyIgnoreHeaders       string
	ProxyPassRewrite         string
	AddHeaders               []AddHeader
	Rewrites                 []string
	HasKeepalive             bool
	ErrorPages               []ErrorPage
	ProxySSLName             string
	InternalProxyPass        string
	Allow                    []string
	Deny                     []string
	LimitReqOptions          LimitReqOptions
	LimitReqs                []LimitReq
	JWTAuth                  *JWTAuth
	EgressMTLS               *EgressMTLS
	OIDC                     bool
	WAF                      *WAF
	PoliciesErrorReturn      *Return
	ServiceName              string
	IsVSR                    bool
	VSRName                  string
	VSRNamespace             string
}

// ReturnLocation defines a location for returning a fixed response.
type ReturnLocation struct {
	Name        string
	DefaultType string
	Return      Return
}

// SplitClient defines a split_clients.
type SplitClient struct {
	Source        string
	Variable      string
	Distributions []Distribution
}

// Return defines a Return directive used for redirects and canned responses.
type Return struct {
	Code int
	Text string
}

// ErrorPage defines an error_page of a location.
type ErrorPage struct {
	Name         string
	Codes        string
	ResponseCode int
}

// ErrorPageLocation defines a named location for an error_page directive.
type ErrorPageLocation struct {
	Name        string
	DefaultType string
	Return      *Return
	Headers     []Header
}

// Header defines a header to use with add_header directive.
type Header struct {
	Name  string
	Value string
}

// AddHeader defines a header to use with add_header directive with an optional Always field.
type AddHeader struct {
	Header
	Always bool
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

// TLSRedirect defines a redirect in a Server.
type TLSRedirect struct {
	Code    int
	BasedOn string
}

// SessionCookie defines a session cookie for an upstream.
type SessionCookie struct {
	Enable   bool
	Name     string
	Path     string
	Expires  string
	Domain   string
	HTTPOnly bool
	Secure   bool
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

// Queue defines a queue in upstream.
type Queue struct {
	Size    int
	Timeout string
}

// LimitReqZone defines a rate limit shared memory zone.
type LimitReqZone struct {
	Key      string
	ZoneName string
	ZoneSize string
	Rate     string
}

func (rlz LimitReqZone) String() string {
	return fmt.Sprintf("{Key %q, ZoneName %q, ZoneSize %v, Rate %q}", rlz.Key, rlz.ZoneName, rlz.ZoneSize, rlz.Rate)
}

// LimitReq defines a rate limit.
type LimitReq struct {
	ZoneName string
	Burst    int
	NoDelay  bool
	Delay    int
}

func (rl LimitReq) String() string {
	return fmt.Sprintf("{ZoneName %q, Burst %q, NoDelay %v, Delay %q}", rl.ZoneName, rl.Burst, rl.NoDelay, rl.Delay)
}

// LimitReqOptions defines rate limit options.
type LimitReqOptions struct {
	DryRun     bool
	LogLevel   string
	RejectCode int
}

func (rl LimitReqOptions) String() string {
	return fmt.Sprintf("{DryRun %v, LogLevel %q, RejectCode %q}", rl.DryRun, rl.LogLevel, rl.RejectCode)
}

// JWTAuth holds JWT authentication configuration.
type JWTAuth struct {
	Secret string
	Realm  string
	Token  string
}

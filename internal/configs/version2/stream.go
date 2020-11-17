package version2

// TransportServerConfig holds NGINX configuration for a TransportServer.
type TransportServerConfig struct {
	Server    StreamServer
	Upstreams []StreamUpstream
}

// StreamUpstream defines a stream upstream.
type StreamUpstream struct {
	Name    string
	Servers []StreamUpstreamServer
}

// StreamUpstreamServer defines a stream upstream server.
type StreamUpstreamServer struct {
	Address string
}

// StreamServer defines a server in the stream module.
type StreamServer struct {
	TLSPassthrough bool
	UnixSocket     string
	Port           int
	UDP            bool
	StatusZone     string
	ProxyRequests  *int
	ProxyResponses *int
	ProxyPass      string
	Name           string
	Namespace      string
}

// TLSPassthroughHostsConfig defines a mapping between TLS Passthrough hosts and the corresponding unix sockets.
type TLSPassthroughHostsConfig map[string]string

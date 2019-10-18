package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualServer defines the VirtualServer resource.
type VirtualServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualServerSpec `json:"spec"`
}

// VirtualServerSpec is the spec of the VirtualServer resource.
type VirtualServerSpec struct {
	Host      string     `json:"host"`
	TLS       *TLS       `json:"tls"`
	Upstreams []Upstream `json:"upstreams"`
	Routes    []Route    `json:"routes"`
}

// Upstream defines an upstream.
type Upstream struct {
	Name                     string            `json:"name"`
	Service                  string            `json:"service"`
	Subselector              map[string]string `json:"subselector"`
	Port                     uint16            `json:"port"`
	LBMethod                 string            `json:"lb-method"`
	FailTimeout              string            `json:"fail-timeout"`
	MaxFails                 *int              `json:"max-fails"`
	MaxConns                 *int              `json:"max-conns"`
	Keepalive                *int              `json:"keepalive"`
	ProxyConnectTimeout      string            `json:"connect-timeout"`
	ProxyReadTimeout         string            `json:"read-timeout"`
	ProxySendTimeout         string            `json:"send-timeout"`
	ProxyNextUpstream        string            `json:"next-upstream"`
	ProxyNextUpstreamTimeout string            `json:"next-upstream-timeout"`
	ProxyNextUpstreamTries   int               `json:"next-upstream-tries"`
	ProxyBuffering           *bool             `json:"buffering"`
	ProxyBuffers             *UpstreamBuffers  `json:"buffers"`
	ProxyBufferSize          string            `json:"buffer-size"`
	ClientMaxBodySize        string            `json:"client-max-body-size"`
	TLS                      UpstreamTLS       `json:"tls"`
	HealthCheck              *HealthCheck      `json:"healthCheck"`
	SlowStart                string            `json:"slow-start"`
	Queue                    *UpstreamQueue    `json:"queue"`
	SessionCookie            *SessionCookie    `json:"sessionCookie"`
}

// UpstreamBuffers defines Buffer Configuration for an Upstream
type UpstreamBuffers struct {
	Number int    `json:"number"`
	Size   string `json:"size"`
}

// UpstreamTLS defines a TLS configuration for an Upstream.
type UpstreamTLS struct {
	Enable bool `json:"enable"`
}

// HealthCheck defines the parameters for active Upstream HealthChecks.
type HealthCheck struct {
	Enable         bool         `json:"enable"`
	Path           string       `json:"path"`
	Interval       string       `json:"interval"`
	Jitter         string       `json:"jitter"`
	Fails          int          `json:"fails"`
	Passes         int          `json:"passes"`
	Port           int          `json:"port"`
	TLS            *UpstreamTLS `json:"tls"`
	ConnectTimeout string       `json:"connect-timeout"`
	ReadTimeout    string       `json:"read-timeout"`
	SendTimeout    string       `json:"send-timeout"`
	Headers        []Header     `json:"headers"`
	StatusMatch    string       `json:"statusMatch"`
}

// Header defines an HTTP Header.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SessionCookie defines the parameters for session persistence.
type SessionCookie struct {
	Enable   bool   `json:"enable"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Expires  string `json:"expires"`
	Domain   string `json:"domain"`
	HTTPOnly bool   `json:"httpOnly"`
	Secure   bool   `json:"secure"`
}

// Route defines a route.
type Route struct {
	Path     string  `json:"path"`
	Upstream string  `json:"upstream"`
	Splits   []Split `json:"splits"`
	Rules    *Rules  `json:"rules"`
	Route    string  `json:"route"`
}

// Split defines a split.
type Split struct {
	Weight   int    `json:"weight"`
	Upstream string `json:"upstream"`
}

// Rules defines rules.
type Rules struct {
	Conditions      []Condition `json:"conditions"`
	Matches         []Match     `json:"matches"`
	DefaultUpstream string      `json:"defaultUpstream"`
}

// Condition defines a condition in a MatchRule.
type Condition struct {
	Header   string `json:"header"`
	Cookie   string `json:"cookie"`
	Argument string `json:"argument"`
	Variable string `json:"variable"`
}

// Match defines a match in a MatchRule.
type Match struct {
	Values   []string `json:"values"`
	Upstream string   `json:"upstream"`
}

// TLS defines TLS configuration for a VirtualServer.
type TLS struct {
	Secret string `json:"secret"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualServerList is a list of the VirtualServer resources.
type VirtualServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualServer `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualServerRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualServerRouteSpec `json:"spec"`
}

type VirtualServerRouteSpec struct {
	Host      string     `json:"host"`
	Upstreams []Upstream `json:"upstreams"`
	Subroutes []Route    `json:"subroutes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualServerRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualServerRoute `json:"items"`
}

// UpstreamQueue defines Queue Configuration for an Upstream
type UpstreamQueue struct {
	Size    int    `json:"size"`
	Timeout string `json:"timeout"`
}

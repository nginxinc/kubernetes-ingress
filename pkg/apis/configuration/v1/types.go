package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// StateWarning is used when the resource has been validated and accepted but it might work in a degraded state.
	StateWarning = "Warning"
	// StateValid is used when the resource has been validated and accepted and is working as expected.
	StateValid = "Valid"
	// StateInvalid is used when the resource failed validation or NGINX failed to reload the corresponding config.
	StateInvalid = "Invalid"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:validation:Optional

// VirtualServer defines the VirtualServer resource.
type VirtualServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualServerSpec   `json:"spec"`
	Status VirtualServerStatus `json:"status"`
}

// VirtualServerSpec is the spec of the VirtualServer resource.
type VirtualServerSpec struct {
	IngressClass string     `json:"class"`
	Host         string     `json:"host"`
	TLS          *TLS       `json:"tls"`
	Upstreams    []Upstream `json:"upstreams"`
	Routes       []Route    `json:"routes"`
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

// UpstreamBuffers defines Buffer Configuration for an Upstream.
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
	Path       string      `json:"path"`
	Route      string      `json:"route"`
	Action     *Action     `json:"action"`
	Splits     []Split     `json:"splits"`
	Matches    []Match     `json:"matches"`
	ErrorPages []ErrorPage `json:"errorPages"`
}

// Action defines an action.
type Action struct {
	Pass     string          `json:"pass"`
	Redirect *ActionRedirect `json:"redirect"`
	Return   *ActionReturn   `json:"return"`
}

// ActionRedirect defines a redirect in an Action.
type ActionRedirect struct {
	URL  string `json:"url"`
	Code int    `json:"code"`
}

// ActionReturn defines a return in an Action.
type ActionReturn struct {
	Code int    `json:"code"`
	Type string `json:"type"`
	Body string `json:"body"`
}

// Split defines a split.
type Split struct {
	Weight int     `json:"weight"`
	Action *Action `json:"action"`
}

// Condition defines a condition in a MatchRule.
type Condition struct {
	Header   string `json:"header"`
	Cookie   string `json:"cookie"`
	Argument string `json:"argument"`
	Variable string `json:"variable"`
	Value    string `json:"value"`
}

// Match defines a match.
type Match struct {
	Conditions []Condition `json:"conditions"`
	Action     *Action     `json:"action"`
	Splits     []Split     `json:"splits"`
}

// ErrorPage defines an ErrorPage in a Route.
type ErrorPage struct {
	Codes    []int              `json:"codes"`
	Return   *ErrorPageReturn   `json:"return"`
	Redirect *ErrorPageRedirect `json:"redirect"`
}

// ErrorPageReturn defines a return for an ErrorPage.
type ErrorPageReturn struct {
	ActionReturn `json:",inline"`
	Headers      []Header `json:"headers"`
}

// ErrorPageRedirect defines a redirect for an ErrorPage.
type ErrorPageRedirect struct {
	ActionRedirect `json:",inline"`
}

// TLS defines TLS configuration for a VirtualServer.
type TLS struct {
	Secret   string       `json:"secret"`
	Redirect *TLSRedirect `json:"redirect"`
}

// TLSRedirect defines a redirect for a TLS.
type TLSRedirect struct {
	Enable  bool   `json:"enable"`
	Code    *int   `json:"code"`
	BasedOn string `json:"basedOn"`
}

// VirtualServerStatus defines the status for the VirtualServer resource.
type VirtualServerStatus struct {
	State             string             `json:"state"`
	Reason            string             `json:"reason"`
	Message           string             `json:"message"`
	ExternalEndpoints []ExternalEndpoint `json:"externalEndpoints,omitempty"`
}

// ExternalEndpoint defines the IP and ports used to connect to this resource.
type ExternalEndpoint struct {
	IP    string `json:"ip"`
	Ports string `json:"ports"`
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

	Spec   VirtualServerRouteSpec   `json:"spec"`
	Status VirtualServerRouteStatus `json:"status"`
}

type VirtualServerRouteSpec struct {
	IngressClass string     `json:"class"`
	Host         string     `json:"host"`
	Upstreams    []Upstream `json:"upstreams"`
	Subroutes    []Route    `json:"subroutes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VirtualServerRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualServerRoute `json:"items"`
}

// UpstreamQueue defines Queue Configuration for an Upstream.
type UpstreamQueue struct {
	Size    int    `json:"size"`
	Timeout string `json:"timeout"`
}

// VirtualServerRouteStatus defines the status for the VirtualServerRoute resource.
type VirtualServerRouteStatus struct {
	State             string             `json:"state"`
	Reason            string             `json:"reason"`
	Message           string             `json:"message"`
	ReferencedBy      string             `json:"referencedBy"`
	ExternalEndpoints []ExternalEndpoint `json:"externalEndpoints,omitempty"`
}

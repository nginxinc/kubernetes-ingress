package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// TLSPassthroughListenerName is the name of a built-in TLS Passthrough listener.
	TLSPassthroughListenerName = "tls-passthrough"
	// TLSPassthroughListenerProtocol is the protocol of a built-in TLS Passthrough listener.
	TLSPassthroughListenerProtocol = "TLS_PASSTHROUGH"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:validation:Optional
// +kubebuilder:resource:shortName=gc

// GlobalConfiguration defines the GlobalConfiguration resource.
type GlobalConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GlobalConfigurationSpec `json:"spec"`
}

// GlobalConfigurationSpec is the spec of the GlobalConfiguration resource.
type GlobalConfigurationSpec struct {
	Listeners []Listener `json:"listeners"`
}

// Listener defines a listener.
type Listener struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GlobalConfigurationList is a list of the GlobalConfiguration resources.
type GlobalConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []GlobalConfiguration `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:validation:Optional
// +kubebuilder:resource:shortName=ts
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="Current state of the TransportServer. If the resource has a valid status, it means it has been validated and accepted by the Ingress Controller."
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.reason`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TransportServer defines the TransportServer resource.
type TransportServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TransportServerSpec   `json:"spec"`
	Status TransportServerStatus `json:"status"`
}

// TransportServerSpec is the spec of the TransportServer resource.
type TransportServerSpec struct {
	IngressClass       string                  `json:"ingressClassName"`
	Listener           TransportServerListener `json:"listener"`
	ServerSnippets     string                  `json:"serverSnippets"`
	StreamSnippets     string                  `json:"streamSnippets"`
	Host               string                  `json:"host"`
	Upstreams          []Upstream              `json:"upstreams"`
	UpstreamParameters *UpstreamParameters     `json:"upstreamParameters"`
	SessionParameters  *SessionParameters      `json:"sessionParameters"`
	Action             *Action                 `json:"action"`
}

// TransportServerListener defines a listener for a TransportServer.
type TransportServerListener struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

// Upstream defines an upstream.
type Upstream struct {
	Name                string       `json:"name"`
	Service             string       `json:"service"`
	Port                int          `json:"port"`
	FailTimeout         string       `json:"failTimeout"`
	MaxFails            *int         `json:"maxFails"`
	MaxConns            *int         `json:"maxConns"`
	HealthCheck         *HealthCheck `json:"healthCheck"`
	LoadBalancingMethod string       `json:"loadBalancingMethod"`
}

// HealthCheck defines the parameters for active Upstream HealthChecks.
type HealthCheck struct {
	Enabled  bool   `json:"enable"`
	Timeout  string `json:"timeout"`
	Jitter   string `json:"jitter"`
	Port     int    `json:"port"`
	Interval string `json:"interval"`
	Passes   int    `json:"passes"`
	Fails    int    `json:"fails"`
	Match    *Match `json:"match"`
}

// Match defines the parameters of a custom health check.
type Match struct {
	Send   string `json:"send"`
	Expect string `json:"expect"`
}

// UpstreamParameters defines parameters for an upstream.
type UpstreamParameters struct {
	UDPRequests  *int `json:"udpRequests"`
	UDPResponses *int `json:"udpResponses"`

	ConnectTimeout      string `json:"connectTimeout"`
	NextUpstream        bool   `json:"nextUpstream"`
	NextUpstreamTimeout string `json:"nextUpstreamTimeout"`
	NextUpstreamTries   int    `json:"nextUpstreamTries"`
}

// SessionParameters defines session parameters.
type SessionParameters struct {
	Timeout string `json:"timeout"`
}

// Action defines an action.
type Action struct {
	Pass string `json:"pass"`
}

// TransportServerStatus defines the status for the TransportServer resource.
type TransportServerStatus struct {
	State   string `json:"state"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TransportServerList is a list of the TransportServer resources.
type TransportServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []TransportServer `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:validation:Optional

// Policy defines a Policy for VirtualServer and VirtualServerRoute resources.
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PolicySpec `json:"spec"`
}

// PolicySpec is the spec of the Policy resource.
// The spec includes multiple fields, where each field represents a different policy.
// Only one policy (field) is allowed.
type PolicySpec struct {
	AccessControl *AccessControl `json:"accessControl"`
	RateLimit     *RateLimit     `json:"rateLimit"`
	JWTAuth       *JWTAuth       `json:"jwt"`
	IngressMTLS   *IngressMTLS   `json:"ingressMTLS"`
	EgressMTLS    *EgressMTLS    `json:"egressMTLS"`
}

// AccessControl defines an access policy based on the source IP of a request.
type AccessControl struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
}

// RateLimit defines a rate limit policy.
type RateLimit struct {
	Rate       string `json:"rate"`
	Key        string `json:"key"`
	Delay      *int   `json:"delay"`
	NoDelay    *bool  `json:"noDelay"`
	Burst      *int   `json:"burst"`
	ZoneSize   string `json:"zoneSize"`
	DryRun     *bool  `json:"dryRun"`
	LogLevel   string `json:"logLevel"`
	RejectCode *int   `json:"rejectCode"`
}

// JWTAuth holds JWT authentication configuration.
type JWTAuth struct {
	Realm  string `json:"realm"`
	Secret string `json:"secret"`
	Token  string `json:"token"`
}

// IngressMTLS defines an Ingress MTLS policy.
type IngressMTLS struct {
	ClientCertSecret string `json:"clientCertSecret"`
	VerifyClient     string `json:"verifyClient"`
	VerifyDepth      *int   `json:"verifyDepth"`
}

// EgressMTLS defines an Egress MTLS policy.
type EgressMTLS struct {
	TLSSecret         string `json:"tlsSecret"`
	VerifyServer      bool   `json:"verifyServer"`
	VerifyDepth       *int   `json:"verifyDepth"`
	Protocols         string `json:"protocols"`
	SessionReuse      *bool  `json:"sessionReuse"`
	Ciphers           string `json:"ciphers"`
	TrustedCertSecret string `json:"trustedCertSecret"`
	ServerName        bool   `json:"serverName"`
	SSLName           string `json:"sslName"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PolicyList is a list of the Policy resources.
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Policy `json:"items"`
}

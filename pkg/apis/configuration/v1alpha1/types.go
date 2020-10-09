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

// TransportServer defines the TransportServer resource.
type TransportServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TransportServerSpec `json:"spec"`
}

// TransportServerSpec is the spec of the TransportServer resource.
type TransportServerSpec struct {
	Listener           TransportServerListener `json:"listener"`
	Host               string                  `json:"host"`
	Upstreams          []Upstream              `json:"upstreams"`
	UpstreamParameters *UpstreamParameters     `json:"upstreamParameters"`
	Action             *Action                 `json:"action"`
}

// TransportServerListener defines a listener for a TransportServer.
type TransportServerListener struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
}

// Upstream defines an upstream.
type Upstream struct {
	Name    string `json:"name"`
	Service string `json:"service"`
	Port    int    `json:"port"`
}

// UpstreamParameters defines parameters for an upstream.
type UpstreamParameters struct {
	UDPRequests  *int `json:"udpRequests"`
	UDPResponses *int `json:"udpResponses"`
}

// Action defines an action.
type Action struct {
	Pass string `json:"pass"`
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

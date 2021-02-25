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
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`,description="Current state of the VirtualServer. If the resource has a valid status, it means it has been validated and accepted by the Ingress Controller."
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
	Name        string       `json:"name"`
	Service     string       `json:"service"`
	Port        int          `json:"port"`
	FailTimeout string       `json:"failTimeout"`
	MaxFails    *int         `json:"maxFails"`
	HealthCheck *HealthCheck `json:"healthCheck"`
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

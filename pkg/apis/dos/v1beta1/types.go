package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:validation:Optional
// +kubebuilder:resource:shortName=pr

// DosProtectedResource defines a Dos protected resource.
type DosProtectedResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DosProtectedResourceSpec `json:"spec"`
}

type DosProtectedResourceSpec struct {
	// Enable enables the DOS feature if set to true
	Enable bool `json:"enable"`
	// Name is the name of protected object, max of 63 characters.
	Name string `json:"name"`
	// ApDosMonitor is the URL to monitor server's stress. Default value: None, URL will be extracted from the first request which arrives and taken from "Host" header or from destination ip+port.
	ApDosMonitor string `json:"apDosMonitor"`
	// DosAccessLogDest is the network address for the access logs
	DosAccessLogDest string `json:"dosAccessLogDest"`
	// ApDosPolicy is the namespace/name of a ApDosPolicy resource
	ApDosPolicy    string          `json:"apDosPolicy"`
	DosSecurityLog *DosSecurityLog `json:"dosSecurityLog"`
}

// DosSecurityLog defines the security log of the DosProtectedResource.
type DosSecurityLog struct {
	// Enable enables the security logging feature if set to true
	Enable bool `json:"enable"`
	// ApDosLogConf is the namespace/name of a APDosLogConf resource
	ApDosLogConf string `json:"apDosLogConf"`
	// DosLogDest is the network address of a logging service, can be either IP or DNS name.
	DosLogDest string `json:"dosLogDest"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DosProtectedResourceList is a list of the DosProtectedResource resources.
type DosProtectedResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []DosProtectedResource `json:"items"`
}

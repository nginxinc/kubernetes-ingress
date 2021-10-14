// Code generated by client-gen. DO NOT EDIT.

package versioned

import (
	"fmt"

	k8sv1 "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned/typed/configuration/v1"
	k8sv1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned/typed/configuration/v1alpha1"
	appprotectdosv1beta1 "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned/typed/dos/v1beta1"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	K8sV1alpha1() k8sv1alpha1.K8sV1alpha1Interface
	K8sV1() k8sv1.K8sV1Interface
	AppprotectdosV1beta1() appprotectdosv1beta1.AppprotectdosV1beta1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	k8sV1alpha1          *k8sv1alpha1.K8sV1alpha1Client
	k8sV1                *k8sv1.K8sV1Client
	appprotectdosV1beta1 *appprotectdosv1beta1.AppprotectdosV1beta1Client
}

// K8sV1alpha1 retrieves the K8sV1alpha1Client
func (c *Clientset) K8sV1alpha1() k8sv1alpha1.K8sV1alpha1Interface {
	return c.k8sV1alpha1
}

// K8sV1 retrieves the K8sV1Client
func (c *Clientset) K8sV1() k8sv1.K8sV1Interface {
	return c.k8sV1
}

// AppprotectdosV1beta1 retrieves the AppprotectdosV1beta1Client
func (c *Clientset) AppprotectdosV1beta1() appprotectdosv1beta1.AppprotectdosV1beta1Interface {
	return c.appprotectdosV1beta1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfig will generate a rate-limiter in configShallowCopy.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		if configShallowCopy.Burst <= 0 {
			return nil, fmt.Errorf("burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0")
		}
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.k8sV1alpha1, err = k8sv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.k8sV1, err = k8sv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.appprotectdosV1beta1, err = appprotectdosv1beta1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.k8sV1alpha1 = k8sv1alpha1.NewForConfigOrDie(c)
	cs.k8sV1 = k8sv1.NewForConfigOrDie(c)
	cs.appprotectdosV1beta1 = appprotectdosv1beta1.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.k8sV1alpha1 = k8sv1alpha1.New(c)
	cs.k8sV1 = k8sv1.New(c)
	cs.appprotectdosV1beta1 = appprotectdosv1beta1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}

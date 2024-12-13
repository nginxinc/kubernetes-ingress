// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned/typed/configuration/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeK8sV1 struct {
	*testing.Fake
}

func (c *FakeK8sV1) GlobalConfigurations(namespace string) v1.GlobalConfigurationInterface {
	return newFakeGlobalConfigurations(c, namespace)
}

func (c *FakeK8sV1) Policies(namespace string) v1.PolicyInterface {
	return newFakePolicies(c, namespace)
}

func (c *FakeK8sV1) TransportServers(namespace string) v1.TransportServerInterface {
	return newFakeTransportServers(c, namespace)
}

func (c *FakeK8sV1) VirtualServers(namespace string) v1.VirtualServerInterface {
	return newFakeVirtualServers(c, namespace)
}

func (c *FakeK8sV1) VirtualServerRoutes(namespace string) v1.VirtualServerRouteInterface {
	return newFakeVirtualServerRoutes(c, namespace)
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeK8sV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}

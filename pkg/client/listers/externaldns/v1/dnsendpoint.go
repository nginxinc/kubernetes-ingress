// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/nginxinc/kubernetes-ingress/v3/pkg/apis/externaldns/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// DNSEndpointLister helps list DNSEndpoints.
// All objects returned here must be treated as read-only.
type DNSEndpointLister interface {
	// List lists all DNSEndpoints in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.DNSEndpoint, err error)
	// DNSEndpoints returns an object that can list and get DNSEndpoints.
	DNSEndpoints(namespace string) DNSEndpointNamespaceLister
	DNSEndpointListerExpansion
}

// dNSEndpointLister implements the DNSEndpointLister interface.
type dNSEndpointLister struct {
	indexer cache.Indexer
}

// NewDNSEndpointLister returns a new DNSEndpointLister.
func NewDNSEndpointLister(indexer cache.Indexer) DNSEndpointLister {
	return &dNSEndpointLister{indexer: indexer}
}

// List lists all DNSEndpoints in the indexer.
func (s *dNSEndpointLister) List(selector labels.Selector) (ret []*v1.DNSEndpoint, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.DNSEndpoint))
	})
	return ret, err
}

// DNSEndpoints returns an object that can list and get DNSEndpoints.
func (s *dNSEndpointLister) DNSEndpoints(namespace string) DNSEndpointNamespaceLister {
	return dNSEndpointNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// DNSEndpointNamespaceLister helps list and get DNSEndpoints.
// All objects returned here must be treated as read-only.
type DNSEndpointNamespaceLister interface {
	// List lists all DNSEndpoints in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.DNSEndpoint, err error)
	// Get retrieves the DNSEndpoint from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.DNSEndpoint, error)
	DNSEndpointNamespaceListerExpansion
}

// dNSEndpointNamespaceLister implements the DNSEndpointNamespaceLister
// interface.
type dNSEndpointNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all DNSEndpoints in the indexer for a given namespace.
func (s dNSEndpointNamespaceLister) List(selector labels.Selector) (ret []*v1.DNSEndpoint, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.DNSEndpoint))
	})
	return ret, err
}

// Get retrieves the DNSEndpoint from the indexer for a given namespace and name.
func (s dNSEndpointNamespaceLister) Get(name string) (*v1.DNSEndpoint, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("dnsendpoint"), name)
	}
	return obj.(*v1.DNSEndpoint), nil
}

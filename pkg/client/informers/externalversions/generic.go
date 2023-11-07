// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	configurationv1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	v1beta1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/dos/v1beta1"
	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/externaldns/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=appprotectdos.f5.com, Version=v1beta1
	case v1beta1.SchemeGroupVersion.WithResource("dosprotectedresources"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Appprotectdos().V1beta1().DosProtectedResources().Informer()}, nil

		// Group=externaldns.nginx.org, Version=v1
	case v1.SchemeGroupVersion.WithResource("dnsendpoints"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Externaldns().V1().DNSEndpoints().Informer()}, nil

		// Group=k8s.nginx.org, Version=v1
	case configurationv1.SchemeGroupVersion.WithResource("globalconfigurations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.K8s().V1().GlobalConfigurations().Informer()}, nil
	case configurationv1.SchemeGroupVersion.WithResource("policies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.K8s().V1().Policies().Informer()}, nil
	case configurationv1.SchemeGroupVersion.WithResource("transportservers"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.K8s().V1().TransportServers().Informer()}, nil
	case configurationv1.SchemeGroupVersion.WithResource("virtualservers"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.K8s().V1().VirtualServers().Informer()}, nil
	case configurationv1.SchemeGroupVersion.WithResource("virtualserverroutes"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.K8s().V1().VirtualServerRoutes().Informer()}, nil

		// Group=k8s.nginx.org, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithResource("globalconfigurations"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.K8s().V1alpha1().GlobalConfigurations().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("policies"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.K8s().V1alpha1().Policies().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("transportservers"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.K8s().V1alpha1().TransportServers().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}

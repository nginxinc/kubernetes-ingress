// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	context "context"
	time "time"

	apisconfigurationv1 "github.com/nginx/kubernetes-ingress/pkg/apis/configuration/v1"
	versioned "github.com/nginx/kubernetes-ingress/pkg/client/clientset/versioned"
	internalinterfaces "github.com/nginx/kubernetes-ingress/pkg/client/informers/externalversions/internalinterfaces"
	configurationv1 "github.com/nginx/kubernetes-ingress/pkg/client/listers/configuration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// TransportServerInformer provides access to a shared informer and lister for
// TransportServers.
type TransportServerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() configurationv1.TransportServerLister
}

type transportServerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewTransportServerInformer constructs a new informer for TransportServer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewTransportServerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredTransportServerInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredTransportServerInformer constructs a new informer for TransportServer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredTransportServerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.K8sV1().TransportServers(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.K8sV1().TransportServers(namespace).Watch(context.TODO(), options)
			},
		},
		&apisconfigurationv1.TransportServer{},
		resyncPeriod,
		indexers,
	)
}

func (f *transportServerInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredTransportServerInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *transportServerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apisconfigurationv1.TransportServer{}, f.defaultInformer)
}

func (f *transportServerInformer) Lister() configurationv1.TransportServerLister {
	return configurationv1.NewTransportServerLister(f.Informer().GetIndexer())
}

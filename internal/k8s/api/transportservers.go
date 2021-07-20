package api

import (
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

type TransportServers struct {
	store cache.Store
}

func NewTransportServers(store cache.Store) *TransportServers {
	return &TransportServers{
		store: store,
	}
}

func (a *TransportServers) Get(obj interface{}) (item *v1alpha1.TransportServer, exists bool, err error) {
	i, exists, err := a.store.Get(obj)
	if i == nil {
		return nil, exists, err
	}
	return i.(*v1alpha1.TransportServer), exists, err
}

func (a *TransportServers) GetByKey(key string) (item *v1alpha1.TransportServer, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1alpha1.TransportServer), exists, err
}

func (a *TransportServers) List() []*v1alpha1.TransportServer {
	var items []*v1alpha1.TransportServer
	for _, p := range a.store.List() {
		items = append(items, p.(*v1alpha1.TransportServer))
	}
	return items
}

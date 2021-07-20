package api

import (
	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/client-go/tools/cache"
)

type VirtualServers struct {
	store cache.Store
}

func NewVirtualServers(store cache.Store) *VirtualServers {
	return &VirtualServers{
		store: store,
	}
}

func (a *VirtualServers) Get(obj interface{}) (item *v1.VirtualServer, exists bool, err error) {
	i, exists, err := a.store.Get(obj)
	if i == nil {
		return nil, exists, err
	}
	return i.(*v1.VirtualServer), exists, err
}

func (a *VirtualServers) GetByKey(key string) (item *v1.VirtualServer, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1.VirtualServer), exists, err
}

func (a *VirtualServers) List() []*v1.VirtualServer {
	var items []*v1.VirtualServer
	for _, p := range a.store.List() {
		items = append(items, p.(*v1.VirtualServer))
	}
	return items
}

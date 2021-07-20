package api

import (
	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/client-go/tools/cache"
)

type VirtualServerRoutes struct {
	store cache.Store
}

func NewVirtualServerRoutes(store cache.Store) *VirtualServerRoutes {
	return &VirtualServerRoutes{
		store: store,
	}
}

func (a *VirtualServerRoutes) Get(obj *v1.VirtualServerRoute) (item *v1.VirtualServerRoute, exists bool, err error) {
	i, exists, err := a.store.Get(obj)
	if i == nil {
		return nil, exists, err
	}
	return i.(*v1.VirtualServerRoute), exists, err
}

func (a *VirtualServerRoutes) GetByKey(key string) (item *v1.VirtualServerRoute, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1.VirtualServerRoute), exists, err
}

func (a *VirtualServerRoutes) List() []*v1.VirtualServerRoute {
	var items []*v1.VirtualServerRoute
	for _, p := range a.store.List() {
		items = append(items, p.(*v1.VirtualServerRoute))
	}
	return items
}

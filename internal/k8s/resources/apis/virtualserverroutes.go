package apis

import "k8s.io/client-go/tools/cache"

type VirtualServerRoutes struct {
	store cache.Store
}

func NewVirtualServerRoutes(store cache.Store) *VirtualServerRoutes {
	return &VirtualServerRoutes{
		store: store,
	}
}

func (a *VirtualServerRoutes) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return a.store.Get(obj)
}

func (a *VirtualServerRoutes) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

func (a *VirtualServerRoutes) List() []interface{} {
	return a.store.List()
}

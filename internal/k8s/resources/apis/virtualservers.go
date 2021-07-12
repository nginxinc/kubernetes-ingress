package apis

import "k8s.io/client-go/tools/cache"

type VirtualServers struct {
	store cache.Store
}

func NewVirtualServers(store cache.Store) *VirtualServers {
	return &VirtualServers{
		store: store,
	}
}

func (a *VirtualServers) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return a.store.Get(obj)
}

func (a *VirtualServers) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

func (a *VirtualServers) List() []interface{} {
	return a.store.List()
}

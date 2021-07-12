package apis

import "k8s.io/client-go/tools/cache"

type TransportServers struct {
	store cache.Store
}

func NewTransportServers(store cache.Store) *TransportServers {
	return &TransportServers{
		store: store,
	}
}

func (a *TransportServers) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return a.store.Get(obj)
}

func (a *TransportServers) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

func (a *TransportServers) List() []interface{} {
	return a.store.List()
}

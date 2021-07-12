package apis

import "k8s.io/client-go/tools/cache"

type GlobalConfigurations struct {
	store cache.Store
}

func NewGlobalConfigurations(store cache.Store) *GlobalConfigurations {
	return &GlobalConfigurations{
		store: store,
	}
}

func (a *GlobalConfigurations) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

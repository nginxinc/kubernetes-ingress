package apis

import "k8s.io/client-go/tools/cache"

type ConfigMaps struct {
	store cache.Store
}

func NewConfigMaps(store cache.Store) *ConfigMaps {
	return &ConfigMaps{
		store: store,
	}
}

func (a *ConfigMaps) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

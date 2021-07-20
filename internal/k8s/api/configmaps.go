package api

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type ConfigMaps struct {
	store cache.Store
}

func NewConfigMaps(store cache.Store) *ConfigMaps {
	return &ConfigMaps{
		store: store,
	}
}

func (a *ConfigMaps) GetByKey(key string) (item *v1.ConfigMap, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1.ConfigMap), exists, err
}

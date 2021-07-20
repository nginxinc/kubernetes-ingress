package api

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type Services struct {
	store cache.Store
}

func NewServices(store cache.Store) *Services {
	return &Services{
		store: store,
	}
}

func (a *Services) GetByKey(key string) (item *v1.Service, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1.Service), exists, err
}

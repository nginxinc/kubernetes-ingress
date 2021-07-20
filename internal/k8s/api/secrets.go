package api

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type Secrets struct {
	store cache.Store
}

func NewSecrets(store cache.Store) *Secrets {
	return &Secrets{
		store: store,
	}
}

func (a *Secrets) GetByKey(key string) (item *v1.Secret, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1.Secret), exists, err
}

func (a *Secrets) List() []*v1.Secret {
	var items []*v1.Secret
	for _, p := range a.store.List() {
		items = append(items, p.(*v1.Secret))
	}
	return items
}

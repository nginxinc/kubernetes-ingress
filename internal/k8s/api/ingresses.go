package api

import (
	"k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

type Ingresses struct {
	store cache.Store
}

func NewIngresses(store cache.Store) *Ingresses {
	return &Ingresses{
		store: store,
	}
}

// GetByKeySafe returns a copy of the ingress that is safe to modify.
func (a *Ingresses) GetByKeySafe(key string) (ing *v1beta1.Ingress, exists bool, err error) {
	item, exists, err := a.store.GetByKey(key)
	if !exists || err != nil {
		return nil, exists, err
	}
	ing = item.(*v1beta1.Ingress).DeepCopy()
	return
}

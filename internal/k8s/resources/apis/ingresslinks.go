package apis

import "k8s.io/client-go/tools/cache"

type IngressLinks struct {
	store cache.Store
}

func NewIngressLinks(store cache.Store) *IngressLinks {
	return &IngressLinks{
		store: store,
	}
}

func (a *IngressLinks) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

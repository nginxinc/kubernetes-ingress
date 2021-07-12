package apis

import "k8s.io/client-go/tools/cache"

type Policies struct {
	store cache.Store
}

func NewPolicies(store cache.Store) *Policies {
	return &Policies{
		store: store,
	}
}

func (a *Policies) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return a.store.Get(obj)
}

func (a *Policies) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

func (a *Policies) List() []interface{} {
	return a.store.List()
}

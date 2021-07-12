package apis

import "k8s.io/client-go/tools/cache"

type Secrets struct {
	store cache.Store
}

func NewSecrets(store cache.Store) *Secrets {
	return &Secrets{
		store: store,
	}
}

func (a *Secrets) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

func (a *Secrets) List() []interface{} {
	return a.store.List()
}

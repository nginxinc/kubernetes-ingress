package apis

import "k8s.io/client-go/tools/cache"

type Services struct {
	store cache.Store
}

func NewServices(store cache.Store) *Services {
	return &Services{
		store: store,
	}
}

func (a *Services) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

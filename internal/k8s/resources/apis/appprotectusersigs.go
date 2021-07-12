package apis

import "k8s.io/client-go/tools/cache"

type AppProtectUserSigs struct {
	store cache.Store
}

func NewAppProtectUserSigs(store cache.Store) *AppProtectUserSigs {
	return &AppProtectUserSigs{
		store: store,
	}
}

func (a *AppProtectUserSigs) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

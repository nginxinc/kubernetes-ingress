package apis

import "k8s.io/client-go/tools/cache"

type AppProtectPolicies struct {
	store cache.Store
}

func NewAppProtectPolicies(store cache.Store) *AppProtectPolicies {
	return &AppProtectPolicies{
		store: store,
	}
}

func (a *AppProtectPolicies) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

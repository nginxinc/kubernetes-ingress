package apis

import "k8s.io/client-go/tools/cache"

type AppProtectLogConfs struct {
	store cache.Store
}

func NewAppProtectLogConfs(store cache.Store) *AppProtectLogConfs {
	return &AppProtectLogConfs{
		store: store,
	}
}

func (a *AppProtectLogConfs) GetByKey(key string) (item interface{}, exists bool, err error) {
	return a.store.GetByKey(key)
}

package api

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

type AppProtectPolicies struct {
	store cache.Store
}

func NewAppProtectPolicies(store cache.Store) *AppProtectPolicies {
	return &AppProtectPolicies{
		store: store,
	}
}

func (a *AppProtectPolicies) GetByKey(key string) (item *unstructured.Unstructured, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*unstructured.Unstructured), exists, err
}

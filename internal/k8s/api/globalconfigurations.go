package api

import (
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

type GlobalConfigurations struct {
	store cache.Store
}

func NewGlobalConfigurations(store cache.Store) *GlobalConfigurations {
	return &GlobalConfigurations{
		store: store,
	}
}

func (a *GlobalConfigurations) GetByKey(key string) (item *v1alpha1.GlobalConfiguration, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1alpha1.GlobalConfiguration), exists, err
}

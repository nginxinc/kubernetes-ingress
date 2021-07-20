package api

import (
	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/client-go/tools/cache"
)

type Policies struct {
	store cache.Store
}

func NewPolicies(store cache.Store) *Policies {
	return &Policies{
		store: store,
	}
}

func (a *Policies) Get(obj interface{}) (item *v1.Policy, exists bool, err error) {
	i, exists, err := a.store.Get(obj)
	if i == nil {
		return nil, exists, err
	}
	return i.(*v1.Policy), exists, err
}

func (a *Policies) GetByKey(key string) (item *v1.Policy, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1.Policy), exists, err
}

func (a *Policies) List() []*v1.Policy {
	var items []*v1.Policy
	for _, p := range a.store.List() {
		items = append(items, p.(*v1.Policy))
	}
	return items
}

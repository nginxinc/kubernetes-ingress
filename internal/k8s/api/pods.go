package api

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

type Pods struct {
	indexer cache.Indexer
}

func NewPods(indexer cache.Indexer) *Pods {
	return &Pods{
		indexer: indexer,
	}
}

func (a Pods) ListByNamespace(ns string, selector labels.Selector) (pods []*v1.Pod, err error) {
	err = cache.ListAllByNamespace(a.indexer, ns, selector, func(m interface{}) {
		pods = append(pods, m.(*v1.Pod))
	})
	return pods, err
}

func (a Pods) GetByKey(key string) (item *v1.Pod, exists bool, err error) {
	obj, exists, err := a.indexer.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1.Pod), exists, err
}

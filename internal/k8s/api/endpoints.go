package api

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type Endpoints struct {
	store cache.Store
}

func NewEndpoints(store cache.Store) *Endpoints {
	return &Endpoints{
		store: store,
	}
}

func (a *Endpoints) GetByKey(key string) (item *v1.Endpoints, exists bool, err error) {
	obj, exists, err := a.store.GetByKey(key)
	if obj == nil {
		return nil, exists, err
	}
	return obj.(*v1.Endpoints), exists, err
}

// GetServiceEndpoints returns the endpoints of a service, matched on service name.
func (a *Endpoints) GetServiceEndpoints(svc *v1.Service) (ep v1.Endpoints, err error) {
	for _, m := range a.store.List() {
		ep = *m.(*v1.Endpoints)
		if svc.Name == ep.Name && svc.Namespace == ep.Namespace {
			return ep, nil
		}
	}
	return ep, fmt.Errorf("could not find endpoints for service: %v", svc.Name)
}

package queue

import (
	"fmt"

	"github.com/nginxinc/kubernetes-ingress/internal/utils"
	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
)

// Kind represents the kind of the Kubernetes resources of a task
type Kind int

const (
	// Ingress resource
	Ingress = iota
	// IngressMinion resource, which is a Minion Ingress resource
	IngressMinion
	// Endpoints resource
	Endpoints
	// ConfigMap resource
	ConfigMap
	// Secret resource
	Secret
	// Service resource
	Service
)

// Task is an element of a TaskQueue
type Task struct {
	Kind Kind
	Key  string
}

// NewTask creates a new task
func NewTask(key string, obj interface{}) (Task, error) {
	var k Kind
	switch t := obj.(type) {
	case *extensions.Ingress:
		ing := obj.(*extensions.Ingress)
		if utils.IsMinion(ing) {
			k = IngressMinion
		} else {
			k = Ingress
		}
	case *api_v1.Endpoints:
		k = Endpoints
	case *api_v1.ConfigMap:
		k = ConfigMap
	case *api_v1.Secret:
		k = Secret
	case *api_v1.Service:
		k = Service
	default:
		return Task{}, fmt.Errorf("Unknow type: %v", t)
	}

	return Task{k, key}, nil
}

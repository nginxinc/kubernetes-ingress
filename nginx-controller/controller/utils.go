/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"

	"github.com/golang/glog"
)

// taskQueue manages a work queue through an independent worker that
// invokes the given sync function for every work item inserted.
type taskQueue struct {
	// queue is the work queue the worker polls
	queue *workqueue.Type
	// sync is called for each item in the queue
	sync func(Task)
	// workerDone is closed when the worker exits
	workerDone chan struct{}
}

func (t *taskQueue) run(period time.Duration, stopCh <-chan struct{}) {
	wait.Until(t.worker, period, stopCh)
}

// enqueue enqueues ns/name of the given api object in the task queue.
func (t *taskQueue) enqueue(obj interface{}) {
	key, err := keyFunc(obj)
	if err != nil {
		glog.V(3).Infof("Couldn't get key for object %v: %v", obj, err)
		return
	}

	task, err := NewTask(key, obj)
	if err != nil {
		glog.V(3).Infof("Couldn't create a task for object %v: %v", obj, err)
		return
	}

	glog.V(3).Infof("Adding an element with a key: %v", task.Key)

	t.queue.Add(task)
}

func (t *taskQueue) requeue(task Task, err error) {
	glog.Errorf("Requeuing %v, err %v", task.Key, err)
	t.queue.Add(task)
}

func (t *taskQueue) requeueAfter(task Task, err error, after time.Duration) {
	glog.Errorf("Requeuing %v after %s, err %v", task.Key, after.String(), err)
	go func(task Task, after time.Duration) {
		time.Sleep(after)
		t.queue.Add(task)
	}(task, after)
}

// worker processes work in the queue through sync.
func (t *taskQueue) worker() {
	for {
		task, quit := t.queue.Get()
		if quit {
			close(t.workerDone)
			return
		}
		glog.V(3).Infof("Syncing %v", task.(Task).Key)
		t.sync(task.(Task))
		t.queue.Done(task)
	}
}

// shutdown shuts down the work queue and waits for the worker to ACK
func (t *taskQueue) shutdown() {
	t.queue.ShutDown()
	<-t.workerDone
}

// NewTaskQueue creates a new task queue with the given sync function.
// The sync function is called for every element inserted into the queue.
func NewTaskQueue(syncFn func(Task)) *taskQueue {
	return &taskQueue{
		queue:      workqueue.New(),
		sync:       syncFn,
		workerDone: make(chan struct{}),
	}
}

// Kind represnts the kind of the Kubernetes resources of a task
type Kind int

const (
	// Ingress resource
	Ingress = iota
	// Endpoints resource
	Endpoints
	// ConfigMap resource
	ConfigMap
	// Secret resource
	Secret
)

// Task is an element of a taskQueue
type Task struct {
	Kind Kind
	Key  string
}

// NewTask creates a new task
func NewTask(key string, obj interface{}) (Task, error) {
	var k Kind
	switch t := obj.(type) {
	case *extensions.Ingress:
		k = Ingress
	case *api_v1.Endpoints:
		k = Endpoints
	case *api_v1.ConfigMap:
		k = ConfigMap
	case *api_v1.Secret:
		k = Secret
	default:
		return Task{}, fmt.Errorf("Unknow type: %v", t)
	}

	return Task{k, key}, nil
}

// compareLinks returns true if the 2 self links are equal.
func compareLinks(l1, l2 string) bool {
	// TODO: These can be partial links
	return l1 == l2 && l1 != ""
}

// StoreToIngressLister makes a Store that lists Ingress.
// TODO: Move this to cache/listers post 1.1.
type StoreToIngressLister struct {
	cache.Store
}

// List lists all Ingress' in the store.
func (s *StoreToIngressLister) List() (ing extensions.IngressList, err error) {
	for _, m := range s.Store.List() {
		ing.Items = append(ing.Items, *(m.(*extensions.Ingress)))
	}
	return ing, nil
}

// GetServiceIngress gets all the Ingress' that have rules pointing to a service.
// Note that this ignores services without the right nodePorts.
func (s *StoreToIngressLister) GetServiceIngress(svc *api_v1.Service) (ings []extensions.Ingress, err error) {
	for _, m := range s.Store.List() {
		ing := *m.(*extensions.Ingress)
		if ing.Namespace != svc.Namespace {
			continue
		}
		if ing.Spec.Backend != nil {
			if ing.Spec.Backend.ServiceName == svc.Name {
				ings = append(ings, ing)
			}
		}
		for _, rules := range ing.Spec.Rules {
			if rules.IngressRuleValue.HTTP == nil {
				continue
			}
			for _, p := range rules.IngressRuleValue.HTTP.Paths {
				if p.Backend.ServiceName == svc.Name {
					ings = append(ings, ing)
				}
			}
		}
	}
	if len(ings) == 0 {
		err = fmt.Errorf("No ingress for service %v", svc.Name)
	}
	return
}

// StoreToConfigMapLister makes a Store that lists ConfigMaps
type StoreToConfigMapLister struct {
	cache.Store
}

// List lists all Ingress' in the store.
func (s *StoreToConfigMapLister) List() (cfgm api_v1.ConfigMapList, err error) {
	for _, m := range s.Store.List() {
		cfgm.Items = append(cfgm.Items, *(m.(*api_v1.ConfigMap)))
	}
	return cfgm, nil
}

// StoreToEndpointLister makes a Store that lists Endponts
type StoreToEndpointLister struct {
	cache.Store
}

// GetServiceEndpoints returns the endpoints of a service, matched on service name.
func (s *StoreToEndpointLister) GetServiceEndpoints(svc *api_v1.Service) (ep api_v1.Endpoints, err error) {
	for _, m := range s.Store.List() {
		ep = *m.(*api_v1.Endpoints)
		if svc.Name == ep.Name && svc.Namespace == ep.Namespace {
			return ep, nil
		}
	}
	err = fmt.Errorf("could not find endpoints for service: %v", svc.Name)
	return
}

// FindPort locates the container port for the given pod and portName.  If the
// targetPort is a number, use that.  If the targetPort is a string, look that
// string up in all named ports in all containers in the target pod.  If no
// match is found, fail.
func FindPort(pod *api_v1.Pod, svcPort *api_v1.ServicePort) (int32, error) {
	portName := svcPort.TargetPort
	switch portName.Type {
	case intstr.String:
		name := portName.StrVal
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				if port.Name == name && port.Protocol == svcPort.Protocol {
					return port.ContainerPort, nil
				}
			}
		}
	case intstr.Int:
		return int32(portName.IntValue()), nil
	}

	return 0, fmt.Errorf("no suitable port for manifest: %s", pod.UID)
}

// StoreToSecretLister makes a Store that lists Secrets
type StoreToSecretLister struct {
	cache.Store
}

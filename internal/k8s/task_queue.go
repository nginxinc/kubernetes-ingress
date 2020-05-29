package k8s

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

// taskQueue manages a work queue through an independent worker that
// invokes the given sync function for every work item inserted.
type taskQueue struct {
	// queue is the work queue the worker polls
	queue *workqueue.Type
	// sync is called for each item in the queue
	sync func(task)
	// workerDone is closed when the worker exits
	workerDone chan struct{}
}

// newTaskQueue creates a new task queue with the given sync function.
// The sync function is called for every element inserted into the queue.
func newTaskQueue(syncFn func(task)) *taskQueue {
	return &taskQueue{
		queue:      workqueue.New(),
		sync:       syncFn,
		workerDone: make(chan struct{}),
	}
}

// Run begins running the worker for the given duration
func (tq *taskQueue) Run(period time.Duration, stopCh <-chan struct{}) {
	wait.Until(tq.worker, period, stopCh)
}

// Enqueue enqueues ns/name of the given api object in the task queue.
func (tq *taskQueue) Enqueue(obj interface{}) {
	key, err := keyFunc(obj)
	if err != nil {
		glog.V(3).Infof("Couldn't get key for object %v: %v", obj, err)
		return
	}

	task, err := newTask(key, obj)
	if err != nil {
		glog.V(3).Infof("Couldn't create a task for object %v: %v", obj, err)
		return
	}

	glog.V(3).Infof("Adding an element with a key: %v", task.Key)

	tq.queue.Add(task)
}

// Requeue adds the task to the queue again and logs the given error
func (tq *taskQueue) Requeue(task task, err error) {
	glog.Errorf("Requeuing %v, err %v", task.Key, err)
	tq.queue.Add(task)
}

// RequeueAfter adds the task to the queue after the given duration
func (tq *taskQueue) RequeueAfter(t task, err error, after time.Duration) {
	glog.Errorf("Requeuing %v after %s, err %v", t.Key, after.String(), err)
	go func(t task, after time.Duration) {
		time.Sleep(after)
		tq.queue.Add(t)
	}(t, after)
}

// Worker processes work in the queue through sync.
func (tq *taskQueue) worker() {
	for {
		t, quit := tq.queue.Get()
		if quit {
			close(tq.workerDone)
			return
		}
		glog.V(3).Infof("Syncing %v", t.(task).Key)
		tq.sync(t.(task))
		tq.queue.Done(t)
	}
}

// Shutdown shuts down the work queue and waits for the worker to ACK
func (tq *taskQueue) Shutdown() {
	tq.queue.ShutDown()
	<-tq.workerDone
}

// kind represents the kind of the Kubernetes resources of a task
type kind int

const (
	// ingress resource
	ingress = iota
	// ingressMinion resource, which is a Minion Ingress resource
	ingressMinion
	// endpoints resource
	endpoints
	// configMap resource
	configMap
	// secret resource
	secret
	// service resource
	service
	// virtualserver resource
	virtualserver
	// virtualServeRoute resource
	virtualServerRoute
	// globalConfiguration resource
	globalConfiguration
	// transportserver resource
	transportserver
	// policy resource
	policy
)

// task is an element of a taskQueue
type task struct {
	Kind kind
	Key  string
}

// newTask creates a new task
func newTask(key string, obj interface{}) (task, error) {
	var k kind
	switch t := obj.(type) {
	case *v1beta1.Ingress:
		ing := obj.(*v1beta1.Ingress)
		if isMinion(ing) {
			k = ingressMinion
		} else {
			k = ingress
		}
	case *v1.Endpoints:
		k = endpoints
	case *v1.ConfigMap:
		k = configMap
	case *v1.Secret:
		k = secret
	case *v1.Service:
		k = service
	case *conf_v1.VirtualServer:
		k = virtualserver
	case *conf_v1.VirtualServerRoute:
		k = virtualServerRoute
	case *conf_v1alpha1.GlobalConfiguration:
		k = globalConfiguration
	case *conf_v1alpha1.TransportServer:
		k = transportserver
	case *conf_v1alpha1.Policy:
		k = policy
	default:
		return task{}, fmt.Errorf("Unknown type: %v", t)
	}

	return task{k, key}, nil
}

package queue

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

// TaskQueue manages a work queue through an independent worker that
// invokes the given sync function for every work item inserted.
type TaskQueue struct {
	// queue is the work queue the worker polls
	queue *workqueue.Type
	// sync is called for each item in the queue
	sync func(Task)
	// workerDone is closed when the worker exits
	workerDone chan struct{}
}

// NewTaskQueue creates a new task queue with the given sync function.
// The sync function is called for every element inserted into the queue.
func NewTaskQueue(syncFn func(Task)) *TaskQueue {
	return &TaskQueue{
		queue:      workqueue.New(),
		sync:       syncFn,
		workerDone: make(chan struct{}),
	}
}

func (t *TaskQueue) Run(period time.Duration, stopCh <-chan struct{}) {
	wait.Until(t.worker, period, stopCh)
}

// Enqueue enqueues ns/name of the given api object in the task queue.
func (t *TaskQueue) Enqueue(obj interface{}) {
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

func (t *TaskQueue) Requeue(task Task, err error) {
	glog.Errorf("Requeuing %v, err %v", task.Key, err)
	t.queue.Add(task)
}

func (t *TaskQueue) RequeueAfter(task Task, err error, after time.Duration) {
	glog.Errorf("Requeuing %v after %s, err %v", task.Key, after.String(), err)
	go func(task Task, after time.Duration) {
		time.Sleep(after)
		t.queue.Add(task)
	}(task, after)
}

// worker processes work in the queue through sync.
func (t *TaskQueue) worker() {
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
func (t *TaskQueue) Shutdown() {
	t.queue.ShutDown()
	<-t.workerDone
}

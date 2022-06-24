package externaldns

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	extdns_clientset "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	k8s_nginx "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	listersV1 "github.com/nginxinc/kubernetes-ingress/pkg/client/listers/configuration/v1"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	k8s_nginx_informers "github.com/nginxinc/kubernetes-ingress/pkg/client/informers/externalversions"
)

const (
	// ControllerName is the name of the externaldns controler.
	ControllerName = "externaldns"

	// resyncPeriod is set to 10 hours
	// TODO: check with Ciara
	resyncPeriod = 10 * time.Hour
)

// ExtDNSController represents ExternalDNS controller.
type ExtDNSController struct {
	vsLister              listersV1.VirtualServerLister
	sync                  SyncFn
	ctx                   context.Context
	mustSync              []cache.InformerSynced
	queue                 workqueue.Interface
	sharedInformerFactory k8s_nginx_informers.SharedInformerFactory
	recorder              record.EventRecorder
	extDNSClient          *extdns_clientset.Clientset
}

// ExtDNSOpts represents config required for building the External DNS Controller.
type ExtDNSOpts struct {
	context       context.Context
	kubeConfig    *rest.Config
	kubeClient    kubernetes.Interface
	namespace     string
	eventRecorder record.EventRecorder
	vsClient      k8s_nginx.Interface
}

// NewController takes external dns config and return a new External DNS Controller.
func NewController(opts *ExtDNSOpts) (*ExtDNSController, error) {
	client, err := extdns_clientset.NewForConfig(opts.kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("%w, creating new externaldns controller", err)
	}

	sharedInformerFactory := k8s_nginx_informers.NewSharedInformerFactoryWithOptions(opts.vsClient, resyncPeriod, k8s_nginx_informers.WithNamespace(opts.namespace))

	c := &ExtDNSController{
		ctx:                   opts.context,
		queue:                 workqueue.NewNamed(ControllerName),
		sharedInformerFactory: sharedInformerFactory,
		recorder:              opts.eventRecorder,
		extDNSClient:          client,
	}
	c.register()
	return c, nil
}

func (c *ExtDNSController) register() workqueue.Interface {
	c.vsLister = c.sharedInformerFactory.K8s().V1().VirtualServers().Lister()
	//c.sharedInformerFactory.K8s().V1().VirtualServers().Informer().AddEventHandler()
	// todo

	// TODO: ?
	/*
		c.vsSharedInformerFactory.K8s().V1().VirtualServers().Informer().AddEventHandler(
			&controllerpkg.QueuingEventHandler{
				Queue: c.queue,
			},
		)
	*/

	// c.sync = SyncFnFor()
	c.sync = SyncFnFor(c.recorder, c.extDNSClient, c.sharedInformerFactory.Externaldns().V1().DNSEndpoints().Lister())

	c.mustSync = []cache.InformerSynced{
		c.sharedInformerFactory.K8s().V1().Policies().Informer().HasSynced,
		c.sharedInformerFactory.Externaldns().V1().DNSEndpoints().Informer().HasSynced,
	}
	return c.queue
}

// Run sets up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *ExtDNSController) Run(stopCh <-chan struct{}) {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()

	glog.Infof("Starting external-dns control loop")

	go c.sharedInformerFactory.Start(c.ctx.Done())

	// wait for all informer caches to be synced
	glog.V(3).Infof("Waiting for %d caches to sync", len(c.mustSync))
	if !cache.WaitForNamedCacheSync(ControllerName, stopCh, c.mustSync...) {
		glog.Fatal("error syncing extDNS queue")
	}

	glog.V(3).Infof("Queue is %v", c.queue.Len())

	go c.runWorker(ctx)

	<-stopCh
	glog.V(3).Infof("shutting down queue as workqueue signaled shutdown")
	c.queue.ShutDown()
}

// runWorker is a long-running function that will continually call the processItem
// function in order to read and process a message on the workqueue.
//
//
func (c *ExtDNSController) runWorker(ctx context.Context) {
	glog.V(3).Infof("processing items on the workqueue")
	for {
		obj, shutdown := c.queue.Get()
		if shutdown {
			break
		}

		func() {
			defer c.queue.Done(obj)
			key, ok := obj.(string)
			if !ok {
				return
			}

			if err := c.processItem(ctx, key); err != nil {
				glog.V(3).Infof("Re-queuing item due to error processing: %v", err)
				c.queue.Add(obj)
				return
			}
			glog.V(3).Infof("finished processing work item")
		}()
	}
}

//
// need to process VS and External DNS
//
// 1) if Kind VS -> call sync func on this kind
// 2) if the kind is ExternalDNS Endpoint -> run the func sth like reconcile - this can happen when
// etrnaldns endpoint is manipulated not from VS
func (c *ExtDNSController) processItem(ctx context.Context, key string) error {
	glog.V(3).Infof("processing external dns resource")
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return err
	}
	vs, err := c.vsLister.VirtualServers(namespace).Get(name)
	if err != nil {
		return err
	}
	return c.sync(ctx, vs)
}

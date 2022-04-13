/*
Copyright 2020 The cert-manager Authors.
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

// Package certmanager provides a controller for creating and managing
// certificates for VS resources.
package certmanager

import (
	"context"
	"fmt"
	"time"

	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cm_clientset "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	cm_informers "github.com/cert-manager/cert-manager/pkg/client/informers/externalversions"
	controllerpkg "github.com/cert-manager/cert-manager/pkg/controller"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	k8s_nginx "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	vsinformers "github.com/nginxinc/kubernetes-ingress/pkg/client/informers/externalversions"
	listers_v1 "github.com/nginxinc/kubernetes-ingress/pkg/client/listers/configuration/v1"
)

const (
	// ControllerName is the name of the certmanager controller
	ControllerName = "vs-cm-shim"

	// resyncPeriod is set to 10 hours across cert-manager. These 10 hours come
	// from a discussion on the controller-runtime project that boils down to:
	// never change this without an explicit reason.
	// https://github.com/kubernetes-sigs/controller-runtime/pull/88#issuecomment-408500629
	resyncPeriod = 10 * time.Hour
)

// CmController watches certificate and virtual server resources,
// and creates/ updates certificates for VS resources as required,
// and VS resources when certificate objects are created/ updated
type CmController struct {
	vsLister                  listers_v1.VirtualServerLister
	sync                      SyncFn
	ctx                       context.Context
	mustSync                  []cache.InformerSynced
	queue                     workqueue.RateLimitingInterface
	vsSharedInformerFactory   vsinformers.SharedInformerFactory
	cmSharedInformerFactory   cm_informers.SharedInformerFactory
	kubeSharedInformerFactory kubeinformers.SharedInformerFactory
	recorder                  record.EventRecorder
	cmClient                  *cm_clientset.Clientset
}

// CmOpts is the options required for building the CmController
type CmOpts struct {
	context       context.Context
	kubeConfig    *rest.Config
	kubeClient    kubernetes.Interface
	namespace     string
	eventRecorder record.EventRecorder
	vsClient      k8s_nginx.Interface
}

func (c *CmController) register() workqueue.RateLimitingInterface {
	c.vsLister = c.vsSharedInformerFactory.K8s().V1().VirtualServers().Lister()
	c.vsSharedInformerFactory.K8s().V1().VirtualServers().Informer().AddEventHandler(&controllerpkg.QueuingEventHandler{
		Queue: c.queue,
	})

	c.sync = SyncFnFor(c.recorder, c.cmClient, c.cmSharedInformerFactory.Certmanager().V1().Certificates().Lister())

	c.cmSharedInformerFactory.Certmanager().V1().Certificates().Informer().AddEventHandler(&controllerpkg.BlockingEventHandler{
		WorkFunc: certificateHandler(c.queue),
	})

	c.mustSync = []cache.InformerSynced{
		c.vsSharedInformerFactory.K8s().V1().VirtualServers().Informer().HasSynced,
		c.cmSharedInformerFactory.Certmanager().V1().Certificates().Informer().HasSynced,
	}
	return c.queue
}

func (c *CmController) processItem(ctx context.Context, key string) error {
	glog.V(3).Infof("processing virtual server resource ")
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

// Whenever a Certificate gets updated, added or deleted, we want to reconcile
// its parent VirtualServer. This parent VirtualServer is called "controller object". For
// example, the following Certificate "cert-1" is controlled by the VirtualServer
// "vs-1":
//
//     kind: Certificate
//     metadata:                                           Note that the owner
//       namespace: cert-1                                 reference does not
//       ownerReferences:                                  have a namespace,
//       - controller: true                                since owner refs
//         apiVersion: networking.x-k8s.io/v1alpha1        only work inside
//         kind: VirtualServer                             the same namespace.
//         name: vs-1
//         blockOwnerDeletion: true
//         uid: 7d3897c2-ce27-4144-883a-e1b5f89bd65a
func certificateHandler(queue workqueue.RateLimitingInterface) func(obj interface{}) {
	return func(obj interface{}) {
		crt, ok := obj.(*cmapi.Certificate)
		if !ok {
			runtime.HandleError(fmt.Errorf("not a Certificate object: %#v", obj))
			return
		}

		ref := metav1.GetControllerOf(crt)
		if ref == nil {
			// No controller should care about orphans being deleted or
			// updated.
			return
		}

		// We don't check the apiVersion
		// because there is no chance that another object called "VirtualServer" be
		// the controller of a Certificate.
		if ref.Kind != "VirtualServer" {
			return
		}

		queue.Add(crt.Namespace + "/" + ref.Name)
	}
}

// NewCmController creates a new CmController
func NewCmController(opts *CmOpts) *CmController {
	// Create a cert-manager api client
	intcl, _ := cm_clientset.NewForConfig(opts.kubeConfig)

	cmSharedInformerFactory := cm_informers.NewSharedInformerFactoryWithOptions(intcl, resyncPeriod, cm_informers.WithNamespace(opts.namespace))
	kubeSharedInformerFactory := kubeinformers.NewSharedInformerFactoryWithOptions(opts.kubeClient, resyncPeriod, kubeinformers.WithNamespace(opts.namespace))
	vsSharedInformerFactory := vsinformers.NewSharedInformerFactoryWithOptions(opts.vsClient, resyncPeriod, vsinformers.WithNamespace(opts.namespace))

	cm := &CmController{
		ctx:                       opts.context,
		queue:                     workqueue.NewNamedRateLimitingQueue(controllerpkg.DefaultItemBasedRateLimiter(), ControllerName),
		cmSharedInformerFactory:   cmSharedInformerFactory,
		kubeSharedInformerFactory: kubeSharedInformerFactory,
		recorder:                  opts.eventRecorder,
		cmClient:                  intcl,
		vsSharedInformerFactory:   vsSharedInformerFactory,
	}
	cm.register()
	return cm
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *CmController) Run(stopCh <-chan struct{}) {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()

	glog.Infof("Starting cert-manager control loop")

	go c.vsSharedInformerFactory.Start(c.ctx.Done())
	go c.cmSharedInformerFactory.Start(c.ctx.Done())
	go c.kubeSharedInformerFactory.Start(c.ctx.Done())
	// // wait for all the informer caches we depend on are synced
	glog.V(3).Infof("Waiting for %d caches to sync", len(c.mustSync))
	if !cache.WaitForNamedCacheSync(ControllerName, stopCh, c.mustSync...) {
		glog.Fatal("error syncing cm queue")
	}

	glog.V(3).Infof("Queue is %v", c.queue.Len())

	go c.runWorker(ctx)

	<-stopCh
	glog.V(3).Infof("shutting down queue as workqueue signaled shutdown")
	c.queue.ShutDown()
}

// runWorker is a long-running function that will continually call the
// processItem function in order to read and process a message on the
// workqueue.
func (c *CmController) runWorker(ctx context.Context) {
	glog.V(3).Infof("processing items on the workqueue")
	for {
		obj, shutdown := c.queue.Get()
		if shutdown {
			break
		}

		var key string
		// use an inlined function so we can use defer
		func() {
			defer c.queue.Done(obj)
			var ok bool
			if key, ok = obj.(string); !ok {
				return
			}

			err := c.processItem(ctx, key)
			if err != nil {
				glog.V(3).Infof("Re-queuing item due to error processing: %v", err)
				c.queue.AddRateLimited(obj)
				return
			}
			glog.V(3).Infof("finished processing work item")
			c.queue.Forget(obj)
		}()
	}
}

// BuildOpts builds a CmOpts from the given parameters
func BuildOpts(ctx context.Context, kc *rest.Config, cl kubernetes.Interface, ns string, er record.EventRecorder, vsc k8s_nginx.Interface) *CmOpts {
	return &CmOpts{
		context:       ctx,
		kubeClient:    cl,
		kubeConfig:    kc,
		namespace:     ns,
		eventRecorder: er,
		vsClient:      vsc,
	}
}

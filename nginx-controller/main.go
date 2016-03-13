package main

import (
	"flag"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/golang/glog"

	"github.com/nginxinc/kubernetes-ingress/nginx-controller/controller"
	"github.com/nginxinc/kubernetes-ingress/nginx-controller/nginx"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

var (
	proxyURL = flag.String("proxy", "",
		`If specified, the controller assumes a kubctl proxy server is running on the
		given url and creates a proxy client. Regenerated NGINX configuration files
    are not written to the disk, instead they are printed to stdout. Also NGINX
    is not getting invoked. This flag is for testing.`)
	resyncPeriod = flag.Duration("resyncPeriod", 30*time.Second, "amount of time between resyncs with the api server")

	lbcs     = make(map[string]*loadBalancerControllerCtx)
	lbcsLock sync.Mutex

	ngxc       *nginx.NGINXController
	kubeClient *client.Client
)

type loadBalancerControllerCtx struct {
	lbc    *controller.LoadBalancerController
	cancel context.CancelFunc
}

func main() {
	flag.Parse()

	var local = false

	if *proxyURL != "" {
		kubeClient = client.NewOrDie(&client.Config{
			Host: *proxyURL,
		})
		local = true
	} else {
		var err error
		kubeClient, err = client.NewInCluster()
		if err != nil {
			glog.Fatalf("Failed to create client: %v.", err)
		}
	}

	resolver := getKubeDNSIP(kubeClient)
	ngxc, _ = nginx.NewNGINXController(resolver, "/etc/nginx/", local)
	ngxc.Start()

	nsHandlers := framework.ResourceEventHandlerFuncs{
		AddFunc:    addNsFunc,
		DeleteFunc: delNsFunc,
		UpdateFunc: updateNsFunc,
	}

	_, nsController := framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  nsListFunc(kubeClient),
			WatchFunc: nsWatchFunc(kubeClient),
		},
		&api.Namespace{}, *resyncPeriod, nsHandlers)

	nsController.Run(make(chan struct{}))
}

func addNsFunc(obj interface{}) {
	addNs := obj.(*api.Namespace)
	glog.Infof("Adding namespace: %v", addNs.Name)
	lbcsLock.Lock()
	defer lbcsLock.Unlock()
	if _, ok := lbcs[addNs.Name]; !ok {
		lbc, err := controller.NewLoadBalancerController(kubeClient, *resyncPeriod, addNs.Name, ngxc)

		if err != nil {
			glog.Errorf("Error starting loadbalancer controller for namespace %v: %s", addNs.Name, err.Error())
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		lbcs[addNs.Name] = &loadBalancerControllerCtx{lbc, cancel}
		go lbc.Run(ctx)
	} else {
		glog.Errorf("Namespace %v already exists!", addNs.Name)
	}
}

func delNsFunc(obj interface{}) {
	remNs := obj.(*api.Namespace)
	glog.Infof("Removing namespace: %v", remNs.Name)
	lbcsLock.Lock()
	defer lbcsLock.Unlock()
	if lbcCtx, ok := lbcs[remNs.Name]; ok {
		lbcCtx.cancel()
		delete(lbcs, remNs.Name)
	} else {
		glog.Errorf("Namespace %v does not exist!", remNs.Name)
	}
}

func updateNsFunc(old, cur interface{}) {
	if old.(*api.Namespace).Name != cur.(*api.Namespace).Name {
		delNsFunc(old)
		addNsFunc(cur)
	}
}

func nsListFunc(c *client.Client) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Namespaces().List(opts)
	}
}

func nsWatchFunc(c *client.Client) func(api.ListOptions) (watch.Interface, error) {
	return func(opts api.ListOptions) (watch.Interface, error) {
		return c.Namespaces().Watch(opts)
	}
}

func getKubeDNSIP(kubeClient *client.Client) string {
	svcClient := kubeClient.Services("kube-system")
	svc, err := svcClient.Get("kube-dns")
	if err != nil {
		glog.Fatalf("Failed to get kube-dns service, err: %v", err)
	}
	return svc.Spec.ClusterIP
}

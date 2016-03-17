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
	"reflect"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/golang/glog"

	"github.com/nginxinc/kubernetes-ingress/nginx-controller/nginx"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/controller/framework"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

// LoadBalancerController watches Kubernetes API and
// reconfigures NGINX via NGINXController when needed
type LoadBalancerController struct {
	client         *client.Client
	ingController  *framework.Controller
	svcController  *framework.Controller
	endpController *framework.Controller
	ingLister      StoreToIngressLister
	svcLister      cache.StoreToServiceLister
	endpLister     cache.StoreToEndpointsLister
	ingQueue       *taskQueue
	nginx          *nginx.NGINXController
}

var keyFunc = framework.DeletionHandlingMetaNamespaceKeyFunc

// NewLoadBalancerController creates a controller
func NewLoadBalancerController(kubeClient *client.Client, resyncPeriod time.Duration, namespace string, nginx *nginx.NGINXController) (*LoadBalancerController, error) {
	lbc := LoadBalancerController{
		client: kubeClient,
		nginx:  nginx,
	}

	lbc.ingQueue = NewTaskQueue(lbc.syncIng)

	ingHandlers := framework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addIng := obj.(*extensions.Ingress)
			glog.Infof("Adding Ingress: %v", addIng.Name)
			lbc.ingQueue.enqueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			remIng := obj.(*extensions.Ingress)
			glog.Infof("Removing Ingress: %v", remIng.Name)
			lbc.ingQueue.enqueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.Infof("Ingress %v changed, syncing",
					cur.(*extensions.Ingress).Name)
				lbc.ingQueue.enqueue(cur)
			}
		},
	}
	lbc.ingLister.Store, lbc.ingController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  ingressListFunc(kubeClient, namespace),
			WatchFunc: ingressWatchFunc(kubeClient, namespace),
		},
		&extensions.Ingress{}, resyncPeriod, ingHandlers)

	svcHandlers := framework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addSvc := obj.(*api.Service)
			glog.Infof("Adding service: %v", addSvc.Name)
			lbc.enqueueIngressForService(obj)
		},
		DeleteFunc: func(obj interface{}) {
			remSvc := obj.(*api.Service)
			glog.Infof("Removing service: %v", remSvc.Name)
			lbc.enqueueIngressForService(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.Infof("Service %v changed, syncing",
					cur.(*api.Service).Name)
				lbc.enqueueIngressForService(cur)
			}
		},
	}
	lbc.svcLister.Store, lbc.svcController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  serviceListFunc(kubeClient, namespace),
			WatchFunc: serviceWatchFunc(kubeClient, namespace),
		},
		&api.Service{}, resyncPeriod, svcHandlers)

	endpHandlers := framework.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addEndp := obj.(*api.Endpoints)
			glog.Infof("Adding endpoints: %v", addEndp.Name)
			lbc.enqueueIngressForEndpoints(obj)
		},
		DeleteFunc: func(obj interface{}) {
			remEndp := obj.(*api.Endpoints)
			glog.Infof("Removing endpoints: %v", remEndp.Name)
			lbc.enqueueIngressForEndpoints(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.Infof("Endpoints %v changed, syncing",
					cur.(*api.Endpoints).Name)
				lbc.enqueueIngressForEndpoints(cur)
			}
		},
	}
	lbc.endpLister.Store, lbc.endpController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc:  endpointsListFunc(kubeClient, namespace),
			WatchFunc: endpointsWatchFunc(kubeClient, namespace),
		},
		&api.Endpoints{}, resyncPeriod, endpHandlers)

	return &lbc, nil
}

// Run starts the loadbalancer controller
func (lbc *LoadBalancerController) Run(ctx context.Context) {
	go lbc.ingController.Run(ctx.Done())
	go lbc.svcController.Run(ctx.Done())
	go lbc.endpController.Run(ctx.Done())
	go lbc.ingQueue.run(time.Second, ctx.Done())
}

func ingressListFunc(c *client.Client, ns string) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Extensions().Ingress(ns).List(opts)
	}
}

func ingressWatchFunc(c *client.Client, ns string) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return c.Extensions().Ingress(ns).Watch(options)
	}
}

func serviceListFunc(c *client.Client, ns string) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Services(ns).List(opts)
	}
}

func serviceWatchFunc(c *client.Client, ns string) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return c.Services(ns).Watch(options)
	}
}

func endpointsListFunc(c *client.Client, ns string) func(api.ListOptions) (runtime.Object, error) {
	return func(opts api.ListOptions) (runtime.Object, error) {
		return c.Endpoints(ns).List(opts)
	}
}

func endpointsWatchFunc(c *client.Client, ns string) func(options api.ListOptions) (watch.Interface, error) {
	return func(options api.ListOptions) (watch.Interface, error) {
		return c.Endpoints(ns).Watch(options)
	}
}

func (lbc *LoadBalancerController) syncIng(key string) {
	glog.Infof("Syncing %v", key)

	obj, ingExists, err := lbc.ingLister.Store.GetByKey(key)
	if err != nil {
		lbc.ingQueue.requeue(key, err)
		return
	}

	// defaut/some-ingress -> default-some-ingress
	name := strings.Replace(key, "/", "-", -1)

	if !ingExists {
		lbc.nginx.DeleteIngress(name)
	} else {
		ing := obj.(*extensions.Ingress)
		lbc.updateNGINX(name, ing)
	}

	lbc.nginx.Reload()
}

func (lbc *LoadBalancerController) enqueueIngressForService(obj interface{}) {
	svc := obj.(*api.Service)
	ings, err := lbc.ingLister.GetServiceIngress(svc)
	if err != nil {
		glog.Infof("ignoring service %v: %v", svc.Name, err)
		return
	}
	for _, ing := range ings {
		lbc.ingQueue.enqueue(&ing)
	}
}

func (lbc *LoadBalancerController) enqueueIngressForEndpoints(obj interface{}) {
	endp := obj.(*api.Endpoints)
	svcKey := endp.GetNamespace() + "/" + endp.GetName()
	svcObj, svcExists, err := lbc.svcLister.Store.GetByKey(svcKey)
	if err != nil {
		glog.Infof("error getting service %v from the cache: %v\n", svcKey, err)
	} else {
		if svcExists && svcObj.(*api.Service).Spec.ClusterIP == "None" {
			lbc.enqueueIngressForService(svcObj)
		}
	}
}

func (lbc *LoadBalancerController) updateNGINX(name string, ing *extensions.Ingress) {
	pems := make(map[string]string)

	for _, tls := range ing.Spec.TLS {
		secretName := tls.SecretName
		secret, err := lbc.client.Secrets(ing.Namespace).Get(secretName)
		if err != nil {
			glog.Warningf("Error retriveing secret %v for ing %v: %v", secretName, ing.Name, err)
			continue
		}
		cert, ok := secret.Data[api.TLSCertKey]
		if !ok {
			glog.Warningf("Secret %v has no private key", secretName)
			continue
		}
		key, ok := secret.Data[api.TLSPrivateKeyKey]
		if !ok {
			glog.Warningf("Secret %v has no cert", secretName)
			continue
		}

		pemFileName := lbc.nginx.AddOrUpdateCertAndKey(secretName, string(cert), string(key))

		for _, host := range tls.Hosts {
			pems[host] = pemFileName
		}
	}

	upstreams := make(map[string]nginx.Upstream)

	for _, rule := range ing.Spec.Rules {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			name := getNameForUpstream(ing, rule.Host, path.Backend.ServiceName)
			if _, exists := upstreams[name]; exists {
				continue
			}
			ups := nginx.NewUpstreamWithDefaultServer(name)

			svcKey := ing.Namespace + "/" + path.Backend.ServiceName
			svcObj, svcExists, err := lbc.svcLister.Store.GetByKey(svcKey)
			if err != nil {
				glog.Infof("error getting service %v from the cache: %v", svcKey, err)
			} else {
				if svcExists {
					svc := svcObj.(*api.Service)
					if svc.Spec.ClusterIP != "None" && svc.Spec.ClusterIP != "" {
						upsServer := nginx.UpstreamServer{Address: svc.Spec.ClusterIP, Port: path.Backend.ServicePort.String()}
						ups.UpstreamServers = []nginx.UpstreamServer{upsServer}
					} else if svc.Spec.ClusterIP == "None" {
						endps, err := lbc.endpLister.GetServiceEndpoints(svc)
						if err != nil {
							glog.Infof("error getting endpoints for service %v from the cache: %v", svc, err)
						} else {
							upsServers := endpointsToUpstreamServers(endps, path.Backend.ServicePort.IntValue())
							if len(upsServers) > 0 {
								ups.UpstreamServers = upsServers
							}
						}
					}
				}
			}
			upstreams[name] = ups

		}
	}

	var servers []nginx.Server
	for _, rule := range ing.Spec.Rules {
		server := nginx.Server{Name: rule.Host}

		if pemFile, ok := pems[rule.Host]; ok {
			server.SSL = true
			server.SSLCertificate = pemFile
			server.SSLCertificateKey = pemFile
		}

		var locations []nginx.Location

		for _, path := range rule.HTTP.Paths {
			loc := nginx.Location{Path: path.Path}
			upsName := getNameForUpstream(ing, rule.Host, path.Backend.ServiceName)

			if ups, ok := upstreams[upsName]; ok {
				loc.Upstream = ups
				locations = append(locations, loc)
			}
		}

		server.Locations = locations
		servers = append(servers, server)
	}

	lbc.nginx.AddOrUpdateIngress(name, nginx.IngressNGINXConfig{Upstreams: upstreamMapToSlice(upstreams), Servers: servers})
}

func endpointsToUpstreamServers(endps api.Endpoints, servicePort int) []nginx.UpstreamServer {
	var upsServers []nginx.UpstreamServer
	for _, subset := range endps.Subsets {
		for _, port := range subset.Ports {
			if port.Port == servicePort {
				for _, address := range subset.Addresses {
					ups := nginx.UpstreamServer{Address: address.IP, Port: fmt.Sprintf("%v", servicePort)}
					upsServers = append(upsServers, ups)
				}
				break
			}
		}
	}

	return upsServers
}

func getNameForUpstream(ing *extensions.Ingress, host string, service string) string {
	return fmt.Sprintf("%v-%v-%v-%v", ing.Namespace, ing.Name, host, service)
}

func upstreamMapToSlice(upstreams map[string]nginx.Upstream) []nginx.Upstream {
	result := make([]nginx.Upstream, 0, len(upstreams))

	for _, ups := range upstreams {
		glog.Info(ups)
		result = append(result, ups)
	}

	return result
}

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

	"github.com/golang/glog"

	"github.com/nginxinc/kubernetes-ingress/nginx-plus-controller/nginx"
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
	client        *client.Client
	ingController *framework.Controller
	ingLister     StoreToIngressLister
	ingQueue      *taskQueue
	stopCh        chan struct{}
	nginx         *nginx.NGINXController
}

var keyFunc = framework.DeletionHandlingMetaNamespaceKeyFunc

// NewLoadBalancerController creates a controller
func NewLoadBalancerController(kubeClient *client.Client, resyncPeriod time.Duration, namespace string, nginx *nginx.NGINXController) (*LoadBalancerController, error) {
	lbc := LoadBalancerController{
		client: kubeClient,
		stopCh: make(chan struct{}),
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

	return &lbc, nil
}

// Run starts the loadbalancer controller
func (lbc *LoadBalancerController) Run() {
	go lbc.ingController.Run(lbc.stopCh)
	go lbc.ingQueue.run(time.Second, lbc.stopCh)
	<-lbc.stopCh
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
			upstream := nginx.Upstream{Name: name}
			var upsServers []nginx.UpstreamServer
			address := fmt.Sprintf("%v.%v.svc.cluster.local", path.Backend.ServiceName, ing.Namespace)
			server := nginx.UpstreamServer{Address: address, Port: path.Backend.ServicePort.String()}
			upsServers = append(upsServers, server)
			upstream.UpstreamServers = upsServers

			upstreams[name] = upstream
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
			loc := nginx.Location{Path: pathOrDefault(path.Path)}
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

func pathOrDefault(path string) string {
	if path == "" {
		return "/"
	}
	return path
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

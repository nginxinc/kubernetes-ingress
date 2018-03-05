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

	"github.com/nginxinc/kubernetes-ingress/nginx-controller/nginx"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ingressClassKey   = "kubernetes.io/ingress.class"
)

// LoadBalancerController watches Kubernetes API and
// reconfigures NGINX via NginxController when needed
type LoadBalancerController struct {
	client               kubernetes.Interface
	ingController        cache.Controller
	svcController        cache.Controller
	endpController       cache.Controller
	cfgmController       cache.Controller
	secrController       cache.Controller
	ingLister            StoreToIngressLister
	svcLister            cache.Store
	endpLister           StoreToEndpointLister
	cfgmLister           StoreToConfigMapLister
	secrLister           StoreToSecretLister
	syncQueue            *taskQueue
	stopCh               chan struct{}
	cnf                  *nginx.Configurator
	watchNginxConfigMaps bool
	nginxPlus            bool
	recorder             record.EventRecorder
	defaultServerSecret  string
	ingressClass         string
	useIngressClassOnly  bool
}

var keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

// NewLoadBalancerController creates a controller
func NewLoadBalancerController(kubeClient kubernetes.Interface, resyncPeriod time.Duration, namespace string, cnf *nginx.Configurator, nginxConfigMaps string, defaultServerSecret string, nginxPlus bool, ingressClass string, useIngressClassOnly bool) (*LoadBalancerController, error) {
	lbc := LoadBalancerController{
		client:              kubeClient,
		stopCh:              make(chan struct{}),
		cnf:                 cnf,
		defaultServerSecret: defaultServerSecret,
		nginxPlus:           nginxPlus,
		ingressClass:        ingressClass,
		useIngressClassOnly: useIngressClassOnly,
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&core_v1.EventSinkImpl{
		Interface: core_v1.New(kubeClient.Core().RESTClient()).Events(""),
	})
	lbc.recorder = eventBroadcaster.NewRecorder(scheme.Scheme,
		api_v1.EventSource{Component: "nginx-ingress-controller"})

	lbc.syncQueue = NewTaskQueue(lbc.sync)

	glog.V(3).Infof("Nginx Ingress Controller has class: %v", ingressClass)

	ingHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addIng := obj.(*extensions.Ingress)
			if !lbc.isNginxIngress(addIng) {
				glog.Infof("Ignoring Ingress %v based on Annotation %v", addIng.Name, ingressClassKey)
				return
			}
			glog.V(3).Infof("Adding Ingress: %v", addIng.Name)
			lbc.syncQueue.enqueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			remIng, isIng := obj.(*extensions.Ingress)
			if !isIng {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				remIng, ok = deletedState.Obj.(*extensions.Ingress)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Ingress object: %v", deletedState.Obj)
					return
				}
			}
			if !lbc.isNginxIngress(remIng) {
				return
			}
			glog.V(3).Infof("Removing Ingress: %v", remIng.Name)
			lbc.syncQueue.enqueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			curIng := cur.(*extensions.Ingress)
			if !lbc.isNginxIngress(curIng) {
				return
			}
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Ingress %v changed, syncing", curIng.Name)
				lbc.syncQueue.enqueue(cur)
			}
		},
	}
	lbc.ingLister.Store, lbc.ingController = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.Extensions().RESTClient(), "ingresses", namespace, fields.Everything()),
		&extensions.Ingress{}, resyncPeriod, ingHandlers)

	svcHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addSvc := obj.(*api_v1.Service)
			glog.V(3).Infof("Adding service: %v", addSvc.Name)
			lbc.enqueueIngressForService(addSvc)
		},
		DeleteFunc: func(obj interface{}) {
			remSvc, isSvc := obj.(*api_v1.Service)
			if !isSvc {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				remSvc, ok = deletedState.Obj.(*api_v1.Service)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Service object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing service: %v", remSvc.Name)
			lbc.enqueueIngressForService(remSvc)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Service %v changed, syncing",
					cur.(*api_v1.Service).Name)
				lbc.enqueueIngressForService(cur.(*api_v1.Service))
			}
		},
	}
	lbc.svcLister, lbc.svcController = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.Core().RESTClient(), "services", namespace, fields.Everything()),
		&api_v1.Service{}, resyncPeriod, svcHandlers)

	endpHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addEndp := obj.(*api_v1.Endpoints)
			glog.V(3).Infof("Adding endpoints: %v", addEndp.Name)
			lbc.syncQueue.enqueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			remEndp, isEndp := obj.(*api_v1.Endpoints)
			if !isEndp {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				remEndp, ok = deletedState.Obj.(*api_v1.Endpoints)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Endpoints object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing endpoints: %v", remEndp.Name)
			lbc.syncQueue.enqueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Endpoints %v changed, syncing",
					cur.(*api_v1.Endpoints).Name)
				lbc.syncQueue.enqueue(cur)
			}
		},
	}
	lbc.endpLister.Store, lbc.endpController = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.Core().RESTClient(), "endpoints", namespace, fields.Everything()),
		&api_v1.Endpoints{}, resyncPeriod, endpHandlers)

	secrHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secr := obj.(*api_v1.Secret)
			nsname := secr.Namespace + "/" + secr.Name
			if nsname == lbc.defaultServerSecret {
				glog.V(3).Infof("Adding default server Secret: %v", secr.Name)
				lbc.syncQueue.enqueue(obj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			remSecr, isSecr := obj.(*api_v1.Secret)
			if !isSecr {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				remSecr, ok = deletedState.Obj.(*api_v1.Secret)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Secret object: %v", deletedState.Obj)
					return
				}
			}
			if err := lbc.ValidateSecret(remSecr); err != nil {
				return
			}

			glog.V(3).Infof("Removing Secret: %v", remSecr.Name)
			lbc.syncQueue.enqueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			errOld := lbc.ValidateSecret(old.(*api_v1.Secret))
			errCur := lbc.ValidateSecret(cur.(*api_v1.Secret))
			if errOld != nil && errCur != nil {
				return
			}

			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Secret %v changed, syncing",
					cur.(*api_v1.Secret).Name)
				lbc.syncQueue.enqueue(cur)
			}
		},
	}

	lbc.secrLister.Store, lbc.secrController = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.Core().RESTClient(), "secrets", namespace, fields.Everything()),
		&api_v1.Secret{}, resyncPeriod, secrHandlers)

	if nginxConfigMaps != "" {
		nginxConfigMapsNS, nginxConfigMapsName, err := ParseNamespaceName(nginxConfigMaps)
		if err != nil {
			glog.Warning(err)
		} else {
			lbc.watchNginxConfigMaps = true

			cfgmHandlers := cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					cfgm := obj.(*api_v1.ConfigMap)
					if cfgm.Name == nginxConfigMapsName {
						glog.V(3).Infof("Adding ConfigMap: %v", cfgm.Name)
						lbc.syncQueue.enqueue(obj)
					}
				},
				DeleteFunc: func(obj interface{}) {
					cfgm, isCfgm := obj.(*api_v1.ConfigMap)
					if !isCfgm {
						deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
						if !ok {
							glog.V(3).Infof("Error received unexpected object: %v", obj)
							return
						}
						cfgm, ok = deletedState.Obj.(*api_v1.ConfigMap)
						if !ok {
							glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-CogfigMap object: %v", deletedState.Obj)
							return
						}
					}
					if cfgm.Name == nginxConfigMapsName {
						glog.V(3).Infof("Removing ConfigMap: %v", cfgm.Name)
						lbc.syncQueue.enqueue(obj)
					}
				},
				UpdateFunc: func(old, cur interface{}) {
					if !reflect.DeepEqual(old, cur) {
						cfgm := cur.(*api_v1.ConfigMap)
						if cfgm.Name == nginxConfigMapsName {
							glog.V(3).Infof("ConfigMap %v changed, syncing",
								cur.(*api_v1.ConfigMap).Name)
							lbc.syncQueue.enqueue(cur)
						}
					}
				},
			}
			lbc.cfgmLister.Store, lbc.cfgmController = cache.NewInformer(
				cache.NewListWatchFromClient(lbc.client.Core().RESTClient(), "configmaps", nginxConfigMapsNS, fields.Everything()),
				&api_v1.ConfigMap{}, resyncPeriod, cfgmHandlers)
		}
	}

	return &lbc, nil
}

// Run starts the loadbalancer controller
func (lbc *LoadBalancerController) Run() {
	go lbc.svcController.Run(lbc.stopCh)
	go lbc.endpController.Run(lbc.stopCh)
	go lbc.secrController.Run(lbc.stopCh)
	if lbc.watchNginxConfigMaps {
		go lbc.cfgmController.Run(lbc.stopCh)
	}
	go lbc.ingController.Run(lbc.stopCh)
	go lbc.syncQueue.run(time.Second, lbc.stopCh)
	<-lbc.stopCh
}

// Stop shutdowns the load balancer controller
func (lbc *LoadBalancerController) Stop() {
	close(lbc.stopCh)

	lbc.syncQueue.shutdown()
}

func (lbc *LoadBalancerController) syncEndp(task Task) {
	key := task.Key
	glog.V(3).Infof("Syncing endpoints %v", key)

	obj, endpExists, err := lbc.endpLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.requeue(task, err)
		return
	}

	if endpExists {
		ings := lbc.getIngressForEndpoints(obj)
		var hasNginxIngress = false

		for _, ing := range ings {
			if !lbc.isNginxIngress(&ing) {
				continue
			}
			if !lbc.cnf.HasIngress(&ing) {
				continue
			}
			hasNginxIngress = true

			ingEx, err := lbc.createIngress(&ing)
			if err != nil {
				glog.Errorf("Error updating endpoints for %v/%v: %v, skipping", ing.Namespace, ing.Name, err)
				continue
			}
			glog.V(3).Infof("Updating Endpoints for %v/%v", ing.Name, ing.Namespace)
			err = lbc.cnf.UpdateEndpoints(ingEx)
			if err != nil {
				glog.Errorf("Error updating endpoints for %v/%v: %v", ing.Namespace, ing.Name, err)
			}
		}

		if hasNginxIngress {
			err = lbc.cnf.ReloadNginx()
			if err != nil {
				glog.Errorf("Error reloading NGINX when updating endpoints: %v", err)
			}
		}
	}
}

func (lbc *LoadBalancerController) syncCfgm(task Task) {
	key := task.Key
	glog.V(3).Infof("Syncing configmap %v", key)

	obj, cfgmExists, err := lbc.cfgmLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.requeue(task, err)
		return
	}
	cfg := nginx.NewDefaultConfig()

	if cfgmExists {
		cfgm := obj.(*api_v1.ConfigMap)

		if serverTokens, exists, err := nginx.GetMapKeyAsBool(cfgm.Data, "server-tokens", cfgm); exists {
			if err != nil {
				if lbc.nginxPlus {
					cfg.ServerTokens = cfgm.Data["server-tokens"]
				} else {
					glog.Error(err)
				}
			} else {
				cfg.ServerTokens = "off"
				if serverTokens {
					cfg.ServerTokens = "on"
				}
			}
		}

		if lbMethod, exists := cfgm.Data["lb-method"]; exists {
			cfg.LBMethod = lbMethod
		}

		if proxyConnectTimeout, exists := cfgm.Data["proxy-connect-timeout"]; exists {
			cfg.ProxyConnectTimeout = proxyConnectTimeout
		}
		if proxyReadTimeout, exists := cfgm.Data["proxy-read-timeout"]; exists {
			cfg.ProxyReadTimeout = proxyReadTimeout
		}
		if proxyHideHeaders, exists, err := nginx.GetMapKeyAsStringSlice(cfgm.Data, "proxy-hide-headers", cfgm, ","); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.ProxyHideHeaders = proxyHideHeaders
			}
		}
		if proxyPassHeaders, exists, err := nginx.GetMapKeyAsStringSlice(cfgm.Data, "proxy-pass-headers", cfgm, ","); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.ProxyPassHeaders = proxyPassHeaders
			}
		}
		if clientMaxBodySize, exists := cfgm.Data["client-max-body-size"]; exists {
			cfg.ClientMaxBodySize = clientMaxBodySize
		}
		if serverNamesHashBucketSize, exists := cfgm.Data["server-names-hash-bucket-size"]; exists {
			cfg.MainServerNamesHashBucketSize = serverNamesHashBucketSize
		}
		if serverNamesHashMaxSize, exists := cfgm.Data["server-names-hash-max-size"]; exists {
			cfg.MainServerNamesHashMaxSize = serverNamesHashMaxSize
		}
		if HTTP2, exists, err := nginx.GetMapKeyAsBool(cfgm.Data, "http2", cfgm); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.HTTP2 = HTTP2
			}
		}
		if redirectToHTTPS, exists, err := nginx.GetMapKeyAsBool(cfgm.Data, "redirect-to-https", cfgm); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.RedirectToHTTPS = redirectToHTTPS
			}
		}
		if sslRedirect, exists, err := nginx.GetMapKeyAsBool(cfgm.Data, "ssl-redirect", cfgm); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.SSLRedirect = sslRedirect
			}
		}

		// HSTS block
		if hsts, exists, err := nginx.GetMapKeyAsBool(cfgm.Data, "hsts", cfgm); exists {
			if err != nil {
				glog.Error(err)
			} else {
				parsingErrors := false

				hstsMaxAge, existsMA, err := nginx.GetMapKeyAsInt(cfgm.Data, "hsts-max-age", cfgm)
				if existsMA && err != nil {
					glog.Error(err)
					parsingErrors = true
				}
				hstsIncludeSubdomains, existsIS, err := nginx.GetMapKeyAsBool(cfgm.Data, "hsts-include-subdomains", cfgm)
				if existsIS && err != nil {
					glog.Error(err)
					parsingErrors = true
				}

				if parsingErrors {
					glog.Errorf("Configmap %s/%s: There are configuration issues with hsts annotations, skipping options for all hsts settings", cfgm.GetNamespace(), cfgm.GetName())
				} else {
					cfg.HSTS = hsts
					if existsMA {
						cfg.HSTSMaxAge = hstsMaxAge
					}
					if existsIS {
						cfg.HSTSIncludeSubdomains = hstsIncludeSubdomains
					}
				}
			}
		}

		if proxyProtocol, exists, err := nginx.GetMapKeyAsBool(cfgm.Data, "proxy-protocol", cfgm); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.ProxyProtocol = proxyProtocol
			}
		}

		// ngx_http_realip_module
		if realIPHeader, exists := cfgm.Data["real-ip-header"]; exists {
			cfg.RealIPHeader = realIPHeader
		}
		if setRealIPFrom, exists, err := nginx.GetMapKeyAsStringSlice(cfgm.Data, "set-real-ip-from", cfgm, ","); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.SetRealIPFrom = setRealIPFrom
			}
		}
		if realIPRecursive, exists, err := nginx.GetMapKeyAsBool(cfgm.Data, "real-ip-recursive", cfgm); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.RealIPRecursive = realIPRecursive
			}
		}

		// SSL block
		if sslProtocols, exists := cfgm.Data["ssl-protocols"]; exists {
			cfg.MainServerSSLProtocols = sslProtocols
		}
		if sslPreferServerCiphers, exists, err := nginx.GetMapKeyAsBool(cfgm.Data, "ssl-prefer-server-ciphers", cfgm); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.MainServerSSLPreferServerCiphers = sslPreferServerCiphers
			}
		}
		if sslCiphers, exists := cfgm.Data["ssl-ciphers"]; exists {
			cfg.MainServerSSLCiphers = strings.Trim(sslCiphers, "\n")
		}
		if sslDHParamFile, exists := cfgm.Data["ssl-dhparam-file"]; exists {
			sslDHParamFile = strings.Trim(sslDHParamFile, "\n")
			fileName, err := lbc.cnf.AddOrUpdateDHParam(sslDHParamFile)
			if err != nil {
				glog.Errorf("Configmap %s/%s: Could not update dhparams: %v", cfgm.GetNamespace(), cfgm.GetName(), err)
			} else {
				cfg.MainServerSSLDHParam = fileName
			}
		}

		if logFormat, exists := cfgm.Data["log-format"]; exists {
			cfg.MainLogFormat = logFormat
		}
		if proxyBuffering, exists, err := nginx.GetMapKeyAsBool(cfgm.Data, "proxy-buffering", cfgm); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.ProxyBuffering = proxyBuffering
			}
		}
		if proxyBuffers, exists := cfgm.Data["proxy-buffers"]; exists {
			cfg.ProxyBuffers = proxyBuffers
		}
		if proxyBufferSize, exists := cfgm.Data["proxy-buffer-size"]; exists {
			cfg.ProxyBufferSize = proxyBufferSize
		}
		if proxyMaxTempFileSize, exists := cfgm.Data["proxy-max-temp-file-size"]; exists {
			cfg.ProxyMaxTempFileSize = proxyMaxTempFileSize
		}

		if mainMainSnippets, exists, err := nginx.GetMapKeyAsStringSlice(cfgm.Data, "main-snippets", cfgm, "\n"); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.MainMainSnippets = mainMainSnippets
			}
		}
		if mainHTTPSnippets, exists, err := nginx.GetMapKeyAsStringSlice(cfgm.Data, "http-snippets", cfgm, "\n"); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.MainHTTPSnippets = mainHTTPSnippets
			}
		}
		if locationSnippets, exists, err := nginx.GetMapKeyAsStringSlice(cfgm.Data, "location-snippets", cfgm, "\n"); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.LocationSnippets = locationSnippets
			}
		}
		if serverSnippets, exists, err := nginx.GetMapKeyAsStringSlice(cfgm.Data, "server-snippets", cfgm, "\n"); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.ServerSnippets = serverSnippets
			}
		}
		if _, exists, err := nginx.GetMapKeyAsInt(cfgm.Data, "worker-processes", cfgm); exists {
			if err != nil && cfgm.Data["worker-processes"] != "auto" {
				glog.Errorf("Configmap %s/%s: Invalid value for worker-processes key: must be an integer or the string 'auto', got %q", cfgm.GetNamespace(), cfgm.GetName(), cfgm.Data["worker-processes"])
			} else {
				cfg.MainWorkerProcesses = cfgm.Data["worker-processes"]
			}
		}
		if workerCPUAffinity, exists := cfgm.Data["worker-cpu-affinity"]; exists {
			cfg.MainWorkerCPUAffinity = workerCPUAffinity
		}
		if workerShutdownTimeout, exists := cfgm.Data["worker-shutdown-timeout"]; exists {
			cfg.MainWorkerShutdownTimeout = workerShutdownTimeout
		}
		if workerConnections, exists := cfgm.Data["worker-connections"]; exists {
			cfg.MainWorkerConnections = workerConnections
		}
		if workerRlimitNofile, exists := cfgm.Data["worker-rlimit-nofile"]; exists {
			cfg.MainWorkerRlimitNofile = workerRlimitNofile
		}
		if keepalive, exists, err := nginx.GetMapKeyAsInt(cfgm.Data, "keepalive", cfgm); exists {
			if err != nil {
				glog.Error(err)
			} else {
				cfg.Keepalive = keepalive
			}
		}
	}

	var ingExes []*nginx.IngressEx
	ings, _ := lbc.ingLister.List()
	for i := range ings.Items {
		if !lbc.isNginxIngress(&ings.Items[i]) {
			continue
		}
		if !lbc.cnf.HasIngress(&ings.Items[i]) {
			continue
		}
		ingEx, err := lbc.createIngress(&ings.Items[i])
		if err != nil {
			continue
		}

		ingExes = append(ingExes, ingEx)
	}

	if err := lbc.cnf.UpdateConfig(cfg, ingExes); err != nil {
		if cfgmExists {
			cfgm := obj.(*api_v1.ConfigMap)
			lbc.recorder.Eventf(cfgm, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration from %v was updated, but not applied: %v", key, err)
		}
		for _, ingEx := range ingExes {
			lbc.recorder.Eventf(ingEx.Ingress, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration for %v/%v was updated, but not applied: %v",
				ingEx.Ingress.Name, ingEx.Ingress.Namespace, err)
		}
	} else {
		if cfgmExists {
			cfgm := obj.(*api_v1.ConfigMap)
			lbc.recorder.Eventf(cfgm, api_v1.EventTypeNormal, "Updated", "Configuration from %v was updated", key)
		}
		for _, ingEx := range ingExes {
			lbc.recorder.Eventf(ingEx.Ingress, api_v1.EventTypeNormal, "Updated", "Configuration for %v/%v was updated", ingEx.Ingress.Name, ingEx.Ingress.Namespace)
		}
	}
}

func (lbc *LoadBalancerController) sync(task Task) {
	glog.V(3).Infof("Syncing %v", task.Key)

	switch task.Kind {
	case Ingress:
		lbc.syncIng(task)
	case ConfigMap:
		lbc.syncCfgm(task)
		return
	case Endpoints:
		lbc.syncEndp(task)
		return
	case Secret:
		lbc.syncSecret(task)
	}
}

func (lbc *LoadBalancerController) syncIng(task Task) {
	key := task.Key
	obj, ingExists, err := lbc.ingLister.Store.GetByKey(key)
	if err != nil {
		lbc.syncQueue.requeue(task, err)
		return
	}

	if !ingExists {
		glog.V(2).Infof("Deleting Ingress: %v\n", key)
		err := lbc.cnf.DeleteIngress(key)
		if err != nil {
			glog.Errorf("Error when deleting configuration for %v: %v", key, err)
		}
	} else {
		glog.V(2).Infof("Adding or Updating Ingress: %v\n", key)

		ing := obj.(*extensions.Ingress)
		ingEx, err := lbc.createIngress(ing)
		if err != nil {
			lbc.syncQueue.requeueAfter(task, err, 5*time.Second)
			lbc.recorder.Eventf(ing, api_v1.EventTypeWarning, "Rejected", "%v was rejected: %v", key, err)
			return
		}
		err = lbc.cnf.AddOrUpdateIngress(ingEx)
		if err != nil {
			lbc.recorder.Eventf(ing, api_v1.EventTypeWarning, "AddedOrUpdatedWithError", "Configuration for %v was added or updated, but not applied: %v", key, err)
		} else {
			lbc.recorder.Eventf(ing, api_v1.EventTypeNormal, "AddedOrUpdated", "Configuration for %v was added or updated", key)
		}
	}
}

func (lbc *LoadBalancerController) syncSecret(task Task) {
	key := task.Key
	obj, secrExists, err := lbc.secrLister.Store.GetByKey(key)
	if err != nil {
		lbc.syncQueue.requeue(task, err)
		return
	}

	_, name, err := ParseNamespaceName(key)
	if err != nil {
		glog.Warningf("Secret key %v is invalid: %v", key, err)
		return
	}

	ings, err := lbc.findIngressesForSecret(name)
	if err != nil {
		glog.Warningf("Failed to find Ingress resources for Secret %v: %v", key, err)
		lbc.syncQueue.requeueAfter(task, err, 5*time.Second)
	}

	glog.V(2).Infof("Found %v Ingress resources with Secret %v", len(ings), key)

	if !secrExists {
		glog.V(2).Infof("Deleting Secret: %v\n", key)

		if err := lbc.cnf.DeleteSecret(key, ings); err != nil {
			glog.Errorf("Error when deleting Secret: %v: %v", key, err)
		}

		for _, ing := range ings {
			lbc.syncQueue.enqueue(&ing)
			lbc.recorder.Eventf(&ing, api_v1.EventTypeWarning, "Rejected", "%v/%v was rejected due to deleted Secret %v: %v", ing.Namespace, ing.Name, key)
		}

		if key == lbc.defaultServerSecret {
			glog.Warningf("The default server Secret %v was removed. Retaining the Secret.")
		}
	} else {
		glog.V(2).Infof("Updating Secret: %v\n", key)

		secret := obj.(*api_v1.Secret)

		if key == lbc.defaultServerSecret {
			err := nginx.ValidateTLSSecret(secret)
			if err != nil {
				glog.Errorf("Couldn't validate the default server Secret %v: %v", key, err)
				lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "Rejected", "the default server Secret %v was rejected, using the previous version: %v", key, err)
			} else {
				err := lbc.cnf.AddOrUpdateDefaultServerTLSSecret(secret)
				if err != nil {
					glog.Errorf("Error when updating the default server Secret %v: %v", key, err)
					lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "the default server Secret %v was updated, but not applied: %v", key, err)

				} else {
					lbc.recorder.Eventf(secret, api_v1.EventTypeNormal, "Updated", "the default server Secret %v was updated", key)
				}
			}
		}

		if len(ings) > 0 {
			err := lbc.ValidateSecret(secret)
			if err != nil {
				glog.Errorf("Couldn't validate secret %v: %v", key, err)
				if err := lbc.cnf.DeleteSecret(key, ings); err != nil {
					glog.Errorf("Error when deleting Secret: %v: %v", key, err)
				}
				for _, ing := range ings {
					lbc.syncQueue.enqueue(&ing)
					lbc.recorder.Eventf(&ing, api_v1.EventTypeWarning, "Rejected", "%v/%v was rejected due to invalid Secret %v: %v", ing.Namespace, ing.Name, key, err)
				}
				lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "Rejected", "%v was rejected: %v", key, err)
				return
			}

			if err := lbc.cnf.AddOrUpdateSecret(secret); err != nil {
				glog.Errorf("Error when updating Secret %v: %v", key, err)
				lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "%v was updated, but not applied: %v", key, err)
				for _, ing := range ings {
					lbc.recorder.Eventf(&ing, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration for %v/%v was updated, but not applied: %v", ing.Namespace, ing.Name, err)
				}
			} else {
				lbc.recorder.Eventf(secret, api_v1.EventTypeNormal, "Updated", "%v was updated", key)
				for _, ing := range ings {
					lbc.recorder.Eventf(&ing, api_v1.EventTypeNormal, "Updated", "Configuration for %v/%v was updated", ing.Namespace, ing.Name)
				}
			}
		}
	}
}

func (lbc *LoadBalancerController) findIngressesForSecret(secret string) ([]extensions.Ingress, error) {
	res := []extensions.Ingress{}
	ings, err := lbc.ingLister.List()
	if err != nil {
		return nil, fmt.Errorf("Couldn't get the list of Ingress resources: %v", err)
	}

items:
	for _, ing := range ings.Items {
		if !lbc.isNginxIngress(&ing) {
			continue
		}
		for _, tls := range ing.Spec.TLS {
			if tls.SecretName == secret {
				res = append(res, ing)
				continue items
			}
		}
		if lbc.nginxPlus {
			if jwtKey, exists := ing.Annotations[nginx.JWTKeyAnnotation]; exists {
				if jwtKey == secret {
					res = append(res, ing)
				}
			}
		}
	}

	return res, nil
}

func (lbc *LoadBalancerController) enqueueIngressForService(svc *api_v1.Service) {
	ings := lbc.getIngressesForService(svc)
	for _, ing := range ings {
		if !lbc.isNginxIngress(&ing) {
			continue
		}
		lbc.syncQueue.enqueue(&ing)
	}
}

func (lbc *LoadBalancerController) getIngressesForService(svc *api_v1.Service) []extensions.Ingress {
	ings, err := lbc.ingLister.GetServiceIngress(svc)
	if err != nil {
		glog.V(3).Infof("ignoring service %v: %v", svc.Name, err)
		return nil
	}
	return ings
}

func (lbc *LoadBalancerController) getIngressForEndpoints(obj interface{}) []extensions.Ingress {
	var ings []extensions.Ingress
	endp := obj.(*api_v1.Endpoints)
	svcKey := endp.GetNamespace() + "/" + endp.GetName()
	svcObj, svcExists, err := lbc.svcLister.GetByKey(svcKey)
	if err != nil {
		glog.V(3).Infof("error getting service %v from the cache: %v\n", svcKey, err)
	} else {
		if svcExists {
			ings = append(ings, lbc.getIngressesForService(svcObj.(*api_v1.Service))...)
		}
	}
	return ings
}

func (lbc *LoadBalancerController) createIngress(ing *extensions.Ingress) (*nginx.IngressEx, error) {
	ingEx := &nginx.IngressEx{
		Ingress: ing,
	}

	ingEx.TLSSecrets = make(map[string]*api_v1.Secret)
	for _, tls := range ing.Spec.TLS {
		secretName := tls.SecretName
		secret, err := lbc.client.Core().Secrets(ing.Namespace).Get(secretName, meta_v1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("Error retrieving secret %v for Ingress %v: %v", secretName, ing.Name, err)
		}
		err = nginx.ValidateTLSSecret(secret)
		if err != nil {
			return nil, fmt.Errorf("Error validating secret %v for Ingress %v: %v", secretName, ing.Name, err)
		}
		ingEx.TLSSecrets[secretName] = secret
	}

	if lbc.nginxPlus {
		if jwtKey, exists := ingEx.Ingress.Annotations[nginx.JWTKeyAnnotation]; exists {
			secretName := jwtKey

			secret, err := lbc.client.Core().Secrets(ing.Namespace).Get(secretName, meta_v1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("Error retrieving secret %v for Ingress %v: %v", secretName, ing.Name, err)
			}

			err = nginx.ValidateJWKSecret(secret)
			if err != nil {
				return nil, fmt.Errorf("Error validating secret %v for Ingress %v: %v", secretName, ing.Name, err)
			}

			ingEx.JWTKey = secret
		}
	}

	ingEx.Endpoints = make(map[string][]string)
	if ing.Spec.Backend != nil {
		endps, err := lbc.getEndpointsForIngressBackend(ing.Spec.Backend, ing.Namespace)
		if err != nil {
			glog.Warningf("Error retrieving endpoints for the service %v: %v", ing.Spec.Backend.ServiceName, err)
			ingEx.Endpoints[ing.Spec.Backend.ServiceName+ing.Spec.Backend.ServicePort.String()] = []string{}
		} else {
			ingEx.Endpoints[ing.Spec.Backend.ServiceName+ing.Spec.Backend.ServicePort.String()] = endps
		}
	}

	validRules := 0

	for _, rule := range ing.Spec.Rules {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		if rule.Host == "" {
			return nil, fmt.Errorf("Ingress rule contains empty host")
		}

		for _, path := range rule.HTTP.Paths {
			endps, err := lbc.getEndpointsForIngressBackend(&path.Backend, ing.Namespace)
			if err != nil {
				glog.Warningf("Error retrieving endpoints for the service %v: %v", path.Backend.ServiceName, err)
				ingEx.Endpoints[path.Backend.ServiceName+path.Backend.ServicePort.String()] = []string{}
			} else {
				ingEx.Endpoints[path.Backend.ServiceName+path.Backend.ServicePort.String()] = endps
			}
		}

		validRules++
	}

	if validRules == 0 {
		return nil, fmt.Errorf("Ingress contains no valid rules")
	}

	return ingEx, nil
}

func (lbc *LoadBalancerController) getEndpointsForIngressBackend(backend *extensions.IngressBackend, namespace string) ([]string, error) {
	svc, err := lbc.getServiceForIngressBackend(backend, namespace)
	if err != nil {
		glog.V(3).Infof("Error getting service %v: %v", backend.ServiceName, err)
		return nil, err
	}

	endps, err := lbc.endpLister.GetServiceEndpoints(svc)
	if err != nil {
		glog.V(3).Infof("Error getting endpoints for service %s from the cache: %v", svc.Name, err)
		return nil, err
	}

	result, err := lbc.getEndpointsForPort(endps, backend.ServicePort, svc)
	if err != nil {
		glog.V(3).Infof("Error getting endpoints for service %s port %v: %v", svc.Name, backend.ServicePort, err)
		return nil, err
	}
	return result, nil
}

func (lbc *LoadBalancerController) getEndpointsForPort(endps api_v1.Endpoints, ingSvcPort intstr.IntOrString, svc *api_v1.Service) ([]string, error) {
	var targetPort int32
	var err error
	found := false

	for _, port := range svc.Spec.Ports {
		if (ingSvcPort.Type == intstr.Int && port.Port == int32(ingSvcPort.IntValue())) || (ingSvcPort.Type == intstr.String && port.Name == ingSvcPort.String()) {
			targetPort, err = lbc.getTargetPort(&port, svc)
			if err != nil {
				return nil, fmt.Errorf("Error determining target port for port %v in Ingress: %v", ingSvcPort, err)
			}
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("No port %v in service %s", ingSvcPort, svc.Name)
	}

	for _, subset := range endps.Subsets {
		for _, port := range subset.Ports {
			if port.Port == targetPort {
				var endpoints []string
				for _, address := range subset.Addresses {
					endpoint := fmt.Sprintf("%v:%v", address.IP, port.Port)
					endpoints = append(endpoints, endpoint)
				}
				return endpoints, nil
			}
		}
	}

	return nil, fmt.Errorf("No endpoints for target port %v in service %s", targetPort, svc.Name)
}

func (lbc *LoadBalancerController) getTargetPort(svcPort *api_v1.ServicePort, svc *api_v1.Service) (int32, error) {
	if (svcPort.TargetPort == intstr.IntOrString{}) {
		return svcPort.Port, nil
	}

	if svcPort.TargetPort.Type == intstr.Int {
		return int32(svcPort.TargetPort.IntValue()), nil
	}

	pods, err := lbc.client.Core().Pods(svc.Namespace).List(meta_v1.ListOptions{LabelSelector: labels.Set(svc.Spec.Selector).String()})
	if err != nil {
		return 0, fmt.Errorf("Error getting pod information: %v", err)
	}

	if len(pods.Items) == 0 {
		return 0, fmt.Errorf("No pods of service %s", svc.Name)
	}

	pod := &pods.Items[0]

	portNum, err := FindPort(pod, svcPort)
	if err != nil {
		return 0, fmt.Errorf("Error finding named port %v in pod %s: %v", svcPort, pod.Name, err)
	}

	return portNum, nil
}

func (lbc *LoadBalancerController) getServiceForIngressBackend(backend *extensions.IngressBackend, namespace string) (*api_v1.Service, error) {
	svcKey := namespace + "/" + backend.ServiceName
	svcObj, svcExists, err := lbc.svcLister.GetByKey(svcKey)
	if err != nil {
		return nil, err
	}

	if svcExists {
		return svcObj.(*api_v1.Service), nil
	}

	return nil, fmt.Errorf("service %s doesn't exists", svcKey)
}

// ParseNamespaceName parses the string in the <namespace>/<name> format and returns the name and the namespace.
// It returns an error in case the string does not follow the <namespace>/<name> format.
func ParseNamespaceName(value string) (ns string, name string, err error) {
	res := strings.Split(value, "/")
	if len(res) != 2 {
		return "", "", fmt.Errorf("%q must follow the format <namespace>/<name>", value)
	}
	return res[0], res[1], nil
}

// Check if resource ingress class annotatios (if exist) is matching with ingress controller class
// If annotatins is absent and use-ingress-class-only enabled - ingress resource would ignore
func (lbc *LoadBalancerController) isNginxIngress(ing *extensions.Ingress) bool {
	if class, exists := ing.Annotations[ingressClassKey]; exists {
		if lbc.useIngressClassOnly {
			return class == lbc.ingressClass
		}
		return class == lbc.ingressClass || class == ""
	} else {
		return ! lbc.useIngressClassOnly
	}
}

// ValidateSecret validates that the secret follows the TLS Secret format.
// For NGINX Plus, it also checks if the secret follows the JWK Secret format.
func (lbc *LoadBalancerController) ValidateSecret(secret *api_v1.Secret) error {
	err1 := nginx.ValidateTLSSecret(secret)
	if !lbc.nginxPlus {
		return err1
	}

	err2 := nginx.ValidateJWKSecret(secret)

	if err1 == nil || err2 == nil {
		return nil
	}

	return fmt.Errorf("Secret is not a TLS or JWK secret")
}

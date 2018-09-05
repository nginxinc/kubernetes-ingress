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

	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	scheme "k8s.io/client-go/kubernetes/scheme"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"

	"sort"

	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ingressClassKey = "kubernetes.io/ingress.class"
)

// LoadBalancerController watches Kubernetes API and
// reconfigures NGINX via NginxController when needed
type LoadBalancerController struct {
	client                kubernetes.Interface
	ingController         cache.Controller
	svcController         cache.Controller
	endpController        cache.Controller
	cfgmController        cache.Controller
	secrController        cache.Controller
	ingLister             StoreToIngressLister
	svcLister             cache.Store
	endpLister            StoreToEndpointLister
	cfgmLister            StoreToConfigMapLister
	secrLister            StoreToSecretLister
	syncQueue             *taskQueue
	stopCh                chan struct{}
	cnf                   *nginx.Configurator
	watchNginxConfigMaps  bool
	nginxPlus             bool
	recorder              record.EventRecorder
	defaultServerSecret   string
	ingressClass          string
	useIngressClassOnly   bool
	statusUpdater         *StatusUpdater
	leaderElector         *leaderelection.LeaderElector
	reportIngressStatus   bool
	leaderElectionEnabled bool
}

var keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

// NewLoadBalancerControllerInput holds the input needed to call NewLoadBalancerController.
type NewLoadBalancerControllerInput struct {
	KubeClient            kubernetes.Interface
	ResyncPeriod          time.Duration
	Namespace             string
	CNF                   *nginx.Configurator
	NginxConfigMaps       string
	DefaultServerSecret   string
	NginxPlus             bool
	IngressClass          string
	UseIngressClassOnly   bool
	ExternalServiceName   string
	ControllerNamespace   string
	ReportIngressStatus   bool
	LeaderElectionEnabled bool
}

// NewLoadBalancerController creates a controller
func NewLoadBalancerController(input NewLoadBalancerControllerInput) *LoadBalancerController {
	lbc := LoadBalancerController{
		client:                input.KubeClient,
		stopCh:                make(chan struct{}),
		cnf:                   input.CNF,
		defaultServerSecret:   input.DefaultServerSecret,
		nginxPlus:             input.NginxPlus,
		ingressClass:          input.IngressClass,
		useIngressClassOnly:   input.UseIngressClassOnly,
		reportIngressStatus:   input.ReportIngressStatus,
		leaderElectionEnabled: input.LeaderElectionEnabled,
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&core_v1.EventSinkImpl{
		Interface: core_v1.New(input.KubeClient.Core().RESTClient()).Events(""),
	})
	lbc.recorder = eventBroadcaster.NewRecorder(scheme.Scheme,
		api_v1.EventSource{Component: "nginx-ingress-controller"})

	lbc.syncQueue = NewTaskQueue(lbc.sync)

	glog.V(3).Infof("Nginx Ingress Controller has class: %v", input.IngressClass)

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
			if isMinion(remIng) {
				master, err := lbc.findMasterForMinion(remIng)
				if err != nil {
					glog.Infof("Ignoring Ingress %v(Minion): %v", remIng.Name, err)
					return
				}
				glog.V(3).Infof("Removing Ingress: %v(Minion) for %v(Master)", remIng.Name, master.Name)
				lbc.syncQueue.enqueue(master)
			} else {
				glog.V(3).Infof("Removing Ingress: %v", remIng.Name)
				lbc.syncQueue.enqueue(obj)
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			curIng := cur.(*extensions.Ingress)
			oldIng := old.(*extensions.Ingress)
			if !lbc.isNginxIngress(curIng) {
				return
			}
			if hasChanges(oldIng, curIng) {
				glog.V(3).Infof("Ingress %v changed, syncing", curIng.Name)
				lbc.syncQueue.enqueue(cur)
			}
		},
	}
	lbc.ingLister.Store, lbc.ingController = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.Extensions().RESTClient(), "ingresses", input.Namespace, fields.Everything()),
		&extensions.Ingress{}, input.ResyncPeriod, ingHandlers)

	// statusUpdater requires ingLister to be instantiated, above.
	lbc.statusUpdater = &StatusUpdater{
		client:              input.KubeClient,
		namespace:           input.ControllerNamespace,
		externalServiceName: input.ExternalServiceName,
		ingLister:           &lbc.ingLister,
		keyFunc:             keyFunc,
	}

	if input.ReportIngressStatus && input.LeaderElectionEnabled {
		leaderCallbacks := leaderelection.LeaderCallbacks{
			OnStartedLeading: func(stop <-chan struct{}) {
				glog.V(3).Info("started leading, updating ingress status")
				ingresses, mergeableIngresses := lbc.getManagedIngresses()
				err := lbc.statusUpdater.UpdateManagedAndMergeableIngresses(ingresses, mergeableIngresses)
				if err != nil {
					glog.V(3).Infof("error updating status when starting leading: %v", err)
				}
			},
		}

		var err error
		lbc.leaderElector, err = NewLeaderElector(input.KubeClient, leaderCallbacks, input.ControllerNamespace)
		if err != nil {
			glog.V(3).Infof("Error starting LeaderElection: %v", err)
		}
	}

	svcHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addSvc := obj.(*api_v1.Service)
			if lbc.isExternalServiceForStatus(addSvc) {
				lbc.syncQueue.enqueue(addSvc)
				return
			}
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
			if lbc.isExternalServiceForStatus(remSvc) {
				lbc.syncQueue.enqueue(remSvc)
				return
			}

			glog.V(3).Infof("Removing service: %v", remSvc.Name)
			lbc.enqueueIngressForService(remSvc)

		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				curSvc := cur.(*api_v1.Service)
				if lbc.isExternalServiceForStatus(curSvc) {
					lbc.syncQueue.enqueue(curSvc)
					return
				}
				glog.V(3).Infof("Service %v changed, syncing", curSvc.Name)
				lbc.enqueueIngressForService(curSvc)
			}
		},
	}
	lbc.svcLister, lbc.svcController = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.Core().RESTClient(), "services", input.Namespace, fields.Everything()),
		&api_v1.Service{}, input.ResyncPeriod, svcHandlers)

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
		cache.NewListWatchFromClient(lbc.client.Core().RESTClient(), "endpoints", input.Namespace, fields.Everything()),
		&api_v1.Endpoints{}, input.ResyncPeriod, endpHandlers)

	secrHandlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secr := obj.(*api_v1.Secret)
			if err := lbc.ValidateSecret(secr); err != nil {
				return
			}
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
		cache.NewListWatchFromClient(lbc.client.Core().RESTClient(), "secrets", input.Namespace, fields.Everything()),
		&api_v1.Secret{}, input.ResyncPeriod, secrHandlers)

	if input.NginxConfigMaps != "" {
		nginxConfigMapsNS, nginxConfigMapsName, err := ParseNamespaceName(input.NginxConfigMaps)
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
							glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-ConfigMap object: %v", deletedState.Obj)
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
				&api_v1.ConfigMap{}, input.ResyncPeriod, cfgmHandlers)
		}
	}

	return &lbc
}

// hasChanges ignores Status or ResourceVersion changes
func hasChanges(oldIng *extensions.Ingress, curIng *extensions.Ingress) bool {
	oldIng.Status.LoadBalancer.Ingress = curIng.Status.LoadBalancer.Ingress
	oldIng.ResourceVersion = curIng.ResourceVersion
	return !reflect.DeepEqual(oldIng, curIng)
}

// Run starts the loadbalancer controller
func (lbc *LoadBalancerController) Run() {
	if lbc.leaderElector != nil {
		go lbc.leaderElector.Run()
	}
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

		for _, ing := range ings {
			if !lbc.isNginxIngress(&ing) {
				continue
			}
			if isMinion(&ing) {
				master, err := lbc.findMasterForMinion(&ing)
				if err != nil {
					glog.Errorf("Ignoring Ingress %v(Minion): %v", ing.Name, err)
					continue
				}
				if !lbc.cnf.HasIngress(master) {
					continue
				}
				mergeableIngresses, err := lbc.createMergableIngresses(master)
				if err != nil {
					glog.Errorf("Ignoring Ingress %v(Minion): %v", ing.Name, err)
					continue
				}

				glog.V(3).Infof("Updating Endpoints for %v/%v", ing.Namespace, ing.Name)
				err = lbc.cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
				if err != nil {
					glog.Errorf("Error updating endpoints for %v/%v: %v", ing.Namespace, ing.Name, err)
				}
				continue
			}
			if !lbc.cnf.HasIngress(&ing) {
				continue
			}
			ingEx, err := lbc.createIngress(&ing)
			if err != nil {
				glog.Errorf("Error updating endpoints for %v/%v: %v, skipping", ing.Namespace, ing.Name, err)
				continue
			}
			glog.V(3).Infof("Updating Endpoints for %v/%v", ing.Namespace, ing.Name)
			err = lbc.cnf.UpdateEndpoints(ingEx)
			if err != nil {
				glog.Errorf("Error updating endpoints for %v/%v: %v", ing.Namespace, ing.Name, err)
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
		cfg = nginx.ParseConfigMap(cfgm, lbc.nginxPlus)

		lbc.statusUpdater.SaveStatusFromExternalStatus(cfgm.Data["external-status-address"])
	}

	ingresses, mergeableIngresses := lbc.getManagedIngresses()
	ingExes := lbc.ingressesToIngressExes(ingresses)

	if lbc.reportStatusEnabled() {
		err = lbc.statusUpdater.UpdateManagedAndMergeableIngresses(ingresses, mergeableIngresses)
		if err != nil {
			glog.V(3).Infof("error updating status on ConfigMap change: %v", err)
		}
	}

	if err := lbc.cnf.UpdateConfig(cfg, ingExes, mergeableIngresses); err != nil {
		if cfgmExists {
			cfgm := obj.(*api_v1.ConfigMap)
			lbc.recorder.Eventf(cfgm, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration from %v was updated, but not applied: %v", key, err)
		}
		for _, ingEx := range ingExes {
			lbc.recorder.Eventf(ingEx.Ingress, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration for %v/%v was updated, but not applied: %v",
				ingEx.Ingress.Namespace, ingEx.Ingress.Name, err)
		}
		for _, mergeableIng := range mergeableIngresses {
			master := mergeableIng.Master
			lbc.recorder.Eventf(master.Ingress, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration for %v/%v(Master) was updated, but not applied: %v",
				master.Ingress.Namespace, master.Ingress.Name, err)
			for _, minion := range mergeableIng.Minions {
				lbc.recorder.Eventf(minion.Ingress, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration for %v/%v(Minion) was updated, but not applied: %v",
					minion.Ingress.Namespace, minion.Ingress.Name, err)
			}
		}
	} else {
		if cfgmExists {
			cfgm := obj.(*api_v1.ConfigMap)
			lbc.recorder.Eventf(cfgm, api_v1.EventTypeNormal, "Updated", "Configuration from %v was updated", key)
		}
		for _, ingEx := range ingExes {
			lbc.recorder.Eventf(ingEx.Ingress, api_v1.EventTypeNormal, "Updated", "Configuration for %v/%v was updated", ingEx.Ingress.Namespace, ingEx.Ingress.Name)
		}
		for _, mergeableIng := range mergeableIngresses {
			master := mergeableIng.Master
			lbc.recorder.Eventf(master.Ingress, api_v1.EventTypeWarning, "Updated", "Configuration for %v/%v(Master) was updated", master.Ingress.Namespace, master.Ingress.Name)
			for _, minion := range mergeableIng.Minions {
				lbc.recorder.Eventf(minion.Ingress, api_v1.EventTypeWarning, "Updated", "Configuration for %v/%v(Minion) was updated", minion.Ingress.Namespace, minion.Ingress.Name)
			}
		}
	}
}

// getManagedIngresses gets Ingress resources that the IC is currently responsible for
func (lbc *LoadBalancerController) getManagedIngresses() ([]extensions.Ingress, map[string]*nginx.MergeableIngresses) {
	mergeableIngresses := make(map[string]*nginx.MergeableIngresses)
	var managedIngresses []extensions.Ingress
	ings, _ := lbc.ingLister.List()
	for i := range ings.Items {
		ing := ings.Items[i]
		if !lbc.isNginxIngress(&ing) {
			continue
		}
		if isMinion(&ing) {
			master, err := lbc.findMasterForMinion(&ing)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", ing, err)
				continue
			}
			if !lbc.cnf.HasIngress(master) {
				continue
			}
			if _, exists := mergeableIngresses[master.Name]; !exists {
				mergeableIngress, err := lbc.createMergableIngresses(master)
				if err != nil {
					glog.Errorf("Ignoring Ingress %v(Master): %v", master, err)
					continue
				}
				mergeableIngresses[master.Name] = mergeableIngress
			}
			continue
		}
		if !lbc.cnf.HasIngress(&ing) {
			continue
		}
		managedIngresses = append(managedIngresses, ing)
	}
	return managedIngresses, mergeableIngresses
}

func (lbc *LoadBalancerController) ingressesToIngressExes(ings []extensions.Ingress) []*nginx.IngressEx {
	var ingExes []*nginx.IngressEx
	for _, ing := range ings {
		ingEx, err := lbc.createIngress(&ing)
		if err != nil {
			continue
		}
		ingExes = append(ingExes, ingEx)
	}
	return ingExes
}

func (lbc *LoadBalancerController) sync(task Task) {
	glog.V(3).Infof("Syncing %v", task.Key)

	switch task.Kind {
	case Ingress:
		lbc.syncIng(task)
	case IngressMinion:
		lbc.syncIngMinion(task)
	case ConfigMap:
		lbc.syncCfgm(task)
		return
	case Endpoints:
		lbc.syncEndp(task)
		return
	case Secret:
		lbc.syncSecret(task)
		return
	case Service:
		lbc.syncExternalService(task)
	}
}

func (lbc *LoadBalancerController) syncIngMinion(task Task) {
	key := task.Key
	obj, ingExists, err := lbc.ingLister.Store.GetByKey(key)
	if err != nil {
		lbc.syncQueue.requeue(task, err)
		return
	}

	if !ingExists {
		glog.V(2).Infof("Minion was deleted: %v\n", key)
		return
	}
	glog.V(2).Infof("Adding or Updating Minion: %v\n", key)

	minion := obj.(*extensions.Ingress)

	master, err := lbc.findMasterForMinion(minion)
	if err != nil {
		lbc.syncQueue.requeueAfter(task, err, 5*time.Second)
		return
	}

	_, err = lbc.createIngress(minion)
	if err != nil {
		lbc.syncQueue.requeueAfter(task, err, 5*time.Second)
		if !lbc.cnf.HasMinion(master, minion) {
			return
		}
	}

	lbc.syncQueue.enqueue(master)
}

func (lbc *LoadBalancerController) syncIng(task Task) {
	key := task.Key
	ing, ingExists, err := lbc.ingLister.GetByKeySafe(key)
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

		if isMaster(ing) {
			mergeableIngExs, err := lbc.createMergableIngresses(ing)
			if err != nil {
				lbc.syncQueue.requeueAfter(task, err, 5*time.Second)
				lbc.recorder.Eventf(ing, api_v1.EventTypeWarning, "Rejected", "%v was rejected: %v", key, err)
				if lbc.reportStatusEnabled() {
					err = lbc.statusUpdater.ClearIngressStatus(*ing)
					if err != nil {
						glog.V(3).Infof("error clearing ing status: %v", err)
					}
				}
				return
			}
			err = lbc.cnf.AddOrUpdateMergeableIngress(mergeableIngExs)
			if err != nil {
				lbc.recorder.Eventf(ing, api_v1.EventTypeWarning, "AddedOrUpdatedWithError", "Configuration for %v(Master) was added or updated, but not applied: %v", key, err)
				for _, minion := range mergeableIngExs.Minions {
					lbc.recorder.Eventf(ing, api_v1.EventTypeWarning, "AddedOrUpdatedWithError", "Configuration for %v/%v(Minion) was added or updated, but not applied: %v", minion.Ingress.Namespace, minion.Ingress.Name, err)
				}
			} else {
				lbc.recorder.Eventf(ing, api_v1.EventTypeNormal, "AddedOrUpdated", "Configuration for %v(Master) was added or updated", key)
				for _, minion := range mergeableIngExs.Minions {
					lbc.recorder.Eventf(ing, api_v1.EventTypeNormal, "AddedOrUpdated", "Configuration for %v/%v(Minion) was added or updated", minion.Ingress.Namespace, minion.Ingress.Name)
				}
			}
			if lbc.reportStatusEnabled() {
				err = lbc.statusUpdater.UpdateMergableIngresses(mergeableIngExs)
				if err != nil {
					glog.V(3).Infof("error updating ing status: %v", err)
				}
			}
			return
		}
		ingEx, err := lbc.createIngress(ing)
		if err != nil {
			lbc.syncQueue.requeueAfter(task, err, 5*time.Second)
			lbc.recorder.Eventf(ing, api_v1.EventTypeWarning, "Rejected", "%v was rejected: %v", key, err)
			if lbc.reportStatusEnabled() {
				err = lbc.statusUpdater.ClearIngressStatus(*ing)
				if err != nil {
					glog.V(3).Infof("error clearing ing status: %v", err)
				}
			}
			return
		}

		err = lbc.cnf.AddOrUpdateIngress(ingEx)
		if err != nil {
			lbc.recorder.Eventf(ing, api_v1.EventTypeWarning, "AddedOrUpdatedWithError", "Configuration for %v was added or updated, but not applied: %v", key, err)
		} else {
			lbc.recorder.Eventf(ing, api_v1.EventTypeNormal, "AddedOrUpdated", "Configuration for %v was added or updated", key)
		}
		if lbc.reportStatusEnabled() {
			err = lbc.statusUpdater.UpdateIngressStatus(*ing)
			if err != nil {
				glog.V(3).Infof("error updating ing status: %v", err)
			}
		}
	}
}

// syncExternalService does not sync all services.
// We only watch the Service specified by the external-service flag.
func (lbc *LoadBalancerController) syncExternalService(task Task) {
	key := task.Key
	obj, exists, err := lbc.svcLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.requeue(task, err)
		return
	}
	statusIngs, mergableIngs := lbc.getManagedIngresses()
	if !exists {
		// service got removed
		lbc.statusUpdater.ClearStatusFromExternalService()
	} else {
		// service added or updated
		lbc.statusUpdater.SaveStatusFromExternalService(obj.(*api_v1.Service))
	}
	if lbc.reportStatusEnabled() {
		err = lbc.statusUpdater.UpdateManagedAndMergeableIngresses(statusIngs, mergableIngs)
		if err != nil {
			glog.Errorf("error updating ingress status in syncExternalService: %v", err)
		}
	}
}

// isExternalServiceForStatus matches the service specified by the external-service arg
func (lbc *LoadBalancerController) isExternalServiceForStatus(svc *api_v1.Service) bool {
	return lbc.statusUpdater.namespace == svc.Namespace && lbc.statusUpdater.externalServiceName == svc.Name
}

// reportStatusEnabled determines if we should attempt to report status
func (lbc *LoadBalancerController) reportStatusEnabled() bool {
	if lbc.reportIngressStatus {
		if lbc.leaderElectionEnabled {
			return lbc.leaderElector != nil && lbc.leaderElector.IsLeader()
		}
		return true
	}
	return false
}

func (lbc *LoadBalancerController) syncSecret(task Task) {
	key := task.Key
	obj, secrExists, err := lbc.secrLister.Store.GetByKey(key)
	if err != nil {
		lbc.syncQueue.requeue(task, err)
		return
	}

	namespace, name, err := ParseNamespaceName(key)
	if err != nil {
		glog.Warningf("Secret key %v is invalid: %v", key, err)
		return
	}

	nonMinions, minions, err := lbc.findIngressesForSecret(namespace, name)
	if err != nil {
		glog.Warningf("Failed to find Ingress resources for Secret %v: %v", key, err)
		lbc.syncQueue.requeueAfter(task, err, 5*time.Second)
	}

	glog.V(2).Infof("Found %v Non-Minion and %v Minion Ingress resources with Secret %v", len(nonMinions), len(minions), key)

	if !secrExists {
		glog.V(2).Infof("Deleting Secret: %v\n", key)

		for _, minion := range minions {
			master, err := lbc.findMasterForMinion(&minion)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", minion.Name, err)
				continue
			}
			mergeableIngress, err := lbc.createMergableIngresses(master)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", minion.Name, err)
				continue
			}
			err = lbc.cnf.AddOrUpdateMergeableIngress(mergeableIngress)
			if err != nil {
				glog.Errorf("Failed to update Ingress %v(Master) of %v(Minion): %v", master.Name, minion.Name, err)
			}
			lbc.recorder.Eventf(&minion, api_v1.EventTypeWarning, "Rejected", "%v/%v was rejected due to deleted Secret %v: %v", minion.Namespace, minion.Name, key)
			lbc.recorder.Eventf(master, api_v1.EventTypeWarning, "Rejected", "Minion %v/%v was rejected due to deleted Secret %v: %v", minion.Namespace, minion.Name, key)
			lbc.syncQueue.enqueue(&minion)
		}

		if err := lbc.cnf.DeleteSecret(key, nonMinions); err != nil {
			glog.Errorf("Error when deleting Secret: %v: %v", key, err)
		}

		for _, ing := range nonMinions {
			lbc.syncQueue.enqueue(&ing)
			lbc.recorder.Eventf(&ing, api_v1.EventTypeWarning, "Rejected", "%v/%v was rejected due to deleted Secret %v", ing.Namespace, ing.Name, key)
		}

		if key == lbc.defaultServerSecret {
			glog.Warningf("The default server Secret %v was removed. Retaining the Secret.", key)
		}
	} else {
		glog.V(2).Infof("Adding / Updating Secret: %v\n", key)

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

		if len(nonMinions) > 0 || len(minions) > 0 {
			err := lbc.ValidateSecret(secret)
			if err != nil {
				// Secret becomes Invalid
				glog.Errorf("Couldn't validate secret %v: %v", key, err)

				for _, minion := range minions {
					master, err := lbc.findMasterForMinion(&minion)
					if err != nil {
						glog.Errorf("Ignoring Ingress %v(Minion): %v", minion.Name, err)
						continue
					}
					mergeableIngress, err := lbc.createMergableIngresses(master)
					if err != nil {
						glog.Errorf("Ignoring Ingress %v(Minion): %v", minion.Name, err)
						continue
					}
					err = lbc.cnf.AddOrUpdateMergeableIngress(mergeableIngress)
					if err != nil {
						glog.Errorf("Failed to update Ingress %v(Master) of %v(Minion): %v", master.Name, minion.Name, err)
					}
					lbc.recorder.Eventf(&minion, api_v1.EventTypeWarning, "Rejected", "%v/%v was rejected due to invalid Secret %v: %v", minion.Namespace, minion.Name, key, err)
					lbc.recorder.Eventf(master, api_v1.EventTypeWarning, "Rejected", "Minion %v/%v was rejected due to invalid Secret %v: %v", minion.Namespace, minion.Name, key, err)
					lbc.syncQueue.enqueue(&minion)
				}

				if err := lbc.cnf.DeleteSecret(key, nonMinions); err != nil {
					glog.Errorf("Error when deleting Secret: %v: %v", key, err)
				}
				for _, ing := range nonMinions {
					lbc.syncQueue.enqueue(&ing)
					lbc.recorder.Eventf(&ing, api_v1.EventTypeWarning, "Rejected", "%v/%v was rejected due to invalid Secret %v: %v", ing.Namespace, ing.Name, key, err)
				}
				lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "Rejected", "%v was rejected: %v", key, err)
				return
			}

			if err := lbc.cnf.AddOrUpdateSecret(secret); err != nil {
				glog.Errorf("Error when updating Secret %v: %v", key, err)
				lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "%v was updated, but not applied: %v", key, err)
				for _, ing := range nonMinions {
					lbc.recorder.Eventf(&ing, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration for %v/%v was updated, but not applied: %v", ing.Namespace, ing.Name, err)
				}
				for _, minion := range minions {
					master, err := lbc.findMasterForMinion(&minion)
					if err != nil {
						glog.Errorf("Ignoring Ingress %v(Minion): %v", minion.Name, err)
						continue
					}
					lbc.recorder.Eventf(master, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration for  minion %v/%v was updated, but not applied: %v", minion.Namespace, minion.Name, err)
					lbc.recorder.Eventf(&minion, api_v1.EventTypeWarning, "UpdatedWithError", "Configuration for %v/%v was updated, but not applied: %v", minion.Namespace, minion.Name, err)
				}
			} else {
				lbc.recorder.Eventf(secret, api_v1.EventTypeNormal, "Updated", "%v was updated", key)
				for _, ing := range nonMinions {
					lbc.recorder.Eventf(&ing, api_v1.EventTypeNormal, "Updated", "Configuration for %v/%v was updated", ing.Namespace, ing.Name)
				}
				for _, minion := range minions {
					master, err := lbc.findMasterForMinion(&minion)
					if err != nil {
						glog.Errorf("Ignoring Ingress %v(Minion): %v", minion.Name, err)
						continue
					}
					lbc.recorder.Eventf(master, api_v1.EventTypeNormal, "Updated", "Configuration for minion %v/%v was updated", minion.Namespace, minion.Name)
					lbc.recorder.Eventf(&minion, api_v1.EventTypeNormal, "Updated", "Configuration for %v/%v was updated", minion.Namespace, minion.Name)
				}
			}
		}
	}
}

func (lbc *LoadBalancerController) findIngressesForSecret(secretNamespace string, secretName string) (nonMinions []extensions.Ingress, minions []extensions.Ingress, error error) {
	ings, err := lbc.ingLister.List()
	if err != nil {
		return nil, nil, fmt.Errorf("Couldn't get the list of Ingress resources: %v", err)
	}

items:
	for _, ing := range ings.Items {
		if ing.Namespace != secretNamespace {
			continue
		}

		if !lbc.isNginxIngress(&ing) {
			continue
		}

		if !isMinion(&ing) {
			if !lbc.cnf.HasIngress(&ing) {
				continue
			}
			for _, tls := range ing.Spec.TLS {
				if tls.SecretName == secretName {
					nonMinions = append(nonMinions, ing)
					continue items
				}
			}
			if lbc.nginxPlus {
				if jwtKey, exists := ing.Annotations[nginx.JWTKeyAnnotation]; exists {
					if jwtKey == secretName {
						nonMinions = append(nonMinions, ing)
					}
				}
			}
			continue
		}

		// we're dealing with a minion
		// minions can only have JWT secrets
		if lbc.nginxPlus {
			master, err := lbc.findMasterForMinion(&ing)
			if err != nil {
				glog.Infof("Ignoring Ingress %v(Minion): %v", ing.Name, err)
				continue
			}

			if !lbc.cnf.HasMinion(master, &ing) {
				continue
			}

			if jwtKey, exists := ing.Annotations[nginx.JWTKeyAnnotation]; exists {
				if jwtKey == secretName {
					minions = append(minions, ing)
				}
			}
		}
	}

	return nonMinions, minions, nil
}

func (lbc *LoadBalancerController) enqueueIngressForService(svc *api_v1.Service) {
	ings := lbc.getIngressesForService(svc)
	for _, ing := range ings {
		if !lbc.isNginxIngress(&ing) {
			continue
		}
		if isMinion(&ing) {
			master, err := lbc.findMasterForMinion(&ing)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", ing.Name, err)
				continue
			}
			ing = *master
		}
		if !lbc.cnf.HasIngress(&ing) {
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
	ingEx.HealthChecks = make(map[string]*api_v1.Probe)

	if ing.Spec.Backend != nil {
		endps, err := lbc.getEndpointsForIngressBackend(ing.Spec.Backend, ing.Namespace)
		if err != nil {
			glog.Warningf("Error retrieving endpoints for the service %v: %v", ing.Spec.Backend.ServiceName, err)
			ingEx.Endpoints[ing.Spec.Backend.ServiceName+ing.Spec.Backend.ServicePort.String()] = []string{}
		} else {
			ingEx.Endpoints[ing.Spec.Backend.ServiceName+ing.Spec.Backend.ServicePort.String()] = endps
		}
		if lbc.nginxPlus && lbc.isHealthCheckEnabled(ing) {
			healthCheck := lbc.getHealthChecksForIngressBackend(ing.Spec.Backend, ing.Namespace)
			if healthCheck != nil {
				ingEx.HealthChecks[ing.Spec.Backend.ServiceName+ing.Spec.Backend.ServicePort.String()] = healthCheck
			}
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
			if lbc.nginxPlus && lbc.isHealthCheckEnabled(ing) {
				// Pull active health checks from k8 api
				healthCheck := lbc.getHealthChecksForIngressBackend(&path.Backend, ing.Namespace)
				if healthCheck != nil {
					ingEx.HealthChecks[path.Backend.ServiceName+path.Backend.ServicePort.String()] = healthCheck
				}
			}
		}
		validRules++
	}

	if validRules == 0 {
		return nil, fmt.Errorf("Ingress contains no valid rules")
	}

	return ingEx, nil
}

func (lbc *LoadBalancerController) getPodsForIngressBackend(svc *api_v1.Service, namespace string) *api_v1.PodList {
	pods, err := lbc.client.CoreV1().Pods(svc.Namespace).List(meta_v1.ListOptions{LabelSelector: labels.Set(svc.Spec.Selector).String()})
	if err != nil {
		glog.V(3).Infof("Error fetching pods for namespace %v: %v", svc.Namespace, err)
		return nil
	}
	return pods
}

func (lbc *LoadBalancerController) getHealthChecksForIngressBackend(backend *extensions.IngressBackend, namespace string) *api_v1.Probe {
	svc, err := lbc.getServiceForIngressBackend(backend, namespace)
	if err != nil {
		glog.V(3).Infof("Error getting service %v: %v", backend.ServiceName, err)
		return nil
	}
	svcPort := lbc.getServicePortForIngressPort(backend.ServicePort, svc)
	if svcPort == nil {
		return nil
	}
	pods := lbc.getPodsForIngressBackend(svc, namespace)
	if pods == nil {
		return nil
	}
	return findProbeForPods(pods.Items, svcPort)
}

func findProbeForPods(pods []api_v1.Pod, svcPort *api_v1.ServicePort) *api_v1.Probe {
	if len(pods) > 0 {
		pod := pods[0]
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				if compareContainerPortAndServicePort(port, *svcPort) {
					// only http ReadinessProbes are useful for us
					if container.ReadinessProbe.Handler.HTTPGet != nil && container.ReadinessProbe.PeriodSeconds > 0 {
						return container.ReadinessProbe
					}
				}
			}
		}
	}
	return nil
}

func compareContainerPortAndServicePort(containerPort api_v1.ContainerPort, svcPort api_v1.ServicePort) bool {
	targetPort := svcPort.TargetPort
	if (targetPort == intstr.IntOrString{}) {
		return svcPort.Port > 0 && svcPort.Port == containerPort.ContainerPort
	}
	switch targetPort.Type {
	case intstr.String:
		return targetPort.StrVal == containerPort.Name && svcPort.Protocol == containerPort.Protocol
	case intstr.Int:
		return targetPort.IntVal > 0 && targetPort.IntVal == containerPort.ContainerPort
	}
	return false
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

func (lbc *LoadBalancerController) getServicePortForIngressPort(ingSvcPort intstr.IntOrString, svc *api_v1.Service) *api_v1.ServicePort {
	for _, port := range svc.Spec.Ports {
		if (ingSvcPort.Type == intstr.Int && port.Port == int32(ingSvcPort.IntValue())) || (ingSvcPort.Type == intstr.String && port.Name == ingSvcPort.String()) {
			return &port
		}
	}
	return nil
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

// Check if resource ingress class annotation (if exists) is matching with ingress controller class
// If annotation is absent and use-ingress-class-only enabled - ingress resource would ignore
func (lbc *LoadBalancerController) isNginxIngress(ing *extensions.Ingress) bool {
	if class, exists := ing.Annotations[ingressClassKey]; exists {
		if lbc.useIngressClassOnly {
			return class == lbc.ingressClass
		}
		return class == lbc.ingressClass || class == ""
	}
	return !lbc.useIngressClassOnly

}

// isHealthCheckEnabled checks if health checks are enabled so we can only query pods if enabled.
func (lbc *LoadBalancerController) isHealthCheckEnabled(ing *extensions.Ingress) bool {
	if healthCheckEnabled, exists, err := nginx.GetMapKeyAsBool(ing.Annotations, "nginx.com/health-checks", ing); exists {
		if err != nil {
			glog.Error(err)
		}
		return healthCheckEnabled
	}
	return false
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

// getMinionsForHost returns a list of all minion ingress resources for a given master
func (lbc *LoadBalancerController) getMinionsForMaster(master *nginx.IngressEx) ([]*nginx.IngressEx, error) {
	ings, err := lbc.ingLister.List()
	if err != nil {
		return []*nginx.IngressEx{}, err
	}

	// ingresses are sorted by creation time
	sort.Slice(ings.Items[:], func(i, j int) bool {
		return ings.Items[i].CreationTimestamp.Time.UnixNano() < ings.Items[j].CreationTimestamp.Time.UnixNano()
	})

	var minions []*nginx.IngressEx
	var minionPaths = make(map[string]*extensions.Ingress)

	for i, _ := range ings.Items {
		if !lbc.isNginxIngress(&ings.Items[i]) {
			continue
		}
		if !isMinion(&ings.Items[i]) {
			continue
		}
		if ings.Items[i].Spec.Rules[0].Host != master.Ingress.Spec.Rules[0].Host {
			continue
		}
		if len(ings.Items[i].Spec.Rules) != 1 {
			glog.Errorf("Ingress Resource %v/%v with the 'nginx.org/mergeable-ingress-type' annotation must contain only one host", ings.Items[i].Namespace, ings.Items[i].Name)
			continue
		}
		if ings.Items[i].Spec.Rules[0].HTTP == nil {
			glog.Errorf("Ingress Resource %v/%v with the 'nginx.org/mergeable-ingress-type' annotation set to 'minion' must contain a Path", ings.Items[i].Namespace, ings.Items[i].Name)
			continue
		}

		uniquePaths := []extensions.HTTPIngressPath{}
		for _, path := range ings.Items[i].Spec.Rules[0].HTTP.Paths {
			if val, ok := minionPaths[path.Path]; ok {
				glog.Errorf("Ingress Resource %v/%v with the 'nginx.org/mergeable-ingress-type' annotation set to 'minion' cannot contain the same path as another ingress resource, %v/%v.",
					ings.Items[i].Namespace, ings.Items[i].Name, val.Namespace, val.Name)
				glog.Errorf("Path %s for Ingress Resource %v/%v will be ignored", path.Path, ings.Items[i].Namespace, ings.Items[i].Name)
			} else {
				minionPaths[path.Path] = &ings.Items[i]
				uniquePaths = append(uniquePaths, path)
			}
		}
		ings.Items[i].Spec.Rules[0].HTTP.Paths = uniquePaths

		ingEx, err := lbc.createIngress(&ings.Items[i])
		if err != nil {
			glog.Errorf("Error creating ingress resource %v/%v: %v", ings.Items[i].Namespace, ings.Items[i].Name, err)
			continue
		}
		if len(ingEx.TLSSecrets) > 0 {
			glog.Errorf("Ingress Resource %v/%v with the 'nginx.org/mergeable-ingress-type' annotation set to 'minion' cannot contain TLS Secrets", ingEx.Ingress.Namespace, ingEx.Ingress.Name)
			continue
		}
		minions = append(minions, ingEx)
	}

	return minions, nil
}

// findMasterForHost returns a master for a given minion
func (lbc *LoadBalancerController) findMasterForMinion(minion *extensions.Ingress) (*extensions.Ingress, error) {
	ings, err := lbc.ingLister.List()
	if err != nil {
		return &extensions.Ingress{}, err
	}

	for i, _ := range ings.Items {
		if !lbc.isNginxIngress(&ings.Items[i]) {
			continue
		}
		if !lbc.cnf.HasIngress(&ings.Items[i]) {
			continue
		}
		if !isMaster(&ings.Items[i]) {
			continue
		}
		if ings.Items[i].Spec.Rules[0].Host != minion.Spec.Rules[0].Host {
			continue
		}
		return &ings.Items[i], nil
	}

	err = fmt.Errorf("Could not find a Master for Minion: '%v/%v'", minion.Namespace, minion.Name)
	return nil, err
}

func (lbc *LoadBalancerController) createMergableIngresses(master *extensions.Ingress) (*nginx.MergeableIngresses, error) {
	mergeableIngresses := nginx.MergeableIngresses{}

	if len(master.Spec.Rules) != 1 {
		err := fmt.Errorf("Ingress Resource %v/%v with the 'nginx.org/mergeable-ingress-type' annotation must contain only one host", master.Namespace, master.Name)
		return &mergeableIngresses, err
	}

	var empty extensions.HTTPIngressRuleValue
	if master.Spec.Rules[0].HTTP != nil {
		if master.Spec.Rules[0].HTTP != &empty {
			if len(master.Spec.Rules[0].HTTP.Paths) != 0 {
				err := fmt.Errorf("Ingress Resource %v/%v with the 'nginx.org/mergeable-ingress-type' annotation set to 'master' cannot contain Paths", master.Namespace, master.Name)
				return &mergeableIngresses, err
			}
		}
	}

	// Makes sure there is an empty path assigned to a master, to allow for lbc.createIngress() to pass
	master.Spec.Rules[0].HTTP = &extensions.HTTPIngressRuleValue{
		Paths: []extensions.HTTPIngressPath{},
	}

	masterIngEx, err := lbc.createIngress(master)
	if err != nil {
		err := fmt.Errorf("Error creating Ingress Resource %v/%v: %v", master.Namespace, master.Name, err)
		return &mergeableIngresses, err
	}
	mergeableIngresses.Master = masterIngEx

	minions, err := lbc.getMinionsForMaster(masterIngEx)
	if err != nil {
		err = fmt.Errorf("Error Obtaining Ingress Resources: %v", err)
		return &mergeableIngresses, err
	}
	mergeableIngresses.Minions = minions

	return &mergeableIngresses, nil
}

func isMinion(ing *extensions.Ingress) bool {
	if ing.Annotations["nginx.org/mergeable-ingress-type"] == "minion" {
		return true
	}
	return false
}

func isMaster(ing *extensions.Ingress) bool {
	if ing.Annotations["nginx.org/mergeable-ingress-type"] == "master" {
		return true
	}
	return false
}

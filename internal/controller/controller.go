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

	"github.com/golang/glog"

	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	"github.com/nginxinc/kubernetes-ingress/internal/queue"
	"github.com/nginxinc/kubernetes-ingress/internal/utils"
	"k8s.io/api/extensions/v1beta1"
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
	client                  kubernetes.Interface
	ingressController       cache.Controller
	svcController           cache.Controller
	endpointController      cache.Controller
	configMapController     cache.Controller
	secretController        cache.Controller
	ingressLister           utils.StoreToIngressLister
	svcLister               cache.Store
	endpointLister          utils.StoreToEndpointLister
	configMapLister         utils.StoreToConfigMapLister
	secretLister            utils.StoreToSecretLister
	syncQueue               *queue.TaskQueue
	stop                    chan struct{}
	configurator            *nginx.Configurator
	watchNginxConfigMaps    bool
	isNginxPlus             bool
	recorder                record.EventRecorder
	defaultServerSecret     string
	ingressClass            string
	useIngressClassOnly     bool
	statusUpdater           *StatusUpdater
	leaderElector           *leaderelection.LeaderElector
	reportIngressStatus     bool
	isLeaderElectionEnabled bool
	resync                  time.Duration
	namespace               string
	controllerNamespace     string
}

var keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

// NewLoadBalancerControllerInput holds the input needed to call NewLoadBalancerController.
type NewLoadBalancerControllerInput struct {
	KubeClient              kubernetes.Interface
	ResyncPeriod            time.Duration
	Namespace               string
	NginxConfigurator       *nginx.Configurator
	DefaultServerSecret     string
	IsNginxPlus             bool
	IngressClass            string
	UseIngressClassOnly     bool
	ExternalServiceName     string
	ControllerNamespace     string
	ReportIngressStatus     bool
	IsLeaderElectionEnabled bool
}

// NewLoadBalancerController creates a controller
func NewLoadBalancerController(input NewLoadBalancerControllerInput) *LoadBalancerController {
	lbc := LoadBalancerController{
		client:                  input.KubeClient,
		stop:                    make(chan struct{}),
		configurator:            input.NginxConfigurator,
		defaultServerSecret:     input.DefaultServerSecret,
		isNginxPlus:             input.IsNginxPlus,
		ingressClass:            input.IngressClass,
		useIngressClassOnly:     input.UseIngressClassOnly,
		reportIngressStatus:     input.ReportIngressStatus,
		isLeaderElectionEnabled: input.IsLeaderElectionEnabled,
		resync:                  input.ResyncPeriod,
		namespace:               input.Namespace,
		controllerNamespace:     input.ControllerNamespace,
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&core_v1.EventSinkImpl{
		Interface: core_v1.New(input.KubeClient.Core().RESTClient()).Events(""),
	})
	lbc.recorder = eventBroadcaster.NewRecorder(scheme.Scheme,
		api_v1.EventSource{Component: "nginx-ingress-controller"})

	lbc.syncQueue = queue.NewTaskQueue(lbc.sync)

	glog.V(3).Infof("Nginx Ingress Controller has class: %v", input.IngressClass)

	lbc.statusUpdater = &StatusUpdater{
		client:              input.KubeClient,
		namespace:           input.ControllerNamespace,
		externalServiceName: input.ExternalServiceName,
		ingLister:           &lbc.ingressLister,
		keyFunc:             keyFunc,
	}

	return &lbc
}

// UpdateManagedAndMergeableIngresses invokes the UpdateManagedAndMergeableIngresses method on the Status Updater
func (lbc *LoadBalancerController) UpdateManagedAndMergeableIngresses(ingresses []v1beta1.Ingress, mergeableIngresses map[string]*nginx.MergeableIngresses) error {
	return lbc.statusUpdater.UpdateManagedAndMergeableIngresses(ingresses, mergeableIngresses)
}

// AddLeaderHandler adds the handler for leader election to the controller
func (lbc *LoadBalancerController) AddLeaderHandler(leaderHandler leaderelection.LeaderCallbacks) {
	var err error
	lbc.leaderElector, err = NewLeaderElector(lbc.client, leaderHandler, lbc.controllerNamespace)
	if err != nil {
		glog.V(3).Infof("Error starting LeaderElection: %v", err)
	}
}

// GetIngressClassKey returns the ingress class key
func (lbc *LoadBalancerController) GetIngressClassKey() string {
	return ingressClassKey
}

// AddSyncQueue enqueues the provided item on the sync queue
func (lbc *LoadBalancerController) AddSyncQueue(item interface{}) {
	lbc.syncQueue.Enqueue(item)
}

// WatchNginxConfigMaps sets the controller to watch config map changes
func (lbc *LoadBalancerController) WatchNginxConfigMaps() {
	lbc.watchNginxConfigMaps = true
}

// AddSecretHandler adds the handler for secrets to the controller
func (lbc *LoadBalancerController) AddSecretHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.secretLister.Store, lbc.secretController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.Core().RESTClient(),
			"secrets",
			lbc.namespace,
			fields.Everything()),
		&api_v1.Secret{},
		lbc.resync,
		handlers,
	)
}

// AddServiceHandler adds the handler for services to the controller
func (lbc *LoadBalancerController) AddServiceHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.svcLister, lbc.svcController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.Core().RESTClient(),
			"services",
			lbc.namespace,
			fields.Everything()),
		&api_v1.Service{},
		lbc.resync,
		handlers,
	)
}

// AddIngressHandler adds the handler for ingresses to the controller
func (lbc *LoadBalancerController) AddIngressHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.ingressLister.Store, lbc.ingressController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.Extensions().RESTClient(),
			"ingresses",
			lbc.namespace,
			fields.Everything()),
		&extensions.Ingress{},
		lbc.resync,
		handlers,
	)
}

// AddEndpointHandler adds the handler for endpoints to the controller
func (lbc *LoadBalancerController) AddEndpointHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.endpointLister.Store, lbc.endpointController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.Core().RESTClient(),
			"endpoints",
			lbc.namespace,
			fields.Everything()),
		&api_v1.Endpoints{},
		lbc.resync,
		handlers,
	)
}

// AddConfigMapHandler adds the handler for config maps to the controller
func (lbc *LoadBalancerController) AddConfigMapHandler(handlers cache.ResourceEventHandlerFuncs, namespace string) {
	lbc.configMapLister.Store, lbc.configMapController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.Core().RESTClient(),
			"configmaps",
			namespace,
			fields.Everything()),
		&api_v1.ConfigMap{},
		lbc.resync,
		handlers,
	)
}

// GetDefaultServerSecret returns the default server secret
func (lbc *LoadBalancerController) GetDefaultServerSecret() string {
	return lbc.defaultServerSecret
}

// Run starts the loadbalancer controller
func (lbc *LoadBalancerController) Run() {
	if lbc.leaderElector != nil {
		go lbc.leaderElector.Run()
	}
	go lbc.svcController.Run(lbc.stop)
	go lbc.endpointController.Run(lbc.stop)
	go lbc.secretController.Run(lbc.stop)
	if lbc.watchNginxConfigMaps {
		go lbc.configMapController.Run(lbc.stop)
	}
	go lbc.ingressController.Run(lbc.stop)
	go lbc.syncQueue.Run(time.Second, lbc.stop)
	<-lbc.stop
}

// Stop shutdowns the load balancer controller
func (lbc *LoadBalancerController) Stop() {
	close(lbc.stop)

	lbc.syncQueue.Shutdown()
}

func (lbc *LoadBalancerController) syncEndpoint(task queue.Task) {
	key := task.Key
	glog.V(3).Infof("Syncing endpoints %v", key)

	obj, endpExists, err := lbc.endpointLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if endpExists {
		ings := lbc.getIngressForEndpoints(obj)

		var ingExes []*nginx.IngressEx
		var mergableIngressesSlice []*nginx.MergeableIngresses

		for i := range ings {
			if !lbc.IsNginxIngress(&ings[i]) {
				continue
			}
			if utils.IsMinion(&ings[i]) {
				master, err := lbc.FindMasterForMinion(&ings[i])
				if err != nil {
					glog.Errorf("Ignoring Ingress %v(Minion): %v", ings[i].Name, err)
					continue
				}
				if !lbc.configurator.HasMinion(master, &ings[i]) {
					continue
				}
				mergeableIngresses, err := lbc.createMergableIngresses(master)
				if err != nil {
					glog.Errorf("Ignoring Ingress %v(Minion): %v", ings[i].Name, err)
					continue
				}

				mergableIngressesSlice = append(mergableIngressesSlice, mergeableIngresses)
				continue
			}
			if !lbc.configurator.HasIngress(&ings[i]) {
				continue
			}
			ingEx, err := lbc.createIngress(&ings[i])
			if err != nil {
				glog.Errorf("Error updating endpoints for %v/%v: %v, skipping", &ings[i].Namespace, &ings[i].Name, err)
				continue
			}
			ingExes = append(ingExes, ingEx)
		}

		if len(ingExes) > 0 {
			glog.V(3).Infof("Updating Endpoints for %v", ingExes)
			err = lbc.configurator.UpdateEndpoints(ingExes)
			if err != nil {
				glog.Errorf("Error updating endpoints for %v: %v", ingExes, err)
			}
		}

		if len(mergableIngressesSlice) > 0 {
			glog.V(3).Infof("Updating Endpoints for %v", mergableIngressesSlice)
			err = lbc.configurator.UpdateEndpointsMergeableIngress(mergableIngressesSlice)
			if err != nil {
				glog.Errorf("Error updating endpoints for %v: %v", mergableIngressesSlice, err)
			}
		}
	}
}

func (lbc *LoadBalancerController) syncConfig(task queue.Task) {
	key := task.Key
	glog.V(3).Infof("Syncing configmap %v", key)

	obj, configExists, err := lbc.configMapLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}
	cfg := nginx.NewDefaultConfig()

	if configExists {
		cfgm := obj.(*api_v1.ConfigMap)
		cfg = nginx.ParseConfigMap(cfgm, lbc.isNginxPlus)

		lbc.statusUpdater.SaveStatusFromExternalStatus(cfgm.Data["external-status-address"])
	}

	ingresses, mergeableIngresses := lbc.GetManagedIngresses()
	ingExes := lbc.ingressesToIngressExes(ingresses)

	if lbc.reportStatusEnabled() {
		err = lbc.statusUpdater.UpdateManagedAndMergeableIngresses(ingresses, mergeableIngresses)
		if err != nil {
			glog.V(3).Infof("error updating status on ConfigMap change: %v", err)
		}
	}

	updateErr := lbc.configurator.UpdateConfig(cfg, ingExes, mergeableIngresses)

	eventTitle := "Updated"
	eventType := api_v1.EventTypeNormal
	eventWarningMessage := ""

	if updateErr != nil {
		eventTitle = "UpdatedWithError"
		eventType = api_v1.EventTypeWarning
		eventWarningMessage = fmt.Sprintf("but was not applied: %v", updateErr)
	}
	if configExists {
		cfgm := obj.(*api_v1.ConfigMap)
		lbc.recorder.Eventf(cfgm, eventType, eventTitle, "Configuration from %v was updated %s", key, eventWarningMessage)
	}
	for _, ingEx := range ingExes {
		lbc.recorder.Eventf(ingEx.Ingress, eventType, eventTitle, "Configuration for %v/%v was updated %s",
			ingEx.Ingress.Namespace, ingEx.Ingress.Name, eventWarningMessage)
	}
	for _, mergeableIng := range mergeableIngresses {
		master := mergeableIng.Master
		lbc.recorder.Eventf(master.Ingress, eventType, eventTitle, "Configuration for %v/%v(Master) was updated %s", master.Ingress.Namespace, master.Ingress.Name, eventWarningMessage)
		for _, minion := range mergeableIng.Minions {
			lbc.recorder.Eventf(minion.Ingress, eventType, eventTitle, "Configuration for %v/%v(Minion) was updated %s",
				minion.Ingress.Namespace, minion.Ingress.Name, eventWarningMessage)
		}
	}
}

// GetManagedIngresses gets Ingress resources that the IC is currently responsible for
func (lbc *LoadBalancerController) GetManagedIngresses() ([]extensions.Ingress, map[string]*nginx.MergeableIngresses) {
	mergeableIngresses := make(map[string]*nginx.MergeableIngresses)
	var managedIngresses []extensions.Ingress
	ings, _ := lbc.ingressLister.List()
	for i := range ings.Items {
		ing := ings.Items[i]
		if !lbc.IsNginxIngress(&ing) {
			continue
		}
		if utils.IsMinion(&ing) {
			master, err := lbc.FindMasterForMinion(&ing)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", ing, err)
				continue
			}
			if !lbc.configurator.HasIngress(master) {
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
		if !lbc.configurator.HasIngress(&ing) {
			continue
		}
		managedIngresses = append(managedIngresses, ing)
	}
	return managedIngresses, mergeableIngresses
}

func (lbc *LoadBalancerController) ingressesToIngressExes(ings []extensions.Ingress) []*nginx.IngressEx {
	var ingExes []*nginx.IngressEx
	for i := range ings {
		ingEx, err := lbc.createIngress(&ings[i])
		if err != nil {
			continue
		}
		ingExes = append(ingExes, ingEx)
	}
	return ingExes
}

func (lbc *LoadBalancerController) sync(task queue.Task) {
	glog.V(3).Infof("Syncing %v", task.Key)

	switch task.Kind {
	case queue.Ingress:
		lbc.syncIng(task)
	case queue.IngressMinion:
		lbc.syncIngMinion(task)
	case queue.ConfigMap:
		lbc.syncConfig(task)
		return
	case queue.Endpoints:
		lbc.syncEndpoint(task)
		return
	case queue.Secret:
		lbc.syncSecret(task)
		return
	case queue.Service:
		lbc.syncExternalService(task)
	}
}

func (lbc *LoadBalancerController) syncIngMinion(task queue.Task) {
	key := task.Key
	obj, ingExists, err := lbc.ingressLister.Store.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if !ingExists {
		glog.V(2).Infof("Minion was deleted: %v\n", key)
		return
	}
	glog.V(2).Infof("Adding or Updating Minion: %v\n", key)

	minion := obj.(*extensions.Ingress)

	master, err := lbc.FindMasterForMinion(minion)
	if err != nil {
		lbc.syncQueue.RequeueAfter(task, err, 5*time.Second)
		return
	}

	_, err = lbc.createIngress(minion)
	if err != nil {
		lbc.syncQueue.RequeueAfter(task, err, 5*time.Second)
		if !lbc.configurator.HasMinion(master, minion) {
			return
		}
	}

	lbc.syncQueue.Enqueue(master)
}

func (lbc *LoadBalancerController) syncIng(task queue.Task) {
	key := task.Key
	ing, ingExists, err := lbc.ingressLister.GetByKeySafe(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if !ingExists {
		glog.V(2).Infof("Deleting Ingress: %v\n", key)

		err := lbc.configurator.DeleteIngress(key)
		if err != nil {
			glog.Errorf("Error when deleting configuration for %v: %v", key, err)
		}
	} else {
		glog.V(2).Infof("Adding or Updating Ingress: %v\n", key)

		if utils.IsMaster(ing) {
			mergeableIngExs, err := lbc.createMergableIngresses(ing)
			if err != nil {
				// we need to requeue because an error can occur even if the master is valid
				// otherwise, we will not be able to generate the config until there is change
				// in the master or minions.
				lbc.syncQueue.RequeueAfter(task, err, 5*time.Second)
				lbc.recorder.Eventf(ing, api_v1.EventTypeWarning, "Rejected", "%v was rejected: %v", key, err)
				if lbc.reportStatusEnabled() {
					err = lbc.statusUpdater.ClearIngressStatus(*ing)
					if err != nil {
						glog.V(3).Infof("error clearing ing status: %v", err)
					}
				}
				return
			}
			addErr := lbc.configurator.AddOrUpdateMergeableIngress(mergeableIngExs)

			// record correct eventType and message depending on the error
			eventTitle := "AddedOrUpdated"
			eventType := api_v1.EventTypeNormal
			eventWarningMessage := ""

			if addErr != nil {
				eventTitle = "AddedOrUpdatedWithError"
				eventType = api_v1.EventTypeWarning
				eventWarningMessage = fmt.Sprintf("but was not applied: %v", addErr)
			}
			lbc.recorder.Eventf(ing, eventType, eventTitle, "Configuration for %v(Master) was added or updated %s", key, eventWarningMessage)
			for _, minion := range mergeableIngExs.Minions {
				lbc.recorder.Eventf(ing, eventType, eventTitle, "Configuration for %v/%v(Minion) was added or updated %s", minion.Ingress.Namespace, minion.Ingress.Name, eventWarningMessage)
			}

			if lbc.reportStatusEnabled() {
				err = lbc.statusUpdater.UpdateMergableIngresses(mergeableIngExs)
				if err != nil {
					glog.V(3).Infof("error updating ingress status: %v", err)
				}
			}
			return
		}
		ingEx, err := lbc.createIngress(ing)
		if err != nil {
			lbc.recorder.Eventf(ing, api_v1.EventTypeWarning, "Rejected", "%v was rejected: %v", key, err)
			if lbc.reportStatusEnabled() {
				err = lbc.statusUpdater.ClearIngressStatus(*ing)
				if err != nil {
					glog.V(3).Infof("error clearing ing status: %v", err)
				}
			}
			return
		}

		err = lbc.configurator.AddOrUpdateIngress(ingEx)
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
func (lbc *LoadBalancerController) syncExternalService(task queue.Task) {
	key := task.Key
	obj, exists, err := lbc.svcLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}
	statusIngs, mergableIngs := lbc.GetManagedIngresses()
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

// IsExternalServiceForStatus matches the service specified by the external-service arg
func (lbc *LoadBalancerController) IsExternalServiceForStatus(svc *api_v1.Service) bool {
	return lbc.statusUpdater.namespace == svc.Namespace && lbc.statusUpdater.externalServiceName == svc.Name
}

// reportStatusEnabled determines if we should attempt to report status
func (lbc *LoadBalancerController) reportStatusEnabled() bool {
	if lbc.reportIngressStatus {
		if lbc.isLeaderElectionEnabled {
			return lbc.leaderElector != nil && lbc.leaderElector.IsLeader()
		}
		return true
	}
	return false
}

func (lbc *LoadBalancerController) syncSecret(task queue.Task) {
	key := task.Key
	obj, secrExists, err := lbc.secretLister.Store.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	namespace, name, err := utils.ParseNamespaceName(key)
	if err != nil {
		glog.Warningf("Secret key %v is invalid: %v", key, err)
		return
	}

	ings, err := lbc.findIngressesForSecret(namespace, name)
	if err != nil {
		glog.Warningf("Failed to find Ingress resources for Secret %v: %v", key, err)
		lbc.syncQueue.RequeueAfter(task, err, 5*time.Second)
	}

	glog.V(2).Infof("Found %v Ingresses with Secret %v", len(ings), key)

	if !secrExists {
		glog.V(2).Infof("Deleting Secret: %v\n", key)

		lbc.handleSecretDeletion(key, ings)

		if key == lbc.defaultServerSecret {
			glog.Warningf("The default server Secret %v was removed. Retaining the Secret.", key)
		}
		return
	}

	glog.V(2).Infof("Adding / Updating Secret: %v\n", key)

	secret := obj.(*api_v1.Secret)

	if key == lbc.defaultServerSecret {
		lbc.handleDefaultSecretUpdate(secret)
		// we don't return here in case the default secret is also used in Ingress resources
	}

	if len(ings) > 0 {
		lbc.handleSecretUpdate(secret, ings)
	}
}

func (lbc *LoadBalancerController) handleSecretDeletion(key string, ings []extensions.Ingress) {
	eventType := api_v1.EventTypeWarning
	title := "Missing Secret"
	message := fmt.Sprintf("Secret %v was removed", key)

	lbc.emitEventForIngresses(eventType, title, message, ings)

	regular, mergeable := lbc.createIngresses(ings)

	eventType = api_v1.EventTypeNormal
	title = "Updated"
	message = fmt.Sprintf("Configuration was updated due to removed secret %v", key)

	if err := lbc.configurator.DeleteSecret(key, regular, mergeable); err != nil {
		glog.Errorf("Error when deleting Secret: %v: %v", key, err)

		eventType = api_v1.EventTypeWarning
		title = "UpdatedWithError"
		message = fmt.Sprintf("Configuration was updated due to removed secret %v, but not applied: %v", key, err)
	}

	lbc.emitEventForIngresses(eventType, title, message, ings)
}

func (lbc *LoadBalancerController) handleSecretUpdate(secret *api_v1.Secret, ings []extensions.Ingress) {
	secretNsName := secret.Namespace + "/" + secret.Name

	err := lbc.ValidateSecret(secret)
	if err != nil {
		// Secret becomes Invalid
		glog.Errorf("Couldn't validate secret %v: %v", secretNsName, err)
		glog.Errorf("Removing invalid secret %v", secretNsName)

		lbc.handleSecretDeletion(secretNsName, ings)

		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "Rejected", "%v was rejected: %v", secretNsName, err)
		return
	}

	eventType := api_v1.EventTypeNormal
	title := "Updated"
	message := fmt.Sprintf("Configuration was updated due to updated secret %v", secretNsName)

	regular, mergeable := lbc.createIngresses(ings)

	if err := lbc.configurator.AddOrUpdateSecret(secret, regular, mergeable); err != nil {
		glog.Errorf("Error when updating Secret %v: %v", secretNsName, err)
		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "%v was updated, but not applied: %v", secretNsName, err)

		eventType = api_v1.EventTypeWarning
		title = "UpdatedWithError"
		message = fmt.Sprintf("Configuration was updated due to updated secret %v, but not applied: %v", secretNsName, err)
	}

	lbc.emitEventForIngresses(eventType, title, message, ings)
}

func (lbc *LoadBalancerController) handleDefaultSecretUpdate(secret *api_v1.Secret) {
	secretNsName := secret.Namespace + "/" + secret.Name

	err := nginx.ValidateTLSSecret(secret)
	if err != nil {
		glog.Errorf("Couldn't validate the default server Secret %v: %v", secretNsName, err)
		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "Rejected", "the default server Secret %v was rejected, using the previous version: %v", secretNsName, err)
		return
	}

	err = lbc.configurator.AddOrUpdateDefaultServerTLSSecret(secret)
	if err != nil {
		glog.Errorf("Error when updating the default server Secret %v: %v", secretNsName, err)
		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "the default server Secret %v was updated, but not applied: %v", secretNsName, err)
		return
	}

	lbc.recorder.Eventf(secret, api_v1.EventTypeNormal, "Updated", "the default server Secret %v was updated", secretNsName)
}

func (lbc *LoadBalancerController) emitEventForIngresses(eventType string, title string, message string, ings []extensions.Ingress) {
	for _, ing := range ings {
		lbc.recorder.Eventf(&ing, eventType, title, message)
		if utils.IsMinion(&ing) {
			master, err := lbc.FindMasterForMinion(&ing)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", ing.Name, err)
				continue
			}
			masterMsg := fmt.Sprintf("%v for Minion %v/%v", message, ing.Namespace, ing.Name)
			lbc.recorder.Eventf(master, eventType, title, masterMsg)
		}
	}
}

func (lbc *LoadBalancerController) createIngresses(ings []extensions.Ingress) (regular []nginx.IngressEx, mergeable []nginx.MergeableIngresses) {
	for i := range ings {
		if utils.IsMaster(&ings[i]) {
			mergeableIng, err := lbc.createMergableIngresses(&ings[i])
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Master): %v", ings[i].Name, err)
				continue
			}
			mergeable = append(mergeable, *mergeableIng)
			continue
		}

		if utils.IsMinion(&ings[i]) {
			master, err := lbc.FindMasterForMinion(&ings[i])
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", ings[i].Name, err)
				continue
			}
			mergeableIng, err := lbc.createMergableIngresses(master)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Master): %v", master, err)
				continue
			}

			mergeable = append(mergeable, *mergeableIng)
			continue
		}

		ingEx, err := lbc.createIngress(&ings[i])
		if err != nil {
			glog.Errorf("Ignoring Ingress %v/%v: $%v", ings[i].Namespace, ings[i].Name, err)
		}
		regular = append(regular, *ingEx)
	}

	return regular, mergeable
}

func (lbc *LoadBalancerController) findIngressesForSecret(secretNamespace string, secretName string) (ings []extensions.Ingress, error error) {
	allIngs, err := lbc.ingressLister.List()
	if err != nil {
		return nil, fmt.Errorf("Couldn't get the list of Ingress resources: %v", err)
	}

items:
	for _, ing := range allIngs.Items {
		if ing.Namespace != secretNamespace {
			continue
		}

		if !lbc.IsNginxIngress(&ing) {
			continue
		}

		if !utils.IsMinion(&ing) {
			if !lbc.configurator.HasIngress(&ing) {
				continue
			}
			for _, tls := range ing.Spec.TLS {
				if tls.SecretName == secretName {
					ings = append(ings, ing)
					continue items
				}
			}
			if lbc.isNginxPlus {
				if jwtKey, exists := ing.Annotations[nginx.JWTKeyAnnotation]; exists {
					if jwtKey == secretName {
						ings = append(ings, ing)
					}
				}
			}
			continue
		}

		// we're dealing with a minion
		// minions can only have JWT secrets
		if lbc.isNginxPlus {
			master, err := lbc.FindMasterForMinion(&ing)
			if err != nil {
				glog.Infof("Ignoring Ingress %v(Minion): %v", ing.Name, err)
				continue
			}

			if !lbc.configurator.HasMinion(master, &ing) {
				continue
			}

			if jwtKey, exists := ing.Annotations[nginx.JWTKeyAnnotation]; exists {
				if jwtKey == secretName {
					ings = append(ings, ing)
				}
			}
		}
	}

	return ings, nil
}

// EnqueueIngressForService enqueues the ingress for the given service
func (lbc *LoadBalancerController) EnqueueIngressForService(svc *api_v1.Service) {
	ings := lbc.getIngressesForService(svc)
	for _, ing := range ings {
		if !lbc.IsNginxIngress(&ing) {
			continue
		}
		if utils.IsMinion(&ing) {
			master, err := lbc.FindMasterForMinion(&ing)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", ing.Name, err)
				continue
			}
			ing = *master
		}
		if !lbc.configurator.HasIngress(&ing) {
			continue
		}
		lbc.syncQueue.Enqueue(&ing)

	}
}

func (lbc *LoadBalancerController) getIngressesForService(svc *api_v1.Service) []extensions.Ingress {
	ings, err := lbc.ingressLister.GetServiceIngress(svc)
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
		secretKey := ing.Namespace + "/" + secretName

		secretObject, secretExists, err := lbc.secretLister.GetByKey(secretKey)
		if err != nil {
			glog.Warningf("Error retrieving secret %v for Ingress %v: %v", secretName, ing.Name, err)
			continue
		}
		if !secretExists {
			glog.Warningf("secret %v not found for Ingress %v", secretKey, ing.Name)
			continue
		}
		secret := secretObject.(*api_v1.Secret)

		err = nginx.ValidateTLSSecret(secret)
		if err != nil {
			glog.Warningf("Error validating secret %v for Ingress %v: %v", secretName, ing.Name, err)
			continue
		}
		ingEx.TLSSecrets[secretName] = secret
	}

	if lbc.isNginxPlus {
		if jwtKey, exists := ingEx.Ingress.Annotations[nginx.JWTKeyAnnotation]; exists {
			secretName := jwtKey

			secret, err := lbc.client.Core().Secrets(ing.Namespace).Get(secretName, meta_v1.GetOptions{})
			if err != nil {
				glog.Warningf("Error retrieving secret %v for Ingress %v: %v", secretName, ing.Name, err)
				secret = nil
			} else {
				err = nginx.ValidateJWKSecret(secret)
				if err != nil {
					glog.Warningf("Error validating secret %v for Ingress %v: %v", secretName, ing.Name, err)
					secret = nil
				}
			}

			ingEx.JWTKey = nginx.JWTKey{
				Name:   jwtKey,
				Secret: secret,
			}
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
		if lbc.isNginxPlus && lbc.isHealthCheckEnabled(ing) {
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
			if lbc.isNginxPlus && lbc.isHealthCheckEnabled(ing) {
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

	endps, err := lbc.endpointLister.GetServiceEndpoints(svc)
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

	portNum, err := utils.FindPort(pod, svcPort)
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

	return nil, fmt.Errorf("service %s doesn't exist", svcKey)
}

// IsNginxIngress checks if resource ingress class annotation (if exists) is matching with ingress controller class
// If annotation is absent and use-ingress-class-only enabled - ingress resource would ignore
func (lbc *LoadBalancerController) IsNginxIngress(ing *extensions.Ingress) bool {
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
	if !lbc.isNginxPlus {
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
	ings, err := lbc.ingressLister.List()
	if err != nil {
		return []*nginx.IngressEx{}, err
	}

	// ingresses are sorted by creation time
	sort.Slice(ings.Items[:], func(i, j int) bool {
		return ings.Items[i].CreationTimestamp.Time.UnixNano() < ings.Items[j].CreationTimestamp.Time.UnixNano()
	})

	var minions []*nginx.IngressEx
	var minionPaths = make(map[string]*extensions.Ingress)

	for i := range ings.Items {
		if !lbc.IsNginxIngress(&ings.Items[i]) {
			continue
		}
		if !utils.IsMinion(&ings.Items[i]) {
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

// FindMasterForMinion returns a master for a given minion
func (lbc *LoadBalancerController) FindMasterForMinion(minion *extensions.Ingress) (*extensions.Ingress, error) {
	ings, err := lbc.ingressLister.List()
	if err != nil {
		return &extensions.Ingress{}, err
	}

	for i := range ings.Items {
		if !lbc.IsNginxIngress(&ings.Items[i]) {
			continue
		}
		if !lbc.configurator.HasIngress(&ings.Items[i]) {
			continue
		}
		if !utils.IsMaster(&ings.Items[i]) {
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

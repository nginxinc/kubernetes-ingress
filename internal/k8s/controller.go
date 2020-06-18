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

package k8s

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/spiffe/go-spiffe/workload"

	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"

	"sort"

	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/validation"
	k8s_nginx "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
)

const (
	ingressClassKey = "kubernetes.io/ingress.class"
)

// LoadBalancerController watches Kubernetes API and
// reconfigures NGINX via NginxController when needed
type LoadBalancerController struct {
	client                        kubernetes.Interface
	confClient                    k8s_nginx.Interface
	ingressController             cache.Controller
	svcController                 cache.Controller
	endpointController            cache.Controller
	configMapController           cache.Controller
	secretController              cache.Controller
	virtualServerController       cache.Controller
	virtualServerRouteController  cache.Controller
	globalConfigurationController cache.Controller
	transportServerController     cache.Controller
	podController                 cache.Controller
	ingressLister                 storeToIngressLister
	svcLister                     cache.Store
	endpointLister                storeToEndpointLister
	configMapLister               storeToConfigMapLister
	podLister                     indexerToPodLister
	secretLister                  storeToSecretLister
	virtualServerLister           cache.Store
	virtualServerRouteLister      cache.Store
	globalConfiguratonLister      cache.Store
	transportServerLister         cache.Store
	syncQueue                     *taskQueue
	ctx                           context.Context
	cancel                        context.CancelFunc
	configurator                  *configs.Configurator
	watchNginxConfigMaps          bool
	watchGlobalConfiguration      bool
	isNginxPlus                   bool
	recorder                      record.EventRecorder
	defaultServerSecret           string
	ingressClass                  string
	useIngressClassOnly           bool
	statusUpdater                 *statusUpdater
	leaderElector                 *leaderelection.LeaderElector
	reportIngressStatus           bool
	isLeaderElectionEnabled       bool
	leaderElectionLockName        string
	resync                        time.Duration
	namespace                     string
	controllerNamespace           string
	wildcardTLSSecret             string
	areCustomResourcesEnabled     bool
	metricsCollector              collectors.ControllerCollector
	globalConfigurationValidator  *validation.GlobalConfigurationValidator
	transportServerValidator      *validation.TransportServerValidator
	spiffeController              *spiffeController
	syncLock                      sync.Mutex
}

var keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

// NewLoadBalancerControllerInput holds the input needed to call NewLoadBalancerController.
type NewLoadBalancerControllerInput struct {
	KubeClient                   kubernetes.Interface
	ConfClient                   k8s_nginx.Interface
	ResyncPeriod                 time.Duration
	Namespace                    string
	NginxConfigurator            *configs.Configurator
	DefaultServerSecret          string
	IsNginxPlus                  bool
	IngressClass                 string
	UseIngressClassOnly          bool
	ExternalServiceName          string
	ControllerNamespace          string
	ReportIngressStatus          bool
	IsLeaderElectionEnabled      bool
	LeaderElectionLockName       string
	WildcardTLSSecret            string
	ConfigMaps                   string
	GlobalConfiguration          string
	AreCustomResourcesEnabled    bool
	MetricsCollector             collectors.ControllerCollector
	GlobalConfigurationValidator *validation.GlobalConfigurationValidator
	TransportServerValidator     *validation.TransportServerValidator
	SpireAgentAddress            string
}

// NewLoadBalancerController creates a controller
func NewLoadBalancerController(input NewLoadBalancerControllerInput) *LoadBalancerController {
	lbc := &LoadBalancerController{
		client:                       input.KubeClient,
		confClient:                   input.ConfClient,
		configurator:                 input.NginxConfigurator,
		defaultServerSecret:          input.DefaultServerSecret,
		isNginxPlus:                  input.IsNginxPlus,
		ingressClass:                 input.IngressClass,
		useIngressClassOnly:          input.UseIngressClassOnly,
		reportIngressStatus:          input.ReportIngressStatus,
		isLeaderElectionEnabled:      input.IsLeaderElectionEnabled,
		leaderElectionLockName:       input.LeaderElectionLockName,
		resync:                       input.ResyncPeriod,
		namespace:                    input.Namespace,
		controllerNamespace:          input.ControllerNamespace,
		wildcardTLSSecret:            input.WildcardTLSSecret,
		areCustomResourcesEnabled:    input.AreCustomResourcesEnabled,
		metricsCollector:             input.MetricsCollector,
		globalConfigurationValidator: input.GlobalConfigurationValidator,
		transportServerValidator:     input.TransportServerValidator,
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&core_v1.EventSinkImpl{
		Interface: core_v1.New(input.KubeClient.CoreV1().RESTClient()).Events(""),
	})
	lbc.recorder = eventBroadcaster.NewRecorder(scheme.Scheme,
		api_v1.EventSource{Component: "nginx-ingress-controller"})

	lbc.syncQueue = newTaskQueue(lbc.sync)
	if input.SpireAgentAddress != "" {
		var err error
		lbc.spiffeController, err = NewSpiffeController(lbc.syncSVIDRotation, input.SpireAgentAddress)
		if err != nil {
			glog.Fatalf("failed to create Spiffe Controller: %v", err)
		}
	}

	glog.V(3).Infof("Nginx Ingress Controller has class: %v", input.IngressClass)
	lbc.statusUpdater = &statusUpdater{
		client:              input.KubeClient,
		namespace:           input.ControllerNamespace,
		externalServiceName: input.ExternalServiceName,
		ingLister:           &lbc.ingressLister,
		keyFunc:             keyFunc,
		confClient:          input.ConfClient,
	}

	// create handlers for resources we care about
	lbc.addSecretHandler(createSecretHandlers(lbc))
	lbc.addIngressHandler(createIngressHandlers(lbc))
	lbc.addServiceHandler(createServiceHandlers(lbc))
	lbc.addEndpointHandler(createEndpointHandlers(lbc))
	lbc.addPodHandler()

	if lbc.areCustomResourcesEnabled {
		lbc.addVirtualServerHandler(createVirtualServerHandlers(lbc))
		lbc.addVirtualServerRouteHandler(createVirtualServerRouteHandlers(lbc))
		lbc.addTransportServerHandler(createTransportServerHandlers(lbc))

		if input.GlobalConfiguration != "" {
			lbc.watchGlobalConfiguration = true

			ns, name, _ := ParseNamespaceName(input.GlobalConfiguration)

			lbc.addGlobalConfigurationHandler(createGlobalConfigurationHandlers(lbc), ns, name)
		}
	}

	if input.ConfigMaps != "" {
		nginxConfigMapsNS, nginxConfigMapsName, err := ParseNamespaceName(input.ConfigMaps)
		if err != nil {
			glog.Warning(err)
		} else {
			lbc.watchNginxConfigMaps = true
			lbc.addConfigMapHandler(createConfigMapHandlers(lbc, nginxConfigMapsName), nginxConfigMapsNS)
		}
	}

	if input.ReportIngressStatus && input.IsLeaderElectionEnabled {
		lbc.addLeaderHandler(createLeaderHandler(lbc))
	}

	lbc.updateIngressMetrics()

	return lbc
}

// UpdateManagedAndMergeableIngresses invokes the UpdateManagedAndMergeableIngresses method on the Status Updater
func (lbc *LoadBalancerController) UpdateManagedAndMergeableIngresses(ingresses []v1beta1.Ingress, mergeableIngresses map[string]*configs.MergeableIngresses) error {
	return lbc.statusUpdater.UpdateManagedAndMergeableIngresses(ingresses, mergeableIngresses)
}

// addLeaderHandler adds the handler for leader election to the controller
func (lbc *LoadBalancerController) addLeaderHandler(leaderHandler leaderelection.LeaderCallbacks) {
	var err error
	lbc.leaderElector, err = newLeaderElector(lbc.client, leaderHandler, lbc.controllerNamespace, lbc.leaderElectionLockName)
	if err != nil {
		glog.V(3).Infof("Error starting LeaderElection: %v", err)
	}
}

// AddSyncQueue enqueues the provided item on the sync queue
func (lbc *LoadBalancerController) AddSyncQueue(item interface{}) {
	lbc.syncQueue.Enqueue(item)
}

// addSecretHandler adds the handler for secrets to the controller
func (lbc *LoadBalancerController) addSecretHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.secretLister.Store, lbc.secretController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.CoreV1().RESTClient(),
			"secrets",
			lbc.namespace,
			fields.Everything()),
		&api_v1.Secret{},
		lbc.resync,
		handlers,
	)
}

// addServiceHandler adds the handler for services to the controller
func (lbc *LoadBalancerController) addServiceHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.svcLister, lbc.svcController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.CoreV1().RESTClient(),
			"services",
			lbc.namespace,
			fields.Everything()),
		&api_v1.Service{},
		lbc.resync,
		handlers,
	)
}

// addIngressHandler adds the handler for ingresses to the controller
func (lbc *LoadBalancerController) addIngressHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.ingressLister.Store, lbc.ingressController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.ExtensionsV1beta1().RESTClient(),
			"ingresses",
			lbc.namespace,
			fields.Everything()),
		&extensions.Ingress{},
		lbc.resync,
		handlers,
	)
}

// addEndpointHandler adds the handler for endpoints to the controller
func (lbc *LoadBalancerController) addEndpointHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.endpointLister.Store, lbc.endpointController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.CoreV1().RESTClient(),
			"endpoints",
			lbc.namespace,
			fields.Everything()),
		&api_v1.Endpoints{},
		lbc.resync,
		handlers,
	)
}

// addConfigMapHandler adds the handler for config maps to the controller
func (lbc *LoadBalancerController) addConfigMapHandler(handlers cache.ResourceEventHandlerFuncs, namespace string) {
	lbc.configMapLister.Store, lbc.configMapController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.CoreV1().RESTClient(),
			"configmaps",
			namespace,
			fields.Everything()),
		&api_v1.ConfigMap{},
		lbc.resync,
		handlers,
	)
}

func (lbc *LoadBalancerController) addPodHandler() {
	lbc.podLister.Indexer, lbc.podController = cache.NewIndexerInformer(
		cache.NewListWatchFromClient(
			lbc.client.CoreV1().RESTClient(),
			"pods",
			lbc.namespace,
			fields.Everything()),
		&api_v1.Pod{},
		lbc.resync,
		cache.ResourceEventHandlerFuncs{},
		cache.Indexers{},
	)
}

func (lbc *LoadBalancerController) addVirtualServerHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.virtualServerLister, lbc.virtualServerController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.confClient.K8sV1().RESTClient(),
			"virtualservers",
			lbc.namespace,
			fields.Everything()),
		&conf_v1.VirtualServer{},
		lbc.resync,
		handlers,
	)
}

func (lbc *LoadBalancerController) addVirtualServerRouteHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.virtualServerRouteLister, lbc.virtualServerRouteController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.confClient.K8sV1().RESTClient(),
			"virtualserverroutes",
			lbc.namespace,
			fields.Everything()),
		&conf_v1.VirtualServerRoute{},
		lbc.resync,
		handlers,
	)
}

func (lbc *LoadBalancerController) addGlobalConfigurationHandler(handlers cache.ResourceEventHandlerFuncs, namespace string, name string) {
	lbc.globalConfiguratonLister, lbc.globalConfigurationController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.confClient.K8sV1alpha1().RESTClient(),
			"globalconfigurations",
			namespace,
			fields.Set{"metadata.name": name}.AsSelector()),
		&conf_v1alpha1.GlobalConfiguration{},
		lbc.resync,
		handlers,
	)
}

func (lbc *LoadBalancerController) addTransportServerHandler(handlers cache.ResourceEventHandlerFuncs) {
	lbc.transportServerLister, lbc.transportServerController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.confClient.K8sV1alpha1().RESTClient(),
			"transportservers",
			lbc.namespace,
			fields.Everything()),
		&conf_v1alpha1.TransportServer{},
		lbc.resync,
		handlers,
	)
}

// Run starts the loadbalancer controller
func (lbc *LoadBalancerController) Run() {
	lbc.ctx, lbc.cancel = context.WithCancel(context.Background())

	if lbc.spiffeController != nil {
		err := lbc.spiffeController.Start(lbc.ctx.Done())
		if err != nil {
			glog.Fatal(err)
		}
	}

	if lbc.leaderElector != nil {
		go lbc.leaderElector.Run(lbc.ctx)
	}
	go lbc.svcController.Run(lbc.ctx.Done())
	go lbc.podController.Run(lbc.ctx.Done())
	go lbc.endpointController.Run(lbc.ctx.Done())
	go lbc.secretController.Run(lbc.ctx.Done())
	if lbc.watchNginxConfigMaps {
		go lbc.configMapController.Run(lbc.ctx.Done())
	}
	go lbc.ingressController.Run(lbc.ctx.Done())
	if lbc.areCustomResourcesEnabled {
		go lbc.virtualServerController.Run(lbc.ctx.Done())
		go lbc.virtualServerRouteController.Run(lbc.ctx.Done())
		go lbc.transportServerController.Run(lbc.ctx.Done())
	}
	if lbc.watchGlobalConfiguration {
		go lbc.globalConfigurationController.Run(lbc.ctx.Done())
	}

	go lbc.syncQueue.Run(time.Second, lbc.ctx.Done())
	<-lbc.ctx.Done()
}

// Stop shutdowns the load balancer controller
func (lbc *LoadBalancerController) Stop() {
	lbc.cancel()

	lbc.syncQueue.Shutdown()
}

func (lbc *LoadBalancerController) syncEndpoint(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing endpoints %v", key)

	obj, endpExists, err := lbc.endpointLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if endpExists {
		ings := lbc.getIngressForEndpoints(obj)

		var ingExes []*configs.IngressEx
		var mergableIngressesSlice []*configs.MergeableIngresses

		for i := range ings {
			if !lbc.IsNginxIngress(&ings[i]) {
				continue
			}
			if isMinion(&ings[i]) {
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

		if lbc.areCustomResourcesEnabled {
			virtualServers := lbc.getVirtualServersForEndpoints(obj.(*api_v1.Endpoints))
			virtualServersExes := lbc.virtualServersToVirtualServerExes(virtualServers)

			if len(virtualServersExes) > 0 {
				glog.V(3).Infof("Updating endpoints for %v", virtualServersExes)
				err := lbc.configurator.UpdateEndpointsForVirtualServers(virtualServersExes)
				if err != nil {
					glog.Errorf("Error updating endpoints for %v: %v", virtualServersExes, err)
				}
			}

			transportServers := lbc.getTransportServersForEndpoints(obj.(*api_v1.Endpoints))
			transportServerExes := lbc.transportServersToTransportServerExes(transportServers)

			if len(transportServerExes) > 0 {
				glog.V(3).Infof("Updating endpoints for %v", transportServerExes)
				err := lbc.configurator.UpdateEndpointsForTransportServers(transportServerExes)
				if err != nil {
					glog.Errorf("Error updating endpoints for %v: %v", transportServerExes, err)
				}
			}
		}
	}
}

func (lbc *LoadBalancerController) syncConfig(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing configmap %v", key)

	obj, configExists, err := lbc.configMapLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}
	cfgParams := configs.NewDefaultConfigParams()

	if configExists {
		cfgm := obj.(*api_v1.ConfigMap)
		cfgParams = configs.ParseConfigMap(cfgm, lbc.isNginxPlus)

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

	var virtualServerExes []*configs.VirtualServerEx
	if lbc.areCustomResourcesEnabled {
		virtualServers := lbc.getVirtualServers()
		virtualServerExes = lbc.virtualServersToVirtualServerExes(virtualServers)
		if lbc.reportVsVsrStatusEnabled() {
			err = lbc.statusUpdater.UpdateVsVsrExternalEndpoints(virtualServers, lbc.getVirtualServerRoutes())
			if err != nil {
				glog.V(3).Infof("error updating status on ConfigMap change: %v", err)
			}
		}
	}

	warnings, updateErr := lbc.configurator.UpdateConfig(cfgParams, ingExes, mergeableIngresses, virtualServerExes)

	eventTitle := "Updated"
	eventType := api_v1.EventTypeNormal
	eventWarningMessage := ""

	if updateErr != nil {
		eventTitle = "UpdatedWithError"
		eventType = api_v1.EventTypeWarning
		eventWarningMessage = fmt.Sprintf("but was not applied: %v", updateErr)
	}
	cmWarningMessage := eventWarningMessage

	if len(warnings) > 0 && updateErr == nil {
		cmWarningMessage = "with warnings. Please check the logs"
	}

	if configExists {
		cfgm := obj.(*api_v1.ConfigMap)
		lbc.recorder.Eventf(cfgm, eventType, eventTitle, "Configuration from %v was updated %s", key, cmWarningMessage)
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
	for _, vsEx := range virtualServerExes {
		vsEventType := eventType
		vsEventTitle := eventTitle
		vsEventWarningMessage := eventWarningMessage
		vsState := conf_v1.StateValid

		if messages, ok := warnings[vsEx.VirtualServer]; ok && updateErr == nil {
			vsEventType = api_v1.EventTypeWarning
			vsEventTitle = "UpdatedWithWarning"
			vsEventWarningMessage = fmt.Sprintf("with warning(s): %v", formatWarningMessages(messages))
			vsState = conf_v1.StateWarning
		}

		msg := fmt.Sprintf("Configuration for %v/%v was updated %s", vsEx.VirtualServer.Namespace, vsEx.VirtualServer.Name, vsEventWarningMessage)
		lbc.recorder.Eventf(vsEx.VirtualServer, vsEventType, vsEventTitle, msg)

		if lbc.reportVsVsrStatusEnabled() {
			err = lbc.statusUpdater.UpdateVirtualServerStatus(vsEx.VirtualServer, vsState, vsEventTitle, msg)

			if err != nil {
				glog.Errorf("Error when updating the status for VirtualServer %v/%v: %v", vsEx.VirtualServer.Namespace, vsEx.VirtualServer.Name, err)
			}
		}

		for _, vsr := range vsEx.VirtualServerRoutes {
			vsrEventType := eventType
			vsrEventTitle := eventTitle
			vsrEventWarningMessage := eventWarningMessage
			vsrState := conf_v1.StateValid

			if messages, ok := warnings[vsr]; ok && updateErr == nil {
				vsrEventType = api_v1.EventTypeWarning
				vsrEventTitle = "UpdatedWithWarning"
				vsrEventWarningMessage = fmt.Sprintf("with warning(s): %v", formatWarningMessages(messages))
				vsrState = conf_v1.StateWarning
			}

			msg := fmt.Sprintf("Configuration for %v/%v was updated %s", vsr.Namespace, vsr.Name, vsrEventWarningMessage)
			lbc.recorder.Eventf(vsr, vsrEventType, vsrEventTitle, msg)

			if lbc.reportVsVsrStatusEnabled() {
				err = lbc.statusUpdater.UpdateVirtualServerRouteStatus(vsr, vsrState, vsrEventTitle, vsrEventWarningMessage)

				if err != nil {
					glog.Errorf("Error when updating the status for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
				}
			}
		}
	}
}

// GetManagedIngresses gets Ingress resources that the IC is currently responsible for
func (lbc *LoadBalancerController) GetManagedIngresses() ([]extensions.Ingress, map[string]*configs.MergeableIngresses) {
	mergeableIngresses := make(map[string]*configs.MergeableIngresses)
	var managedIngresses []extensions.Ingress
	ings, _ := lbc.ingressLister.List()
	for i := range ings.Items {
		ing := ings.Items[i]
		if !lbc.IsNginxIngress(&ing) {
			continue
		}
		if isMinion(&ing) {
			master, err := lbc.FindMasterForMinion(&ing)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", ing.Name, err)
				continue
			}
			if !lbc.configurator.HasIngress(master) {
				continue
			}
			if _, exists := mergeableIngresses[master.Name]; !exists {
				mergeableIngress, err := lbc.createMergableIngresses(master)
				if err != nil {
					glog.Errorf("Ignoring Ingress %v(Master): %v", master.Name, err)
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

func (lbc *LoadBalancerController) ingressesToIngressExes(ings []extensions.Ingress) []*configs.IngressEx {
	var ingExes []*configs.IngressEx
	for i := range ings {
		ingEx, err := lbc.createIngress(&ings[i])
		if err != nil {
			continue
		}
		ingExes = append(ingExes, ingEx)
	}
	return ingExes
}

func (lbc *LoadBalancerController) virtualServersToVirtualServerExes(virtualServers []*conf_v1.VirtualServer) []*configs.VirtualServerEx {
	var virtualServersExes []*configs.VirtualServerEx

	for _, vs := range virtualServers {
		vsEx, _ := lbc.createVirtualServer(vs) // ignoring VirtualServerRouteErrors
		virtualServersExes = append(virtualServersExes, vsEx)
	}

	return virtualServersExes
}

func (lbc *LoadBalancerController) transportServersToTransportServerExes(transportServers []*conf_v1alpha1.TransportServer) []*configs.TransportServerEx {
	var transportServerExes []*configs.TransportServerEx

	for _, ts := range transportServers {
		tsEx := lbc.createTransportServer(ts)
		transportServerExes = append(transportServerExes, tsEx)
	}

	return transportServerExes
}

func (lbc *LoadBalancerController) sync(task task) {
	glog.V(3).Infof("Syncing %v", task.Key)
	if lbc.spiffeController != nil {
		lbc.syncLock.Lock()
		defer lbc.syncLock.Unlock()
	}
	switch task.Kind {
	case ingress:
		lbc.syncIng(task)
		lbc.updateIngressMetrics()
	case ingressMinion:
		lbc.syncIngMinion(task)
		lbc.updateIngressMetrics()
	case configMap:
		lbc.syncConfig(task)
	case endpoints:
		lbc.syncEndpoint(task)
	case secret:
		lbc.syncSecret(task)
	case service:
		lbc.syncExternalService(task)
	case virtualserver:
		lbc.syncVirtualServer(task)
		lbc.updateVirtualServerMetrics()
	case virtualServerRoute:
		lbc.syncVirtualServerRoute(task)
		lbc.updateVirtualServerMetrics()
	case globalConfiguration:
		lbc.syncGlobalConfiguration(task)
	case transportserver:
		lbc.syncTransportServer(task)
	}
}

func (lbc *LoadBalancerController) syncTransportServer(task task) {
	key := task.Key
	obj, tsExists, err := lbc.transportServerLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if !tsExists {
		glog.V(2).Infof("Deleting TransportServer: %v\n", key)

		err := lbc.configurator.DeleteTransportServer(key)
		if err != nil {
			glog.Errorf("Error when deleting configuration for %v: %v", key, err)
		}
		return
	}

	glog.V(2).Infof("Adding or Updating TransportServer: %v\n", key)

	ts := obj.(*conf_v1alpha1.TransportServer)

	validationErr := lbc.transportServerValidator.ValidateTransportServer(ts)
	if validationErr != nil {
		err := lbc.configurator.DeleteTransportServer(key)
		if err != nil {
			glog.Errorf("Error when deleting configuration for %v: %v", key, err)
		}
		lbc.recorder.Eventf(ts, api_v1.EventTypeWarning, "Rejected", "TransportServer %v is invalid and was rejected: %v", key, validationErr)
		return
	}

	if !lbc.configurator.CheckIfListenerExists(&ts.Spec.Listener) {
		err := lbc.configurator.DeleteTransportServer(key)
		if err != nil {
			glog.Errorf("Error when deleting configuration for %v: %v", key, err)
		}
		lbc.recorder.Eventf(ts, api_v1.EventTypeWarning, "Rejected", "TransportServer %v references a non-existing listener and was rejected", key)
		return
	}

	tsEx := lbc.createTransportServer(ts)

	addErr := lbc.configurator.AddOrUpdateTransportServer(tsEx)

	eventTitle := "AddedOrUpdated"
	eventType := api_v1.EventTypeNormal
	eventWarningMessage := ""

	if addErr != nil {
		eventTitle = "AddedOrUpdatedWithError"
		eventType = api_v1.EventTypeWarning
		eventWarningMessage = fmt.Sprintf("but was not applied: %v", addErr)
	}

	lbc.recorder.Eventf(ts, eventType, eventTitle, "Configuration for %v was added or updated %v", key, eventWarningMessage)
}

func (lbc *LoadBalancerController) syncGlobalConfiguration(task task) {
	key := task.Key
	obj, gcExists, err := lbc.globalConfiguratonLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if !gcExists {
		glog.Warningf("GlobalConfiguration %v was removed. Retaining the GlobalConfiguration,", key)
		return
	}

	glog.V(2).Infof("GlobalConfiguration was updated: %v\n", key)

	gc := obj.(*conf_v1alpha1.GlobalConfiguration)

	validationErr := lbc.globalConfigurationValidator.ValidateGlobalConfiguration(gc)
	if validationErr != nil {
		lbc.recorder.Eventf(gc, api_v1.EventTypeWarning, "Rejected", "GlobalConfiguration %v is invalid and was rejected: %v", key, validationErr)
		return
	}

	// GlobalConfiguration configures listeners
	// As a result, a change in a GC might affect all TransportServers

	transportServerExes := lbc.transportServersToTransportServerExes(lbc.getTransportServers())

	updatedTransportServerExes, deletedTransportServerExes, updateErr := lbc.configurator.UpdateGlobalConfiguration(gc, transportServerExes)

	for _, tsEx := range deletedTransportServerExes {
		eventTitle := "Rejected"
		eventType := api_v1.EventTypeWarning
		eventWarningMessage := ""

		if updateErr != nil {
			eventTitle = "RejectedWithError"
			eventWarningMessage = fmt.Sprintf("but was not applied: %v", updateErr)
		}

		lbc.recorder.Eventf(tsEx.TransportServer, eventType, eventTitle, "TransportServer %v/%v references a non-existing listener %v", tsEx.TransportServer.Namespace, tsEx.TransportServer.Name, eventWarningMessage)
	}

	eventTitle := "Updated"
	eventType := api_v1.EventTypeNormal
	eventWarningMessage := ""

	if updateErr != nil {
		eventTitle = "UpdatedWithError"
		eventType = api_v1.EventTypeWarning
		eventWarningMessage = fmt.Sprintf("but was not applied: %v", updateErr)
	}

	lbc.recorder.Eventf(gc, eventType, eventTitle, "GlobalConfiguration %v was updated %v", key, eventWarningMessage)

	for _, tsEx := range updatedTransportServerExes {
		lbc.recorder.Eventf(tsEx.TransportServer, eventType, eventTitle, "TransportServer %v/%v was updated %v", tsEx.TransportServer.Namespace, tsEx.TransportServer.Name, eventWarningMessage)
	}
}

func (lbc *LoadBalancerController) syncVirtualServer(task task) {
	key := task.Key
	obj, vsExists, err := lbc.virtualServerLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}
	vsrlist := findAllVirtualServerRoutesForVirtualServer(key, lbc.getVirtualServerRoutes())

	if !vsExists {
		glog.V(2).Infof("Deleting VirtualServer: %v\n", key)

		err := lbc.configurator.DeleteVirtualServer(key)
		if err != nil {
			glog.Errorf("Error when deleting configuration for %v: %v", key, err)
		} else {
			reason := "NoVirtualServerFound"
			for _, vsr := range vsrlist {
				msg := fmt.Sprintf("No VirtualServer references VirtualServerRoute %v/%v", vsr.Namespace, vsr.Name)
				lbc.recorder.Eventf(vsr, api_v1.EventTypeWarning, reason, msg)
				lbc.statusUpdater.UpdateVirtualServerRouteStatus(vsr, conf_v1.StateInvalid, reason, msg)
			}
		}

		return
	}

	glog.V(2).Infof("Adding or Updating VirtualServer: %v\n", key)
	vs := obj.(*conf_v1.VirtualServer)

	validationErr := validation.ValidateVirtualServer(vs, lbc.isNginxPlus)
	if validationErr != nil {
		err := lbc.configurator.DeleteVirtualServer(key)
		if err != nil {
			glog.Errorf("Error when deleting configuration for %v: %v", key, err)
		}
		reason := "NoVirtualServerFound"
		for _, vsr := range vsrlist {
			msg := fmt.Sprintf("No VirtualServer references VirtualServerRoute %v/%v", vsr.Namespace, vsr.Name)
			lbc.recorder.Eventf(vsr, api_v1.EventTypeWarning, reason, msg)
			err := lbc.statusUpdater.UpdateVirtualServerRouteStatus(vsr, conf_v1.StateInvalid, reason, msg)
			if err != nil {
				glog.Errorf("Error when updating the status for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
			}
		}
		lbc.recorder.Eventf(vs, api_v1.EventTypeWarning, "Rejected", "VirtualServer %v is invalid and was rejected: %v", key, validationErr)
		return
	}

	vsEx, vsrErrors := lbc.createVirtualServer(vs)

	for _, vsrError := range vsrErrors {
		lbc.recorder.Eventf(vs, api_v1.EventTypeWarning, "IgnoredVirtualServerRoute", "Ignored VirtualServerRoute %v: %v", vsrError.VirtualServerRouteNsName, vsrError.Error)
		if vsrError.VirtualServerRoute != nil {
			lbc.recorder.Eventf(vsrError.VirtualServerRoute, api_v1.EventTypeWarning, "Ignored", "Ignored by VirtualServer %v/%v: %v", vs.Namespace, vs.Name, vsrError.Error)
		}
	}

	warnings, addErr := lbc.configurator.AddOrUpdateVirtualServer(vsEx)

	eventTitle := "AddedOrUpdated"
	eventType := api_v1.EventTypeNormal
	eventWarningMessage := ""
	state := conf_v1.StateValid

	if addErr != nil {
		eventTitle = "AddedOrUpdatedWithError"
		eventType = api_v1.EventTypeWarning
		eventWarningMessage = fmt.Sprintf("but was not applied: %v", addErr)
		state = conf_v1.StateInvalid
	}

	vsEventType := eventType
	vsEventTitle := eventTitle
	vsEventWarningMessage := eventWarningMessage

	if messages, ok := warnings[vsEx.VirtualServer]; ok && addErr == nil {
		vsEventType = api_v1.EventTypeWarning
		vsEventTitle = "AddedOrUpdatedWithWarning"
		vsEventWarningMessage = fmt.Sprintf("with warning(s): %v", formatWarningMessages(messages))
		state = conf_v1.StateWarning
	}

	msg := fmt.Sprintf("Configuration for %v was added or updated %s", key, vsEventWarningMessage)
	lbc.recorder.Eventf(vs, vsEventType, vsEventTitle, msg)

	if lbc.reportVsVsrStatusEnabled() {
		err = lbc.statusUpdater.UpdateVirtualServerStatus(vs, state, vsEventTitle, msg)

		if err != nil {
			glog.Errorf("Error when updating the status for VirtualServer %v/%v: %v", vs.Namespace, vs.Name, err)
		}
	}

	key = fmt.Sprintf("%v/%v", vs.Namespace, vs.Name)
	vsrlist = findAllVirtualServerRoutesForVirtualServer(key, lbc.getVirtualServerRoutes())

	for _, vsr := range vsEx.VirtualServerRoutes {
		vsrEventType := eventType
		vsrEventTitle := eventTitle
		vsrEventWarningMessage := eventWarningMessage
		state := conf_v1.StateValid

		if messages, ok := warnings[vsr]; ok && addErr == nil {
			vsrEventType = api_v1.EventTypeWarning
			vsrEventTitle = "AddedOrUpdatedWithWarning"
			vsrEventWarningMessage = fmt.Sprintf("with warning(s): %v", formatWarningMessages(messages))
			state = conf_v1.StateWarning
		}
		msg := fmt.Sprintf("Configuration for %v/%v was added or updated %s", vsr.Namespace, vsr.Name, vsrEventWarningMessage)
		lbc.recorder.Eventf(vsr, vsrEventType, vsrEventTitle, msg)

		if lbc.reportVsVsrStatusEnabled() {
			virtualServersForVSR := findVirtualServersForVirtualServerRoute(lbc.getVirtualServers(), vsr)
			err = lbc.statusUpdater.UpdateVirtualServerRouteStatusWithReferencedBy(vsr, state, vsrEventTitle, msg, virtualServersForVSR)

			if err != nil {
				glog.Errorf("Error when updating the status for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
			}
		}

		vsrkey := fmt.Sprintf("%s/%s", vsr.Namespace, vsr.Name)
		vsrlist = checkForVirtualServerRoute(vsrkey, vsrlist)
	}
	reason := "Ignored"
	for _, vsr := range vsrlist {
		msg := fmt.Sprintf("Ignored by VirtualServer %v/%v", vs.Namespace, vs.Name)
		lbc.recorder.Eventf(vsr, api_v1.EventTypeWarning, "Ignored", msg)
		err := lbc.statusUpdater.UpdateVirtualServerRouteStatus(vsr, conf_v1.StateInvalid, reason, msg)
		if err != nil {
			glog.Errorf("Error when updating the status for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
		}
	}
}

func checkForVirtualServerRoute(vsrkey string, vsrlist []*conf_v1.VirtualServerRoute) []*conf_v1.VirtualServerRoute {
	var vsrNotRef []*conf_v1.VirtualServerRoute
	for i, r := range vsrlist {
		key := fmt.Sprintf("%s/%s", r.Namespace, r.Name)
		if key == vsrkey {
			if len(vsrlist) > 1 {
				vsrNotRef = append(vsrlist[:i], vsrlist[i+1:]...)
			} else {
				vsrNotRef = []*conf_v1.VirtualServerRoute{}
			}
		}
	}

	return vsrNotRef
}

func findAllVirtualServerRoutesForVirtualServer(key string, vsrs []*conf_v1.VirtualServerRoute) []*conf_v1.VirtualServerRoute {
	var result []*conf_v1.VirtualServerRoute
	for _, r := range vsrs {
		if r.Status.ReferencedBy == key {
			result = append(result, r)
		}
	}
	return result
}

func findVirtualServerRoutesForVirtualServer(vs *conf_v1.VirtualServer, vsrs []*conf_v1.VirtualServerRoute) []*conf_v1.VirtualServerRoute {
	var result []*conf_v1.VirtualServerRoute
	vslist := []*conf_v1.VirtualServer{vs}
	for _, vsr := range vsrs {
		key := fmt.Sprintf("%s/%s", vsr.Namespace, vsr.Name)
		if len(findVirtualServersForVirtualServerRouteKey(vslist, key)) != 0 {
			result = append(result, vsr)
		}
	}
	return result
}

func (lbc *LoadBalancerController) syncVirtualServerRoute(task task) {
	key := task.Key

	obj, exists, err := lbc.virtualServerRouteLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if !exists {
		glog.V(2).Infof("Deleting VirtualServerRoute: %v\n", key)

		lbc.enqueueVirtualServersForVirtualServerRouteKey(key)
		return
	}

	glog.V(2).Infof("Adding or Updating VirtualServerRoute: %v\n", key)

	vsr := obj.(*conf_v1.VirtualServerRoute)

	validationErr := validation.ValidateVirtualServerRoute(vsr, lbc.isNginxPlus)
	if validationErr != nil {
		reason := "Rejected"
		msg := fmt.Sprintf("VirtualServerRoute %s is invalid and was rejected: %v", key, validationErr)
		lbc.recorder.Eventf(vsr, api_v1.EventTypeWarning, reason, msg)
		if lbc.reportVsVsrStatusEnabled() {
			err = lbc.statusUpdater.UpdateVirtualServerRouteStatus(vsr, conf_v1.StateInvalid, reason, msg)
			if err != nil {
				glog.Errorf("Error when updating the status for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
			}
		}
	}
	vsCount := lbc.enqueueVirtualServersForVirtualServerRouteKey(key)

	if vsCount == 0 {
		reason := "NoVirtualServersFound"
		msg := fmt.Sprintf("No VirtualServer references VirtualServerRoute %s", key)
		lbc.recorder.Eventf(vsr, api_v1.EventTypeWarning, reason, msg)

		if lbc.reportVsVsrStatusEnabled() {
			err = lbc.statusUpdater.UpdateVirtualServerRouteStatus(vsr, conf_v1.StateInvalid, reason, msg)
			if err != nil {
				glog.Errorf("Error when updating the status for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
			}
		}
	}
}

func (lbc *LoadBalancerController) syncIngMinion(task task) {
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

func (lbc *LoadBalancerController) syncIng(task task) {
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

		if isMaster(ing) {
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
				lbc.recorder.Eventf(minion.Ingress, eventType, eventTitle, "Configuration for %v/%v(Minion) was added or updated %s", minion.Ingress.Namespace, minion.Ingress.Name, eventWarningMessage)
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

func (lbc *LoadBalancerController) updateIngressMetrics() {
	counters := lbc.configurator.GetIngressCounts()
	for nType, count := range counters {
		lbc.metricsCollector.SetIngresses(nType, count)
	}
}

func (lbc *LoadBalancerController) updateVirtualServerMetrics() {
	vsCount, vsrCount := lbc.configurator.GetVirtualServerCounts()
	lbc.metricsCollector.SetVirtualServers(vsCount)
	lbc.metricsCollector.SetVirtualServerRoutes(vsrCount)
}

// syncExternalService does not sync all services.
// We only watch the Service specified by the external-service flag.
func (lbc *LoadBalancerController) syncExternalService(task task) {
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

	if lbc.areCustomResourcesEnabled && lbc.reportVsVsrStatusEnabled() {
		err = lbc.statusUpdater.UpdateVsVsrExternalEndpoints(lbc.getVirtualServers(), lbc.getVirtualServerRoutes())
		if err != nil {
			glog.V(3).Infof("error updating VirtualServer/VirtualServerRoute status in syncExternalService: %v", err)
		}
	}
}

// IsExternalServiceForStatus matches the service specified by the external-service arg
func (lbc *LoadBalancerController) IsExternalServiceForStatus(svc *api_v1.Service) bool {
	return lbc.statusUpdater.namespace == svc.Namespace && lbc.statusUpdater.externalServiceName == svc.Name
}

// reportStatusEnabled determines if we should attempt to report status for Ingress resources.
func (lbc *LoadBalancerController) reportStatusEnabled() bool {
	if lbc.reportIngressStatus {
		if lbc.isLeaderElectionEnabled {
			return lbc.leaderElector != nil && lbc.leaderElector.IsLeader()
		}
		return true
	}
	return false
}

// reportVsVsrStatusEnabled determines if we should attempt to report status for VirtualServers and VirtualServerRoutes.
func (lbc *LoadBalancerController) reportVsVsrStatusEnabled() bool {
	if lbc.isLeaderElectionEnabled {
		return lbc.leaderElector != nil && lbc.leaderElector.IsLeader()
	}

	return true
}

func (lbc *LoadBalancerController) syncSecret(task task) {
	key := task.Key
	obj, secrExists, err := lbc.secretLister.Store.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	namespace, name, err := ParseNamespaceName(key)
	if err != nil {
		glog.Warningf("Secret key %v is invalid: %v", key, err)
		return
	}

	ings, err := lbc.findIngressesForSecret(namespace, name)
	if err != nil {
		glog.Warningf("Failed to find Ingress resources for Secret %v: %v", key, err)
		lbc.syncQueue.RequeueAfter(task, err, 5*time.Second)
	}

	var virtualServers []*conf_v1.VirtualServer
	if lbc.areCustomResourcesEnabled {
		virtualServers = lbc.getVirtualServersForSecret(namespace, name)
		glog.V(2).Infof("Found %v VirtualServers with Secret %v", len(virtualServers), key)
	}

	glog.V(2).Infof("Found %v Ingresses with Secret %v", len(ings), key)

	if !secrExists {
		glog.V(2).Infof("Deleting Secret: %v\n", key)

		lbc.handleRegularSecretDeletion(key, ings, virtualServers)
		if lbc.isSpecialSecret(key) {
			glog.Warningf("A special TLS Secret %v was removed. Retaining the Secret.", key)
		}
		return
	}

	glog.V(2).Infof("Adding / Updating Secret: %v\n", key)

	secret := obj.(*api_v1.Secret)

	if lbc.isSpecialSecret(key) {
		lbc.handleSpecialSecretUpdate(secret)
		// we don't return here in case the special secret is also used in Ingress or VirtualServer resources.
	}

	if len(ings)+len(virtualServers) > 0 {
		lbc.handleSecretUpdate(secret, ings, virtualServers)
	}
}

func (lbc *LoadBalancerController) isSpecialSecret(secretName string) bool {
	return secretName == lbc.defaultServerSecret || secretName == lbc.wildcardTLSSecret
}

func (lbc *LoadBalancerController) handleRegularSecretDeletion(key string, ings []extensions.Ingress, virtualServers []*conf_v1.VirtualServer) {
	eventType := api_v1.EventTypeWarning
	title := "Missing Secret"
	message := fmt.Sprintf("Secret %v was removed", key)
	state := conf_v1.StateInvalid

	lbc.emitEventForIngresses(eventType, title, message, ings)
	lbc.emitEventForVirtualServers(eventType, title, message, virtualServers)
	lbc.updateStatusForVirtualServers(state, title, message, virtualServers)

	regular, mergeable := lbc.createIngresses(ings)

	virtualServerExes := lbc.virtualServersToVirtualServerExes(virtualServers)

	eventType = api_v1.EventTypeNormal
	title = "Updated"
	message = fmt.Sprintf("Configuration was updated due to removed secret %v", key)

	if err := lbc.configurator.DeleteSecret(key, regular, mergeable, virtualServerExes); err != nil {
		glog.Errorf("Error when deleting Secret: %v: %v", key, err)

		eventType = api_v1.EventTypeWarning
		title = "UpdatedWithError"
		message = fmt.Sprintf("Configuration was updated due to removed secret %v, but not applied: %v", key, err)
		state = conf_v1.StateInvalid
	}

	lbc.emitEventForIngresses(eventType, title, message, ings)
	lbc.emitEventForVirtualServers(eventType, title, message, virtualServers)
	lbc.updateStatusForVirtualServers(state, title, message, virtualServers)
}

func (lbc *LoadBalancerController) handleSecretUpdate(secret *api_v1.Secret, ings []extensions.Ingress, virtualServers []*conf_v1.VirtualServer) {
	secretNsName := secret.Namespace + "/" + secret.Name

	err := lbc.ValidateSecret(secret)
	if err != nil {
		// Secret becomes Invalid
		glog.Errorf("Couldn't validate secret %v: %v", secretNsName, err)
		glog.Errorf("Removing invalid secret %v", secretNsName)

		lbc.handleRegularSecretDeletion(secretNsName, ings, virtualServers)

		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "Rejected", "%v was rejected: %v", secretNsName, err)
		return
	}

	eventType := api_v1.EventTypeNormal
	title := "Updated"
	message := fmt.Sprintf("Configuration was updated due to updated secret %v", secretNsName)
	state := conf_v1.StateValid

	// we can safely ignore the error because the secret is valid in this function
	kind, _ := GetSecretKind(secret)

	if kind == JWK {
		lbc.configurator.AddOrUpdateJWKSecret(secret)
	} else {
		regular, mergeable := lbc.createIngresses(ings)

		virtualServerExes := lbc.virtualServersToVirtualServerExes(virtualServers)

		err := lbc.configurator.AddOrUpdateTLSSecret(secret, regular, mergeable, virtualServerExes)
		if err != nil {
			glog.Errorf("Error when updating Secret %v: %v", secretNsName, err)
			lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "%v was updated, but not applied: %v", secretNsName, err)

			eventType = api_v1.EventTypeWarning
			title = "UpdatedWithError"
			message = fmt.Sprintf("Configuration was updated due to updated secret %v, but not applied: %v", secretNsName, err)
			state = conf_v1.StateInvalid
		}
	}

	lbc.emitEventForIngresses(eventType, title, message, ings)
	lbc.emitEventForVirtualServers(eventType, title, message, virtualServers)
	lbc.updateStatusForVirtualServers(state, title, message, virtualServers)
}

func (lbc *LoadBalancerController) handleSpecialSecretUpdate(secret *api_v1.Secret) {
	var specialSecretsToUpdate []string
	secretNsName := secret.Namespace + "/" + secret.Name
	err := ValidateTLSSecret(secret)
	if err != nil {
		glog.Errorf("Couldn't validate the special Secret %v: %v", secretNsName, err)
		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "Rejected", "the special Secret %v was rejected, using the previous version: %v", secretNsName, err)
		return
	}

	if secretNsName == lbc.defaultServerSecret {
		specialSecretsToUpdate = append(specialSecretsToUpdate, configs.DefaultServerSecretName)
	}
	if secretNsName == lbc.wildcardTLSSecret {
		specialSecretsToUpdate = append(specialSecretsToUpdate, configs.WildcardSecretName)
	}

	err = lbc.configurator.AddOrUpdateSpecialTLSSecrets(secret, specialSecretsToUpdate)
	if err != nil {
		glog.Errorf("Error when updating the special Secret %v: %v", secretNsName, err)
		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "the special Secret %v was updated, but not applied: %v", secretNsName, err)
		return
	}

	lbc.recorder.Eventf(secret, api_v1.EventTypeNormal, "Updated", "the special Secret %v was updated", secretNsName)
}

func (lbc *LoadBalancerController) emitEventForIngresses(eventType string, title string, message string, ings []extensions.Ingress) {
	for _, ing := range ings {
		lbc.recorder.Eventf(&ing, eventType, title, message)
		if isMinion(&ing) {
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

func (lbc *LoadBalancerController) emitEventForVirtualServers(eventType string, title string, message string, virtualServers []*conf_v1.VirtualServer) {
	for _, vs := range virtualServers {
		lbc.recorder.Eventf(vs, eventType, title, message)
	}
}

func (lbc *LoadBalancerController) updateStatusForVirtualServers(state string, reason string, message string, virtualServers []*conf_v1.VirtualServer) {
	for _, vs := range virtualServers {
		err := lbc.statusUpdater.UpdateVirtualServerStatus(vs, state, reason, message)
		if err != nil {
			glog.Errorf("Error when updating the status for VirtualServer %v/%v: %v", vs.Namespace, vs.Name, err)
		}
	}
}

func (lbc *LoadBalancerController) createIngresses(ings []extensions.Ingress) (regular []configs.IngressEx, mergeable []configs.MergeableIngresses) {
	for i := range ings {
		if isMaster(&ings[i]) {
			mergeableIng, err := lbc.createMergableIngresses(&ings[i])
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Master): %v", ings[i].Name, err)
				continue
			}
			mergeable = append(mergeable, *mergeableIng)
			continue
		}

		if isMinion(&ings[i]) {
			master, err := lbc.FindMasterForMinion(&ings[i])
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Minion): %v", ings[i].Name, err)
				continue
			}
			mergeableIng, err := lbc.createMergableIngresses(master)
			if err != nil {
				glog.Errorf("Ignoring Ingress %v(Master): %v", master.Name, err)
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

		if !isMinion(&ing) {
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
				if jwtKey, exists := ing.Annotations[configs.JWTKeyAnnotation]; exists {
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

			if jwtKey, exists := ing.Annotations[configs.JWTKeyAnnotation]; exists {
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
		if isMinion(&ing) {
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

// EnqueueVirtualServersForService enqueues VirtualServers for the given service.
func (lbc *LoadBalancerController) EnqueueVirtualServersForService(service *api_v1.Service) {
	virtualServers := lbc.getVirtualServersForService(service)
	for _, vs := range virtualServers {
		lbc.syncQueue.Enqueue(vs)
	}
}

// EnqueueTransportServerForService enqueues TransportServers for the given service.
func (lbc *LoadBalancerController) EnqueueTransportServerForService(service *api_v1.Service) {
	transportServers := lbc.getTransportServersForService(service)
	for _, ts := range transportServers {
		lbc.syncQueue.Enqueue(ts)
	}
}

func (lbc *LoadBalancerController) getIngressesForService(svc *api_v1.Service) []extensions.Ingress {
	ings, err := lbc.ingressLister.GetServiceIngress(svc)
	if err != nil {
		glog.V(3).Infof("For service %v: %v", svc.Name, err)
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

func (lbc *LoadBalancerController) getVirtualServersForEndpoints(endpoints *api_v1.Endpoints) []*conf_v1.VirtualServer {
	svcKey := fmt.Sprintf("%s/%s", endpoints.Namespace, endpoints.Name)

	svc, exists, err := lbc.svcLister.GetByKey(svcKey)
	if err != nil {
		glog.V(3).Infof("Error getting service %v from the cache: %v", svcKey, err)
		return nil
	}
	if !exists {
		glog.V(3).Infof("Service %v doesn't exist", svcKey)
		return nil
	}

	return lbc.getVirtualServersForService(svc.(*api_v1.Service))
}

func (lbc *LoadBalancerController) getVirtualServersForService(service *api_v1.Service) []*conf_v1.VirtualServer {
	var result []*conf_v1.VirtualServer

	allVirtualServers := lbc.getVirtualServers()

	// find VirtualServers that reference VirtualServerRoutes that reference the service
	virtualServerRoutes := findVirtualServerRoutesForService(lbc.getVirtualServerRoutes(), service)
	for _, vsr := range virtualServerRoutes {
		virtualServers := findVirtualServersForVirtualServerRoute(allVirtualServers, vsr)
		result = append(result, virtualServers...)
	}

	// find VirtualServers that reference the service
	virtualServers := findVirtualServersForService(lbc.getVirtualServers(), service)
	result = append(result, virtualServers...)

	return result
}

func findVirtualServersForService(virtualServers []*conf_v1.VirtualServer, service *api_v1.Service) []*conf_v1.VirtualServer {
	var result []*conf_v1.VirtualServer

	for _, vs := range virtualServers {
		if vs.Namespace != service.Namespace {
			continue
		}

		isReferenced := false
		for _, u := range vs.Spec.Upstreams {
			if u.Service == service.Name {
				isReferenced = true
				break
			}
		}
		if !isReferenced {
			continue
		}

		result = append(result, vs)
	}

	return result
}

func (lbc *LoadBalancerController) getTransportServersForEndpoints(endpoints *api_v1.Endpoints) []*conf_v1alpha1.TransportServer {
	svcKey := fmt.Sprintf("%s/%s", endpoints.Namespace, endpoints.Name)

	svc, exists, err := lbc.svcLister.GetByKey(svcKey)
	if err != nil {
		glog.V(3).Infof("Error getting service %v from the cache: %v", svcKey, err)
		return nil
	}
	if !exists {
		glog.V(3).Infof("Service %v doesn't exist", svcKey)
		return nil
	}

	return lbc.getTransportServersForService(svc.(*api_v1.Service))
}

func (lbc *LoadBalancerController) getTransportServersForService(service *api_v1.Service) []*conf_v1alpha1.TransportServer {
	filtered := lbc.filterOutTransportServersWithNonExistingListener(lbc.getTransportServers())
	return findTransportServersForService(filtered, service)
}

func findTransportServersForService(transportServers []*conf_v1alpha1.TransportServer, service *api_v1.Service) []*conf_v1alpha1.TransportServer {
	var result []*conf_v1alpha1.TransportServer

	for _, ts := range transportServers {
		if ts.Namespace != service.Namespace {
			continue
		}

		for _, u := range ts.Spec.Upstreams {
			if u.Service == service.Name {
				result = append(result, ts)
				break
			}
		}
	}

	return result
}

func (lbc *LoadBalancerController) filterOutTransportServersWithNonExistingListener(transportServers []*conf_v1alpha1.TransportServer) []*conf_v1alpha1.TransportServer {
	var result []*conf_v1alpha1.TransportServer

	for _, ts := range transportServers {
		if lbc.configurator.CheckIfListenerExists(&ts.Spec.Listener) {
			result = append(result, ts)
		} else {
			glog.V(3).Infof("Ignoring TransportServer %s/%s references a non-existing listener", ts.Namespace, ts.Name)
		}
	}

	return result
}

func findVirtualServerRoutesForService(virtualServerRoutes []*conf_v1.VirtualServerRoute, service *api_v1.Service) []*conf_v1.VirtualServerRoute {
	var result []*conf_v1.VirtualServerRoute

	for _, vsr := range virtualServerRoutes {
		if vsr.Namespace != service.Namespace {
			continue
		}

		isReferenced := false
		for _, u := range vsr.Spec.Upstreams {
			if u.Service == service.Name {
				isReferenced = true
				break
			}
		}
		if !isReferenced {
			continue
		}

		result = append(result, vsr)
	}

	return result
}

func (lbc *LoadBalancerController) getVirtualServersForSecret(secretNamespace string, secretName string) []*conf_v1.VirtualServer {
	virtualServers := lbc.getVirtualServers()
	return findVirtualServersForSecret(virtualServers, secretNamespace, secretName)
}

func findVirtualServersForSecret(virtualServers []*conf_v1.VirtualServer, secretNamespace string, secretName string) []*conf_v1.VirtualServer {
	var result []*conf_v1.VirtualServer

	for _, vs := range virtualServers {
		if vs.Spec.TLS == nil {
			continue
		}
		if vs.Spec.TLS.Secret == "" {
			continue
		}

		if vs.Namespace == secretNamespace && vs.Spec.TLS.Secret == secretName {
			result = append(result, vs)
		}
	}

	return result
}

func (lbc *LoadBalancerController) getVirtualServers() []*conf_v1.VirtualServer {
	var virtualServers []*conf_v1.VirtualServer

	for _, obj := range lbc.virtualServerLister.List() {
		vs := obj.(*conf_v1.VirtualServer)

		err := validation.ValidateVirtualServer(vs, lbc.isNginxPlus)
		if err != nil {
			glog.V(3).Infof("Skipping invalid VirtualServer %s/%s: %v", vs.Namespace, vs.Name, err)
			continue
		}

		virtualServers = append(virtualServers, vs)
	}

	return virtualServers
}

func (lbc *LoadBalancerController) getVirtualServerRoutes() []*conf_v1.VirtualServerRoute {
	var virtualServerRoutes []*conf_v1.VirtualServerRoute

	for _, obj := range lbc.virtualServerRouteLister.List() {
		vsr := obj.(*conf_v1.VirtualServerRoute)

		err := validation.ValidateVirtualServerRoute(vsr, lbc.isNginxPlus)
		if err != nil {
			glog.V(3).Infof("Skipping invalid VirtualServerRoute %s/%s: %v", vsr.Namespace, vsr.Name, err)
			continue
		}

		virtualServerRoutes = append(virtualServerRoutes, vsr)
	}

	return virtualServerRoutes
}

func (lbc *LoadBalancerController) getTransportServers() []*conf_v1alpha1.TransportServer {
	var transportServers []*conf_v1alpha1.TransportServer

	for _, obj := range lbc.transportServerLister.List() {
		ts := obj.(*conf_v1alpha1.TransportServer)

		err := lbc.transportServerValidator.ValidateTransportServer(ts)
		if err != nil {
			glog.V(3).Infof("Skipping invalid TransportServer %s/%s: %v", ts.Namespace, ts.Name, err)
			continue
		}

		transportServers = append(transportServers, ts)
	}

	return transportServers
}

func (lbc *LoadBalancerController) enqueueVirtualServersForVirtualServerRouteKey(key string) int {
	virtualServers := findVirtualServersForVirtualServerRouteKey(lbc.getVirtualServers(), key)

	for _, vs := range virtualServers {
		lbc.syncQueue.Enqueue(vs)
	}

	return len(virtualServers)
}

func findVirtualServersForVirtualServerRoute(virtualServers []*conf_v1.VirtualServer, virtualServerRoute *conf_v1.VirtualServerRoute) []*conf_v1.VirtualServer {
	key := fmt.Sprintf("%s/%s", virtualServerRoute.Namespace, virtualServerRoute.Name)
	return findVirtualServersForVirtualServerRouteKey(virtualServers, key)
}

func findVirtualServersForVirtualServerRouteKey(virtualServers []*conf_v1.VirtualServer, key string) []*conf_v1.VirtualServer {
	var result []*conf_v1.VirtualServer

	for _, vs := range virtualServers {
		for _, r := range vs.Spec.Routes {
			// if route is defined without a namespace, use the namespace of VirtualServer.
			vsrKey := r.Route
			if !strings.Contains(r.Route, "/") {
				vsrKey = fmt.Sprintf("%s/%s", vs.Namespace, r.Route)
			}

			if vsrKey == key {
				result = append(result, vs)
				break
			}
		}
	}

	return result
}

func (lbc *LoadBalancerController) getAndValidateSecret(secretKey string) (*api_v1.Secret, error) {
	secretObject, secretExists, err := lbc.secretLister.GetByKey(secretKey)
	if err != nil {
		return nil, fmt.Errorf("error retrieving secret %v", secretKey)
	}
	if !secretExists {
		return nil, fmt.Errorf("secret %v not found", secretKey)
	}
	secret := secretObject.(*api_v1.Secret)

	err = ValidateTLSSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("error validating secret %v", secretKey)
	}
	return secret, nil
}

func (lbc *LoadBalancerController) createIngress(ing *extensions.Ingress) (*configs.IngressEx, error) {
	ingEx := &configs.IngressEx{
		Ingress: ing,
	}

	ingEx.TLSSecrets = make(map[string]*api_v1.Secret)
	for _, tls := range ing.Spec.TLS {
		secretName := tls.SecretName
		secretKey := ing.Namespace + "/" + secretName
		secret, err := lbc.getAndValidateSecret(secretKey)
		if err != nil {
			glog.Warningf("Error trying to get the secret %v for Ingress %v: %v", secretName, ing.Name, err)
			continue
		}
		ingEx.TLSSecrets[secretName] = secret
	}

	if lbc.isNginxPlus {
		if jwtKey, exists := ingEx.Ingress.Annotations[configs.JWTKeyAnnotation]; exists {
			secretName := jwtKey

			secret, err := lbc.client.CoreV1().Secrets(ing.Namespace).Get(context.TODO(), secretName, meta_v1.GetOptions{})
			if err != nil {
				glog.Warningf("Error retrieving secret %v for Ingress %v: %v", secretName, ing.Name, err)
				secret = nil
			} else {
				err = ValidateJWKSecret(secret)
				if err != nil {
					glog.Warningf("Error validating secret %v for Ingress %v: %v", secretName, ing.Name, err)
					secret = nil
				}
			}

			ingEx.JWTKey = configs.JWTKey{
				Name:   jwtKey,
				Secret: secret,
			}
		}
	}

	ingEx.Endpoints = make(map[string][]string)
	ingEx.HealthChecks = make(map[string]*api_v1.Probe)
	ingEx.ExternalNameSvcs = make(map[string]bool)

	if ing.Spec.Backend != nil {
		endps := []string{}
		var external bool
		svc, err := lbc.getServiceForIngressBackend(ing.Spec.Backend, ing.Namespace)
		if err != nil {
			glog.V(3).Infof("Error getting service %v: %v", ing.Spec.Backend.ServiceName, err)
		} else {
			endps, external, err = lbc.getEndpointsForIngressBackend(ing.Spec.Backend, svc)
			if err == nil && external && lbc.isNginxPlus {
				ingEx.ExternalNameSvcs[svc.Name] = true
			}
		}

		if err != nil {
			glog.Warningf("Error retrieving endpoints for the service %v: %v", ing.Spec.Backend.ServiceName, err)
		}
		// endps is empty if there was any error before this point
		ingEx.Endpoints[ing.Spec.Backend.ServiceName+ing.Spec.Backend.ServicePort.String()] = endps

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
			endps := []string{}
			var external bool
			svc, err := lbc.getServiceForIngressBackend(&path.Backend, ing.Namespace)
			if err != nil {
				glog.V(3).Infof("Error getting service %v: %v", &path.Backend.ServiceName, err)
			} else {
				endps, external, err = lbc.getEndpointsForIngressBackend(&path.Backend, svc)
				if err == nil && external && lbc.isNginxPlus {
					ingEx.ExternalNameSvcs[svc.Name] = true
				}
			}

			if err != nil {
				glog.Warningf("Error retrieving endpoints for the service %v: %v", path.Backend.ServiceName, err)
			}
			// endps is empty if there was any error before this point
			ingEx.Endpoints[path.Backend.ServiceName+path.Backend.ServicePort.String()] = endps

			// Pull active health checks from k8 api
			if lbc.isNginxPlus && lbc.isHealthCheckEnabled(ing) {
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

type virtualServerRouteError struct {
	VirtualServerRouteNsName string
	VirtualServerRoute       *conf_v1.VirtualServerRoute
	Error                    error
}

func newVirtualServerRouteErrorFromNsName(nsName string, err error) virtualServerRouteError {
	return virtualServerRouteError{
		VirtualServerRouteNsName: nsName,
		Error:                    err,
	}
}

func newVirtualServerRouteErrorFromVSR(virtualServerRoute *conf_v1.VirtualServerRoute, err error) virtualServerRouteError {
	return virtualServerRouteError{
		VirtualServerRoute:       virtualServerRoute,
		VirtualServerRouteNsName: fmt.Sprintf("%s/%s", virtualServerRoute.Namespace, virtualServerRoute.Name),
		Error:                    err,
	}
}

func (lbc *LoadBalancerController) createVirtualServer(virtualServer *conf_v1.VirtualServer) (*configs.VirtualServerEx, []virtualServerRouteError) {
	virtualServerEx := configs.VirtualServerEx{
		VirtualServer: virtualServer,
	}

	if virtualServer.Spec.TLS != nil && virtualServer.Spec.TLS.Secret != "" {
		secretKey := virtualServer.Namespace + "/" + virtualServer.Spec.TLS.Secret
		secret, err := lbc.getAndValidateSecret(secretKey)
		if err != nil {
			glog.Warningf("Error trying to get the secret %v for VirtualServer %v: %v", secretKey, virtualServer.Name, err)
		} else {
			virtualServerEx.TLSSecret = secret
		}
	}

	endpoints := make(map[string][]string)
	externalNameSvcs := make(map[string]bool)

	for _, u := range virtualServer.Spec.Upstreams {
		endpointsKey := configs.GenerateEndpointsKey(virtualServer.Namespace, u.Service, u.Subselector, u.Port)

		var endps []string
		var err error

		if len(u.Subselector) > 0 {
			endps, err = lbc.getEndpointsForSubselector(virtualServer.Namespace, u)
		} else {
			var external bool
			endps, external, err = lbc.getEndpointsForUpstream(virtualServer.Namespace, u.Service, u.Port)

			if err == nil && external && lbc.isNginxPlus {
				externalNameSvcs[configs.GenerateExternalNameSvcKey(virtualServer.Namespace, u.Service)] = true
			}
		}

		if err != nil {
			glog.Warningf("Error getting Endpoints for Upstream %v: %v", u.Name, err)
		}

		endpoints[endpointsKey] = endps
	}

	var virtualServerRoutes []*conf_v1.VirtualServerRoute
	var virtualServerRouteErrors []virtualServerRouteError

	// gather all referenced VirtualServerRoutes
	for _, r := range virtualServer.Spec.Routes {
		if r.Route == "" {
			continue
		}

		vsrKey := r.Route

		// if route is defined without a namespace, use the namespace of VirtualServer.
		if !strings.Contains(r.Route, "/") {
			vsrKey = fmt.Sprintf("%s/%s", virtualServer.Namespace, r.Route)
		}

		obj, exists, err := lbc.virtualServerRouteLister.GetByKey(vsrKey)
		if err != nil {
			glog.Warningf("Failed to get VirtualServerRoute %s for VirtualServer %s/%s: %v", vsrKey, virtualServer.Namespace, virtualServer.Name, err)
			virtualServerRouteErrors = append(virtualServerRouteErrors, newVirtualServerRouteErrorFromNsName(vsrKey, err))
			continue
		}

		if !exists {
			glog.Warningf("VirtualServer %s/%s references VirtualServerRoute %s that doesn't exist", virtualServer.Name, virtualServer.Namespace, vsrKey)
			virtualServerRouteErrors = append(virtualServerRouteErrors, newVirtualServerRouteErrorFromNsName(vsrKey, errors.New("VirtualServerRoute doesn't exist")))
			continue
		}

		vsr := obj.(*conf_v1.VirtualServerRoute)

		err = validation.ValidateVirtualServerRouteForVirtualServer(vsr, virtualServer.Spec.Host, r.Path, lbc.isNginxPlus)
		if err != nil {
			glog.Warningf("VirtualServer %s/%s references invalid VirtualServerRoute %s: %v", virtualServer.Name, virtualServer.Namespace, vsrKey, err)
			virtualServerRouteErrors = append(virtualServerRouteErrors, newVirtualServerRouteErrorFromVSR(vsr, err))
			continue
		}

		virtualServerRoutes = append(virtualServerRoutes, vsr)

		for _, u := range vsr.Spec.Upstreams {
			endpointsKey := configs.GenerateEndpointsKey(vsr.Namespace, u.Service, u.Subselector, u.Port)

			var endps []string
			var err error
			if len(u.Subselector) > 0 {
				endps, err = lbc.getEndpointsForSubselector(vsr.Namespace, u)
			} else {
				var external bool
				endps, external, err = lbc.getEndpointsForUpstream(vsr.Namespace, u.Service, u.Port)

				if err == nil && external && lbc.isNginxPlus {
					externalNameSvcs[configs.GenerateExternalNameSvcKey(vsr.Namespace, u.Service)] = true
				}
			}
			if err != nil {
				glog.Warningf("Error getting Endpoints for Upstream %v: %v", u.Name, err)
			}
			endpoints[endpointsKey] = endps
		}
	}

	virtualServerEx.Endpoints = endpoints
	virtualServerEx.VirtualServerRoutes = virtualServerRoutes
	virtualServerEx.ExternalNameSvcs = externalNameSvcs

	return &virtualServerEx, virtualServerRouteErrors
}

func (lbc *LoadBalancerController) createTransportServer(transportServer *conf_v1alpha1.TransportServer) *configs.TransportServerEx {
	endpoints := make(map[string][]string)

	for _, u := range transportServer.Spec.Upstreams {
		endps, external, err := lbc.getEndpointsForUpstream(transportServer.Namespace, u.Service, uint16(u.Port))
		if err != nil {
			glog.Warningf("Error getting Endpoints for Upstream %v: %v", u.Name, err)
		}

		if external {
			glog.Warningf("ExternalName services are not yet supported in TransportServer upstreams")
		}

		// subselector is not supported yet in TransportServer upstreams. That's why we pass "nil" here
		endpointsKey := configs.GenerateEndpointsKey(transportServer.Namespace, u.Service, nil, uint16(u.Port))

		endpoints[endpointsKey] = endps
	}

	return &configs.TransportServerEx{
		TransportServer: transportServer,
		Endpoints:       endpoints,
	}
}

func (lbc *LoadBalancerController) getEndpointsForUpstream(namespace string, upstreamService string, upstreamPort uint16) (endps []string, isExternal bool, err error) {
	svc, err := lbc.getServiceForUpstream(namespace, upstreamService, upstreamPort)
	if err != nil {
		return nil, false, fmt.Errorf("Error getting service %v: %v", upstreamService, err)
	}

	backend := &extensions.IngressBackend{
		ServiceName: upstreamService,
		ServicePort: intstr.FromInt(int(upstreamPort)),
	}

	endps, isExternal, err = lbc.getEndpointsForIngressBackend(backend, svc)
	if err != nil {
		return nil, false, fmt.Errorf("Error retrieving endpoints for the service %v: %v", upstreamService, err)
	}

	return endps, isExternal, err
}

func (lbc *LoadBalancerController) getEndpointsForSubselector(namespace string, upstream conf_v1.Upstream) (endps []string, err error) {
	svc, err := lbc.getServiceForUpstream(namespace, upstream.Service, upstream.Port)
	if err != nil {
		return nil, fmt.Errorf("Error getting service %v: %v", upstream.Service, err)
	}

	var targetPort int32

	for _, port := range svc.Spec.Ports {
		if port.Port == int32(upstream.Port) {
			targetPort, err = lbc.getTargetPort(&port, svc)
			if err != nil {
				return nil, fmt.Errorf("Error determining target port for port %v in service %v: %v", upstream.Port, svc.Name, err)
			}
			break
		}
	}

	if targetPort == 0 {
		return nil, fmt.Errorf("No port %v in service %s", upstream.Port, svc.Name)
	}

	endps, err = lbc.getEndpointsForServiceWithSubselector(targetPort, upstream.Subselector, svc)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving endpoints for the service %v: %v", upstream.Service, err)
	}

	return endps, err
}

func (lbc *LoadBalancerController) getEndpointsForServiceWithSubselector(targetPort int32, subselector map[string]string, svc *api_v1.Service) (endps []string, err error) {
	pods, err := lbc.podLister.ListByNamespace(svc.Namespace, labels.Merge(svc.Spec.Selector, subselector).AsSelector())
	if err != nil {
		return nil, fmt.Errorf("Error getting pods in namespace %v that match the selector %v: %v", svc.Namespace, labels.Merge(svc.Spec.Selector, subselector), err)
	}

	svcEps, err := lbc.endpointLister.GetServiceEndpoints(svc)
	if err != nil {
		glog.V(3).Infof("Error getting endpoints for service %s from the cache: %v", svc.Name, err)
		return nil, err
	}

	endps = getEndpointsBySubselectedPods(targetPort, pods, svcEps)
	return endps, nil
}

func getEndpointsBySubselectedPods(targetPort int32, pods []*api_v1.Pod, svcEps api_v1.Endpoints) (endps []string) {
	for _, pod := range pods {
		for _, subset := range svcEps.Subsets {
			for _, port := range subset.Ports {
				if port.Port != targetPort {
					continue
				}
				for _, address := range subset.Addresses {
					if address.IP == pod.Status.PodIP {
						podEndpoint := fmt.Sprintf("%v:%v", pod.Status.PodIP, targetPort)
						endps = append(endps, podEndpoint)
					}
				}
			}
		}
	}
	return endps
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
	pods, err := lbc.podLister.ListByNamespace(svc.Namespace, labels.Set(svc.Spec.Selector).AsSelector())
	if err != nil {
		glog.V(3).Infof("Error fetching pods for namespace %v: %v", svc.Namespace, err)
		return nil
	}
	return findProbeForPods(pods, svcPort)
}

func findProbeForPods(pods []*api_v1.Pod, svcPort *api_v1.ServicePort) *api_v1.Probe {
	if len(pods) > 0 {
		pod := pods[0]
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				if compareContainerPortAndServicePort(port, *svcPort) {
					// only http ReadinessProbes are useful for us
					if container.ReadinessProbe != nil && container.ReadinessProbe.Handler.HTTPGet != nil && container.ReadinessProbe.PeriodSeconds > 0 {
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

func (lbc *LoadBalancerController) getExternalEndpointsForIngressBackend(backend *extensions.IngressBackend, svc *api_v1.Service) []string {
	endpoint := fmt.Sprintf("%s:%d", svc.Spec.ExternalName, int32(backend.ServicePort.IntValue()))
	endpoints := []string{endpoint}
	return endpoints
}

func (lbc *LoadBalancerController) getEndpointsForIngressBackend(backend *extensions.IngressBackend, svc *api_v1.Service) (result []string, isExternal bool, err error) {
	endps, err := lbc.endpointLister.GetServiceEndpoints(svc)
	if err != nil {
		if svc.Spec.Type == api_v1.ServiceTypeExternalName {
			if !lbc.isNginxPlus {
				return nil, false, fmt.Errorf("Type ExternalName Services feature is only available in NGINX Plus")
			}
			result = lbc.getExternalEndpointsForIngressBackend(backend, svc)
			return result, true, nil
		}
		glog.V(3).Infof("Error getting endpoints for service %s from the cache: %v", svc.Name, err)
		return nil, false, err
	}

	result, err = lbc.getEndpointsForPort(endps, backend.ServicePort, svc)
	if err != nil {
		glog.V(3).Infof("Error getting endpoints for service %s port %v: %v", svc.Name, backend.ServicePort, err)
		return nil, false, err
	}
	return result, false, nil
}

func (lbc *LoadBalancerController) getEndpointsForPort(endps api_v1.Endpoints, ingSvcPort intstr.IntOrString, svc *api_v1.Service) ([]string, error) {
	var targetPort int32
	var err error

	for _, port := range svc.Spec.Ports {
		if (ingSvcPort.Type == intstr.Int && port.Port == int32(ingSvcPort.IntValue())) || (ingSvcPort.Type == intstr.String && port.Name == ingSvcPort.String()) {
			targetPort, err = lbc.getTargetPort(&port, svc)
			if err != nil {
				return nil, fmt.Errorf("Error determining target port for port %v in Ingress: %v", ingSvcPort, err)
			}
			break
		}
	}

	if targetPort == 0 {
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

	pods, err := lbc.podLister.ListByNamespace(svc.Namespace, labels.Set(svc.Spec.Selector).AsSelector())
	if err != nil {
		return 0, fmt.Errorf("Error getting pod information: %v", err)
	}

	if len(pods) == 0 {
		return 0, fmt.Errorf("No pods of service %s", svc.Name)
	}

	pod := pods[0]

	portNum, err := findPort(pod, svcPort)
	if err != nil {
		return 0, fmt.Errorf("Error finding named port %v in pod %s: %v", svcPort, pod.Name, err)
	}

	return portNum, nil
}

func (lbc *LoadBalancerController) getServiceForUpstream(namespace string, upstreamService string, upstreamPort uint16) (*api_v1.Service, error) {
	backend := &extensions.IngressBackend{
		ServiceName: upstreamService,
		ServicePort: intstr.FromInt(int(upstreamPort)),
	}
	return lbc.getServiceForIngressBackend(backend, namespace)
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
	if healthCheckEnabled, exists, err := configs.GetMapKeyAsBool(ing.Annotations, "nginx.com/health-checks", ing); exists {
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
	err1 := ValidateTLSSecret(secret)
	if !lbc.isNginxPlus {
		return err1
	}

	err2 := ValidateJWKSecret(secret)

	if err1 == nil || err2 == nil {
		return nil
	}

	return fmt.Errorf("Secret is not a TLS or JWK secret")
}

// getMinionsForHost returns a list of all minion ingress resources for a given master
func (lbc *LoadBalancerController) getMinionsForMaster(master *configs.IngressEx) ([]*configs.IngressEx, error) {
	ings, err := lbc.ingressLister.List()
	if err != nil {
		return []*configs.IngressEx{}, err
	}

	// ingresses are sorted by creation time
	sort.Slice(ings.Items[:], func(i, j int) bool {
		return ings.Items[i].CreationTimestamp.Time.UnixNano() < ings.Items[j].CreationTimestamp.Time.UnixNano()
	})

	var minions []*configs.IngressEx
	var minionPaths = make(map[string]*extensions.Ingress)

	for i := range ings.Items {
		if !lbc.IsNginxIngress(&ings.Items[i]) {
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

func (lbc *LoadBalancerController) createMergableIngresses(master *extensions.Ingress) (*configs.MergeableIngresses, error) {
	mergeableIngresses := configs.MergeableIngresses{}

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

func formatWarningMessages(w []string) string {
	return strings.Join(w, "; ")
}

func (lbc *LoadBalancerController) syncSVIDRotation(svidResponse *workload.X509SVIDs) {
	lbc.syncLock.Lock()
	defer lbc.syncLock.Unlock()
	glog.V(3).Info("Rotating SPIFFE Certificates")
	err := lbc.configurator.AddOrUpdateSpiffeCerts(svidResponse)
	if err != nil {
		glog.Errorf("failed to rotate SPIFFE certificates: %v", err)
	}
}

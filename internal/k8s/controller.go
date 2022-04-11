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
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nginxinc/kubernetes-ingress/pkg/apis/dos/v1beta1"

	"github.com/nginxinc/kubernetes-ingress/internal/k8s/appprotect"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/appprotectcommon"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/appprotectdos"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	"github.com/spiffe/go-spiffe/workload"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"

	cm_controller "github.com/nginxinc/kubernetes-ingress/internal/certmanager"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"

	api_v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/validation"
	k8s_nginx "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	k8s_nginx_informers "github.com/nginxinc/kubernetes-ingress/pkg/client/informers/externalversions"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
)

const (
	ingressClassKey = "kubernetes.io/ingress.class"
	// IngressControllerName holds Ingress Controller name
	IngressControllerName = "nginx.org/ingress-controller"
)

var (
	ingressLinkGVR = schema.GroupVersionResource{
		Group:    "cis.f5.com",
		Version:  "v1",
		Resource: "ingresslinks",
	}
	ingressLinkGVK = schema.GroupVersionKind{
		Group:   "cis.f5.com",
		Version: "v1",
		Kind:    "IngressLink",
	}
)

type podEndpoint struct {
	Address string
	PodName string
	// MeshPodOwner is used for NGINX Service Mesh metrics
	configs.MeshPodOwner
}

// LoadBalancerController watches Kubernetes API and
// reconfigures NGINX via NginxController when needed
type LoadBalancerController struct {
	client                        kubernetes.Interface
	confClient                    k8s_nginx.Interface
	dynClient                     dynamic.Interface
	restConfig                    *rest.Config
	cacheSyncs                    []cache.InformerSynced
	sharedInformerFactory         informers.SharedInformerFactory
	confSharedInformerFactorry    k8s_nginx_informers.SharedInformerFactory
	configMapController           cache.Controller
	dynInformerFactory            dynamicinformer.DynamicSharedInformerFactory
	globalConfigurationController cache.Controller
	ingressLinkInformer           cache.SharedIndexInformer
	ingressLister                 storeToIngressLister
	svcLister                     cache.Store
	endpointLister                storeToEndpointLister
	configMapLister               storeToConfigMapLister
	podLister                     indexerToPodLister
	secretLister                  cache.Store
	virtualServerLister           cache.Store
	virtualServerRouteLister      cache.Store
	appProtectPolicyLister        cache.Store
	appProtectLogConfLister       cache.Store
	appProtectDosPolicyLister     cache.Store
	appProtectDosLogConfLister    cache.Store
	appProtectDosProtectedLister  cache.Store
	globalConfigurationLister     cache.Store
	appProtectUserSigLister       cache.Store
	transportServerLister         cache.Store
	policyLister                  cache.Store
	ingressLinkLister             cache.Store
	syncQueue                     *taskQueue
	ctx                           context.Context
	cancel                        context.CancelFunc
	configurator                  *configs.Configurator
	watchNginxConfigMaps          bool
	watchGlobalConfiguration      bool
	watchIngressLink              bool
	isNginxPlus                   bool
	appProtectEnabled             bool
	appProtectDosEnabled          bool
	recorder                      record.EventRecorder
	defaultServerSecret           string
	ingressClass                  string
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
	enableOIDC                    bool
	metricsCollector              collectors.ControllerCollector
	globalConfigurationValidator  *validation.GlobalConfigurationValidator
	transportServerValidator      *validation.TransportServerValidator
	spiffeController              *SpiffeController
	internalRoutesEnabled         bool
	syncLock                      sync.Mutex
	isNginxReady                  bool
	isPrometheusEnabled           bool
	isLatencyMetricsEnabled       bool
	configuration                 *Configuration
	secretStore                   secrets.SecretStore
	appProtectConfiguration       appprotect.Configuration
	dosConfiguration              *appprotectdos.Configuration
	configMap                     *api_v1.ConfigMap
	certManagerController         *cm_controller.CmController
}

var keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

// NewLoadBalancerControllerInput holds the input needed to call NewLoadBalancerController.
type NewLoadBalancerControllerInput struct {
	KubeClient                   kubernetes.Interface
	ConfClient                   k8s_nginx.Interface
	DynClient                    dynamic.Interface
	RestConfig                   *rest.Config
	ResyncPeriod                 time.Duration
	Namespace                    string
	NginxConfigurator            *configs.Configurator
	DefaultServerSecret          string
	AppProtectEnabled            bool
	AppProtectDosEnabled         bool
	IsNginxPlus                  bool
	IngressClass                 string
	ExternalServiceName          string
	IngressLink                  string
	ControllerNamespace          string
	ReportIngressStatus          bool
	IsLeaderElectionEnabled      bool
	LeaderElectionLockName       string
	WildcardTLSSecret            string
	ConfigMaps                   string
	GlobalConfiguration          string
	AreCustomResourcesEnabled    bool
	EnableOIDC                   bool
	MetricsCollector             collectors.ControllerCollector
	GlobalConfigurationValidator *validation.GlobalConfigurationValidator
	TransportServerValidator     *validation.TransportServerValidator
	VirtualServerValidator       *validation.VirtualServerValidator
	SpireAgentAddress            string
	InternalRoutesEnabled        bool
	IsPrometheusEnabled          bool
	IsLatencyMetricsEnabled      bool
	IsTLSPassthroughEnabled      bool
	SnippetsEnabled              bool
	CertManagerEnabled           bool
}

// NewLoadBalancerController creates a controller
func NewLoadBalancerController(input NewLoadBalancerControllerInput) *LoadBalancerController {
	lbc := &LoadBalancerController{
		client:                       input.KubeClient,
		confClient:                   input.ConfClient,
		dynClient:                    input.DynClient,
		restConfig:                   input.RestConfig,
		configurator:                 input.NginxConfigurator,
		defaultServerSecret:          input.DefaultServerSecret,
		appProtectEnabled:            input.AppProtectEnabled,
		appProtectDosEnabled:         input.AppProtectDosEnabled,
		isNginxPlus:                  input.IsNginxPlus,
		ingressClass:                 input.IngressClass,
		reportIngressStatus:          input.ReportIngressStatus,
		isLeaderElectionEnabled:      input.IsLeaderElectionEnabled,
		leaderElectionLockName:       input.LeaderElectionLockName,
		resync:                       input.ResyncPeriod,
		namespace:                    input.Namespace,
		controllerNamespace:          input.ControllerNamespace,
		wildcardTLSSecret:            input.WildcardTLSSecret,
		areCustomResourcesEnabled:    input.AreCustomResourcesEnabled,
		enableOIDC:                   input.EnableOIDC,
		metricsCollector:             input.MetricsCollector,
		globalConfigurationValidator: input.GlobalConfigurationValidator,
		transportServerValidator:     input.TransportServerValidator,
		internalRoutesEnabled:        input.InternalRoutesEnabled,
		isPrometheusEnabled:          input.IsPrometheusEnabled,
		isLatencyMetricsEnabled:      input.IsLatencyMetricsEnabled,
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

	if input.CertManagerEnabled {
		lbc.certManagerController = cm_controller.NewCmController(cm_controller.BuildOpts(context.TODO(), lbc.restConfig, lbc.client, lbc.namespace, lbc.recorder, lbc.confClient))
	}

	glog.V(3).Infof("Nginx Ingress Controller has class: %v", input.IngressClass)

	lbc.sharedInformerFactory = informers.NewSharedInformerFactoryWithOptions(lbc.client, input.ResyncPeriod, informers.WithNamespace(lbc.namespace))

	// create handlers for resources we care about
	lbc.addSecretHandler(createSecretHandlers(lbc))
	lbc.addIngressHandler(createIngressHandlers(lbc))
	lbc.addServiceHandler(createServiceHandlers(lbc))
	lbc.addEndpointHandler(createEndpointHandlers(lbc))
	lbc.addPodHandler()

	if lbc.areCustomResourcesEnabled {
		lbc.confSharedInformerFactorry = k8s_nginx_informers.NewSharedInformerFactoryWithOptions(lbc.confClient, input.ResyncPeriod, k8s_nginx_informers.WithNamespace(lbc.namespace))

		lbc.addVirtualServerHandler(createVirtualServerHandlers(lbc))
		lbc.addVirtualServerRouteHandler(createVirtualServerRouteHandlers(lbc))
		lbc.addTransportServerHandler(createTransportServerHandlers(lbc))
		lbc.addPolicyHandler(createPolicyHandlers(lbc))

		if input.GlobalConfiguration != "" {
			lbc.watchGlobalConfiguration = true
			ns, name, _ := ParseNamespaceName(input.GlobalConfiguration)
			lbc.addGlobalConfigurationHandler(createGlobalConfigurationHandlers(lbc), ns, name)
		}
	}

	if lbc.appProtectEnabled || lbc.appProtectDosEnabled {
		lbc.dynInformerFactory = dynamicinformer.NewFilteredDynamicSharedInformerFactory(lbc.dynClient, 0, lbc.namespace, nil)

		if lbc.appProtectEnabled {
			lbc.addAppProtectPolicyHandler(createAppProtectPolicyHandlers(lbc))
			lbc.addAppProtectLogConfHandler(createAppProtectLogConfHandlers(lbc))
			lbc.addAppProtectUserSigHandler(createAppProtectUserSigHandlers(lbc))
		}

		if lbc.appProtectDosEnabled {
			lbc.addAppProtectDosPolicyHandler(createAppProtectDosPolicyHandlers(lbc))
			lbc.addAppProtectDosLogConfHandler(createAppProtectDosLogConfHandlers(lbc))
			lbc.addAppProtectDosProtectedResourceHandler(createAppProtectDosProtectedResourceHandlers(lbc))
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

	if input.IngressLink != "" {
		lbc.watchIngressLink = true
		lbc.addIngressLinkHandler(createIngressLinkHandlers(lbc), input.IngressLink)
	}

	if input.IsLeaderElectionEnabled {
		lbc.addLeaderHandler(createLeaderHandler(lbc))
	}

	lbc.statusUpdater = &statusUpdater{
		client:                   input.KubeClient,
		namespace:                input.ControllerNamespace,
		externalServiceName:      input.ExternalServiceName,
		ingressLister:            &lbc.ingressLister,
		virtualServerLister:      lbc.virtualServerLister,
		virtualServerRouteLister: lbc.virtualServerRouteLister,
		transportServerLister:    lbc.transportServerLister,
		policyLister:             lbc.policyLister,
		keyFunc:                  keyFunc,
		confClient:               input.ConfClient,
		hasCorrectIngressClass:   lbc.HasCorrectIngressClass,
	}

	lbc.configuration = NewConfiguration(
		lbc.HasCorrectIngressClass,
		input.IsNginxPlus,
		input.AppProtectEnabled,
		input.AppProtectDosEnabled,
		input.InternalRoutesEnabled,
		input.VirtualServerValidator,
		input.GlobalConfigurationValidator,
		input.TransportServerValidator,
		input.IsTLSPassthroughEnabled,
		input.SnippetsEnabled)

	lbc.appProtectConfiguration = appprotect.NewConfiguration()
	lbc.dosConfiguration = appprotectdos.NewConfiguration(input.AppProtectDosEnabled)

	lbc.secretStore = secrets.NewLocalSecretStore(lbc.configurator)

	return lbc
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

// addAppProtectPolicyHandler creates dynamic informers for custom appprotect policy resource
func (lbc *LoadBalancerController) addAppProtectPolicyHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.dynInformerFactory.ForResource(appprotect.PolicyGVR).Informer()
	informer.AddEventHandler(handlers)
	lbc.appProtectPolicyLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addAppProtectLogConfHandler creates dynamic informer for custom appprotect logging config resource
func (lbc *LoadBalancerController) addAppProtectLogConfHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.dynInformerFactory.ForResource(appprotect.LogConfGVR).Informer()
	informer.AddEventHandler(handlers)
	lbc.appProtectLogConfLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addAppProtectUserSigHandler creates dynamic informer for custom appprotect user defined signature resource
func (lbc *LoadBalancerController) addAppProtectUserSigHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.dynInformerFactory.ForResource(appprotect.UserSigGVR).Informer()
	informer.AddEventHandler(handlers)
	lbc.appProtectUserSigLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addAppProtectDosPolicyHandler creates dynamic informers for custom appprotectdos policy resource
func (lbc *LoadBalancerController) addAppProtectDosPolicyHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.dynInformerFactory.ForResource(appprotectdos.DosPolicyGVR).Informer()
	informer.AddEventHandler(handlers)
	lbc.appProtectDosPolicyLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addAppProtectDosLogConfHandler creates dynamic informer for custom appprotectdos logging config resource
func (lbc *LoadBalancerController) addAppProtectDosLogConfHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.dynInformerFactory.ForResource(appprotectdos.DosLogConfGVR).Informer()
	informer.AddEventHandler(handlers)
	lbc.appProtectDosLogConfLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addAppProtectDosLogConfHandler creates dynamic informer for custom appprotectdos logging config resource
func (lbc *LoadBalancerController) addAppProtectDosProtectedResourceHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.confSharedInformerFactorry.Appprotectdos().V1beta1().DosProtectedResources().Informer()
	informer.AddEventHandler(handlers)
	lbc.appProtectDosProtectedLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addSecretHandler adds the handler for secrets to the controller
func (lbc *LoadBalancerController) addSecretHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.sharedInformerFactory.Core().V1().Secrets().Informer()
	informer.AddEventHandler(handlers)
	lbc.secretLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addServiceHandler adds the handler for services to the controller
func (lbc *LoadBalancerController) addServiceHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.sharedInformerFactory.Core().V1().Services().Informer()
	informer.AddEventHandler(handlers)
	lbc.svcLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addIngressHandler adds the handler for ingresses to the controller
func (lbc *LoadBalancerController) addIngressHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.sharedInformerFactory.Networking().V1().Ingresses().Informer()
	informer.AddEventHandler(handlers)
	lbc.ingressLister.Store = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addEndpointHandler adds the handler for endpoints to the controller
func (lbc *LoadBalancerController) addEndpointHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.sharedInformerFactory.Core().V1().Endpoints().Informer()
	informer.AddEventHandler(handlers)
	lbc.endpointLister.Store = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
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
	lbc.cacheSyncs = append(lbc.cacheSyncs, lbc.configMapController.HasSynced)
}

func (lbc *LoadBalancerController) addPodHandler() {
	informer := lbc.sharedInformerFactory.Core().V1().Pods().Informer()
	lbc.podLister.Indexer = informer.GetIndexer()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func (lbc *LoadBalancerController) addVirtualServerHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.confSharedInformerFactorry.K8s().V1().VirtualServers().Informer()
	informer.AddEventHandler(handlers)
	lbc.virtualServerLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func (lbc *LoadBalancerController) addVirtualServerRouteHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.confSharedInformerFactorry.K8s().V1().VirtualServerRoutes().Informer()
	informer.AddEventHandler(handlers)
	lbc.virtualServerRouteLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func (lbc *LoadBalancerController) addPolicyHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.confSharedInformerFactorry.K8s().V1().Policies().Informer()
	informer.AddEventHandler(handlers)
	lbc.policyLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func (lbc *LoadBalancerController) addGlobalConfigurationHandler(handlers cache.ResourceEventHandlerFuncs, namespace string, name string) {
	lbc.globalConfigurationLister, lbc.globalConfigurationController = cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.confClient.K8sV1alpha1().RESTClient(),
			"globalconfigurations",
			namespace,
			fields.Set{"metadata.name": name}.AsSelector()),
		&conf_v1alpha1.GlobalConfiguration{},
		lbc.resync,
		handlers,
	)
	lbc.cacheSyncs = append(lbc.cacheSyncs, lbc.globalConfigurationController.HasSynced)
}

func (lbc *LoadBalancerController) addTransportServerHandler(handlers cache.ResourceEventHandlerFuncs) {
	informer := lbc.confSharedInformerFactorry.K8s().V1alpha1().TransportServers().Informer()
	informer.AddEventHandler(handlers)
	lbc.transportServerLister = informer.GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func (lbc *LoadBalancerController) addIngressLinkHandler(handlers cache.ResourceEventHandlerFuncs, name string) {
	optionsModifier := func(options *meta_v1.ListOptions) {
		options.FieldSelector = fields.Set{"metadata.name": name}.String()
	}

	informer := dynamicinformer.NewFilteredDynamicInformer(lbc.dynClient, ingressLinkGVR, lbc.controllerNamespace, lbc.resync,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, optionsModifier)

	informer.Informer().AddEventHandlerWithResyncPeriod(handlers, lbc.resync)

	lbc.ingressLinkInformer = informer.Informer()
	lbc.ingressLinkLister = informer.Informer().GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, lbc.ingressLinkInformer.HasSynced)
}

// Run starts the loadbalancer controller
func (lbc *LoadBalancerController) Run() {
	lbc.ctx, lbc.cancel = context.WithCancel(context.Background())

	if lbc.spiffeController != nil {
		err := lbc.spiffeController.Start(lbc.ctx.Done(), lbc.addInternalRouteServer)
		if err != nil {
			glog.Fatal(err)
		}
	}
	if lbc.certManagerController != nil {
		go lbc.certManagerController.Run(lbc.ctx.Done())
	}
	if lbc.leaderElector != nil {
		go lbc.leaderElector.Run(lbc.ctx)
	}

	go lbc.sharedInformerFactory.Start(lbc.ctx.Done())
	if lbc.watchNginxConfigMaps {
		go lbc.configMapController.Run(lbc.ctx.Done())
	}
	if lbc.areCustomResourcesEnabled {
		go lbc.confSharedInformerFactorry.Start(lbc.ctx.Done())
	}
	if lbc.watchGlobalConfiguration {
		go lbc.globalConfigurationController.Run(lbc.ctx.Done())
	}
	if lbc.watchIngressLink {
		go lbc.ingressLinkInformer.Run(lbc.ctx.Done())
	}
	if lbc.appProtectEnabled || lbc.appProtectDosEnabled {
		go lbc.dynInformerFactory.Start(lbc.ctx.Done())
	}

	glog.V(3).Infof("Waiting for %d caches to sync", len(lbc.cacheSyncs))

	if !cache.WaitForCacheSync(lbc.ctx.Done(), lbc.cacheSyncs...) {
		return
	}

	lbc.preSyncSecrets()

	glog.V(3).Infof("Starting the queue with %d initial elements", lbc.syncQueue.Len())

	go lbc.syncQueue.Run(time.Second, lbc.ctx.Done())
	<-lbc.ctx.Done()
}

// Stop shutdowns the load balancer controller
func (lbc *LoadBalancerController) Stop() {
	lbc.cancel()

	lbc.syncQueue.Shutdown()
}

func (lbc *LoadBalancerController) syncEndpoints(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing endpoints %v", key)

	obj, endpExists, err := lbc.endpointLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if !endpExists {
		return
	}

	endp := obj.(*api_v1.Endpoints)
	resources := lbc.configuration.FindResourcesForEndpoints(endp.Namespace, endp.Name)

	resourceExes := lbc.createExtendedResources(resources)

	if len(resourceExes.IngressExes) > 0 {
		glog.V(3).Infof("Updating Endpoints for %v", resourceExes.IngressExes)
		err = lbc.configurator.UpdateEndpoints(resourceExes.IngressExes)
		if err != nil {
			glog.Errorf("Error updating endpoints for %v: %v", resourceExes.IngressExes, err)
		}
	}

	if len(resourceExes.MergeableIngresses) > 0 {
		glog.V(3).Infof("Updating Endpoints for %v", resourceExes.MergeableIngresses)
		err = lbc.configurator.UpdateEndpointsMergeableIngress(resourceExes.MergeableIngresses)
		if err != nil {
			glog.Errorf("Error updating endpoints for %v: %v", resourceExes.MergeableIngresses, err)
		}
	}

	if lbc.areCustomResourcesEnabled {
		if len(resourceExes.VirtualServerExes) > 0 {
			glog.V(3).Infof("Updating endpoints for %v", resourceExes.VirtualServerExes)
			err := lbc.configurator.UpdateEndpointsForVirtualServers(resourceExes.VirtualServerExes)
			if err != nil {
				glog.Errorf("Error updating endpoints for %v: %v", resourceExes.VirtualServerExes, err)
			}
		}

		if len(resourceExes.TransportServerExes) > 0 {
			glog.V(3).Infof("Updating endpoints for %v", resourceExes.TransportServerExes)
			err := lbc.configurator.UpdateEndpointsForTransportServers(resourceExes.TransportServerExes)
			if err != nil {
				glog.Errorf("Error updating endpoints for %v: %v", resourceExes.TransportServerExes, err)
			}
		}
	}
}

func (lbc *LoadBalancerController) createExtendedResources(resources []Resource) configs.ExtendedResources {
	var result configs.ExtendedResources

	for _, r := range resources {
		switch impl := r.(type) {
		case *VirtualServerConfiguration:
			vs := impl.VirtualServer
			vsEx := lbc.createVirtualServerEx(vs, impl.VirtualServerRoutes)
			result.VirtualServerExes = append(result.VirtualServerExes, vsEx)
		case *IngressConfiguration:

			if impl.IsMaster {
				mergeableIng := lbc.createMergeableIngresses(impl)
				result.MergeableIngresses = append(result.MergeableIngresses, mergeableIng)
			} else {
				ingEx := lbc.createIngressEx(impl.Ingress, impl.ValidHosts, nil)
				result.IngressExes = append(result.IngressExes, ingEx)
			}
		case *TransportServerConfiguration:
			tsEx := lbc.createTransportServerEx(impl.TransportServer, impl.ListenerPort)
			result.TransportServerExes = append(result.TransportServerExes, tsEx)
		}
	}

	return result
}

func (lbc *LoadBalancerController) syncConfigMap(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing configmap %v", key)

	obj, configExists, err := lbc.configMapLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}
	if configExists {
		lbc.configMap = obj.(*api_v1.ConfigMap)
		lbc.statusUpdater.SaveStatusFromExternalStatus(lbc.configMap.Data["external-status-address"])
	} else {
		lbc.configMap = nil
	}

	if !lbc.isNginxReady {
		glog.V(3).Infof("Skipping ConfigMap update because the pod is not ready yet")
		return
	}

	lbc.updateAllConfigs()
}

func (lbc *LoadBalancerController) updateAllConfigs() {
	cfgParams := configs.NewDefaultConfigParams(lbc.isNginxPlus)

	if lbc.configMap != nil {
		cfgParams = configs.ParseConfigMap(lbc.configMap, lbc.isNginxPlus, lbc.appProtectEnabled, lbc.appProtectDosEnabled)
	}

	resources := lbc.configuration.GetResources()

	glog.V(3).Infof("Updating %v resources", len(resources))

	resourceExes := lbc.createExtendedResources(resources)

	warnings, updateErr := lbc.configurator.UpdateConfig(cfgParams, resourceExes)

	eventTitle := "Updated"
	eventType := api_v1.EventTypeNormal
	eventWarningMessage := ""

	if updateErr != nil {
		eventTitle = "UpdatedWithError"
		eventType = api_v1.EventTypeWarning
		eventWarningMessage = fmt.Sprintf("but was not applied: %v", updateErr)
	}

	if len(warnings) > 0 && updateErr == nil {
		eventWarningMessage = "with warnings. Please check the logs"
	}

	if lbc.configMap != nil {
		key := getResourceKey(&lbc.configMap.ObjectMeta)
		lbc.recorder.Eventf(lbc.configMap, eventType, eventTitle, "Configuration from %v was updated %s", key, eventWarningMessage)
	}

	gc := lbc.configuration.GetGlobalConfiguration()
	if gc != nil {
		key := getResourceKey(&lbc.configMap.ObjectMeta)
		lbc.recorder.Eventf(gc, eventType, eventTitle, fmt.Sprintf("GlobalConfiguration %s was updated %s", key, eventWarningMessage))
	}

	lbc.updateResourcesStatusAndEvents(resources, warnings, updateErr)
}

// preSyncSecrets adds Secret resources to the SecretStore.
// It must be called after the caches are synced but before the queue starts processing elements.
// If we don't add Secrets, there is a chance that during the IC start
// syncing an Ingress or other resource that references a Secret will happen before that Secret was synced.
// As a result, the IC will generate configuration for that resource assuming that the Secret is missing and
// it will report warnings. (See https://github.com/nginxinc/kubernetes-ingress/issues/1448 )
func (lbc *LoadBalancerController) preSyncSecrets() {
	objects := lbc.secretLister.List()
	glog.V(3).Infof("PreSync %d Secrets", len(objects))

	for _, obj := range objects {
		secret := obj.(*api_v1.Secret)

		if !secrets.IsSupportedSecretType(secret.Type) {
			glog.V(3).Infof("Ignoring Secret %s/%s of unsupported type %s", secret.Namespace, secret.Name, secret.Type)
			continue
		}

		glog.V(3).Infof("Adding Secret: %s/%s", secret.Namespace, secret.Name)
		lbc.secretStore.AddOrUpdateSecret(secret)
	}
}

func (lbc *LoadBalancerController) sync(task task) {
	glog.V(3).Infof("Syncing %v", task.Key)
	if lbc.spiffeController != nil {
		lbc.syncLock.Lock()
		defer lbc.syncLock.Unlock()
	}
	switch task.Kind {
	case ingress:
		lbc.syncIngress(task)
		lbc.updateIngressMetrics()
		lbc.updateTransportServerMetrics()
	case configMap:
		lbc.syncConfigMap(task)
	case endpoints:
		lbc.syncEndpoints(task)
	case secret:
		lbc.syncSecret(task)
	case service:
		lbc.syncService(task)
	case virtualserver:
		lbc.syncVirtualServer(task)
		lbc.updateVirtualServerMetrics()
		lbc.updateTransportServerMetrics()
	case virtualServerRoute:
		lbc.syncVirtualServerRoute(task)
		lbc.updateVirtualServerMetrics()
	case globalConfiguration:
		lbc.syncGlobalConfiguration(task)
		lbc.updateTransportServerMetrics()
	case transportserver:
		lbc.syncTransportServer(task)
		lbc.updateTransportServerMetrics()
	case policy:
		lbc.syncPolicy(task)
	case appProtectPolicy:
		lbc.syncAppProtectPolicy(task)
	case appProtectLogConf:
		lbc.syncAppProtectLogConf(task)
	case appProtectUserSig:
		lbc.syncAppProtectUserSig(task)
	case appProtectDosPolicy:
		lbc.syncAppProtectDosPolicy(task)
	case appProtectDosLogConf:
		lbc.syncAppProtectDosLogConf(task)
	case appProtectDosProtectedResource:
		lbc.syncDosProtectedResource(task)
	case ingressLink:
		lbc.syncIngressLink(task)
	}

	if !lbc.isNginxReady && lbc.syncQueue.Len() == 0 {
		lbc.configurator.EnableReloads()
		lbc.updateAllConfigs()

		lbc.isNginxReady = true
		glog.V(3).Infof("NGINX is ready")
	}
}

func (lbc *LoadBalancerController) syncIngressLink(task task) {
	key := task.Key
	glog.V(2).Infof("Adding, Updating or Deleting IngressLink: %v", key)

	obj, exists, err := lbc.ingressLinkLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if !exists {
		// IngressLink got removed
		lbc.statusUpdater.ClearStatusFromIngressLink()
	} else {
		// IngressLink is added or updated
		link := obj.(*unstructured.Unstructured)

		// spec.virtualServerAddress contains the IP of the BIG-IP device
		ip, found, err := unstructured.NestedString(link.Object, "spec", "virtualServerAddress")
		if err != nil {
			glog.Errorf("Failed to get virtualServerAddress from IngressLink %s: %v", key, err)
			lbc.statusUpdater.ClearStatusFromIngressLink()
		} else if !found {
			glog.Errorf("virtualServerAddress is not found in IngressLink %s", key)
			lbc.statusUpdater.ClearStatusFromIngressLink()
		} else if ip == "" {
			glog.Warningf("IngressLink %s has the empty virtualServerAddress field", key)
			lbc.statusUpdater.ClearStatusFromIngressLink()
		} else {
			lbc.statusUpdater.SaveStatusFromIngressLink(ip)
		}
	}

	if lbc.reportStatusEnabled() {
		ingresses := lbc.configuration.GetResourcesWithFilter(resourceFilter{Ingresses: true})

		glog.V(3).Infof("Updating status for %v Ingresses", len(ingresses))

		err := lbc.statusUpdater.UpdateExternalEndpointsForResources(ingresses)
		if err != nil {
			glog.Errorf("Error updating ingress status in syncIngressLink: %v", err)
		}
	}

	if lbc.areCustomResourcesEnabled && lbc.reportCustomResourceStatusEnabled() {
		virtualServers := lbc.configuration.GetResourcesWithFilter(resourceFilter{VirtualServers: true})

		glog.V(3).Infof("Updating status for %v VirtualServers", len(virtualServers))

		err := lbc.statusUpdater.UpdateExternalEndpointsForResources(virtualServers)
		if err != nil {
			glog.V(3).Infof("Error updating VirtualServer/VirtualServerRoute status in syncIngressLink: %v", err)
		}
	}
}

func (lbc *LoadBalancerController) syncPolicy(task task) {
	key := task.Key
	obj, polExists, err := lbc.policyLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	glog.V(2).Infof("Adding, Updating or Deleting Policy: %v\n", key)

	if polExists && lbc.HasCorrectIngressClass(obj) {
		pol := obj.(*conf_v1.Policy)
		err := validation.ValidatePolicy(pol, lbc.isNginxPlus, lbc.enableOIDC, lbc.appProtectEnabled)
		if err != nil {
			msg := fmt.Sprintf("Policy %v/%v is invalid and was rejected: %v", pol.Namespace, pol.Name, err)
			lbc.recorder.Eventf(pol, api_v1.EventTypeWarning, "Rejected", msg)

			if lbc.reportCustomResourceStatusEnabled() {
				err = lbc.statusUpdater.UpdatePolicyStatus(pol, conf_v1.StateInvalid, "Rejected", msg)
				if err != nil {
					glog.V(3).Infof("Failed to update policy %s status: %v", key, err)
				}
			}
		} else {
			msg := fmt.Sprintf("Policy %v/%v was added or updated", pol.Namespace, pol.Name)
			lbc.recorder.Eventf(pol, api_v1.EventTypeNormal, "AddedOrUpdated", msg)

			if lbc.reportCustomResourceStatusEnabled() {
				err = lbc.statusUpdater.UpdatePolicyStatus(pol, conf_v1.StateValid, "AddedOrUpdated", msg)
				if err != nil {
					glog.V(3).Infof("Failed to update policy %s status: %v", key, err)
				}
			}
		}
	}

	// it is safe to ignore the error
	namespace, name, _ := ParseNamespaceName(key)

	resources := lbc.configuration.FindResourcesForPolicy(namespace, name)
	resourceExes := lbc.createExtendedResources(resources)

	// Only VirtualServers support policies
	if len(resourceExes.VirtualServerExes) == 0 {
		return
	}

	warnings, updateErr := lbc.configurator.AddOrUpdateVirtualServers(resourceExes.VirtualServerExes)
	lbc.updateResourcesStatusAndEvents(resources, warnings, updateErr)

	// Note: updating the status of a policy based on a reload is not needed.
}

func (lbc *LoadBalancerController) syncTransportServer(task task) {
	key := task.Key
	obj, tsExists, err := lbc.transportServerLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []ResourceChange
	var problems []ConfigurationProblem

	if !tsExists {
		glog.V(2).Infof("Deleting TransportServer: %v\n", key)
		changes, problems = lbc.configuration.DeleteTransportServer(key)
	} else {
		glog.V(2).Infof("Adding or Updating TransportServer: %v\n", key)
		ts := obj.(*conf_v1alpha1.TransportServer)
		changes, problems = lbc.configuration.AddOrUpdateTransportServer(ts)
	}

	lbc.processChanges(changes)
	lbc.processProblems(problems)
}

func (lbc *LoadBalancerController) syncGlobalConfiguration(task task) {
	key := task.Key
	obj, gcExists, err := lbc.globalConfigurationLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []ResourceChange
	var problems []ConfigurationProblem
	var validationErr error

	if !gcExists {
		glog.V(2).Infof("Deleting GlobalConfiguration: %v\n", key)

		changes, problems = lbc.configuration.DeleteGlobalConfiguration()
	} else {
		glog.V(2).Infof("Adding or Updating GlobalConfiguration: %v\n", key)

		gc := obj.(*conf_v1alpha1.GlobalConfiguration)
		changes, problems, validationErr = lbc.configuration.AddOrUpdateGlobalConfiguration(gc)
	}

	updateErr := lbc.processChangesFromGlobalConfiguration(changes)

	if gcExists {
		eventTitle := "Updated"
		eventType := api_v1.EventTypeNormal
		eventMessage := fmt.Sprintf("GlobalConfiguration %s was added or updated", key)

		if validationErr != nil {
			eventTitle = "Rejected"
			eventType = api_v1.EventTypeWarning
			eventMessage = fmt.Sprintf("GlobalConfiguration %s is invalid and was rejected: %v", key, validationErr)
		}

		if updateErr != nil {
			eventTitle += "WithError"
			eventType = api_v1.EventTypeWarning
			eventMessage = fmt.Sprintf("%s; with reload error: %v", eventMessage, updateErr)
		}

		gc := obj.(*conf_v1alpha1.GlobalConfiguration)
		lbc.recorder.Eventf(gc, eventType, eventTitle, eventMessage)
	}

	lbc.processProblems(problems)
}

func (lbc *LoadBalancerController) syncVirtualServer(task task) {
	key := task.Key
	obj, vsExists, err := lbc.virtualServerLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []ResourceChange
	var problems []ConfigurationProblem

	if !vsExists {
		glog.V(2).Infof("Deleting VirtualServer: %v\n", key)

		changes, problems = lbc.configuration.DeleteVirtualServer(key)
	} else {
		glog.V(2).Infof("Adding or Updating VirtualServer: %v\n", key)

		vs := obj.(*conf_v1.VirtualServer)
		changes, problems = lbc.configuration.AddOrUpdateVirtualServer(vs)
	}

	lbc.processChanges(changes)
	lbc.processProblems(problems)
}

func (lbc *LoadBalancerController) processProblems(problems []ConfigurationProblem) {
	glog.V(3).Infof("Processing %v problems", len(problems))

	for _, p := range problems {
		eventType := api_v1.EventTypeWarning
		lbc.recorder.Event(p.Object, eventType, p.Reason, p.Message)

		if lbc.reportCustomResourceStatusEnabled() {
			state := conf_v1.StateWarning
			if p.IsError {
				state = conf_v1.StateInvalid
			}

			switch obj := p.Object.(type) {
			case *networking.Ingress:
				err := lbc.statusUpdater.ClearIngressStatus(*obj)
				if err != nil {
					glog.V(3).Infof("Error when updating the status for Ingress %v/%v: %v", obj.Namespace, obj.Name, err)
				}
			case *conf_v1.VirtualServer:
				err := lbc.statusUpdater.UpdateVirtualServerStatus(obj, state, p.Reason, p.Message)
				if err != nil {
					glog.Errorf("Error when updating the status for VirtualServer %v/%v: %v", obj.Namespace, obj.Name, err)
				}
			case *conf_v1alpha1.TransportServer:
				err := lbc.statusUpdater.UpdateTransportServerStatus(obj, state, p.Reason, p.Message)
				if err != nil {
					glog.Errorf("Error when updating the status for TransportServer %v/%v: %v", obj.Namespace, obj.Name, err)
				}
			case *conf_v1.VirtualServerRoute:
				var emptyVSes []*conf_v1.VirtualServer
				err := lbc.statusUpdater.UpdateVirtualServerRouteStatusWithReferencedBy(obj, state, p.Reason, p.Message, emptyVSes)
				if err != nil {
					glog.Errorf("Error when updating the status for VirtualServerRoute %v/%v: %v", obj.Namespace, obj.Name, err)
				}
			}
		}
	}
}

func (lbc *LoadBalancerController) processChanges(changes []ResourceChange) {
	glog.V(3).Infof("Processing %v changes", len(changes))

	for _, c := range changes {
		if c.Op == AddOrUpdate {
			switch impl := c.Resource.(type) {
			case *VirtualServerConfiguration:
				vsEx := lbc.createVirtualServerEx(impl.VirtualServer, impl.VirtualServerRoutes)

				warnings, addOrUpdateErr := lbc.configurator.AddOrUpdateVirtualServer(vsEx)
				lbc.updateVirtualServerStatusAndEvents(impl, warnings, addOrUpdateErr)
			case *IngressConfiguration:
				if impl.IsMaster {
					mergeableIng := lbc.createMergeableIngresses(impl)

					warnings, addOrUpdateErr := lbc.configurator.AddOrUpdateMergeableIngress(mergeableIng)
					lbc.updateMergeableIngressStatusAndEvents(impl, warnings, addOrUpdateErr)
				} else {
					// for regular Ingress, validMinionPaths is nil
					ingEx := lbc.createIngressEx(impl.Ingress, impl.ValidHosts, nil)

					warnings, addOrUpdateErr := lbc.configurator.AddOrUpdateIngress(ingEx)
					lbc.updateRegularIngressStatusAndEvents(impl, warnings, addOrUpdateErr)
				}
			case *TransportServerConfiguration:
				tsEx := lbc.createTransportServerEx(impl.TransportServer, impl.ListenerPort)

				addOrUpdateErr := lbc.configurator.AddOrUpdateTransportServer(tsEx)
				lbc.updateTransportServerStatusAndEvents(impl, addOrUpdateErr)
			}
		} else if c.Op == Delete {
			switch impl := c.Resource.(type) {
			case *VirtualServerConfiguration:
				key := getResourceKey(&impl.VirtualServer.ObjectMeta)

				deleteErr := lbc.configurator.DeleteVirtualServer(key)
				if deleteErr != nil {
					glog.Errorf("Error when deleting configuration for VirtualServer %v: %v", key, deleteErr)
				}

				_, vsExists, err := lbc.virtualServerLister.GetByKey(key)
				if err != nil {
					glog.Errorf("Error when getting VirtualServer for %v: %v", key, err)
				}

				if vsExists {
					lbc.UpdateVirtualServerStatusAndEventsOnDelete(impl, c.Error, deleteErr)
				}
			case *IngressConfiguration:
				key := getResourceKey(&impl.Ingress.ObjectMeta)

				glog.V(2).Infof("Deleting Ingress: %v\n", key)

				deleteErr := lbc.configurator.DeleteIngress(key)
				if deleteErr != nil {
					glog.Errorf("Error when deleting configuration for Ingress %v: %v", key, deleteErr)
				}

				_, ingExists, err := lbc.ingressLister.GetByKeySafe(key)
				if err != nil {
					glog.Errorf("Error when getting Ingress for %v: %v", key, err)
				}

				if ingExists {
					lbc.UpdateIngressStatusAndEventsOnDelete(impl, c.Error, deleteErr)
				}
			case *TransportServerConfiguration:
				key := getResourceKey(&impl.TransportServer.ObjectMeta)

				deleteErr := lbc.configurator.DeleteTransportServer(key)

				if deleteErr != nil {
					glog.Errorf("Error when deleting configuration for TransportServer %v: %v", key, deleteErr)
				}

				_, tsExists, err := lbc.transportServerLister.GetByKey(key)
				if err != nil {
					glog.Errorf("Error when getting TransportServer for %v: %v", key, err)
				}
				if tsExists {
					lbc.updateTransportServerStatusAndEventsOnDelete(impl, c.Error, deleteErr)
				}
			}
		}
	}
}

// processChangesFromGlobalConfiguration processes changes that come from updates to the GlobalConfiguration resource.
// Such changes need to be processed at once to prevent any inconsistencies in the generated NGINX config.
func (lbc *LoadBalancerController) processChangesFromGlobalConfiguration(changes []ResourceChange) error {
	var updatedTSExes []*configs.TransportServerEx
	var deletedKeys []string

	var updatedResources []Resource

	for _, c := range changes {
		tsConfig := c.Resource.(*TransportServerConfiguration)

		if c.Op == AddOrUpdate {
			tsEx := lbc.createTransportServerEx(tsConfig.TransportServer, tsConfig.ListenerPort)

			updatedTSExes = append(updatedTSExes, tsEx)
			updatedResources = append(updatedResources, tsConfig)
		} else if c.Op == Delete {
			key := getResourceKey(&tsConfig.TransportServer.ObjectMeta)

			deletedKeys = append(deletedKeys, key)
		}
	}

	updateErr := lbc.configurator.UpdateTransportServers(updatedTSExes, deletedKeys)

	lbc.updateResourcesStatusAndEvents(updatedResources, configs.Warnings{}, updateErr)

	return updateErr
}

func (lbc *LoadBalancerController) processAppProtectChanges(changes []appprotect.Change) {
	glog.V(3).Infof("Processing %v App Protect changes", len(changes))

	for _, c := range changes {
		if c.Op == appprotect.AddOrUpdate {
			switch impl := c.Resource.(type) {
			case *appprotect.PolicyEx:
				namespace := impl.Obj.GetNamespace()
				name := impl.Obj.GetName()
				resources := lbc.configuration.FindResourcesForAppProtectPolicyAnnotation(namespace, name)

				for _, wafPol := range getWAFPoliciesForAppProtectPolicy(lbc.getAllPolicies(), namespace+"/"+name) {
					resources = append(resources, lbc.configuration.FindResourcesForPolicy(wafPol.Namespace, wafPol.Name)...)
				}

				resourceExes := lbc.createExtendedResources(resources)

				warnings, updateErr := lbc.configurator.AddOrUpdateAppProtectResource(impl.Obj, resourceExes.IngressExes, resourceExes.MergeableIngresses, resourceExes.VirtualServerExes)
				lbc.updateResourcesStatusAndEvents(resources, warnings, updateErr)
				lbc.recorder.Eventf(impl.Obj, api_v1.EventTypeNormal, "AddedOrUpdated", "AppProtectPolicy %v was added or updated", namespace+"/"+name)
			case *appprotect.LogConfEx:
				namespace := impl.Obj.GetNamespace()
				name := impl.Obj.GetName()
				resources := lbc.configuration.FindResourcesForAppProtectLogConfAnnotation(namespace, name)

				for _, wafPol := range getWAFPoliciesForAppProtectLogConf(lbc.getAllPolicies(), namespace+"/"+name) {
					resources = append(resources, lbc.configuration.FindResourcesForPolicy(wafPol.Namespace, wafPol.Name)...)
				}

				resourceExes := lbc.createExtendedResources(resources)

				warnings, updateErr := lbc.configurator.AddOrUpdateAppProtectResource(impl.Obj, resourceExes.IngressExes, resourceExes.MergeableIngresses, resourceExes.VirtualServerExes)
				lbc.updateResourcesStatusAndEvents(resources, warnings, updateErr)
				lbc.recorder.Eventf(impl.Obj, api_v1.EventTypeNormal, "AddedOrUpdated", "AppProtectLogConfig %v was added or updated", namespace+"/"+name)
			}
		} else if c.Op == appprotect.Delete {
			switch impl := c.Resource.(type) {
			case *appprotect.PolicyEx:
				namespace := impl.Obj.GetNamespace()
				name := impl.Obj.GetName()
				resources := lbc.configuration.FindResourcesForAppProtectPolicyAnnotation(namespace, name)

				for _, wafPol := range getWAFPoliciesForAppProtectPolicy(lbc.getAllPolicies(), namespace+"/"+name) {
					resources = append(resources, lbc.configuration.FindResourcesForPolicy(wafPol.Namespace, wafPol.Name)...)
				}

				resourceExes := lbc.createExtendedResources(resources)

				warnings, deleteErr := lbc.configurator.DeleteAppProtectPolicy(impl.Obj, resourceExes.IngressExes, resourceExes.MergeableIngresses, resourceExes.VirtualServerExes)

				lbc.updateResourcesStatusAndEvents(resources, warnings, deleteErr)

			case *appprotect.LogConfEx:
				namespace := impl.Obj.GetNamespace()
				name := impl.Obj.GetName()
				resources := lbc.configuration.FindResourcesForAppProtectLogConfAnnotation(namespace, name)

				for _, wafPol := range getWAFPoliciesForAppProtectLogConf(lbc.getAllPolicies(), namespace+"/"+name) {
					resources = append(resources, lbc.configuration.FindResourcesForPolicy(wafPol.Namespace, wafPol.Name)...)
				}

				resourceExes := lbc.createExtendedResources(resources)

				warnings, deleteErr := lbc.configurator.DeleteAppProtectLogConf(impl.Obj, resourceExes.IngressExes, resourceExes.MergeableIngresses, resourceExes.VirtualServerExes)

				lbc.updateResourcesStatusAndEvents(resources, warnings, deleteErr)
			}
		}
	}
}

func (lbc *LoadBalancerController) processAppProtectUserSigChange(change appprotect.UserSigChange) {
	var delPols []string
	var allIngExes []*configs.IngressEx
	var allMergeableIngresses []*configs.MergeableIngresses
	var allVsExes []*configs.VirtualServerEx
	var allResources []Resource

	for _, poladd := range change.PolicyAddsOrUpdates {
		resources := lbc.configuration.FindResourcesForAppProtectPolicyAnnotation(poladd.GetNamespace(), poladd.GetName())

		for _, wafPol := range getWAFPoliciesForAppProtectPolicy(lbc.getAllPolicies(), appprotectcommon.GetNsName(poladd)) {
			resources = append(resources, lbc.configuration.FindResourcesForPolicy(wafPol.Namespace, wafPol.Name)...)
		}

		resourceExes := lbc.createExtendedResources(resources)
		allIngExes = append(allIngExes, resourceExes.IngressExes...)
		allMergeableIngresses = append(allMergeableIngresses, resourceExes.MergeableIngresses...)
		allVsExes = append(allVsExes, resourceExes.VirtualServerExes...)
		allResources = append(allResources, resources...)
	}
	for _, poldel := range change.PolicyDeletions {
		resources := lbc.configuration.FindResourcesForAppProtectPolicyAnnotation(poldel.GetNamespace(), poldel.GetName())

		polNsName := appprotectcommon.GetNsName(poldel)
		for _, wafPol := range getWAFPoliciesForAppProtectPolicy(lbc.getAllPolicies(), polNsName) {
			resources = append(resources, lbc.configuration.FindResourcesForPolicy(wafPol.Namespace, wafPol.Name)...)
		}

		resourceExes := lbc.createExtendedResources(resources)
		allIngExes = append(allIngExes, resourceExes.IngressExes...)
		allMergeableIngresses = append(allMergeableIngresses, resourceExes.MergeableIngresses...)
		allVsExes = append(allVsExes, resourceExes.VirtualServerExes...)
		allResources = append(allResources, resources...)
		if len(resourceExes.IngressExes)+len(resourceExes.MergeableIngresses)+len(resourceExes.VirtualServerExes) > 0 {
			delPols = append(delPols, polNsName)
		}
	}

	warnings, err := lbc.configurator.RefreshAppProtectUserSigs(change.UserSigs, delPols, allIngExes, allMergeableIngresses, allVsExes)
	if err != nil {
		glog.Errorf("Error when refreshing App Protect Policy User defined signatures: %v", err)
	}
	lbc.updateResourcesStatusAndEvents(allResources, warnings, err)
}

func (lbc *LoadBalancerController) processAppProtectProblems(problems []appprotect.Problem) {
	glog.V(3).Infof("Processing %v App Protect problems", len(problems))

	for _, p := range problems {
		eventType := api_v1.EventTypeWarning
		lbc.recorder.Event(p.Object, eventType, p.Reason, p.Message)
	}
}

func (lbc *LoadBalancerController) processAppProtectDosChanges(changes []appprotectdos.Change) {
	glog.V(3).Infof("Processing %v App Protect Dos changes", len(changes))

	for _, c := range changes {
		if c.Op == appprotectdos.AddOrUpdate {
			switch impl := c.Resource.(type) {
			case *appprotectdos.DosProtectedResourceEx:
				glog.V(3).Infof("handling change UPDATE OR ADD for DOS protected %s/%s", impl.Obj.Namespace, impl.Obj.Name)
				resources := lbc.configuration.FindResourcesForAppProtectDosProtected(impl.Obj.Namespace, impl.Obj.Name)
				resourceExes := lbc.createExtendedResources(resources)
				warnings, err := lbc.configurator.AddOrUpdateResourcesThatUseDosProtected(resourceExes.IngressExes, resourceExes.MergeableIngresses, resourceExes.VirtualServerExes)
				lbc.updateResourcesStatusAndEvents(resources, warnings, err)
				msg := fmt.Sprintf("Configuration for %s/%s was added or updated", impl.Obj.Namespace, impl.Obj.Name)
				lbc.recorder.Event(impl.Obj, api_v1.EventTypeNormal, "AddedOrUpdated", msg)
			}
		} else if c.Op == appprotectdos.Delete {
			switch impl := c.Resource.(type) {
			case *appprotectdos.DosPolicyEx:
				lbc.configurator.DeleteAppProtectDosPolicy(impl.Obj)

			case *appprotectdos.DosLogConfEx:
				lbc.configurator.DeleteAppProtectDosLogConf(impl.Obj)

			case *appprotectdos.DosProtectedResourceEx:
				glog.V(3).Infof("handling change DELETE for DOS protected %s/%s", impl.Obj.Namespace, impl.Obj.Name)
				resources := lbc.configuration.FindResourcesForAppProtectDosProtected(impl.Obj.Namespace, impl.Obj.Name)
				resourceExes := lbc.createExtendedResources(resources)
				warnings, err := lbc.configurator.AddOrUpdateResourcesThatUseDosProtected(resourceExes.IngressExes, resourceExes.MergeableIngresses, resourceExes.VirtualServerExes)
				lbc.updateResourcesStatusAndEvents(resources, warnings, err)
			}
		}
	}
}

func (lbc *LoadBalancerController) processAppProtectDosProblems(problems []appprotectdos.Problem) {
	glog.V(3).Infof("Processing %v App Protect Dos problems", len(problems))

	for _, p := range problems {
		eventType := api_v1.EventTypeWarning
		lbc.recorder.Event(p.Object, eventType, p.Reason, p.Message)
	}
}

func (lbc *LoadBalancerController) updateTransportServerStatusAndEventsOnDelete(tsConfig *TransportServerConfiguration, changeError string, deleteErr error) {
	eventType := api_v1.EventTypeWarning
	eventTitle := "Rejected"
	eventWarningMessage := ""
	var state string

	// TransportServer either became invalid or lost its host or listener
	if changeError != "" {
		state = conf_v1.StateInvalid
		eventWarningMessage = fmt.Sprintf("with error: %s", changeError)
	} else if len(tsConfig.Warnings) > 0 {
		state = conf_v1.StateWarning
		eventWarningMessage = fmt.Sprintf("with warning(s): %s", formatWarningMessages(tsConfig.Warnings))
	}

	// we don't need to report anything if eventWarningMessage is empty
	// in that case, the resource was deleted because its class became incorrect
	// (some other Ingress Controller will handle it)

	if eventWarningMessage != "" {
		if deleteErr != nil {
			eventType = api_v1.EventTypeWarning
			eventTitle = "RejectedWithError"
			eventWarningMessage = fmt.Sprintf("%s; but was not applied: %v", eventWarningMessage, deleteErr)
			state = conf_v1.StateInvalid
		}

		msg := fmt.Sprintf("TransportServer %s was rejected %s", getResourceKey(&tsConfig.TransportServer.ObjectMeta), eventWarningMessage)
		lbc.recorder.Eventf(tsConfig.TransportServer, eventType, eventTitle, msg)

		if lbc.reportCustomResourceStatusEnabled() {
			err := lbc.statusUpdater.UpdateTransportServerStatus(tsConfig.TransportServer, state, eventTitle, msg)
			if err != nil {
				glog.Errorf("Error when updating the status for TransportServer %v/%v: %v", tsConfig.TransportServer.Namespace, tsConfig.TransportServer.Name, err)
			}
		}
	}
}

// UpdateVirtualServerStatusAndEventsOnDelete updates the virtual server status and events
func (lbc *LoadBalancerController) UpdateVirtualServerStatusAndEventsOnDelete(vsConfig *VirtualServerConfiguration, changeError string, deleteErr error) {
	eventType := api_v1.EventTypeWarning
	eventTitle := "Rejected"
	eventWarningMessage := ""
	state := ""

	// VirtualServer either became invalid or lost its host
	if changeError != "" {
		eventWarningMessage = fmt.Sprintf("with error: %s", changeError)
		state = conf_v1.StateInvalid
	} else if len(vsConfig.Warnings) > 0 {
		eventWarningMessage = fmt.Sprintf("with warning(s): %s", formatWarningMessages(vsConfig.Warnings))
		state = conf_v1.StateWarning
	}

	// we don't need to report anything if eventWarningMessage is empty
	// in that case, the resource was deleted because its class became incorrect
	// (some other Ingress Controller will handle it)
	if eventWarningMessage != "" {
		if deleteErr != nil {
			eventType = api_v1.EventTypeWarning
			eventTitle = "RejectedWithError"
			eventWarningMessage = fmt.Sprintf("%s; but was not applied: %v", eventWarningMessage, deleteErr)
			state = conf_v1.StateInvalid
		}

		msg := fmt.Sprintf("VirtualServer %s was rejected %s", getResourceKey(&vsConfig.VirtualServer.ObjectMeta), eventWarningMessage)
		lbc.recorder.Eventf(vsConfig.VirtualServer, eventType, eventTitle, msg)

		if lbc.reportCustomResourceStatusEnabled() {
			err := lbc.statusUpdater.UpdateVirtualServerStatus(vsConfig.VirtualServer, state, eventTitle, msg)
			if err != nil {
				glog.Errorf("Error when updating the status for VirtualServer %v/%v: %v", vsConfig.VirtualServer.Namespace, vsConfig.VirtualServer.Name, err)
			}
		}
	}

	// for delete, no need to report VirtualServerRoutes
	// for each VSR, a dedicated problem exists
}

// UpdateIngressStatusAndEventsOnDelete updates the ingress status and events.
func (lbc *LoadBalancerController) UpdateIngressStatusAndEventsOnDelete(ingConfig *IngressConfiguration, changeError string, deleteErr error) {
	eventTitle := "Rejected"
	eventWarningMessage := ""

	// Ingress either became invalid or lost all its hosts
	if changeError != "" {
		eventWarningMessage = fmt.Sprintf("with error: %s", changeError)
	} else if len(ingConfig.Warnings) > 0 {
		eventWarningMessage = fmt.Sprintf("with warning(s): %s", formatWarningMessages(ingConfig.Warnings))
	}

	// we don't need to report anything if eventWarningMessage is empty
	// in that case, the resource was deleted because its class became incorrect
	// (some other Ingress Controller will handle it)
	if eventWarningMessage != "" {
		if deleteErr != nil {
			eventTitle = "RejectedWithError"
			eventWarningMessage = fmt.Sprintf("%s; but was not applied: %v", eventWarningMessage, deleteErr)
		}

		lbc.recorder.Eventf(ingConfig.Ingress, api_v1.EventTypeWarning, eventTitle, "%v was rejected: %v", getResourceKey(&ingConfig.Ingress.ObjectMeta), eventWarningMessage)
		if lbc.reportStatusEnabled() {
			err := lbc.statusUpdater.ClearIngressStatus(*ingConfig.Ingress)
			if err != nil {
				glog.V(3).Infof("Error clearing Ingress status: %v", err)
			}
		}
	}

	// for delete, no need to report minions
	// for each minion, a dedicated problem exists
}

func (lbc *LoadBalancerController) updateResourcesStatusAndEvents(resources []Resource, warnings configs.Warnings, operationErr error) {
	for _, r := range resources {
		switch impl := r.(type) {
		case *VirtualServerConfiguration:
			lbc.updateVirtualServerStatusAndEvents(impl, warnings, operationErr)
		case *IngressConfiguration:
			if impl.IsMaster {
				lbc.updateMergeableIngressStatusAndEvents(impl, warnings, operationErr)
			} else {
				lbc.updateRegularIngressStatusAndEvents(impl, warnings, operationErr)
			}
		case *TransportServerConfiguration:
			lbc.updateTransportServerStatusAndEvents(impl, operationErr)
		}
	}
}

func (lbc *LoadBalancerController) updateMergeableIngressStatusAndEvents(ingConfig *IngressConfiguration, warnings configs.Warnings, operationErr error) {
	eventType := api_v1.EventTypeNormal
	eventTitle := "AddedOrUpdated"
	eventWarningMessage := ""

	if len(ingConfig.Warnings) > 0 {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithWarning"
		eventWarningMessage = fmt.Sprintf("with warning(s): %s", formatWarningMessages(ingConfig.Warnings))
	}

	if messages, ok := warnings[ingConfig.Ingress]; ok {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithWarning"
		eventWarningMessage = fmt.Sprintf("%s; with warning(s): %v", eventWarningMessage, formatWarningMessages(messages))
	}

	if operationErr != nil {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithError"
		eventWarningMessage = fmt.Sprintf("%s; but was not applied: %v", eventWarningMessage, operationErr)
	}

	msg := fmt.Sprintf("Configuration for %v was added or updated %s", getResourceKey(&ingConfig.Ingress.ObjectMeta), eventWarningMessage)
	lbc.recorder.Eventf(ingConfig.Ingress, eventType, eventTitle, msg)

	for _, fm := range ingConfig.Minions {
		minionEventType := api_v1.EventTypeNormal
		minionEventTitle := "AddedOrUpdated"
		minionEventWarningMessage := ""

		minionChangeWarnings := ingConfig.ChildWarnings[getResourceKey(&fm.Ingress.ObjectMeta)]
		if len(minionChangeWarnings) > 0 {
			minionEventType = api_v1.EventTypeWarning
			minionEventTitle = "AddedOrUpdatedWithWarning"
			minionEventWarningMessage = fmt.Sprintf("with warning(s): %s", formatWarningMessages(minionChangeWarnings))
		}

		if messages, ok := warnings[fm.Ingress]; ok {
			minionEventType = api_v1.EventTypeWarning
			minionEventTitle = "AddedOrUpdatedWithWarning"
			minionEventWarningMessage = fmt.Sprintf("%s; with warning(s): %v", minionEventWarningMessage, formatWarningMessages(messages))
		}

		if operationErr != nil {
			minionEventType = api_v1.EventTypeWarning
			minionEventTitle = "AddedOrUpdatedWithError"
			minionEventWarningMessage = fmt.Sprintf("%s; but was not applied: %v", minionEventWarningMessage, operationErr)
		}

		minionMsg := fmt.Sprintf("Configuration for %v/%v was added or updated %s", fm.Ingress.Namespace, fm.Ingress.Name, minionEventWarningMessage)
		lbc.recorder.Eventf(fm.Ingress, minionEventType, minionEventTitle, minionMsg)
	}

	if lbc.reportStatusEnabled() {
		ings := []networking.Ingress{*ingConfig.Ingress}

		for _, fm := range ingConfig.Minions {
			ings = append(ings, *fm.Ingress)
		}

		err := lbc.statusUpdater.BulkUpdateIngressStatus(ings)
		if err != nil {
			glog.V(3).Infof("error updating ing status: %v", err)
		}
	}
}

func (lbc *LoadBalancerController) updateRegularIngressStatusAndEvents(ingConfig *IngressConfiguration, warnings configs.Warnings, operationErr error) {
	eventType := api_v1.EventTypeNormal
	eventTitle := "AddedOrUpdated"
	eventWarningMessage := ""

	if len(ingConfig.Warnings) > 0 {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithWarning"
		eventWarningMessage = fmt.Sprintf("with warning(s): %s", formatWarningMessages(ingConfig.Warnings))
	}

	if messages, ok := warnings[ingConfig.Ingress]; ok {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithWarning"
		eventWarningMessage = fmt.Sprintf("%s; with warning(s): %v", eventWarningMessage, formatWarningMessages(messages))
	}

	if operationErr != nil {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithError"
		eventWarningMessage = fmt.Sprintf("%s; but was not applied: %v", eventWarningMessage, operationErr)
	}

	msg := fmt.Sprintf("Configuration for %v was added or updated %s", getResourceKey(&ingConfig.Ingress.ObjectMeta), eventWarningMessage)
	lbc.recorder.Eventf(ingConfig.Ingress, eventType, eventTitle, msg)

	if lbc.reportStatusEnabled() {
		err := lbc.statusUpdater.UpdateIngressStatus(*ingConfig.Ingress)
		if err != nil {
			glog.V(3).Infof("error updating ing status: %v", err)
		}
	}
}

func (lbc *LoadBalancerController) updateTransportServerStatusAndEvents(tsConfig *TransportServerConfiguration, operationErr error) {
	eventTitle := "AddedOrUpdated"
	eventType := api_v1.EventTypeNormal
	eventWarningMessage := ""
	state := conf_v1.StateValid

	if len(tsConfig.Warnings) > 0 {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithWarning"
		eventWarningMessage = fmt.Sprintf("with warning(s): %s", formatWarningMessages(tsConfig.Warnings))
		state = conf_v1.StateWarning
	}

	if operationErr != nil {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithError"
		eventWarningMessage = fmt.Sprintf("%s; but was not applied: %v", eventWarningMessage, operationErr)
		state = conf_v1.StateInvalid
	}

	msg := fmt.Sprintf("Configuration for %v was added or updated %s", getResourceKey(&tsConfig.TransportServer.ObjectMeta), eventWarningMessage)
	lbc.recorder.Eventf(tsConfig.TransportServer, eventType, eventTitle, msg)

	if lbc.reportCustomResourceStatusEnabled() {
		err := lbc.statusUpdater.UpdateTransportServerStatus(tsConfig.TransportServer, state, eventTitle, msg)
		if err != nil {
			glog.Errorf("Error when updating the status for TransportServer %v/%v: %v", tsConfig.TransportServer.Namespace, tsConfig.TransportServer.Name, err)
		}
	}
}

func (lbc *LoadBalancerController) updateVirtualServerStatusAndEvents(vsConfig *VirtualServerConfiguration, warnings configs.Warnings, operationErr error) {
	eventType := api_v1.EventTypeNormal
	eventTitle := "AddedOrUpdated"
	eventWarningMessage := ""
	state := conf_v1.StateValid

	if len(vsConfig.Warnings) > 0 {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithWarning"
		eventWarningMessage = fmt.Sprintf("with warning(s): %s", formatWarningMessages(vsConfig.Warnings))
		state = conf_v1.StateWarning
	}

	if messages, ok := warnings[vsConfig.VirtualServer]; ok {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithWarning"
		eventWarningMessage = fmt.Sprintf("%s; with warning(s): %v", eventWarningMessage, formatWarningMessages(messages))
		state = conf_v1.StateWarning
	}

	if operationErr != nil {
		eventType = api_v1.EventTypeWarning
		eventTitle = "AddedOrUpdatedWithError"
		eventWarningMessage = fmt.Sprintf("%s; but was not applied: %v", eventWarningMessage, operationErr)
		state = conf_v1.StateInvalid
	}

	msg := fmt.Sprintf("Configuration for %v was added or updated %s", getResourceKey(&vsConfig.VirtualServer.ObjectMeta), eventWarningMessage)
	lbc.recorder.Eventf(vsConfig.VirtualServer, eventType, eventTitle, msg)

	if lbc.reportCustomResourceStatusEnabled() {
		err := lbc.statusUpdater.UpdateVirtualServerStatus(vsConfig.VirtualServer, state, eventTitle, msg)
		if err != nil {
			glog.Errorf("Error when updating the status for VirtualServer %v/%v: %v", vsConfig.VirtualServer.Namespace, vsConfig.VirtualServer.Name, err)
		}
	}

	for _, vsr := range vsConfig.VirtualServerRoutes {
		vsrEventType := api_v1.EventTypeNormal
		vsrEventTitle := "AddedOrUpdated"
		vsrEventWarningMessage := ""
		vsrState := conf_v1.StateValid

		if messages, ok := warnings[vsr]; ok {
			vsrEventType = api_v1.EventTypeWarning
			vsrEventTitle = "AddedOrUpdatedWithWarning"
			vsrEventWarningMessage = fmt.Sprintf("with warning(s): %v", formatWarningMessages(messages))
			vsrState = conf_v1.StateWarning
		}

		if operationErr != nil {
			vsrEventType = api_v1.EventTypeWarning
			vsrEventTitle = "AddedOrUpdatedWithError"
			vsrEventWarningMessage = fmt.Sprintf("%s; but was not applied: %v", vsrEventWarningMessage, operationErr)
			vsrState = conf_v1.StateInvalid
		}

		msg := fmt.Sprintf("Configuration for %v/%v was added or updated %s", vsr.Namespace, vsr.Name, vsrEventWarningMessage)
		lbc.recorder.Eventf(vsr, vsrEventType, vsrEventTitle, msg)

		if lbc.reportCustomResourceStatusEnabled() {
			vss := []*conf_v1.VirtualServer{vsConfig.VirtualServer}
			err := lbc.statusUpdater.UpdateVirtualServerRouteStatusWithReferencedBy(vsr, vsrState, vsrEventTitle, msg, vss)
			if err != nil {
				glog.Errorf("Error when updating the status for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
			}
		}
	}
}

func (lbc *LoadBalancerController) syncVirtualServerRoute(task task) {
	key := task.Key
	obj, exists, err := lbc.virtualServerRouteLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []ResourceChange
	var problems []ConfigurationProblem

	if !exists {
		glog.V(2).Infof("Deleting VirtualServerRoute: %v\n", key)

		changes, problems = lbc.configuration.DeleteVirtualServerRoute(key)
	} else {
		glog.V(2).Infof("Adding or Updating VirtualServerRoute: %v\n", key)

		vsr := obj.(*conf_v1.VirtualServerRoute)
		changes, problems = lbc.configuration.AddOrUpdateVirtualServerRoute(vsr)
	}

	lbc.processChanges(changes)
	lbc.processProblems(problems)
}

func (lbc *LoadBalancerController) syncIngress(task task) {
	key := task.Key
	ing, ingExists, err := lbc.ingressLister.GetByKeySafe(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []ResourceChange
	var problems []ConfigurationProblem

	if !ingExists {
		glog.V(2).Infof("Deleting Ingress: %v\n", key)

		changes, problems = lbc.configuration.DeleteIngress(key)
	} else {
		glog.V(2).Infof("Adding or Updating Ingress: %v\n", key)

		changes, problems = lbc.configuration.AddOrUpdateIngress(ing)
	}

	lbc.processChanges(changes)
	lbc.processProblems(problems)
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

func (lbc *LoadBalancerController) updateTransportServerMetrics() {
	if !lbc.areCustomResourcesEnabled {
		return
	}

	metrics := lbc.configuration.GetTransportServerMetrics()
	lbc.metricsCollector.SetTransportServers(metrics.TotalTLSPassthrough, metrics.TotalTCP, metrics.TotalUDP)
}

func (lbc *LoadBalancerController) syncService(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing service %v", key)

	obj, exists, err := lbc.svcLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	// First case: the service is the external service for the Ingress Controller
	// In that case we need to update the statuses of all resources

	if lbc.IsExternalServiceKeyForStatus(key) {

		if !exists {
			// service got removed
			lbc.statusUpdater.ClearStatusFromExternalService()
		} else {
			// service added or updated
			lbc.statusUpdater.SaveStatusFromExternalService(obj.(*api_v1.Service))
		}

		if lbc.reportStatusEnabled() {
			ingresses := lbc.configuration.GetResourcesWithFilter(resourceFilter{Ingresses: true})

			glog.V(3).Infof("Updating status for %v Ingresses", len(ingresses))

			err := lbc.statusUpdater.UpdateExternalEndpointsForResources(ingresses)
			if err != nil {
				glog.Errorf("error updating ingress status in syncService: %v", err)
			}
		}

		if lbc.areCustomResourcesEnabled && lbc.reportCustomResourceStatusEnabled() {
			virtualServers := lbc.configuration.GetResourcesWithFilter(resourceFilter{VirtualServers: true})

			glog.V(3).Infof("Updating status for %v VirtualServers", len(virtualServers))

			err := lbc.statusUpdater.UpdateExternalEndpointsForResources(virtualServers)
			if err != nil {
				glog.V(3).Infof("error updating VirtualServer/VirtualServerRoute status in syncService: %v", err)
			}
		}

		// we don't return here because technically the same service could be used in the second case
	}

	// Second case: the service is referenced by some resources in the cluster

	// it is safe to ignore the error
	namespace, name, _ := ParseNamespaceName(key)

	resources := lbc.configuration.FindResourcesForService(namespace, name)

	if len(resources) == 0 {
		return
	}

	glog.V(3).Infof("Updating %v resources", len(resources))

	resourceExes := lbc.createExtendedResources(resources)

	warnings, updateErr := lbc.configurator.AddOrUpdateResources(resourceExes)
	lbc.updateResourcesStatusAndEvents(resources, warnings, updateErr)
}

// IsExternalServiceForStatus matches the service specified by the external-service cli arg
func (lbc *LoadBalancerController) IsExternalServiceForStatus(svc *api_v1.Service) bool {
	return lbc.statusUpdater.namespace == svc.Namespace && lbc.statusUpdater.externalServiceName == svc.Name
}

// IsExternalServiceKeyForStatus matches the service key specified by the external-service cli arg
func (lbc *LoadBalancerController) IsExternalServiceKeyForStatus(key string) bool {
	externalSvcKey := fmt.Sprintf("%s/%s", lbc.statusUpdater.namespace, lbc.statusUpdater.externalServiceName)
	return key == externalSvcKey
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

// reportCustomResourceStatusEnabled determines if we should attempt to report status for Custom Resources.
func (lbc *LoadBalancerController) reportCustomResourceStatusEnabled() bool {
	if lbc.isLeaderElectionEnabled {
		return lbc.leaderElector != nil && lbc.leaderElector.IsLeader()
	}

	return true
}

func (lbc *LoadBalancerController) syncSecret(task task) {
	key := task.Key
	obj, secrExists, err := lbc.secretLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	namespace, name, err := ParseNamespaceName(key)
	if err != nil {
		glog.Warningf("Secret key %v is invalid: %v", key, err)
		return
	}

	resources := lbc.configuration.FindResourcesForSecret(namespace, name)

	if lbc.areCustomResourcesEnabled {
		secretPols := lbc.getPoliciesForSecret(namespace, name)
		for _, pol := range secretPols {
			resources = append(resources, lbc.configuration.FindResourcesForPolicy(pol.Namespace, pol.Name)...)
		}

		resources = removeDuplicateResources(resources)
	}

	glog.V(2).Infof("Found %v Resources with Secret %v", len(resources), key)

	if !secrExists {
		lbc.secretStore.DeleteSecret(key)

		glog.V(2).Infof("Deleting Secret: %v\n", key)

		if len(resources) > 0 {
			lbc.handleRegularSecretDeletion(resources)
		}
		if lbc.isSpecialSecret(key) {
			glog.Warningf("A special TLS Secret %v was removed. Retaining the Secret.", key)
		}
		return
	}

	glog.V(2).Infof("Adding / Updating Secret: %v\n", key)

	secret := obj.(*api_v1.Secret)

	lbc.secretStore.AddOrUpdateSecret(secret)

	if lbc.isSpecialSecret(key) {
		lbc.handleSpecialSecretUpdate(secret)
		// we don't return here in case the special secret is also used in resources.
	}

	if len(resources) > 0 {
		lbc.handleSecretUpdate(secret, resources)
	}
}

func removeDuplicateResources(resources []Resource) []Resource {
	encountered := make(map[string]bool)
	var uniqueResources []Resource
	for _, r := range resources {
		key := r.GetKeyWithKind()
		if !encountered[key] {
			encountered[key] = true
			uniqueResources = append(uniqueResources, r)
		}
	}

	return uniqueResources
}

func (lbc *LoadBalancerController) isSpecialSecret(secretName string) bool {
	return secretName == lbc.defaultServerSecret || secretName == lbc.wildcardTLSSecret
}

func (lbc *LoadBalancerController) handleRegularSecretDeletion(resources []Resource) {
	resourceExes := lbc.createExtendedResources(resources)

	warnings, addOrUpdateErr := lbc.configurator.AddOrUpdateResources(resourceExes)

	lbc.updateResourcesStatusAndEvents(resources, warnings, addOrUpdateErr)
}

func (lbc *LoadBalancerController) handleSecretUpdate(secret *api_v1.Secret, resources []Resource) {
	secretNsName := secret.Namespace + "/" + secret.Name

	var warnings configs.Warnings
	var addOrUpdateErr error

	resourceExes := lbc.createExtendedResources(resources)
	warnings, addOrUpdateErr = lbc.configurator.AddOrUpdateResources(resourceExes)

	if addOrUpdateErr != nil {
		glog.Errorf("Error when updating Secret %v: %v", secretNsName, addOrUpdateErr)
		lbc.recorder.Eventf(secret, api_v1.EventTypeWarning, "UpdatedWithError", "%v was updated, but not applied: %v", secretNsName, addOrUpdateErr)
	}

	lbc.updateResourcesStatusAndEvents(resources, warnings, addOrUpdateErr)
}

func (lbc *LoadBalancerController) handleSpecialSecretUpdate(secret *api_v1.Secret) {
	var specialSecretsToUpdate []string
	secretNsName := secret.Namespace + "/" + secret.Name
	err := secrets.ValidateTLSSecret(secret)
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

func getStatusFromEventTitle(eventTitle string) string {
	switch eventTitle {
	case "AddedOrUpdatedWithError", "Rejected", "NoVirtualServersFound", "Missing Secret", "UpdatedWithError":
		return conf_v1.StateInvalid
	case "AddedOrUpdatedWithWarning", "UpdatedWithWarning":
		return conf_v1.StateWarning
	case "AddedOrUpdated", "Updated":
		return conf_v1.StateValid
	}

	return ""
}

func (lbc *LoadBalancerController) updateVirtualServersStatusFromEvents() error {
	var allErrs []error
	for _, obj := range lbc.virtualServerLister.List() {
		vs := obj.(*conf_v1.VirtualServer)

		if !lbc.HasCorrectIngressClass(vs) {
			glog.V(3).Infof("Ignoring VirtualServer %v based on class %v", vs.Name, vs.Spec.IngressClass)
			continue
		}

		events, err := lbc.client.CoreV1().Events(vs.Namespace).List(context.TODO(),
			meta_v1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%v,involvedObject.uid=%v", vs.Name, vs.UID)})
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("error trying to get events for VirtualServer %v/%v: %w", vs.Namespace, vs.Name, err))
			break
		}

		if len(events.Items) == 0 {
			continue
		}

		var timestamp time.Time
		var latestEvent api_v1.Event
		for _, event := range events.Items {
			if event.CreationTimestamp.After(timestamp) {
				latestEvent = event
			}
		}

		err = lbc.statusUpdater.UpdateVirtualServerStatus(vs, getStatusFromEventTitle(latestEvent.Reason), latestEvent.Reason, latestEvent.Message)
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if len(allErrs) > 0 {
		return fmt.Errorf("not all VirtualServers statuses were updated: %v", allErrs)
	}

	return nil
}

func (lbc *LoadBalancerController) updateVirtualServerRoutesStatusFromEvents() error {
	var allErrs []error
	for _, obj := range lbc.virtualServerRouteLister.List() {
		vsr := obj.(*conf_v1.VirtualServerRoute)

		if !lbc.HasCorrectIngressClass(vsr) {
			glog.V(3).Infof("Ignoring VirtualServerRoute %v based on class %v", vsr.Name, vsr.Spec.IngressClass)
			continue
		}

		events, err := lbc.client.CoreV1().Events(vsr.Namespace).List(context.TODO(),
			meta_v1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%v,involvedObject.uid=%v", vsr.Name, vsr.UID)})
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("error trying to get events for VirtualServerRoute %v/%v: %w", vsr.Namespace, vsr.Name, err))
			break
		}

		if len(events.Items) == 0 {
			continue
		}

		var timestamp time.Time
		var latestEvent api_v1.Event
		for _, event := range events.Items {
			if event.CreationTimestamp.After(timestamp) {
				latestEvent = event
			}
		}

		err = lbc.statusUpdater.UpdateVirtualServerRouteStatus(vsr, getStatusFromEventTitle(latestEvent.Reason), latestEvent.Reason, latestEvent.Message)
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if len(allErrs) > 0 {
		return fmt.Errorf("not all VirtualServerRoutes statuses were updated: %v", allErrs)
	}

	return nil
}

func (lbc *LoadBalancerController) updatePoliciesStatus() error {
	var allErrs []error
	for _, obj := range lbc.policyLister.List() {
		pol := obj.(*conf_v1.Policy)

		err := validation.ValidatePolicy(pol, lbc.isNginxPlus, lbc.enableOIDC, lbc.appProtectEnabled)
		if err != nil {
			msg := fmt.Sprintf("Policy %v/%v is invalid and was rejected: %v", pol.Namespace, pol.Name, err)
			err = lbc.statusUpdater.UpdatePolicyStatus(pol, conf_v1.StateInvalid, "Rejected", msg)
			if err != nil {
				allErrs = append(allErrs, err)
			}
		} else {
			msg := fmt.Sprintf("Policy %v/%v was added or updated", pol.Namespace, pol.Name)
			err = lbc.statusUpdater.UpdatePolicyStatus(pol, conf_v1.StateValid, "AddedOrUpdated", msg)
			if err != nil {
				allErrs = append(allErrs, err)
			}
		}
	}

	if len(allErrs) != 0 {
		return fmt.Errorf("not all Policies statuses were updated: %v", allErrs)
	}

	return nil
}

func (lbc *LoadBalancerController) updateTransportServersStatusFromEvents() error {
	var allErrs []error
	for _, obj := range lbc.transportServerLister.List() {
		ts := obj.(*conf_v1alpha1.TransportServer)

		events, err := lbc.client.CoreV1().Events(ts.Namespace).List(context.TODO(),
			meta_v1.ListOptions{FieldSelector: fmt.Sprintf("involvedObject.name=%v,involvedObject.uid=%v", ts.Name, ts.UID)})
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("error trying to get events for TransportServer %v/%v: %w", ts.Namespace, ts.Name, err))
			break
		}

		if len(events.Items) == 0 {
			continue
		}

		var timestamp time.Time
		var latestEvent api_v1.Event
		for _, event := range events.Items {
			if event.CreationTimestamp.After(timestamp) {
				latestEvent = event
			}
		}

		err = lbc.statusUpdater.UpdateTransportServerStatus(ts, getStatusFromEventTitle(latestEvent.Reason), latestEvent.Reason, latestEvent.Message)
		if err != nil {
			allErrs = append(allErrs, err)
		}
	}

	if len(allErrs) > 0 {
		return fmt.Errorf("not all TransportServers statuses were updated: %v", allErrs)
	}

	return nil
}

func getIPAddressesFromEndpoints(endpoints []podEndpoint) []string {
	var endps []string
	for _, ep := range endpoints {
		endps = append(endps, ep.Address)
	}
	return endps
}

func (lbc *LoadBalancerController) createMergeableIngresses(ingConfig *IngressConfiguration) *configs.MergeableIngresses {
	// for master Ingress, validMinionPaths are nil
	masterIngressEx := lbc.createIngressEx(ingConfig.Ingress, ingConfig.ValidHosts, nil)

	var minions []*configs.IngressEx

	for _, m := range ingConfig.Minions {
		minions = append(minions, lbc.createIngressEx(m.Ingress, ingConfig.ValidHosts, m.ValidPaths))
	}

	return &configs.MergeableIngresses{
		Master:  masterIngressEx,
		Minions: minions,
	}
}

func (lbc *LoadBalancerController) createIngressEx(ing *networking.Ingress, validHosts map[string]bool, validMinionPaths map[string]bool) *configs.IngressEx {
	ingEx := &configs.IngressEx{
		Ingress:          ing,
		ValidHosts:       validHosts,
		ValidMinionPaths: validMinionPaths,
	}

	ingEx.SecretRefs = make(map[string]*secrets.SecretReference)

	for _, tls := range ing.Spec.TLS {
		secretName := tls.SecretName
		secretKey := ing.Namespace + "/" + secretName

		secretRef := lbc.secretStore.GetSecret(secretKey)
		if secretRef.Error != nil {
			glog.Warningf("Error trying to get the secret %v for Ingress %v: %v", secretName, ing.Name, secretRef.Error)
		}

		ingEx.SecretRefs[secretName] = secretRef
	}

	if lbc.isNginxPlus {
		if jwtKey, exists := ingEx.Ingress.Annotations[configs.JWTKeyAnnotation]; exists {
			secretName := jwtKey
			secretKey := ing.Namespace + "/" + secretName

			secretRef := lbc.secretStore.GetSecret(secretKey)
			if secretRef.Error != nil {
				glog.Warningf("Error trying to get the secret %v for Ingress %v/%v: %v", secretName, ing.Namespace, ing.Name, secretRef.Error)
			}

			ingEx.SecretRefs[secretName] = secretRef
		}
		if lbc.appProtectEnabled {
			if apPolicyAntn, exists := ingEx.Ingress.Annotations[configs.AppProtectPolicyAnnotation]; exists {
				policy, err := lbc.getAppProtectPolicy(ing)
				if err != nil {
					glog.Warningf("Error Getting App Protect policy %v for Ingress %v/%v: %v", apPolicyAntn, ing.Namespace, ing.Name, err)
				} else {
					ingEx.AppProtectPolicy = policy
				}
			}

			if apLogConfAntn, exists := ingEx.Ingress.Annotations[configs.AppProtectLogConfAnnotation]; exists {
				logConf, err := lbc.getAppProtectLogConfAndDst(ing)
				if err != nil {
					glog.Warningf("Error Getting App Protect Log Config %v for Ingress %v/%v: %v", apLogConfAntn, ing.Namespace, ing.Name, err)
				} else {
					ingEx.AppProtectLogs = logConf
				}
			}
		}

		if lbc.appProtectDosEnabled {
			if dosProtectedAnnotationValue, exists := ingEx.Ingress.Annotations[configs.AppProtectDosProtectedAnnotation]; exists {
				dosResEx, err := lbc.dosConfiguration.GetValidDosEx(ing.Namespace, dosProtectedAnnotationValue)
				if err != nil {
					glog.Warningf("Error Getting Dos Protected Resource %v for Ingress %v/%v: %v", dosProtectedAnnotationValue, ing.Namespace, ing.Name, err)
				}
				if dosResEx != nil {
					ingEx.DosEx = dosResEx
				}
			}
		}
	}

	ingEx.Endpoints = make(map[string][]string)
	ingEx.HealthChecks = make(map[string]*api_v1.Probe)
	ingEx.ExternalNameSvcs = make(map[string]bool)
	ingEx.PodsByIP = make(map[string]configs.PodInfo)

	if ing.Spec.DefaultBackend != nil {
		podEndps := []podEndpoint{}
		var external bool
		svc, err := lbc.getServiceForIngressBackend(ing.Spec.DefaultBackend, ing.Namespace)
		if err != nil {
			glog.V(3).Infof("Error getting service %v: %v", ing.Spec.DefaultBackend.Service.Name, err)
		} else {
			podEndps, external, err = lbc.getEndpointsForIngressBackend(ing.Spec.DefaultBackend, svc)
			if err == nil && external && lbc.isNginxPlus {
				ingEx.ExternalNameSvcs[svc.Name] = true
			}
		}

		if err != nil {
			glog.Warningf("Error retrieving endpoints for the service %v: %v", ing.Spec.DefaultBackend.Service.Name, err)
		}

		endps := getIPAddressesFromEndpoints(podEndps)

		// endps is empty if there was any error before this point
		ingEx.Endpoints[ing.Spec.DefaultBackend.Service.Name+configs.GetBackendPortAsString(ing.Spec.DefaultBackend.Service.Port)] = endps

		if lbc.isNginxPlus && lbc.isHealthCheckEnabled(ing) {
			healthCheck := lbc.getHealthChecksForIngressBackend(ing.Spec.DefaultBackend, ing.Namespace)
			if healthCheck != nil {
				ingEx.HealthChecks[ing.Spec.DefaultBackend.Service.Name+configs.GetBackendPortAsString(ing.Spec.DefaultBackend.Service.Port)] = healthCheck
			}
		}

		if (lbc.isNginxPlus && lbc.isPrometheusEnabled) || lbc.isLatencyMetricsEnabled {
			for _, endpoint := range podEndps {
				ingEx.PodsByIP[endpoint.Address] = configs.PodInfo{
					Name:         endpoint.PodName,
					MeshPodOwner: endpoint.MeshPodOwner,
				}
			}
		}
	}

	for _, rule := range ing.Spec.Rules {
		if !validHosts[rule.Host] {
			glog.V(3).Infof("Skipping host %s for Ingress %s", rule.Host, ing.Name)
			continue
		}

		// check if rule has any paths
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}

		for _, path := range rule.HTTP.Paths {
			podEndps := []podEndpoint{}
			if validMinionPaths != nil && !validMinionPaths[path.Path] {
				glog.V(3).Infof("Skipping path %s for minion Ingress %s", path.Path, ing.Name)
				continue
			}

			var external bool
			svc, err := lbc.getServiceForIngressBackend(&path.Backend, ing.Namespace)
			if err != nil {
				glog.V(3).Infof("Error getting service %v: %v", &path.Backend.Service.Name, err)
			} else {
				podEndps, external, err = lbc.getEndpointsForIngressBackend(&path.Backend, svc)
				if err == nil && external && lbc.isNginxPlus {
					ingEx.ExternalNameSvcs[svc.Name] = true
				}
			}

			if err != nil {
				glog.Warningf("Error retrieving endpoints for the service %v: %v", path.Backend.Service.Name, err)
			}

			endps := getIPAddressesFromEndpoints(podEndps)

			// endps is empty if there was any error before this point
			ingEx.Endpoints[path.Backend.Service.Name+configs.GetBackendPortAsString(path.Backend.Service.Port)] = endps

			// Pull active health checks from k8 api
			if lbc.isNginxPlus && lbc.isHealthCheckEnabled(ing) {
				healthCheck := lbc.getHealthChecksForIngressBackend(&path.Backend, ing.Namespace)
				if healthCheck != nil {
					ingEx.HealthChecks[path.Backend.Service.Name+configs.GetBackendPortAsString(path.Backend.Service.Port)] = healthCheck
				}
			}

			if lbc.isNginxPlus || lbc.isLatencyMetricsEnabled {
				for _, endpoint := range podEndps {
					ingEx.PodsByIP[endpoint.Address] = configs.PodInfo{
						Name:         endpoint.PodName,
						MeshPodOwner: endpoint.MeshPodOwner,
					}
				}
			}
		}
	}

	return ingEx
}

func (lbc *LoadBalancerController) getAppProtectLogConfAndDst(ing *networking.Ingress) ([]configs.AppProtectLog, error) {
	var apLogs []configs.AppProtectLog
	if _, exists := ing.Annotations[configs.AppProtectLogConfDstAnnotation]; !exists {
		return apLogs, fmt.Errorf("Error: %v requires %v in %v", configs.AppProtectLogConfAnnotation, configs.AppProtectLogConfDstAnnotation, ing.Name)
	}

	logDsts := strings.Split(ing.Annotations[configs.AppProtectLogConfDstAnnotation], ",")
	logConfNsNs := appprotectcommon.ParseResourceReferenceAnnotationList(ing.Namespace, ing.Annotations[configs.AppProtectLogConfAnnotation])
	if len(logDsts) != len(logConfNsNs) {
		return apLogs, fmt.Errorf("Error Validating App Protect Destination and Config for Ingress %v: LogConf and LogDestination must have equal number of items", ing.Name)
	}

	for _, logDst := range logDsts {
		err := validation.ValidateAppProtectLogDestination(logDst)
		if err != nil {
			return apLogs, fmt.Errorf("Error Validating App Protect Destination Config for Ingress %v: %w", ing.Name, err)
		}
	}

	for i, logConfNsN := range logConfNsNs {
		logConf, err := lbc.appProtectConfiguration.GetAppResource(appprotect.LogConfGVK.Kind, logConfNsN)
		if err != nil {
			return apLogs, fmt.Errorf("Error retrieving App Protect Log Config for Ingress %v: %w", ing.Name, err)
		}
		apLogs = append(apLogs, configs.AppProtectLog{
			LogConf: logConf,
			Dest:    logDsts[i],
		})
	}

	return apLogs, nil
}

func (lbc *LoadBalancerController) getAppProtectPolicy(ing *networking.Ingress) (apPolicy *unstructured.Unstructured, err error) {
	polNsN := appprotectcommon.ParseResourceReferenceAnnotation(ing.Namespace, ing.Annotations[configs.AppProtectPolicyAnnotation])

	apPolicy, err = lbc.appProtectConfiguration.GetAppResource(appprotect.PolicyGVK.Kind, polNsN)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving App Protect Policy for Ingress %v: %w", ing.Name, err)
	}

	return apPolicy, nil
}

func (lbc *LoadBalancerController) createVirtualServerEx(virtualServer *conf_v1.VirtualServer, virtualServerRoutes []*conf_v1.VirtualServerRoute) *configs.VirtualServerEx {
	virtualServerEx := configs.VirtualServerEx{
		VirtualServer:  virtualServer,
		SecretRefs:     make(map[string]*secrets.SecretReference),
		ApPolRefs:      make(map[string]*unstructured.Unstructured),
		LogConfRefs:    make(map[string]*unstructured.Unstructured),
		DosProtectedEx: make(map[string]*configs.DosEx),
	}

	if virtualServer.Spec.TLS != nil && virtualServer.Spec.TLS.Secret != "" {
		secretKey := virtualServer.Namespace + "/" + virtualServer.Spec.TLS.Secret

		secretRef := lbc.secretStore.GetSecret(secretKey)
		if secretRef.Error != nil {
			glog.Warningf("Error trying to get the secret %v for VirtualServer %v: %v", secretKey, virtualServer.Name, secretRef.Error)
		}

		virtualServerEx.SecretRefs[secretKey] = secretRef
	}

	policies, policyErrors := lbc.getPolicies(virtualServer.Spec.Policies, virtualServer.Namespace)
	for _, err := range policyErrors {
		glog.Warningf("Error getting policy for VirtualServer %s/%s: %v", virtualServer.Namespace, virtualServer.Name, err)
	}

	err := lbc.addJWTSecretRefs(virtualServerEx.SecretRefs, policies)
	if err != nil {
		glog.Warningf("Error getting JWT secrets for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
	}
	err = lbc.addIngressMTLSSecretRefs(virtualServerEx.SecretRefs, policies)
	if err != nil {
		glog.Warningf("Error getting IngressMTLS secret for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
	}
	err = lbc.addEgressMTLSSecretRefs(virtualServerEx.SecretRefs, policies)
	if err != nil {
		glog.Warningf("Error getting EgressMTLS secrets for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
	}
	err = lbc.addOIDCSecretRefs(virtualServerEx.SecretRefs, policies)
	if err != nil {
		glog.Warningf("Error getting OIDC secrets for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
	}

	err = lbc.addWAFPolicyRefs(virtualServerEx.ApPolRefs, virtualServerEx.LogConfRefs, policies)
	if err != nil {
		glog.Warningf("Error getting App Protect resource for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
	}

	if virtualServer.Spec.Dos != "" {
		dosEx, err := lbc.dosConfiguration.GetValidDosEx(virtualServer.Namespace, virtualServer.Spec.Dos)
		if err != nil {
			glog.Warningf("Error getting App Protect Dos resource for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
		}
		if dosEx != nil {
			virtualServerEx.DosProtectedEx[""] = dosEx
		}
	}

	endpoints := make(map[string][]string)
	externalNameSvcs := make(map[string]bool)
	podsByIP := make(map[string]configs.PodInfo)

	for _, u := range virtualServer.Spec.Upstreams {
		endpointsKey := configs.GenerateEndpointsKey(virtualServer.Namespace, u.Service, u.Subselector, u.Port)

		var endps []string
		if u.UseClusterIP {
			s, err := lbc.getServiceForUpstream(virtualServer.Namespace, u.Service, u.Port)
			if err != nil {
				glog.Warningf("Error getting Service for Upstream %v: %v", u.Service, err)
			} else {
				endps = append(endps, ipv6SafeAddrPort(s.Spec.ClusterIP, int32(u.Port)))
			}

		} else {
			var podEndps []podEndpoint
			var err error

			if len(u.Subselector) > 0 {
				podEndps, err = lbc.getEndpointsForSubselector(virtualServer.Namespace, u)
			} else {
				var external bool
				podEndps, external, err = lbc.getEndpointsForUpstream(virtualServer.Namespace, u.Service, u.Port)

				if err == nil && external && lbc.isNginxPlus {
					externalNameSvcs[configs.GenerateExternalNameSvcKey(virtualServer.Namespace, u.Service)] = true
				}
			}

			if err != nil {
				glog.Warningf("Error getting Endpoints for Upstream %v: %v", u.Name, err)
			}

			endps = getIPAddressesFromEndpoints(podEndps)

			if (lbc.isNginxPlus && lbc.isPrometheusEnabled) || lbc.isLatencyMetricsEnabled {
				for _, endpoint := range podEndps {
					podsByIP[endpoint.Address] = configs.PodInfo{
						Name:         endpoint.PodName,
						MeshPodOwner: endpoint.MeshPodOwner,
					}
				}
			}
		}

		endpoints[endpointsKey] = endps

	}

	for _, r := range virtualServer.Spec.Routes {
		vsRoutePolicies, policyErrors := lbc.getPolicies(r.Policies, virtualServer.Namespace)
		for _, err := range policyErrors {
			glog.Warningf("Error getting policy for VirtualServer %s/%s: %v", virtualServer.Namespace, virtualServer.Name, err)
		}
		policies = append(policies, vsRoutePolicies...)

		err = lbc.addJWTSecretRefs(virtualServerEx.SecretRefs, vsRoutePolicies)
		if err != nil {
			glog.Warningf("Error getting JWT secrets for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
		}
		err = lbc.addEgressMTLSSecretRefs(virtualServerEx.SecretRefs, vsRoutePolicies)
		if err != nil {
			glog.Warningf("Error getting EgressMTLS secrets for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
		}

		err = lbc.addWAFPolicyRefs(virtualServerEx.ApPolRefs, virtualServerEx.LogConfRefs, vsRoutePolicies)
		if err != nil {
			glog.Warningf("Error getting WAF policies for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
		}

		if r.Dos != "" {
			routeDosEx, err := lbc.dosConfiguration.GetValidDosEx(virtualServer.Namespace, r.Dos)
			if err != nil {
				glog.Warningf("Error getting App Protect Dos resource for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
			}
			virtualServerEx.DosProtectedEx[r.Path] = routeDosEx
		}

		err = lbc.addOIDCSecretRefs(virtualServerEx.SecretRefs, vsRoutePolicies)
		if err != nil {
			glog.Warningf("Error getting OIDC secrets for VirtualServer %v/%v: %v", virtualServer.Namespace, virtualServer.Name, err)
		}
	}

	for _, vsr := range virtualServerRoutes {
		for _, sr := range vsr.Spec.Subroutes {
			vsrSubroutePolicies, policyErrors := lbc.getPolicies(sr.Policies, vsr.Namespace)
			for _, err := range policyErrors {
				glog.Warningf("Error getting policy for VirtualServerRoute %s/%s: %v", vsr.Namespace, vsr.Name, err)
			}
			policies = append(policies, vsrSubroutePolicies...)

			err = lbc.addJWTSecretRefs(virtualServerEx.SecretRefs, vsrSubroutePolicies)
			if err != nil {
				glog.Warningf("Error getting JWT secrets for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
			}

			err = lbc.addEgressMTLSSecretRefs(virtualServerEx.SecretRefs, vsrSubroutePolicies)
			if err != nil {
				glog.Warningf("Error getting EgressMTLS secrets for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
			}

			err = lbc.addOIDCSecretRefs(virtualServerEx.SecretRefs, vsrSubroutePolicies)
			if err != nil {
				glog.Warningf("Error getting OIDC secrets for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
			}

			err = lbc.addWAFPolicyRefs(virtualServerEx.ApPolRefs, virtualServerEx.LogConfRefs, vsrSubroutePolicies)
			if err != nil {
				glog.Warningf("Error getting WAF policies for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
			}

			if sr.Dos != "" {
				routeDosEx, err := lbc.dosConfiguration.GetValidDosEx(vsr.Namespace, sr.Dos)
				if err != nil {
					glog.Warningf("Error getting App Protect Dos resource for VirtualServerRoute %v/%v: %v", vsr.Namespace, vsr.Name, err)
				}
				virtualServerEx.DosProtectedEx[sr.Path] = routeDosEx
			}
		}

		for _, u := range vsr.Spec.Upstreams {
			endpointsKey := configs.GenerateEndpointsKey(vsr.Namespace, u.Service, u.Subselector, u.Port)

			var endps []string
			if u.UseClusterIP {
				s, err := lbc.getServiceForUpstream(vsr.Namespace, u.Service, u.Port)
				if err != nil {
					glog.Warningf("Error getting Service for Upstream %v: %v", u.Service, err)
				} else {
					endps = append(endps, fmt.Sprintf("%s:%d", s.Spec.ClusterIP, u.Port))
				}

			} else {
				var podEndps []podEndpoint
				var err error
				if len(u.Subselector) > 0 {
					podEndps, err = lbc.getEndpointsForSubselector(vsr.Namespace, u)
				} else {
					var external bool
					podEndps, external, err = lbc.getEndpointsForUpstream(vsr.Namespace, u.Service, u.Port)

					if err == nil && external && lbc.isNginxPlus {
						externalNameSvcs[configs.GenerateExternalNameSvcKey(vsr.Namespace, u.Service)] = true
					}
				}
				if err != nil {
					glog.Warningf("Error getting Endpoints for Upstream %v: %v", u.Name, err)
				}

				endps = getIPAddressesFromEndpoints(podEndps)

				if lbc.isNginxPlus || lbc.isLatencyMetricsEnabled {
					for _, endpoint := range podEndps {
						podsByIP[endpoint.Address] = configs.PodInfo{
							Name:         endpoint.PodName,
							MeshPodOwner: endpoint.MeshPodOwner,
						}
					}
				}
			}
			endpoints[endpointsKey] = endps
		}
	}

	virtualServerEx.Endpoints = endpoints
	virtualServerEx.VirtualServerRoutes = virtualServerRoutes
	virtualServerEx.ExternalNameSvcs = externalNameSvcs
	virtualServerEx.Policies = createPolicyMap(policies)
	virtualServerEx.PodsByIP = podsByIP

	return &virtualServerEx
}

func createPolicyMap(policies []*conf_v1.Policy) map[string]*conf_v1.Policy {
	result := make(map[string]*conf_v1.Policy)

	for _, p := range policies {
		key := fmt.Sprintf("%s/%s", p.Namespace, p.Name)
		result[key] = p
	}

	return result
}

func (lbc *LoadBalancerController) getAllPolicies() []*conf_v1.Policy {
	var policies []*conf_v1.Policy

	for _, obj := range lbc.policyLister.List() {
		pol := obj.(*conf_v1.Policy)

		err := validation.ValidatePolicy(pol, lbc.isNginxPlus, lbc.enableOIDC, lbc.appProtectEnabled)
		if err != nil {
			glog.V(3).Infof("Skipping invalid Policy %s/%s: %v", pol.Namespace, pol.Name, err)
			continue
		}

		policies = append(policies, pol)
	}

	return policies
}

func (lbc *LoadBalancerController) getPolicies(policies []conf_v1.PolicyReference, ownerNamespace string) ([]*conf_v1.Policy, []error) {
	var result []*conf_v1.Policy
	var errors []error

	for _, p := range policies {
		polNamespace := p.Namespace
		if polNamespace == "" {
			polNamespace = ownerNamespace
		}

		policyKey := fmt.Sprintf("%s/%s", polNamespace, p.Name)

		policyObj, exists, err := lbc.policyLister.GetByKey(policyKey)
		if err != nil {
			errors = append(errors, fmt.Errorf("Failed to get policy %s: %w", policyKey, err))
			continue
		}

		if !exists {
			errors = append(errors, fmt.Errorf("Policy %s doesn't exist", policyKey))
			continue
		}

		policy := policyObj.(*conf_v1.Policy)

		if !lbc.HasCorrectIngressClass(policy) {
			errors = append(errors, fmt.Errorf("referenced policy %s has incorrect ingress class: %s (controller ingress class: %s)", policyKey, policy.Spec.IngressClass, lbc.ingressClass))
			continue
		}

		err = validation.ValidatePolicy(policy, lbc.isNginxPlus, lbc.enableOIDC, lbc.appProtectEnabled)
		if err != nil {
			errors = append(errors, fmt.Errorf("Policy %s is invalid: %w", policyKey, err))
			continue
		}

		result = append(result, policy)
	}

	return result, errors
}

func (lbc *LoadBalancerController) addJWTSecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.JWTAuth == nil {
			continue
		}

		secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.JWTAuth.Secret)
		secretRef := lbc.secretStore.GetSecret(secretKey)

		secretRefs[secretKey] = secretRef

		if secretRef.Error != nil {
			return secretRef.Error
		}
	}

	return nil
}

func (lbc *LoadBalancerController) addIngressMTLSSecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.IngressMTLS == nil {
			continue
		}

		secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.IngressMTLS.ClientCertSecret)
		secretRef := lbc.secretStore.GetSecret(secretKey)

		secretRefs[secretKey] = secretRef

		return secretRef.Error
	}

	return nil
}

func (lbc *LoadBalancerController) addEgressMTLSSecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.EgressMTLS == nil {
			continue
		}
		if pol.Spec.EgressMTLS.TLSSecret != "" {
			secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.EgressMTLS.TLSSecret)
			secretRef := lbc.secretStore.GetSecret(secretKey)

			secretRefs[secretKey] = secretRef

			if secretRef.Error != nil {
				return secretRef.Error
			}
		}
		if pol.Spec.EgressMTLS.TrustedCertSecret != "" {
			secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.EgressMTLS.TrustedCertSecret)
			secretRef := lbc.secretStore.GetSecret(secretKey)

			secretRefs[secretKey] = secretRef

			if secretRef.Error != nil {
				return secretRef.Error
			}
		}
	}

	return nil
}

func (lbc *LoadBalancerController) addOIDCSecretRefs(secretRefs map[string]*secrets.SecretReference, policies []*conf_v1.Policy) error {
	for _, pol := range policies {
		if pol.Spec.OIDC == nil {
			continue
		}

		secretKey := fmt.Sprintf("%v/%v", pol.Namespace, pol.Spec.OIDC.ClientSecret)
		secretRef := lbc.secretStore.GetSecret(secretKey)

		secretRefs[secretKey] = secretRef

		if secretRef.Error != nil {
			return secretRef.Error
		}
	}
	return nil
}

// addWAFPolicyRefs ensures the app protect resources that are referenced in policies exist.
func (lbc *LoadBalancerController) addWAFPolicyRefs(
	apPolRef, logConfRef map[string]*unstructured.Unstructured,
	policies []*conf_v1.Policy,
) error {
	for _, pol := range policies {
		if pol.Spec.WAF == nil {
			continue
		}

		if pol.Spec.WAF.ApPolicy != "" {
			apPolKey := pol.Spec.WAF.ApPolicy
			if !strings.Contains(pol.Spec.WAF.ApPolicy, "/") {
				apPolKey = fmt.Sprintf("%v/%v", pol.Namespace, apPolKey)
			}

			apPolicy, err := lbc.appProtectConfiguration.GetAppResource(appprotect.PolicyGVK.Kind, apPolKey)
			if err != nil {
				return fmt.Errorf("WAF policy %q is invalid: %w", apPolKey, err)
			}
			apPolRef[apPolKey] = apPolicy
		}

		if pol.Spec.WAF.SecurityLog != nil && pol.Spec.WAF.SecurityLog.ApLogConf != "" {
			logConfKey := pol.Spec.WAF.SecurityLog.ApLogConf
			if !strings.Contains(pol.Spec.WAF.SecurityLog.ApLogConf, "/") {
				logConfKey = fmt.Sprintf("%v/%v", pol.Namespace, logConfKey)
			}

			logConf, err := lbc.appProtectConfiguration.GetAppResource(appprotect.LogConfGVK.Kind, logConfKey)
			if err != nil {
				return fmt.Errorf("WAF policy %q is invalid: %w", logConfKey, err)
			}
			logConfRef[logConfKey] = logConf
		}

	}
	return nil
}

func (lbc *LoadBalancerController) getPoliciesForSecret(secretNamespace string, secretName string) []*conf_v1.Policy {
	return findPoliciesForSecret(lbc.getAllPolicies(), secretNamespace, secretName)
}

func findPoliciesForSecret(policies []*conf_v1.Policy, secretNamespace string, secretName string) []*conf_v1.Policy {
	var res []*conf_v1.Policy

	for _, pol := range policies {
		if pol.Spec.IngressMTLS != nil && pol.Spec.IngressMTLS.ClientCertSecret == secretName && pol.Namespace == secretNamespace {
			res = append(res, pol)
		} else if pol.Spec.JWTAuth != nil && pol.Spec.JWTAuth.Secret == secretName && pol.Namespace == secretNamespace {
			res = append(res, pol)
		} else if pol.Spec.EgressMTLS != nil && pol.Spec.EgressMTLS.TLSSecret == secretName && pol.Namespace == secretNamespace {
			res = append(res, pol)
		} else if pol.Spec.EgressMTLS != nil && pol.Spec.EgressMTLS.TrustedCertSecret == secretName && pol.Namespace == secretNamespace {
			res = append(res, pol)
		} else if pol.Spec.OIDC != nil && pol.Spec.OIDC.ClientSecret == secretName && pol.Namespace == secretNamespace {
			res = append(res, pol)
		}
	}

	return res
}

func getWAFPoliciesForAppProtectPolicy(pols []*conf_v1.Policy, key string) []*conf_v1.Policy {
	var policies []*conf_v1.Policy

	for _, pol := range pols {
		if pol.Spec.WAF != nil && isMatchingResourceRef(pol.Namespace, pol.Spec.WAF.ApPolicy, key) {
			policies = append(policies, pol)
		}
	}

	return policies
}

func getWAFPoliciesForAppProtectLogConf(pols []*conf_v1.Policy, key string) []*conf_v1.Policy {
	var policies []*conf_v1.Policy

	for _, pol := range pols {
		if pol.Spec.WAF != nil && pol.Spec.WAF.SecurityLog != nil && isMatchingResourceRef(pol.Namespace, pol.Spec.WAF.SecurityLog.ApLogConf, key) {
			policies = append(policies, pol)
		}
	}

	return policies
}

func isMatchingResourceRef(ownerNs, resRef, key string) bool {
	hasNamespace := strings.Contains(resRef, "/")
	if !hasNamespace {
		resRef = fmt.Sprintf("%v/%v", ownerNs, resRef)
	}
	return resRef == key
}

func (lbc *LoadBalancerController) createTransportServerEx(transportServer *conf_v1alpha1.TransportServer, listenerPort int) *configs.TransportServerEx {
	endpoints := make(map[string][]string)
	podsByIP := make(map[string]string)

	for _, u := range transportServer.Spec.Upstreams {
		podEndps, external, err := lbc.getEndpointsForUpstream(transportServer.Namespace, u.Service, uint16(u.Port))
		if err != nil {
			glog.Warningf("Error getting Endpoints for Upstream %v: %v", u.Name, err)
		}

		if external {
			glog.Warningf("ExternalName services are not yet supported in TransportServer upstreams")
		}

		// subselector is not supported yet in TransportServer upstreams. That's why we pass "nil" here
		endpointsKey := configs.GenerateEndpointsKey(transportServer.Namespace, u.Service, nil, uint16(u.Port))

		endps := getIPAddressesFromEndpoints(podEndps)
		endpoints[endpointsKey] = endps

		if lbc.isNginxPlus && lbc.isPrometheusEnabled {
			for _, endpoint := range podEndps {
				podsByIP[endpoint.Address] = endpoint.PodName
			}
		}
	}

	return &configs.TransportServerEx{
		ListenerPort:    listenerPort,
		TransportServer: transportServer,
		Endpoints:       endpoints,
		PodsByIP:        podsByIP,
	}
}

func (lbc *LoadBalancerController) getEndpointsForUpstream(namespace string, upstreamService string, upstreamPort uint16) (endps []podEndpoint, isExternal bool, err error) {
	svc, err := lbc.getServiceForUpstream(namespace, upstreamService, upstreamPort)
	if err != nil {
		return nil, false, fmt.Errorf("Error getting service %v: %w", upstreamService, err)
	}

	backend := &networking.IngressBackend{
		Service: &networking.IngressServiceBackend{
			Name: upstreamService,
			Port: networking.ServiceBackendPort{
				Number: int32(upstreamPort),
			},
		},
	}

	endps, isExternal, err = lbc.getEndpointsForIngressBackend(backend, svc)
	if err != nil {
		return nil, false, fmt.Errorf("Error retrieving endpoints for the service %v: %w", upstreamService, err)
	}

	return endps, isExternal, err
}

func (lbc *LoadBalancerController) getEndpointsForSubselector(namespace string, upstream conf_v1.Upstream) (endps []podEndpoint, err error) {
	svc, err := lbc.getServiceForUpstream(namespace, upstream.Service, upstream.Port)
	if err != nil {
		return nil, fmt.Errorf("Error getting service %v: %w", upstream.Service, err)
	}

	var targetPort int32

	for _, port := range svc.Spec.Ports {
		if port.Port == int32(upstream.Port) {
			targetPort, err = lbc.getTargetPort(port, svc)
			if err != nil {
				return nil, fmt.Errorf("Error determining target port for port %v in service %v: %w", upstream.Port, svc.Name, err)
			}
			break
		}
	}

	if targetPort == 0 {
		return nil, fmt.Errorf("No port %v in service %s", upstream.Port, svc.Name)
	}

	endps, err = lbc.getEndpointsForServiceWithSubselector(targetPort, upstream.Subselector, svc)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving endpoints for the service %v: %w", upstream.Service, err)
	}

	return endps, err
}

func (lbc *LoadBalancerController) getEndpointsForServiceWithSubselector(targetPort int32, subselector map[string]string, svc *api_v1.Service) (endps []podEndpoint, err error) {
	pods, err := lbc.podLister.ListByNamespace(svc.Namespace, labels.Merge(svc.Spec.Selector, subselector).AsSelector())
	if err != nil {
		return nil, fmt.Errorf("Error getting pods in namespace %v that match the selector %v: %w", svc.Namespace, labels.Merge(svc.Spec.Selector, subselector), err)
	}

	svcEps, err := lbc.endpointLister.GetServiceEndpoints(svc)
	if err != nil {
		glog.V(3).Infof("Error getting endpoints for service %s from the cache: %v", svc.Name, err)
		return nil, err
	}

	endps = getEndpointsBySubselectedPods(targetPort, pods, svcEps)
	return endps, nil
}

func getEndpointsBySubselectedPods(targetPort int32, pods []*api_v1.Pod, svcEps api_v1.Endpoints) (endps []podEndpoint) {
	for _, pod := range pods {
		for _, subset := range svcEps.Subsets {
			for _, port := range subset.Ports {
				if port.Port != targetPort {
					continue
				}
				for _, address := range subset.Addresses {
					if address.IP == pod.Status.PodIP {
						addr := ipv6SafeAddrPort(pod.Status.PodIP, targetPort)
						ownerType, ownerName := getPodOwnerTypeAndName(pod)
						podEnd := podEndpoint{
							Address: addr,
							PodName: getPodName(address.TargetRef),
							MeshPodOwner: configs.MeshPodOwner{
								OwnerType: ownerType,
								OwnerName: ownerName,
							},
						}
						endps = append(endps, podEnd)
					}
				}
			}
		}
	}
	return endps
}

func ipv6SafeAddrPort(addr string, port int32) string {
	return net.JoinHostPort(addr, strconv.Itoa(int(port)))
}

func getPodName(pod *api_v1.ObjectReference) string {
	if pod != nil {
		return pod.Name
	}
	return ""
}

func (lbc *LoadBalancerController) getHealthChecksForIngressBackend(backend *networking.IngressBackend, namespace string) *api_v1.Probe {
	svc, err := lbc.getServiceForIngressBackend(backend, namespace)
	if err != nil {
		glog.V(3).Infof("Error getting service %v: %v", backend.Service.Name, err)
		return nil
	}
	svcPort := lbc.getServicePortForIngressPort(backend.Service.Port, svc)
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
					if container.ReadinessProbe != nil && container.ReadinessProbe.ProbeHandler.HTTPGet != nil && container.ReadinessProbe.PeriodSeconds > 0 {
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

func (lbc *LoadBalancerController) getExternalEndpointsForIngressBackend(backend *networking.IngressBackend, svc *api_v1.Service) []podEndpoint {
	address := fmt.Sprintf("%s:%d", svc.Spec.ExternalName, backend.Service.Port.Number)
	endpoints := []podEndpoint{
		{
			Address: address,
			PodName: "",
		},
	}
	return endpoints
}

func (lbc *LoadBalancerController) getEndpointsForIngressBackend(backend *networking.IngressBackend, svc *api_v1.Service) (result []podEndpoint, isExternal bool, err error) {
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

	result, err = lbc.getEndpointsForPort(endps, backend.Service.Port, svc)
	if err != nil {
		glog.V(3).Infof("Error getting endpoints for service %s port %v: %v", svc.Name, configs.GetBackendPortAsString(backend.Service.Port), err)
		return nil, false, err
	}
	return result, false, nil
}

func (lbc *LoadBalancerController) getEndpointsForPort(endps api_v1.Endpoints, backendPort networking.ServiceBackendPort, svc *api_v1.Service) ([]podEndpoint, error) {
	var targetPort int32
	var err error

	for _, port := range svc.Spec.Ports {
		if (backendPort.Name == "" && port.Port == backendPort.Number) || port.Name == backendPort.Name {
			targetPort, err = lbc.getTargetPort(port, svc)
			if err != nil {
				return nil, fmt.Errorf("Error determining target port for port %v in Ingress: %w", backendPort, err)
			}
			break
		}
	}

	if targetPort == 0 {
		return nil, fmt.Errorf("No port %v in service %s", backendPort, svc.Name)
	}

	for _, subset := range endps.Subsets {
		for _, port := range subset.Ports {
			if port.Port == targetPort {
				var endpoints []podEndpoint
				for _, address := range subset.Addresses {
					addr := ipv6SafeAddrPort(address.IP, port.Port)
					podEnd := podEndpoint{
						Address: addr,
					}
					if address.TargetRef != nil {
						parentType, parentName := lbc.getPodOwnerTypeAndNameFromAddress(address.TargetRef.Namespace, address.TargetRef.Name)
						podEnd.OwnerType = parentType
						podEnd.OwnerName = parentName
						podEnd.PodName = address.TargetRef.Name
					}
					endpoints = append(endpoints, podEnd)
				}
				return endpoints, nil
			}
		}
	}

	return nil, fmt.Errorf("No endpoints for target port %v in service %s", targetPort, svc.Name)
}

func (lbc *LoadBalancerController) getPodOwnerTypeAndNameFromAddress(ns, name string) (parentType, parentName string) {
	obj, exists, err := lbc.podLister.GetByKey(fmt.Sprintf("%s/%s", ns, name))
	if err != nil {
		glog.Warningf("could not get pod by key %s/%s: %v", ns, name, err)
		return "", ""
	}
	if exists {
		pod := obj.(*api_v1.Pod)
		return getPodOwnerTypeAndName(pod)
	}
	return "", ""
}

func getPodOwnerTypeAndName(pod *api_v1.Pod) (parentType, parentName string) {
	parentType = "deployment"
	for _, owner := range pod.GetOwnerReferences() {
		parentName = owner.Name
		if owner.Controller != nil && *owner.Controller {
			if owner.Kind == "StatefulSet" || owner.Kind == "DaemonSet" {
				parentType = strings.ToLower(owner.Kind)
			}
			if owner.Kind == "ReplicaSet" && strings.HasSuffix(owner.Name, pod.Labels["pod-template-hash"]) {
				parentName = strings.TrimSuffix(owner.Name, "-"+pod.Labels["pod-template-hash"])
			}
		}
	}
	return parentType, parentName
}

func (lbc *LoadBalancerController) getServicePortForIngressPort(backendPort networking.ServiceBackendPort, svc *api_v1.Service) *api_v1.ServicePort {
	for _, port := range svc.Spec.Ports {
		if (backendPort.Name == "" && port.Port == backendPort.Number) || port.Name == backendPort.Name {
			return &port
		}
	}
	return nil
}

func (lbc *LoadBalancerController) getTargetPort(svcPort api_v1.ServicePort, svc *api_v1.Service) (int32, error) {
	if (svcPort.TargetPort == intstr.IntOrString{}) {
		return svcPort.Port, nil
	}

	if svcPort.TargetPort.Type == intstr.Int {
		return int32(svcPort.TargetPort.IntValue()), nil
	}

	pods, err := lbc.podLister.ListByNamespace(svc.Namespace, labels.Set(svc.Spec.Selector).AsSelector())
	if err != nil {
		return 0, fmt.Errorf("Error getting pod information: %w", err)
	}

	if len(pods) == 0 {
		return 0, fmt.Errorf("No pods of service %s", svc.Name)
	}

	pod := pods[0]

	portNum, err := findPort(pod, svcPort)
	if err != nil {
		return 0, fmt.Errorf("Error finding named port %v in pod %s: %w", svcPort, pod.Name, err)
	}

	return portNum, nil
}

func (lbc *LoadBalancerController) getServiceForUpstream(namespace string, upstreamService string, upstreamPort uint16) (*api_v1.Service, error) {
	backend := &networking.IngressBackend{
		Service: &networking.IngressServiceBackend{
			Name: upstreamService,
			Port: networking.ServiceBackendPort{
				Number: int32(upstreamPort),
			},
		},
	}
	return lbc.getServiceForIngressBackend(backend, namespace)
}

func (lbc *LoadBalancerController) getServiceForIngressBackend(backend *networking.IngressBackend, namespace string) (*api_v1.Service, error) {
	svcKey := namespace + "/" + backend.Service.Name
	svcObj, svcExists, err := lbc.svcLister.GetByKey(svcKey)
	if err != nil {
		return nil, err
	}

	if svcExists {
		return svcObj.(*api_v1.Service), nil
	}

	return nil, fmt.Errorf("service %s doesn't exist", svcKey)
}

// HasCorrectIngressClass checks if resource ingress class annotation (if exists) or ingressClass string for VS/VSR is matching with ingress controller class
func (lbc *LoadBalancerController) HasCorrectIngressClass(obj interface{}) bool {
	var class string
	switch obj := obj.(type) {
	case *conf_v1.VirtualServer:
		class = obj.Spec.IngressClass
	case *conf_v1.VirtualServerRoute:
		class = obj.Spec.IngressClass
	case *conf_v1alpha1.TransportServer:
		class = obj.Spec.IngressClass
	case *conf_v1.Policy:
		class = obj.Spec.IngressClass
	case *networking.Ingress:
		class = obj.Annotations[ingressClassKey]
		if class == "" && obj.Spec.IngressClassName != nil {
			class = *obj.Spec.IngressClassName
		} else {
			// the annotation takes precedence over the field
			glog.Warningln("Using the DEPRECATED annotation 'kubernetes.io/ingress.class'. The 'ingressClassName' field will be ignored.")
		}
		return class == lbc.ingressClass

	default:
		return false
	}

	return class == lbc.ingressClass || class == ""
}

// isHealthCheckEnabled checks if health checks are enabled so we can only query pods if enabled.
func (lbc *LoadBalancerController) isHealthCheckEnabled(ing *networking.Ingress) bool {
	if healthCheckEnabled, exists, err := configs.GetMapKeyAsBool(ing.Annotations, "nginx.com/health-checks", ing); exists {
		if err != nil {
			glog.Error(err)
		}
		return healthCheckEnabled
	}
	return false
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

func (lbc *LoadBalancerController) syncAppProtectPolicy(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing AppProtectPolicy %v", key)
	obj, polExists, err := lbc.appProtectPolicyLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []appprotect.Change
	var problems []appprotect.Problem

	if !polExists {
		glog.V(2).Infof("Deleting AppProtectPolicy: %v\n", key)

		changes, problems = lbc.appProtectConfiguration.DeletePolicy(key)
	} else {
		glog.V(2).Infof("Adding or Updating AppProtectPolicy: %v\n", key)

		changes, problems = lbc.appProtectConfiguration.AddOrUpdatePolicy(obj.(*unstructured.Unstructured))
	}

	lbc.processAppProtectChanges(changes)
	lbc.processAppProtectProblems(problems)
}

func (lbc *LoadBalancerController) syncAppProtectLogConf(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing AppProtectLogConf %v", key)
	obj, confExists, err := lbc.appProtectLogConfLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []appprotect.Change
	var problems []appprotect.Problem

	if !confExists {
		glog.V(2).Infof("Deleting AppProtectLogConf: %v\n", key)

		changes, problems = lbc.appProtectConfiguration.DeleteLogConf(key)
	} else {
		glog.V(2).Infof("Adding or Updating AppProtectLogConf: %v\n", key)

		changes, problems = lbc.appProtectConfiguration.AddOrUpdateLogConf(obj.(*unstructured.Unstructured))
	}

	lbc.processAppProtectChanges(changes)
	lbc.processAppProtectProblems(problems)
}

func (lbc *LoadBalancerController) syncAppProtectUserSig(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing AppProtectUserSig %v", key)
	obj, sigExists, err := lbc.appProtectUserSigLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var change appprotect.UserSigChange
	var problems []appprotect.Problem

	if !sigExists {
		glog.V(2).Infof("Deleting AppProtectUserSig: %v\n", key)

		change, problems = lbc.appProtectConfiguration.DeleteUserSig(key)
	} else {
		glog.V(2).Infof("Adding or Updating AppProtectUserSig: %v\n", key)

		change, problems = lbc.appProtectConfiguration.AddOrUpdateUserSig(obj.(*unstructured.Unstructured))
	}

	lbc.processAppProtectUserSigChange(change)
	lbc.processAppProtectProblems(problems)
}

func (lbc *LoadBalancerController) syncAppProtectDosPolicy(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing AppProtectDosPolicy %v", key)
	obj, polExists, err := lbc.appProtectDosPolicyLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []appprotectdos.Change
	var problems []appprotectdos.Problem

	if !polExists {
		glog.V(2).Infof("Deleting APDosPolicy: %v\n", key)
		changes, problems = lbc.dosConfiguration.DeletePolicy(key)
	} else {
		glog.V(2).Infof("Adding or Updating APDosPolicy: %v\n", key)
		changes, problems = lbc.dosConfiguration.AddOrUpdatePolicy(obj.(*unstructured.Unstructured))
	}

	lbc.processAppProtectDosChanges(changes)
	lbc.processAppProtectDosProblems(problems)
}

func (lbc *LoadBalancerController) syncAppProtectDosLogConf(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing APDosLogConf %v", key)
	obj, confExists, err := lbc.appProtectDosLogConfLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []appprotectdos.Change
	var problems []appprotectdos.Problem

	if !confExists {
		glog.V(2).Infof("Deleting APDosLogConf: %v\n", key)
		changes, problems = lbc.dosConfiguration.DeleteLogConf(key)
	} else {
		glog.V(2).Infof("Adding or Updating APDosLogConf: %v\n", key)
		changes, problems = lbc.dosConfiguration.AddOrUpdateLogConf(obj.(*unstructured.Unstructured))
	}

	lbc.processAppProtectDosChanges(changes)
	lbc.processAppProtectDosProblems(problems)
}

func (lbc *LoadBalancerController) syncDosProtectedResource(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing DosProtectedResource %v", key)
	obj, confExists, err := lbc.appProtectDosProtectedLister.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	var changes []appprotectdos.Change
	var problems []appprotectdos.Problem

	if confExists {
		glog.V(2).Infof("Adding or Updating DosProtectedResource: %v\n", key)
		changes, problems = lbc.dosConfiguration.AddOrUpdateDosProtectedResource(obj.(*v1beta1.DosProtectedResource))
	} else {
		glog.V(2).Infof("Deleting DosProtectedResource: %v\n", key)
		changes, problems = lbc.dosConfiguration.DeleteProtectedResource(key)
	}

	lbc.processAppProtectDosChanges(changes)
	lbc.processAppProtectDosProblems(problems)
}

// IsNginxReady returns ready status of NGINX
func (lbc *LoadBalancerController) IsNginxReady() bool {
	return lbc.isNginxReady
}

func (lbc *LoadBalancerController) addInternalRouteServer() {
	if lbc.internalRoutesEnabled {
		if err := lbc.configurator.AddInternalRouteConfig(); err != nil {
			glog.Warningf("failed to configure internal route server: %v", err)
		}
	}
}

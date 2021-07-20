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
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/spiffe/go-spiffe/workload"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/api"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/appprotect"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/upstreams"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"

	api_v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	core_v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"

	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/validation"
	k8s_nginx "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	k8s_nginx_informers "github.com/nginxinc/kubernetes-ingress/pkg/client/informers/externalversions"
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

// LoadBalancerController watches Kubernetes API and
// reconfigures NGINX via NginxController when needed
type LoadBalancerController struct {
	client                        kubernetes.Interface
	confClient                    k8s_nginx.Interface
	dynClient                     dynamic.Interface
	cacheSyncs                    []cache.InformerSynced
	sharedInformerFactory         informers.SharedInformerFactory
	confSharedInformerFactory     k8s_nginx_informers.SharedInformerFactory
	configMapController           cache.Controller
	dynInformerFactory            dynamicinformer.DynamicSharedInformerFactory
	globalConfigurationController cache.Controller
	ingressLinkInformer           cache.SharedIndexInformer
	syncQueue                     *taskQueue
	ctx                           context.Context
	cancel                        context.CancelFunc
	configurator                  *configs.Configurator
	isNginxPlus                   bool
	appProtectEnabled             bool
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
	enablePreviewPolicies         bool
	metricsCollector              collectors.ControllerCollector
	globalConfigurationValidator  *validation.GlobalConfigurationValidator
	transportServerValidator      *validation.TransportServerValidator
	spiffeController              *spiffeController
	internalRoutesEnabled         bool
	syncLock                      sync.Mutex
	isNginxReady                  bool
	isPrometheusEnabled           bool
	isLatencyMetricsEnabled       bool
	configuration                 *Configuration
	secretStore                   secrets.SecretStore
	appProtectConfiguration       appprotect.Configuration
	api                           *api.Apis
	upstreams                     *upstreams.Upstreams
}

var keyFunc = cache.DeletionHandlingMetaNamespaceKeyFunc

// NewLoadBalancerControllerInput holds the input needed to call NewLoadBalancerController.
type NewLoadBalancerControllerInput struct {
	KubeClient                   kubernetes.Interface
	ConfClient                   k8s_nginx.Interface
	DynClient                    dynamic.Interface
	ResyncPeriod                 time.Duration
	Namespace                    string
	NginxConfigurator            *configs.Configurator
	DefaultServerSecret          string
	AppProtectEnabled            bool
	IsNginxPlus                  bool
	IngressClass                 string
	UseIngressClassOnly          bool
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
	EnablePreviewPolicies        bool
	MetricsCollector             collectors.ControllerCollector
	GlobalConfigurationValidator *validation.GlobalConfigurationValidator
	TransportServerValidator     *validation.TransportServerValidator
	VirtualServerValidator       *validation.VirtualServerValidator
	SpireAgentAddress            string
	InternalRoutesEnabled        bool
	IsPrometheusEnabled          bool
	IsLatencyMetricsEnabled      bool
	IsTLSPassthroughEnabled      bool
}

// NewLoadBalancerController creates a controller
func NewLoadBalancerController(input NewLoadBalancerControllerInput) *LoadBalancerController {
	lbc := &LoadBalancerController{
		client:                       input.KubeClient,
		confClient:                   input.ConfClient,
		dynClient:                    input.DynClient,
		configurator:                 input.NginxConfigurator,
		defaultServerSecret:          input.DefaultServerSecret,
		appProtectEnabled:            input.AppProtectEnabled,
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
		enablePreviewPolicies:        input.EnablePreviewPolicies,
		metricsCollector:             input.MetricsCollector,
		globalConfigurationValidator: input.GlobalConfigurationValidator,
		transportServerValidator:     input.TransportServerValidator,
		internalRoutesEnabled:        input.InternalRoutesEnabled,
		isPrometheusEnabled:          input.IsPrometheusEnabled,
		isLatencyMetricsEnabled:      input.IsLatencyMetricsEnabled,
		api:                          api.NewApis(),
	}
	lbc.upstreams = upstreams.NewUpstreams(lbc.api)

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

	lbc.sharedInformerFactory = informers.NewSharedInformerFactoryWithOptions(lbc.client, input.ResyncPeriod, informers.WithNamespace(lbc.namespace))
	if lbc.appProtectEnabled {
		lbc.dynInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(lbc.dynClient, 0)
	}
	if lbc.areCustomResourcesEnabled {
		lbc.confSharedInformerFactory = k8s_nginx_informers.NewSharedInformerFactoryWithOptions(
			lbc.confClient,
			input.ResyncPeriod,
			k8s_nginx_informers.WithNamespace(lbc.namespace),
		)
	}

	addHandlers(lbc, &input)

	lbc.statusUpdater = &statusUpdater{
		client:              input.KubeClient,
		namespace:           input.ControllerNamespace,
		externalServiceName: input.ExternalServiceName,
		keyFunc:             keyFunc,
		confClient:          input.ConfClient,
		api:                 lbc.api,
	}

	lbc.configuration = NewConfiguration(
		lbc.HasCorrectIngressClass,
		input.IsNginxPlus,
		input.AppProtectEnabled,
		input.InternalRoutesEnabled,
		input.VirtualServerValidator,
		input.GlobalConfigurationValidator,
		input.TransportServerValidator,
		input.IsTLSPassthroughEnabled)

	lbc.appProtectConfiguration = appprotect.NewConfiguration()

	lbc.secretStore = secrets.NewLocalSecretStore(lbc.configurator)

	lbc.updateIngressMetrics()
	return lbc
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
	if lbc.leaderElector != nil {
		go lbc.leaderElector.Run(lbc.ctx)
	}

	go lbc.sharedInformerFactory.Start(lbc.ctx.Done())
	if lbc.configMapController != nil {
		go lbc.configMapController.Run(lbc.ctx.Done())
	}
	if lbc.areCustomResourcesEnabled {
		go lbc.confSharedInformerFactory.Start(lbc.ctx.Done())
	}
	if lbc.globalConfigurationController != nil {
		go lbc.globalConfigurationController.Run(lbc.ctx.Done())
	}
	if lbc.ingressLinkInformer != nil {
		go lbc.ingressLinkInformer.Run(lbc.ctx.Done())
	}
	if lbc.appProtectEnabled {
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

	endp, endpExists, err := lbc.api.Endpoints.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if !endpExists {
		return
	}

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

	cfmp, configExists, err := lbc.api.ConfigMaps.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}
	cfgParams := configs.NewDefaultConfigParams()

	if configExists {
		cfgParams = configs.ParseConfigMap(cfmp, lbc.isNginxPlus, lbc.appProtectEnabled)
		lbc.statusUpdater.SaveStatusFromExternalStatus(cfmp.Data["external-status-address"])
	}

	resources := lbc.configuration.GetResources()

	glog.V(3).Infof("Updating %v resources", len(resources))

	resourceExes := lbc.createExtendedResources(resources)

	warnings, updateErr := lbc.configurator.UpdateConfig(cfgParams, resourceExes.IngressExes, resourceExes.MergeableIngresses, resourceExes.VirtualServerExes)

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

	if configExists {
		lbc.recorder.Eventf(cfmp, eventType, eventTitle, "Configuration from %v was updated %s", key, eventWarningMessage)
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
	secretList := lbc.api.Secrets.List()
	glog.V(3).Infof("PreSync %d Secrets", len(secretList))

	for _, secret := range secretList {
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
	case virtualServerRoute:
		lbc.syncVirtualServerRoute(task)
		lbc.updateVirtualServerMetrics()
	case globalConfiguration:
		lbc.syncGlobalConfiguration(task)
	case transportserver:
		lbc.syncTransportServer(task)
	case policy:
		lbc.syncPolicy(task)
	case appProtectPolicy:
		lbc.syncAppProtectPolicy(task)
	case appProtectLogConf:
		lbc.syncAppProtectLogConf(task)
	case appProtectUserSig:
		lbc.syncAppProtectUserSig(task)
	case ingressLink:
		lbc.syncIngressLink(task)
	}

	if !lbc.isNginxReady && lbc.syncQueue.Len() == 0 {
		lbc.isNginxReady = true
		glog.V(3).Infof("NGINX is ready")
	}
}

func (lbc *LoadBalancerController) syncIngressLink(task task) {
	key := task.Key
	glog.V(2).Infof("Adding, Updating or Deleting IngressLink: %v", key)

	link, exists, err := lbc.api.IngressLinks.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	if !exists {
		// IngressLink got removed
		lbc.statusUpdater.ClearStatusFromIngressLink()
	} else {
		// IngressLink is added or updated

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
	pol, polExists, err := lbc.api.Policies.GetByKey(key)
	if err != nil {
		lbc.syncQueue.Requeue(task, err)
		return
	}

	glog.V(2).Infof("Adding, Updating or Deleting Policy: %v\n", key)

	if polExists {
		err := validation.ValidatePolicy(pol, lbc.isNginxPlus, lbc.enablePreviewPolicies, lbc.appProtectEnabled)
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
	ts, tsExists, err := lbc.api.TransportServers.GetByKey(key)
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
		changes, problems = lbc.configuration.AddOrUpdateTransportServer(ts)
	}

	lbc.processChanges(changes)
	lbc.processProblems(problems)
}

func (lbc *LoadBalancerController) syncGlobalConfiguration(task task) {
	key := task.Key
	gc, gcExists, err := lbc.api.GlobalConfigurations.GetByKey(key)
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

		lbc.recorder.Eventf(gc, eventType, eventTitle, eventMessage)
	}

	lbc.processProblems(problems)
}

func (lbc *LoadBalancerController) syncVirtualServer(task task) {
	key := task.Key
	vs, vsExists, err := lbc.api.VirtualServers.GetByKey(key)
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

				_, vsExists, err := lbc.api.VirtualServers.GetByKey(key)
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

				_, ingExists, err := lbc.api.Ingresses.GetByKeySafe(key)
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

				_, tsExists, err := lbc.api.TransportServers.GetByKey(key)
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

				warnings, deleteErr := lbc.configurator.DeleteAppProtectPolicy(namespace+"/"+name, resourceExes.IngressExes, resourceExes.MergeableIngresses, resourceExes.VirtualServerExes)

				lbc.updateResourcesStatusAndEvents(resources, warnings, deleteErr)

			case *appprotect.LogConfEx:
				namespace := impl.Obj.GetNamespace()
				name := impl.Obj.GetName()
				resources := lbc.configuration.FindResourcesForAppProtectLogConfAnnotation(namespace, name)

				for _, wafPol := range getWAFPoliciesForAppProtectLogConf(lbc.getAllPolicies(), namespace+"/"+name) {
					resources = append(resources, lbc.configuration.FindResourcesForPolicy(wafPol.Namespace, wafPol.Name)...)
				}

				resourceExes := lbc.createExtendedResources(resources)

				warnings, deleteErr := lbc.configurator.DeleteAppProtectLogConf(namespace+"/"+name, resourceExes.IngressExes, resourceExes.MergeableIngresses, resourceExes.VirtualServerExes)

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

		for _, wafPol := range getWAFPoliciesForAppProtectPolicy(lbc.getAllPolicies(), appprotect.GetNsName(poladd)) {
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

		polNsName := appprotect.GetNsName(poldel)
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
	vsr, exists, err := lbc.api.VirtualServerRoutes.GetByKey(key)
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
		changes, problems = lbc.configuration.AddOrUpdateVirtualServerRoute(vsr)
	}

	lbc.processChanges(changes)
	lbc.processProblems(problems)
}

func (lbc *LoadBalancerController) syncIngress(task task) {
	key := task.Key
	ing, ingExists, err := lbc.api.Ingresses.GetByKeySafe(key)
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

func (lbc *LoadBalancerController) syncService(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing service %v", key)

	service, exists, err := lbc.api.Services.GetByKey(key)
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
			lbc.statusUpdater.SaveStatusFromExternalService(service)
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
	return lbc.statusUpdater.IsExternalServiceForStatus(svc)
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

// reportCustomResourceStatusEnabled determines if we should attempt to report status for Custom resources.
func (lbc *LoadBalancerController) reportCustomResourceStatusEnabled() bool {
	if lbc.isLeaderElectionEnabled {
		return lbc.leaderElector != nil && lbc.leaderElector.IsLeader()
	}

	return true
}

func (lbc *LoadBalancerController) syncSecret(task task) {
	key := task.Key
	secret, secrExists, err := lbc.api.Secrets.GetByKey(key)
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

	glog.V(2).Infof("Found %v resources with Secret %v", len(resources), key)

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
	for _, vs := range lbc.api.VirtualServers.List() {
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
	for _, vsr := range lbc.api.VirtualServerRoutes.List() {
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
	for _, pol := range lbc.api.Policies.List() {
		err := validation.ValidatePolicy(pol, lbc.isNginxPlus, lbc.enablePreviewPolicies, lbc.appProtectEnabled)
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
	for _, ts := range lbc.api.TransportServers.List() {
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

func getIPAddressesFromEndpoints(endpoints []upstreams.PodEndpoint) []string {
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
	}

	ingEx.Endpoints = make(map[string][]string)
	ingEx.HealthChecks = make(map[string]*api_v1.Probe)
	ingEx.ExternalNameSvcs = make(map[string]bool)
	ingEx.PodsByIP = make(map[string]configs.PodInfo)

	if ing.Spec.Backend != nil {
		podEndps := []upstreams.PodEndpoint{}
		var external bool
		svc, err := lbc.upstreams.GetServiceForIngressBackend(ing.Spec.Backend, ing.Namespace)
		if err != nil {
			glog.V(3).Infof("Error getting service %v: %v", ing.Spec.Backend.ServiceName, err)
		} else {
			podEndps, external, err = lbc.upstreams.GetEndpointsForIngressBackend(ing.Spec.Backend, svc, lbc.isNginxPlus)
			if err == nil && external && lbc.isNginxPlus {
				ingEx.ExternalNameSvcs[svc.Name] = true
			}
		}

		if err != nil {
			glog.Warningf("Error retrieving endpoints for the service %v: %v", ing.Spec.Backend.ServiceName, err)
		}

		endps := getIPAddressesFromEndpoints(podEndps)

		// endps is empty if there was any error before this point
		ingEx.Endpoints[ing.Spec.Backend.ServiceName+ing.Spec.Backend.ServicePort.String()] = endps

		if lbc.isNginxPlus && lbc.isHealthCheckEnabled(ing) {
			healthCheck := lbc.upstreams.GetHealthChecksForIngressBackend(ing.Spec.Backend, ing.Namespace)
			if healthCheck != nil {
				ingEx.HealthChecks[ing.Spec.Backend.ServiceName+ing.Spec.Backend.ServicePort.String()] = healthCheck
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
			podEndps := []upstreams.PodEndpoint{}
			if validMinionPaths != nil && !validMinionPaths[path.Path] {
				glog.V(3).Infof("Skipping path %s for minion Ingress %s", path.Path, ing.Name)
				continue
			}

			var external bool
			svc, err := lbc.upstreams.GetServiceForIngressBackend(&path.Backend, ing.Namespace)
			if err != nil {
				glog.V(3).Infof("Error getting service %v: %v", &path.Backend.ServiceName, err)
			} else {
				podEndps, external, err = lbc.upstreams.GetEndpointsForIngressBackend(&path.Backend, svc, lbc.isNginxPlus)
				if err == nil && external && lbc.isNginxPlus {
					ingEx.ExternalNameSvcs[svc.Name] = true
				}
			}

			if err != nil {
				glog.Warningf("Error retrieving endpoints for the service %v: %v", path.Backend.ServiceName, err)
			}

			endps := getIPAddressesFromEndpoints(podEndps)

			// endps is empty if there was any error before this point
			ingEx.Endpoints[path.Backend.ServiceName+path.Backend.ServicePort.String()] = endps

			// Pull active health checks from k8 api
			if lbc.isNginxPlus && lbc.isHealthCheckEnabled(ing) {
				healthCheck := lbc.upstreams.GetHealthChecksForIngressBackend(&path.Backend, ing.Namespace)
				if healthCheck != nil {
					ingEx.HealthChecks[path.Backend.ServiceName+path.Backend.ServicePort.String()] = healthCheck
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
	logConfNsNs := appprotect.ParseResourceReferenceAnnotationList(ing.Namespace, ing.Annotations[configs.AppProtectLogConfAnnotation])
	if len(logDsts) != len(logConfNsNs) {
		return apLogs, fmt.Errorf("Error Validating App Protect Destination and Config for Ingress %v: LogConf and LogDestination must have equal number of items", ing.Name)
	}

	for _, logDst := range logDsts {
		err := appprotect.ValidateAppProtectLogDestination(logDst)
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
	polNsN := appprotect.ParseResourceReferenceAnnotation(ing.Namespace, ing.Annotations[configs.AppProtectPolicyAnnotation])

	apPolicy, err = lbc.appProtectConfiguration.GetAppResource(appprotect.PolicyGVK.Kind, polNsN)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving App Protect Policy for Ingress %v: %w", ing.Name, err)
	}

	return apPolicy, nil
}

func (lbc *LoadBalancerController) createVirtualServerEx(virtualServer *conf_v1.VirtualServer, virtualServerRoutes []*conf_v1.VirtualServerRoute) *configs.VirtualServerEx {
	virtualServerEx := configs.VirtualServerEx{
		VirtualServer: virtualServer,
		SecretRefs:    make(map[string]*secrets.SecretReference),
		ApPolRefs:     make(map[string]*unstructured.Unstructured),
		LogConfRefs:   make(map[string]*unstructured.Unstructured),
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

	endpoints := make(map[string][]string)
	externalNameSvcs := make(map[string]bool)
	podsByIP := make(map[string]configs.PodInfo)

	for _, u := range virtualServer.Spec.Upstreams {
		endpointsKey := configs.GenerateEndpointsKey(virtualServer.Namespace, u.Service, u.Subselector, u.Port)

		var endps []string
		if u.UseClusterIP {
			s, err := lbc.upstreams.GetServiceForUpstream(virtualServer.Namespace, u.Service, u.Port)
			if err != nil {
				glog.Warningf("Error getting Service for Upstream %v: %v", u.Service, err)
			} else {
				endps = append(endps, fmt.Sprintf("%s:%d", s.Spec.ClusterIP, u.Port))
			}

		} else {
			var podEndps []upstreams.PodEndpoint
			var err error

			if len(u.Subselector) > 0 {
				podEndps, err = lbc.upstreams.GetEndpointsForSubselector(virtualServer.Namespace, u)
			} else {
				var external bool
				podEndps, external, err = lbc.upstreams.GetEndpointsForUpstream(virtualServer.Namespace, u.Service, u.Port, lbc.isNginxPlus)

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
		}

		for _, u := range vsr.Spec.Upstreams {
			endpointsKey := configs.GenerateEndpointsKey(vsr.Namespace, u.Service, u.Subselector, u.Port)

			var endps []string
			if u.UseClusterIP {
				s, err := lbc.upstreams.GetServiceForUpstream(vsr.Namespace, u.Service, u.Port)
				if err != nil {
					glog.Warningf("Error getting Service for Upstream %v: %v", u.Service, err)
				} else {
					endps = append(endps, fmt.Sprintf("%s:%d", s.Spec.ClusterIP, u.Port))
				}

			} else {
				var podEndps []upstreams.PodEndpoint
				var err error
				if len(u.Subselector) > 0 {
					podEndps, err = lbc.upstreams.GetEndpointsForSubselector(vsr.Namespace, u)
				} else {
					var external bool
					podEndps, external, err = lbc.upstreams.GetEndpointsForUpstream(vsr.Namespace, u.Service, u.Port, lbc.isNginxPlus)

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

	for _, pol := range lbc.api.Policies.List() {
		err := validation.ValidatePolicy(pol, lbc.isNginxPlus, lbc.enablePreviewPolicies, lbc.appProtectEnabled)
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

		policy, exists, err := lbc.api.Policies.GetByKey(policyKey)
		if err != nil {
			errors = append(errors, fmt.Errorf("Failed to get policy %s: %w", policyKey, err))
			continue
		}

		if !exists {
			errors = append(errors, fmt.Errorf("Policy %s doesn't exist", policyKey))
			continue
		}

		err = validation.ValidatePolicy(policy, lbc.isNginxPlus, lbc.enablePreviewPolicies, lbc.appProtectEnabled)
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
		podEndps, external, err := lbc.upstreams.GetEndpointsForUpstream(transportServer.Namespace, u.Service, uint16(u.Port), lbc.isNginxPlus)
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

// HasCorrectIngressClass checks if resource ingress class annotation (if exists) or ingressClass string for VS/VSR is matching with ingress controller class
func (lbc *LoadBalancerController) HasCorrectIngressClass(obj interface{}) bool {
	var class string
	var isIngress bool
	switch obj := obj.(type) {
	case *conf_v1.VirtualServer:
		class = obj.Spec.IngressClass
	case *conf_v1.VirtualServerRoute:
		class = obj.Spec.IngressClass
	case *conf_v1alpha1.TransportServer:
		class = obj.Spec.IngressClass
	case *networking.Ingress:
		isIngress = true
		class = obj.Annotations[ingressClassKey]
		if class == "" && obj.Spec.IngressClassName != nil {
			class = *obj.Spec.IngressClassName
		} else {
			// the annotation takes precedence over the field
			glog.Warningln("Using the DEPRECATED annotation 'kubernetes.io/ingress.class'. The 'ingressClassName' field will be ignored.")
		}

	default:
		return false
	}

	// useIngressClassOnly only applies for Ingress resources
	if lbc.useIngressClassOnly && isIngress {
		return class == lbc.ingressClass
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
	obj, polExists, err := lbc.api.AppProtectPolicies.GetByKey(key)
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

		changes, problems = lbc.appProtectConfiguration.AddOrUpdatePolicy(obj)
	}

	lbc.processAppProtectChanges(changes)
	lbc.processAppProtectProblems(problems)
}

func (lbc *LoadBalancerController) syncAppProtectLogConf(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing AppProtectLogConf %v", key)
	obj, confExists, err := lbc.api.AppProtectLogConfs.GetByKey(key)
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

		changes, problems = lbc.appProtectConfiguration.AddOrUpdateLogConf(obj)
	}

	lbc.processAppProtectChanges(changes)
	lbc.processAppProtectProblems(problems)
}

func (lbc *LoadBalancerController) syncAppProtectUserSig(task task) {
	key := task.Key
	glog.V(3).Infof("Syncing AppProtectUserSig %v", key)
	obj, sigExists, err := lbc.api.AppProtectUserSigLister.GetByKey(key)
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

		change, problems = lbc.appProtectConfiguration.AddOrUpdateUserSig(obj)
	}

	lbc.processAppProtectUserSigChange(change)
	lbc.processAppProtectProblems(problems)
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

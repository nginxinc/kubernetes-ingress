package k8s

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/api"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/appprotect"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	api_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

func addHandlers(lbc *LoadBalancerController, input *NewLoadBalancerControllerInput) {
	addSecretHandler(lbc)
	addIngressHandler(lbc)
	addServiceHandler(lbc)
	addEndpointHandler(lbc)
	addPodHandler(lbc)

	if lbc.appProtectEnabled {
		addAppProtectPolicyHandler(lbc)
		addAppProtectLogConfHandler(lbc)
		addAppProtectUserSigHandler(lbc)
	}

	if lbc.areCustomResourcesEnabled {
		addVirtualServerHandler(lbc)
		addVirtualServerRouteHandler(lbc)
		addTransportServerHandler(lbc)
		addPolicyHandler(lbc)

		if input.GlobalConfiguration != "" {
			namespace, name, _ := ParseNamespaceName(input.GlobalConfiguration)
			addGlobalConfigurationHandler(lbc, namespace, name)
		}
	}

	if input.ConfigMaps != "" {
		namespace, name, err := ParseNamespaceName(input.ConfigMaps)
		if err != nil {
			glog.Warning(err)
		} else {
			addConfigMapHandler(lbc, namespace, name)
		}
	}

	if input.IngressLink != "" {
		addIngressLinkHandler(lbc, input.IngressLink)
	}

	if input.IsLeaderElectionEnabled {
		addLeaderHandler(lbc)
	}
}

func addLeaderHandler(lbc *LoadBalancerController) {
	handler := createLeaderHandler(lbc)
	var err error
	lbc.leaderElector, err = newLeaderElector(lbc.client, handler, lbc.controllerNamespace, lbc.leaderElectionLockName)
	if err != nil {
		glog.V(3).Infof("Error starting LeaderElection: %v", err)
	}
}

// addappProtectPolicyHandler creates dynamic informers for custom appprotect policy resource
func addAppProtectPolicyHandler(lbc *LoadBalancerController) {
	handlers := createAppProtectPolicyHandlers(lbc.syncQueue)
	informer := lbc.dynInformerFactory.ForResource(appprotect.PolicyGVR).Informer()
	informer.AddEventHandler(handlers)
	lbc.api.AppProtectPolicies = api.NewAppProtectPolicies(informer.GetStore())
	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addappProtectLogConfHandler creates dynamic informer for custom appprotect logging config resource
func addAppProtectLogConfHandler(lbc *LoadBalancerController) {
	handlers := createAppProtectLogConfHandlers(lbc.syncQueue)
	informer := lbc.dynInformerFactory.ForResource(appprotect.LogConfGVR).Informer()
	informer.AddEventHandler(handlers)
	lbc.api.AppProtectLogConfs = api.NewAppProtectLogConfs(informer.GetStore())
	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

// addappProtectUserSigHandler creates dynamic informer for custom appprotect user defined signature resource
func addAppProtectUserSigHandler(lbc *LoadBalancerController) {
	handlers := createAppProtectUserSigHandlers(lbc.syncQueue)
	informer := lbc.dynInformerFactory.ForResource(appprotect.UserSigGVR).Informer()
	informer.AddEventHandler(handlers)
	lbc.api.AppProtectUserSigLister = api.NewAppProtectUserSigs(informer.GetStore())
	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addSecretHandler(lbc *LoadBalancerController) {
	handler := createSecretHandlers(lbc.syncQueue)
	informer := lbc.sharedInformerFactory.Core().V1().Secrets().Informer()
	informer.AddEventHandler(handler)
	lbc.api.Secrets = api.NewSecrets(informer.GetStore())
	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addServiceHandler(lbc *LoadBalancerController) {
	handlers := createServiceHandlers(lbc.syncQueue, lbc.statusUpdater)
	informer := lbc.sharedInformerFactory.Core().V1().Services().Informer()
	informer.AddEventHandler(handlers)
	lbc.api.Services = api.NewServices(informer.GetStore())

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addIngressHandler(lbc *LoadBalancerController) {
	handlers := createIngressHandlers(lbc.syncQueue)
	informer := lbc.sharedInformerFactory.Networking().V1beta1().Ingresses().Informer()
	informer.AddEventHandler(handlers)
	lbc.api.Ingresses = api.NewIngresses(informer.GetStore())

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addEndpointHandler(lbc *LoadBalancerController) {
	handlers := createEndpointHandlers(lbc.syncQueue)
	informer := lbc.sharedInformerFactory.Core().V1().Endpoints().Informer()
	informer.AddEventHandler(handlers)
	lbc.api.Endpoints = api.NewEndpoints(informer.GetStore())

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addConfigMapHandler(lbc *LoadBalancerController, namespace string, name string) {
	handlers := createConfigMapHandlers(lbc.syncQueue, name)
	store, controller := cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.client.CoreV1().RESTClient(),
			"configmaps",
			namespace,
			fields.Everything()),
		&api_v1.ConfigMap{},
		lbc.resync,
		handlers,
	)
	lbc.configMapController = controller
	lbc.api.ConfigMaps = api.NewConfigMaps(store)
	lbc.cacheSyncs = append(lbc.cacheSyncs, lbc.configMapController.HasSynced)
}

func addPodHandler(lbc *LoadBalancerController) {
	informer := lbc.sharedInformerFactory.Core().V1().Pods().Informer()
	lbc.api.Pods = api.NewPods(informer.GetIndexer())
	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addVirtualServerHandler(lbc *LoadBalancerController) {
	handlers := createVirtualServerHandlers(lbc.syncQueue)
	informer := lbc.confSharedInformerFactory.K8s().V1().VirtualServers().Informer()
	informer.AddEventHandler(handlers)
	lbc.api.VirtualServers = api.NewVirtualServers(informer.GetStore())

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addVirtualServerRouteHandler(lbc *LoadBalancerController) {
	handlers := createVirtualServerRouteHandlers(lbc.syncQueue)
	informer := lbc.confSharedInformerFactory.K8s().V1().VirtualServerRoutes().Informer()
	informer.AddEventHandler(handlers)
	lbc.api.VirtualServerRoutes = api.NewVirtualServerRoutes(informer.GetStore())

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addPolicyHandler(lbc *LoadBalancerController) {
	handlers := createPolicyHandlers(lbc.syncQueue)
	informer := lbc.confSharedInformerFactory.K8s().V1().Policies().Informer()
	informer.AddEventHandler(handlers)
	lbc.api.Policies = api.NewPolicies(informer.GetStore())

	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addGlobalConfigurationHandler(lbc *LoadBalancerController, namespace string, name string) {
	handlers := createGlobalConfigurationHandlers(lbc.syncQueue)
	store, controller := cache.NewInformer(
		cache.NewListWatchFromClient(
			lbc.confClient.K8sV1alpha1().RESTClient(),
			"globalconfigurations",
			namespace,
			fields.Set{"metadata.name": name}.AsSelector()),
		&conf_v1alpha1.GlobalConfiguration{},
		lbc.resync,
		handlers,
	)
	lbc.globalConfigurationController = controller
	lbc.api.GlobalConfigurations = api.NewGlobalConfigurations(store)
	lbc.cacheSyncs = append(lbc.cacheSyncs, lbc.globalConfigurationController.HasSynced)
}

func addTransportServerHandler(lbc *LoadBalancerController) {
	handlers := createTransportServerHandlers(lbc.syncQueue)
	informer := lbc.confSharedInformerFactory.K8s().V1alpha1().TransportServers().Informer()
	informer.AddEventHandler(handlers)
	lbc.api.TransportServers = api.NewTransportServers(informer.GetStore())
	lbc.cacheSyncs = append(lbc.cacheSyncs, informer.HasSynced)
}

func addIngressLinkHandler(lbc *LoadBalancerController, name string) {
	handlers := createIngressLinkHandlers(lbc.syncQueue)
	optionsModifier := func(options *meta_v1.ListOptions) {
		options.FieldSelector = fields.Set{"metadata.name": name}.String()
	}

	informer := dynamicinformer.NewFilteredDynamicInformer(lbc.dynClient, ingressLinkGVR, lbc.controllerNamespace, lbc.resync,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, optionsModifier)

	informer.Informer().AddEventHandlerWithResyncPeriod(handlers, lbc.resync)

	lbc.ingressLinkInformer = informer.Informer()
	lbc.api.IngressLinks = api.NewIngressLinks(informer.Informer().GetStore())

	lbc.cacheSyncs = append(lbc.cacheSyncs, lbc.ingressLinkInformer.HasSynced)
}

// createConfigMapHandlers builds the handler funcs for config maps
func createConfigMapHandlers(q Queue, name string) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			configMap := obj.(*v1.ConfigMap)
			if configMap.Name == name {
				glog.V(3).Infof("Adding ConfigMap: %v", configMap.Name)
				q.Enqueue(obj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			configMap, isConfigMap := obj.(*v1.ConfigMap)
			if !isConfigMap {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				configMap, ok = deletedState.Obj.(*v1.ConfigMap)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-ConfigMap object: %v", deletedState.Obj)
					return
				}
			}
			if configMap.Name == name {
				glog.V(3).Infof("Removing ConfigMap: %v", configMap.Name)
				q.Enqueue(obj)
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				configMap := cur.(*v1.ConfigMap)
				if configMap.Name == name {
					glog.V(3).Infof("ConfigMap %v changed, syncing", cur.(*v1.ConfigMap).Name)
					q.Enqueue(cur)
				}
			}
		},
	}
}

// createEndpointHandlers builds the handler funcs for endpoints
func createEndpointHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			endpoint := obj.(*v1.Endpoints)
			glog.V(3).Infof("Adding endpoints: %v", endpoint.Name)
			q.Enqueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			endpoint, isEndpoint := obj.(*v1.Endpoints)
			if !isEndpoint {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				endpoint, ok = deletedState.Obj.(*v1.Endpoints)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Endpoints object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing endpoints: %v", endpoint.Name)
			q.Enqueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Endpoints %v changed, syncing", cur.(*v1.Endpoints).Name)
				q.Enqueue(cur)
			}
		},
	}
}

// createIngressHandlers builds the handler funcs for ingresses
func createIngressHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ingress := obj.(*networking.Ingress)
			glog.V(3).Infof("Adding Ingress: %v", ingress.Name)
			q.Enqueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			ingress, isIng := obj.(*networking.Ingress)
			if !isIng {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				ingress, ok = deletedState.Obj.(*networking.Ingress)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Ingress object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing Ingress: %v", ingress.Name)
			q.Enqueue(obj)
		},
		UpdateFunc: func(old, current interface{}) {
			c := current.(*networking.Ingress)
			o := old.(*networking.Ingress)
			if hasChanges(o, c) {
				glog.V(3).Infof("Ingress %v changed, syncing", c.Name)
				q.Enqueue(c)
			}
		},
	}
}

// createSecretHandlers builds the handler funcs for secrets
func createSecretHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret := obj.(*v1.Secret)
			if !secrets.IsSupportedSecretType(secret.Type) {
				glog.V(3).Infof("Ignoring Secret %v of unsupported type %v", secret.Name, secret.Type)
				return
			}
			glog.V(3).Infof("Adding Secret: %v", secret.Name)
			q.Enqueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			secret, isSecr := obj.(*v1.Secret)
			if !isSecr {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				secret, ok = deletedState.Obj.(*v1.Secret)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Secret object: %v", deletedState.Obj)
					return
				}
			}
			if !secrets.IsSupportedSecretType(secret.Type) {
				glog.V(3).Infof("Ignoring Secret %v of unsupported type %v", secret.Name, secret.Type)
				return
			}

			glog.V(3).Infof("Removing Secret: %v", secret.Name)
			q.Enqueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			// A secret cannot change its type. That's why we only need to check the type of the current secret.
			curSecret := cur.(*v1.Secret)
			if !secrets.IsSupportedSecretType(curSecret.Type) {
				glog.V(3).Infof("Ignoring Secret %v of unsupported type %v", curSecret.Name, curSecret.Type)
				return
			}

			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Secret %v changed, syncing", cur.(*v1.Secret).Name)
				q.Enqueue(cur)
			}
		},
	}
}

// createServiceHandlers builds the handler funcs for services.
//
// In the update handlers below we catch two cases:
// (1) the service is the external service
// (2) the service had a change like a change of the port field of a service port (for such a change Kubernetes doesn't
// update the corresponding endpoints resource, that we monitor as well)
// or a change of the externalName field of an ExternalName service.
//
// In both cases we enqueue the service to be processed by syncService
func createServiceHandlers(q Queue, su *statusUpdater) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)

			glog.V(3).Infof("Adding service: %v", svc.Name)
			q.Enqueue(svc)
		},
		DeleteFunc: func(obj interface{}) {
			svc, isSvc := obj.(*v1.Service)
			if !isSvc {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				svc, ok = deletedState.Obj.(*v1.Service)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Service object: %v", deletedState.Obj)
					return
				}
			}

			glog.V(3).Infof("Removing service: %v", svc.Name)
			q.Enqueue(svc)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				curSvc := cur.(*v1.Service)
				if su.IsExternalServiceForStatus(curSvc) {
					q.Enqueue(curSvc)
					return
				}
				oldSvc := old.(*v1.Service)
				if hasServiceChanges(oldSvc, curSvc) {
					glog.V(3).Infof("Service %v changed, syncing", curSvc.Name)
					q.Enqueue(curSvc)
				}
			}
		},
	}
}

type portSort []v1.ServicePort

func (a portSort) Len() int {
	return len(a)
}

func (a portSort) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a portSort) Less(i, j int) bool {
	if a[i].Name == a[j].Name {
		return a[i].Port < a[j].Port
	}
	return a[i].Name < a[j].Name
}

// hasServicedChanged checks if the service has changed based on custom rules we define (eg. port).
func hasServiceChanges(oldSvc, curSvc *v1.Service) bool {
	if hasServicePortChanges(oldSvc.Spec.Ports, curSvc.Spec.Ports) {
		return true
	}
	if hasServiceExternalNameChanges(oldSvc, curSvc) {
		return true
	}
	return false
}

// hasServiceExternalNameChanges only compares Service.Spec.Externalname for Type ExternalName services.
func hasServiceExternalNameChanges(oldSvc, curSvc *v1.Service) bool {
	return curSvc.Spec.Type == v1.ServiceTypeExternalName && oldSvc.Spec.ExternalName != curSvc.Spec.ExternalName
}

// hasServicePortChanges only compares ServicePort.Name and .Port.
func hasServicePortChanges(oldServicePorts []v1.ServicePort, curServicePorts []v1.ServicePort) bool {
	if len(oldServicePorts) != len(curServicePorts) {
		return true
	}

	sort.Sort(portSort(oldServicePorts))
	sort.Sort(portSort(curServicePorts))

	for i := range oldServicePorts {
		if oldServicePorts[i].Port != curServicePorts[i].Port ||
			oldServicePorts[i].Name != curServicePorts[i].Name {
			return true
		}
	}
	return false
}

func createVirtualServerHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			vs := obj.(*conf_v1.VirtualServer)
			glog.V(3).Infof("Adding VirtualServer: %v", vs.Name)
			q.Enqueue(vs)
		},
		DeleteFunc: func(obj interface{}) {
			vs, isVs := obj.(*conf_v1.VirtualServer)
			if !isVs {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				vs, ok = deletedState.Obj.(*conf_v1.VirtualServer)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-VirtualServer object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing VirtualServer: %v", vs.Name)
			q.Enqueue(vs)
		},
		UpdateFunc: func(old, cur interface{}) {
			curVs := cur.(*conf_v1.VirtualServer)
			oldVs := old.(*conf_v1.VirtualServer)
			if !reflect.DeepEqual(oldVs.Spec, curVs.Spec) {
				glog.V(3).Infof("VirtualServer %v changed, syncing", curVs.Name)
				q.Enqueue(curVs)
			}
		},
	}
}

func createVirtualServerRouteHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			vsr := obj.(*conf_v1.VirtualServerRoute)
			glog.V(3).Infof("Adding VirtualServerRoute: %v", vsr.Name)
			q.Enqueue(vsr)
		},
		DeleteFunc: func(obj interface{}) {
			vsr, isVsr := obj.(*conf_v1.VirtualServerRoute)
			if !isVsr {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				vsr, ok = deletedState.Obj.(*conf_v1.VirtualServerRoute)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-VirtualServerRoute object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing VirtualServerRoute: %v", vsr.Name)
			q.Enqueue(vsr)
		},
		UpdateFunc: func(old, cur interface{}) {
			curVsr := cur.(*conf_v1.VirtualServerRoute)
			oldVsr := old.(*conf_v1.VirtualServerRoute)
			if !reflect.DeepEqual(oldVsr.Spec, curVsr.Spec) {
				glog.V(3).Infof("VirtualServerRoute %v changed, syncing", curVsr.Name)
				q.Enqueue(curVsr)
			}
		},
	}
}

func createGlobalConfigurationHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			gc := obj.(*conf_v1alpha1.GlobalConfiguration)
			glog.V(3).Infof("Adding GlobalConfiguration: %v", gc.Name)
			q.Enqueue(gc)
		},
		DeleteFunc: func(obj interface{}) {
			gc, isGc := obj.(*conf_v1alpha1.GlobalConfiguration)
			if !isGc {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				gc, ok = deletedState.Obj.(*conf_v1alpha1.GlobalConfiguration)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-GlobalConfiguration object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing GlobalConfiguration: %v", gc.Name)
			q.Enqueue(gc)
		},
		UpdateFunc: func(old, cur interface{}) {
			curGc := cur.(*conf_v1alpha1.GlobalConfiguration)
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("GlobalConfiguration %v changed, syncing", curGc.Name)
				q.Enqueue(curGc)
			}
		},
	}
}

func createTransportServerHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ts := obj.(*conf_v1alpha1.TransportServer)
			glog.V(3).Infof("Adding TransportServer: %v", ts.Name)
			q.Enqueue(ts)
		},
		DeleteFunc: func(obj interface{}) {
			ts, isTs := obj.(*conf_v1alpha1.TransportServer)
			if !isTs {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				ts, ok = deletedState.Obj.(*conf_v1alpha1.TransportServer)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-TransportServer object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing TransportServer: %v", ts.Name)
			q.Enqueue(ts)
		},
		UpdateFunc: func(old, cur interface{}) {
			curTs := cur.(*conf_v1alpha1.TransportServer)
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("TransportServer %v changed, syncing", curTs.Name)
				q.Enqueue(curTs)
			}
		},
	}
}

func createPolicyHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pol := obj.(*conf_v1.Policy)
			glog.V(3).Infof("Adding Policy: %v", pol.Name)
			q.Enqueue(pol)
		},
		DeleteFunc: func(obj interface{}) {
			pol, isPol := obj.(*conf_v1.Policy)
			if !isPol {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				pol, ok = deletedState.Obj.(*conf_v1.Policy)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Policy object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing Policy: %v", pol.Name)
			q.Enqueue(pol)
		},
		UpdateFunc: func(old, cur interface{}) {
			curPol := cur.(*conf_v1.Policy)
			oldPol := old.(*conf_v1.Policy)
			if !reflect.DeepEqual(oldPol.Spec, curPol.Spec) {
				glog.V(3).Infof("Policy %v changed, syncing", curPol.Name)
				q.Enqueue(curPol)
			}
		},
	}
}

func createIngressLinkHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			link := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding IngressLink: %v", link.GetName())
			q.Enqueue(link)
		},
		DeleteFunc: func(obj interface{}) {
			link, isUnstructured := obj.(*unstructured.Unstructured)

			if !isUnstructured {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				link, ok = deletedState.Obj.(*unstructured.Unstructured)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Unstructured object: %v", deletedState.Obj)
					return
				}
			}

			glog.V(3).Infof("Removing IngressLink: %v", link.GetName())
			q.Enqueue(link)
		},
		UpdateFunc: func(old, cur interface{}) {
			oldLink := old.(*unstructured.Unstructured)
			curLink := cur.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldLink, curLink)
			if err != nil {
				glog.V(3).Infof("Error when comparing IngressLinks: %v", err)
				q.Enqueue(curLink)
			}
			if different {
				glog.V(3).Infof("IngressLink %v changed, syncing", oldLink.GetName())
				q.Enqueue(curLink)
			}
		},
	}
}

func createAppProtectPolicyHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pol := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding AppProtectPolicy: %v", pol.GetName())
			q.Enqueue(pol)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			oldPol := oldObj.(*unstructured.Unstructured)
			newPol := obj.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldPol, newPol)
			if err != nil {
				glog.V(3).Infof("Error when comparing policy %v", err)
				q.Enqueue(newPol)
			}
			if different {
				glog.V(3).Infof("ApPolicy %v changed, syncing", oldPol.GetName())
				q.Enqueue(newPol)
			}
		},
		DeleteFunc: func(obj interface{}) {
			q.Enqueue(obj)
		},
	}
	return handlers
}

// areResourcesDifferent returns true if the resources are different based on their spec.
func areResourcesDifferent(oldresource, resource *unstructured.Unstructured) (bool, error) {
	oldSpec, found, err := unstructured.NestedMap(oldresource.Object, "spec")
	if !found {
		glog.V(3).Infof("Warning, oldspec has unexpected format")
	}
	if err != nil {
		return false, err
	}
	spec, found, err := unstructured.NestedMap(resource.Object, "spec")
	if !found {
		return false, fmt.Errorf("Error, spec has unexpected format")
	}
	if err != nil {
		return false, err
	}
	eq := reflect.DeepEqual(oldSpec, spec)
	if eq {
		glog.V(3).Infof("New spec of %v same as old spec", oldresource.GetName())
	}
	return !eq, nil
}

func createAppProtectLogConfHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			conf := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding AppProtectLogConf: %v", conf.GetName())
			q.Enqueue(conf)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			oldConf := oldObj.(*unstructured.Unstructured)
			newConf := obj.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldConf, newConf)
			if err != nil {
				glog.V(3).Infof("Error when comparing LogConfs %v", err)
				q.Enqueue(newConf)
			}
			if different {
				glog.V(3).Infof("ApLogConf %v changed, syncing", oldConf.GetName())
				q.Enqueue(newConf)
			}
		},
		DeleteFunc: func(obj interface{}) {
			q.Enqueue(obj)
		},
	}
	return handlers
}

func createAppProtectUserSigHandlers(q Queue) cache.ResourceEventHandlerFuncs {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sig := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding AppProtectUserSig: %v", sig.GetName())
			q.Enqueue(sig)
		},
		UpdateFunc: func(oldObj, obj interface{}) {
			oldSig := oldObj.(*unstructured.Unstructured)
			newSig := obj.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldSig, newSig)
			if err != nil {
				glog.V(3).Infof("Error when comparing UserSigs %v", err)
				q.Enqueue(newSig)
			}
			if different {
				glog.V(3).Infof("ApUserSig %v changed, syncing", oldSig.GetName())
				q.Enqueue(newSig)
			}
		},
		DeleteFunc: func(obj interface{}) {
			q.Enqueue(obj)
		},
	}
	return handlers
}

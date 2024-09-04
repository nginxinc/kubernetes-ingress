package k8s

import (
	"github.com/golang/glog"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

func createIngressLinkHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			link := obj.(*unstructured.Unstructured)
			glog.V(3).Infof("Adding IngressLink: %v", link.GetName())
			lbc.AddSyncQueue(link)
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
			lbc.AddSyncQueue(link)
		},
		UpdateFunc: func(old, cur interface{}) {
			oldLink := old.(*unstructured.Unstructured)
			curLink := cur.(*unstructured.Unstructured)
			different, err := areResourcesDifferent(oldLink, curLink)
			if err != nil {
				glog.V(3).Infof("Error when comparing IngressLinks: %v", err)
				lbc.AddSyncQueue(curLink)
			}
			if different {
				glog.V(3).Infof("IngressLink %v changed, syncing", oldLink.GetName())
				lbc.AddSyncQueue(curLink)
			}
		},
	}
}

func (lbc *LoadBalancerController) addIngressLinkHandler(handlers cache.ResourceEventHandlerFuncs, name string) {
	optionsModifier := func(options *meta_v1.ListOptions) {
		options.FieldSelector = fields.Set{"metadata.name": name}.String()
	}

	informer := dynamicinformer.NewFilteredDynamicInformer(lbc.dynClient, ingressLinkGVR, lbc.controllerNamespace, lbc.resync,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, optionsModifier)

	informer.Informer().AddEventHandlerWithResyncPeriod(handlers, lbc.resync) //nolint:errcheck,gosec

	lbc.ingressLinkInformer = informer.Informer()
	lbc.ingressLinkLister = informer.Informer().GetStore()

	lbc.cacheSyncs = append(lbc.cacheSyncs, lbc.ingressLinkInformer.HasSynced)
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

package k8s

import (
	"fmt"
	"reflect"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func createGlobalConfigurationHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			gc := obj.(*conf_v1.GlobalConfiguration)
			glog.V(3).Infof("Adding GlobalConfiguration: %v", gc.Name)
			lbc.AddSyncQueue(gc)
		},
		DeleteFunc: func(obj interface{}) {
			gc, isGc := obj.(*conf_v1.GlobalConfiguration)
			if !isGc {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				gc, ok = deletedState.Obj.(*conf_v1.GlobalConfiguration)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-GlobalConfiguration object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing GlobalConfiguration: %v", gc.Name)
			lbc.AddSyncQueue(gc)
		},
		UpdateFunc: func(old, cur interface{}) {
			curGc := cur.(*conf_v1.GlobalConfiguration)
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("GlobalConfiguration %v changed, syncing", curGc.Name)
				lbc.AddSyncQueue(curGc)
			}
		},
	}
}

func (lbc *LoadBalancerController) addGlobalConfigurationHandler(handlers cache.ResourceEventHandlerFuncs, namespace string, name string) {
	options := cache.InformerOptions{
		ListerWatcher: cache.NewListWatchFromClient(
			lbc.confClient.K8sV1().RESTClient(),
			"globalconfigurations",
			namespace,
			fields.Set{"metadata.name": name}.AsSelector()),
		ObjectType:   &conf_v1.GlobalConfiguration{},
		ResyncPeriod: lbc.resync,
		Handler:      handlers,
	}
	lbc.globalConfigurationLister, lbc.globalConfigurationController = cache.NewInformerWithOptions(options)
	lbc.cacheSyncs = append(lbc.cacheSyncs, lbc.globalConfigurationController.HasSynced)
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

		gc := obj.(*conf_v1.GlobalConfiguration)
		changes, problems, validationErr = lbc.configuration.AddOrUpdateGlobalConfiguration(gc)
	}

	updateErr := lbc.processChangesFromGlobalConfiguration(changes)

	if gcExists {
		eventTitle := "Updated"
		eventType := api_v1.EventTypeNormal
		eventMessage := fmt.Sprintf("GlobalConfiguration %s was added or updated", key)

		if validationErr != nil {
			eventTitle = "AddedOrUpdatedWithError"
			eventType = api_v1.EventTypeWarning
			eventMessage = fmt.Sprintf("GlobalConfiguration %s is updated with errors: %v", key, validationErr)
		}

		if updateErr != nil {
			eventTitle += "WithError"
			eventType = api_v1.EventTypeWarning
			eventMessage = fmt.Sprintf("%s; with reload error: %v", eventMessage, updateErr)
		}

		gc := obj.(*conf_v1.GlobalConfiguration)
		lbc.recorder.Eventf(gc, eventType, eventTitle, eventMessage)
	}

	lbc.processProblems(problems)
}

// processChangesFromGlobalConfiguration processes changes that come from updates to the GlobalConfiguration resource.
// Such changes need to be processed at once to prevent any inconsistencies in the generated NGINX config.
func (lbc *LoadBalancerController) processChangesFromGlobalConfiguration(changes []ResourceChange) error {
	var updatedTSExes []*configs.TransportServerEx
	var updatedVSExes []*configs.VirtualServerEx
	var deletedTSKeys []string
	var deletedVSKeys []string

	var updatedResources []Resource

	for _, c := range changes {
		switch impl := c.Resource.(type) {
		case *VirtualServerConfiguration:
			if c.Op == AddOrUpdate {
				vsEx := lbc.createVirtualServerEx(impl.VirtualServer, impl.VirtualServerRoutes)

				updatedVSExes = append(updatedVSExes, vsEx)
				updatedResources = append(updatedResources, impl)
			} else if c.Op == Delete {
				key := getResourceKey(&impl.VirtualServer.ObjectMeta)

				deletedVSKeys = append(deletedVSKeys, key)
			}
		case *TransportServerConfiguration:
			if c.Op == AddOrUpdate {
				tsEx := lbc.createTransportServerEx(impl.TransportServer, impl.ListenerPort)

				updatedTSExes = append(updatedTSExes, tsEx)
				updatedResources = append(updatedResources, impl)
			} else if c.Op == Delete {
				key := getResourceKey(&impl.TransportServer.ObjectMeta)

				deletedTSKeys = append(deletedTSKeys, key)
			}
		}
	}

	var updateErr error

	if len(updatedTSExes) > 0 || len(deletedTSKeys) > 0 {
		tsUpdateErrs := lbc.configurator.UpdateTransportServers(updatedTSExes, deletedTSKeys)

		if len(tsUpdateErrs) > 0 {
			updateErr = fmt.Errorf("errors received from updating TransportServers after GlobalConfiguration change: %v", tsUpdateErrs)
		}
	}

	if len(updatedVSExes) > 0 || len(deletedVSKeys) > 0 {
		vsUpdateErrs := lbc.configurator.UpdateVirtualServers(updatedVSExes, deletedVSKeys)

		if len(vsUpdateErrs) > 0 {
			updateErr = fmt.Errorf("errors received from updating VirtualSrvers after GlobalConfiguration change: %v", vsUpdateErrs)
		}
	}

	lbc.updateResourcesStatusAndEvents(updatedResources, configs.Warnings{}, updateErr)

	return updateErr
}

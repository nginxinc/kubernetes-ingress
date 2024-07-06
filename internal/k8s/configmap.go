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
	"reflect"

	"github.com/golang/glog"
	api_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// createConfigMapHandlers builds the handler funcs for config maps
func createConfigMapHandlers(lbc *LoadBalancerController, name string) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			configMap := obj.(*v1.ConfigMap)
			if configMap.Name == name {
				glog.V(3).Infof("Adding ConfigMap: %v", configMap.Name)
				lbc.AddSyncQueue(obj)
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
				lbc.AddSyncQueue(obj)
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				configMap := cur.(*v1.ConfigMap)
				if configMap.Name == name {
					glog.V(3).Infof("ConfigMap %v changed, syncing", cur.(*v1.ConfigMap).Name)
					lbc.AddSyncQueue(cur)
				}
			}
		},
	}
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
		externalStatusAddress, exists := lbc.configMap.Data["external-status-address"]
		if exists {
			lbc.statusUpdater.SaveStatusFromExternalStatus(externalStatusAddress)
		}
	} else {
		lbc.configMap = nil
	}

	if !lbc.isNginxReady {
		glog.V(3).Infof("Skipping ConfigMap update because the pod is not ready yet")
		return
	}

	if lbc.batchSyncEnabled {
		glog.V(3).Infof("Skipping ConfigMap update because batch sync is on")
		return
	}

	lbc.updateAllConfigs()
}

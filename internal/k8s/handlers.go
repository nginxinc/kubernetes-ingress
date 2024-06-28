package k8s

import (
	"fmt"
	"reflect"

	"github.com/jinzhu/copier"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func createVirtualServerHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			vs := obj.(*conf_v1.VirtualServer)
			glog.V(3).Infof("Adding VirtualServer: %v", vs.Name)
			lbc.AddSyncQueue(vs)
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
			lbc.AddSyncQueue(vs)
		},
		UpdateFunc: func(old, cur interface{}) {
			curVs := cur.(*conf_v1.VirtualServer)
			oldVs := old.(*conf_v1.VirtualServer)

			if lbc.weightChangesDynamicReload {
				var curVsCopy, oldVsCopy conf_v1.VirtualServer
				err := copier.CopyWithOption(&curVsCopy, curVs, copier.Option{DeepCopy: true})
				if err != nil {
					glog.V(3).Infof("Error copying VirtualServer %v: %v for Dynamic Weight Changes", curVs.Name, err)
					return
				}

				err = copier.CopyWithOption(&oldVsCopy, oldVs, copier.Option{DeepCopy: true})
				if err != nil {
					glog.V(3).Infof("Error copying VirtualServer %v: %v for Dynamic Weight Changes", oldVs.Name, err)
					return
				}

				zeroOutVirtualServerSplitWeights(&curVsCopy)
				zeroOutVirtualServerSplitWeights(&oldVsCopy)

				if reflect.DeepEqual(oldVsCopy.Spec, curVsCopy.Spec) {
					lbc.processVSWeightChangesDynamicReload(oldVs, curVs)
					return
				}

			}

			if !reflect.DeepEqual(oldVs.Spec, curVs.Spec) {
				glog.V(3).Infof("VirtualServer %v changed, syncing", curVs.Name)
				lbc.AddSyncQueue(curVs)
			}
		},
	}
}

func createVirtualServerRouteHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			vsr := obj.(*conf_v1.VirtualServerRoute)
			glog.V(3).Infof("Adding VirtualServerRoute: %v", vsr.Name)
			lbc.AddSyncQueue(vsr)
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
			lbc.AddSyncQueue(vsr)
		},
		UpdateFunc: func(old, cur interface{}) {
			curVsr := cur.(*conf_v1.VirtualServerRoute)
			oldVsr := old.(*conf_v1.VirtualServerRoute)

			if lbc.weightChangesDynamicReload {
				var curVsrCopy, oldVsrCopy conf_v1.VirtualServerRoute
				err := copier.CopyWithOption(&curVsrCopy, curVsr, copier.Option{DeepCopy: true})
				if err != nil {
					glog.V(3).Infof("Error copying VirtualServerRoute %v: %v for Dynamic Weight Changes", curVsr.Name, err)
					return
				}

				err = copier.CopyWithOption(&oldVsrCopy, oldVsr, copier.Option{DeepCopy: true})
				if err != nil {
					glog.V(3).Infof("Error copying VirtualServerRoute %v: %v for Dynamic Weight Changes", oldVsr.Name, err)
					return
				}

				zeroOutVirtualServerRouteSplitWeights(&curVsrCopy)
				zeroOutVirtualServerRouteSplitWeights(&oldVsrCopy)

				if reflect.DeepEqual(oldVsrCopy.Spec, curVsrCopy.Spec) {
					lbc.processVSRWeightChangesDynamicReload(oldVsr, curVsr)
					return
				}

			}

			if !reflect.DeepEqual(oldVsr.Spec, curVsr.Spec) {
				glog.V(3).Infof("VirtualServerRoute %v changed, syncing", curVsr.Name)
				lbc.AddSyncQueue(curVsr)
			}
		},
	}
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
	if err != nil {
		return false, err
	}
	if !found {
		return false, fmt.Errorf("spec has unexpected format")
	}
	eq := reflect.DeepEqual(oldSpec, spec)
	if eq {
		glog.V(3).Infof("New spec of %v same as old spec", oldresource.GetName())
	}
	return !eq, nil
}

// createNamespaceHandlers builds the handler funcs for namespaces
func createNamespaceHandlers(lbc *LoadBalancerController) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ns := obj.(*v1.Namespace)
			glog.V(3).Infof("Adding Namespace to list of watched Namespaces: %v", ns.Name)
			lbc.AddSyncQueue(obj)
		},
		DeleteFunc: func(obj interface{}) {
			ns, isNs := obj.(*v1.Namespace)
			if !isNs {
				deletedState, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					glog.V(3).Infof("Error received unexpected object: %v", obj)
					return
				}
				ns, ok = deletedState.Obj.(*v1.Namespace)
				if !ok {
					glog.V(3).Infof("Error DeletedFinalStateUnknown contained non-Namespace object: %v", deletedState.Obj)
					return
				}
			}
			glog.V(3).Infof("Removing Namespace from list of watched Namespaces: %v", ns.Name)
			lbc.AddSyncQueue(obj)
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				glog.V(3).Infof("Namespace %v changed, syncing", cur.(*v1.Namespace).Name)
				lbc.AddSyncQueue(cur)
			}
		},
	}
}

func zeroOutVirtualServerSplitWeights(vs *conf_v1.VirtualServer) {
	for _, route := range vs.Spec.Routes {
		for _, match := range route.Matches {
			if len(match.Splits) == 2 {
				match.Splits[0].Weight = 0
				match.Splits[1].Weight = 0
			}
		}

		if len(route.Splits) == 2 {
			route.Splits[0].Weight = 0
			route.Splits[1].Weight = 0
		}
	}
}

func zeroOutVirtualServerRouteSplitWeights(vs *conf_v1.VirtualServerRoute) {
	for _, route := range vs.Spec.Subroutes {
		for _, match := range route.Matches {
			if len(match.Splits) == 2 {
				match.Splits[0].Weight = 0
				match.Splits[1].Weight = 0
			}
		}

		if len(route.Splits) == 2 {
			route.Splits[0].Weight = 0
			route.Splits[1].Weight = 0
		}
	}
}

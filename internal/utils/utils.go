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

package utils

import (
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"

	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
)

// compareLinks returns true if the 2 self links are equal.
func compareLinks(l1, l2 string) bool {
	// TODO: These can be partial links
	return l1 == l2 && l1 != ""
}

// StoreToIngressLister makes a Store that lists Ingress.
// TODO: Move this to cache/listers post 1.1.
type StoreToIngressLister struct {
	cache.Store
}

// GetByKeySafe calls Store.GetByKeySafe and returns a copy of the ingress so it is
// safe to modify.
func (s *StoreToIngressLister) GetByKeySafe(key string) (ing *extensions.Ingress, exists bool, err error) {
	item, exists, err := s.Store.GetByKey(key)
	if !exists || err != nil {
		return nil, exists, err
	}
	ing = item.(*extensions.Ingress).DeepCopy()
	return
}

// List lists all Ingress' in the store.
func (s *StoreToIngressLister) List() (ing extensions.IngressList, err error) {
	for _, m := range s.Store.List() {
		ing.Items = append(ing.Items, *(m.(*extensions.Ingress)).DeepCopy())
	}
	return ing, nil
}

// GetServiceIngress gets all the Ingress' that have rules pointing to a service.
// Note that this ignores services without the right nodePorts.
func (s *StoreToIngressLister) GetServiceIngress(svc *api_v1.Service) (ings []extensions.Ingress, err error) {
	for _, m := range s.Store.List() {
		ing := *m.(*extensions.Ingress).DeepCopy()
		if ing.Namespace != svc.Namespace {
			continue
		}
		if ing.Spec.Backend != nil {
			if ing.Spec.Backend.ServiceName == svc.Name {
				ings = append(ings, ing)
			}
		}
		for _, rules := range ing.Spec.Rules {
			if rules.IngressRuleValue.HTTP == nil {
				continue
			}
			for _, p := range rules.IngressRuleValue.HTTP.Paths {
				if p.Backend.ServiceName == svc.Name {
					ings = append(ings, ing)
				}
			}
		}
	}
	if len(ings) == 0 {
		err = fmt.Errorf("No ingress for service %v", svc.Name)
	}
	return
}

// StoreToConfigMapLister makes a Store that lists ConfigMaps
type StoreToConfigMapLister struct {
	cache.Store
}

// List lists all Ingress' in the store.
func (s *StoreToConfigMapLister) List() (cfgm api_v1.ConfigMapList, err error) {
	for _, m := range s.Store.List() {
		cfgm.Items = append(cfgm.Items, *(m.(*api_v1.ConfigMap)))
	}
	return cfgm, nil
}

// StoreToEndpointLister makes a Store that lists Endponts
type StoreToEndpointLister struct {
	cache.Store
}

// GetServiceEndpoints returns the endpoints of a service, matched on service name.
func (s *StoreToEndpointLister) GetServiceEndpoints(svc *api_v1.Service) (ep api_v1.Endpoints, err error) {
	for _, m := range s.Store.List() {
		ep = *m.(*api_v1.Endpoints)
		if svc.Name == ep.Name && svc.Namespace == ep.Namespace {
			return ep, nil
		}
	}
	err = fmt.Errorf("could not find endpoints for service: %v", svc.Name)
	return
}

// FindPort locates the container port for the given pod and portName.  If the
// targetPort is a number, use that.  If the targetPort is a string, look that
// string up in all named ports in all containers in the target pod.  If no
// match is found, fail.
func FindPort(pod *api_v1.Pod, svcPort *api_v1.ServicePort) (int32, error) {
	portName := svcPort.TargetPort
	switch portName.Type {
	case intstr.String:
		name := portName.StrVal
		for _, container := range pod.Spec.Containers {
			for _, port := range container.Ports {
				if port.Name == name && port.Protocol == svcPort.Protocol {
					return port.ContainerPort, nil
				}
			}
		}
	case intstr.Int:
		return int32(portName.IntValue()), nil
	}

	return 0, fmt.Errorf("no suitable port for manifest: %s", pod.UID)
}

// StoreToSecretLister makes a Store that lists Secrets
type StoreToSecretLister struct {
	cache.Store
}

func IsMinion(ing *extensions.Ingress) bool {
	if ing.Annotations["nginx.org/mergeable-ingress-type"] == "minion" {
		return true
	}
	return false
}

func IsMaster(ing *extensions.Ingress) bool {
	if ing.Annotations["nginx.org/mergeable-ingress-type"] == "master" {
		return true
	}
	return false
}

func HasChanges(old *extensions.Ingress, current *extensions.Ingress) bool {
	old.Status.LoadBalancer.Ingress = current.Status.LoadBalancer.Ingress
	old.ResourceVersion = current.ResourceVersion
	return !reflect.DeepEqual(old, current)
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

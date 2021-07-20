package upstreams

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/api"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	api_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Upstreams is an additional layer on top of the kubernetes resource apis. It provides methods that involve
// more than one type of Resource they don't belong to a specific resource api.
type Upstreams struct {
	pods      *api.Pods
	services  *api.Services
	endpoints *api.Endpoints
}

func NewUpstreams(api *api.Apis) *Upstreams {
	return &Upstreams{
		pods:      api.Pods,
		services:  api.Services,
		endpoints: api.Endpoints,
	}
}

func (u *Upstreams) GetHealthChecksForIngressBackend(backend *networking.IngressBackend, namespace string) *api_v1.Probe {
	svc, err := u.GetServiceForIngressBackend(backend, namespace)
	if err != nil {
		glog.V(3).Infof("Error getting service %v: %v", backend.ServiceName, err)
		return nil
	}
	svcPort := getServicePortForIngressPort(backend.ServicePort, svc)
	if svcPort == nil {
		return nil
	}
	ps, err := u.pods.ListByNamespace(svc.Namespace, labels.Set(svc.Spec.Selector).AsSelector())
	if err != nil {
		glog.V(3).Infof("Error fetching pods for namespace %v: %v", svc.Namespace, err)
		return nil
	}
	return findProbeForPods(ps, svcPort)
}

func (u *Upstreams) getPodOwnerTypeAndNameFromAddress(ns, name string) (parentType, parentName string) {
	pod, exists, err := u.pods.GetByKey(fmt.Sprintf("%s/%s", ns, name))
	if err != nil {
		glog.Warningf("could not get pod by key %s/%s: %v", ns, name, err)
		return "", ""
	}
	if !exists {
		return "", ""
	}
	return getPodOwnerTypeAndName(pod)
}

func (u *Upstreams) GetServiceForUpstream(namespace string, upstreamService string, upstreamPort uint16) (*api_v1.Service, error) {
	backend := &networking.IngressBackend{
		ServiceName: upstreamService,
		ServicePort: intstr.FromInt(int(upstreamPort)),
	}
	return u.GetServiceForIngressBackend(backend, namespace)
}

func (u *Upstreams) getTargetPort(svcPort api_v1.ServicePort, svc *api_v1.Service) (int32, error) {
	if (svcPort.TargetPort == intstr.IntOrString{}) {
		return svcPort.Port, nil
	}

	if svcPort.TargetPort.Type == intstr.Int {
		return int32(svcPort.TargetPort.IntValue()), nil
	}

	ps, err := u.pods.ListByNamespace(svc.Namespace, labels.Set(svc.Spec.Selector).AsSelector())
	if err != nil {
		return 0, fmt.Errorf("Error getting pod information: %w", err)
	}

	if len(ps) == 0 {
		return 0, fmt.Errorf("No pods of service %s", svc.Name)
	}

	pod := ps[0]

	portNum, err := findPort(pod, svcPort)
	if err != nil {
		return 0, fmt.Errorf("Error finding named port %v in pod %s: %w", svcPort, pod.Name, err)
	}

	return portNum, nil
}

func (u *Upstreams) GetServiceForIngressBackend(backend *networking.IngressBackend, namespace string) (*api_v1.Service, error) {
	svcKey := namespace + "/" + backend.ServiceName
	svc, svcExists, err := u.services.GetByKey(svcKey)
	if err != nil {
		return nil, err
	}

	if !svcExists {
		return nil, fmt.Errorf("service %s doesn't exist", svcKey)
	}

	return svc, nil
}

func (u *Upstreams) getEndpointsForServiceWithSubselector(targetPort int32, subselector map[string]string, svc *api_v1.Service) (endps []PodEndpoint, err error) {
	ps, err := u.pods.ListByNamespace(svc.Namespace, labels.Merge(svc.Spec.Selector, subselector).AsSelector())
	if err != nil {
		return nil, fmt.Errorf("Error getting pods in namespace %v that match the selector %v: %w", svc.Namespace, labels.Merge(svc.Spec.Selector, subselector), err)
	}

	svcEps, err := u.endpoints.GetServiceEndpoints(svc)
	if err != nil {
		glog.V(3).Infof("Error getting endpoints for service %s from the cache: %v", svc.Name, err)
		return nil, err
	}

	endps = getEndpointsBySubselectedPods(targetPort, ps, svcEps)
	return endps, nil
}

func (u *Upstreams) GetEndpointsForSubselector(namespace string, upstream conf_v1.Upstream) (endps []PodEndpoint, err error) {
	svc, err := u.GetServiceForUpstream(namespace, upstream.Service, upstream.Port)
	if err != nil {
		return nil, fmt.Errorf("Error getting service %v: %w", upstream.Service, err)
	}

	var targetPort int32

	for _, port := range svc.Spec.Ports {
		if port.Port == int32(upstream.Port) {
			targetPort, err = u.getTargetPort(port, svc)
			if err != nil {
				return nil, fmt.Errorf("Error determining target port for port %v in service %v: %w", upstream.Port, svc.Name, err)
			}
			break
		}
	}

	if targetPort == 0 {
		return nil, fmt.Errorf("No port %v in service %s", upstream.Port, svc.Name)
	}

	endps, err = u.getEndpointsForServiceWithSubselector(targetPort, upstream.Subselector, svc)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving endpoints for the service %v: %w", upstream.Service, err)
	}

	return endps, err
}

func (u *Upstreams) GetEndpointsForUpstream(namespace string, upstreamService string, upstreamPort uint16, isNginxPlus bool) (endps []PodEndpoint, isExternal bool, err error) {
	svc, err := u.GetServiceForUpstream(namespace, upstreamService, upstreamPort)
	if err != nil {
		return nil, false, fmt.Errorf("Error getting service %v: %w", upstreamService, err)
	}

	backend := &networking.IngressBackend{
		ServiceName: upstreamService,
		ServicePort: intstr.FromInt(int(upstreamPort)),
	}

	endps, isExternal, err = u.GetEndpointsForIngressBackend(backend, svc, isNginxPlus)
	if err != nil {
		return nil, false, fmt.Errorf("Error retrieving endpoints for the service %v: %w", upstreamService, err)
	}

	return endps, isExternal, err
}

func (u *Upstreams) GetEndpointsForIngressBackend(backend *networking.IngressBackend, svc *api_v1.Service, isNginxPlus bool) (result []PodEndpoint, isExternal bool, err error) {
	endps, err := u.endpoints.GetServiceEndpoints(svc)
	if err != nil {
		if svc.Spec.Type == api_v1.ServiceTypeExternalName {
			if !isNginxPlus {
				return nil, false, fmt.Errorf("Type ExternalName Services feature is only available in NGINX Plus")
			}
			result = getExternalEndpointsForIngressBackend(backend, svc)
			return result, true, nil
		}
		glog.V(3).Infof("Error getting endpoints for service %s from the cache: %v", svc.Name, err)
		return nil, false, err
	}

	result, err = u.getEndpointsForPort(endps, backend.ServicePort, svc)
	if err != nil {
		glog.V(3).Infof("Error getting endpoints for service %s port %v: %v", svc.Name, backend.ServicePort, err)
		return nil, false, err
	}
	return result, false, nil
}

func (u *Upstreams) getEndpointsForPort(endps api_v1.Endpoints, ingSvcPort intstr.IntOrString, svc *api_v1.Service) ([]PodEndpoint, error) {
	var targetPort int32
	var err error

	for _, port := range svc.Spec.Ports {
		if (ingSvcPort.Type == intstr.Int && port.Port == int32(ingSvcPort.IntValue())) || (ingSvcPort.Type == intstr.String && port.Name == ingSvcPort.String()) {
			targetPort, err = u.getTargetPort(port, svc)
			if err != nil {
				return nil, fmt.Errorf("Error determining target port for port %v in Ingress: %w", ingSvcPort, err)
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
				var endpoints []PodEndpoint
				for _, address := range subset.Addresses {
					addr := fmt.Sprintf("%v:%v", address.IP, port.Port)
					podEnd := PodEndpoint{
						Address: addr,
					}
					if address.TargetRef != nil {
						parentType, parentName := u.getPodOwnerTypeAndNameFromAddress(address.TargetRef.Namespace, address.TargetRef.Name)
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

// findPort locates the container port for the given pod and portName.  If the
// targetPort is a number, use that.  If the targetPort is a string, look that
// string up in all named ports in all containers in the target pod.  If no
// match is found, fail.
func findPort(pod *v1.Pod, svcPort v1.ServicePort) (int32, error) {
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

func getPodOwnerTypeAndName(pod *v1.Pod) (parentType, parentName string) {
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

type PodEndpoint struct {
	Address string
	PodName string
	// MeshPodOwner is used for NGINX Service Mesh metrics
	configs.MeshPodOwner
}

func getExternalEndpointsForIngressBackend(backend *networking.IngressBackend, svc *api_v1.Service) []PodEndpoint {
	address := fmt.Sprintf("%s:%d", svc.Spec.ExternalName, int32(backend.ServicePort.IntValue()))
	endpoints := []PodEndpoint{
		{
			Address: address,
			PodName: "",
		},
	}
	return endpoints
}

func getEndpointsBySubselectedPods(targetPort int32, pods []*v1.Pod, svcEps v1.Endpoints) (endps []PodEndpoint) {
	for _, pod := range pods {
		for _, subset := range svcEps.Subsets {
			for _, port := range subset.Ports {
				if port.Port != targetPort {
					continue
				}
				for _, address := range subset.Addresses {
					if address.IP == pod.Status.PodIP {
						addr := fmt.Sprintf("%v:%v", pod.Status.PodIP, targetPort)
						ownerType, ownerName := getPodOwnerTypeAndName(pod)
						podEnd := PodEndpoint{
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

func getPodName(pod *v1.ObjectReference) string {
	if pod != nil {
		return pod.Name
	}
	return ""
}

func findProbeForPods(pods []*v1.Pod, svcPort *v1.ServicePort) *v1.Probe {
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

func compareContainerPortAndServicePort(containerPort v1.ContainerPort, svcPort v1.ServicePort) bool {
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

func getServicePortForIngressPort(ingSvcPort intstr.IntOrString, svc *v1.Service) *v1.ServicePort {
	for _, port := range svc.Spec.Ports {
		if (ingSvcPort.Type == intstr.Int && port.Port == int32(ingSvcPort.IntValue())) || (ingSvcPort.Type == intstr.String && port.Name == ingSvcPort.String()) {
			return &port
		}
	}
	return nil
}

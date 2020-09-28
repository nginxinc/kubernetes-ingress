package k8s

import (
	"context"
	"reflect"
	"testing"

	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestStatusUpdate(t *testing.T) {
	ing := networking.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "ing-1",
			Namespace: "namespace",
		},
		Status: networking.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{
						IP: "1.2.3.4",
					},
				},
			},
		},
	}
	fakeClient := fake.NewSimpleClientset(
		&networking.IngressList{Items: []networking.Ingress{
			ing,
		}},
	)
	ingLister := storeToIngressLister{}
	ingLister.Store, _ = cache.NewInformer(
		cache.NewListWatchFromClient(fakeClient.NetworkingV1beta1().RESTClient(), "ingresses", "nginx-ingress", fields.Everything()),
		&networking.Ingress{}, 2, nil)

	err := ingLister.Store.Add(&ing)
	if err != nil {
		t.Errorf("Error adding Ingress to the ingress lister: %v", err)
	}

	su := statusUpdater{
		client:                fakeClient,
		namespace:             "namespace",
		externalServiceName:   "service-name",
		externalStatusAddress: "123.123.123.123",
		ingressLister:         &ingLister,
		keyFunc:               cache.DeletionHandlingMetaNamespaceKeyFunc,
	}
	err = su.ClearIngressStatus(ing)
	if err != nil {
		t.Errorf("error clearing ing status: %v", err)
	}
	ings, _ := fakeClient.NetworkingV1beta1().Ingresses("namespace").List(context.TODO(), meta_v1.ListOptions{})
	ingf := ings.Items[0]
	if !checkStatus("", ingf) {
		t.Errorf("expected: %v actual: %v", "", ingf.Status.LoadBalancer.Ingress[0])
	}

	su.SaveStatusFromExternalStatus("1.1.1.1")
	err = su.UpdateIngressStatus(ing)
	if err != nil {
		t.Errorf("error updating ing status: %v", err)
	}
	ring, _ := fakeClient.NetworkingV1beta1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, meta_v1.GetOptions{})
	if !checkStatus("1.1.1.1", *ring) {
		t.Errorf("expected: %v actual: %v", "", ring.Status.LoadBalancer.Ingress)
	}

	svc := v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "namespace",
			Name:      "service-name",
		},
		Status: v1.ServiceStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{{
					IP: "2.2.2.2",
				}},
			},
		},
	}
	su.SaveStatusFromExternalService(&svc)
	err = su.UpdateIngressStatus(ing)
	if err != nil {
		t.Errorf("error updating ing status: %v", err)
	}
	ring, _ = fakeClient.NetworkingV1beta1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, meta_v1.GetOptions{})
	if !checkStatus("1.1.1.1", *ring) {
		t.Errorf("expected: %v actual: %v", "1.1.1.1", ring.Status.LoadBalancer.Ingress)
	}

	su.SaveStatusFromExternalStatus("")
	err = su.UpdateIngressStatus(ing)
	if err != nil {
		t.Errorf("error updating ing status: %v", err)
	}
	ring, _ = fakeClient.NetworkingV1beta1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, meta_v1.GetOptions{})
	if !checkStatus("2.2.2.2", *ring) {
		t.Errorf("expected: %v actual: %v", "2.2.2.2", ring.Status.LoadBalancer.Ingress)
	}

	su.ClearStatusFromExternalService()
	err = su.UpdateIngressStatus(ing)
	if err != nil {
		t.Errorf("error updating ing status: %v", err)
	}
	ring, _ = fakeClient.NetworkingV1beta1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, meta_v1.GetOptions{})
	if !checkStatus("", *ring) {
		t.Errorf("expected: %v actual: %v", "", ring.Status.LoadBalancer.Ingress)
	}
}

func checkStatus(expected string, actual networking.Ingress) bool {
	if len(actual.Status.LoadBalancer.Ingress) == 0 {
		return expected == ""
	}
	return expected == actual.Status.LoadBalancer.Ingress[0].IP
}

func TestGenerateExternalEndpointsFromStatus(t *testing.T) {
	su := statusUpdater{
		status: []v1.LoadBalancerIngress{
			{
				IP: "8.8.8.8",
			},
		},
	}

	expectedEndpoints := []conf_v1.ExternalEndpoint{
		{IP: "8.8.8.8", Ports: ""},
	}

	endpoints := su.generateExternalEndpointsFromStatus(su.status)

	if !reflect.DeepEqual(endpoints, expectedEndpoints) {
		t.Errorf("generateExternalEndpointsFromStatus(%v) returned %v but expected %v", su.status, endpoints, expectedEndpoints)
	}
}

func TestHasVsStatusChanged(t *testing.T) {
	state := "Valid"
	reason := "AddedOrUpdated"
	msg := "Configuration was added or updated"

	tests := []struct {
		expected bool
		vs       conf_v1.VirtualServer
	}{
		{
			expected: false,
			vs: conf_v1.VirtualServer{
				Status: conf_v1.VirtualServerStatus{
					State:   state,
					Reason:  reason,
					Message: msg,
				},
			},
		},
		{
			expected: true,
			vs: conf_v1.VirtualServer{
				Status: conf_v1.VirtualServerStatus{
					State:   "DifferentState",
					Reason:  reason,
					Message: msg,
				},
			},
		},
		{
			expected: true,
			vs: conf_v1.VirtualServer{
				Status: conf_v1.VirtualServerStatus{
					State:   state,
					Reason:  "DifferentReason",
					Message: msg,
				},
			},
		},
		{
			expected: true,
			vs: conf_v1.VirtualServer{
				Status: conf_v1.VirtualServerStatus{
					State:   state,
					Reason:  reason,
					Message: "DifferentMessage",
				},
			},
		},
	}

	for _, test := range tests {
		changed := hasVsStatusChanged(&test.vs, state, reason, msg)

		if changed != test.expected {
			t.Errorf("hasVsStatusChanged(%v, %v, %v, %v) returned %v but expected %v.", test.vs, state, reason, msg, changed, test.expected)
		}
	}
}

func TestHasVsrStatusChanged(t *testing.T) {

	referencedBy := "namespace/name"
	state := "Valid"
	reason := "AddedOrUpdated"
	msg := "Configuration was added or updated"

	tests := []struct {
		expected bool
		vsr      conf_v1.VirtualServerRoute
	}{
		{
			expected: false,
			vsr: conf_v1.VirtualServerRoute{
				Status: conf_v1.VirtualServerRouteStatus{
					State:        state,
					Reason:       reason,
					Message:      msg,
					ReferencedBy: referencedBy,
				},
			},
		},
		{
			expected: true,
			vsr: conf_v1.VirtualServerRoute{
				Status: conf_v1.VirtualServerRouteStatus{
					State:        "DifferentState",
					Reason:       reason,
					Message:      msg,
					ReferencedBy: referencedBy,
				},
			},
		},
		{
			expected: true,
			vsr: conf_v1.VirtualServerRoute{
				Status: conf_v1.VirtualServerRouteStatus{
					State:        state,
					Reason:       "DifferentReason",
					Message:      msg,
					ReferencedBy: referencedBy,
				},
			},
		},
		{
			expected: true,
			vsr: conf_v1.VirtualServerRoute{
				Status: conf_v1.VirtualServerRouteStatus{
					State:        state,
					Reason:       reason,
					Message:      "DifferentMessage",
					ReferencedBy: referencedBy,
				},
			},
		},
		{
			expected: true,
			vsr: conf_v1.VirtualServerRoute{
				Status: conf_v1.VirtualServerRouteStatus{
					State:        state,
					Reason:       reason,
					Message:      msg,
					ReferencedBy: "DifferentReferencedBy",
				},
			},
		},
	}

	for _, test := range tests {
		changed := hasVsrStatusChanged(&test.vsr, state, reason, msg, referencedBy)

		if changed != test.expected {
			t.Errorf("hasVsrStatusChanged(%v, %v, %v, %v) returned %v but expected %v.", test.vsr, state, reason, msg, changed, test.expected)
		}
	}
}

func TestGetExternalServicePorts(t *testing.T) {
	svc := v1.Service{
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port: int32(80),
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 80,
					},
				},
				{
					Port: int32(443),
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 443,
					},
				},
			},
		},
	}

	expected := "[80,443]"
	ports := getExternalServicePorts(&svc)

	if ports != expected {
		t.Errorf("getExternalServicePorts(%v) returned %v but expected %v", svc, ports, expected)
	}
}

func TestIsRequiredPort(t *testing.T) {
	tests := []struct {
		port     intstr.IntOrString
		expected bool
	}{
		{
			port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 999,
			},
			expected: false,
		},
		{
			port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 80,
			},
			expected: true,
		},
		{
			port: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 443,
			},
			expected: true,
		},
		{
			port: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "name",
			},
			expected: false,
		},
		{
			port: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "http",
			},
			expected: true,
		},
		{
			port: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "https",
			},
			expected: true,
		},
	}

	for _, test := range tests {
		result := isRequiredPort(test.port)

		if result != test.expected {
			t.Errorf("isRequiredPort(%+v) returned %v but expected %v", test.port, result, test.expected)
		}
	}
}

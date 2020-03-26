package k8s

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestStatusUpdate(t *testing.T) {
	ing := extensions.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "ing-1",
			Namespace: "namespace",
		},
		Status: extensions.IngressStatus{
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
		&extensions.IngressList{Items: []extensions.Ingress{
			ing,
		}},
	)
	ingLister := storeToIngressLister{}
	ingLister.Store, _ = cache.NewInformer(
		cache.NewListWatchFromClient(fakeClient.ExtensionsV1beta1().RESTClient(), "ingresses", "nginx-ingress", fields.Everything()),
		&extensions.Ingress{}, 2, nil)

	err := ingLister.Store.Add(&ing)
	if err != nil {
		t.Errorf("Error adding Ingress to the ingress lister: %v", err)
	}

	su := statusUpdater{
		client:                fakeClient,
		namespace:             "namespace",
		externalServiceName:   "service-name",
		externalStatusAddress: "123.123.123.123",
		ingLister:             &ingLister,
		keyFunc:               cache.DeletionHandlingMetaNamespaceKeyFunc,
	}
	err = su.ClearIngressStatus(ing)
	if err != nil {
		t.Errorf("error clearing ing status: %v", err)
	}
	ings, _ := fakeClient.ExtensionsV1beta1().Ingresses("namespace").List(context.TODO(), meta_v1.ListOptions{})
	ingf := ings.Items[0]
	if !checkStatus("", ingf) {
		t.Errorf("expected: %v actual: %v", "", ingf.Status.LoadBalancer.Ingress[0])
	}

	su.SaveStatusFromExternalStatus("1.1.1.1")
	err = su.UpdateIngressStatus(ing)
	if err != nil {
		t.Errorf("error updating ing status: %v", err)
	}
	ring, _ := fakeClient.ExtensionsV1beta1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, meta_v1.GetOptions{})
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
	ring, _ = fakeClient.ExtensionsV1beta1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, meta_v1.GetOptions{})
	if !checkStatus("1.1.1.1", *ring) {
		t.Errorf("expected: %v actual: %v", "1.1.1.1", ring.Status.LoadBalancer.Ingress)
	}

	su.SaveStatusFromExternalStatus("")
	err = su.UpdateIngressStatus(ing)
	if err != nil {
		t.Errorf("error updating ing status: %v", err)
	}
	ring, _ = fakeClient.ExtensionsV1beta1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, meta_v1.GetOptions{})
	if !checkStatus("2.2.2.2", *ring) {
		t.Errorf("expected: %v actual: %v", "2.2.2.2", ring.Status.LoadBalancer.Ingress)
	}

	su.ClearStatusFromExternalService()
	err = su.UpdateIngressStatus(ing)
	if err != nil {
		t.Errorf("error updating ing status: %v", err)
	}
	ring, _ = fakeClient.ExtensionsV1beta1().Ingresses(ing.Namespace).Get(context.TODO(), ing.Name, meta_v1.GetOptions{})
	if !checkStatus("", *ring) {
		t.Errorf("expected: %v actual: %v", "", ring.Status.LoadBalancer.Ingress)
	}
}

func checkStatus(expected string, actual extensions.Ingress) bool {
	if len(actual.Status.LoadBalancer.Ingress) == 0 {
		return expected == ""
	}
	return expected == actual.Status.LoadBalancer.Ingress[0].IP
}

package main

import (
	"fmt"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/k8s"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiVersion "k8s.io/apimachinery/pkg/version"
	fakeDisc "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateConfigClient(t *testing.T) {
	*enableCustomResources = true
	{
		*proxyURL = "localhost"
		config, err := mustGetClientConfig()
		if err != nil {
			t.Errorf("Failed to get client config: %v", err)
		}

		// This code block tests the working scenario
		{
			_, err := mustCreateConfigClient(config)
			if err != nil {
				t.Errorf("Failed to create client config: %v", err)
			}
		}
	}
}

func TestMinimumK8sVersion(t *testing.T) {
	// Create a fake client  -
	// WARNING: NewSimpleClientset is deprecated
	clientset := fake.NewSimpleClientset()

	// Override the ServerVersion method on the fake Discovery client
	discoveryClient, ok := clientset.Discovery().(*fakeDisc.FakeDiscovery)
	if !ok {
		fmt.Println("couldn't convert Discovery() to *FakeDiscovery")
	}

	// This test block is when the correct/expected k8s version is returned
	{
		correctVersion := &apiVersion.Info{
			Major: "1", Minor: "22", GitVersion: "v1.22.2",
		}
		discoveryClient.FakedServerVersion = correctVersion

		// Get the server version as a sanity check
		_, err := discoveryClient.ServerVersion()
		if err != nil {
			t.Fatalf("Failed to get server version: %v", err)
		}

		// Verify if the mocked server version is as expected.
		if err := mustConfirmMinimumK8sVersionCriteria(clientset); err != nil {
			t.Fatalf("Error in checking minimum k8s version: %v", err)
		}
	}

	// This test block is when the incorrect/unexpected k8s version is returned
	// i.e. not the min supported version
	{
		wrongVersion := &apiVersion.Info{
			Major: "1", Minor: "19", GitVersion: "v1.19.2",
		}
		discoveryClient.FakedServerVersion = wrongVersion

		// Get the server version as a sanity check
		_, err := discoveryClient.ServerVersion()
		if err != nil {
			t.Fatalf("Failed to get server version: %v", err)
		}

		// Verify if the mocked server version returns an error as we are testing for < 1.22 (v1.19.2).
		if err := mustConfirmMinimumK8sVersionCriteria(clientset); err == nil {
			t.Fatalf("Expected an error when checking minimum k8s version but got none: %v", err)
		}
	}
}

// Test valid (nginx) and invalid (other) ingress classes
func TestValidateIngressClass(t *testing.T) {
	// Define an IngressClass
	{
		ingressClass := &networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx",
			},
			Spec: networkingv1.IngressClassSpec{
				Controller: k8s.IngressControllerName,
			},
		}
		// Create a fake client
		clientset := fake.NewSimpleClientset(ingressClass)

		validData := []struct {
			clientset kubernetes.Interface
		}{
			{
				clientset: clientset,
			},
		}

		if err := mustValidateIngressClass(validData[0].clientset); err != nil {
			t.Fatalf("error in ingress class, error: %v", err)
		}
	}

	// Test invalid case
	{
		ingressClass := &networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "not-nginx",
			},
			Spec: networkingv1.IngressClassSpec{
				Controller: "www.example.com/ingress-controller",
			},
		}
		clientset := fake.NewSimpleClientset(ingressClass)
		inValidData := []struct {
			clientset kubernetes.Interface
		}{
			{
				clientset: clientset,
			},
		}

		if err := mustValidateIngressClass(inValidData[0].clientset); err == nil {
			t.Fatalf("validateIngressClass() returned no error for invalid input, error: %v", err)
		}
	}
}

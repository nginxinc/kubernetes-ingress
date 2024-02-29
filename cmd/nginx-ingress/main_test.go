package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/k8s"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
)

// Test for getNginxVersionInfo()
func TestGetNginxVersionInfo(t *testing.T) {
	os.Args = append(os.Args, "-nginx-plus")
	os.Args = append(os.Args, "-proxy")
	os.Args = append(os.Args, "test-proxy")
	flag.Parse()
	constLabels := map[string]string{"class": *ingressClass}
	mc := collectors.NewLocalManagerMetricsCollector(constLabels)
	nginxManager, _ := createNginxManager(mc)
	nginxInfo, _ := getNginxVersionInfo(nginxManager)
	if nginxInfo.String() == "" {
		t.Errorf("Error when getting nginx version, empty string")
	}
}

func TestGetAppProtectVersionInfo(t *testing.T) {
	dataToWrite := "1.2.3\n"
	dirPath := path.Dir(appProtectVersionPath)
	versionFile := path.Base(appProtectVersionPath)
	f, err := os.CreateTemp(dirPath, versionFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	l, err := f.WriteString(dataToWrite)
	if err != nil {
		fmt.Println(err)
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}

		return
	}
	fmt.Println(l, "bytes written successfully")
	if err := f.Close(); err != nil {
		fmt.Println(err)
		return
	}
	version, err := getAppProtectVersionInfo()
	if err != nil {
		t.Errorf("Error reading AppProtect Version file #{err}\n")
	}

	if version == "" {
		t.Errorf("Error with AppProtect Version, is empty")
	}
}

func TestCreateGlobalConfigurationValidator(t *testing.T) {
	globalConfiguration := conf_v1.GlobalConfiguration{
		Spec: conf_v1.GlobalConfigurationSpec{
			Listeners: []conf_v1.Listener{
				{
					Name:     "tcp-listener",
					Port:     53,
					Protocol: "TCP",
				},
				{
					Name:     "udp-listener",
					Port:     53,
					Protocol: "UDP",
				},
			},
		},
	}

	gcv := createGlobalConfigurationValidator()

	if err := gcv.ValidateGlobalConfiguration(&globalConfiguration); err != nil {
		t.Errorf("ValidateGlobalConfiguration() returned error %v for valid input", err)
	}

	incorrectGlobalConf := conf_v1.GlobalConfiguration{
		Spec: conf_v1.GlobalConfigurationSpec{
			Listeners: []conf_v1.Listener{
				{
					Name:     "tcp-listener",
					Port:     53,
					Protocol: "TCPT",
				},
				{
					Name:     "udp-listener",
					Port:     53,
					Protocol: "UDP",
				},
			},
		},
	}

	if err := gcv.ValidateGlobalConfiguration(&incorrectGlobalConf); err == nil {
		t.Errorf("ValidateGlobalConfiguration() returned error %v for invalid input", err)
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

		if err := validateIngressClass(validData[0].clientset); err != nil {
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

		if err := validateIngressClass(inValidData[0].clientset); err == nil {
			t.Fatalf("validateIngressClass() returned no error for invalid input, error: %v", err)
		}
	}
}

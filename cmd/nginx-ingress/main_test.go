package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/k8s"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiVersion "k8s.io/apimachinery/pkg/version"
	fakeDisc "k8s.io/client-go/discovery/fake"
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

type MockFileHandle struct {
	FileContent []byte
	ReadErr     error
}

func (m *MockFileHandle) ReadFile(_ string) ([]byte, error) {
	if m.ReadErr != nil {
		return nil, m.ReadErr
	}
	return m.FileContent, nil
}

func TestGetAppProtectVersionInfo(t *testing.T) {
	// Test for file reader returning valid/correct info and no errors
	{
		mockFileHandle := &MockFileHandle{
			FileContent: []byte("1.2.3\n"),
			ReadErr:     nil,
		}
		_, err := getAppProtectVersionInfo(mockFileHandle)
		if err != nil {
			t.Errorf("Error reading AppProtect Version file: %v", err)
		}
	}
	// Test for file reader returning an error
	{
		mockFileHandle := &MockFileHandle{
			FileContent: []byte("1.2.3\n"),
			ReadErr:     errors.ErrUnsupported,
		}
		_, err := getAppProtectVersionInfo(mockFileHandle)
		if err == nil {
			t.Errorf("Error reading AppProtect Version file: %v", err)
		}
	}
	// Test for file reader returning an empty version
	//{
	//	mockFileHandle := &MockFileHandle{
	//		FileContent: []byte("\n"),
	//		ReadErr:     nil,
	//	}
	//	version, _ := getAppProtectVersionInfo(mockFileHandle)
	//	if version == "" {
	//		t.Errorf("Error with AppProtect Version, is empty")
	//	}
	//}
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

func TestMinimumK8sVersion3(t *testing.T) {
	// Create a fake client.
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
		if err := confirmMinimumK8sVersionCriteria(clientset); err != nil {
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
		if err := confirmMinimumK8sVersionCriteria(clientset); err == nil {
			t.Fatalf("Expected an error when checking minimum k8s version but got none: %v", err)
		}
	}
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			log.Fatalf("Unable to marshal ECDSA private key: %v", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func genCertKeyPair() (string, string) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	out := &bytes.Buffer{}
	if err = pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatal(err)
	}
	cert := out.String()

	out.Reset()
	if err = pem.Encode(out, pemBlockForKey(priv)); err != nil {
		log.Fatal(err)
	}
	privKey := out.String()

	return cert, privKey
}

func TestGetAndValidateSecret(t *testing.T) {
	// Test for the working case where nothing goes wrong with valid data
	cert, privKey := genCertKeyPair()
	{
		secret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "default",
			},
			Type: "kubernetes.io/tls",
			Data: map[string][]byte{
				"tls.crt": []byte(cert),
				"tls.key": []byte(privKey),
			},
		}

		kAPI := &KubernetesAPI{
			Client: fake.NewSimpleClientset(&secret),
		}
		_, err := kAPI.getAndValidateSecret("default/my-secret")
		if err != nil {
			t.Errorf("Error in retrieving secret: %v", err)
		}
	}

	// Test for the non-existent secret
	{
		secret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "default",
			},
			Type: "kubernetes.io/tls",
			Data: map[string][]byte{
				"tls.crt": []byte(cert),
				"tls.key": []byte(privKey),
			},
		}

		kAPI := &KubernetesAPI{
			Client: fake.NewSimpleClientset(&secret),
		}
		_, err := kAPI.getAndValidateSecret("default/non-existent-secret")
		if err == nil {
			t.Errorf("Expected an error in retrieving secret but %v returned", err)
		}
	}

	// Test for the TLS cert/key without the key
	{
		secret := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "default",
			},
			Type: "kubernetes.io/tls",
			Data: map[string][]byte{
				"tls.crt": []byte(cert),
				"tls.key": []byte(""),
			},
		}

		kAPI := &KubernetesAPI{
			Client: fake.NewSimpleClientset(&secret),
		}
		_, err := kAPI.getAndValidateSecret("default/my-secret")
		if err == nil {
			t.Errorf("Expected an error in retrieving secret but %v returned", err)
		}
	}
}

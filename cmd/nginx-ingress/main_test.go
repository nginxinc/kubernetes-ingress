package main

import (
	"bytes"
	"context"
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
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/k8s"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
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

	if !nginxInfo.IsPlus {
		t.Errorf("Error version is not nginx-plus")
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

// Utility function to check if a namespace
func expectedNs(watchNsLabelList string, ns []string) bool {
	wNs := strings.Split(watchNsLabelList, ",")
	resultOk := false
	for _, n := range wNs {
		nsNameWithDelimiter := strings.Split(n, "=")
		nsNameOnly := ""
		if len(nsNameWithDelimiter) > 1 {
			nsNameOnly = nsNameWithDelimiter[1]
		}
		isValid := slices.Contains(ns, nsNameOnly)
		resultOk = resultOk || isValid
	}
	return resultOk
}

// This test uses a fake client to create 2 namespaces, ns1 and ns2
// We use these objects to test the retreival of namespaces based on the
// watchedNamespacesLabel input
func TestGetWatchedNamespaces(t *testing.T) {
	// Create a new fake clientset
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	// Create label for test1-namespace
	ns1Labels := map[string]string{
		"namespace": "ns1",
		"app":       "my-application",
		"version":   "v1",
	}

	// Create the ns1 namespace using the fake clientset
	_, err := clientset.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ns1",
			Labels: ns1Labels,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	// Create label for test2-namespace
	ns2Labels := map[string]string{
		"namespace": "ns2",
		"app":       "my-application",
		"version":   "v1",
	}

	// Create the ns2 namespace using the fake clientset
	_, err = clientset.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ns2",
			Labels: ns2Labels,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	// This section is testing the presence of the watchedNamespaceLabels
	{
		// Create a list of 'watched' namespaces
		watchNsLabelList := "namespace=ns2, version=v1"
		watchNamespaceLabel = &watchNsLabelList
		ns := getWatchedNamespaces(clientset)

		if len(ns) == 0 {
			t.Errorf("Expected namespaces-list not to be empty")
		}

		resultOk := expectedNs(watchNsLabelList, ns)
		if !resultOk {
			t.Errorf("Expected namespaces-list to be %v, got %v", watchNsLabelList, ns)
		}
	}

	// This section is testing the absence (ns3) of the watchedNamespaceLabels
	{
		watchNsLabelList := "namespace=ns3, version=v1"
		watchNamespaceLabel = &watchNsLabelList
		ns := getWatchedNamespaces(clientset)
		if len(ns) != 0 {
			t.Errorf("Expected expected an empty namespaces-list but got %v", ns)
		}
	}
}

func TestCheckNamespaceExists(t *testing.T) {
	// Create a new fake clientset
	clientset := fake.NewSimpleClientset()
	ctx := context.Background()

	// Create label for test1-namespace
	ns1Labels := map[string]string{
		"namespace": "ns1",
		"app":       "my-application",
		"version":   "v1",
	}

	// Create the ns1 namespace using the fake clientset
	_, err := clientset.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "ns1",
			Labels: ns1Labels,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	// This block is to test the successful case i.e. where the searched namespace exists
	{
		nsList := []string{"ns1"}
		hasErrors := checkNamespaceExists(clientset, nsList)
		if hasErrors {
			t.Errorf("Expected namespaces-list %v to be present, got error", nsList)
		}
	}

	// This block is to test the failure case i.e. where the searched namespace does not exists
	{
		nsList := []string{"ns2"}
		hasErrors := checkNamespaceExists(clientset, nsList)
		if !hasErrors {
			t.Errorf("Expected namespaces-list %v to be absent, but got no errors", nsList)
		}
	}
}

func TestCreateConfigClient(t *testing.T) {
	*enableCustomResources = true
	{
		*proxyURL = "localhost"
		config, err := getClientConfig()
		if err != nil {
			t.Errorf("Failed to get client config: %v", err)
		}

		// This code block tests the working scenario
		{
			_, err := createConfigClient(config)
			if err != nil {
				t.Errorf("Failed to create client config: %v", err)
			}
		}
	}
}

func TestCreateNginxManager(t *testing.T) {
	constLabels := map[string]string{"class": *ingressClass}
	mgrCollector, _, _ := createManagerAndControllerCollectors(constLabels)
	nginxMgr, _ := createNginxManager(mgrCollector)

	if nginxMgr == nil {
		t.Errorf("Failed to create nginx manager")
	}
}

func TestProcessDefaultServerSecret(t *testing.T) {
	kAPI := &KubernetesAPI{
		Client: fake.NewSimpleClientset(),
	}
	mgr := nginx.NewFakeManager("/etc/nginx")
	{
		sslRejectHandshake, err := kAPI.processDefaultServerSecret(mgr)
		if err != nil {
			t.Errorf("Failed to process default server secret: %v", err)
		}

		if !sslRejectHandshake {
			t.Errorf("Expected sslRejectHandshake to be false")
		}
	}

	{
		*defaultServerSecret = "/etc/nginx/ssl/myNonExistentSecret.crt"
		sslRejectHandshake, err := kAPI.processDefaultServerSecret(mgr)
		if err == nil {
			t.Errorf("Failed to process default server secret")
		}

		if sslRejectHandshake {
			t.Errorf("Expected sslRejectHandshake to be true")
		}

	}
}

func TestProcessWildcardSecret(t *testing.T) {
	kAPI := &KubernetesAPI{
		Client: fake.NewSimpleClientset(),
	}
	mgr := nginx.NewFakeManager("/etc/nginx")
	{
		wildcardTLSSecret, err := kAPI.processWildcardSecret(mgr)
		if err != nil {
			t.Errorf("Failed to process wildcard server secret: %v", err)
		}

		if wildcardTLSSecret {
			t.Errorf("Expected wildcardTLSSecret to be false")
		}
	}

	{
		*wildcardTLSSecret = "/etc/nginx/ssl/myNonExistentSecret.crt"
		wildcardTLSSecret, err := kAPI.processWildcardSecret(mgr)
		if err == nil {
			t.Errorf("Failed to process wildcard server secret, expected error")
		}

		if wildcardTLSSecret {
			t.Errorf("Expected wildcardTLSSecret to be false")
		}

	}
}

package configs

import (
	"reflect"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestStaticConfigParams() *StaticConfigParams {
	return &StaticConfigParams{
		HealthStatus:                   true,
		HealthStatusURI:                "/nginx-health",
		NginxStatus:                    true,
		NginxStatusAllowCIDRs:          []string{"127.0.0.1"},
		NginxStatusPort:                8080,
		StubStatusOverUnixSocketForOSS: false,
	}
}

func createTestConfigurator() (*Configurator, error) {
	templateExecutor, err := version1.NewTemplateExecutor("version1/nginx-plus.tmpl", "version1/nginx-plus.ingress.tmpl")
	if err != nil {
		return nil, err
	}

	templateExecutorV2, err := version2.NewTemplateExecutor("version2/nginx-plus.virtualserver.tmpl", "version2/nginx-plus.transportserver.tmpl")
	if err != nil {
		return nil, err
	}

	manager := nginx.NewFakeManager("/etc/nginx")

	return NewConfigurator(manager, createTestStaticConfigParams(), NewDefaultConfigParams(), NewDefaultGlobalConfigParams(), templateExecutor, templateExecutorV2, false, false), nil
}

func createTestConfiguratorInvalidIngressTemplate() (*Configurator, error) {
	templateExecutor, err := version1.NewTemplateExecutor("version1/nginx-plus.tmpl", "version1/nginx-plus.ingress.tmpl")
	if err != nil {
		return nil, err
	}

	invalidIngressTemplate := "{{.Upstreams.This.Field.Does.Not.Exist}}"
	if err := templateExecutor.UpdateIngressTemplate(&invalidIngressTemplate); err != nil {
		return nil, err
	}

	manager := nginx.NewFakeManager("/etc/nginx")

	return NewConfigurator(manager, createTestStaticConfigParams(), NewDefaultConfigParams(), NewDefaultGlobalConfigParams(), templateExecutor, &version2.TemplateExecutor{}, false, false), nil
}

func TestAddOrUpdateIngress(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()

	err = cnf.AddOrUpdateIngress(&ingress)
	if err != nil {
		t.Errorf("AddOrUpdateIngress returned:  \n%v, but expected: \n%v", err, nil)
	}

	cnfHasIngress := cnf.HasIngress(ingress.Ingress)
	if !cnfHasIngress {
		t.Errorf("AddOrUpdateIngress didn't add ingress successfully. HasIngress returned %v, expected %v", cnfHasIngress, true)
	}
}

func TestAddOrUpdateMergeableIngress(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngess := createMergeableCafeIngress()

	err = cnf.AddOrUpdateMergeableIngress(mergeableIngess)
	if err != nil {
		t.Errorf("AddOrUpdateMergeableIngress returned \n%v, expected \n%v", err, nil)
	}

	cnfHasMergeableIngress := cnf.HasIngress(mergeableIngess.Master.Ingress)
	if !cnfHasMergeableIngress {
		t.Errorf("AddOrUpdateMergeableIngress didn't add mergeable ingress successfully. HasIngress returned %v, expected %v", cnfHasMergeableIngress, true)
	}
}

func TestAddOrUpdateIngressFailsWithInvalidIngressTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()

	err = cnf.AddOrUpdateIngress(&ingress)
	if err == nil {
		t.Errorf("AddOrUpdateIngressFailsWithInvalidTemplate returned \n%v,  but expected \n%v", nil, "template execution error")
	}
}

func TestAddOrUpdateMergeableIngressFailsWithInvalidIngressTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngess := createMergeableCafeIngress()

	err = cnf.AddOrUpdateMergeableIngress(mergeableIngess)
	if err == nil {
		t.Errorf("AddOrUpdateMergeableIngress returned \n%v, but expected \n%v", nil, "template execution error")
	}
}

func TestUpdateEndpoints(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()
	ingresses := []*IngressEx{&ingress}

	err = cnf.UpdateEndpoints(ingresses)
	if err != nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", err, nil)
	}

	err = cnf.UpdateEndpoints(ingresses)
	if err != nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", err, nil)
	}
}

func TestUpdateEndpointsMergeableIngress(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngress := createMergeableCafeIngress()
	mergeableIngresses := []*MergeableIngresses{mergeableIngress}

	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
	if err != nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", err, nil)
	}

	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
	if err != nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", err, nil)
	}
}

func TestUpdateEndpointsFailsWithInvalidTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()
	ingresses := []*IngressEx{&ingress}

	err = cnf.UpdateEndpoints(ingresses)
	if err == nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", nil, "template execution error")
	}
}

func TestUpdateEndpointsMergeableIngressFailsWithInvalidTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngress := createMergeableCafeIngress()
	mergeableIngresses := []*MergeableIngresses{mergeableIngress}

	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
	if err == nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", nil, "template execution error")
	}
}

func TestGetVirtualServerConfigFileName(t *testing.T) {
	vs := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "test",
			Name:      "virtual-server",
		},
	}

	expected := "vs_test_virtual-server"

	result := getFileNameForVirtualServer(&vs)
	if result != expected {
		t.Errorf("getFileNameForVirtualServer returned %v, but expected %v", result, expected)
	}
}

func TestGetFileNameForVirtualServerFromKey(t *testing.T) {
	key := "default/cafe"

	expected := "vs_default_cafe"

	result := getFileNameForVirtualServerFromKey(key)
	if result != expected {
		t.Errorf("getFileNameForVirtualServerFromKey returned %v, but expected %v", result, expected)
	}
}

func TestCheckIfListenerExists(t *testing.T) {
	tests := []struct {
		listener conf_v1alpha1.TransportServerListener
		expected bool
		msg      string
	}{
		{
			listener: conf_v1alpha1.TransportServerListener{
				Name:     "tcp-listener",
				Protocol: "TCP",
			},
			expected: true,
			msg:      "name and protocol match",
		},
		{
			listener: conf_v1alpha1.TransportServerListener{
				Name:     "some-listener",
				Protocol: "TCP",
			},
			expected: false,
			msg:      "only protocol matches",
		},
		{
			listener: conf_v1alpha1.TransportServerListener{
				Name:     "tcp-listener",
				Protocol: "UDP",
			},
			expected: false,
			msg:      "only name matches",
		},
	}

	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	cnf.globalCfgParams.Listeners = map[string]Listener{
		"tcp-listener": {
			Port:     53,
			Protocol: "TCP",
		},
	}

	for _, test := range tests {
		result := cnf.CheckIfListenerExists(&test.listener)
		if result != test.expected {
			t.Errorf("CheckIfListenerExists() returned %v but expected %v for the case of %q", result, test.expected, test.msg)
		}
	}
}

func TestGetFileNameForTransportServer(t *testing.T) {
	transportServer := &conf_v1alpha1.TransportServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "default",
			Name:      "test-server",
		},
	}

	expected := "ts_default_test-server"

	result := getFileNameForTransportServer(transportServer)
	if result != expected {
		t.Errorf("getFileNameForTransportServer() returned %q but expected %q", result, expected)
	}
}

func TestGetFileNameForTransportServerFromKey(t *testing.T) {
	key := "default/test-server"

	expected := "ts_default_test-server"

	result := getFileNameForTransportServerFromKey(key)
	if result != expected {
		t.Errorf("getFileNameForTransportServerFromKey(%q) returned %q but expected %q", key, result, expected)
	}
}

func TestGenerateNamespaceNameKey(t *testing.T) {
	objectMeta := &meta_v1.ObjectMeta{
		Namespace: "default",
		Name:      "test-server",
	}

	expected := "default/test-server"

	result := generateNamespaceNameKey(objectMeta)
	if result != expected {
		t.Errorf("generateNamespaceNameKey() returned %q but expected %q", result, expected)
	}
}

func TestUpdateGlobalConfiguration(t *testing.T) {
	globalConfiguration := &conf_v1alpha1.GlobalConfiguration{
		Spec: conf_v1alpha1.GlobalConfigurationSpec{
			Listeners: []conf_v1alpha1.Listener{
				{
					Name:     "tcp-listener",
					Port:     53,
					Protocol: "TCP",
				},
			},
		},
	}

	tsExTCP := &TransportServerEx{
		TransportServer: &conf_v1alpha1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "tcp-server",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.TransportServerSpec{
				Listener: conf_v1alpha1.TransportServerListener{
					Name:     "tcp-listener",
					Protocol: "TCP",
				},
				Upstreams: []conf_v1alpha1.Upstream{
					{
						Name:    "tcp-app",
						Service: "tcp-app-svc",
						Port:    5001,
					},
				},
				Action: &conf_v1alpha1.Action{
					Pass: "tcp-app",
				},
			},
		},
	}

	tsExUDP := &TransportServerEx{
		TransportServer: &conf_v1alpha1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "udp-server",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.TransportServerSpec{
				Listener: conf_v1alpha1.TransportServerListener{
					Name:     "udp-listener",
					Protocol: "UDP",
				},
				Upstreams: []conf_v1alpha1.Upstream{
					{
						Name:    "udp-app",
						Service: "udp-app-svc",
						Port:    5001,
					},
				},
				Action: &conf_v1alpha1.Action{
					Pass: "udp-app",
				},
			},
		},
	}

	cnf, err := createTestConfigurator()
	if err != nil {
		t.Fatalf("Failed to create a test configurator: %v", err)
	}

	transportServerExes := []*TransportServerEx{tsExTCP, tsExUDP}

	expectedUpdatedTransportServerExes := []*TransportServerEx{tsExTCP}
	expectedDeletedTransportServerExes := []*TransportServerEx{tsExUDP}

	updatedTransportServerExes, deletedTransportServerExes, err := cnf.UpdateGlobalConfiguration(globalConfiguration, transportServerExes)

	if !reflect.DeepEqual(updatedTransportServerExes, expectedUpdatedTransportServerExes) {
		t.Errorf("UpdateGlobalConfiguration() returned %v but expected %v", updatedTransportServerExes, expectedUpdatedTransportServerExes)
	}
	if !reflect.DeepEqual(deletedTransportServerExes, expectedDeletedTransportServerExes) {
		t.Errorf("UpdateGlobalConfiguration() returned %v but expected %v", deletedTransportServerExes, expectedDeletedTransportServerExes)
	}
	if err != nil {
		t.Errorf("UpdateGlobalConfiguration() returned an unexpected error %v", err)
	}

}

package configs

import (
	"os"
	"reflect"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	networking "k8s.io/api/networking/v1beta1"
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

	return NewConfigurator(manager, createTestStaticConfigParams(), NewDefaultConfigParams(), NewDefaultGlobalConfigParams(), templateExecutor, templateExecutorV2, false, false, nil, false), nil
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

	return NewConfigurator(manager, createTestStaticConfigParams(), NewDefaultConfigParams(), NewDefaultGlobalConfigParams(), templateExecutor, &version2.TemplateExecutor{}, false, false, nil, false), nil
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

func TestGenerateTLSPassthroughHostsConfig(t *testing.T) {
	tlsPassthroughPairs := map[string]tlsPassthroughPair{
		"default/ts-1": {
			Host:       "app.example.com",
			UnixSocket: "socket1.sock",
		},
		"default/ts-2": {
			Host:       "app.example.com",
			UnixSocket: "socket2.sock",
		},
		"default/ts-3": {
			Host:       "some.example.com",
			UnixSocket: "socket3.sock",
		},
	}

	expectedCfg := &version2.TLSPassthroughHostsConfig{
		"app.example.com":  "socket2.sock",
		"some.example.com": "socket3.sock",
	}
	expectedDuplicatedHosts := []string{"app.example.com"}

	resultCfg, resultDuplicatedHosts := generateTLSPassthroughHostsConfig(tlsPassthroughPairs)
	if !reflect.DeepEqual(resultCfg, expectedCfg) {
		t.Errorf("generateTLSPassthroughHostsConfig() returned %v but expected %v", resultCfg, expectedCfg)
	}

	if !reflect.DeepEqual(resultDuplicatedHosts, expectedDuplicatedHosts) {
		t.Errorf("generateTLSPassthroughHostsConfig() returned %v but expected %v", resultDuplicatedHosts, expectedDuplicatedHosts)
	}
}

func TestAddInternalRouteConfig(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}
	// set pod name in env
	err = os.Setenv("POD_NAME", "nginx-ingress")
	if err != nil {
		t.Errorf("Failed to set pod name in environment: %v", err)
	}
	err = cnf.AddInternalRouteConfig()
	if err != nil {
		t.Errorf("AddInternalRouteConfig returned:  \n%v, but expected: \n%v", err, nil)
	}

	if !cnf.staticCfgParams.EnableInternalRoutes {
		t.Errorf("AddInternalRouteConfig failed to set EnableInteralRoutes field of staticCfgParams to true")
	}
	if cnf.staticCfgParams.PodName != "nginx-ingress" {
		t.Errorf("AddInternalRouteConfig failed to set PodName field of staticCfgParams")
	}
}

func TestFindRemovedKeys(t *testing.T) {
	tests := []struct {
		currentKeys []string
		newKeys     map[string]bool
		expected    []string
	}{
		{
			currentKeys: []string{"key1", "key2"},
			newKeys:     map[string]bool{"key1": true, "key2": true},
			expected:    nil,
		},
		{
			currentKeys: []string{"key1", "key2"},
			newKeys:     map[string]bool{"key2": true, "key3": true},
			expected:    []string{"key1"},
		},
		{
			currentKeys: []string{"key1", "key2"},
			newKeys:     map[string]bool{"key3": true, "key4": true},
			expected:    []string{"key1", "key2"},
		},
		{
			currentKeys: []string{"key1", "key2"},
			newKeys:     map[string]bool{"key3": true},
			expected:    []string{"key1", "key2"},
		},
	}
	for _, test := range tests {
		result := findRemovedKeys(test.currentKeys, test.newKeys)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("findRemovedKeys(%v, %v) returned %v but expected %v", test.currentKeys, test.newKeys, result, test.expected)
		}
	}
}

func TestCreateUpstreamServerLabels(t *testing.T) {
	expected := []string{"coffee-svc", "ingress", "cafe", "default"}
	result := createUpstreamServerLabels("coffee-svc", "ingress", "cafe", "default")
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("createUpstreamServerLabels(%v, %v, %v, %v) returned %v but expected %v", "coffee-svc", "ingress", "cafe", "default", result, expected)
	}
}

func TestCreateServerZoneLabels(t *testing.T) {
	expected := []string{"ingress", "cafe", "default"}
	result := createServerZoneLabels("ingress", "cafe", "default")
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("createServerZoneLabels(%v, %v, %v) returned %v but expected %v", "ingress", "cafe", "default", result, expected)
	}
}

type mockLabelUpdater struct {
	upstreamServerLabels     map[string][]string
	serverZoneLabels         map[string][]string
	upstreamServerPeerLabels map[string][]string
}

func newFakeLabelUpdater() *mockLabelUpdater {
	return &mockLabelUpdater{
		upstreamServerLabels:     make(map[string][]string),
		serverZoneLabels:         make(map[string][]string),
		upstreamServerPeerLabels: make(map[string][]string),
	}
}

// UpdateUpstreamServerPeerLabels updates the Upstream Server Peer Labels
func (u *mockLabelUpdater) UpdateUpstreamServerPeerLabels(upstreamServerPeerLabels map[string][]string) {
	for k, v := range upstreamServerPeerLabels {
		u.upstreamServerPeerLabels[k] = v
	}
}

// DeleteUpstreamServerPeerLabels deletes the Upstream Server Peer Labels
func (u *mockLabelUpdater) DeleteUpstreamServerPeerLabels(peers []string) {
	for _, k := range peers {
		delete(u.upstreamServerPeerLabels, k)
	}
}

// UpdateUpstreamServerLabels updates the Upstream Server Labels
func (u *mockLabelUpdater) UpdateUpstreamServerLabels(upstreamServerLabelValues map[string][]string) {
	for k, v := range upstreamServerLabelValues {
		u.upstreamServerLabels[k] = v
	}
}

// DeleteUpstreamServerLabels deletes the Upstream Server Labels
func (u *mockLabelUpdater) DeleteUpstreamServerLabels(upstreamNames []string) {
	for _, k := range upstreamNames {
		delete(u.upstreamServerLabels, k)
	}
}

// UpdateServerZoneLabels updates the Server Zone Labels
func (u *mockLabelUpdater) UpdateServerZoneLabels(serverZoneLabelValues map[string][]string) {
	for k, v := range serverZoneLabelValues {
		u.serverZoneLabels[k] = v
	}
}

// DeleteServerZoneLabels deletes the Server Zone Labels
func (u *mockLabelUpdater) DeleteServerZoneLabels(zoneNames []string) {
	for _, k := range zoneNames {
		delete(u.serverZoneLabels, k)
	}
}

func TestUpdateIngressMetricsLabels(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Fatalf("Failed to create a test configurator: %v", err)
	}

	cnf.isPlus = true
	cnf.labelUpdater = newFakeLabelUpdater()

	ingEx := &IngressEx{
		Ingress: &networking.Ingress{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
			},
			Spec: networking.IngressSpec{
				Rules: []networking.IngressRule{
					{
						Host: "example.com",
					},
				},
			},
		},
		PodsByIP: map[string]string{
			"10.0.0.1:80": "pod-1",
			"10.0.0.2:80": "pod-2",
		},
	}

	upstreams := []version1.Upstream{
		{
			Name: "upstream-1",
			UpstreamServers: []version1.UpstreamServer{
				{
					Address: "10.0.0.1",
					Port:    "80",
				},
			},
			UpstreamLabels: version1.UpstreamLabels{
				Service:           "service-1",
				ResourceType:      "ingress",
				ResourceName:      ingEx.Ingress.Name,
				ResourceNamespace: ingEx.Ingress.Namespace,
			},
		},
		{
			Name: "upstream-2",
			UpstreamServers: []version1.UpstreamServer{
				{
					Address: "10.0.0.2",
					Port:    "80",
				},
			},
			UpstreamLabels: version1.UpstreamLabels{
				Service:           "service-2",
				ResourceType:      "ingress",
				ResourceName:      ingEx.Ingress.Name,
				ResourceNamespace: ingEx.Ingress.Namespace,
			},
		},
	}

	expectedLabelUpdater := &mockLabelUpdater{
		upstreamServerLabels: map[string][]string{
			"upstream-1": {"service-1", "ingress", "test-ingress", "default"},
			"upstream-2": {"service-2", "ingress", "test-ingress", "default"},
		},
		serverZoneLabels: map[string][]string{
			"example.com": {"ingress", "test-ingress", "default"},
		},
		upstreamServerPeerLabels: map[string][]string{
			"upstream-1/10.0.0.1:80": {"pod-1"},
			"upstream-2/10.0.0.2:80": {"pod-2"},
		},
	}

	// add labels for a new Ingress resource
	cnf.updateIngressMetricsLabels(ingEx, upstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateIngressMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}

	updatedUpstreams := []version1.Upstream{
		{
			Name: "upstream-1",
			UpstreamServers: []version1.UpstreamServer{
				{
					Address: "10.0.0.1",
					Port:    "80",
				},
			},
			UpstreamLabels: version1.UpstreamLabels{
				Service:           "service-1",
				ResourceType:      "ingress",
				ResourceName:      ingEx.Ingress.Name,
				ResourceNamespace: ingEx.Ingress.Namespace,
			},
		},
	}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels: map[string][]string{
			"upstream-1": {"service-1", "ingress", "test-ingress", "default"},
		},
		serverZoneLabels: map[string][]string{
			"example.com": {"ingress", "test-ingress", "default"},
		},
		upstreamServerPeerLabels: map[string][]string{
			"upstream-1/10.0.0.1:80": {"pod-1"},
		},
	}

	// update labels for an updated Ingress with deleted upstream-2
	cnf.updateIngressMetricsLabels(ingEx, updatedUpstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateIngressMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels:     map[string][]string{},
		serverZoneLabels:         map[string][]string{},
		upstreamServerPeerLabels: map[string][]string{},
	}

	// delete labels for a deleted Ingress
	cnf.deleteIngressMetricsLabels("default/test-ingress")
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("deleteIngressMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}
}

func TestUpdateVirtualServerMetricsLabels(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Fatalf("Failed to create a test configurator: %v", err)
	}

	cnf.isPlus = true
	cnf.labelUpdater = newFakeLabelUpdater()

	vsEx := &VirtualServerEx{
		VirtualServer: &conf_v1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "test-vs",
				Namespace: "default",
			},
			Spec: conf_v1.VirtualServerSpec{
				Host: "example.com",
			},
		},
		PodsByIP: map[string]string{
			"10.0.0.1:80": "pod-1",
			"10.0.0.2:80": "pod-2",
		},
	}

	upstreams := []version2.Upstream{
		{
			Name: "upstream-1",
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.1:80",
				},
			},
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "service-1",
				ResourceType:      "virtualserver",
				ResourceName:      vsEx.VirtualServer.Name,
				ResourceNamespace: vsEx.VirtualServer.Namespace,
			},
		},
		{
			Name: "upstream-2",
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.2:80",
				},
			},
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "service-2",
				ResourceType:      "virtualserver",
				ResourceName:      vsEx.VirtualServer.Name,
				ResourceNamespace: vsEx.VirtualServer.Namespace,
			},
		},
	}

	expectedLabelUpdater := &mockLabelUpdater{
		upstreamServerLabels: map[string][]string{
			"upstream-1": {"service-1", "virtualserver", "test-vs", "default"},
			"upstream-2": {"service-2", "virtualserver", "test-vs", "default"},
		},
		serverZoneLabels: map[string][]string{
			"example.com": {"virtualserver", "test-vs", "default"},
		},
		upstreamServerPeerLabels: map[string][]string{
			"upstream-1/10.0.0.1:80": {"pod-1"},
			"upstream-2/10.0.0.2:80": {"pod-2"},
		},
	}

	// add labels for a new VirtualServer resource
	cnf.updateVirtualServerMetricsLabels(vsEx, upstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateVirtualServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}

	updatedUpstreams := []version2.Upstream{
		{
			Name: "upstream-1",
			Servers: []version2.UpstreamServer{
				{
					Address: "10.0.0.1:80",
				},
			},
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "service-1",
				ResourceType:      "virtualserver",
				ResourceName:      vsEx.VirtualServer.Name,
				ResourceNamespace: vsEx.VirtualServer.Namespace,
			},
		},
	}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels: map[string][]string{
			"upstream-1": {"service-1", "virtualserver", "test-vs", "default"},
		},
		serverZoneLabels: map[string][]string{
			"example.com": {"virtualserver", "test-vs", "default"},
		},
		upstreamServerPeerLabels: map[string][]string{
			"upstream-1/10.0.0.1:80": {"pod-1"},
		},
	}

	// update labels for an updated VirtualServer with deleted upstream-2
	cnf.updateVirtualServerMetricsLabels(vsEx, updatedUpstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateVirtualServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels:     map[string][]string{},
		serverZoneLabels:         map[string][]string{},
		upstreamServerPeerLabels: map[string][]string{},
	}

	// delete labels for a deleted VirtualServer
	cnf.deleteVirtualServerMetricsLabels("default/test-vs")
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("deleteVirtualServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}
}

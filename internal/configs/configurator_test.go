package configs

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/dos/v1beta1"
	api_v1 "k8s.io/api/core/v1"
)

func createTestStaticConfigParams() *StaticConfigParams {
	return &StaticConfigParams{
		HealthStatus:                   true,
		HealthStatusURI:                "/nginx-health",
		NginxStatus:                    true,
		NginxStatusAllowCIDRs:          []string{"127.0.0.1"},
		NginxStatusPort:                8080,
		StubStatusOverUnixSocketForOSS: false,
		NginxVersion:                   nginx.NewVersion("nginx version: nginx/1.25.3 (nginx-plus-r31)"),
	}
}

func createTestConfigurator(t *testing.T) *Configurator {
	t.Helper()
	templateExecutor, err := version1.NewTemplateExecutor("version1/nginx-plus.tmpl", "version1/nginx-plus.ingress.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	templateExecutorV2, err := version2.NewTemplateExecutor("version2/nginx-plus.virtualserver.tmpl", "version2/nginx-plus.transportserver.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	manager := nginx.NewFakeManager("/etc/nginx")
	cnf := NewConfigurator(ConfiguratorParams{
		NginxManager:            manager,
		StaticCfgParams:         createTestStaticConfigParams(),
		Config:                  NewDefaultConfigParams(context.Background(), false),
		TemplateExecutor:        templateExecutor,
		TemplateExecutorV2:      templateExecutorV2,
		LatencyCollector:        nil,
		LabelUpdater:            nil,
		IsPlus:                  false,
		IsWildcardEnabled:       false,
		IsPrometheusEnabled:     false,
		IsLatencyMetricsEnabled: false,
		NginxVersion:            nginx.NewVersion("nginx version: nginx/1.25.3 (nginx-plus-r31)"),
	})
	cnf.isReloadsEnabled = true
	return cnf
}

func createTestConfiguratorInvalidIngressTemplate(t *testing.T) *Configurator {
	t.Helper()
	templateExecutor, err := version1.NewTemplateExecutor("version1/nginx-plus.tmpl", "version1/nginx-plus.ingress.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	invalidIngressTemplate := "{{.Upstreams.This.Field.Does.Not.Exist}}"
	if err := templateExecutor.UpdateIngressTemplate(&invalidIngressTemplate); err != nil {
		t.Fatal(err)
	}

	manager := nginx.NewFakeManager("/etc/nginx")
	cnf := NewConfigurator(ConfiguratorParams{
		NginxManager:            manager,
		StaticCfgParams:         createTestStaticConfigParams(),
		Config:                  NewDefaultConfigParams(context.Background(), false),
		TemplateExecutor:        templateExecutor,
		TemplateExecutorV2:      &version2.TemplateExecutor{},
		LatencyCollector:        nil,
		LabelUpdater:            nil,
		IsPlus:                  false,
		IsWildcardEnabled:       false,
		IsPrometheusEnabled:     false,
		IsLatencyMetricsEnabled: false,
	})
	cnf.isReloadsEnabled = true
	return cnf
}

func TestConfiguratorUpdatesConfigWithNilCustomMainTemplate(t *testing.T) {
	t.Parallel()

	cnf := createTestConfigurator(t)
	warnings, err := cnf.UpdateConfig(&ConfigParams{
		MainTemplate: nil,
	}, ExtendedResources{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("Got warnings when updating config: %+v", warnings)
	}
	if cnf.CfgParams.MainTemplate != nil {
		t.Errorf("Want nil MainTemplate, got %+v\n", cnf.CfgParams.MainTemplate)
	}
}

func TestConfiguratorUpdatesConfigWithCustomMainTemplate(t *testing.T) {
	t.Parallel()

	cnf := createTestConfigurator(t)
	warnings, err := cnf.UpdateConfig(&ConfigParams{
		MainTemplate: &customTestMainTemplate,
	}, ExtendedResources{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Got warnings when updating config: %+v", warnings)
	}

	got := *cnf.CfgParams.MainTemplate
	want := customTestMainTemplate

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestConfiguratorUpdatesConfigWithNilCustomIngressTemplate(t *testing.T) {
	t.Parallel()

	cnf := createTestConfigurator(t)
	warnings, err := cnf.UpdateConfig(&ConfigParams{
		IngressTemplate: nil,
	}, ExtendedResources{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("Got warnings when updating config: %+v", warnings)
	}
	if cnf.CfgParams.IngressTemplate != nil {
		t.Errorf("Want nil MainTemplate, got %+v\n", cnf.CfgParams.IngressTemplate)
	}
}

func TestConfiguratorUpdatesConfigWithCustomIngressTemplate(t *testing.T) {
	t.Parallel()

	cnf := createTestConfigurator(t)
	warnings, err := cnf.UpdateConfig(&ConfigParams{
		IngressTemplate: &customTestIngressTemplate,
	}, ExtendedResources{})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Got warnings when updating config: %+v", warnings)
	}

	got := *cnf.CfgParams.IngressTemplate
	want := customTestIngressTemplate

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestAddOrUpdateIngress(t *testing.T) {
	t.Parallel()
	cnf := createTestConfigurator(t)

	ingress := createCafeIngressEx()

	warnings, err := cnf.AddOrUpdateIngress(&ingress)
	if err != nil {
		t.Errorf("AddOrUpdateIngress returned:  \n%v, but expected: \n%v", err, nil)
	}
	if len(warnings) != 0 {
		t.Errorf("AddOrUpdateIngress returned warnings: %v", warnings)
	}

	cnfHasIngress := cnf.HasIngress(ingress.Ingress)
	if !cnfHasIngress {
		t.Errorf("AddOrUpdateIngress didn't add ingress successfully. HasIngress returned %v, expected %v", cnfHasIngress, true)
	}
}

func TestAddOrUpdateMergeableIngress(t *testing.T) {
	t.Parallel()
	cnf := createTestConfigurator(t)

	mergeableIngress := createMergeableCafeIngress()

	warnings, err := cnf.AddOrUpdateMergeableIngress(mergeableIngress)
	if err != nil {
		t.Errorf("AddOrUpdateMergeableIngress returned \n%v, expected \n%v", err, nil)
	}
	if len(warnings) != 0 {
		t.Errorf("AddOrUpdateMergeableIngress returned warnings: %v", warnings)
	}

	cnfHasMergeableIngress := cnf.HasIngress(mergeableIngress.Master.Ingress)
	if !cnfHasMergeableIngress {
		t.Errorf("AddOrUpdateMergeableIngress didn't add mergeable ingress successfully. HasIngress returned %v, expected %v", cnfHasMergeableIngress, true)
	}
}

func TestAddOrUpdateIngressFailsWithInvalidIngressTemplate(t *testing.T) {
	t.Parallel()
	cnf := createTestConfiguratorInvalidIngressTemplate(t)

	ingress := createCafeIngressEx()

	warnings, err := cnf.AddOrUpdateIngress(&ingress)
	if err == nil {
		t.Errorf("AddOrUpdateIngress returned \n%v,  but expected \n%v", nil, "template execution error")
	}
	if len(warnings) != 0 {
		t.Errorf("AddOrUpdateIngress returned warnings: %v", warnings)
	}
}

func TestAddOrUpdateMergeableIngressFailsWithInvalidIngressTemplate(t *testing.T) {
	t.Parallel()
	cnf := createTestConfiguratorInvalidIngressTemplate(t)

	mergeableIngress := createMergeableCafeIngress()

	warnings, err := cnf.AddOrUpdateMergeableIngress(mergeableIngress)
	if err == nil {
		t.Errorf("AddOrUpdateMergeableIngress returned \n%v, but expected \n%v", nil, "template execution error")
	}
	if len(warnings) != 0 {
		t.Errorf("AddOrUpdateMergeableIngress returned warnings: %v", warnings)
	}
}

func TestUpdateEndpoints(t *testing.T) {
	t.Parallel()
	cnf := createTestConfigurator(t)

	ingress := createCafeIngressEx()
	ingresses := []*IngressEx{&ingress}

	err := cnf.UpdateEndpoints(ingresses)
	if err != nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", err, nil)
	}

	err = cnf.UpdateEndpoints(ingresses)
	if err != nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", err, nil)
	}
}

func TestUpdateEndpointsMergeableIngress(t *testing.T) {
	t.Parallel()
	cnf := createTestConfigurator(t)

	mergeableIngress := createMergeableCafeIngress()
	mergeableIngresses := []*MergeableIngresses{mergeableIngress}

	err := cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
	if err != nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", err, nil)
	}

	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
	if err != nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", err, nil)
	}
}

func TestUpdateEndpointsFailsWithInvalidTemplate(t *testing.T) {
	t.Parallel()
	cnf := createTestConfiguratorInvalidIngressTemplate(t)

	ingress := createCafeIngressEx()
	ingresses := []*IngressEx{&ingress}

	err := cnf.UpdateEndpoints(ingresses)
	if err == nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", nil, "template execution error")
	}
}

func TestUpdateEndpointsMergeableIngressFailsWithInvalidTemplate(t *testing.T) {
	t.Parallel()
	cnf := createTestConfiguratorInvalidIngressTemplate(t)

	mergeableIngress := createMergeableCafeIngress()
	mergeableIngresses := []*MergeableIngresses{mergeableIngress}

	err := cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
	if err == nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", nil, "template execution error")
	}
}

func TestGetVirtualServerConfigFileName(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	key := "default/cafe"

	expected := "vs_default_cafe"

	result := getFileNameForVirtualServerFromKey(key)
	if result != expected {
		t.Errorf("getFileNameForVirtualServerFromKey returned %v, but expected %v", result, expected)
	}
}

func TestGetFileNameForTransportServer(t *testing.T) {
	t.Parallel()
	transportServer := &conf_v1.TransportServer{
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
	t.Parallel()
	key := "default/test-server"

	expected := "ts_default_test-server"

	result := getFileNameForTransportServerFromKey(key)
	if result != expected {
		t.Errorf("getFileNameForTransportServerFromKey(%q) returned %q but expected %q", key, result, expected)
	}
}

func TestGenerateNamespaceNameKey(t *testing.T) {
	t.Parallel()
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

func TestGenerateTLSPassthroughHostsConfig(t *testing.T) {
	t.Parallel()
	tlsPassthroughPairs := map[string]tlsPassthroughPair{
		"default/ts-1": {
			Host:       "one.example.com",
			UnixSocket: "socket1.sock",
		},
		"default/ts-2": {
			Host:       "two.example.com",
			UnixSocket: "socket2.sock",
		},
	}

	expectedCfg := &version2.TLSPassthroughHostsConfig{
		"one.example.com": "socket1.sock",
		"two.example.com": "socket2.sock",
	}

	resultCfg := generateTLSPassthroughHostsConfig(tlsPassthroughPairs)
	if !reflect.DeepEqual(resultCfg, expectedCfg) {
		t.Errorf("generateTLSPassthroughHostsConfig() returned %v but expected %v", resultCfg, expectedCfg)
	}
}

func TestAddInternalRouteConfig(t *testing.T) {
	t.Parallel()
	cnf := createTestConfigurator(t)

	// set service account in env
	err := os.Setenv("POD_SERVICEACCOUNT", "nginx-ingress")
	if err != nil {
		t.Fatalf("Failed to set pod name in environment: %v", err)
	}
	// set namespace in env
	err = os.Setenv("POD_NAMESPACE", "default")
	if err != nil {
		t.Fatalf("Failed to set pod name in environment: %v", err)
	}

	err = cnf.AddInternalRouteConfig()
	if err != nil {
		t.Errorf("AddInternalRouteConfig returned:  \n%v, but expected: \n%v", err, nil)
	}

	if !cnf.staticCfgParams.EnableInternalRoutes {
		t.Error("AddInternalRouteConfig failed to set EnableInternalRoutes field of staticCfgParams to true")
	}
	if cnf.staticCfgParams.InternalRouteServerName != "nginx-ingress.default.svc" {
		t.Error("AddInternalRouteConfig failed to set InternalRouteServerName field of staticCfgParams")
	}
}

func TestFindRemovedKeys(t *testing.T) {
	t.Parallel()
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

type mockLabelUpdater struct {
	upstreamServerLabels           map[string][]string
	serverZoneLabels               map[string][]string
	upstreamServerPeerLabels       map[string][]string
	streamUpstreamServerPeerLabels map[string][]string
	streamUpstreamServerLabels     map[string][]string
	streamServerZoneLabels         map[string][]string
	cacheZoneLabels                map[string][]string
	workerPIDVariableLabels        map[string][]string
}

func newFakeLabelUpdater() *mockLabelUpdater {
	return &mockLabelUpdater{
		upstreamServerLabels:           make(map[string][]string),
		serverZoneLabels:               make(map[string][]string),
		upstreamServerPeerLabels:       make(map[string][]string),
		streamUpstreamServerPeerLabels: make(map[string][]string),
		streamUpstreamServerLabels:     make(map[string][]string),
		streamServerZoneLabels:         make(map[string][]string),
		cacheZoneLabels:                make(map[string][]string),
		workerPIDVariableLabels:        make(map[string][]string),
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

// UpdateStreamUpstreamServerPeerLabels updates the Upstream Server Peer Labels
func (u *mockLabelUpdater) UpdateStreamUpstreamServerPeerLabels(upstreamServerPeerLabels map[string][]string) {
	for k, v := range upstreamServerPeerLabels {
		u.streamUpstreamServerPeerLabels[k] = v
	}
}

// DeleteStreamUpstreamServerPeerLabels deletes the Upstream Server Peer Labels
func (u *mockLabelUpdater) DeleteStreamUpstreamServerPeerLabels(peers []string) {
	for _, k := range peers {
		delete(u.streamUpstreamServerPeerLabels, k)
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

// UpdateStreamUpstreamServerLabels updates the Stream Upstream Server Labels
func (u *mockLabelUpdater) UpdateStreamUpstreamServerLabels(streamUpstreamServerLabelValues map[string][]string) {
	for k, v := range streamUpstreamServerLabelValues {
		u.streamUpstreamServerLabels[k] = v
	}
}

// DeleteStreamUpstreamServerLabels deletes the Stream Upstream Server Labels
func (u *mockLabelUpdater) DeleteStreamUpstreamServerLabels(streamUpstreamServerNames []string) {
	for _, k := range streamUpstreamServerNames {
		delete(u.streamUpstreamServerLabels, k)
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

// UpdateStreamServerZoneLabels updates the Server Zone Labels
func (u *mockLabelUpdater) UpdateStreamServerZoneLabels(streamServerZoneLabelValues map[string][]string) {
	for k, v := range streamServerZoneLabelValues {
		u.streamServerZoneLabels[k] = v
	}
}

// DeleteStreamServerZoneLabels deletes the Server Zone Labels
func (u *mockLabelUpdater) DeleteStreamServerZoneLabels(zoneNames []string) {
	for _, k := range zoneNames {
		delete(u.streamServerZoneLabels, k)
	}
}

// UpdateCacheZoneLabels updates the Cache Zone Labels
func (u *mockLabelUpdater) UpdateCacheZoneLabels(cacheZoneLabelValues map[string][]string) {
	for k, v := range cacheZoneLabelValues {
		u.cacheZoneLabels[k] = v
	}
}

// DeleteCacheZoneLabels deletes the Cache Zone Labels
func (u *mockLabelUpdater) DeleteCacheZoneLabels(cacheZoneNames []string) {
	for _, k := range cacheZoneNames {
		delete(u.cacheZoneLabels, k)
	}
}

// UpdateWorkerLabels updates the Worker Labels
func (u *mockLabelUpdater) UpdateWorkerLabels(workerValues map[string][]string) {
	for k, v := range workerValues {
		u.workerPIDVariableLabels[k] = v
	}
}

// DeleteWorkerLabels deletes the Worker Labels
func (u *mockLabelUpdater) DeleteWorkerLabels(workerNames []string) {
	for _, k := range workerNames {
		delete(u.workerPIDVariableLabels, k)
	}
}

type mockLatencyCollector struct {
	upstreamServerLabels        map[string][]string
	upstreamServerPeerLabels    map[string][]string
	upstreamServerPeersToDelete []string
}

func newMockLatencyCollector() *mockLatencyCollector {
	return &mockLatencyCollector{
		upstreamServerLabels:     make(map[string][]string),
		upstreamServerPeerLabels: make(map[string][]string),
	}
}

// DeleteMetrics deletes metrics for the given upstream server peers
func (u *mockLatencyCollector) DeleteMetrics(upstreamServerPeerNames []string) {
	u.upstreamServerPeersToDelete = upstreamServerPeerNames
}

// UpdateUpstreamServerLabels updates the Upstream Server Labels
func (u *mockLatencyCollector) UpdateUpstreamServerLabels(upstreamServerLabelValues map[string][]string) {
	for k, v := range upstreamServerLabelValues {
		u.upstreamServerLabels[k] = v
	}
}

// DeleteUpstreamServerLabels deletes the Upstream Server Labels
func (u *mockLatencyCollector) DeleteUpstreamServerLabels(upstreamNames []string) {
	for _, k := range upstreamNames {
		delete(u.upstreamServerLabels, k)
	}
}

// UpdateUpstreamServerPeerLabels updates the Upstream Server Peer Labels
func (u *mockLatencyCollector) UpdateUpstreamServerPeerLabels(upstreamServerPeerLabels map[string][]string) {
	for k, v := range upstreamServerPeerLabels {
		u.upstreamServerPeerLabels[k] = v
	}
}

// DeleteUpstreamServerPeerLabels deletes the Upstream Server Peer Labels
func (u *mockLatencyCollector) DeleteUpstreamServerPeerLabels(peers []string) {
	for _, k := range peers {
		delete(u.upstreamServerPeerLabels, k)
	}
}

// RecordLatency implements a fake RecordLatency method
func (u *mockLatencyCollector) RecordLatency(string) {}

// Register implements a fake Register method
func (u *mockLatencyCollector) Register(*prometheus.Registry) error { return nil }

func TestUpdateIngressMetricsLabels(t *testing.T) {
	t.Parallel()
	cnf := createTestConfigurator(t)

	cnf.isPlus = true
	cnf.labelUpdater = newFakeLabelUpdater()
	testLatencyCollector := newMockLatencyCollector()
	cnf.latencyCollector = testLatencyCollector

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
		PodsByIP: map[string]PodInfo{
			"10.0.0.1:80": {Name: "pod-1"},
			"10.0.0.2:80": {Name: "pod-2"},
		},
	}

	upstreams := []version1.Upstream{
		{
			Name: "upstream-1",
			UpstreamServers: []version1.UpstreamServer{
				{
					Address: "10.0.0.1:80",
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
					Address: "10.0.0.2:80",
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
	upstreamServerLabels := map[string][]string{
		"upstream-1": {"service-1", "ingress", "test-ingress", "default"},
		"upstream-2": {"service-2", "ingress", "test-ingress", "default"},
	}
	upstreamServerPeerLabels := map[string][]string{
		"upstream-1/10.0.0.1:80": {"pod-1"},
		"upstream-2/10.0.0.2:80": {"pod-2"},
	}
	expectedLabelUpdater := &mockLabelUpdater{
		upstreamServerLabels: upstreamServerLabels,
		serverZoneLabels: map[string][]string{
			"example.com": {"ingress", "test-ingress", "default"},
		},
		upstreamServerPeerLabels:       upstreamServerPeerLabels,
		streamUpstreamServerPeerLabels: make(map[string][]string),
		streamUpstreamServerLabels:     make(map[string][]string),
		streamServerZoneLabels:         make(map[string][]string),
		cacheZoneLabels:                make(map[string][]string),
		workerPIDVariableLabels:        make(map[string][]string),
	}
	expectedLatencyCollector := &mockLatencyCollector{
		upstreamServerLabels:     upstreamServerLabels,
		upstreamServerPeerLabels: upstreamServerPeerLabels,
	}

	// add labels for a new Ingress resource
	cnf.updateIngressMetricsLabels(ingEx, upstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateIngressMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}
	if !reflect.DeepEqual(testLatencyCollector, expectedLatencyCollector) {
		t.Errorf("updateIngressMetricsLabels() updated latency collector labels to \n%+v but expected \n%+v", testLatencyCollector, expectedLatencyCollector)
	}

	updatedUpstreams := []version1.Upstream{
		{
			Name: "upstream-1",
			UpstreamServers: []version1.UpstreamServer{
				{
					Address: "10.0.0.1:80",
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

	upstreamServerLabels = map[string][]string{
		"upstream-1": {"service-1", "ingress", "test-ingress", "default"},
	}

	upstreamServerPeerLabels = map[string][]string{
		"upstream-1/10.0.0.1:80": {"pod-1"},
	}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels: upstreamServerLabels,
		serverZoneLabels: map[string][]string{
			"example.com": {"ingress", "test-ingress", "default"},
		},
		upstreamServerPeerLabels:       upstreamServerPeerLabels,
		streamUpstreamServerPeerLabels: make(map[string][]string),
		streamUpstreamServerLabels:     make(map[string][]string),
		streamServerZoneLabels:         make(map[string][]string),
		cacheZoneLabels:                make(map[string][]string),
		workerPIDVariableLabels:        make(map[string][]string),
	}
	expectedLatencyCollector = &mockLatencyCollector{
		upstreamServerLabels:        upstreamServerLabels,
		upstreamServerPeerLabels:    upstreamServerPeerLabels,
		upstreamServerPeersToDelete: []string{"upstream-2/10.0.0.2:80"},
	}

	// update labels for an updated Ingress with deleted upstream-2
	cnf.updateIngressMetricsLabels(ingEx, updatedUpstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateIngressMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}
	if !reflect.DeepEqual(testLatencyCollector, expectedLatencyCollector) {
		t.Errorf("updateIngressMetricsLabels() updated latency collector labels to \n%+v but expected \n%+v", testLatencyCollector, expectedLatencyCollector)
	}

	upstreamServerLabels = map[string][]string{}
	upstreamServerPeerLabels = map[string][]string{}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels:           map[string][]string{},
		serverZoneLabels:               map[string][]string{},
		upstreamServerPeerLabels:       map[string][]string{},
		streamUpstreamServerPeerLabels: map[string][]string{},
		streamUpstreamServerLabels:     map[string][]string{},
		streamServerZoneLabels:         map[string][]string{},
		cacheZoneLabels:                map[string][]string{},
		workerPIDVariableLabels:        map[string][]string{},
	}
	expectedLatencyCollector = &mockLatencyCollector{
		upstreamServerLabels:        upstreamServerLabels,
		upstreamServerPeerLabels:    upstreamServerPeerLabels,
		upstreamServerPeersToDelete: []string{"upstream-1/10.0.0.1:80"},
	}

	// delete labels for a deleted Ingress
	cnf.deleteIngressMetricsLabels("default/test-ingress")
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("deleteIngressMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}
	if !reflect.DeepEqual(testLatencyCollector, expectedLatencyCollector) {
		t.Errorf("updateIngressMetricsLabels() updated latency collector labels to \n%+v but expected \n%+v", testLatencyCollector, expectedLatencyCollector)
	}
}

func TestUpdateVirtualServerMetricsLabels(t *testing.T) {
	t.Parallel()
	cnf := createTestConfigurator(t)

	cnf.isPlus = true
	cnf.labelUpdater = newFakeLabelUpdater()
	testLatencyCollector := newMockLatencyCollector()
	cnf.latencyCollector = testLatencyCollector

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
		PodsByIP: map[string]PodInfo{
			"10.0.0.1:80": {Name: "pod-1"},
			"10.0.0.2:80": {Name: "pod-2"},
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

	upstreamServerLabels := map[string][]string{
		"upstream-1": {"service-1", "virtualserver", "test-vs", "default"},
		"upstream-2": {"service-2", "virtualserver", "test-vs", "default"},
	}

	upstreamServerPeerLabels := map[string][]string{
		"upstream-1/10.0.0.1:80": {"pod-1"},
		"upstream-2/10.0.0.2:80": {"pod-2"},
	}

	expectedLabelUpdater := &mockLabelUpdater{
		upstreamServerLabels: upstreamServerLabels,
		serverZoneLabels: map[string][]string{
			"example.com": {"virtualserver", "test-vs", "default"},
		},
		upstreamServerPeerLabels:       upstreamServerPeerLabels,
		streamUpstreamServerPeerLabels: map[string][]string{},
		streamUpstreamServerLabels:     map[string][]string{},
		streamServerZoneLabels:         map[string][]string{},
		cacheZoneLabels:                map[string][]string{},
		workerPIDVariableLabels:        map[string][]string{},
	}

	expectedLatencyCollector := &mockLatencyCollector{
		upstreamServerLabels:     upstreamServerLabels,
		upstreamServerPeerLabels: upstreamServerPeerLabels,
	}

	// add labels for a new VirtualServer resource
	cnf.updateVirtualServerMetricsLabels(vsEx, upstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateVirtualServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}
	if !reflect.DeepEqual(testLatencyCollector, expectedLatencyCollector) {
		t.Errorf("updateVirtualServerMetricsLabels() updated latency collector's labels to \n%+v but expected \n%+v", testLatencyCollector, expectedLatencyCollector)
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

	upstreamServerLabels = map[string][]string{
		"upstream-1": {"service-1", "virtualserver", "test-vs", "default"},
	}
	upstreamServerPeerLabels = map[string][]string{
		"upstream-1/10.0.0.1:80": {"pod-1"},
	}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels: upstreamServerLabels,
		serverZoneLabels: map[string][]string{
			"example.com": {"virtualserver", "test-vs", "default"},
		},
		upstreamServerPeerLabels:       upstreamServerPeerLabels,
		streamUpstreamServerPeerLabels: map[string][]string{},
		streamUpstreamServerLabels:     map[string][]string{},
		streamServerZoneLabels:         map[string][]string{},
		cacheZoneLabels:                map[string][]string{},
		workerPIDVariableLabels:        map[string][]string{},
	}

	expectedLatencyCollector = &mockLatencyCollector{
		upstreamServerLabels:        upstreamServerLabels,
		upstreamServerPeerLabels:    upstreamServerPeerLabels,
		upstreamServerPeersToDelete: []string{"upstream-2/10.0.0.2:80"},
	}

	// update labels for an updated VirtualServer with deleted upstream-2
	cnf.updateVirtualServerMetricsLabels(vsEx, updatedUpstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateVirtualServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}
	if !reflect.DeepEqual(testLatencyCollector, expectedLatencyCollector) {
		t.Errorf("updateVirtualServerMetricsLabels() updated latency collector's labels to \n%+v but expected \n%+v", testLatencyCollector, expectedLatencyCollector)
	}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels:           map[string][]string{},
		serverZoneLabels:               map[string][]string{},
		upstreamServerPeerLabels:       map[string][]string{},
		streamUpstreamServerPeerLabels: map[string][]string{},
		streamUpstreamServerLabels:     map[string][]string{},
		streamServerZoneLabels:         map[string][]string{},
		cacheZoneLabels:                map[string][]string{},
		workerPIDVariableLabels:        map[string][]string{},
	}

	expectedLatencyCollector = &mockLatencyCollector{
		upstreamServerLabels:        map[string][]string{},
		upstreamServerPeerLabels:    map[string][]string{},
		upstreamServerPeersToDelete: []string{"upstream-1/10.0.0.1:80"},
	}

	// delete labels for a deleted VirtualServer
	cnf.deleteVirtualServerMetricsLabels("default/test-vs")
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("deleteVirtualServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}

	if !reflect.DeepEqual(testLatencyCollector, expectedLatencyCollector) {
		t.Errorf("updateVirtualServerMetricsLabels() updated latency collector's labels to \n%+v but expected \n%+v", testLatencyCollector, expectedLatencyCollector)
	}
}

func TestUpdateTransportServerMetricsLabels(t *testing.T) {
	t.Parallel()
	cnf := createTestConfigurator(t)

	cnf.isPlus = true
	cnf.labelUpdater = newFakeLabelUpdater()

	tsEx := &TransportServerEx{
		TransportServer: &conf_v1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "test-transportserver",
				Namespace: "default",
			},
			Spec: conf_v1.TransportServerSpec{
				Listener: conf_v1.TransportServerListener{
					Name:     "dns-tcp",
					Protocol: "TCP",
				},
			},
		},
		PodsByIP: map[string]string{
			"10.0.0.1:80": "pod-1",
			"10.0.0.2:80": "pod-2",
		},
	}

	streamUpstreams := []version2.StreamUpstream{
		{
			Name: "upstream-1",
			Servers: []version2.StreamUpstreamServer{
				{
					Address: "10.0.0.1:80",
				},
			},
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "service-1",
				ResourceType:      "transportserver",
				ResourceName:      tsEx.TransportServer.Name,
				ResourceNamespace: tsEx.TransportServer.Namespace,
			},
		},
		{
			Name: "upstream-2",
			Servers: []version2.StreamUpstreamServer{
				{
					Address: "10.0.0.2:80",
				},
			},
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "service-2",
				ResourceType:      "transportserver",
				ResourceName:      tsEx.TransportServer.Name,
				ResourceNamespace: tsEx.TransportServer.Namespace,
			},
		},
	}

	streamUpstreamServerLabels := map[string][]string{
		"upstream-1": {"service-1", "transportserver", "test-transportserver", "default"},
		"upstream-2": {"service-2", "transportserver", "test-transportserver", "default"},
	}

	streamUpstreamServerPeerLabels := map[string][]string{
		"upstream-1/10.0.0.1:80": {"pod-1"},
		"upstream-2/10.0.0.2:80": {"pod-2"},
	}

	expectedLabelUpdater := &mockLabelUpdater{
		streamUpstreamServerLabels: streamUpstreamServerLabels,
		streamServerZoneLabels: map[string][]string{
			"dns-tcp": {"transportserver", "test-transportserver", "default"},
		},
		streamUpstreamServerPeerLabels: streamUpstreamServerPeerLabels,
		upstreamServerPeerLabels:       make(map[string][]string),
		upstreamServerLabels:           make(map[string][]string),
		serverZoneLabels:               make(map[string][]string),
		cacheZoneLabels:                make(map[string][]string),
		workerPIDVariableLabels:        make(map[string][]string),
	}

	cnf.updateTransportServerMetricsLabels(tsEx, streamUpstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateTransportServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}

	updatedStreamUpstreams := []version2.StreamUpstream{
		{
			Name: "upstream-1",
			Servers: []version2.StreamUpstreamServer{
				{
					Address: "10.0.0.1:80",
				},
			},
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "service-1",
				ResourceType:      "transportserver",
				ResourceName:      tsEx.TransportServer.Name,
				ResourceNamespace: tsEx.TransportServer.Namespace,
			},
		},
	}

	streamUpstreamServerLabels = map[string][]string{
		"upstream-1": {"service-1", "transportserver", "test-transportserver", "default"},
	}

	streamUpstreamServerPeerLabels = map[string][]string{
		"upstream-1/10.0.0.1:80": {"pod-1"},
	}

	expectedLabelUpdater = &mockLabelUpdater{
		streamUpstreamServerLabels: streamUpstreamServerLabels,
		streamServerZoneLabels: map[string][]string{
			"dns-tcp": {"transportserver", "test-transportserver", "default"},
		},
		streamUpstreamServerPeerLabels: streamUpstreamServerPeerLabels,
		upstreamServerPeerLabels:       map[string][]string{},
		upstreamServerLabels:           map[string][]string{},
		serverZoneLabels:               map[string][]string{},
		cacheZoneLabels:                map[string][]string{},
		workerPIDVariableLabels:        map[string][]string{},
	}

	cnf.updateTransportServerMetricsLabels(tsEx, updatedStreamUpstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateTransportServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels:           map[string][]string{},
		serverZoneLabels:               map[string][]string{},
		upstreamServerPeerLabels:       map[string][]string{},
		streamUpstreamServerPeerLabels: map[string][]string{},
		streamUpstreamServerLabels:     map[string][]string{},
		streamServerZoneLabels:         map[string][]string{},
		cacheZoneLabels:                map[string][]string{},
		workerPIDVariableLabels:        map[string][]string{},
	}

	cnf.deleteTransportServerMetricsLabels("default/test-transportserver")
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("deleteTransportServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}

	tsExTLS := &TransportServerEx{
		TransportServer: &conf_v1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "test-transportserver-tls",
				Namespace: "default",
			},
			Spec: conf_v1.TransportServerSpec{
				Listener: conf_v1.TransportServerListener{
					Name:     "tls-passthrough",
					Protocol: "TLS_PASSTHROUGH",
				},
				Host: "example.com",
			},
		},
		PodsByIP: map[string]string{
			"10.0.0.3:80": "pod-3",
		},
	}

	streamUpstreams = []version2.StreamUpstream{
		{
			Name: "upstream-3",
			Servers: []version2.StreamUpstreamServer{
				{
					Address: "10.0.0.3:80",
				},
			},
			UpstreamLabels: version2.UpstreamLabels{
				Service:           "service-3",
				ResourceType:      "transportserver",
				ResourceName:      tsExTLS.TransportServer.Name,
				ResourceNamespace: tsExTLS.TransportServer.Namespace,
			},
		},
	}

	streamUpstreamServerLabels = map[string][]string{
		"upstream-3": {"service-3", "transportserver", "test-transportserver-tls", "default"},
	}

	streamUpstreamServerPeerLabels = map[string][]string{
		"upstream-3/10.0.0.3:80": {"pod-3"},
	}

	expectedLabelUpdater = &mockLabelUpdater{
		streamUpstreamServerLabels: streamUpstreamServerLabels,
		streamServerZoneLabels: map[string][]string{
			"example.com": {"transportserver", "test-transportserver-tls", "default"},
		},
		streamUpstreamServerPeerLabels: streamUpstreamServerPeerLabels,
		upstreamServerPeerLabels:       make(map[string][]string),
		upstreamServerLabels:           make(map[string][]string),
		serverZoneLabels:               make(map[string][]string),
		cacheZoneLabels:                make(map[string][]string),
		workerPIDVariableLabels:        make(map[string][]string),
	}

	cnf.updateTransportServerMetricsLabels(tsExTLS, streamUpstreams)
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("updateTransportServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}

	expectedLabelUpdater = &mockLabelUpdater{
		upstreamServerLabels:           map[string][]string{},
		serverZoneLabels:               map[string][]string{},
		upstreamServerPeerLabels:       map[string][]string{},
		streamUpstreamServerPeerLabels: map[string][]string{},
		streamUpstreamServerLabels:     map[string][]string{},
		streamServerZoneLabels:         map[string][]string{},
		cacheZoneLabels:                map[string][]string{},
		workerPIDVariableLabels:        map[string][]string{},
	}

	cnf.deleteTransportServerMetricsLabels("default/test-transportserver-tls")
	if !reflect.DeepEqual(cnf.labelUpdater, expectedLabelUpdater) {
		t.Errorf("deleteTransportServerMetricsLabels() updated labels to \n%+v but expected \n%+v", cnf.labelUpdater, expectedLabelUpdater)
	}
}

func TestUpdateApResources(t *testing.T) {
	t.Parallel()
	conf := createTestConfigurator(t)

	appProtectPolicy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "test-ns",
				"name":      "test-name",
			},
		},
	}
	appProtectLogConf := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "test-ns",
				"name":      "test-name",
			},
		},
	}
	appProtectLogDst := "test-dst"

	tests := []struct {
		ingEx    *IngressEx
		expected *AppProtectResources
		msg      string
	}{
		{
			ingEx: &IngressEx{
				Ingress: &networking.Ingress{
					ObjectMeta: meta_v1.ObjectMeta{},
				},
			},
			expected: &AppProtectResources{},
			msg:      "no app protect resources",
		},
		{
			ingEx: &IngressEx{
				Ingress: &networking.Ingress{
					ObjectMeta: meta_v1.ObjectMeta{},
				},
				AppProtectPolicy: appProtectPolicy,
			},
			expected: &AppProtectResources{
				AppProtectPolicy: "/etc/nginx/waf/nac-policies/test-ns_test-name",
			},
			msg: "app protect policy",
		},
		{
			ingEx: &IngressEx{
				Ingress: &networking.Ingress{
					ObjectMeta: meta_v1.ObjectMeta{},
				},
				AppProtectLogs: []AppProtectLog{
					{
						LogConf: appProtectLogConf,
						Dest:    appProtectLogDst,
					},
				},
			},
			expected: &AppProtectResources{
				AppProtectLogconfs: []string{"/etc/nginx/waf/nac-logconfs/test-ns_test-name test-dst"},
			},
			msg: "app protect log conf",
		},
		{
			ingEx: &IngressEx{
				Ingress: &networking.Ingress{
					ObjectMeta: meta_v1.ObjectMeta{},
				},
				AppProtectPolicy: appProtectPolicy,
				AppProtectLogs: []AppProtectLog{
					{
						LogConf: appProtectLogConf,
						Dest:    appProtectLogDst,
					},
				},
			},
			expected: &AppProtectResources{
				AppProtectPolicy:   "/etc/nginx/waf/nac-policies/test-ns_test-name",
				AppProtectLogconfs: []string{"/etc/nginx/waf/nac-logconfs/test-ns_test-name test-dst"},
			},
			msg: "app protect policy and log conf",
		},
	}

	for _, test := range tests {
		result := conf.updateApResources(test.ingEx)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("updateApResources() returned \n%v but expected\n%v for the case of %s", result, test.expected, test.msg)
		}
	}
}

func TestUpdateApResourcesForVs(t *testing.T) {
	t.Parallel()
	conf := createTestConfigurator(t)

	apPolRefs := map[string]*unstructured.Unstructured{
		"test-ns-1/test-name-1": {
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"namespace": "test-ns-1",
					"name":      "test-name-1",
				},
			},
		},
		"test-ns-2/test-name-2": {
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"namespace": "test-ns-2",
					"name":      "test-name-2",
				},
			},
		},
	}
	logConfRefs := map[string]*unstructured.Unstructured{
		"test-ns-1/test-name-1": {
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"namespace": "test-ns-1",
					"name":      "test-name-1",
				},
			},
		},
		"test-ns-2/test-name-2": {
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"namespace": "test-ns-2",
					"name":      "test-name-2",
				},
			},
		},
	}

	tests := []struct {
		vsEx     *VirtualServerEx
		expected *appProtectResourcesForVS
		msg      string
	}{
		{
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{},
				},
			},
			expected: &appProtectResourcesForVS{
				Policies: map[string]string{},
				LogConfs: map[string]string{},
			},
			msg: "no app protect resources",
		},
		{
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{},
				},
				ApPolRefs: apPolRefs,
			},
			expected: &appProtectResourcesForVS{
				Policies: map[string]string{
					"test-ns-1/test-name-1": "/etc/nginx/waf/nac-policies/test-ns-1_test-name-1",
					"test-ns-2/test-name-2": "/etc/nginx/waf/nac-policies/test-ns-2_test-name-2",
				},
				LogConfs: map[string]string{},
			},
			msg: "app protect policies",
		},
		{
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{},
				},
				LogConfRefs: logConfRefs,
			},
			expected: &appProtectResourcesForVS{
				Policies: map[string]string{},
				LogConfs: map[string]string{
					"test-ns-1/test-name-1": "/etc/nginx/waf/nac-logconfs/test-ns-1_test-name-1",
					"test-ns-2/test-name-2": "/etc/nginx/waf/nac-logconfs/test-ns-2_test-name-2",
				},
			},
			msg: "app protect log confs",
		},
		{
			vsEx: &VirtualServerEx{
				VirtualServer: &conf_v1.VirtualServer{
					ObjectMeta: meta_v1.ObjectMeta{},
				},
				ApPolRefs:   apPolRefs,
				LogConfRefs: logConfRefs,
			},
			expected: &appProtectResourcesForVS{
				Policies: map[string]string{
					"test-ns-1/test-name-1": "/etc/nginx/waf/nac-policies/test-ns-1_test-name-1",
					"test-ns-2/test-name-2": "/etc/nginx/waf/nac-policies/test-ns-2_test-name-2",
				},
				LogConfs: map[string]string{
					"test-ns-1/test-name-1": "/etc/nginx/waf/nac-logconfs/test-ns-1_test-name-1",
					"test-ns-2/test-name-2": "/etc/nginx/waf/nac-logconfs/test-ns-2_test-name-2",
				},
			},
			msg: "app protect policies and log confs",
		},
	}

	for _, test := range tests {
		result := conf.updateApResourcesForVs(test.vsEx)
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("updateApResourcesForVs() '%s' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestUpstreamsForHost_ReturnsNilForNoVirtualServers(t *testing.T) {
	t.Parallel()

	tcnf := createTestConfigurator(t)
	tcnf.virtualServers = map[string]*VirtualServerEx{
		"vs": invalidVirtualServerEx,
	}

	got := tcnf.UpstreamsForHost("tea.example.com")
	if got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestUpstreamsForHost_DoesNotReturnUpstreamsOnBogusHostname(t *testing.T) {
	t.Parallel()

	tcnf := createTestConfigurator(t)
	tcnf.virtualServers = map[string]*VirtualServerEx{
		"vs": validVirtualServerExWithUpstreams,
	}

	got := tcnf.UpstreamsForHost("bogus.host.org")
	if got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestUpstreamsForHost_ReturnsUpstreamsNamesForValidHostname(t *testing.T) {
	t.Parallel()
	tcnf := createTestConfigurator(t)
	tcnf.virtualServers = map[string]*VirtualServerEx{
		"vs": validVirtualServerExWithUpstreams,
	}

	want := []string{"vs_default_test-vs_tea-app"}
	got := tcnf.UpstreamsForHost("tea.example.com")
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestStreamUpstreamsForName_DoesNotReturnUpstreamsForBogusName(t *testing.T) {
	t.Parallel()

	tcnf := createTestConfigurator(t)
	tcnf.transportServers = map[string]*TransportServerEx{
		"ts": validTransportServerExWithUpstreams,
	}

	got := tcnf.StreamUpstreamsForName("bogus-service-name")
	if got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestStreamUpstreamsForName_ReturnsStreamUpstreamsNamesOnValidServiceName(t *testing.T) {
	t.Parallel()

	tcnf := createTestConfigurator(t)
	tcnf.transportServers = map[string]*TransportServerEx{
		"ts": validTransportServerExWithUpstreams,
	}

	want := []string{"ts_default_secure-app_secure-app"}
	got := tcnf.StreamUpstreamsForName("secure-app")
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestGetIngressAnnotations(t *testing.T) {
	t.Parallel()

	tcnf := createTestConfigurator(t)

	ingress := &IngressEx{
		Ingress: &networking.Ingress{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
				Annotations: map[string]string{
					"appprotect.f5.com/app-protect-enable": "False",
					"nginx.org/proxy-set-header":           "X-Forwarded-ABC",
					"ingress.kubernetes.io/ssl-redirect":   "True",
				},
			},
		},
	}

	_, err := tcnf.AddOrUpdateIngress(ingress)
	if err != nil {
		t.Fatalf("AddOrUpdateIngress returned error: %v", err)
	}

	annotationList := tcnf.GetIngressAnnotations()

	expectedAnnotations := []string{
		"appprotect.f5.com/app-protect-enable",
		"nginx.org/proxy-set-header",
		"ingress.kubernetes.io/ssl-redirect",
	}

	if len(annotationList) != len(expectedAnnotations) {
		t.Errorf("got %d annotations, want %d", len(annotationList), len(expectedAnnotations))
	}

	foundAnnotations := make(map[string]bool)
	for _, annotation := range annotationList {
		foundAnnotations[annotation] = true
	}

	for _, expected := range expectedAnnotations {
		if !foundAnnotations[expected] {
			t.Errorf("expected annotation %q not found", expected)
		}
	}
}

func TestGetInvalidIngressAnnotations(t *testing.T) {
	t.Parallel()

	tcnf := createTestConfigurator(t)

	ingress := &IngressEx{
		Ingress: &networking.Ingress{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
				Annotations: map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": "s",
					"alb.ingress.kubernetes.io/group.order":            "0",
					"alb.ingress.kubernetes.io/ip-address-type":        "ipv4",
					"alb.ingress.kubernetes.io/scheme":                 "internal",
				},
			},
		},
	}

	_, err := tcnf.AddOrUpdateIngress(ingress)
	if err != nil {
		t.Fatalf("AddOrUpdateIngress returned error: %v", err)
	}

	expectedAnnotations := []string{
		"alb.ingress.kubernetes.io/scheme",
		"alb.ingress.kubernetes.io/group.order",
		"alb.ingress.kubernetes.io/ip-address-type",
	}

	annotationList := tcnf.GetIngressAnnotations()

	foundAnnotations := make(map[string]bool)
	for _, annotation := range annotationList {
		foundAnnotations[annotation] = true
	}

	for _, expected := range expectedAnnotations {
		if foundAnnotations[expected] {
			t.Errorf("expected annotation %q not found", expected)
		}
	}
}

func TestGetMixedIngressAnnotations(t *testing.T) {
	t.Parallel()

	tcnf := createTestConfigurator(t)

	ingress := &IngressEx{
		Ingress: &networking.Ingress{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
				Annotations: map[string]string{
					"kubectl.kubernetes.io/last-applied-configuration": "s",
					"alb.ingress.kubernetes.io/group.order":            "0",
					"alb.ingress.kubernetes.io/ip-address-type":        "ipv4",
					"alb.ingress.kubernetes.io/scheme":                 "internal",
					"appprotect.f5.com/app-protect-enable":             "False",
					"nginx.org/proxy-set-header":                       "X-Forwarded-ABC",
					"ingress.kubernetes.io/ssl-redirect":               "True",
				},
			},
		},
	}

	_, err := tcnf.AddOrUpdateIngress(ingress)
	if err != nil {
		t.Fatalf("AddOrUpdateIngress returned error: %v", err)
	}

	expectedAnnotations := []string{
		"ingress.kubernetes.io/ssl-redirect",
		"nginx.org/proxy-set-header",
		"appprotect.f5.com/app-protect-enable",
	}

	annotationList := tcnf.GetIngressAnnotations()

	foundAnnotations := make(map[string]bool)
	for _, annotation := range annotationList {
		foundAnnotations[annotation] = true
	}

	for _, expected := range expectedAnnotations {
		if !foundAnnotations[expected] {
			t.Errorf("expected annotation %q not found", expected)
		}
	}
}

func TestGetVitualServerCountsReportsNumberOfVSAndVSR(t *testing.T) {
	t.Parallel()

	tcnf := createTestConfigurator(t)
	tcnf.virtualServers = map[string]*VirtualServerEx{
		"vs": validVirtualServerExWithUpstreams,
	}

	gotVS, gotVSRoutes := tcnf.GetVirtualServerCounts()
	wantVS, wantVSRoutes := 1, 0

	if gotVS != wantVS {
		t.Errorf("GetVirtualServerCounts() = %d, %d, want %d, %d", gotVS, gotVSRoutes, wantVS, wantVSRoutes)
	}
	if gotVSRoutes != wantVSRoutes {
		t.Errorf("GetVirtualServerCounts() = %d, %d, want %d, %d", gotVS, gotVSRoutes, wantVS, wantVSRoutes)
	}
}

func TestGetVitualServerCountsNotExistingVS(t *testing.T) {
	t.Parallel()

	tcnf := createTestConfigurator(t)
	tcnf.virtualServers = nil

	gotVS, gotVSRoutes := tcnf.GetVirtualServerCounts()
	wantVS, wantVSRoutes := 0, 0

	if gotVS != wantVS {
		t.Errorf("GetVirtualServerCounts() = %d, %d, want %d, %d", gotVS, gotVSRoutes, wantVS, wantVSRoutes)
	}
	if gotVSRoutes != wantVSRoutes {
		t.Errorf("GetVirtualServerCounts() = %d, %d, want %d, %d", gotVS, gotVSRoutes, wantVS, wantVSRoutes)
	}
}

func TestAddOrUpdateTransportServer(t *testing.T) {
	t.Parallel()
	cnf := createTestConfigurator(t)

	ts := createTransportServerExWithHostNoTLSPassthrough()

	warnings, err := cnf.AddOrUpdateTransportServer(&ts)
	if err != nil {
		t.Errorf("AddOrUpdateTransportServer returned:  \n%v, but expected: \n%v", err, nil)
	}
	if len(warnings) != 0 {
		t.Errorf("AddOrUpdateTransportServer returned warnings: %v", warnings)
	}
}

var (
	invalidVirtualServerEx = &VirtualServerEx{
		VirtualServer: &conf_v1.VirtualServer{},
	}

	validVirtualServerExWithUpstreams = &VirtualServerEx{
		VirtualServer: &conf_v1.VirtualServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "test-vs",
				Namespace: "default",
			},
			Spec: conf_v1.VirtualServerSpec{
				Host: "tea.example.com",
				Upstreams: []conf_v1.Upstream{
					{
						Name: "tea-app",
					},
				},
			},
		},
	}

	validTransportServerExWithUpstreams = &TransportServerEx{
		TransportServer: &conf_v1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "secure-app",
				Namespace: "default",
			},
			Spec: conf_v1.TransportServerSpec{
				Listener: conf_v1.TransportServerListener{
					Name:     "tls-passthrough",
					Protocol: "TLS_PASSTHROUGH",
				},
				Host: "example.com",
				Upstreams: []conf_v1.TransportServerUpstream{
					{
						Name:    "secure-app",
						Service: "secure-app",
						Port:    8443,
					},
				},
				Action: &conf_v1.TransportServerAction{
					Pass: "secure-app",
				},
			},
		},
	}
)

func TestGenerateApDosAllowListFileContent(t *testing.T) {
	tests := []struct {
		name      string
		allowList []v1beta1.AllowListEntry
		want      []byte
		wantErr   bool
	}{
		{
			name:      "Empty allow list",
			allowList: []v1beta1.AllowListEntry{},
			want:      []byte(`{"policy":{"ip-address-lists":[{"ipAddresses":[],"blockRequests":"transparent"}]}}`),
			wantErr:   false,
		},
		{
			name: "Single valid IPv4 entry",
			allowList: []v1beta1.AllowListEntry{
				{IPWithMask: "192.168.1.1/32"},
			},
			want:    []byte(`{"policy":{"ip-address-lists":[{"ipAddresses":[{"ipAddress":"192.168.1.1/32"}],"blockRequests":"transparent"}]}}`),
			wantErr: false,
		},
		{
			name: "Single valid IPv6 entry",
			allowList: []v1beta1.AllowListEntry{
				{IPWithMask: "2001:0db8:85a3:0000:0000:8a2e:0370:7334/128"},
			},
			want:    []byte(`{"policy":{"ip-address-lists":[{"ipAddresses":[{"ipAddress":"2001:0db8:85a3:0000:0000:8a2e:0370:7334/128"}],"blockRequests":"transparent"}]}}`),
			wantErr: false,
		},
		{
			name: "Multiple valid entries",
			allowList: []v1beta1.AllowListEntry{
				{IPWithMask: "192.168.1.1/32"},
				{IPWithMask: "2001:0db8:85a3:0000:0000:8a2e:0370:7334/128"},
			},
			want:    []byte(`{"policy":{"ip-address-lists":[{"ipAddresses":[{"ipAddress":"192.168.1.1/32"},{"ipAddress":"2001:0db8:85a3:0000:0000:8a2e:0370:7334/128"}],"blockRequests":"transparent"}]}}`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateApDosAllowListFileContent(tt.allowList)
			if (got == nil) != tt.wantErr {
				t.Errorf("generateApDosAllowListFileContent() error = %v, wantErr %v", got == nil, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				var gotFormatted, wantFormatted interface{}
				if err := json.Unmarshal(got, &gotFormatted); err != nil {
					t.Errorf("Failed to unmarshal got: %v", err)
				}
				if err := json.Unmarshal(tt.want, &wantFormatted); err != nil {
					t.Errorf("Failed to unmarshal want: %v", err)
				}
				t.Errorf("generateApDosAllowListFileContent() = \n%#v, \nwant \n%#v", gotFormatted, wantFormatted)
			}
		})
	}
}

func createTransportServerExWithHostNoTLSPassthrough() TransportServerEx {
	return TransportServerEx{
		SecretRefs: map[string]*secrets.SecretReference{
			"default/echo-secret": {
				Secret: &api_v1.Secret{
					Type: api_v1.SecretTypeTLS,
				},
				Path: "secret.pem",
			},
		},
		TransportServer: &conf_v1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "echo-app",
				Namespace: "default",
			},
			Spec: conf_v1.TransportServerSpec{
				Listener: conf_v1.TransportServerListener{
					Name:     "tcp-listener",
					Protocol: "TCP",
				},
				Host: "example.com",
				TLS: &conf_v1.TransportServerTLS{
					Secret: "echo-secret",
				},
				Upstreams: []conf_v1.TransportServerUpstream{
					{
						Name:    "echo-app",
						Service: "echo-app",
						Port:    7000,
					},
				},
				Action: &conf_v1.TransportServerAction{
					Pass: "echo-app",
				},
			},
		},
	}
}

var (
	// custom test Main Template represents a main-template passed via ConfigMap
	customTestMainTemplate = `# TEST NEW MAIN TEMPLATE
{{- /*gotype: github.com/nginxinc/kubernetes-ingress/internal/configs/version1.MainConfig*/ -}}
worker_processes  {{.WorkerProcesses}};
{{- if .WorkerRlimitNofile}}
worker_rlimit_nofile {{.WorkerRlimitNofile}};{{end}}
{{- if .WorkerCPUAffinity}}
worker_cpu_affinity {{.WorkerCPUAffinity}};{{end}}
{{- if .WorkerShutdownTimeout}}
worker_shutdown_timeout {{.WorkerShutdownTimeout}};{{end}}

daemon off;

error_log  stderr {{.ErrorLogLevel}};
pid        /var/lib/nginx/nginx.pid;

{{- if .OpenTracingLoadModule}}
load_module modules/ngx_http_opentracing_module.so;
{{- end}}
{{- if .AppProtectLoadModule}}
load_module modules/ngx_http_app_protect_module.so;
{{- end}}
{{- if .AppProtectDosLoadModule}}
load_module modules/ngx_http_app_protect_dos_module.so;
{{- end}}
load_module modules/ngx_fips_check_module.so;
{{- if .MainSnippets}}
{{range $value := .MainSnippets}}
{{$value}}{{end}}
{{- end}}

load_module modules/ngx_http_js_module.so;

events {
    worker_connections  {{.WorkerConnections}};
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    map_hash_max_size {{.MapHashMaxSize}};
    map_hash_bucket_size {{.MapHashBucketSize}};

    js_import /etc/nginx/njs/apikey_auth.js;
    js_set $apikey_auth_hash apikey_auth.hash;

    {{- if .HTTPSnippets}}
    {{range $value := .HTTPSnippets}}
    {{$value}}{{end}}
    {{- end}}

    {{if .LogFormat -}}
    log_format  main {{if .LogFormatEscaping}}escape={{ .LogFormatEscaping }} {{end}}
                     {{range $i, $value := .LogFormat -}}
                     {{with $value}}'{{if $i}} {{end}}{{$value}}'
                     {{end}}{{end}};
    {{- else -}}
    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';
    {{- end}}

    map $upstream_trailer_grpc_status $grpc_status {
        default $upstream_trailer_grpc_status;
        '' $sent_http_grpc_status;
    }

    {{- if .DynamicSSLReloadEnabled }}
    map $nginx_version $secret_dir_path {
        default "{{ .StaticSSLPath }}";
    }
    {{- end }}

    {{- if .AppProtectDosLoadModule}}
    {{- if .AppProtectDosLogFormat}}
    log_format  log_dos {{if .AppProtectDosLogFormatEscaping}}escape={{ .AppProtectDosLogFormatEscaping }} {{end}}
                    {{range $i, $value := .AppProtectDosLogFormat -}}
                    {{with $value}}'{{if $i}} {{end}}{{$value}}'
                    {{end}}{{end}};
    {{- else }}
    log_format  log_dos ', vs_name_al=$app_protect_dos_vs_name, ip=$remote_addr, tls_fp=$app_protect_dos_tls_fp, '
                        'outcome=$app_protect_dos_outcome, reason=$app_protect_dos_outcome_reason, '
                        'ip_tls=$remote_addr:$app_protect_dos_tls_fp, ';

    {{- end}}
    {{- if .AppProtectDosArbFqdn}}
    app_protect_dos_arb_fqdn {{.AppProtectDosArbFqdn}};
    {{- end}}
    {{- end}}

    {{- if .AppProtectV5LoadModule}}
    app_protect_enforcer_address {{ .AppProtectV5EnforcerAddr }};
    {{- end}}

    access_log {{.AccessLog}};

    {{- if .LatencyMetrics}}
    log_format response_time '{"upstreamAddress":"$upstream_addr", "upstreamResponseTime":"$upstream_response_time", "proxyHost":"$proxy_host", "upstreamStatus": "$upstream_status"}';
    access_log syslog:server=unix:/var/lib/nginx/nginx-syslog.sock,nohostname,tag=nginx response_time;
    {{- end}}

    {{- if .AppProtectLoadModule}}
    {{if .AppProtectFailureModeAction}}app_protect_failure_mode_action {{.AppProtectFailureModeAction}};{{end}}
    {{if .AppProtectCompressedRequestsAction}}app_protect_compressed_requests_action {{.AppProtectCompressedRequestsAction}};{{end}}
    {{if .AppProtectCookieSeed}}app_protect_cookie_seed {{.AppProtectCookieSeed}};{{end}}
    {{if .AppProtectCPUThresholds}}app_protect_cpu_thresholds {{.AppProtectCPUThresholds}};{{end}}
    {{if .AppProtectPhysicalMemoryThresholds}}app_protect_physical_memory_util_thresholds {{.AppProtectPhysicalMemoryThresholds}};{{end}}
    {{if .AppProtectReconnectPeriod}}app_protect_reconnect_period_seconds {{.AppProtectReconnectPeriod}};{{end}}
    include /etc/nginx/waf/nac-usersigs/index.conf;
    {{- end}}

    sendfile        on;
    #tcp_nopush     on;

    keepalive_timeout {{.KeepaliveTimeout}};
    keepalive_requests {{.KeepaliveRequests}};

    #gzip  on;

    server_names_hash_max_size {{.ServerNamesHashMaxSize}};
    {{if .ServerNamesHashBucketSize}}server_names_hash_bucket_size {{.ServerNamesHashBucketSize}};{{end}}

    variables_hash_bucket_size {{.VariablesHashBucketSize}};
    variables_hash_max_size {{.VariablesHashMaxSize}};

    map $request_uri $request_uri_no_args {
        "~^(?P<path>[^?]*)(\?.*)?$" $path;
    }

    map $http_upgrade $connection_upgrade {
        default upgrade;
        ''      close;
    }
    map $http_upgrade $vs_connection_header {
        default upgrade;
        ''      $default_connection_header;
    }
    {{- if .SSLProtocols}}
    ssl_protocols {{.SSLProtocols}};
    {{- end}}
    {{- if .SSLCiphers}}
    ssl_ciphers "{{.SSLCiphers}}";
    {{- end}}
    {{- if .SSLPreferServerCiphers}}
    ssl_prefer_server_ciphers on;
    {{- end}}
    {{- if .SSLDHParam}}
    ssl_dhparam {{.SSLDHParam}};
    {{- end}}

    {{- if .OpenTracingEnabled}}
    opentracing on;
    {{- end}}
    {{- if .OpenTracingLoadModule}}
    opentracing_load_tracer {{ .OpenTracingTracer }} /var/lib/nginx/tracer-config.json;
    {{- end}}

    {{- if .ResolverAddresses}}
    resolver {{range $resolver := .ResolverAddresses}}{{$resolver}}{{end}}{{if .ResolverValid}} valid={{.ResolverValid}}{{end}}{{if not .ResolverIPV6}} ipv6=off{{end}};
    {{- if .ResolverTimeout}}resolver_timeout {{.ResolverTimeout}};{{end}}
    {{- end}}

    {{- if .OIDC}}
    include oidc/oidc_common.conf;
    {{- end}}

    server {
        # required to support the Websocket protocol in VirtualServer/VirtualServerRoutes
        set $default_connection_header "";
        set $resource_type "";
        set $resource_name "";
        set $resource_namespace "";
        set $service "";

        listen {{ .DefaultHTTPListenerPort }} default_server{{if .ProxyProtocol}} proxy_protocol{{end}};
        {{- if not .DisableIPV6}}listen [::]:{{ .DefaultHTTPListenerPort }} default_server{{if .ProxyProtocol}} proxy_protocol{{end}};{{end}}

        {{- if .TLSPassthrough}}
        listen unix:/var/lib/nginx/passthrough-https.sock ssl default_server proxy_protocol;
        set_real_ip_from unix:;
        real_ip_header proxy_protocol;
        {{- else}}
        listen {{ .DefaultHTTPSListenerPort }} ssl default_server{{if .ProxyProtocol}} proxy_protocol{{end}};
        {{if not .DisableIPV6}}listen [::]:{{ .DefaultHTTPSListenerPort }} ssl default_server{{if .ProxyProtocol}} proxy_protocol{{end}};{{end}}
        {{- end}}

        {{- if .HTTP2}}
        http2 on;
        {{- end}}

        {{- if .SSLRejectHandshake}}
        ssl_reject_handshake on;
        {{- else}}
        ssl_certificate {{ makeSecretPath "/etc/nginx/secrets/default" .StaticSSLPath "$secret_dir_path" .DynamicSSLReloadEnabled }};
        ssl_certificate_key {{ makeSecretPath "/etc/nginx/secrets/default" .StaticSSLPath "$secret_dir_path" .DynamicSSLReloadEnabled }};
        {{- end}}

        {{- range $setRealIPFrom := .SetRealIPFrom}}
        set_real_ip_from {{$setRealIPFrom}};{{end}}
        {{- if .RealIPHeader}}real_ip_header {{.RealIPHeader}};{{end}}
        {{- if .RealIPRecursive}}real_ip_recursive on;{{end}}

        server_name _;
        server_tokens "{{.ServerTokens}}";
        {{- if .DefaultServerAccessLogOff}}
        access_log off;
        {{end -}}

        {{- if .OpenTracingEnabled}}
        opentracing off;
        {{- end}}

        {{- if .HealthStatus}}
        location {{.HealthStatusURI}} {
            default_type text/plain;
            return 200 "healthy\n";
        }
        {{end}}

        location / {
            return {{.DefaultServerReturn}};
        }
    }

    {{- if .NginxStatus}}
    # NGINX Plus APIs
    server {
        listen {{.NginxStatusPort}};
        {{if not .DisableIPV6}}listen [::]:{{.NginxStatusPort}};{{end}}

        root /usr/share/nginx/html;

        access_log off;

        {{if .OpenTracingEnabled}}
        opentracing off;
        {{end}}

        location  = /dashboard.html {
        }
        {{if .AppProtectDosLoadModule}}
        location = /dashboard-dos.html {
        }
        {{end}}
        {{range $value := .NginxStatusAllowCIDRs}}
        allow {{$value}};{{end}}

        deny all;
        location /api {
            {{if .AppProtectDosLoadModule}}
            app_protect_dos_api on;
            {{end}}
            api write=off;
        }
    }
    {{- end}}

    # NGINX Plus API over unix socket
    server {
        listen unix:/var/lib/nginx/nginx-plus-api.sock;
        access_log off;

        {{- if .OpenTracingEnabled}}
        opentracing off;
        {{- end}}

        # $config_version_mismatch is defined in /etc/nginx/config-version.conf
        location /configVersionCheck {
            if ($config_version_mismatch) {
                return 503;
            }
            return 200;
        }

        location /api {
            api write=on;
        }
    }

    include /etc/nginx/config-version.conf;
    include /etc/nginx/conf.d/*.conf;

    server {
        listen unix:/var/lib/nginx/nginx-418-server.sock;
        access_log off;

        {{- if .OpenTracingEnabled}}
        opentracing off;
        {{- end -}}

        return 418;
    }
    {{- if .InternalRouteServer}}
    server {
        listen 443 ssl;
        {{if not .DisableIPV6}}listen [::]:443 ssl;{{end}}
        server_name {{.InternalRouteServerName}};
        ssl_certificate {{ makeSecretPath "/etc/nginx/secrets/spiffe_cert.pem" .StaticSSLPath "$secret_dir_path" .DynamicSSLReloadEnabled }};
        ssl_certificate_key {{ makeSecretPath "/etc/nginx/secrets/spiffe_key.pem" .StaticSSLPath "$secret_dir_path" .DynamicSSLReloadEnabled }};
        ssl_client_certificate /etc/nginx/secrets/spiffe_rootca.pem;
        ssl_verify_client on;
        ssl_verify_depth 25;
    }
    {{- end}}
}

stream {
    {{if .StreamLogFormat -}}
    log_format  stream-main {{if .StreamLogFormatEscaping}}escape={{ .StreamLogFormatEscaping }} {{end}}
                            {{range $i, $value := .StreamLogFormat -}}
                            {{with $value}}'{{if $i}} {{end}}{{$value}}'
                            {{end}}{{end}};
    {{- else -}}
    log_format  stream-main  '$remote_addr [$time_local] '
                      '$protocol $status $bytes_sent $bytes_received '
                      '$session_time "$ssl_preread_server_name"';
    {{- end}}

    access_log  /dev/stdout  stream-main;

    {{- range $value := .StreamSnippets}}
    {{$value}}{{end}}

    {{- if .ResolverAddresses}}
    resolver {{range $resolver := .ResolverAddresses}}{{$resolver}}{{end}}{{if .ResolverValid}} valid={{.ResolverValid}}{{end}}{{if not .ResolverIPV6}} ipv6=off{{end}};
    {{if .ResolverTimeout}}resolver_timeout {{.ResolverTimeout}};{{end}}
    {{- end}}

    map_hash_max_size {{.MapHashMaxSize}};
    {{if .MapHashBucketSize}}map_hash_bucket_size {{.MapHashBucketSize}};{{end}}

    {{- if .DynamicSSLReloadEnabled }}
    map $nginx_version $secret_dir_path {
        default "{{ .StaticSSLPath }}";
    }
    {{- end }}

    {{- if .TLSPassthrough}}
    map $ssl_preread_server_name $dest_internal_passthrough  {
        default unix:/var/lib/nginx/passthrough-https.sock;
        include /etc/nginx/tls-passthrough-hosts.conf;
    }

    server {
        listen {{.TLSPassthroughPort}}{{if .ProxyProtocol}} proxy_protocol{{end}};
        {{if not .DisableIPV6}}listen [::]:{{.TLSPassthroughPort}}{{if .ProxyProtocol}} proxy_protocol{{end}};{{end}}

        {{if .ProxyProtocol}}
        {{range $setRealIPFrom := .SetRealIPFrom}}
        set_real_ip_from {{$setRealIPFrom}};{{end}}
        {{end}}

        ssl_preread on;

        proxy_protocol on;
        proxy_pass $dest_internal_passthrough;
    }
    {{end}}

    include /etc/nginx/stream-conf.d/*.conf;
}

{{- if (.NginxVersion.PlusGreaterThanOrEqualTo "nginx-plus-r31") }}
mgmt {
    usage_report interval=0s;
}
{{- end}}
`

	// custom test Ingress Template represents an ingress-template passed via ConfigMap
	customTestIngressTemplate = `# TEST NEW CUSTOM INGRESS TEMPLATE
{{- /*gotype: github.com/nginxinc/kubernetes-ingress/internal/configs/version1.IngressNginxConfig*/ -}}
# configuration for {{.Ingress.Namespace}}/{{.Ingress.Name}}
{{- range $upstream := .Upstreams}}
upstream {{$upstream.Name}} {
	{{- if ne $upstream.UpstreamZoneSize "0"}}zone {{$upstream.Name}} {{$upstream.UpstreamZoneSize}};{{end}}
	{{- if $upstream.LBMethod }}
	{{$upstream.LBMethod}};
	{{- end}}
	{{- range $server := $upstream.UpstreamServers}}
	server {{$server.Address}} max_fails={{$server.MaxFails}} fail_timeout={{$server.FailTimeout}} max_conns={{$server.MaxConns}};{{end}}
	{{- if $.Keepalive}}keepalive {{$.Keepalive}};{{end}}
}
{{end -}}

{{range $limitReqZone := .LimitReqZones}}
limit_req_zone {{ $limitReqZone.Key }} zone={{ $limitReqZone.Name }}:{{$limitReqZone.Size}} rate={{$limitReqZone.Rate}};
{{end}}

{{range $server := .Servers}}
server {
	{{- if $server.SpiffeCerts}}
	listen 443 ssl;
	{{- if not $server.DisableIPV6}}listen [::]:443 ssl;{{end}}
	ssl_certificate {{ makeSecretPath "/etc/nginx/secrets/spiffe_cert.pem" $.StaticSSLPath "$secret_dir_path" $.DynamicSSLReloadEnabled }};
	ssl_certificate_key {{ makeSecretPath "/etc/nginx/secrets/spiffe_key.pem" $.StaticSSLPath "$secret_dir_path" $.DynamicSSLReloadEnabled }};
	{{- else}}
	{{- if not $server.GRPCOnly}}
	{{- range $port := $server.Ports}}
	listen {{$port}}{{if $server.ProxyProtocol}} proxy_protocol{{end}};
	{{- if not $server.DisableIPV6}}listen [::]:{{$port}}{{if $server.ProxyProtocol}} proxy_protocol{{end}};{{end}}
	{{- end}}
	{{- end}}

	{{- if $server.SSL}}
	{{- if $server.TLSPassthrough}}
	listen unix:/var/lib/nginx/passthrough-https.sock ssl proxy_protocol;
	set_real_ip_from unix:;
	real_ip_header proxy_protocol;
	{{- else}}
	{{- range $port := $server.SSLPorts}}
	listen {{$port}} ssl{{if $server.ProxyProtocol}} proxy_protocol{{end}};
	{{- if not $server.DisableIPV6}}listen [::]:{{$port}} ssl{{if $server.ProxyProtocol}} proxy_protocol{{end}};{{end}}
	{{- end}}
	{{- end}}
	{{- if $server.HTTP2}}
	http2 on;
	{{- end}}
	{{- if $server.SSLRejectHandshake}}
	ssl_reject_handshake on;
	{{- else}}
	ssl_certificate {{ makeSecretPath $server.SSLCertificate $.StaticSSLPath "$secret_dir_path" $.DynamicSSLReloadEnabled }};
	ssl_certificate_key {{ makeSecretPath $server.SSLCertificateKey $.StaticSSLPath "$secret_dir_path" $.DynamicSSLReloadEnabled }};
	{{- end}}
	{{- end}}
	{{- end}}

	{{- range $setRealIPFrom := $server.SetRealIPFrom}}
	set_real_ip_from {{$setRealIPFrom}};{{end}}
	{{- if $server.RealIPHeader}}real_ip_header {{$server.RealIPHeader}};{{end}}
	{{- if $server.RealIPRecursive}}real_ip_recursive on;{{end}}

	server_tokens {{$server.ServerTokens}};

	server_name {{$server.Name}};

	set $resource_type "ingress";
	set $resource_name "{{$.Ingress.Name}}";
	set $resource_namespace "{{$.Ingress.Namespace}}";

	{{- range $proxyHideHeader := $server.ProxyHideHeaders}}
	proxy_hide_header {{$proxyHideHeader}};{{end}}
	{{- range $proxyPassHeader := $server.ProxyPassHeaders}}
	proxy_pass_header {{$proxyPassHeader}};{{end}}

	{{- if and $server.HSTS (or $server.SSL $server.HSTSBehindProxy)}}
	set $hsts_header_val "";
	proxy_hide_header Strict-Transport-Security;
	{{- if $server.HSTSBehindProxy}}
	if ($http_x_forwarded_proto = 'https') {
	{{- else}}
	if ($https = on) {
	{{- end}}
		set $hsts_header_val "max-age={{$server.HSTSMaxAge}}; {{if $server.HSTSIncludeSubdomains}}includeSubDomains; {{end}}preload";
	}

	add_header Strict-Transport-Security "$hsts_header_val" always;
	{{- end}}

	{{- if $server.SSL}}
	{{- if not $server.GRPCOnly}}
	{{- if $server.SSLRedirect}}
	if ($scheme = http) {
		return 301 https://$host:{{index $server.SSLPorts 0}}$request_uri;
	}
	{{- end}}
	{{- end}}
	{{- end}}

	{{- if $server.RedirectToHTTPS}}
	if ($http_x_forwarded_proto = 'http') {
		return 301 https://$host$request_uri;
	}
	{{- end}}

	{{- with $server.BasicAuth }}
	auth_basic {{ printf "%q" .Realm }};
	auth_basic_user_file {{ .Secret }};
	{{- end }}

	{{- if $server.ServerSnippets}}
	{{- range $value := $server.ServerSnippets}}
	{{$value}}{{end}}
	{{- end}}

	{{- range $location := $server.Locations}}
	location {{  makeLocationPath $location $.Ingress.Annotations | printf }} {
		set $service "{{$location.ServiceName}}";
		{{- with $location.MinionIngress}}
		# location for minion {{$location.MinionIngress.Namespace}}/{{$location.MinionIngress.Name}}
		set $resource_name "{{$location.MinionIngress.Name}}";
		set $resource_namespace "{{$location.MinionIngress.Namespace}}";
		{{- end}}
		{{- if $location.GRPC}}
		{{- if not $server.GRPCOnly}}
		error_page 400 @grpcerror400;
		error_page 401 @grpcerror401;
		error_page 403 @grpcerror403;
		error_page 404 @grpcerror404;
		error_page 405 @grpcerror405;
		error_page 408 @grpcerror408;
		error_page 414 @grpcerror414;
		error_page 426 @grpcerror426;
		error_page 500 @grpcerror500;
		error_page 501 @grpcerror501;
		error_page 502 @grpcerror502;
		error_page 503 @grpcerror503;
		error_page 504 @grpcerror504;
		{{- end}}

		{{- if $location.LocationSnippets}}
		{{- range $value := $location.LocationSnippets}}
		{{$value}}{{end}}
		{{- end}}

		{{- with $location.BasicAuth }}
		auth_basic {{ printf "%q" .Realm }};
		auth_basic_user_file {{ .Secret }};
		{{- end }}

		grpc_connect_timeout {{$location.ProxyConnectTimeout}};
		grpc_read_timeout {{$location.ProxyReadTimeout}};
		grpc_send_timeout {{$location.ProxySendTimeout}};
		grpc_set_header Host $host;
		grpc_set_header X-Real-IP $remote_addr;
		grpc_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		grpc_set_header X-Forwarded-Host $host;
		grpc_set_header X-Forwarded-Port $server_port;
		grpc_set_header X-Forwarded-Proto {{if $server.RedirectToHTTPS}}https{{else}}$scheme{{end}};

		{{- if $location.ProxyBufferSize}}
		grpc_buffer_size {{$location.ProxyBufferSize}};
		{{- end}}
		{{- if $.SpiffeClientCerts}}
		grpc_ssl_certificate {{ makeSecretPath "/etc/nginx/secrets/spiffe_cert.pem" $.StaticSSLPath "$secret_dir_path" $.DynamicSSLReloadEnabled }};
		grpc_ssl_certificate_key {{ makeSecretPath "/etc/nginx/secrets/spiffe_key.pem" $.StaticSSLPath "$secret_dir_path" $.DynamicSSLReloadEnabled }};
		grpc_ssl_trusted_certificate /etc/nginx/secrets/spiffe_rootca.pem;
		grpc_ssl_server_name on;
		grpc_ssl_verify on;
		grpc_ssl_verify_depth 25;
		grpc_ssl_name {{$location.ProxySSLName}};
		{{- end}}
		{{- if $location.SSL}}
		grpc_pass grpcs://{{$location.Upstream.Name}}{{$location.Rewrite}};
		{{- else}}
		grpc_pass grpc://{{$location.Upstream.Name}}{{$location.Rewrite}};
		{{- end}}
		{{- else}}
		proxy_http_version 1.1;
		{{- if $location.Websocket}}
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection $connection_upgrade;
		{{- else}}
		{{- if $.Keepalive}}
		proxy_set_header Connection "";{{end}}
		{{- end}}
		{{- if $location.LocationSnippets}}
		{{range $value := $location.LocationSnippets}}
		{{$value}}{{end}}
		{{- end}}
		{{- with $location.BasicAuth }}
		auth_basic {{ printf "%q" .Realm }};
		auth_basic_user_file {{ .Secret }};
		{{- end }}
		proxy_connect_timeout {{$location.ProxyConnectTimeout}};
		proxy_read_timeout {{$location.ProxyReadTimeout}};
		proxy_send_timeout {{$location.ProxySendTimeout}};
		client_max_body_size {{$location.ClientMaxBodySize}};
		{{- $proxySetHeaders := generateProxySetHeaders $location $.Ingress.Annotations -}}
		{{$proxySetHeaders}}
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Host $host;
		proxy_set_header X-Forwarded-Port $server_port;
		proxy_set_header X-Forwarded-Proto {{if $server.RedirectToHTTPS}}https{{else}}$scheme{{end}};
		proxy_buffering {{if $location.ProxyBuffering}}on{{else}}off{{end}};

		{{- if $location.ProxyBuffers}}
		proxy_buffers {{$location.ProxyBuffers}};
		{{- end}}
		{{- if $location.ProxyBufferSize}}
		proxy_buffer_size {{$location.ProxyBufferSize}};
		{{- end}}
		{{- if $location.ProxyMaxTempFileSize}}
		proxy_max_temp_file_size {{$location.ProxyMaxTempFileSize}};
		{{- end}}
		{{- if $.SpiffeClientCerts}}
		proxy_ssl_certificate {{ makeSecretPath "/etc/nginx/secrets/spiffe_cert.pem" $.StaticSSLPath "$secret_dir_path" $.DynamicSSLReloadEnabled }};
		proxy_ssl_certificate_key {{ makeSecretPath "/etc/nginx/secrets/spiffe_key.pem" $.StaticSSLPath "$secret_dir_path" $.DynamicSSLReloadEnabled }};
		proxy_ssl_trusted_certificate /etc/nginx/secrets/spiffe_rootca.pem;
		proxy_ssl_server_name on;
		proxy_ssl_verify on;
		proxy_ssl_verify_depth 25;
		proxy_ssl_name {{$location.ProxySSLName}};
		{{- end}}
		{{- if $location.SSL}}
		proxy_pass https://{{$location.Upstream.Name}}{{$location.Rewrite}};
		{{- else}}
		proxy_pass http://{{$location.Upstream.Name}}{{$location.Rewrite}};
		{{- end}}
		{{- end}}

		{{with $location.LimitReq}}
		limit_req zone={{ $location.LimitReq.Zone }} {{if $location.LimitReq.Burst}}burst={{$location.LimitReq.Burst}}{{end}} {{if $location.LimitReq.NoDelay}}nodelay{{else if $location.LimitReq.Delay}}delay={{$location.LimitReq.Delay}}{{end}};
		{{if $location.LimitReq.DryRun}}limit_req_dry_run on;{{end}}
		{{if $location.LimitReq.LogLevel}}limit_req_log_level {{$location.LimitReq.LogLevel}};{{end}}
		{{if $location.LimitReq.RejectCode}}limit_req_status {{$location.LimitReq.RejectCode}};{{end}}
		{{end}}
	}
	{{end -}}
	{{- if $server.GRPCOnly}}
	error_page 400 @grpcerror400;
	error_page 401 @grpcerror401;
	error_page 403 @grpcerror403;
	error_page 404 @grpcerror404;
	error_page 405 @grpcerror405;
	error_page 408 @grpcerror408;
	error_page 414 @grpcerror414;
	error_page 426 @grpcerror426;
	error_page 500 @grpcerror500;
	error_page 501 @grpcerror501;
	error_page 502 @grpcerror502;
	error_page 503 @grpcerror503;
	error_page 504 @grpcerror504;
	{{- end}}
	{{- if $server.HTTP2}}
	location @grpcerror400 { default_type application/grpc; return 400 "\n"; }
	location @grpcerror401 { default_type application/grpc; return 401 "\n"; }
	location @grpcerror403 { default_type application/grpc; return 403 "\n"; }
	location @grpcerror404 { default_type application/grpc; return 404 "\n"; }
	location @grpcerror405 { default_type application/grpc; return 405 "\n"; }
	location @grpcerror408 { default_type application/grpc; return 408 "\n"; }
	location @grpcerror414 { default_type application/grpc; return 414 "\n"; }
	location @grpcerror426 { default_type application/grpc; return 426 "\n"; }
	location @grpcerror500 { default_type application/grpc; return 500 "\n"; }
	location @grpcerror501 { default_type application/grpc; return 501 "\n"; }
	location @grpcerror502 { default_type application/grpc; return 502 "\n"; }
	location @grpcerror503 { default_type application/grpc; return 503 "\n"; }
	location @grpcerror504 { default_type application/grpc; return 504 "\n"; }
	{{- end}}
}{{end}}
`
)

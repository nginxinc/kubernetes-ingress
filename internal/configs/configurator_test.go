package configs

import (
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
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
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
		Config:                  NewDefaultConfigParams(false),
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
		Config:                  NewDefaultConfigParams(false),
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

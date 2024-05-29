package telemetry_test

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/telemetry"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	tel "github.com/nginxinc/telemetry-exporter/pkg/telemetry"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"

	testClient "k8s.io/client-go/kubernetes/fake"
)

func TestCreateNewCollectorWithCustomReportingPeriod(t *testing.T) {
	t.Parallel()

	cfg := telemetry.CollectorConfig{
		Period: 24 * time.Hour,
	}

	c, err := telemetry.NewCollector(cfg)
	if err != nil {
		t.Fatal(err)
	}

	want := 24.0
	got := c.Config.Period.Hours()

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCreateNewCollectorWithCustomExporter(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}

	cfg := telemetry.CollectorConfig{
		K8sClientReader: newTestClientset(),
		Configurator:    newConfigurator(t),
		Version:         telemetryNICData.ProjectVersion,
	}
	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		Data: tel.Data{
			ProjectName:         telemetryNICData.ProjectName,
			ProjectVersion:      telemetryNICData.ProjectVersion,
			ClusterVersion:      telemetryNICData.ClusterVersion,
			ProjectArchitecture: runtime.GOARCH,
		},
	}
	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCollectNodeCountInClusterWithOneNode(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	cfg := telemetry.CollectorConfig{
		Configurator:    newConfigurator(t),
		K8sClientReader: newTestClientset(node1),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		Data: tel.Data{
			ProjectName:         telemetryNICData.ProjectName,
			ProjectVersion:      telemetryNICData.ProjectVersion,
			ClusterVersion:      telemetryNICData.ClusterVersion,
			ProjectArchitecture: runtime.GOARCH,
			ClusterNodeCount:    1,
			ClusterPlatform:     "other",
		},
		NICResourceCounts: telemetry.NICResourceCounts{
			VirtualServers:      0,
			VirtualServerRoutes: 0,
			TransportServers:    0,
		},
	}

	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCollectNodeCountInClusterWithThreeNodes(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	cfg := telemetry.CollectorConfig{
		Configurator:    newConfigurator(t),
		K8sClientReader: newTestClientset(node1, node2, node3),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	telData := tel.Data{
		ProjectName:         telemetryNICData.ProjectName,
		ProjectVersion:      telemetryNICData.ProjectVersion,
		ClusterVersion:      telemetryNICData.ClusterVersion,
		ClusterPlatform:     "other",
		ProjectArchitecture: runtime.GOARCH,
		ClusterNodeCount:    3,
	}

	nicResourceCounts := telemetry.NICResourceCounts{
		VirtualServers:      0,
		VirtualServerRoutes: 0,
		TransportServers:    0,
	}

	td := telemetry.Data{
		Data:              telData,
		NICResourceCounts: nicResourceCounts,
	}

	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCollectClusterIDInClusterWithOneNode(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	cfg := telemetry.CollectorConfig{
		Configurator:    newConfigurator(t),
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		Data: tel.Data{
			ProjectName:         telemetryNICData.ProjectName,
			ProjectVersion:      telemetryNICData.ProjectVersion,
			ClusterVersion:      telemetryNICData.ClusterVersion,
			ClusterPlatform:     "other",
			ProjectArchitecture: runtime.GOARCH,
			ClusterNodeCount:    1,
			ClusterID:           telemetryNICData.ClusterID,
		},
		NICResourceCounts: telemetry.NICResourceCounts{
			VirtualServers:      0,
			VirtualServerRoutes: 0,
			TransportServers:    0,
		},
	}
	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCollectClusterVersion(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	cfg := telemetry.CollectorConfig{
		Configurator:    newConfigurator(t),
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	telData := tel.Data{
		ProjectName:         telemetryNICData.ProjectName,
		ProjectVersion:      telemetryNICData.ProjectVersion,
		ProjectArchitecture: telemetryNICData.ProjectArchitecture,
		ClusterNodeCount:    1,
		ClusterID:           telemetryNICData.ClusterID,
		ClusterVersion:      telemetryNICData.ClusterVersion,
		ClusterPlatform:     "other",
	}

	nicResourceCounts := telemetry.NICResourceCounts{
		VirtualServers:      0,
		VirtualServerRoutes: 0,
		TransportServers:    0,
	}

	td := telemetry.Data{
		Data:              telData,
		NICResourceCounts: nicResourceCounts,
	}

	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCollectPolicyCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		policies func() []*conf_v1.Policy
		want     int
	}{
		{
			name: "SinglePolicy",
			policies: func() []*conf_v1.Policy {
				return []*conf_v1.Policy{egressMTLSPolicy}
			},
			want: 1,
		},
		{
			name: "MultiplePolicies",
			policies: func() []*conf_v1.Policy {
				return []*conf_v1.Policy{rateLimitPolicy, wafPolicy, oidcPolicy}
			},
			want: 3,
		},
		{
			name: "MultipleSamePolicies",
			policies: func() []*conf_v1.Policy {
				return []*conf_v1.Policy{rateLimitPolicy, rateLimitPolicy}
			},
			want: 2,
		},
		{
			name: "SingleInvalidPolicy",
			policies: func() []*conf_v1.Policy {
				return []*conf_v1.Policy{rateLimitPolicyInvalid}
			},
			want: 0,
		},
		{
			name: "MultiplePoliciesOneValidOneInvalid",
			policies: func() []*conf_v1.Policy {
				return []*conf_v1.Policy{rateLimitPolicy, rateLimitPolicyInvalid}
			},
			want: 1,
		},
		{
			name:     "NoPolicies",
			policies: func() []*conf_v1.Policy { return []*conf_v1.Policy{} },
			want:     0,
		},
		{
			name:     "NilPolicies",
			policies: nil,
			want:     0,
		},
		{
			name:     "NilPolicyValue",
			policies: func() []*conf_v1.Policy { return nil },
			want:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var got int
			cfg := telemetry.CollectorConfig{
				Policies: tc.policies,
			}
			collector, err := telemetry.NewCollector(cfg)
			if err != nil {
				t.Fatal(err)
			}

			for _, polCount := range collector.PolicyCount() {
				got += polCount
			}

			if tc.want != got {
				t.Errorf("want %d policies, got %d", tc.want, got)
			}
		})
	}
}

func TestCollectPoliciesReport(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	cfg := telemetry.CollectorConfig{
		Configurator:    newConfigurator(t),
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
		Policies: func() []*conf_v1.Policy {
			return []*conf_v1.Policy{
				egressMTLSPolicy,
				egressMTLSPolicy,
				rateLimitPolicyInvalid,
				wafPolicy,
				wafPolicy,
				oidcPolicy,
			}
		},
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	telData := tel.Data{
		ProjectName:         telemetryNICData.ProjectName,
		ProjectVersion:      telemetryNICData.ProjectVersion,
		ProjectArchitecture: telemetryNICData.ProjectArchitecture,
		ClusterNodeCount:    1,
		ClusterID:           telemetryNICData.ClusterID,
		ClusterVersion:      telemetryNICData.ClusterVersion,
		ClusterPlatform:     "other",
	}

	nicResourceCounts := telemetry.NICResourceCounts{
		RateLimitPolicies:  0,
		WAFPolicies:        2,
		OIDCPolicies:       1,
		EgressMTLSPolicies: 2,
	}

	td := telemetry.Data{
		Data:              telData,
		NICResourceCounts: nicResourceCounts,
	}

	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCollectIsPlus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		isPlus bool
		want   bool
	}{
		{
			name:   "Plus enabled",
			isPlus: true,
			want:   true,
		},
		{
			name:   "Plus disabled",
			isPlus: false,
			want:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			exp := &telemetry.StdoutExporter{Endpoint: buf}

			configurator := newConfiguratorWithIngress(t)

			cfg := telemetry.CollectorConfig{
				Configurator:    configurator,
				K8sClientReader: newTestClientset(node1, kubeNS),
				Version:         telemetryNICData.ProjectVersion,
				IsPlus:          tc.isPlus,
			}

			c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
			if err != nil {
				t.Fatal(err)
			}
			c.Collect(context.Background())

			ver := c.IsPlusEnabled()

			if tc.want != ver {
				t.Errorf("want: %t, got: %t", tc.want, ver)
			}
		})
	}
}

func TestCollectInvalidIsPlus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		isPlus bool
		want   bool
	}{
		{
			name:   "Plus disabled but want enabled",
			isPlus: false,
			want:   true,
		},
		{
			name:   "Plus disabled but want enabled",
			isPlus: false,
			want:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			exp := &telemetry.StdoutExporter{Endpoint: buf}

			configurator := newConfiguratorWithIngress(t)

			cfg := telemetry.CollectorConfig{
				Configurator:    configurator,
				K8sClientReader: newTestClientset(node1, kubeNS),
				Version:         telemetryNICData.ProjectVersion,
				IsPlus:          tc.isPlus,
			}

			c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
			if err != nil {
				t.Fatal(err)
			}
			c.Collect(context.Background())

			ver := c.IsPlusEnabled()

			if tc.want == ver {
				t.Errorf("want: %t, got: %t", tc.want, ver)
			}
		})
	}
}

func TestIngressCountReportsNoDeployedIngresses(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	cfg := telemetry.CollectorConfig{
		Configurator:    newConfigurator(t),
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	telData := tel.Data{
		ProjectName:         telemetryNICData.ProjectName,
		ProjectVersion:      telemetryNICData.ProjectVersion,
		ProjectArchitecture: telemetryNICData.ProjectArchitecture,
		ClusterNodeCount:    1,
		ClusterID:           telemetryNICData.ClusterID,
		ClusterVersion:      telemetryNICData.ClusterVersion,
		ClusterPlatform:     "other",
	}

	nicResourceCounts := telemetry.NICResourceCounts{
		VirtualServers:      0,
		VirtualServerRoutes: 0,
		TransportServers:    0,
		RegularIngressCount: 0,
	}

	td := telemetry.Data{
		Data:              telData,
		NICResourceCounts: nicResourceCounts,
	}

	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestMergeableIngressAnnotations(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}

	masterAnnotations := map[string]string{
		"nginx.org/mergeable-ingress-type": "master",
	}
	coffeeAnnotations := map[string]string{
		"nginx.org/mergeable-ingress-type":     "minion",
		"nginx.org/proxy-set-header":           "X-Forwarded-ABC",
		"appprotect.f5.com/app-protect-enable": "False",
	}
	teaAnnotations := map[string]string{
		"nginx.org/mergeable-ingress-type": "minion",
		"nginx.org/proxy-set-header":       "X-Forwarded-Tea: Chai",
		"nginx.com/health-checks":          "False",
	}

	configurator := newConfiguratorWithMergeableIngressCustomAnnotations(t, masterAnnotations, coffeeAnnotations, teaAnnotations)

	cfg := telemetry.CollectorConfig{
		Configurator:    configurator,
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	expectedAnnotations := []string{
		"nginx.org/mergeable-ingress-type",
		"nginx.org/proxy-set-header",
		"nginx.com/health-checks",
		"appprotect.f5.com/app-protect-enable",
	}

	got := buf.String()
	for _, expectedAnnotation := range expectedAnnotations {
		if !strings.Contains(got, expectedAnnotation) {
			t.Errorf("expected %v in %v", expectedAnnotation, got)
		}
	}
}

func TestInvalidMergeableIngressAnnotations(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}

	masterAnnotations := map[string]string{
		"nginx.org/ingress-type":                           "master",
		"kubectl.kubernetes.io/last-applied-configuration": "s",
	}
	coffeeAnnotations := map[string]string{
		"nginx.org/mergeable-ingress-type": "minion",
	}
	teaAnnotations := map[string]string{
		"nginx.org/mergeable-type":                   "minion",
		"nginx.org/proxy-set":                        "X-$-ABC",
		"nginx.ingress.kubernetes.io/rewrite-target": "/",
	}

	configurator := newConfiguratorWithMergeableIngressCustomAnnotations(t, masterAnnotations, coffeeAnnotations, teaAnnotations)

	cfg := telemetry.CollectorConfig{
		Configurator:    configurator,
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	expectedAnnotations := []string{
		"kubectl.kubernetes.io/last-applied-configuration",
		"nginx.org/proxy-set-header",
		"nginx.ingress.kubernetes.io/rewrite-target",
	}

	got := buf.String()
	for _, expectedAnnotation := range expectedAnnotations {
		if strings.Contains(got, expectedAnnotation) {
			t.Errorf("expected %v in %v", expectedAnnotation, got)
		}
	}
}

func TestStandardIngressAnnotations(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}

	annotations := map[string]string{
		"appprotect.f5.com/app-protect-enable": "False",
		"nginx.org/proxy-set-header":           "X-Forwarded-ABC",
		"ingress.kubernetes.io/ssl-redirect":   "True",
		"nginx.com/slow-start":                 "0s",
	}

	configurator := newConfiguratorWithIngressWithCustomAnnotations(t, annotations)

	cfg := telemetry.CollectorConfig{
		Configurator:    configurator,
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	expectedAnnotations := []string{
		"appprotect.f5.com/app-protect-enable",
		"nginx.org/proxy-set-header",
		"nginx.com/slow-start",
		"ingress.kubernetes.io/ssl-redirect",
	}

	got := buf.String()
	for _, expectedAnnotation := range expectedAnnotations {
		if !strings.Contains(got, expectedAnnotation) {
			t.Errorf("expected %v in %v", expectedAnnotation, got)
		}
	}
}

func TestInvalidStandardIngressAnnotations(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}

	annotations := map[string]string{
		"alb.ingress.kubernetes.io/group.order":      "0",
		"alb.ingress.kubernetes.io/ip-address-type":  "ipv4",
		"alb.ingress.kubernetes.io/scheme":           "internal",
		"nginx.ingress.kubernetes.io/rewrite-target": "/",
	}

	configurator := newConfiguratorWithIngressWithCustomAnnotations(t, annotations)

	cfg := telemetry.CollectorConfig{
		Configurator:    configurator,
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	expectedAnnotations := []string{
		"alb.ingress.kubernetes.io/scheme",
		"alb.ingress.kubernetes.io/group.order",
		"alb.ingress.kubernetes.io/ip-address-type",
		"nginx.ingress.kubernetes.io/rewrite-target",
	}

	got := buf.String()
	for _, expectedAnnotation := range expectedAnnotations {
		if strings.Contains(got, expectedAnnotation) {
			t.Errorf("expected %v in %v", expectedAnnotation, got)
		}
	}
}

func TestIngressCountReportsNumberOfDeployedIngresses(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}

	configurator := newConfiguratorWithIngress(t)

	cfg := telemetry.CollectorConfig{
		Configurator:    configurator,
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	telData := tel.Data{
		ProjectName:         telemetryNICData.ProjectName,
		ProjectVersion:      telemetryNICData.ProjectVersion,
		ProjectArchitecture: telemetryNICData.ProjectArchitecture,
		ClusterNodeCount:    1,
		ClusterID:           telemetryNICData.ClusterID,
		ClusterVersion:      telemetryNICData.ClusterVersion,
		ClusterPlatform:     "other",
	}

	nicResourceCounts := telemetry.NICResourceCounts{
		VirtualServers:      0,
		VirtualServerRoutes: 0,
		TransportServers:    0,
		RegularIngressCount: 1,
		Services:            2,
	}

	td := telemetry.Data{
		Data:              telData,
		NICResourceCounts: nicResourceCounts,
	}

	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestMasterMinionIngressCountReportsNumberOfDeployedIngresses(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}

	configurator := newConfiguratorWithMergeableIngress(t)

	cfg := telemetry.CollectorConfig{
		Configurator:    configurator,
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         telemetryNICData.ProjectVersion,
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	telData := tel.Data{
		ProjectName:         telemetryNICData.ProjectName,
		ProjectVersion:      telemetryNICData.ProjectVersion,
		ProjectArchitecture: telemetryNICData.ProjectArchitecture,
		ClusterNodeCount:    1,
		ClusterID:           telemetryNICData.ClusterID,
		ClusterVersion:      telemetryNICData.ClusterVersion,
		ClusterPlatform:     "other",
	}

	nicResourceCounts := telemetry.NICResourceCounts{
		VirtualServers:      0,
		VirtualServerRoutes: 0,
		TransportServers:    0,
		MasterIngressCount:  1,
		MinionIngressCount:  2,
		Services:            2,
		IngressAnnotations:  []string{"nginx.org/mergeable-ingress-type"},
	}

	td := telemetry.Data{
		Data:              telData,
		NICResourceCounts: nicResourceCounts,
	}

	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCollectAppProtectVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		appProtectVersion string
		wantVersion       string
	}{
		{
			name:              "AppProtect 4.8",
			appProtectVersion: "4.8.1",
			wantVersion:       "4.8.1",
		},
		{
			name:              "AppProtect 4.9",
			appProtectVersion: "4.9",
			wantVersion:       "4.9",
		},
		{
			name:              "AppProtect 5.1",
			appProtectVersion: "5.1",
			wantVersion:       "5.1",
		},
		{
			name:              "No AppProtect Installed",
			appProtectVersion: "",
			wantVersion:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			exp := &telemetry.StdoutExporter{Endpoint: buf}

			configurator := newConfiguratorWithIngress(t)

			cfg := telemetry.CollectorConfig{
				Configurator:      configurator,
				K8sClientReader:   newTestClientset(node1, kubeNS),
				Version:           telemetryNICData.ProjectVersion,
				AppProtectVersion: tc.appProtectVersion,
			}

			c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
			if err != nil {
				t.Fatal(err)
			}
			c.Collect(context.Background())

			ver := c.AppProtectVersion()

			if tc.wantVersion != ver {
				t.Errorf("want: %s, got: %s", tc.wantVersion, ver)
			}
		})
	}
}

func TestCollectInvalidAppProtectVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		appProtectVersion string
		wantVersion       string
	}{
		{
			name:              "AppProtect Not Installed",
			appProtectVersion: "",
			wantVersion:       "4.8.1",
		},
		{
			name:              "Cant Find AppProtect 4.9",
			appProtectVersion: "4.9",
			wantVersion:       "",
		},
		{
			name:              "Found Different AppProtect Version",
			appProtectVersion: "5.1",
			wantVersion:       "4.9",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			exp := &telemetry.StdoutExporter{Endpoint: buf}

			configurator := newConfiguratorWithIngress(t)

			cfg := telemetry.CollectorConfig{
				Configurator:      configurator,
				K8sClientReader:   newTestClientset(node1, kubeNS),
				Version:           telemetryNICData.ProjectVersion,
				AppProtectVersion: tc.appProtectVersion,
			}

			c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
			if err != nil {
				t.Fatal(err)
			}
			c.Collect(context.Background())

			ver := c.AppProtectVersion()

			if tc.wantVersion == ver {
				t.Errorf("want: %s, got: %s", tc.wantVersion, ver)
			}
		})
	}
}

func TestCollectInstallationFlags(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		setFlags  []string
		wantFlags []string
	}{
		{
			name: "first flag",
			setFlags: []string{
				"nginx-plus=true",
			},
			wantFlags: []string{
				"nginx-plus=true",
			},
		},
		{
			name: "second flag",
			setFlags: []string{
				"-v=3",
			},
			wantFlags: []string{
				"-v=3",
			},
		},
		{
			name: "multiple flags",
			setFlags: []string{
				"nginx-plus=true",
				"-v=3",
			},
			wantFlags: []string{
				"nginx-plus=true",
				"-v=3",
			},
		},
		{
			name:      "no flags",
			setFlags:  []string{},
			wantFlags: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			exp := &telemetry.StdoutExporter{Endpoint: buf}

			configurator := newConfigurator(t)

			cfg := telemetry.CollectorConfig{
				Configurator:      configurator,
				K8sClientReader:   newTestClientset(node1, kubeNS),
				Version:           telemetryNICData.ProjectVersion,
				InstallationFlags: tc.setFlags,
			}

			c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
			if err != nil {
				t.Fatal(err)
			}
			c.Collect(context.Background())

			telData := tel.Data{
				ProjectName:         telemetryNICData.ProjectName,
				ProjectVersion:      telemetryNICData.ProjectVersion,
				ProjectArchitecture: telemetryNICData.ProjectArchitecture,
				ClusterNodeCount:    1,
				ClusterID:           telemetryNICData.ClusterID,
				ClusterVersion:      telemetryNICData.ClusterVersion,
				ClusterPlatform:     "other",
			}

			nicResourceCounts := telemetry.NICResourceCounts{
				InstallationFlags: tc.wantFlags,
			}

			td := telemetry.Data{
				Data:              telData,
				NICResourceCounts: nicResourceCounts,
			}

			want := fmt.Sprintf("%+v", &td)

			got := buf.String()
			if !cmp.Equal(want, got) {
				t.Error(cmp.Diff(got, want))
			}
		})
	}
}

func TestCountVirtualServers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName                  string
		expectedTraceDataOnAdd    telemetry.Report
		expectedTraceDataOnDelete telemetry.Report
		virtualServers            []*configs.VirtualServerEx
		deleteCount               int
	}{
		{
			testName: "Create and delete 1 VirtualServer",
			expectedTraceDataOnAdd: telemetry.Report{
				VirtualServers: 1,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				VirtualServers: 0,
			},
			virtualServers: []*configs.VirtualServerEx{
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.VirtualServerSpec{},
					},
				},
			},
			deleteCount: 1,
		},
		{
			testName: "Create 2 VirtualServers and delete 2",
			expectedTraceDataOnAdd: telemetry.Report{
				VirtualServers: 2,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				VirtualServers: 0,
			},
			virtualServers: []*configs.VirtualServerEx{
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.VirtualServerSpec{},
					},
				},
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "tea",
						},
						Spec: conf_v1.VirtualServerSpec{},
					},
				},
			},
			deleteCount: 2,
		},
		{
			testName: "Create 2 VirtualServers and delete 1",
			expectedTraceDataOnAdd: telemetry.Report{
				VirtualServers: 2,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				VirtualServers: 1,
			},
			virtualServers: []*configs.VirtualServerEx{
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.VirtualServerSpec{},
					},
				},
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "tea",
						},
						Spec: conf_v1.VirtualServerSpec{},
					},
				},
			},
			deleteCount: 1,
		},
	}

	for _, test := range testCases {
		configurator := newConfigurator(t)

		c, err := telemetry.NewCollector(telemetry.CollectorConfig{
			K8sClientReader: newTestClientset(kubeNS, node1, pod1, replica),
			SecretStore:     newSecretStore(t),
			Configurator:    configurator,
			Version:         telemetryNICData.ProjectVersion,
		})
		if err != nil {
			t.Fatal(err)
		}
		c.Config.PodNSName = types.NamespacedName{
			Namespace: "nginx-ingress",
			Name:      "nginx-ingress",
		}

		for _, vs := range test.virtualServers {
			_, err := configurator.AddOrUpdateVirtualServer(vs)
			if err != nil {
				t.Fatal(err)
			}
		}

		gotTraceDataOnAdd, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnAdd.VirtualServers, gotTraceDataOnAdd.VirtualServers) {
			t.Error(cmp.Diff(test.expectedTraceDataOnAdd.VirtualServers, gotTraceDataOnAdd.VirtualServers))
		}

		for i := 0; i < test.deleteCount; i++ {
			vs := test.virtualServers[i]
			key := getResourceKey(vs.VirtualServer.Namespace, vs.VirtualServer.Name)
			err := configurator.DeleteVirtualServer(key, false)
			if err != nil {
				t.Fatal(err)
			}
		}

		gotTraceDataOnDelete, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnDelete.VirtualServers, gotTraceDataOnDelete.VirtualServers) {
			t.Error(cmp.Diff(test.expectedTraceDataOnDelete.VirtualServers, gotTraceDataOnDelete.VirtualServers))
		}
	}
}

func TestCountTransportServers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName                  string
		expectedTraceDataOnAdd    telemetry.Report
		expectedTraceDataOnDelete telemetry.Report
		transportServers          []*configs.TransportServerEx
		deleteCount               int
	}{
		{
			testName: "Create and delete 1 TransportServer",
			expectedTraceDataOnAdd: telemetry.Report{
				TransportServers: 1,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				TransportServers: 0,
			},
			transportServers: []*configs.TransportServerEx{
				{
					TransportServer: &conf_v1.TransportServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.TransportServerSpec{
							Action: &conf_v1.TransportServerAction{
								Pass: "coffee",
							},
						},
					},
				},
			},
			deleteCount: 1,
		},
		{
			testName: "Create 2 and delete 2 TransportServer",
			expectedTraceDataOnAdd: telemetry.Report{
				TransportServers: 2,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				TransportServers: 0,
			},
			transportServers: []*configs.TransportServerEx{
				{
					TransportServer: &conf_v1.TransportServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.TransportServerSpec{
							Action: &conf_v1.TransportServerAction{
								Pass: "coffee",
							},
						},
					},
				},
				{
					TransportServer: &conf_v1.TransportServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "tea",
						},
						Spec: conf_v1.TransportServerSpec{
							Action: &conf_v1.TransportServerAction{
								Pass: "tea",
							},
						},
					},
				},
			},
			deleteCount: 2,
		},
		{
			testName: "Create 2 and delete 1 TransportServer",
			expectedTraceDataOnAdd: telemetry.Report{
				TransportServers: 2,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				TransportServers: 1,
			},
			transportServers: []*configs.TransportServerEx{
				{
					TransportServer: &conf_v1.TransportServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.TransportServerSpec{
							Action: &conf_v1.TransportServerAction{
								Pass: "coffee",
							},
						},
					},
				},
				{
					TransportServer: &conf_v1.TransportServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "tea",
						},
						Spec: conf_v1.TransportServerSpec{
							Action: &conf_v1.TransportServerAction{
								Pass: "tea",
							},
						},
					},
				},
			},
			deleteCount: 1,
		},
	}

	for _, test := range testCases {
		configurator := newConfigurator(t)

		c, err := telemetry.NewCollector(telemetry.CollectorConfig{
			K8sClientReader: newTestClientset(kubeNS, node1, pod1, replica),
			SecretStore:     newSecretStore(t),
			Configurator:    configurator,
			Version:         telemetryNICData.ProjectVersion,
		})
		if err != nil {
			t.Fatal(err)
		}
		c.Config.PodNSName = types.NamespacedName{
			Namespace: "nginx-ingress",
			Name:      "nginx-ingress",
		}

		for _, ts := range test.transportServers {
			_, err = configurator.AddOrUpdateTransportServer(ts)
			if err != nil {
				t.Fatal(err)
			}
		}

		gotTraceDataOnAdd, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnAdd.TransportServers, gotTraceDataOnAdd.TransportServers) {
			t.Error(cmp.Diff(test.expectedTraceDataOnAdd.TransportServers, gotTraceDataOnAdd.TransportServers))
		}

		for i := 0; i < test.deleteCount; i++ {
			ts := test.transportServers[i]
			key := getResourceKey(ts.TransportServer.Namespace, ts.TransportServer.Name)
			err = configurator.DeleteTransportServer(key)
			if err != nil {
				t.Fatal(err)
			}
		}

		gotTraceDataOnDelete, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnDelete.TransportServers, gotTraceDataOnDelete.TransportServers) {
			t.Error(cmp.Diff(test.expectedTraceDataOnDelete.TransportServers, gotTraceDataOnDelete.TransportServers))
		}
	}
}

func TestCountSecretsWithTwoSecrets(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	cfg := telemetry.CollectorConfig{
		Configurator:    newConfigurator(t),
		K8sClientReader: newTestClientset(node1, kubeNS),
		SecretStore:     newSecretStore(t),
		Version:         telemetryNICData.ProjectVersion,
	}

	// Add multiple secrets.
	cfg.SecretStore.AddOrUpdateSecret(secret1)
	cfg.SecretStore.AddOrUpdateSecret(secret2)

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	telData := tel.Data{
		ProjectName:         telemetryNICData.ProjectName,
		ProjectVersion:      telemetryNICData.ProjectVersion,
		ProjectArchitecture: telemetryNICData.ProjectArchitecture,
		ClusterNodeCount:    1,
		ClusterID:           telemetryNICData.ClusterID,
		ClusterVersion:      telemetryNICData.ClusterVersion,
		ClusterPlatform:     "other",
	}

	nicResourceCounts := telemetry.NICResourceCounts{
		VirtualServers:      0,
		VirtualServerRoutes: 0,
		TransportServers:    0,
		Secrets:             2,
	}

	td := telemetry.Data{
		Data:              telData,
		NICResourceCounts: nicResourceCounts,
	}

	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCountSecretsAddTwoSecretsAndDeleteOne(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	cfg := telemetry.CollectorConfig{
		Configurator:    newConfigurator(t),
		K8sClientReader: newTestClientset(node1, kubeNS),
		SecretStore:     newSecretStore(t),
		Version:         telemetryNICData.ProjectVersion,
	}

	// Add multiple secrets.
	cfg.SecretStore.AddOrUpdateSecret(secret1)
	cfg.SecretStore.AddOrUpdateSecret(secret2)

	// Delete one secret.
	cfg.SecretStore.DeleteSecret(fmt.Sprintf("%s/%s", secret2.Namespace, secret2.Name))

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	telData := tel.Data{
		ProjectName:         telemetryNICData.ProjectName,
		ProjectVersion:      telemetryNICData.ProjectVersion,
		ProjectArchitecture: telemetryNICData.ProjectArchitecture,
		ClusterNodeCount:    1,
		ClusterID:           telemetryNICData.ClusterID,
		ClusterVersion:      telemetryNICData.ClusterVersion,
		ClusterPlatform:     "other",
	}

	nicResourceCounts := telemetry.NICResourceCounts{
		VirtualServers:      0,
		VirtualServerRoutes: 0,
		TransportServers:    0,
		Secrets:             1,
	}

	td := telemetry.Data{
		Data:              telData,
		NICResourceCounts: nicResourceCounts,
	}

	want := fmt.Sprintf("%+v", &td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCountVirtualServersServices(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName                  string
		expectedTraceDataOnAdd    telemetry.Report
		expectedTraceDataOnDelete telemetry.Report
		virtualServers            []*configs.VirtualServerEx
		deleteCount               int
	}{
		{
			testName: "Create and delete 1 VirtualServer with 2 upstreams",
			expectedTraceDataOnAdd: telemetry.Report{
				ServiceCount: 2,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				ServiceCount: 0,
			},
			virtualServers: []*configs.VirtualServerEx{
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.VirtualServerSpec{
							Upstreams: []conf_v1.Upstream{
								{
									Name:    "coffee",
									Service: "coffee-svc",
								},
								{
									Name:    "coffee2",
									Service: "coffee-svc2",
								},
							},
						},
					},
				},
			},
			deleteCount: 1,
		},
		{
			testName: "Same service in 2 upstreams is only counted once",
			expectedTraceDataOnAdd: telemetry.Report{
				ServiceCount: 1,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				ServiceCount: 0,
			},
			virtualServers: []*configs.VirtualServerEx{
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.VirtualServerSpec{
							Upstreams: []conf_v1.Upstream{
								{
									Name:    "coffee",
									Service: "same-svc",
								},
								{
									Name:    "coffee2",
									Service: "same-svc",
								},
							},
						},
					},
				},
			},
			deleteCount: 1,
		},
		{
			testName: "A backup service is counted in addition to the primary service",
			expectedTraceDataOnAdd: telemetry.Report{
				ServiceCount: 2,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				ServiceCount: 0,
			},
			virtualServers: []*configs.VirtualServerEx{
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.VirtualServerSpec{
							Upstreams: []conf_v1.Upstream{
								{
									Name:    "coffee",
									Service: "same-svc",
									Backup:  "backup-service",
								},
							},
						},
					},
				},
			},
			deleteCount: 1,
		},
		{
			testName: "A grpc service is counted in addition to the primary service and backup service",
			expectedTraceDataOnAdd: telemetry.Report{
				ServiceCount: 3,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				ServiceCount: 0,
			},
			virtualServers: []*configs.VirtualServerEx{
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.VirtualServerSpec{
							Upstreams: []conf_v1.Upstream{
								{
									Name:    "coffee",
									Service: "same-svc",
									Backup:  "backup-service",
									HealthCheck: &conf_v1.HealthCheck{
										GRPCService: "grpc-service",
									},
								},
							},
						},
					},
				},
			},
			deleteCount: 1,
		},
	}

	for _, test := range testCases {
		configurator := newConfigurator(t)

		c, err := telemetry.NewCollector(telemetry.CollectorConfig{
			K8sClientReader: newTestClientset(kubeNS, node1, pod1, replica),
			Configurator:    configurator,
			Version:         telemetryNICData.ProjectVersion,
			SecretStore:     newSecretStore(t),
		})
		if err != nil {
			t.Fatal(err)
		}
		c.Config.PodNSName = types.NamespacedName{
			Namespace: "nginx-ingress",
			Name:      "nginx-ingress",
		}

		for _, vs := range test.virtualServers {
			_, err := configurator.AddOrUpdateVirtualServer(vs)
			if err != nil {
				t.Fatal(err)
			}
		}

		gotTraceDataOnAdd, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnAdd.ServiceCount, gotTraceDataOnAdd.ServiceCount) {
			t.Error(cmp.Diff(test.expectedTraceDataOnAdd.ServiceCount, gotTraceDataOnAdd.ServiceCount))
		}

		for i := 0; i < test.deleteCount; i++ {
			vs := test.virtualServers[i]
			key := getResourceKey(vs.VirtualServer.Namespace, vs.VirtualServer.Name)
			err := configurator.DeleteVirtualServer(key, false)
			if err != nil {
				t.Fatal(err)
			}
		}

		gotTraceDataOnDelete, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnDelete.ServiceCount, gotTraceDataOnDelete.ServiceCount) {
			t.Error(cmp.Diff(test.expectedTraceDataOnDelete.ServiceCount, gotTraceDataOnDelete.ServiceCount))
		}
	}
}

func TestCountTransportServersServices(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName                  string
		expectedTraceDataOnAdd    telemetry.Report
		expectedTraceDataOnDelete telemetry.Report
		transportServers          []*configs.TransportServerEx
		deleteCount               int
	}{
		{
			testName: "Create and delete 1 TransportServer with 2 upstreams",
			expectedTraceDataOnAdd: telemetry.Report{
				ServiceCount: 2,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				ServiceCount: 0,
			},
			transportServers: []*configs.TransportServerEx{
				{
					TransportServer: &conf_v1.TransportServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.TransportServerSpec{
							Action: &conf_v1.TransportServerAction{
								Pass: "coffee",
							},
							Upstreams: []conf_v1.TransportServerUpstream{
								{
									Name:    "coffee",
									Service: "coffee-svc",
								},
								{
									Name:    "coffee2",
									Service: "coffee-svc2",
								},
							},
						},
					},
				},
			},
			deleteCount: 1,
		},
		{
			testName: "Same service in 2 upstreams is only counted once",
			expectedTraceDataOnAdd: telemetry.Report{
				ServiceCount: 1,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				ServiceCount: 0,
			},
			transportServers: []*configs.TransportServerEx{
				{
					TransportServer: &conf_v1.TransportServer{
						ObjectMeta: metaV1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.TransportServerSpec{
							Action: &conf_v1.TransportServerAction{
								Pass: "coffee",
							},
							Upstreams: []conf_v1.TransportServerUpstream{
								{
									Name:    "coffee",
									Service: "same-svc",
								},
								{
									Name:    "coffee2",
									Service: "same-svc",
								},
							},
						},
					},
				},
			},
			deleteCount: 1,
		},
	}

	for _, test := range testCases {
		configurator := newConfigurator(t)

		c, err := telemetry.NewCollector(telemetry.CollectorConfig{
			K8sClientReader: newTestClientset(kubeNS, node1, pod1, replica),
			Configurator:    configurator,
			Version:         telemetryNICData.ProjectVersion,
			SecretStore:     newSecretStore(t),
		})
		if err != nil {
			t.Fatal(err)
		}
		c.Config.PodNSName = types.NamespacedName{
			Namespace: "nginx-ingress",
			Name:      "nginx-ingress",
		}

		for _, ts := range test.transportServers {
			_, err := configurator.AddOrUpdateTransportServer(ts)
			if err != nil {
				t.Fatal(err)
			}
		}

		gotTraceDataOnAdd, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnAdd.ServiceCount, gotTraceDataOnAdd.ServiceCount) {
			t.Error(cmp.Diff(test.expectedTraceDataOnAdd.ServiceCount, gotTraceDataOnAdd.ServiceCount))
		}

		for i := 0; i < test.deleteCount; i++ {
			ts := test.transportServers[i]
			key := getResourceKey(ts.TransportServer.Namespace, ts.TransportServer.Name)
			err := configurator.DeleteTransportServer(key)
			if err != nil {
				t.Fatal(err)
			}
		}

		gotTraceDataOnDelete, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnDelete.ServiceCount, gotTraceDataOnDelete.ServiceCount) {
			t.Error(cmp.Diff(test.expectedTraceDataOnDelete.ServiceCount, gotTraceDataOnDelete.ServiceCount))
		}
	}
}

func TestCountIngressesServices(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName                  string
		expectedTraceDataOnAdd    telemetry.Report
		expectedTraceDataOnDelete telemetry.Report
		ingress                   configs.IngressEx
		deleteCount               int
	}{
		{
			testName: "Create and delete 1 Ingress with 2 services",
			expectedTraceDataOnAdd: telemetry.Report{
				ServiceCount: 2,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				ServiceCount: 0,
			},
			ingress:     createCafeIngressEx(),
			deleteCount: 1,
		},
	}

	for _, tc := range testCases {
		test := tc
		configurator := newConfigurator(t)

		c, err := telemetry.NewCollector(telemetry.CollectorConfig{
			K8sClientReader: newTestClientset(kubeNS, node1, pod1, replica),
			Configurator:    configurator,
			Version:         telemetryNICData.ProjectVersion,
			SecretStore:     newSecretStore(t),
		})
		if err != nil {
			t.Fatal(err)
		}
		c.Config.PodNSName = types.NamespacedName{
			Namespace: "nginx-ingress",
			Name:      "nginx-ingress",
		}

		_, err = configurator.AddOrUpdateIngress(&test.ingress)
		if err != nil {
			t.Fatal(err)
		}

		gotTraceDataOnAdd, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnAdd.ServiceCount, gotTraceDataOnAdd.ServiceCount) {
			t.Error(cmp.Diff(test.expectedTraceDataOnAdd.ServiceCount, gotTraceDataOnAdd.ServiceCount))
		}

		for i := 0; i < test.deleteCount; i++ {
			ing := test.ingress

			key := fmt.Sprintf("%s/%s", ing.Ingress.Namespace, ing.Ingress.Name)
			err := configurator.DeleteIngress(key, false)
			if err != nil {
				t.Fatal(err)
			}
		}

		if err != nil {
			t.Fatal(err)
		}

		gotTraceDataOnDelete, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnDelete.ServiceCount, gotTraceDataOnDelete.ServiceCount) {
			t.Error(cmp.Diff(test.expectedTraceDataOnDelete.ServiceCount, gotTraceDataOnDelete.ServiceCount))
		}
	}
}

func TestCountMergeableIngressesServices(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName                  string
		expectedTraceDataOnAdd    telemetry.Report
		expectedTraceDataOnDelete telemetry.Report
		ingress                   *configs.MergeableIngresses
		deleteCount               int
	}{
		{
			testName: "Create and delete 1 MergeableIngress with 2 services",
			expectedTraceDataOnAdd: telemetry.Report{
				ServiceCount: 2,
			},
			expectedTraceDataOnDelete: telemetry.Report{
				ServiceCount: 0,
			},
			ingress:     createMergeableCafeIngress(),
			deleteCount: 1,
		},
	}

	for _, tc := range testCases {
		test := tc

		configurator := newConfigurator(t)

		c, err := telemetry.NewCollector(telemetry.CollectorConfig{
			K8sClientReader: newTestClientset(kubeNS, node1, pod1, replica),
			Configurator:    configurator,
			Version:         telemetryNICData.ProjectVersion,
			SecretStore:     newSecretStore(t),
		})
		if err != nil {
			t.Fatal(err)
		}
		c.Config.PodNSName = types.NamespacedName{
			Namespace: "nginx-ingress",
			Name:      "nginx-ingress",
		}

		_, err = configurator.AddOrUpdateMergeableIngress(test.ingress)
		if err != nil {
			t.Fatal(err)
		}

		gotTraceDataOnAdd, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnAdd.ServiceCount, gotTraceDataOnAdd.ServiceCount) {
			t.Error(cmp.Diff(test.expectedTraceDataOnAdd.ServiceCount, gotTraceDataOnAdd.ServiceCount))
		}

		for i := 0; i < test.deleteCount; i++ {
			ing := test.ingress

			key := fmt.Sprintf("%s/%s", ing.Master.Ingress.Namespace, ing.Master.Ingress.Name)
			err := configurator.DeleteIngress(key, false)
			if err != nil {
				t.Fatal(err)
			}
		}

		if err != nil {
			t.Fatal(err)
		}

		gotTraceDataOnDelete, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceDataOnDelete.ServiceCount, gotTraceDataOnDelete.ServiceCount) {
			t.Error(cmp.Diff(test.expectedTraceDataOnDelete.ServiceCount, gotTraceDataOnDelete.ServiceCount))
		}
	}
}

func createCafeIngressEx() configs.IngressEx {
	cafeIngress := networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cafe-ingress",
			Namespace: "default",
		},
		Spec: networkingV1.IngressSpec{
			TLS: []networkingV1.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []networkingV1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{
							Paths: []networkingV1.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: networkingV1.IngressBackend{
										Service: &networkingV1.IngressServiceBackend{
											Name: "coffee-svc",
											Port: networkingV1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
								{
									Path: "/tea",
									Backend: networkingV1.IngressBackend{
										Service: &networkingV1.IngressServiceBackend{
											Name: "tea-svc",
											Port: networkingV1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	cafeIngressEx := configs.IngressEx{
		Ingress: &cafeIngress,
		Endpoints: map[string][]string{
			"coffee-svc80": {"10.0.0.1:80"},
			"tea-svc80":    {"10.0.0.2:80"},
		},
		ExternalNameSvcs: map[string]bool{},
		ValidHosts: map[string]bool{
			"cafe.example.com": true,
		},
		SecretRefs: map[string]*secrets.SecretReference{
			"cafe-secret": {
				Secret: &coreV1.Secret{
					Type: coreV1.SecretTypeTLS,
				},
				Path: "/etc/nginx/secrets/default-cafe-secret",
			},
		},
	}
	return cafeIngressEx
}

func createMergeableCafeIngress() *configs.MergeableIngresses {
	master := networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
		Spec: networkingV1.IngressSpec{
			TLS: []networkingV1.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []networkingV1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{ // HTTP must not be nil for Master
							Paths: []networkingV1.HTTPIngressPath{},
						},
					},
				},
			},
		},
	}

	coffeeMinion := networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cafe-ingress-coffee-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{
							Paths: []networkingV1.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: networkingV1.IngressBackend{
										Service: &networkingV1.IngressServiceBackend{
											Name: "coffee-svc",
											Port: networkingV1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	teaMinion := networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cafe-ingress-tea-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{
							Paths: []networkingV1.HTTPIngressPath{
								{
									Path: "/tea",
									Backend: networkingV1.IngressBackend{
										Service: &networkingV1.IngressServiceBackend{
											Name: "tea-svc",
											Port: networkingV1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	mergeableIngresses := &configs.MergeableIngresses{
		Master: &configs.IngressEx{
			Ingress: &master,
			Endpoints: map[string][]string{
				"coffee-svc80": {"10.0.0.1:80"},
				"tea-svc80":    {"10.0.0.2:80"},
			},
			ValidHosts: map[string]bool{
				"cafe.example.com": true,
			},
			SecretRefs: map[string]*secrets.SecretReference{
				"cafe-secret": {
					Secret: &coreV1.Secret{
						Type: coreV1.SecretTypeTLS,
					},
					Path:  "/etc/nginx/secrets/default-cafe-secret",
					Error: nil,
				},
			},
		},
		Minions: []*configs.IngressEx{
			{
				Ingress: &coffeeMinion,
				Endpoints: map[string][]string{
					"coffee-svc80": {"10.0.0.1:80"},
				},
				ValidHosts: map[string]bool{
					"cafe.example.com": true,
				},
				ValidMinionPaths: map[string]bool{
					"/coffee": true,
				},
				SecretRefs: map[string]*secrets.SecretReference{},
			},
			{
				Ingress: &teaMinion,
				Endpoints: map[string][]string{
					"tea-svc80": {"10.0.0.2:80"},
				},
				ValidHosts: map[string]bool{
					"cafe.example.com": true,
				},
				ValidMinionPaths: map[string]bool{
					"/tea": true,
				},
				SecretRefs: map[string]*secrets.SecretReference{},
			},
		},
	}

	return mergeableIngresses
}

func createMergeableIngressWithCustomAnnotations(masterAnnotations, coffeeAnnotations, teaAnnotations map[string]string) *configs.MergeableIngresses {
	master := networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        "cafe-ingress-master",
			Namespace:   "default",
			Annotations: masterAnnotations,
		},
		Spec: networkingV1.IngressSpec{
			TLS: []networkingV1.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []networkingV1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{ // HTTP must not be nil for Master
							Paths: []networkingV1.HTTPIngressPath{},
						},
					},
				},
			},
		},
	}

	coffeeMinion := networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        "cafe-ingress-coffee-minion",
			Namespace:   "default",
			Annotations: coffeeAnnotations,
		},
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{
							Paths: []networkingV1.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: networkingV1.IngressBackend{
										Service: &networkingV1.IngressServiceBackend{
											Name: "coffee-svc",
											Port: networkingV1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	teaMinion := networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        "cafe-ingress-tea-minion",
			Namespace:   "default",
			Annotations: teaAnnotations,
		},
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{
							Paths: []networkingV1.HTTPIngressPath{
								{
									Path: "/tea",
									Backend: networkingV1.IngressBackend{
										Service: &networkingV1.IngressServiceBackend{
											Name: "tea-svc",
											Port: networkingV1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	mergeableIngresses := &configs.MergeableIngresses{
		Master: &configs.IngressEx{
			Ingress: &master,
			Endpoints: map[string][]string{
				"coffee-svc80": {"10.0.0.1:80"},
				"tea-svc80":    {"10.0.0.2:80"},
			},
			ValidHosts: map[string]bool{
				"cafe.example.com": true,
			},
			SecretRefs: map[string]*secrets.SecretReference{
				"cafe-secret": {
					Secret: &coreV1.Secret{
						Type: coreV1.SecretTypeTLS,
					},
					Path:  "/etc/nginx/secrets/default-cafe-secret",
					Error: nil,
				},
			},
		},
		Minions: []*configs.IngressEx{
			{
				Ingress: &coffeeMinion,
				Endpoints: map[string][]string{
					"coffee-svc80": {"10.0.0.1:80"},
				},
				ValidHosts: map[string]bool{
					"cafe.example.com": true,
				},
				ValidMinionPaths: map[string]bool{
					"/coffee": true,
				},
				SecretRefs: map[string]*secrets.SecretReference{},
			},
			{
				Ingress: &teaMinion,
				Endpoints: map[string][]string{
					"tea-svc80": {"10.0.0.2:80"},
				},
				ValidHosts: map[string]bool{
					"cafe.example.com": true,
				},
				ValidMinionPaths: map[string]bool{
					"/tea": true,
				},
				SecretRefs: map[string]*secrets.SecretReference{},
			},
		},
	}

	return mergeableIngresses
}

func createCafeIngressExWithCustomAnnotations(annotations map[string]string) configs.IngressEx {
	cafeIngress := networkingV1.Ingress{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        "cafe-ingress",
			Namespace:   "default",
			Annotations: annotations,
		},
		Spec: networkingV1.IngressSpec{
			TLS: []networkingV1.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []networkingV1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networkingV1.IngressRuleValue{
						HTTP: &networkingV1.HTTPIngressRuleValue{
							Paths: []networkingV1.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: networkingV1.IngressBackend{
										Service: &networkingV1.IngressServiceBackend{
											Name: "coffee-svc",
											Port: networkingV1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
								{
									Path: "/tea",
									Backend: networkingV1.IngressBackend{
										Service: &networkingV1.IngressServiceBackend{
											Name: "tea-svc",
											Port: networkingV1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	cafeIngressEx := configs.IngressEx{
		Ingress: &cafeIngress,
		Endpoints: map[string][]string{
			"coffee-svc80": {"10.0.0.1:80"},
			"tea-svc80":    {"10.0.0.2:80"},
		},
		ExternalNameSvcs: map[string]bool{},
		ValidHosts: map[string]bool{
			"cafe.example.com": true,
		},
		SecretRefs: map[string]*secrets.SecretReference{
			"cafe-secret": {
				Secret: &coreV1.Secret{
					Type: coreV1.SecretTypeTLS,
				},
				Path: "/etc/nginx/secrets/default-cafe-secret",
			},
		},
	}
	return cafeIngressEx
}

func getResourceKey(namespace, name string) string {
	return fmt.Sprintf("%s_%s", namespace, name)
}

func newConfiguratorWithIngress(t *testing.T) *configs.Configurator {
	t.Helper()

	ingressEx := createCafeIngressEx()
	c := newConfigurator(t)
	_, err := c.AddOrUpdateIngress(&ingressEx)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func newConfiguratorWithIngressWithCustomAnnotations(t *testing.T, annotations map[string]string) *configs.Configurator {
	t.Helper()

	ingressEx := createCafeIngressExWithCustomAnnotations(annotations)
	c := newConfigurator(t)
	_, err := c.AddOrUpdateIngress(&ingressEx)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func newConfiguratorWithMergeableIngress(t *testing.T) *configs.Configurator {
	t.Helper()

	ingressEx := createMergeableCafeIngress()
	c := newConfigurator(t)
	_, err := c.AddOrUpdateMergeableIngress(ingressEx)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func newConfiguratorWithMergeableIngressCustomAnnotations(t *testing.T, masterAnnotations, coffeeAnnotations, teaAnnotations map[string]string) *configs.Configurator {
	t.Helper()

	ingressEx := createMergeableIngressWithCustomAnnotations(masterAnnotations, coffeeAnnotations, teaAnnotations)
	c := newConfigurator(t)
	_, err := c.AddOrUpdateMergeableIngress(ingressEx)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func newConfigurator(t *testing.T) *configs.Configurator {
	t.Helper()

	templateExecutor, err := version1.NewTemplateExecutor(mainTemplatePath, ingressTemplatePath)
	if err != nil {
		t.Fatal(err)
	}

	templateExecutorV2, err := version2.NewTemplateExecutor(virtualServerTemplatePath, transportServerTemplatePath)
	if err != nil {
		t.Fatal(err)
	}

	manager := nginx.NewFakeManager("/etc/nginx")
	cnf := configs.NewConfigurator(configs.ConfiguratorParams{
		NginxManager: manager,
		StaticCfgParams: &configs.StaticConfigParams{
			HealthStatus:                   true,
			HealthStatusURI:                "/nginx-health",
			NginxStatus:                    true,
			NginxStatusAllowCIDRs:          []string{"127.0.0.1"},
			NginxStatusPort:                8080,
			StubStatusOverUnixSocketForOSS: false,
			NginxVersion:                   nginx.NewVersion("nginx version: nginx/1.25.3 (nginx-plus-r31)"),
		},
		Config:                  configs.NewDefaultConfigParams(false),
		TemplateExecutor:        templateExecutor,
		TemplateExecutorV2:      templateExecutorV2,
		LatencyCollector:        nil,
		LabelUpdater:            nil,
		IsPlus:                  false,
		IsWildcardEnabled:       false,
		IsPrometheusEnabled:     false,
		IsLatencyMetricsEnabled: false,
	})

	return cnf
}

func newSecretStore(t *testing.T) *secrets.LocalSecretStore {
	t.Helper()
	configurator := newConfigurator(t)
	return secrets.NewLocalSecretStore(configurator)
}

// newTestClientset takes k8s runtime objects and returns a k8s fake clientset.
// The clientset is configured to return kubernetes version v1.29.2.
// (call to Discovery().ServerVersion())
//
// version.Info struct can hold more information about K8s platform, for example:
//
//	type Info struct {
//	  Major        string
//	  Minor        string
//	  GitVersion   string
//	  GitCommit    string
//	  GitTreeState string
//	  BuildDate    string
//	  GoVersion    string
//	  Compiler     string
//	  Platform     string
//	}
func newTestClientset(objects ...k8sruntime.Object) *testClient.Clientset {
	client := testClient.NewSimpleClientset(objects...)
	client.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
		GitVersion: "v1.29.2",
	}
	return client
}

const (
	mainTemplatePath            = "../configs/version1/nginx-plus.tmpl"
	ingressTemplatePath         = "../configs/version1/nginx-plus.ingress.tmpl"
	virtualServerTemplatePath   = "../configs/version2/nginx-plus.virtualserver.tmpl"
	transportServerTemplatePath = "../configs/version2/nginx-plus.transportserver.tmpl"
)

// telemetryNICData holds static test data for telemetry tests.
var telemetryNICData = tel.Data{
	ProjectName:         "NIC",
	ProjectVersion:      "3.5.0",
	ClusterVersion:      "v1.29.2",
	ProjectArchitecture: runtime.GOARCH,
	ClusterID:           "329766ff-5d78-4c9e-8736-7faad1f2e937",
	ClusterNodeCount:    1,
	ClusterPlatform:     "other",
}

// Policies used for testing for PolicyCount method
var (
	rateLimitPolicy = &conf_v1.Policy{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Policy",
			APIVersion: "k8s.nginx.org/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "rate-limit-policy3",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			RateLimit: &conf_v1.RateLimit{},
		},
		Status: conf_v1.PolicyStatus{},
	}

	rateLimitPolicyInvalid = &conf_v1.Policy{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Policy",
			APIVersion: "k8s.nginx.org/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "INVALID-rate-limit-policy",
			Namespace: "default",
		},
		Spec:   conf_v1.PolicySpec{},
		Status: conf_v1.PolicyStatus{},
	}

	egressMTLSPolicy = &conf_v1.Policy{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Policy",
			APIVersion: "k8s.nginx.org/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "rate-limit-policy3",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			EgressMTLS: &conf_v1.EgressMTLS{},
		},
		Status: conf_v1.PolicyStatus{},
	}

	oidcPolicy = &conf_v1.Policy{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Policy",
			APIVersion: "k8s.nginx.org/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "rate-limit-policy3",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			OIDC: &conf_v1.OIDC{},
		},
		Status: conf_v1.PolicyStatus{},
	}

	wafPolicy = &conf_v1.Policy{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Policy",
			APIVersion: "k8s.nginx.org/v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "rate-limit-policy3",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			WAF: &conf_v1.WAF{},
		},
		Status: conf_v1.PolicyStatus{},
	}
)

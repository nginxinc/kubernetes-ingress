package telemetry_test

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/telemetry"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	tel "github.com/nginxinc/telemetry-exporter/pkg/telemetry"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		telData,
		nicResourceCounts,
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.VirtualServerSpec{},
					},
				},
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
							Namespace: "ns-1",
							Name:      "coffee",
						},
						Spec: conf_v1.VirtualServerSpec{},
					},
				},
				{
					VirtualServer: &conf_v1.VirtualServer{
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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
						ObjectMeta: v1.ObjectMeta{
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

func getResourceKey(namespace, name string) string {
	return fmt.Sprintf("%s_%s", namespace, name)
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

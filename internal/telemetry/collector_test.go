package telemetry_test

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/telemetry"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	_ "github.com/nginxinc/telemetry-exporter/pkg/telemetry"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
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
		Version:         "3.5.0",
	}
	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		ProjectMeta: telemetry.ProjectMeta{
			Name:    "NIC",
			Version: "3.5.0",
		},
		K8sVersion: "v1.29.2",
		Arch:       runtime.GOARCH,
	}
	want := fmt.Sprintf("%+v", td)
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
		Version:         "3.5.0",
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		ProjectMeta: telemetry.ProjectMeta{
			Name:    "NIC",
			Version: "3.5.0",
		},
		NICResourceCounts: telemetry.NICResourceCounts{
			VirtualServers:      0,
			VirtualServerRoutes: 0,
			TransportServers:    0,
		},
		NodeCount:  1,
		K8sVersion: "v1.29.2",
		Arch:       runtime.GOARCH,
	}
	want := fmt.Sprintf("%+v", td)
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
		Version:         "3.5.0",
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		ProjectMeta: telemetry.ProjectMeta{
			Name:    "NIC",
			Version: "3.5.0",
		},
		NICResourceCounts: telemetry.NICResourceCounts{
			VirtualServers:      0,
			VirtualServerRoutes: 0,
			TransportServers:    0,
		},
		NodeCount:  3,
		K8sVersion: "v1.29.2",
		Arch:       runtime.GOARCH,
	}
	want := fmt.Sprintf("%+v", td)
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
		Version:         "3.5.0",
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		ProjectMeta: telemetry.ProjectMeta{
			Name:    "NIC",
			Version: "3.5.0",
		},
		NICResourceCounts: telemetry.NICResourceCounts{
			VirtualServers:      0,
			VirtualServerRoutes: 0,
			TransportServers:    0,
		},
		NodeCount:  1,
		ClusterID:  "329766ff-5d78-4c9e-8736-7faad1f2e937",
		K8sVersion: "v1.29.2",
		Arch:       runtime.GOARCH,
	}
	want := fmt.Sprintf("%+v", td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCollectK8sVersion(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	cfg := telemetry.CollectorConfig{
		Configurator:    newConfigurator(t),
		K8sClientReader: newTestClientset(node1, kubeNS),
		Version:         "3.5.0",
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		ProjectMeta: telemetry.ProjectMeta{
			Name:    "NIC",
			Version: "3.5.0",
		},
		NICResourceCounts: telemetry.NICResourceCounts{
			VirtualServers:      0,
			VirtualServerRoutes: 0,
			TransportServers:    0,
		},
		NodeCount:  1,
		ClusterID:  "329766ff-5d78-4c9e-8736-7faad1f2e937",
		K8sVersion: "v1.29.2",
		Arch:       runtime.GOARCH,
	}
	want := fmt.Sprintf("%+v", td)
	got := buf.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCountVirtualServers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName                  string
		expectedTraceDataOnAdd    telemetry.Data
		expectedTraceDataOnDelete telemetry.Data
		virtualServers            []*configs.VirtualServerEx
		deleteCount               int
	}{
		{
			testName: "Create and delete 1 VirtualServer",
			expectedTraceDataOnAdd: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 1,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
			},
			expectedTraceDataOnDelete: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 0,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
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
			expectedTraceDataOnAdd: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 2,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
			},
			expectedTraceDataOnDelete: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 0,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
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
			expectedTraceDataOnAdd: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 2,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
			},
			expectedTraceDataOnDelete: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 1,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
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
			K8sClientReader: newTestClientset(dummyKubeNS),
			Configurator:    configurator,
			Version:         "3.5.0",
		})
		if err != nil {
			t.Fatal(err)
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

		if !cmp.Equal(test.expectedTraceDataOnAdd, gotTraceDataOnAdd) {
			t.Error(cmp.Diff(test.expectedTraceDataOnAdd, gotTraceDataOnAdd))
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

		if !cmp.Equal(test.expectedTraceDataOnDelete, gotTraceDataOnDelete) {
			t.Error(cmp.Diff(test.expectedTraceDataOnDelete, gotTraceDataOnDelete))
		}
	}
}

func TestCountTransportServers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testName                  string
		expectedTraceDataOnAdd    telemetry.Data
		expectedTraceDataOnDelete telemetry.Data
		transportServers          []*configs.TransportServerEx
		deleteCount               int
	}{
		{
			testName: "Create and delete 1 TransportServer",
			expectedTraceDataOnAdd: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 1,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
			},
			expectedTraceDataOnDelete: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 0,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
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
			expectedTraceDataOnAdd: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 2,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
			},
			expectedTraceDataOnDelete: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 0,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
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
			expectedTraceDataOnAdd: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 2,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
			},
			expectedTraceDataOnDelete: telemetry.Data{
				ProjectMeta: telemetry.ProjectMeta{
					Name:    "NIC",
					Version: "3.5.0",
				},
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 1,
				},
				K8sVersion: "v1.29.2",
				Arch:       runtime.GOARCH,
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
			K8sClientReader: newTestClientset(dummyKubeNS),
			Configurator:    configurator,
			Version:         "3.5.0",
		})
		if err != nil {
			t.Fatal(err)
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

		if !cmp.Equal(test.expectedTraceDataOnAdd, gotTraceDataOnAdd) {
			t.Error(cmp.Diff(test.expectedTraceDataOnAdd, gotTraceDataOnAdd))
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

		if !cmp.Equal(test.expectedTraceDataOnDelete, gotTraceDataOnDelete) {
			t.Error(cmp.Diff(test.expectedTraceDataOnDelete, gotTraceDataOnDelete))
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
	testClient := testClient.NewSimpleClientset(objects...)
	testClient.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
		GitVersion: "v1.29.2",
	}
	return testClient
}

const (
	mainTemplatePath            = "../configs/version1/nginx-plus.tmpl"
	ingressTemplatePath         = "../configs/version1/nginx-plus.ingress.tmpl"
	virtualServerTemplatePath   = "../configs/version2/nginx-plus.virtualserver.tmpl"
	transportServerTemplatePath = "../configs/version2/nginx-plus.transportserver.tmpl"
)

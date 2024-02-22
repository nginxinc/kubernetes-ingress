package telemetry_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/telemetry"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	td := telemetry.Data{}

	cfg := telemetry.CollectorConfig{
		K8sClientReader: testClient.NewSimpleClientset(),
		Configurator:    newConfigurator(t),
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

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
		K8sClientReader: testClient.NewSimpleClientset(node1),
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		ProjectMeta: telemetry.ProjectMeta{
			Name:    "",
			Version: "",
		},
		NICResourceCounts: telemetry.NICResourceCounts{
			VirtualServers:      0,
			VirtualServerRoutes: 0,
			TransportServers:    0,
		},
		NodeCount: 1,
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
		K8sClientReader: testClient.NewSimpleClientset(node1, node2, node3),
	}

	c, err := telemetry.NewCollector(cfg, telemetry.WithExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	c.Collect(context.Background())

	td := telemetry.Data{
		ProjectMeta: telemetry.ProjectMeta{
			Name:    "",
			Version: "",
		},
		NICResourceCounts: telemetry.NICResourceCounts{
			VirtualServers:      0,
			VirtualServerRoutes: 0,
			TransportServers:    0,
		},
		NodeCount: 3,
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
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 1,
				},
			},
			expectedTraceDataOnDelete: telemetry.Data{
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 0,
				},
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
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 2,
				},
			},
			expectedTraceDataOnDelete: telemetry.Data{
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 0,
				},
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
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 2,
				},
			},
			expectedTraceDataOnDelete: telemetry.Data{
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 1,
				},
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
			K8sClientReader: testClient.NewSimpleClientset(),
			Configurator:    configurator,
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
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 1,
				},
			},
			expectedTraceDataOnDelete: telemetry.Data{
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 0,
				},
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
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 2,
				},
			},
			expectedTraceDataOnDelete: telemetry.Data{
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 0,
				},
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
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 2,
				},
			},
			expectedTraceDataOnDelete: telemetry.Data{
				NICResourceCounts: telemetry.NICResourceCounts{
					TransportServers: 1,
				},
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
			K8sClientReader: testClient.NewSimpleClientset(),
			Configurator:    configurator,
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

const (
	mainTemplatePath            = "../configs/version1/nginx-plus.tmpl"
	ingressTemplatePath         = "../configs/version1/nginx-plus.ingress.tmpl"
	virtualServerTemplatePath   = "../configs/version2/nginx-plus.virtualserver.tmpl"
	transportServerTemplatePath = "../configs/version2/nginx-plus.transportserver.tmpl"
)

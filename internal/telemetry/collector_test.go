package telemetry_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/telemetry"
	customk8sfake "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/google/go-cmp/cmp"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateNewDefaultCollector(t *testing.T) {
	t.Parallel()

	cfg := telemetry.CollectorConfig{}

	c, err := telemetry.NewCollector(cfg)
	if err != nil {
		t.Fatal(err)
	}

	want := 24.0
	got := c.Period.Hours()

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCreateNewCollectorWithCustomReportingPeriod(t *testing.T) {
	t.Parallel()

	cfg := telemetry.CollectorConfig{}

	c, err := telemetry.NewCollector(cfg, telemetry.WithTimePeriod("4h"))
	if err != nil {
		t.Fatal(err)
	}

	want := 4.0
	got := c.Period.Hours()

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCreateNewCollectorWithCustomExporter(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}
	exp := &telemetry.StdoutExporter{Endpoint: buf}
	td := telemetry.Data{}

	cfg := telemetry.CollectorConfig{}

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

func TestBuildReport(t *testing.T) {
	t.Parallel()

	c, err := telemetry.NewCollector(telemetry.CollectorConfig{})
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		testName          string
		collectorConfig   telemetry.CollectorConfig
		expectedTraceData telemetry.Data
		virtualServers    []*conf_v1.VirtualServer
		transportServers  []*conf_v1.TransportServer
	}{
		{
			testName: "Resources deployed in a namespace that is watched",
			expectedTraceData: telemetry.Data{
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 2,
				},
			},
			collectorConfig: telemetry.CollectorConfig{
				K8sClientReader:       k8sfake.NewSimpleClientset(),
				CustomK8sClientReader: customk8sfake.NewSimpleClientset(),
				Namespaces:            []string{"ns-1"},
			},
			virtualServers: []*conf_v1.VirtualServer{
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "coffee",
					},
					Spec: conf_v1.VirtualServerSpec{},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "tea",
					},
					Spec: conf_v1.VirtualServerSpec{},
				},
			},
		},
		{
			testName: "Resource is deployed in a namespace that is not watched",
			expectedTraceData: telemetry.Data{
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 0,
				},
			},
			collectorConfig: telemetry.CollectorConfig{
				K8sClientReader:       k8sfake.NewSimpleClientset(),
				CustomK8sClientReader: customk8sfake.NewSimpleClientset(),
				Namespaces:            []string{"ns-2"},
			},
			virtualServers: []*conf_v1.VirtualServer{
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "coffee",
					},
					Spec: conf_v1.VirtualServerSpec{},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "tea",
					},
					Spec: conf_v1.VirtualServerSpec{},
				},
			},
		},
		{
			testName: "Resource is deployed in a watched namespace with more than 1 watched namespace",
			expectedTraceData: telemetry.Data{
				NICResourceCounts: telemetry.NICResourceCounts{
					VirtualServers: 3,
				},
			},
			collectorConfig: telemetry.CollectorConfig{
				K8sClientReader:       k8sfake.NewSimpleClientset(),
				CustomK8sClientReader: customk8sfake.NewSimpleClientset(),
				Namespaces:            []string{"ns-1", "ns-2", "ns-3"},
			},
			virtualServers: []*conf_v1.VirtualServer{
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns-1",
						Name:      "coffee",
					},
					Spec: conf_v1.VirtualServerSpec{},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns-2",
						Name:      "tea",
					},
					Spec: conf_v1.VirtualServerSpec{},
				},
				{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns-3",
						Name:      "latte",
					},
					Spec: conf_v1.VirtualServerSpec{},
				},
			},
		},
	}

	for _, test := range testCases {
		c.SetConfig(test.collectorConfig)

		for _, vs := range test.virtualServers {
			if _, err = test.collectorConfig.CustomK8sClientReader.K8sV1().
				VirtualServers(vs.Namespace).
				Create(context.Background(), vs, v1.CreateOptions{}); err != nil {
				t.Fatal(err)
			}
		}

		for _, ts := range test.transportServers {
			if _, err = test.collectorConfig.CustomK8sClientReader.K8sV1().
				TransportServers(ts.Namespace).
				Create(context.Background(), ts, v1.CreateOptions{}); err != nil {
				t.Fatal(err)
			}
		}

		c.Config.Namespaces = test.collectorConfig.Namespaces
		gotTraceData, err := c.BuildReport(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if !cmp.Equal(test.expectedTraceData, gotTraceData) {
			t.Error(cmp.Diff(test.expectedTraceData, gotTraceData))
		}
	}
}

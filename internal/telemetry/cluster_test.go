package telemetry_test

import (
	"context"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/telemetry"
	apiCoreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testClient "k8s.io/client-go/kubernetes/fake"
)

func TestNodeCountInAClusterWithThreeNodes(t *testing.T) {
	t.Parallel()

	c := newTestCollectorForCluserWithNodes(t, node1, node2, node3)

	got, err := c.NodeCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := 3
	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNodeCountInAClusterWithOneNode(t *testing.T) {
	t.Parallel()

	c := newTestCollectorForCluserWithNodes(t, node1)
	got, err := c.NodeCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := 1
	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}

// newTestCollectorForClusterWithNodes returns a telemetry collector configured
// to simulate collecting data on a cluser with provided nodes.
func newTestCollectorForCluserWithNodes(t *testing.T, nodes ...runtime.Object) *telemetry.Collector {
	t.Helper()

	c, err := telemetry.NewCollector(
		telemetry.CollectorConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}
	c.Config.K8sClientReader = testClient.NewSimpleClientset(nodes...)
	return c
}

var (
	node1 = &apiCoreV1.Node{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "test-node-1",
			Namespace: "default",
		},
		Spec: apiCoreV1.NodeSpec{},
	}

	node2 = &apiCoreV1.Node{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "test-node-2",
			Namespace: "default",
		},
		Spec: apiCoreV1.NodeSpec{},
	}

	node3 = &apiCoreV1.Node{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "test-node-3",
			Namespace: "default",
		},
		Spec: apiCoreV1.NodeSpec{},
	}
)

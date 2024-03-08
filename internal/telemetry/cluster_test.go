package telemetry_test

import (
	"context"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/telemetry"
	apiCoreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestNodeCountInAClusterWithThreeNodes(t *testing.T) {
	t.Parallel()

	c := newTestCollectorForClusterWithNodes(t, node1, node2, node3)

	got, err := c.NodeCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var want int64 = 3
	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestNodeCountInAClusterWithOneNode(t *testing.T) {
	t.Parallel()

	c := newTestCollectorForClusterWithNodes(t, node1)
	got, err := c.NodeCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var want int64 = 1
	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestClusterIDRetrievesK8sClusterUID(t *testing.T) {
	t.Parallel()

	c := newTestCollectorForClusterWithNodes(t, node1, kubeNS)

	got, err := c.ClusterID(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	want := "329766ff-5d78-4c9e-8736-7faad1f2e937"
	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}

func TestClusterIDErrorsOnNotExistingService(t *testing.T) {
	t.Parallel()

	c := newTestCollectorForClusterWithNodes(t, node1)
	_, err := c.ClusterID(context.Background())
	if err == nil {
		t.Error("want error, got nil")
	}
}

func TestK8sVersionRetrievesClusterVersion(t *testing.T) {
	t.Parallel()

	c := newTestCollectorForClusterWithNodes(t, node1)
	got, err := c.K8sVersion()
	if err != nil {
		t.Fatal(err)
	}

	want := "v1.29.2"
	if want != got {
		t.Errorf("want %s, got %s", want, got)
	}
}

// newTestCollectorForClusterWithNodes returns a telemetry collector configured
// to simulate collecting data on a cluser with provided nodes.
func newTestCollectorForClusterWithNodes(t *testing.T, nodes ...runtime.Object) *telemetry.Collector {
	t.Helper()

	c, err := telemetry.NewCollector(
		telemetry.CollectorConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}
	c.Config.K8sClientReader = newTestClientset(nodes...)
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

	kubeNS = &apiCoreV1.Namespace{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name: "kube-system",
			UID:  "329766ff-5d78-4c9e-8736-7faad1f2e937",
		},
		Spec: apiCoreV1.NamespaceSpec{},
	}

	dummyKubeNS = &apiCoreV1.Namespace{
		TypeMeta: metaV1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name: "kube-system",
			UID:  "",
		},
		Spec: apiCoreV1.NamespaceSpec{},
	}
)

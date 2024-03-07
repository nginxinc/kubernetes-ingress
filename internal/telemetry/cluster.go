package telemetry

import (
	"context"
	"fmt"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NodeCount returns the total number of nodes in the cluster.
// It returns an error if the underlying k8s API client errors.
func (c *Collector) NodeCount(ctx context.Context) (int64, error) {
	nodes, err := c.Config.K8sClientReader.CoreV1().Nodes().List(ctx, metaV1.ListOptions{})
	if err != nil {
		return 0, err
	}
	return int64(len(nodes.Items)), nil
}

// ClusterID returns the UID of the kube-system namespace representing cluster id.
// It returns an error if the underlying k8s API client errors.
func (c *Collector) ClusterID(ctx context.Context) (string, error) {
	cluster, err := c.Config.K8sClientReader.CoreV1().Namespaces().Get(ctx, "kube-system", metaV1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(cluster.UID), nil
}

// K8sVersion returns a string respresenting the K8s version.
// It returns an error if the underlying k8s API client errors.
func (c *Collector) K8sVersion() (string, error) {
	sv, err := c.Config.K8sClientReader.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return sv.String(), nil
}

// ReplicaCount returns a number of running NIC replicas.
func (c *Collector) ReplicaCount(ctx context.Context) (int64, error) {
	pod, err := c.Config.K8sClientReader.CoreV1().Pods(c.Config.PodNSName.Namespace).Get(ctx, c.Config.PodNSName.Name, metaV1.GetOptions{})
	if err != nil {
		return 0, err
	}
	podRef := pod.GetOwnerReferences()
	if len(podRef) != 1 {
		return 0, fmt.Errorf("expected pod owner reference to be 1, got %d", len(podRef))
	}
	if podRef[0].Kind != "ReplicaSet" {
		return 0, fmt.Errorf("expected pod owner reference to be ReplicaSet, got %s", pod.OwnerReferences[0].Kind)
	}
	rs, err := c.Config.K8sClientReader.AppsV1().ReplicaSets(c.Config.PodNSName.Namespace).Get(ctx, podRef[0].Name, metaV1.GetOptions{})
	if err != nil {
		return 0, err
	}
	return int64(*rs.Spec.Replicas), nil
}

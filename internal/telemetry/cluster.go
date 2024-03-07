package telemetry

import (
	"context"
	"errors"
	"strings"

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

// Platform returns a string representing platform name.
func (c *Collector) Platform(ctx context.Context) (string, error) {
	nodes, err := c.Config.K8sClientReader.CoreV1().Nodes().List(ctx, metaV1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(nodes.Items) == 0 {
		return "", errors.New("no nodes in the cluster, cannot determine platform name")
	}
	return lookupPlatform(nodes.Items[0].Spec.ProviderID), nil
}

// lookupPlatform takes a string representing a K8s PlatformID
// retrieved from a cluster node and returns a string
// representing the platform name.
func lookupPlatform(platformID string) string {
	platform := strings.ToLower(platformID)
	if strings.HasPrefix(platform, "aws") {
		return "aws"
	}
	if strings.HasPrefix(platform, "azure") {
		return "azure"
	}
	if strings.HasPrefix(platform, "gce") {
		return "gke"
	}
	if strings.HasPrefix(platform, "kind") {
		return "kind"
	}
	if strings.HasPrefix(platform, "vsphere") {
		return "vsphere"
	}
	if strings.HasPrefix(platform, "k3s") {
		return "k3s"
	}
	return "other"
}

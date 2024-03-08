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
//
// Cloud providers identified by PlatformID (in K8s SIGs):
// https://github.com/orgs/kubernetes-sigs/repositories?q=cluster-api-provider
//
//gocyclo:ignore
func lookupPlatform(providerID string) string {
	provider := strings.TrimSpace(providerID)
	// The case when the ProviderID field not used by the cloud provider.
	if provider == "" {
		return "other"
	}

	provider = strings.ToLower(providerID)
	if strings.HasPrefix(provider, "aws") {
		return "aws"
	}
	if strings.HasPrefix(provider, "azure") {
		return "azure"
	}
	if strings.HasPrefix(provider, "gce") {
		return "gke"
	}
	if strings.HasPrefix(provider, "kind") {
		return "kind"
	}
	if strings.HasPrefix(provider, "vsphere") {
		return "vsphere"
	}
	if strings.HasPrefix(provider, "k3s") {
		return "k3s"
	}
	if strings.HasPrefix(provider, "ibmcloud") {
		return "ibmcloud"
	}
	if strings.HasPrefix(provider, "ibmpowervs") {
		return "ibmpowervs"
	}
	if strings.HasPrefix(provider, "cloudstack") {
		return "cloudstack"
	}
	if strings.HasPrefix(provider, "openstack") {
		return "openstack"
	}
	if strings.HasPrefix(provider, "digitalocean") {
		return "digitalocean"
	}
	if strings.HasPrefix(provider, "equinixmetal") {
		return "equinixmetal"
	}
	if strings.HasPrefix(provider, "alicloud") {
		return "alicloud"
	}

	p := strings.Split(provider, ":")
	if len(p) == 0 {
		return "other"
	}
	if p[0] == "" {
		return "other"
	}
	return p[0]
}

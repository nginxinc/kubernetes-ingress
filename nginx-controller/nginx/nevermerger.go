package nginx

import (
	"fmt"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

type NeverMerger struct {
	hostIngressMapping map[string]string
}

func NewNeverMerger() Merger {
	return &NeverMerger{
		map[string]string{},
	}
}

func (n *NeverMerger) Merge(ingress *extensions.Ingress, configs []IngressNginxConfig) []IngressNginxConfig {
	ingressName := fmt.Sprintf("%s/%s", ingress.GetNamespace(), ingress.GetName())
	configsWithoutConflict := []IngressNginxConfig{}
	for _, config := range configs {
		if conflictingIngress, exists := n.hostIngressMapping[config.Server.Name]; exists {
			// there is already a config using this servername, reject it
			glog.Errorf("Conflicting server with name '%s' in ingress %s/%s, a server with this name is already defined in ingress %s, ignored", config.Server.Name, ingress.GetNamespace(), ingress.GetName(), conflictingIngress)
			continue
		}
		configsWithoutConflict = append(configsWithoutConflict, config)
		n.hostIngressMapping[config.Server.Name] = ingressName
	}
	return configsWithoutConflict
}

func (n *NeverMerger) Separate(ingressName string) ([]IngressNginxConfig, []string) {
	deleted := []string{}
	for host, ingress := range n.hostIngressMapping {
		if ingress == ingressName {
			deleted = append(deleted, host)
		}
	}
	return []IngressNginxConfig{}, deleted
}

package nginx

import "k8s.io/kubernetes/pkg/apis/extensions"

type Merger interface {
	// Merge takes a ingress object and the IngressNginxConfig generated for it and
	// returns a list of affected/changed IngressNginxConfigs
	Merge(*extensions.Ingress, []IngressNginxConfig) []IngressNginxConfig

	// Separate takes the name of a ingress object and will remove all changes made to other existing IngressNginxConfigs
	// returns a list of affected/changed IngressNginxConfigs and a list of server names that are no longer referenced by any ingress
	Separate(string) ([]IngressNginxConfig, []string)
}

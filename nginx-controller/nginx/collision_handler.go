package nginx

import "k8s.io/kubernetes/pkg/apis/extensions"

// CollisionHandler rosolves collisions in the generated configuration, so the nginx configuration stays valid.
type CollisionHandler interface {
	// AddConfigs is resolving definition collisions that would result in a invalid nginx configuration
	AddConfigs(ing *extensions.Ingress, ingServers []Server) (changed []Server, err error)

	// RemoveConfigs frees server definitions, so they can be used again
	RemoveConfigs(key string) (changed []Server, deleted []Server, err error)
}

type cacheEntry struct {
	Ingress extensions.Ingress
	Servers []Server
}

type cacheEntryList []cacheEntry

func (list cacheEntryList) Len() int {
	return len(list)
}
func (list cacheEntryList) Less(i, j int) bool {
	return list[i].Ingress.CreationTimestamp.Before(list[j].Ingress.CreationTimestamp)
}
func (list cacheEntryList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

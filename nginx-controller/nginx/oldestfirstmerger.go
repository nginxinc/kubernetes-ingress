package nginx

import (
	"fmt"
	"sort"

	"k8s.io/kubernetes/pkg/apis/extensions"
)

type cacheEntry struct {
	Ingress extensions.Ingress
	Configs []IngressNginxConfig
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

type OldestFirstMerger struct {
	cache              map[string]cacheEntry
	hostIngressMapping map[string]map[string]bool
}

func NewOldestFirstMerger() Merger {
	return &OldestFirstMerger{
		map[string]cacheEntry{},
		map[string]map[string]bool{},
	}
}

func (i *OldestFirstMerger) Merge(ingress *extensions.Ingress, configs []IngressNginxConfig) []IngressNginxConfig {
	ingressName := fmt.Sprintf("%s/%s", ingress.GetNamespace(), ingress.GetName())
	hosts := []string{}
	i.cache[ingressName] = cacheEntry{
		Ingress: *ingress,
		Configs: configs,
	}
	for _, config := range configs {
		hosts = append(hosts, config.Server.Name)
	}
	i.updateHostIngressMapping(ingressName, hosts)

	result := []IngressNginxConfig{}
	for _, config := range configs {
		result = append(result, *i.addOrUpdateServer(&config.Server))
	}
	return result
}

func (i *OldestFirstMerger) Separate(ingressName string) ([]IngressNginxConfig, []string) {
	results := []IngressNginxConfig{}
	affectedHosts := []string{}
	deletedEntry, exists := i.cache[ingressName]
	if !exists {
		return results, affectedHosts
	}
	for _, config := range deletedEntry.Configs {
		affectedHosts = append(affectedHosts, config.Server.Name)
	}

	stillExistingHosts := map[string]bool{}
	i.removeHostIngressMapping(ingressName)

	for _, host := range affectedHosts {
		servers := i.getOrderedServerList(host)
		for _, server := range servers {
			if server.Name == host {
				stillExistingHosts[host] = true
			}
		}

		if len(servers) == 0 {
			continue
		}
		baseServer := &servers[0]
		if len(servers) > 1 {
			for _, server := range servers {
				baseServer = i.mergeServers(*baseServer, &server)
			}
		}
		results = append(results, IngressNginxConfig{
			Server:    *baseServer,
			Upstreams: i.getUpstreamsForServer(baseServer),
		})
	}

	deletedHosts := []string{}
	for _, host := range affectedHosts {
		if _, ok := stillExistingHosts[host]; !ok {
			deletedHosts = append(deletedHosts, host)
		}
	}

	delete(i.cache, ingressName)
	return results, deletedHosts
}

func (i *OldestFirstMerger) addOrUpdateServer(server *Server) *IngressNginxConfig {
	var baseServer Server
	if len(i.hostIngressMapping[server.Name]) > 1 {
		// the server must be composed of multiple ingress objects
		servers := i.getOrderedServerList(server.Name)
		for si, server := range servers {
			if si == 0 {
				baseServer = server
			} else {
				baseServer = *(i.mergeServers(baseServer, &server))
			}
		}
	} else {
		// the server is not composed
		baseServer = *server
	}
	upstreams := i.getUpstreamsForServer(&baseServer)
	return &IngressNginxConfig{
		Server:    baseServer,
		Upstreams: upstreams,
	}
}

func (i *OldestFirstMerger) getOrderedServerList(host string) []Server {
	affectedCacheEntries := cacheEntryList{}
	for ingressName := range i.hostIngressMapping[host] {
		affectedCacheEntries = append(affectedCacheEntries, i.cache[ingressName])
	}
	sort.Sort(affectedCacheEntries)

	results := []Server{}
	for _, cacheEntry := range affectedCacheEntries {
		for _, config := range cacheEntry.Configs {
			if config.Server.Name == host {
				results = append(results, config.Server)
			}
		}
	}
	return results
}

func (i *OldestFirstMerger) getUpstreamsForServer(server *Server) []Upstream {
	tmp := map[string]Upstream{}
	for _, location := range server.Locations {
		tmp[location.Upstream.Name] = location.Upstream
	}

	result := []Upstream{}
	for _, upstream := range tmp {
		result = append(result, upstream)
	}
	return result
}

func (i *OldestFirstMerger) mergeServers(base Server, merge *Server) *Server {
	locationMap := map[string]Location{}
	for _, location := range base.Locations {
		locationMap[location.Path] = location
	}
	for _, location := range merge.Locations {
		locationMap[location.Path] = location
	}

	if merge.SSL {
		base.SSL = true
		base.SSLCertificate = merge.SSLCertificate
		base.SSLCertificateKey = merge.SSLCertificateKey
	}
	if merge.HTTP2 {
		base.HTTP2 = true
	}
	if merge.HSTS {
		base.HSTS = true
		base.HSTSMaxAge = merge.HSTSMaxAge
		base.HSTSIncludeSubdomains = merge.HSTSIncludeSubdomains
	}

	base.Locations = []Location{}
	for _, location := range locationMap {
		base.Locations = append(base.Locations, location)
	}
	return &base
}

func (i *OldestFirstMerger) removeHostIngressMapping(ingressName string) {
	for _, ingMap := range i.hostIngressMapping {
		delete(ingMap, ingressName)
	}
}

func (i *OldestFirstMerger) updateHostIngressMapping(ingressName string, hosts []string) {
	i.removeHostIngressMapping(ingressName)
	for _, host := range hosts {
		if _, ok := i.hostIngressMapping[host]; !ok {
			i.hostIngressMapping[host] = map[string]bool{}
		}
		i.hostIngressMapping[host][ingressName] = true
	}
}

package nginx

import (
	"fmt"
	"sort"

	"k8s.io/kubernetes/pkg/apis/extensions"
)

type MergingCollisionHandler struct {
	cache              map[string]cacheEntry
	hostIngressMapping map[string]map[string]bool
}

func NewMergingCollisionHandler() CollisionHandler {
	return &MergingCollisionHandler{
		map[string]cacheEntry{},
		map[string]map[string]bool{},
	}
}

func (m *MergingCollisionHandler) AddConfigs(ingress *extensions.Ingress, servers []Server) ([]Server, error) {
	ingressKey := fmt.Sprintf("%s/%s", ingress.GetNamespace(), ingress.GetName())
	m.cache[ingressKey] = cacheEntry{
		Ingress: *ingress,
		Servers: servers,
	}

	hosts := []string{}
	for _, server := range servers {
		hosts = append(hosts, server.Name)
	}
	m.updateHostIngressMapping(ingressKey, hosts)

	result := []Server{}
	for _, server := range servers {
		result = append(result, *m.addOrUpdateServer(&server))
	}
	return result, nil
}

func (m *MergingCollisionHandler) RemoveConfigs(ingressKey string) ([]Server, []Server, error) {
	deletedCacheEntry, exists := m.cache[ingressKey]
	if !exists {
		return nil, nil, fmt.Errorf("Ingress '%s' cannot be removed, because it was not found in the mapping", ingressKey)
	}
	delete(m.cache, ingressKey)

	changed := []Server{}
	affectedHosts := []string{}
	for _, server := range deletedCacheEntry.Servers {
		affectedHosts = append(affectedHosts, server.Name)
	}

	stillExistingHosts := map[string]bool{}
	m.removeHostIngressMapping(ingressKey)

	for _, server := range deletedCacheEntry.Servers {
		host := server.Name
		servers := m.getOrderedServerList(host)
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
				baseServer = m.mergeServers(*baseServer, &server)
			}
		}
		baseServer.Upstreams = m.getUpstreamsForServer(baseServer)
		changed = append(changed, *baseServer)
	}

	deleted := []Server{}
	for _, server := range deletedCacheEntry.Servers {
		if _, ok := stillExistingHosts[server.Name]; !ok {
			deleted = append(deleted, server)
		}
	}

	return changed, deleted, nil
}

func (m *MergingCollisionHandler) addOrUpdateServer(server *Server) *Server {
	var baseServer Server
	if len(m.hostIngressMapping[server.Name]) > 1 {
		// the server must be composed of multiple ingress objects
		servers := m.getOrderedServerList(server.Name)
		for si, server := range servers {
			if si == 0 {
				baseServer = server
			} else {
				baseServer = *(m.mergeServers(baseServer, &server))
			}
		}
	} else {
		// the server is not composed
		baseServer = *server
	}
	baseServer.Upstreams = m.getUpstreamsForServer(&baseServer)
	return &baseServer
}

func (m *MergingCollisionHandler) getOrderedServerList(host string) []Server {
	affectedCacheEntries := cacheEntryList{}
	for ingressName := range m.hostIngressMapping[host] {
		affectedCacheEntries = append(affectedCacheEntries, m.cache[ingressName])
	}
	sort.Sort(affectedCacheEntries)

	results := []Server{}
	for _, cacheEntry := range affectedCacheEntries {
		for _, server := range cacheEntry.Servers {
			if server.Name == host {
				results = append(results, server)
			}
		}
	}
	return results
}

func (m *MergingCollisionHandler) getUpstreamsForServer(server *Server) []Upstream {
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

func (i *MergingCollisionHandler) mergeServers(base Server, merge *Server) *Server {
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

func (m *MergingCollisionHandler) removeHostIngressMapping(ingressKey string) {
	for _, ingMap := range m.hostIngressMapping {
		delete(ingMap, ingressKey)
	}
}

func (m *MergingCollisionHandler) updateHostIngressMapping(ingressKey string, hosts []string) {
	m.removeHostIngressMapping(ingressKey)
	for _, host := range hosts {
		if _, ok := m.hostIngressMapping[host]; !ok {
			m.hostIngressMapping[host] = map[string]bool{}
		}
		m.hostIngressMapping[host][ingressKey] = true
	}
}

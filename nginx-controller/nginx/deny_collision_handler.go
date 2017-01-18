package nginx

import (
	"fmt"
	"sort"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

// DenyCollisionHandler skips colliding server definitions and emmits an error message
type DenyCollisionHandler struct {
	cache              map[string]cacheEntry
	hostIngressMapping map[string]string
}

func NewDenyCollisionHandler() CollisionHandler {
	return &DenyCollisionHandler{
		map[string]cacheEntry{},
		map[string]string{},
	}
}

func (n *DenyCollisionHandler) AddConfigs(ingress *extensions.Ingress, servers []Server) ([]Server, error) {
	ingressKey := fmt.Sprintf("%s/%s", ingress.GetNamespace(), ingress.GetName())
	n.cache[ingressKey] = cacheEntry{
		Ingress: *ingress,
		Servers: servers,
	}
	serversWithoutConflict := []Server{}
	for _, server := range servers {
		if ingressLockingServerName, exists := n.hostIngressMapping[server.Name]; exists {
			// there is already a config using this servername
			if n.cache[ingressLockingServerName].Ingress.CreationTimestamp.Before(ingress.CreationTimestamp) {
				// the ingress object currently holding the lock on this server name is older than the new one,
				// so it is still holder of the log
				glog.Errorf("Conflicting server with name '%s' in ingress %s/%s, a server with this name is already defined in ingress %s, ignored", server.Name, ingress.GetNamespace(), ingress.GetName(), ingressLockingServerName)
				continue
			}
		}
		serversWithoutConflict = append(serversWithoutConflict, server)
		n.hostIngressMapping[server.Name] = ingressKey
	}

	return serversWithoutConflict, nil
}

func (n *DenyCollisionHandler) RemoveConfigs(ingressKey string) ([]Server, []Server, error) {
	deletedCacheEntry, exists := n.cache[ingressKey]
	if !exists {
		return nil, nil, fmt.Errorf("Ingress '%s' cannot be removed, because it was not found in the mapping", ingressKey)
	}
	delete(n.cache, ingressKey)

	freedServerNames := n.getServerNamesBlockedByIngressKey(ingressKey)
	changedServers := []Server{}
	deletedServers := []Server{}
	for _, serverName := range freedServerNames {
		// server name is now free again
		delete(n.hostIngressMapping, serverName)
		cacheEntries := n.getCacheEntryServersByServerName(serverName)
		for _, entry := range cacheEntries {
			// there is another cacheEntry that wants to use this host,
			// check this server for collisions and add it to the changed servers
			serversWithoutConflict, err := n.AddConfigs(&entry.Ingress, entry.Servers)
			if err != nil {
				return nil, nil, err
			}
			changedServers = append(changedServers, serversWithoutConflict...)
		}

		if len(cacheEntries) == 0 {
			// this serverName is not in use anymore and can be deleted
			for _, server := range deletedCacheEntry.Servers {
				if server.Name == serverName {
					deletedServers = append(deletedServers, server)
				}
			}
		}
	}

	return changedServers, deletedServers, nil
}

func (n *DenyCollisionHandler) getServerNamesBlockedByIngressKey(ingressKey string) []string {
	blocked := []string{}
	for host, ingress := range n.hostIngressMapping {
		if ingress == ingressKey {
			blocked = append(blocked, host)
		}
	}
	return blocked
}

// returns a sorted list of cache entries including only servers with a matching server name
func (n *DenyCollisionHandler) getCacheEntryServersByServerName(serverName string) []*cacheEntry {
	entryList := cacheEntryList{}
	for _, cacheEntry := range n.cache {
		entryList = append(entryList, cacheEntry)
	}
	sort.Sort(entryList)

	results := []*cacheEntry{}
	for _, entry := range entryList {
		servers := []Server{}
		for _, server := range entry.Servers {
			if server.Name == serverName {
				servers = append(servers, server)
			}
		}
		if len(servers) > 0 {
			results = append(results, &cacheEntry{entry.Ingress, servers})
		}
	}
	return results
}

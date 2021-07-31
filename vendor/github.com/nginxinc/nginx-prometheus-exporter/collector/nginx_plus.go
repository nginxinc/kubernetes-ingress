package collector

import (
	"fmt"
	"log"
	"sync"

	plusclient "github.com/nginxinc/nginx-plus-go-client/client"
	"github.com/prometheus/client_golang/prometheus"
)

// LabelUpdater updates the labels of upstream server and server zone metrics
type LabelUpdater interface {
	UpdateUpstreamServerPeerLabels(upstreamServerPeerLabels map[string][]string)
	DeleteUpstreamServerPeerLabels(peers []string)
	UpdateUpstreamServerLabels(upstreamServerLabelValues map[string][]string)
	DeleteUpstreamServerLabels(upstreamNames []string)
	UpdateStreamUpstreamServerPeerLabels(streamUpstreamServerPeerLabels map[string][]string)
	DeleteStreamUpstreamServerPeerLabels(peers []string)
	UpdateStreamUpstreamServerLabels(streamUpstreamServerPeerLabels map[string][]string)
	DeleteStreamUpstreamServerLabels(peers []string)
	UpdateServerZoneLabels(serverZoneLabelValues map[string][]string)
	DeleteServerZoneLabels(zoneNames []string)
	UpdateStreamServerZoneLabels(streamServerZoneLabelValues map[string][]string)
	DeleteStreamServerZoneLabels(zoneNames []string)
}

// NginxPlusCollector collects NGINX Plus metrics. It implements prometheus.Collector interface.
type NginxPlusCollector struct {
	nginxClient                    *plusclient.NginxClient
	totalMetrics                   map[string]*prometheus.Desc
	serverZoneMetrics              map[string]*prometheus.Desc
	upstreamMetrics                map[string]*prometheus.Desc
	upstreamServerMetrics          map[string]*prometheus.Desc
	streamServerZoneMetrics        map[string]*prometheus.Desc
	streamZoneSyncMetrics          map[string]*prometheus.Desc
	streamUpstreamMetrics          map[string]*prometheus.Desc
	streamUpstreamServerMetrics    map[string]*prometheus.Desc
	locationZoneMetrics            map[string]*prometheus.Desc
	resolverMetrics                map[string]*prometheus.Desc
	upMetric                       prometheus.Gauge
	mutex                          sync.Mutex
	variableLabelNames             VariableLabelNames
	upstreamServerLabels           map[string][]string
	streamUpstreamServerLabels     map[string][]string
	serverZoneLabels               map[string][]string
	streamServerZoneLabels         map[string][]string
	upstreamServerPeerLabels       map[string][]string
	streamUpstreamServerPeerLabels map[string][]string
	variableLabelsMutex            sync.RWMutex
}

// UpdateUpstreamServerPeerLabels updates the Upstream Server Peer Labels
func (c *NginxPlusCollector) UpdateUpstreamServerPeerLabels(upstreamServerPeerLabels map[string][]string) {
	c.variableLabelsMutex.Lock()
	for k, v := range upstreamServerPeerLabels {
		c.upstreamServerPeerLabels[k] = v
	}
	c.variableLabelsMutex.Unlock()
}

// DeleteUpstreamServerPeerLabels deletes the Upstream Server Peer Labels
func (c *NginxPlusCollector) DeleteUpstreamServerPeerLabels(peers []string) {
	c.variableLabelsMutex.Lock()
	for _, k := range peers {
		delete(c.upstreamServerPeerLabels, k)
	}
	c.variableLabelsMutex.Unlock()
}

// UpdateStreamUpstreamServerPeerLabels updates the Upstream Server Peer Labels
func (c *NginxPlusCollector) UpdateStreamUpstreamServerPeerLabels(streamUpstreamServerPeerLabels map[string][]string) {
	c.variableLabelsMutex.Lock()
	for k, v := range streamUpstreamServerPeerLabels {
		c.streamUpstreamServerPeerLabels[k] = v
	}
	c.variableLabelsMutex.Unlock()
}

// DeleteStreamUpstreamServerPeerLabels deletes the Upstream Server Peer Labels
func (c *NginxPlusCollector) DeleteStreamUpstreamServerPeerLabels(peers []string) {
	c.variableLabelsMutex.Lock()
	for _, k := range peers {
		delete(c.streamUpstreamServerPeerLabels, k)
	}
	c.variableLabelsMutex.Unlock()
}

// UpdateUpstreamServerLabels updates the Upstream Server Labels
func (c *NginxPlusCollector) UpdateUpstreamServerLabels(upstreamServerLabelValues map[string][]string) {
	c.variableLabelsMutex.Lock()
	for k, v := range upstreamServerLabelValues {
		c.upstreamServerLabels[k] = v
	}
	c.variableLabelsMutex.Unlock()
}

// DeleteUpstreamServerLabels deletes the Upstream Server Labels
func (c *NginxPlusCollector) DeleteUpstreamServerLabels(upstreamNames []string) {
	c.variableLabelsMutex.Lock()
	for _, k := range upstreamNames {
		delete(c.upstreamServerLabels, k)
	}
	c.variableLabelsMutex.Unlock()
}

// UpdateStreamUpstreamServerLabels updates the Upstream Server Labels
func (c *NginxPlusCollector) UpdateStreamUpstreamServerLabels(streamUpstreamServerLabelValues map[string][]string) {
	c.variableLabelsMutex.Lock()
	for k, v := range streamUpstreamServerLabelValues {
		c.streamUpstreamServerLabels[k] = v
	}
	c.variableLabelsMutex.Unlock()
}

// DeleteStreamUpstreamServerLabels deletes the Upstream Server Labels
func (c *NginxPlusCollector) DeleteStreamUpstreamServerLabels(streamUpstreamNames []string) {
	c.variableLabelsMutex.Lock()
	for _, k := range streamUpstreamNames {
		delete(c.streamUpstreamServerLabels, k)
	}
	c.variableLabelsMutex.Unlock()
}

// UpdateServerZoneLabels updates the Server Zone Labels
func (c *NginxPlusCollector) UpdateServerZoneLabels(serverZoneLabelValues map[string][]string) {
	c.variableLabelsMutex.Lock()
	for k, v := range serverZoneLabelValues {
		c.serverZoneLabels[k] = v
	}
	c.variableLabelsMutex.Unlock()
}

// DeleteServerZoneLabels deletes the Server Zone Labels
func (c *NginxPlusCollector) DeleteServerZoneLabels(zoneNames []string) {
	c.variableLabelsMutex.Lock()
	for _, k := range zoneNames {
		delete(c.serverZoneLabels, k)
	}
	c.variableLabelsMutex.Unlock()
}

// UpdateStreamServerZoneLabels updates the Stream Server Zone Labels
func (c *NginxPlusCollector) UpdateStreamServerZoneLabels(streamServerZoneLabelValues map[string][]string) {
	c.variableLabelsMutex.Lock()
	for k, v := range streamServerZoneLabelValues {
		c.streamServerZoneLabels[k] = v
	}
	c.variableLabelsMutex.Unlock()
}

// DeleteStreamServerZoneLabels deletes the Stream Server Zone Labels
func (c *NginxPlusCollector) DeleteStreamServerZoneLabels(zoneNames []string) {
	c.variableLabelsMutex.Lock()
	for _, k := range zoneNames {
		delete(c.streamServerZoneLabels, k)
	}
	c.variableLabelsMutex.Unlock()
}

func (c *NginxPlusCollector) getUpstreamServerLabelValues(upstreamName string) []string {
	c.variableLabelsMutex.RLock()
	defer c.variableLabelsMutex.RUnlock()
	return c.upstreamServerLabels[upstreamName]
}

func (c *NginxPlusCollector) getStreamUpstreamServerLabelValues(upstreamName string) []string {
	c.variableLabelsMutex.RLock()
	defer c.variableLabelsMutex.RUnlock()
	return c.streamUpstreamServerLabels[upstreamName]
}

func (c *NginxPlusCollector) getServerZoneLabelValues(zoneName string) []string {
	c.variableLabelsMutex.RLock()
	defer c.variableLabelsMutex.RUnlock()
	return c.serverZoneLabels[zoneName]
}

func (c *NginxPlusCollector) getStreamServerZoneLabelValues(zoneName string) []string {
	c.variableLabelsMutex.RLock()
	defer c.variableLabelsMutex.RUnlock()
	return c.streamServerZoneLabels[zoneName]
}

func (c *NginxPlusCollector) getUpstreamServerPeerLabelValues(peer string) []string {
	c.variableLabelsMutex.RLock()
	defer c.variableLabelsMutex.RUnlock()
	return c.upstreamServerPeerLabels[peer]
}

func (c *NginxPlusCollector) getStreamUpstreamServerPeerLabelValues(peer string) []string {
	c.variableLabelsMutex.RLock()
	defer c.variableLabelsMutex.RUnlock()
	return c.streamUpstreamServerPeerLabels[peer]
}

// VariableLabelNames holds all the variable label names for the different metrics
type VariableLabelNames struct {
	UpstreamServerVariableLabelNames           []string
	ServerZoneVariableLabelNames               []string
	UpstreamServerPeerVariableLabelNames       []string
	StreamUpstreamServerPeerVariableLabelNames []string
	StreamServerZoneVariableLabelNames         []string
	StreamUpstreamServerVariableLabelNames     []string
}

// NewVariableLabels creates a new struct for VariableNames for the collector
func NewVariableLabelNames(upstreamServerVariableLabelNames []string, serverZoneVariableLabelNames []string, upstreamServerPeerVariableLabelNames []string,
	streamUpstreamServerVariableLabelNames []string, streamServerZoneLabels []string, streamUpstreamServerPeerVariableLabelNames []string) VariableLabelNames {
	return VariableLabelNames{
		UpstreamServerVariableLabelNames:           upstreamServerVariableLabelNames,
		ServerZoneVariableLabelNames:               serverZoneVariableLabelNames,
		UpstreamServerPeerVariableLabelNames:       upstreamServerPeerVariableLabelNames,
		StreamUpstreamServerVariableLabelNames:     streamUpstreamServerVariableLabelNames,
		StreamServerZoneVariableLabelNames:         streamServerZoneLabels,
		StreamUpstreamServerPeerVariableLabelNames: streamUpstreamServerPeerVariableLabelNames,
	}
}

// NewNginxPlusCollector creates an NginxPlusCollector.
func NewNginxPlusCollector(nginxClient *plusclient.NginxClient, namespace string, variableLabelNames VariableLabelNames, constLabels map[string]string) *NginxPlusCollector {
	upstreamServerVariableLabelNames := append(variableLabelNames.UpstreamServerVariableLabelNames, variableLabelNames.UpstreamServerPeerVariableLabelNames...)
	streamUpstreamServerVariableLabelNames := append(variableLabelNames.StreamUpstreamServerVariableLabelNames, variableLabelNames.StreamUpstreamServerPeerVariableLabelNames...)
	return &NginxPlusCollector{
		variableLabelNames:             variableLabelNames,
		upstreamServerLabels:           make(map[string][]string),
		serverZoneLabels:               make(map[string][]string),
		streamServerZoneLabels:         make(map[string][]string),
		upstreamServerPeerLabels:       make(map[string][]string),
		streamUpstreamServerPeerLabels: make(map[string][]string),
		streamUpstreamServerLabels:     make(map[string][]string),
		nginxClient:                    nginxClient,
		totalMetrics: map[string]*prometheus.Desc{
			"connections_accepted":  newGlobalMetric(namespace, "connections_accepted", "Accepted client connections", constLabels),
			"connections_dropped":   newGlobalMetric(namespace, "connections_dropped", "Dropped client connections", constLabels),
			"connections_active":    newGlobalMetric(namespace, "connections_active", "Active client connections", constLabels),
			"connections_idle":      newGlobalMetric(namespace, "connections_idle", "Idle client connections", constLabels),
			"http_requests_total":   newGlobalMetric(namespace, "http_requests_total", "Total http requests", constLabels),
			"http_requests_current": newGlobalMetric(namespace, "http_requests_current", "Current http requests", constLabels),
			"ssl_handshakes":        newGlobalMetric(namespace, "ssl_handshakes", "Successful SSL handshakes", constLabels),
			"ssl_handshakes_failed": newGlobalMetric(namespace, "ssl_handshakes_failed", "Failed SSL handshakes", constLabels),
			"ssl_session_reuses":    newGlobalMetric(namespace, "ssl_session_reuses", "Session reuses during SSL handshake", constLabels),
		},
		serverZoneMetrics: map[string]*prometheus.Desc{
			"processing":    newServerZoneMetric(namespace, "processing", "Client requests that are currently being processed", variableLabelNames.ServerZoneVariableLabelNames, constLabels),
			"requests":      newServerZoneMetric(namespace, "requests", "Total client requests", variableLabelNames.ServerZoneVariableLabelNames, constLabels),
			"responses_1xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", variableLabelNames.ServerZoneVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "1xx"})),
			"responses_2xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", variableLabelNames.ServerZoneVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "2xx"})),
			"responses_3xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", variableLabelNames.ServerZoneVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "3xx"})),
			"responses_4xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", variableLabelNames.ServerZoneVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "4xx"})),
			"responses_5xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", variableLabelNames.ServerZoneVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "5xx"})),
			"discarded":     newServerZoneMetric(namespace, "discarded", "Requests completed without sending a response", variableLabelNames.ServerZoneVariableLabelNames, constLabels),
			"received":      newServerZoneMetric(namespace, "received", "Bytes received from clients", variableLabelNames.ServerZoneVariableLabelNames, constLabels),
			"sent":          newServerZoneMetric(namespace, "sent", "Bytes sent to clients", variableLabelNames.ServerZoneVariableLabelNames, constLabels),
		},
		streamServerZoneMetrics: map[string]*prometheus.Desc{
			"processing":   newStreamServerZoneMetric(namespace, "processing", "Client connections that are currently being processed", variableLabelNames.StreamServerZoneVariableLabelNames, constLabels),
			"connections":  newStreamServerZoneMetric(namespace, "connections", "Total connections", variableLabelNames.StreamServerZoneVariableLabelNames, constLabels),
			"sessions_2xx": newStreamServerZoneMetric(namespace, "sessions", "Total sessions completed", variableLabelNames.StreamServerZoneVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "2xx"})),
			"sessions_4xx": newStreamServerZoneMetric(namespace, "sessions", "Total sessions completed", variableLabelNames.StreamServerZoneVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "4xx"})),
			"sessions_5xx": newStreamServerZoneMetric(namespace, "sessions", "Total sessions completed", variableLabelNames.StreamServerZoneVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "5xx"})),
			"discarded":    newStreamServerZoneMetric(namespace, "discarded", "Connections completed without creating a session", variableLabelNames.StreamServerZoneVariableLabelNames, constLabels),
			"received":     newStreamServerZoneMetric(namespace, "received", "Bytes received from clients", variableLabelNames.StreamServerZoneVariableLabelNames, constLabels),
			"sent":         newStreamServerZoneMetric(namespace, "sent", "Bytes sent to clients", variableLabelNames.StreamServerZoneVariableLabelNames, constLabels),
		},
		upstreamMetrics: map[string]*prometheus.Desc{
			"keepalives": newUpstreamMetric(namespace, "keepalives", "Idle keepalive connections", constLabels),
			"zombies":    newUpstreamMetric(namespace, "zombies", "Servers removed from the group but still processing active client requests", constLabels),
		},
		streamUpstreamMetrics: map[string]*prometheus.Desc{
			"zombies": newStreamUpstreamMetric(namespace, "zombies", "Servers removed from the group but still processing active client connections", constLabels),
		},
		upstreamServerMetrics: map[string]*prometheus.Desc{
			"state":                   newUpstreamServerMetric(namespace, "state", "Current state", upstreamServerVariableLabelNames, constLabels),
			"active":                  newUpstreamServerMetric(namespace, "active", "Active connections", upstreamServerVariableLabelNames, constLabels),
			"requests":                newUpstreamServerMetric(namespace, "requests", "Total client requests", upstreamServerVariableLabelNames, constLabels),
			"responses_1xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", upstreamServerVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "1xx"})),
			"responses_2xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", upstreamServerVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "2xx"})),
			"responses_3xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", upstreamServerVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "3xx"})),
			"responses_4xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", upstreamServerVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "4xx"})),
			"responses_5xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", upstreamServerVariableLabelNames, MergeLabels(constLabels, prometheus.Labels{"code": "5xx"})),
			"sent":                    newUpstreamServerMetric(namespace, "sent", "Bytes sent to this server", upstreamServerVariableLabelNames, constLabels),
			"received":                newUpstreamServerMetric(namespace, "received", "Bytes received to this server", upstreamServerVariableLabelNames, constLabels),
			"fails":                   newUpstreamServerMetric(namespace, "fails", "Active connections", upstreamServerVariableLabelNames, constLabels),
			"unavail":                 newUpstreamServerMetric(namespace, "unavail", "How many times the server became unavailable for client requests (state 'unavail') due to the number of unsuccessful attempts reaching the max_fails threshold", upstreamServerVariableLabelNames, constLabels),
			"header_time":             newUpstreamServerMetric(namespace, "header_time", "Average time to get the response header from the server", upstreamServerVariableLabelNames, constLabels),
			"response_time":           newUpstreamServerMetric(namespace, "response_time", "Average time to get the full response from the server", upstreamServerVariableLabelNames, constLabels),
			"health_checks_checks":    newUpstreamServerMetric(namespace, "health_checks_checks", "Total health check requests", upstreamServerVariableLabelNames, constLabels),
			"health_checks_fails":     newUpstreamServerMetric(namespace, "health_checks_fails", "Failed health checks", upstreamServerVariableLabelNames, constLabels),
			"health_checks_unhealthy": newUpstreamServerMetric(namespace, "health_checks_unhealthy", "How many times the server became unhealthy (state 'unhealthy')", upstreamServerVariableLabelNames, constLabels),
		},
		streamUpstreamServerMetrics: map[string]*prometheus.Desc{
			"state":                   newStreamUpstreamServerMetric(namespace, "state", "Current state", streamUpstreamServerVariableLabelNames, constLabels),
			"active":                  newStreamUpstreamServerMetric(namespace, "active", "Active connections", streamUpstreamServerVariableLabelNames, constLabels),
			"sent":                    newStreamUpstreamServerMetric(namespace, "sent", "Bytes sent to this server", streamUpstreamServerVariableLabelNames, constLabels),
			"received":                newStreamUpstreamServerMetric(namespace, "received", "Bytes received from this server", streamUpstreamServerVariableLabelNames, constLabels),
			"fails":                   newStreamUpstreamServerMetric(namespace, "fails", "Number of unsuccessful attempts to communicate with the server", streamUpstreamServerVariableLabelNames, constLabels),
			"unavail":                 newStreamUpstreamServerMetric(namespace, "unavail", "How many times the server became unavailable for client connections (state 'unavail') due to the number of unsuccessful attempts reaching the max_fails threshold", streamUpstreamServerVariableLabelNames, constLabels),
			"connections":             newStreamUpstreamServerMetric(namespace, "connections", "Total number of client connections forwarded to this server", streamUpstreamServerVariableLabelNames, constLabels),
			"connect_time":            newStreamUpstreamServerMetric(namespace, "connect_time", "Average time to connect to the upstream server", streamUpstreamServerVariableLabelNames, constLabels),
			"first_byte_time":         newStreamUpstreamServerMetric(namespace, "first_byte_time", "Average time to receive the first byte of data", streamUpstreamServerVariableLabelNames, constLabels),
			"response_time":           newStreamUpstreamServerMetric(namespace, "response_time", "Average time to receive the last byte of data", streamUpstreamServerVariableLabelNames, constLabels),
			"health_checks_checks":    newStreamUpstreamServerMetric(namespace, "health_checks_checks", "Total health check requests", streamUpstreamServerVariableLabelNames, constLabels),
			"health_checks_fails":     newStreamUpstreamServerMetric(namespace, "health_checks_fails", "Failed health checks", streamUpstreamServerVariableLabelNames, constLabels),
			"health_checks_unhealthy": newStreamUpstreamServerMetric(namespace, "health_checks_unhealthy", "How many times the server became unhealthy (state 'unhealthy')", streamUpstreamServerVariableLabelNames, constLabels),
		},
		streamZoneSyncMetrics: map[string]*prometheus.Desc{
			"bytes_in":        newStreamZoneSyncMetric(namespace, "bytes_in", "Bytes received by this node", constLabels),
			"bytes_out":       newStreamZoneSyncMetric(namespace, "bytes_out", "Bytes sent by this node", constLabels),
			"msgs_in":         newStreamZoneSyncMetric(namespace, "msgs_in", "Total messages received by this node", constLabels),
			"msgs_out":        newStreamZoneSyncMetric(namespace, "msgs_out", "Total messages sent by this node", constLabels),
			"nodes_online":    newStreamZoneSyncMetric(namespace, "nodes_online", "Number of peers this node is connected to", constLabels),
			"records_pending": newStreamZoneSyncZoneMetric(namespace, "records_pending", "The number of records that need to be sent to the cluster", constLabels),
			"records_total":   newStreamZoneSyncZoneMetric(namespace, "records_total", "The total number of records stored in the shared memory zone", constLabels),
		},
		locationZoneMetrics: map[string]*prometheus.Desc{
			"requests":      newLocationZoneMetric(namespace, "requests", "Total client requests", constLabels),
			"responses_1xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "1xx"})),
			"responses_2xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "2xx"})),
			"responses_3xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "3xx"})),
			"responses_4xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "4xx"})),
			"responses_5xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "5xx"})),
			"discarded":     newLocationZoneMetric(namespace, "discarded", "Requests completed without sending a response", constLabels),
			"received":      newLocationZoneMetric(namespace, "received", "Bytes received from clients", constLabels),
			"sent":          newLocationZoneMetric(namespace, "sent", "Bytes sent to clients", constLabels),
		},
		resolverMetrics: map[string]*prometheus.Desc{
			"name":     newResolverMetric(namespace, "name", "Total requests to resolve names to addresses", constLabels),
			"srv":      newResolverMetric(namespace, "srv", "Total requests to resolve SRV records", constLabels),
			"addr":     newResolverMetric(namespace, "addr", "Total requests to resolve addresses to names", constLabels),
			"noerror":  newResolverMetric(namespace, "noerror", "Total number of successful responses", constLabels),
			"formerr":  newResolverMetric(namespace, "formerr", "Total number of FORMERR responses", constLabels),
			"servfail": newResolverMetric(namespace, "servfail", "Total number of SERVFAIL responses", constLabels),
			"nxdomain": newResolverMetric(namespace, "nxdomain", "Total number of NXDOMAIN responses", constLabels),
			"notimp":   newResolverMetric(namespace, "notimp", "Total number of NOTIMP responses", constLabels),
			"refused":  newResolverMetric(namespace, "refused", "Total number of REFUSED responses", constLabels),
			"timedout": newResolverMetric(namespace, "timedout", "Total number of timed out requests", constLabels),
			"unknown":  newResolverMetric(namespace, "unknown", "Total requests completed with an unknown error", constLabels),
		},
		upMetric: newUpMetric(namespace, constLabels),
	}
}

// Describe sends the super-set of all possible descriptors of NGINX Plus metrics
// to the provided channel.
func (c *NginxPlusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upMetric.Desc()

	for _, m := range c.totalMetrics {
		ch <- m
	}
	for _, m := range c.serverZoneMetrics {
		ch <- m
	}
	for _, m := range c.upstreamMetrics {
		ch <- m
	}
	for _, m := range c.upstreamServerMetrics {
		ch <- m
	}
	for _, m := range c.streamServerZoneMetrics {
		ch <- m
	}
	for _, m := range c.streamUpstreamMetrics {
		ch <- m
	}
	for _, m := range c.streamUpstreamServerMetrics {
		ch <- m
	}
	for _, m := range c.streamZoneSyncMetrics {
		ch <- m
	}
	for _, m := range c.locationZoneMetrics {
		ch <- m
	}
	for _, m := range c.resolverMetrics {
		ch <- m
	}
}

// Collect fetches metrics from NGINX Plus and sends them to the provided channel.
func (c *NginxPlusCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock() // To protect metrics from concurrent collects
	defer c.mutex.Unlock()

	stats, err := c.nginxClient.GetStats()
	if err != nil {
		c.upMetric.Set(nginxDown)
		ch <- c.upMetric
		log.Printf("Error getting stats: %v", err)
		return
	}

	c.upMetric.Set(nginxUp)
	ch <- c.upMetric

	ch <- prometheus.MustNewConstMetric(c.totalMetrics["connections_accepted"],
		prometheus.CounterValue, float64(stats.Connections.Accepted))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["connections_dropped"],
		prometheus.CounterValue, float64(stats.Connections.Dropped))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["connections_active"],
		prometheus.GaugeValue, float64(stats.Connections.Active))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["connections_idle"],
		prometheus.GaugeValue, float64(stats.Connections.Idle))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["http_requests_total"],
		prometheus.CounterValue, float64(stats.HTTPRequests.Total))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["http_requests_current"],
		prometheus.GaugeValue, float64(stats.HTTPRequests.Current))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["ssl_handshakes"],
		prometheus.CounterValue, float64(stats.SSL.Handshakes))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["ssl_handshakes_failed"],
		prometheus.CounterValue, float64(stats.SSL.HandshakesFailed))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["ssl_session_reuses"],
		prometheus.CounterValue, float64(stats.SSL.SessionReuses))

	for name, zone := range stats.ServerZones {
		labelValues := []string{name}
		varLabelValues := c.getServerZoneLabelValues(name)

		if c.variableLabelNames.ServerZoneVariableLabelNames != nil && len(varLabelValues) != len(c.variableLabelNames.ServerZoneVariableLabelNames) {
			log.Printf("wrong number of labels for http zone %v. For labels %v, got values: %v. Empty labels will be used instead",
				name, c.variableLabelNames.ServerZoneVariableLabelNames, varLabelValues)
			for range c.variableLabelNames.ServerZoneVariableLabelNames {
				labelValues = append(labelValues, "")
			}
		} else {
			labelValues = append(labelValues, varLabelValues...)
		}

		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["processing"],
			prometheus.GaugeValue, float64(zone.Processing), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["requests"],
			prometheus.CounterValue, float64(zone.Requests), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_1xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses1xx), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_2xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses2xx), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_3xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses3xx), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_4xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses4xx), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_5xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses5xx), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["discarded"],
			prometheus.CounterValue, float64(zone.Discarded), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["received"],
			prometheus.CounterValue, float64(zone.Received), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["sent"],
			prometheus.CounterValue, float64(zone.Sent), labelValues...)
	}

	for name, zone := range stats.StreamServerZones {
		labelValues := []string{name}
		varLabelValues := c.getStreamServerZoneLabelValues(name)

		if c.variableLabelNames.StreamServerZoneVariableLabelNames != nil && len(varLabelValues) != len(c.variableLabelNames.StreamServerZoneVariableLabelNames) {
			log.Printf("wrong number of labels for stream server zone %v. For labels %v, got values: %v. Empty labels will be used instead",
				name, c.variableLabelNames.StreamServerZoneVariableLabelNames, varLabelValues)
			for range c.variableLabelNames.StreamServerZoneVariableLabelNames {
				labelValues = append(labelValues, "")
			}
		} else {
			labelValues = append(labelValues, varLabelValues...)
		}
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["processing"],
			prometheus.GaugeValue, float64(zone.Processing), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["connections"],
			prometheus.CounterValue, float64(zone.Connections), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sessions_2xx"],
			prometheus.CounterValue, float64(zone.Sessions.Sessions2xx), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sessions_4xx"],
			prometheus.CounterValue, float64(zone.Sessions.Sessions4xx), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sessions_5xx"],
			prometheus.CounterValue, float64(zone.Sessions.Sessions5xx), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["discarded"],
			prometheus.CounterValue, float64(zone.Discarded), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["received"],
			prometheus.CounterValue, float64(zone.Received), labelValues...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sent"],
			prometheus.CounterValue, float64(zone.Sent), labelValues...)
	}

	for name, upstream := range stats.Upstreams {
		for _, peer := range upstream.Peers {
			labelValues := []string{name, peer.Server}
			varLabelValues := c.getUpstreamServerLabelValues(name)

			if c.variableLabelNames.UpstreamServerVariableLabelNames != nil && len(varLabelValues) != len(c.variableLabelNames.UpstreamServerVariableLabelNames) {
				log.Printf("wrong number of labels for upstream %v. For labels %v, got values: %v. Empty labels will be used instead",
					name, c.variableLabelNames.UpstreamServerVariableLabelNames, varLabelValues)
				for range c.variableLabelNames.UpstreamServerVariableLabelNames {
					labelValues = append(labelValues, "")
				}
			} else {
				labelValues = append(labelValues, varLabelValues...)
			}

			upstreamServer := fmt.Sprintf("%v/%v", name, peer.Server)
			varPeerLabelValues := c.getUpstreamServerPeerLabelValues(upstreamServer)
			if c.variableLabelNames.UpstreamServerPeerVariableLabelNames != nil && len(varPeerLabelValues) != len(c.variableLabelNames.UpstreamServerPeerVariableLabelNames) {
				log.Printf("wrong number of labels for upstream peer %v. For labels %v, got values: %v. Empty labels will be used instead",
					peer.Server, c.variableLabelNames.UpstreamServerPeerVariableLabelNames, varPeerLabelValues)
				for range c.variableLabelNames.UpstreamServerPeerVariableLabelNames {
					labelValues = append(labelValues, "")
				}
			} else {
				labelValues = append(labelValues, varPeerLabelValues...)
			}

			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["state"],
				prometheus.GaugeValue, upstreamServerStates[peer.State], labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["active"],
				prometheus.GaugeValue, float64(peer.Active), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["requests"],
				prometheus.CounterValue, float64(peer.Requests), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_1xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses1xx), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_2xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses2xx), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_3xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses3xx), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_4xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses4xx), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_5xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses5xx), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["sent"],
				prometheus.CounterValue, float64(peer.Sent), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["received"],
				prometheus.CounterValue, float64(peer.Received), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["fails"],
				prometheus.CounterValue, float64(peer.Fails), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["unavail"],
				prometheus.CounterValue, float64(peer.Unavail), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["header_time"],
				prometheus.GaugeValue, float64(peer.HeaderTime), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["response_time"],
				prometheus.GaugeValue, float64(peer.ResponseTime), labelValues...)

			if peer.HealthChecks != (plusclient.HealthChecks{}) {
				ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["health_checks_checks"],
					prometheus.CounterValue, float64(peer.HealthChecks.Checks), labelValues...)
				ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["health_checks_fails"],
					prometheus.CounterValue, float64(peer.HealthChecks.Fails), labelValues...)
				ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["health_checks_unhealthy"],
					prometheus.CounterValue, float64(peer.HealthChecks.Unhealthy), labelValues...)
			}
		}
		ch <- prometheus.MustNewConstMetric(c.upstreamMetrics["keepalives"],
			prometheus.GaugeValue, float64(upstream.Keepalives), name)
		ch <- prometheus.MustNewConstMetric(c.upstreamMetrics["zombies"],
			prometheus.GaugeValue, float64(upstream.Zombies), name)
	}

	for name, upstream := range stats.StreamUpstreams {
		for _, peer := range upstream.Peers {
			labelValues := []string{name, peer.Server}
			varLabelValues := c.getStreamUpstreamServerLabelValues(name)

			if c.variableLabelNames.StreamUpstreamServerVariableLabelNames != nil && len(varLabelValues) != len(c.variableLabelNames.StreamUpstreamServerVariableLabelNames) {
				log.Printf("wrong number of labels for stream server %v. For labels %v, got values: %v. Empty labels will be used instead",
					name, c.variableLabelNames.StreamUpstreamServerVariableLabelNames, varLabelValues)
				for range c.variableLabelNames.StreamUpstreamServerVariableLabelNames {
					labelValues = append(labelValues, "")
				}
			} else {
				labelValues = append(labelValues, varLabelValues...)
			}

			upstreamServer := fmt.Sprintf("%v/%v", name, peer.Server)
			varPeerLabelValues := c.getStreamUpstreamServerPeerLabelValues(upstreamServer)
			if c.variableLabelNames.StreamUpstreamServerPeerVariableLabelNames != nil && len(varPeerLabelValues) != len(c.variableLabelNames.StreamUpstreamServerPeerVariableLabelNames) {
				log.Printf("wrong number of labels for stream upstream peer %v. For labels %v, got values: %v. Empty labels will be used instead",
					peer.Server, c.variableLabelNames.StreamUpstreamServerPeerVariableLabelNames, varPeerLabelValues)
				for range c.variableLabelNames.StreamUpstreamServerPeerVariableLabelNames {
					labelValues = append(labelValues, "")
				}
			} else {
				labelValues = append(labelValues, varPeerLabelValues...)
			}

			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["state"],
				prometheus.GaugeValue, upstreamServerStates[peer.State], labelValues...)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["active"],
				prometheus.GaugeValue, float64(peer.Active), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["connections"],
				prometheus.CounterValue, float64(peer.Connections), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["connect_time"],
				prometheus.GaugeValue, float64(peer.ConnectTime), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["first_byte_time"],
				prometheus.GaugeValue, float64(peer.FirstByteTime), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["response_time"],
				prometheus.GaugeValue, float64(peer.ResponseTime), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["sent"],
				prometheus.CounterValue, float64(peer.Sent), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["received"],
				prometheus.CounterValue, float64(peer.Received), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["fails"],
				prometheus.CounterValue, float64(peer.Fails), labelValues...)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["unavail"],
				prometheus.CounterValue, float64(peer.Unavail), labelValues...)
			if peer.HealthChecks != (plusclient.HealthChecks{}) {
				ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["health_checks_checks"],
					prometheus.CounterValue, float64(peer.HealthChecks.Checks), labelValues...)
				ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["health_checks_fails"],
					prometheus.CounterValue, float64(peer.HealthChecks.Fails), labelValues...)
				ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["health_checks_unhealthy"],
					prometheus.CounterValue, float64(peer.HealthChecks.Unhealthy), labelValues...)
			}
		}
		ch <- prometheus.MustNewConstMetric(c.streamUpstreamMetrics["zombies"],
			prometheus.GaugeValue, float64(upstream.Zombies), name)
	}

	if stats.StreamZoneSync != nil {
		for name, zone := range stats.StreamZoneSync.Zones {
			ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["records_pending"],
				prometheus.GaugeValue, float64(zone.RecordsPending), name)
			ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["records_total"],
				prometheus.GaugeValue, float64(zone.RecordsTotal), name)
		}

		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["bytes_in"],
			prometheus.CounterValue, float64(stats.StreamZoneSync.Status.BytesIn))
		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["bytes_out"],
			prometheus.CounterValue, float64(stats.StreamZoneSync.Status.BytesOut))
		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["msgs_in"],
			prometheus.CounterValue, float64(stats.StreamZoneSync.Status.MsgsIn))
		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["msgs_out"],
			prometheus.CounterValue, float64(stats.StreamZoneSync.Status.MsgsOut))
		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["nodes_online"],
			prometheus.GaugeValue, float64(stats.StreamZoneSync.Status.NodesOnline))
	}

	for name, zone := range stats.LocationZones {
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["requests"],
			prometheus.CounterValue, float64(zone.Requests), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_1xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses1xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_2xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses2xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_3xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses3xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_4xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses4xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_5xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses5xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["discarded"],
			prometheus.CounterValue, float64(zone.Discarded), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["received"],
			prometheus.CounterValue, float64(zone.Received), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["sent"],
			prometheus.CounterValue, float64(zone.Sent), name)
	}

	for name, zone := range stats.Resolvers {
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["name"],
			prometheus.CounterValue, float64(zone.Requests.Name), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["srv"],
			prometheus.CounterValue, float64(zone.Requests.Srv), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["addr"],
			prometheus.CounterValue, float64(zone.Requests.Addr), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["noerror"],
			prometheus.CounterValue, float64(zone.Responses.Noerror), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["formerr"],
			prometheus.CounterValue, float64(zone.Responses.Formerr), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["servfail"],
			prometheus.CounterValue, float64(zone.Responses.Servfail), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["nxdomain"],
			prometheus.CounterValue, float64(zone.Responses.Nxdomain), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["notimp"],
			prometheus.CounterValue, float64(zone.Responses.Notimp), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["refused"],
			prometheus.CounterValue, float64(zone.Responses.Refused), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["timedout"],
			prometheus.CounterValue, float64(zone.Responses.Timedout), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["unknown"],
			prometheus.CounterValue, float64(zone.Responses.Unknown), name)
	}
}

var upstreamServerStates = map[string]float64{
	"up":        1.0,
	"draining":  2.0,
	"down":      3.0,
	"unavail":   4.0,
	"checking":  5.0,
	"unhealthy": 6.0,
}

func newServerZoneMetric(namespace string, metricName string, docString string, variableLabelNames []string, constLabels prometheus.Labels) *prometheus.Desc {
	labels := []string{"server_zone"}
	labels = append(labels, variableLabelNames...)
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "server_zone", metricName), docString, labels, constLabels)
}

func newStreamServerZoneMetric(namespace string, metricName string, docString string, variableLabelNames []string, constLabels prometheus.Labels) *prometheus.Desc {
	labels := []string{"server_zone"}
	labels = append(labels, variableLabelNames...)
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_server_zone", metricName), docString, labels, constLabels)
}

func newUpstreamMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "upstream", metricName), docString, []string{"upstream"}, constLabels)
}

func newStreamUpstreamMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_upstream", metricName), docString, []string{"upstream"}, constLabels)
}

func newUpstreamServerMetric(namespace string, metricName string, docString string, variableLabelNames []string, constLabels prometheus.Labels) *prometheus.Desc {
	labels := []string{"upstream", "server"}
	labels = append(labels, variableLabelNames...)
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "upstream_server", metricName), docString, labels, constLabels)
}

func newStreamUpstreamServerMetric(namespace string, metricName string, docString string, variableLabelNames []string, constLabels prometheus.Labels) *prometheus.Desc {
	labels := []string{"upstream", "server"}
	labels = append(labels, variableLabelNames...)
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_upstream_server", metricName), docString, labels, constLabels)
}

func newStreamZoneSyncMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_zone_sync_status", metricName), docString, nil, constLabels)
}

func newStreamZoneSyncZoneMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_zone_sync_zone", metricName), docString, []string{"zone"}, constLabels)
}

func newLocationZoneMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "location_zone", metricName), docString, []string{"location_zone"}, constLabels)
}

func newResolverMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "resolver", metricName), docString, []string{"resolver"}, constLabels)
}

package collector

import (
	"log"
	"sync"

	plusclient "github.com/nginxinc/nginx-plus-go-client/client"
	"github.com/prometheus/client_golang/prometheus"
)

// NginxPlusCollector collects NGINX Plus metrics. It implements prometheus.Collector interface.
type NginxPlusCollector struct {
	nginxClient                 *plusclient.NginxClient
	totalMetrics                map[string]*prometheus.Desc
	serverZoneMetrics           map[string]*prometheus.Desc
	upstreamMetrics             map[string]*prometheus.Desc
	upstreamServerMetrics       map[string]*prometheus.Desc
	streamServerZoneMetrics     map[string]*prometheus.Desc
	streamUpstreamMetrics       map[string]*prometheus.Desc
	streamUpstreamServerMetrics map[string]*prometheus.Desc
	streamZoneSyncMetrics       map[string]*prometheus.Desc
	upMetric                    prometheus.Gauge
	mutex                       sync.Mutex
}

// NewNginxPlusCollector creates an NginxPlusCollector.
func NewNginxPlusCollector(nginxClient *plusclient.NginxClient, namespace string) *NginxPlusCollector {
	return &NginxPlusCollector{
		nginxClient: nginxClient,
		totalMetrics: map[string]*prometheus.Desc{
			"connections_accepted":  newGlobalMetric(namespace, "connections_accepted", "Accepted client connections"),
			"connections_dropped":   newGlobalMetric(namespace, "connections_dropped", "Dropped client connections"),
			"connections_active":    newGlobalMetric(namespace, "connections_active", "Active client connections"),
			"connections_idle":      newGlobalMetric(namespace, "connections_idle", "Idle client connections"),
			"http_requests_total":   newGlobalMetric(namespace, "http_requests_total", "Total http requests"),
			"http_requests_current": newGlobalMetric(namespace, "http_requests_current", "Current http requests"),
			"ssl_handshakes":        newGlobalMetric(namespace, "ssl_handshakes", "Successful SSL handshakes"),
			"ssl_handshakes_failed": newGlobalMetric(namespace, "ssl_handshakes_failed", "Failed SSL handshakes"),
			"ssl_session_reuses":    newGlobalMetric(namespace, "ssl_session_reuses", "Session reuses during SSL handshake"),
		},
		serverZoneMetrics: map[string]*prometheus.Desc{
			"processing":    newServerZoneMetric(namespace, "processing", "Client requests that are currently being processed", nil),
			"requests":      newServerZoneMetric(namespace, "requests", "Total client requests", nil),
			"responses_1xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "1xx"}),
			"responses_2xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "2xx"}),
			"responses_3xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "3xx"}),
			"responses_4xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "4xx"}),
			"responses_5xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "5xx"}),
			"discarded":     newServerZoneMetric(namespace, "discarded", "Requests completed without sending a response", nil),
			"received":      newServerZoneMetric(namespace, "received", "Bytes received from clients", nil),
			"sent":          newServerZoneMetric(namespace, "sent", "Bytes sent to clients", nil),
		},
		streamServerZoneMetrics: map[string]*prometheus.Desc{
			"processing":   newStreamServerZoneMetric(namespace, "processing", "Client connections that are currently being processed", nil),
			"connections":  newStreamServerZoneMetric(namespace, "connections", "Total connections", nil),
			"sessions_2xx": newStreamServerZoneMetric(namespace, "sessions", "Total sessions completed", prometheus.Labels{"code": "2xx"}),
			"sessions_4xx": newStreamServerZoneMetric(namespace, "sessions", "Total sessions completed", prometheus.Labels{"code": "4xx"}),
			"sessions_5xx": newStreamServerZoneMetric(namespace, "sessions", "Total sessions completed", prometheus.Labels{"code": "5xx"}),
			"discarded":    newStreamServerZoneMetric(namespace, "discarded", "Connections completed without creating a session", nil),
			"received":     newStreamServerZoneMetric(namespace, "received", "Bytes received from clients", nil),
			"sent":         newStreamServerZoneMetric(namespace, "sent", "Bytes sent to clients", nil),
		},
		upstreamMetrics: map[string]*prometheus.Desc{
			"keepalives": newUpstreamMetric(namespace, "keepalives", "Idle keepalive connections"),
			"zombies":    newUpstreamMetric(namespace, "zombies", "Servers removed from the group but still processing active client requests"),
		},
		streamUpstreamMetrics: map[string]*prometheus.Desc{
			"zombies": newStreamUpstreamMetric(namespace, "zombies", "Servers removed from the group but still processing active client connections"),
		},
		upstreamServerMetrics: map[string]*prometheus.Desc{
			"state":                   newUpstreamServerMetric(namespace, "state", "Current state", nil),
			"active":                  newUpstreamServerMetric(namespace, "active", "Active connections", nil),
			"requests":                newUpstreamServerMetric(namespace, "requests", "Total client requests", nil),
			"responses_1xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "1xx"}),
			"responses_2xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "2xx"}),
			"responses_3xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "3xx"}),
			"responses_4xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "4xx"}),
			"responses_5xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", prometheus.Labels{"code": "5xx"}),
			"sent":                    newUpstreamServerMetric(namespace, "sent", "Bytes sent to this server", nil),
			"received":                newUpstreamServerMetric(namespace, "received", "Bytes received to this server", nil),
			"fails":                   newUpstreamServerMetric(namespace, "fails", "Active connections", nil),
			"unavail":                 newUpstreamServerMetric(namespace, "unavail", "How many times the server became unavailable for client requests (state 'unavail') due to the number of unsuccessful attempts reaching the max_fails threshold", nil),
			"header_time":             newUpstreamServerMetric(namespace, "header_time", "Average time to get the response header from the server", nil),
			"response_time":           newUpstreamServerMetric(namespace, "response_time", "Average time to get the full response from the server", nil),
			"health_checks_checks":    newUpstreamServerMetric(namespace, "health_checks_checks", "Total health check requests", nil),
			"health_checks_fails":     newUpstreamServerMetric(namespace, "health_checks_fails", "Failed health checks", nil),
			"health_checks_unhealthy": newUpstreamServerMetric(namespace, "health_checks_unhealthy", "How many times the server became unhealthy (state 'unhealthy')", nil),
		},
		streamUpstreamServerMetrics: map[string]*prometheus.Desc{
			"state":                   newStreamUpstreamServerMetric(namespace, "state", "Current state"),
			"active":                  newStreamUpstreamServerMetric(namespace, "active", "Active connections"),
			"sent":                    newStreamUpstreamServerMetric(namespace, "sent", "Bytes sent to this server"),
			"received":                newStreamUpstreamServerMetric(namespace, "received", "Bytes received from this server"),
			"fails":                   newStreamUpstreamServerMetric(namespace, "fails", "Number of unsuccessful attempts to communicate with the server"),
			"unavail":                 newStreamUpstreamServerMetric(namespace, "unavail", "How many times the server became unavailable for client connections (state 'unavail') due to the number of unsuccessful attempts reaching the max_fails threshold"),
			"connections":             newStreamUpstreamServerMetric(namespace, "connections", "Total number of client connections forwarded to this server"),
			"connect_time":            newStreamUpstreamServerMetric(namespace, "connect_time", "Average time to connect to the upstream server"),
			"first_byte_time":         newStreamUpstreamServerMetric(namespace, "first_byte_time", "Average time to receive the first byte of data"),
			"response_time":           newStreamUpstreamServerMetric(namespace, "response_time", "Average time to receive the last byte of data"),
			"health_checks_checks":    newStreamUpstreamServerMetric(namespace, "health_checks_checks", "Total health check requests"),
			"health_checks_fails":     newStreamUpstreamServerMetric(namespace, "health_checks_fails", "Failed health checks"),
			"health_checks_unhealthy": newStreamUpstreamServerMetric(namespace, "health_checks_unhealthy", "How many times the server became unhealthy (state 'unhealthy')"),
		},
		streamZoneSyncMetrics: map[string]*prometheus.Desc{
			"bytes_in":        newStreamZoneSyncMetric(namespace, "bytes_in", "Bytes received by this node"),
			"bytes_out":       newStreamZoneSyncMetric(namespace, "bytes_out", "Bytes sent by this node"),
			"msgs_in":         newStreamZoneSyncMetric(namespace, "msgs_in", "Total messages received by this node"),
			"msgs_out":        newStreamZoneSyncMetric(namespace, "msgs_out", "Total messages sent by this node"),
			"nodes_online":    newStreamZoneSyncMetric(namespace, "nodes_online", "Number of peers this node is conected to"),
			"records_pending": newStreamZoneSyncZoneMetric(namespace, "records_pending", "The number of records that need to be sent to the cluster"),
			"records_total":   newStreamZoneSyncZoneMetric(namespace, "records_total", "The total number of records stored in the shared memory zone"),
		},
		upMetric: newUpMetric(namespace),
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
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["processing"],
			prometheus.GaugeValue, float64(zone.Processing), name)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["requests"],
			prometheus.CounterValue, float64(zone.Requests), name)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_1xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses1xx), name)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_2xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses2xx), name)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_3xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses3xx), name)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_4xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses4xx), name)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_5xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses5xx), name)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["discarded"],
			prometheus.CounterValue, float64(zone.Discarded), name)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["received"],
			prometheus.CounterValue, float64(zone.Received), name)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["sent"],
			prometheus.CounterValue, float64(zone.Sent), name)
	}

	for name, zone := range stats.StreamServerZones {
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["processing"],
			prometheus.GaugeValue, float64(zone.Processing), name)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["connections"],
			prometheus.CounterValue, float64(zone.Connections), name)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sessions_2xx"],
			prometheus.CounterValue, float64(zone.Sessions.Sessions2xx), name)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sessions_4xx"],
			prometheus.CounterValue, float64(zone.Sessions.Sessions4xx), name)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sessions_5xx"],
			prometheus.CounterValue, float64(zone.Sessions.Sessions5xx), name)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["discarded"],
			prometheus.CounterValue, float64(zone.Discarded), name)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["received"],
			prometheus.CounterValue, float64(zone.Received), name)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sent"],
			prometheus.CounterValue, float64(zone.Sent), name)
	}

	for name, upstream := range stats.Upstreams {
		for _, peer := range upstream.Peers {
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["state"],
				prometheus.GaugeValue, upstreamServerStates[peer.State], name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["active"],
				prometheus.GaugeValue, float64(peer.Active), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["requests"],
				prometheus.CounterValue, float64(peer.Requests), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_1xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses1xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_2xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses2xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_3xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses3xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_4xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses4xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_5xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses5xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["sent"],
				prometheus.CounterValue, float64(peer.Sent), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["received"],
				prometheus.CounterValue, float64(peer.Received), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["fails"],
				prometheus.CounterValue, float64(peer.Fails), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["unavail"],
				prometheus.CounterValue, float64(peer.Unavail), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["header_time"],
				prometheus.GaugeValue, float64(peer.HeaderTime), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["response_time"],
				prometheus.GaugeValue, float64(peer.ResponseTime), name, peer.Server)

			if peer.HealthChecks != (plusclient.HealthChecks{}) {
				ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["health_checks_checks"],
					prometheus.CounterValue, float64(peer.HealthChecks.Checks), name, peer.Server)
				ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["health_checks_fails"],
					prometheus.CounterValue, float64(peer.HealthChecks.Fails), name, peer.Server)
				ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["health_checks_unhealthy"],
					prometheus.CounterValue, float64(peer.HealthChecks.Unhealthy), name, peer.Server)
			}
		}
		ch <- prometheus.MustNewConstMetric(c.upstreamMetrics["keepalives"],
			prometheus.GaugeValue, float64(upstream.Keepalives), name)
		ch <- prometheus.MustNewConstMetric(c.upstreamMetrics["zombies"],
			prometheus.GaugeValue, float64(upstream.Zombies), name)
	}

	for name, upstream := range stats.StreamUpstreams {
		for _, peer := range upstream.Peers {
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["state"],
				prometheus.GaugeValue, upstreamServerStates[peer.State], name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["active"],
				prometheus.GaugeValue, float64(peer.Active), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["connections"],
				prometheus.CounterValue, float64(peer.Connections), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["connect_time"],
				prometheus.GaugeValue, float64(peer.ConnectTime), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["first_byte_time"],
				prometheus.GaugeValue, float64(peer.FirstByteTime), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["response_time"],
				prometheus.GaugeValue, float64(peer.ResponseTime), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["sent"],
				prometheus.CounterValue, float64(peer.Sent), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["received"],
				prometheus.CounterValue, float64(peer.Received), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["fails"],
				prometheus.CounterValue, float64(peer.Fails), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["unavail"],
				prometheus.CounterValue, float64(peer.Unavail), name, peer.Server)
			if peer.HealthChecks != (plusclient.HealthChecks{}) {
				ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["health_checks_checks"],
					prometheus.CounterValue, float64(peer.HealthChecks.Checks), name, peer.Server)
				ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["health_checks_fails"],
					prometheus.CounterValue, float64(peer.HealthChecks.Fails), name, peer.Server)
				ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["health_checks_unhealthy"],
					prometheus.CounterValue, float64(peer.HealthChecks.Unhealthy), name, peer.Server)
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
}

var upstreamServerStates = map[string]float64{
	"up":        1.0,
	"draining":  2.0,
	"down":      3.0,
	"unavail":   4.0,
	"checking":  5.0,
	"unhealthy": 6.0,
}

func newServerZoneMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "server_zone", metricName), docString, []string{"server_zone"}, constLabels)
}

func newStreamServerZoneMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_server_zone", metricName), docString, []string{"server_zone"}, constLabels)
}

func newUpstreamMetric(namespace string, metricName string, docString string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "upstream", metricName), docString, []string{"upstream"}, nil)
}

func newStreamUpstreamMetric(namespace string, metricName string, docString string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_upstream", metricName), docString, []string{"upstream"}, nil)
}

func newUpstreamServerMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "upstream_server", metricName), docString, []string{"upstream", "server"}, constLabels)
}

func newStreamUpstreamServerMetric(namespace string, metricName string, docString string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_upstream_server", metricName), docString, []string{"upstream", "server"}, nil)
}

func newStreamZoneSyncMetric(namespace string, metricName string, docString string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_zone_sync_status", metricName), docString, nil, nil)
}

func newStreamZoneSyncZoneMetric(namespace string, metricName string, docString string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_zone_sync_zone", metricName), docString, []string{"zone"}, nil)
}

package configs

import (
	"fmt"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
)

const nginxNonExistingUnixSocket = "unix:/var/lib/nginx/non-existing-unix-socket.sock"

// TransportServerEx holds a TransportServer along with the resources referenced by it.
type TransportServerEx struct {
	TransportServer *conf_v1alpha1.TransportServer
	Endpoints       map[string][]string
	PodsByIP        map[string]string
}

func (tsEx *TransportServerEx) String() string {
	if tsEx == nil {
		return "<nil>"
	}

	if tsEx.TransportServer == nil {
		return "TransportServerEx has no TransportServer"
	}

	return fmt.Sprintf("%s/%s", tsEx.TransportServer.Namespace, tsEx.TransportServer.Name)
}

// generateTransportServerConfig generates a full configuration for a TransportServer.
func generateTransportServerConfig(transportServerEx *TransportServerEx, listenerPort int, isPlus bool) version2.TransportServerConfig {
	upstreamNamer := newUpstreamNamerForTransportServer(transportServerEx.TransportServer)

	upstreams := generateStreamUpstreams(transportServerEx, upstreamNamer, isPlus)

	healthCheck := generateTransportServerHealthCheck(transportServerEx.TransportServer.Spec.Action.Pass,
		transportServerEx.TransportServer.Spec.Upstreams)
	var proxyRequests, proxyResponses *int
	var connectTimeout, nextUpstreamTimeout string
	var nextUpstream bool
	var nextUpstreamTries int
	if transportServerEx.TransportServer.Spec.UpstreamParameters != nil {
		proxyRequests = transportServerEx.TransportServer.Spec.UpstreamParameters.UDPRequests
		proxyResponses = transportServerEx.TransportServer.Spec.UpstreamParameters.UDPResponses

		nextUpstream = transportServerEx.TransportServer.Spec.UpstreamParameters.NextUpstream
		if nextUpstream {
			nextUpstreamTries = transportServerEx.TransportServer.Spec.UpstreamParameters.NextUpstreamTries
			nextUpstreamTimeout = transportServerEx.TransportServer.Spec.UpstreamParameters.NextUpstreamTimeout
		}

		connectTimeout = transportServerEx.TransportServer.Spec.UpstreamParameters.ConnectTimeout
	}

	var proxyTimeout string
	if transportServerEx.TransportServer.Spec.SessionParameters != nil {
		proxyTimeout = transportServerEx.TransportServer.Spec.SessionParameters.Timeout
	}

	statusZone := ""
	if transportServerEx.TransportServer.Spec.Listener.Name == conf_v1alpha1.TLSPassthroughListenerName {
		statusZone = transportServerEx.TransportServer.Spec.Host
	} else {
		statusZone = transportServerEx.TransportServer.Spec.Listener.Name
	}

	return version2.TransportServerConfig{
		Server: version2.StreamServer{
			TLSPassthrough:           transportServerEx.TransportServer.Spec.Listener.Name == conf_v1alpha1.TLSPassthroughListenerName,
			UnixSocket:               generateUnixSocket(transportServerEx),
			Port:                     listenerPort,
			UDP:                      transportServerEx.TransportServer.Spec.Listener.Protocol == "UDP",
			StatusZone:               statusZone,
			ProxyRequests:            proxyRequests,
			ProxyResponses:           proxyResponses,
			ProxyPass:                upstreamNamer.GetNameForUpstream(transportServerEx.TransportServer.Spec.Action.Pass),
			Name:                     transportServerEx.TransportServer.Name,
			Namespace:                transportServerEx.TransportServer.Namespace,
			ProxyConnectTimeout:      generateTime(connectTimeout, "60s"),
			ProxyTimeout:             generateTime(proxyTimeout, "10m"),
			ProxyNextUpstream:        nextUpstream,
			ProxyNextUpstreamTimeout: generateTime(nextUpstreamTimeout, "0"),
			ProxyNextUpstreamTries:   nextUpstreamTries,
			HealthCheck:              healthCheck,
		},
		Upstreams: upstreams,
	}
}

func generateUnixSocket(transportServerEx *TransportServerEx) string {
	if transportServerEx.TransportServer.Spec.Listener.Name == conf_v1alpha1.TLSPassthroughListenerName {
		return fmt.Sprintf("unix:/var/lib/nginx/passthrough-%s_%s.sock", transportServerEx.TransportServer.Namespace, transportServerEx.TransportServer.Name)
	}

	return ""
}

func generateStreamUpstreams(transportServerEx *TransportServerEx, upstreamNamer *upstreamNamer, isPlus bool) []version2.StreamUpstream {
	var upstreams []version2.StreamUpstream

	for _, u := range transportServerEx.TransportServer.Spec.Upstreams {

		// subselector is not supported yet in TransportServer upstreams. That's why we pass "nil" here
		endpointsKey := GenerateEndpointsKey(transportServerEx.TransportServer.Namespace, u.Service, nil, uint16(u.Port))
		endpoints := transportServerEx.Endpoints[endpointsKey]

		ups := generateStreamUpstream(&u, upstreamNamer, endpoints, isPlus)

		ups.UpstreamLabels.Service = u.Service
		ups.UpstreamLabels.ResourceType = "transportserver"
		ups.UpstreamLabels.ResourceName = transportServerEx.TransportServer.Name
		ups.UpstreamLabels.ResourceNamespace = transportServerEx.TransportServer.Namespace

		upstreams = append(upstreams, ups)
	}

	return upstreams
}

func generateTransportServerHealthCheck(upstreamHealthCheckName string, upstreams []conf_v1alpha1.Upstream) *version2.StreamHealthCheck {
	var hc *version2.StreamHealthCheck
	for _, u := range upstreams {
		if u.Name == upstreamHealthCheckName {
			if u.HealthCheck == nil || !u.HealthCheck.Enabled {
				return nil
			}
			hc = generateTransportServerHealthCheckWithDefaults(u)

			hc.Enabled = u.HealthCheck.Enabled
			hc.Interval = generateTime(u.HealthCheck.Interval, hc.Interval)
			hc.Jitter = generateTime(u.HealthCheck.Jitter, hc.Jitter)
			hc.Timeout = generateTime(u.HealthCheck.Timeout, hc.Timeout)

			if u.HealthCheck.Fails > 0 {
				hc.Fails = u.HealthCheck.Fails
			}

			if u.HealthCheck.Passes > 0 {
				hc.Passes = u.HealthCheck.Passes
			}

			if u.HealthCheck.Port > 0 {
				hc.Port = u.HealthCheck.Port
			}
		}
	}
	return hc
}

func generateTransportServerHealthCheckWithDefaults(up conf_v1alpha1.Upstream) *version2.StreamHealthCheck {
	return &version2.StreamHealthCheck{
		Enabled:  false,
		Timeout:  "5s",
		Jitter:   "0s",
		Port:     up.Port,
		Interval: "5s",
		Passes:   1,
		Fails:    1,
	}
}

func generateStreamUpstream(upstream *conf_v1alpha1.Upstream, upstreamNamer *upstreamNamer, endpoints []string, isPlus bool) version2.StreamUpstream {
	var upsServers []version2.StreamUpstreamServer

	name := upstreamNamer.GetNameForUpstream(upstream.Name)
	maxFails := generateIntFromPointer(upstream.MaxFails, 1)
	failTimeout := generateTime(upstream.FailTimeout, "10s")

	for _, e := range endpoints {
		s := version2.StreamUpstreamServer{
			Address:     e,
			MaxFails:    maxFails,
			FailTimeout: failTimeout,
		}

		upsServers = append(upsServers, s)
	}

	if !isPlus && len(endpoints) == 0 {
		upsServers = append(upsServers, version2.StreamUpstreamServer{
			Address:     nginxNonExistingUnixSocket,
			MaxFails:    maxFails,
			FailTimeout: failTimeout,
		})
	}

	return version2.StreamUpstream{
		Name:    name,
		Servers: upsServers,
	}
}

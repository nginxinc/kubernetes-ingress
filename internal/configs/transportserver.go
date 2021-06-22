package configs

import (
	"fmt"
	"strings"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
)

const nginxNonExistingUnixSocket = "unix:/var/lib/nginx/non-existing-unix-socket.sock"

// TransportServerEx holds a TransportServer along with the resources referenced by it.
type TransportServerEx struct {
	ListenerPort    int
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
func generateTransportServerConfig(transportServerEx *TransportServerEx, listenerPort int, isPlus bool) *version2.TransportServerConfig {
	upstreamNamer := newUpstreamNamerForTransportServer(transportServerEx.TransportServer)

	upstreams := generateStreamUpstreams(transportServerEx, upstreamNamer, isPlus)

	healthCheck, match := generateTransportServerHealthCheck(transportServerEx.TransportServer.Spec.Action.Pass,
		upstreamNamer.GetNameForUpstream(transportServerEx.TransportServer.Spec.Action.Pass),
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

	serverSnippets := generateSnippets(true, transportServerEx.TransportServer.Spec.ServerSnippets, []string{})

	streamSnippets := generateSnippets(true, transportServerEx.TransportServer.Spec.StreamSnippets, []string{})

	statusZone := transportServerEx.TransportServer.Spec.Listener.Name
	if transportServerEx.TransportServer.Spec.Listener.Name == conf_v1alpha1.TLSPassthroughListenerName {
		statusZone = transportServerEx.TransportServer.Spec.Host
	}

	tsConfig := &version2.TransportServerConfig{
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
			ProxyConnectTimeout:      generateTimeWithDefault(connectTimeout, "60s"),
			ProxyTimeout:             generateTimeWithDefault(proxyTimeout, "10m"),
			ProxyNextUpstream:        nextUpstream,
			ProxyNextUpstreamTimeout: generateTimeWithDefault(nextUpstreamTimeout, "0s"),
			ProxyNextUpstreamTries:   nextUpstreamTries,
			HealthCheck:              healthCheck,
			ServerSnippets:           serverSnippets,
		},
		Match:          match,
		Upstreams:      upstreams,
		StreamSnippets: streamSnippets,
	}

	return tsConfig
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

		ups := generateStreamUpstream(u, upstreamNamer, endpoints, isPlus)

		ups.UpstreamLabels.Service = u.Service
		ups.UpstreamLabels.ResourceType = "transportserver"
		ups.UpstreamLabels.ResourceName = transportServerEx.TransportServer.Name
		ups.UpstreamLabels.ResourceNamespace = transportServerEx.TransportServer.Namespace

		upstreams = append(upstreams, ups)
	}

	return upstreams
}

func generateTransportServerHealthCheck(upstreamName string, generatedUpstreamName string, upstreams []conf_v1alpha1.Upstream) (*version2.StreamHealthCheck, *version2.Match) {
	var hc *version2.StreamHealthCheck
	var match *version2.Match

	for _, u := range upstreams {
		if u.Name == upstreamName {
			if u.HealthCheck == nil || !u.HealthCheck.Enabled {
				return nil, nil
			}
			hc = generateTransportServerHealthCheckWithDefaults(u)

			hc.Enabled = u.HealthCheck.Enabled
			hc.Interval = generateTimeWithDefault(u.HealthCheck.Interval, hc.Interval)
			hc.Jitter = generateTimeWithDefault(u.HealthCheck.Jitter, hc.Jitter)
			hc.Timeout = generateTimeWithDefault(u.HealthCheck.Timeout, hc.Timeout)

			if u.HealthCheck.Fails > 0 {
				hc.Fails = u.HealthCheck.Fails
			}

			if u.HealthCheck.Passes > 0 {
				hc.Passes = u.HealthCheck.Passes
			}

			if u.HealthCheck.Port > 0 {
				hc.Port = u.HealthCheck.Port
			}

			if u.HealthCheck.Match != nil {
				name := "match_" + generatedUpstreamName
				match = generateHealthCheckMatch(u.HealthCheck.Match, name)
				hc.Match = name
			}

			break
		}
	}

	return hc, match
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
		Match:    "",
	}
}

func generateHealthCheckMatch(match *conf_v1alpha1.Match, name string) *version2.Match {
	var modifier string
	var expect string

	if strings.HasPrefix(match.Expect, "~*") {
		modifier = "~*"
		expect = strings.TrimPrefix(match.Expect, "~*")
	} else if strings.HasPrefix(match.Expect, "~") {
		modifier = "~"
		expect = strings.TrimPrefix(match.Expect, "~")
	} else {
		expect = match.Expect
	}

	return &version2.Match{
		Name:                name,
		Send:                match.Send,
		ExpectRegexModifier: modifier,
		Expect:              expect,
	}
}

func generateStreamUpstream(upstream conf_v1alpha1.Upstream, upstreamNamer *upstreamNamer, endpoints []string, isPlus bool) version2.StreamUpstream {
	var upsServers []version2.StreamUpstreamServer

	name := upstreamNamer.GetNameForUpstream(upstream.Name)
	maxFails := generateIntFromPointer(upstream.MaxFails, 1)
	maxConns := generateIntFromPointer(upstream.MaxConns, 0)
	failTimeout := generateTimeWithDefault(upstream.FailTimeout, "10s")

	for _, e := range endpoints {
		s := version2.StreamUpstreamServer{
			Address:        e,
			MaxFails:       maxFails,
			FailTimeout:    failTimeout,
			MaxConnections: maxConns,
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
		Name:                name,
		Servers:             upsServers,
		LoadBalancingMethod: generateLoadBalancingMethod(upstream.LoadBalancingMethod),
	}
}

func generateLoadBalancingMethod(method string) string {
	if method == "" {
		// By default, if unspecified, Nginx uses the 'round_robin' load balancing method.
		// We override this default which suits the Ingress Controller better.
		return "random two least_conn"
	}
	if method == "round_robin" {
		// By default, Nginx uses round robin. We select this method by not specifying any method.
		return ""
	}
	return method
}

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

	var proxyRequests, proxyResponses *int
	if transportServerEx.TransportServer.Spec.UpstreamParameters != nil {
		proxyRequests = transportServerEx.TransportServer.Spec.UpstreamParameters.UDPRequests
		proxyResponses = transportServerEx.TransportServer.Spec.UpstreamParameters.UDPResponses
	}

	return version2.TransportServerConfig{
		Server: version2.StreamServer{
			TLSPassthrough: transportServerEx.TransportServer.Spec.Listener.Name == conf_v1alpha1.TLSPassthroughListenerName,
			UnixSocket:     generateUnixSocket(transportServerEx),
			Port:           listenerPort,
			UDP:            transportServerEx.TransportServer.Spec.Listener.Protocol == "UDP",
			StatusZone:     transportServerEx.TransportServer.Spec.Listener.Name,
			ProxyRequests:  proxyRequests,
			ProxyResponses: proxyResponses,
			ProxyPass:      upstreamNamer.GetNameForUpstream(transportServerEx.TransportServer.Spec.Action.Pass),
			Name:           transportServerEx.TransportServer.Name,
			Namespace:      transportServerEx.TransportServer.Namespace,
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
		name := upstreamNamer.GetNameForUpstream(u.Name)

		// subselector is not supported yet in TransportServer upstreams. That's why we pass "nil" here
		endpointsKey := GenerateEndpointsKey(transportServerEx.TransportServer.Namespace, u.Service, nil, uint16(u.Port))
		endpoints := transportServerEx.Endpoints[endpointsKey]

		ups := generateStreamUpstream(name, endpoints, isPlus)

		upstreams = append(upstreams, ups)
	}

	return upstreams
}

func generateStreamUpstream(upstreamName string, endpoints []string, isPlus bool) version2.StreamUpstream {
	var upsServers []version2.StreamUpstreamServer

	for _, e := range endpoints {
		s := version2.StreamUpstreamServer{
			Address: e,
		}

		upsServers = append(upsServers, s)
	}

	if !isPlus && len(endpoints) == 0 {
		upsServers = append(upsServers, version2.StreamUpstreamServer{
			Address: nginxNonExistingUnixSocket,
		})
	}

	return version2.StreamUpstream{
		Name:    upstreamName,
		Servers: upsServers,
	}
}

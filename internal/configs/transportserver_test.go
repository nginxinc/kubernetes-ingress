package configs

import (
	"reflect"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpstreamNamerForTransportServer(t *testing.T) {
	transportServer := conf_v1alpha1.TransportServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tcp-app",
			Namespace: "default",
		},
	}
	upstreamNamer := newUpstreamNamerForTransportServer(&transportServer)
	upstream := "test"

	expected := "ts_default_tcp-app_test"

	result := upstreamNamer.GetNameForUpstream(upstream)
	if result != expected {
		t.Errorf("GetNameForUpstream() returned %s but expected %v", result, expected)
	}
}

func TestTransportServerExString(t *testing.T) {
	tests := []struct {
		input    *TransportServerEx
		expected string
	}{
		{
			input: &TransportServerEx{
				TransportServer: &conf_v1alpha1.TransportServer{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "test-server",
						Namespace: "default",
					},
				},
			},
			expected: "default/test-server",
		},
		{
			input:    &TransportServerEx{},
			expected: "TransportServerEx has no TransportServer",
		},
		{
			input:    nil,
			expected: "<nil>",
		},
	}

	for _, test := range tests {
		result := test.input.String()
		if result != test.expected {
			t.Errorf("TransportServerEx.String() returned %v but expected %v", result, test.expected)
		}
	}
}

func TestGenerateTransportServerConfigForTCP(t *testing.T) {
	transportServerEx := TransportServerEx{
		TransportServer: &conf_v1alpha1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "tcp-server",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.TransportServerSpec{
				Listener: conf_v1alpha1.TransportServerListener{
					Name:     "tcp-listener",
					Protocol: "TCP",
				},
				Upstreams: []conf_v1alpha1.Upstream{
					{
						Name:    "tcp-app",
						Service: "tcp-app-svc",
						Port:    5001,
					},
				},
				Action: &conf_v1alpha1.Action{
					Pass: "tcp-app",
				},
			},
		},
		Endpoints: map[string][]string{
			"default/tcp-app-svc:5001": {
				"10.0.0.20:5001",
			},
		},
	}

	listenerPort := 2020

	expected := version2.TransportServerConfig{
		Upstreams: []version2.StreamUpstream{
			{
				Name: "ts_default_tcp-server_tcp-app",
				Servers: []version2.StreamUpstreamServer{
					{
						Address: "10.0.0.20:5001",
					},
				},
			},
		},
		Server: version2.StreamServer{
			Port:       2020,
			UDP:        false,
			StatusZone: "tcp-listener",
			ProxyPass:  "ts_default_tcp-server_tcp-app",
			Name:       "tcp-server",
			Namespace:  "default",
		},
	}

	isPlus := true
	result := generateTransportServerConfig(&transportServerEx, listenerPort, isPlus)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateTransportServerConfig() returned \n%+v but expected \n%+v", result, expected)
	}
}

func TestGenerateTransportServerConfigForUDP(t *testing.T) {
	udpRequests := 1
	udpResponses := 5

	transportServerEx := TransportServerEx{
		TransportServer: &conf_v1alpha1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "udp-server",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.TransportServerSpec{
				Listener: conf_v1alpha1.TransportServerListener{
					Name:     "udp-listener",
					Protocol: "UDP",
				},
				Upstreams: []conf_v1alpha1.Upstream{
					{
						Name:    "udp-app",
						Service: "udp-app-svc",
						Port:    5001,
					},
				},
				UpstreamParameters: &conf_v1alpha1.UpstreamParameters{
					UDPRequests:  &udpRequests,
					UDPResponses: &udpResponses,
				},
				Action: &conf_v1alpha1.Action{
					Pass: "udp-app",
				},
			},
		},
		Endpoints: map[string][]string{
			"default/udp-app-svc:5001": {
				"10.0.0.20:5001",
			},
		},
	}

	listenerPort := 2020

	expected := version2.TransportServerConfig{
		Upstreams: []version2.StreamUpstream{
			{
				Name: "ts_default_udp-server_udp-app",
				Servers: []version2.StreamUpstreamServer{
					{
						Address: "10.0.0.20:5001",
					},
				},
			},
		},
		Server: version2.StreamServer{
			Port:           2020,
			UDP:            true,
			StatusZone:     "udp-listener",
			ProxyRequests:  &udpRequests,
			ProxyResponses: &udpResponses,
			ProxyPass:      "ts_default_udp-server_udp-app",
			Name:           "udp-server",
			Namespace:      "default",
		},
	}

	isPlus := true
	result := generateTransportServerConfig(&transportServerEx, listenerPort, isPlus)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateTransportServerConfig() returned \n%+v but expected \n%+v", result, expected)
	}
}

func TestGenerateUnixSocket(t *testing.T) {
	transportServerEx := &TransportServerEx{
		TransportServer: &conf_v1alpha1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "tcp-server",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.TransportServerSpec{
				Listener: conf_v1alpha1.TransportServerListener{
					Name: "tls-passthrough",
				},
			},
		},
	}

	expected := "unix:/var/lib/nginx/passthrough-default_tcp-server.sock"

	result := generateUnixSocket(transportServerEx)
	if result != expected {
		t.Errorf("generateUnixSocket() returned %q but expected %q", result, expected)
	}

	transportServerEx.TransportServer.Spec.Listener.Name = "some-listener"
	expected = ""

	result = generateUnixSocket(transportServerEx)
	if result != expected {
		t.Errorf("generateUnixSocket() returned %q but expected %q", result, expected)
	}
}

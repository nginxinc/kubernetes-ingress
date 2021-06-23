package configs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestGenerateTransportServerConfigForTCPSnippets(t *testing.T) {
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
				ServerSnippets: "deny  192.168.1.1;\nallow 192.168.1.0/24;",
				StreamSnippets: "limit_conn_zone $binary_remote_addr zone=addr:10m;",
			},
		},
		Endpoints: map[string][]string{
			"default/tcp-app-svc:5001": {
				"10.0.0.20:5001",
			},
		},
	}

	listenerPort := 2020

	expected := &version2.TransportServerConfig{
		Upstreams: []version2.StreamUpstream{
			{
				Name: "ts_default_tcp-server_tcp-app",
				Servers: []version2.StreamUpstreamServer{
					{
						Address:     "10.0.0.20:5001",
						MaxFails:    1,
						FailTimeout: "10s",
					},
				},
				UpstreamLabels: version2.UpstreamLabels{
					ResourceName:      "tcp-server",
					ResourceType:      "transportserver",
					ResourceNamespace: "default",
					Service:           "tcp-app-svc",
				},
				LoadBalancingMethod: "random two least_conn",
			},
		},
		Server: version2.StreamServer{
			Port:                     listenerPort,
			UDP:                      false,
			StatusZone:               "tcp-listener",
			ProxyPass:                "ts_default_tcp-server_tcp-app",
			Name:                     "tcp-server",
			Namespace:                "default",
			ProxyConnectTimeout:      "60s",
			ProxyNextUpstream:        false,
			ProxyNextUpstreamTries:   0,
			ProxyNextUpstreamTimeout: "0s",
			ProxyTimeout:             "10m",
			HealthCheck:              nil,
			ServerSnippets:           []string{"deny  192.168.1.1;", "allow 192.168.1.0/24;"},
		},
		StreamSnippets: []string{"limit_conn_zone $binary_remote_addr zone=addr:10m;"},
	}

	result := generateTransportServerConfig(&transportServerEx, listenerPort, true)
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateTransportServerConfig() mismatch (-want +got):\n%s", diff)
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
						Name:        "tcp-app",
						Service:     "tcp-app-svc",
						Port:        5001,
						MaxFails:    intPointer(3),
						FailTimeout: "40s",
					},
				},
				UpstreamParameters: &conf_v1alpha1.UpstreamParameters{
					ConnectTimeout: "30s",
					NextUpstream:   false,
				},
				SessionParameters: &conf_v1alpha1.SessionParameters{
					Timeout: "50s",
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

	expected := &version2.TransportServerConfig{
		Upstreams: []version2.StreamUpstream{
			{
				Name: "ts_default_tcp-server_tcp-app",
				Servers: []version2.StreamUpstreamServer{
					{
						Address:     "10.0.0.20:5001",
						MaxFails:    3,
						FailTimeout: "40s",
					},
				},
				UpstreamLabels: version2.UpstreamLabels{
					ResourceName:      "tcp-server",
					ResourceType:      "transportserver",
					ResourceNamespace: "default",
					Service:           "tcp-app-svc",
				},
				LoadBalancingMethod: "random two least_conn",
			},
		},
		Server: version2.StreamServer{
			Port:                     2020,
			UDP:                      false,
			StatusZone:               "tcp-listener",
			ProxyPass:                "ts_default_tcp-server_tcp-app",
			Name:                     "tcp-server",
			Namespace:                "default",
			ProxyConnectTimeout:      "30s",
			ProxyNextUpstream:        false,
			ProxyNextUpstreamTries:   0,
			ProxyNextUpstreamTimeout: "0s",
			ProxyTimeout:             "50s",
			HealthCheck:              nil,
			ServerSnippets:           []string{},
		},
		StreamSnippets: []string{},
	}

	result := generateTransportServerConfig(&transportServerEx, listenerPort, true)
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateTransportServerConfig() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateTransportServerConfigForTCPMaxConnections(t *testing.T) {
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
						Name:        "tcp-app",
						Service:     "tcp-app-svc",
						Port:        5001,
						MaxFails:    intPointer(3),
						MaxConns:    intPointer(3),
						FailTimeout: "40s",
					},
				},
				UpstreamParameters: &conf_v1alpha1.UpstreamParameters{
					ConnectTimeout: "30s",
					NextUpstream:   false,
				},
				SessionParameters: &conf_v1alpha1.SessionParameters{
					Timeout: "50s",
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

	expected := &version2.TransportServerConfig{
		Upstreams: []version2.StreamUpstream{
			{
				Name: "ts_default_tcp-server_tcp-app",
				Servers: []version2.StreamUpstreamServer{
					{
						Address:        "10.0.0.20:5001",
						MaxFails:       3,
						FailTimeout:    "40s",
						MaxConnections: 3,
					},
				},
				UpstreamLabels: version2.UpstreamLabels{
					ResourceName:      "tcp-server",
					ResourceType:      "transportserver",
					ResourceNamespace: "default",
					Service:           "tcp-app-svc",
				},
				LoadBalancingMethod: "random two least_conn",
			},
		},
		Server: version2.StreamServer{
			Port:                     2020,
			UDP:                      false,
			StatusZone:               "tcp-listener",
			ProxyPass:                "ts_default_tcp-server_tcp-app",
			Name:                     "tcp-server",
			Namespace:                "default",
			ProxyConnectTimeout:      "30s",
			ProxyNextUpstream:        false,
			ProxyNextUpstreamTries:   0,
			ProxyNextUpstreamTimeout: "0s",
			ProxyTimeout:             "50s",
			HealthCheck:              nil,
			ServerSnippets:           []string{},
		},
		StreamSnippets: []string{},
	}

	result := generateTransportServerConfig(&transportServerEx, listenerPort, true)
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateTransportServerConfig() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateTransportServerConfigForTLSPasstrhough(t *testing.T) {
	transportServerEx := TransportServerEx{
		TransportServer: &conf_v1alpha1.TransportServer{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "tcp-server",
				Namespace: "default",
			},
			Spec: conf_v1alpha1.TransportServerSpec{
				Listener: conf_v1alpha1.TransportServerListener{
					Name:     "tls-passthrough",
					Protocol: "TLS_PASSTHROUGH",
				},
				Host: "example.com",
				Upstreams: []conf_v1alpha1.Upstream{
					{
						Name:    "tcp-app",
						Service: "tcp-app-svc",
						Port:    5001,
					},
				},
				UpstreamParameters: &conf_v1alpha1.UpstreamParameters{
					ConnectTimeout:      "30s",
					NextUpstream:        false,
					NextUpstreamTries:   0,
					NextUpstreamTimeout: "",
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

	expected := &version2.TransportServerConfig{
		Upstreams: []version2.StreamUpstream{
			{
				Name: "ts_default_tcp-server_tcp-app",
				Servers: []version2.StreamUpstreamServer{
					{
						Address:     "10.0.0.20:5001",
						MaxFails:    1,
						FailTimeout: "10s",
					},
				},
				UpstreamLabels: version2.UpstreamLabels{
					ResourceName:      "tcp-server",
					ResourceType:      "transportserver",
					ResourceNamespace: "default",
					Service:           "tcp-app-svc",
				},
				LoadBalancingMethod: "random two least_conn",
			},
		},
		Server: version2.StreamServer{
			TLSPassthrough:           true,
			UnixSocket:               "unix:/var/lib/nginx/passthrough-default_tcp-server.sock",
			Port:                     2020,
			UDP:                      false,
			StatusZone:               "example.com",
			ProxyPass:                "ts_default_tcp-server_tcp-app",
			Name:                     "tcp-server",
			Namespace:                "default",
			ProxyConnectTimeout:      "30s",
			ProxyNextUpstream:        false,
			ProxyNextUpstreamTimeout: "0s",
			ProxyNextUpstreamTries:   0,
			ProxyTimeout:             "10m",
			HealthCheck:              nil,
			ServerSnippets:           []string{},
		},
		StreamSnippets: []string{},
	}

	result := generateTransportServerConfig(&transportServerEx, listenerPort, true)
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateTransportServerConfig() mismatch (-want +got):\n%s", diff)
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
						Name:        "udp-app",
						Service:     "udp-app-svc",
						Port:        5001,
						HealthCheck: &conf_v1alpha1.HealthCheck{},
					},
				},
				UpstreamParameters: &conf_v1alpha1.UpstreamParameters{
					UDPRequests:         &udpRequests,
					UDPResponses:        &udpResponses,
					ConnectTimeout:      "30s",
					NextUpstream:        true,
					NextUpstreamTimeout: "",
					NextUpstreamTries:   0,
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

	expected := &version2.TransportServerConfig{
		Upstreams: []version2.StreamUpstream{
			{
				Name: "ts_default_udp-server_udp-app",
				Servers: []version2.StreamUpstreamServer{
					{
						Address:     "10.0.0.20:5001",
						MaxFails:    1,
						FailTimeout: "10s",
					},
				},
				UpstreamLabels: version2.UpstreamLabels{
					ResourceName:      "udp-server",
					ResourceType:      "transportserver",
					ResourceNamespace: "default",
					Service:           "udp-app-svc",
				},
				LoadBalancingMethod: "random two least_conn",
			},
		},
		Server: version2.StreamServer{
			Port:                     2020,
			UDP:                      true,
			StatusZone:               "udp-listener",
			ProxyRequests:            &udpRequests,
			ProxyResponses:           &udpResponses,
			ProxyPass:                "ts_default_udp-server_udp-app",
			Name:                     "udp-server",
			Namespace:                "default",
			ProxyConnectTimeout:      "30s",
			ProxyNextUpstream:        true,
			ProxyNextUpstreamTimeout: "0s",
			ProxyNextUpstreamTries:   0,
			ProxyTimeout:             "10m",
			HealthCheck:              nil,
			ServerSnippets:           []string{},
		},
		StreamSnippets: []string{},
	}

	result := generateTransportServerConfig(&transportServerEx, listenerPort, true)
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateTransportServerConfig() mismatch (-want +got):\n%s", diff)
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

func TestGenerateTransportServerHealthChecks(t *testing.T) {
	upstreamName := "dns-tcp"
	generatedUpsteamName := "ts_namespace_name_dns-tcp"

	tests := []struct {
		upstreams     []conf_v1alpha1.Upstream
		expectedHC    *version2.StreamHealthCheck
		expectedMatch *version2.Match
		msg           string
	}{
		{
			upstreams: []conf_v1alpha1.Upstream{
				{
					Name: "dns-tcp",
					HealthCheck: &conf_v1alpha1.HealthCheck{
						Enabled:  false,
						Timeout:  "30s",
						Jitter:   "30s",
						Port:     80,
						Interval: "20s",
						Passes:   4,
						Fails:    5,
					},
				},
			},
			expectedHC:    nil,
			expectedMatch: nil,
			msg:           "health checks disabled",
		},
		{
			upstreams: []conf_v1alpha1.Upstream{
				{
					Name:        "dns-tcp",
					HealthCheck: &conf_v1alpha1.HealthCheck{},
				},
			},
			expectedHC:    nil,
			expectedMatch: nil,
			msg:           "empty health check",
		},
		{
			upstreams: []conf_v1alpha1.Upstream{
				{
					Name: "dns-tcp",
					HealthCheck: &conf_v1alpha1.HealthCheck{
						Enabled:  true,
						Timeout:  "40s",
						Jitter:   "30s",
						Port:     88,
						Interval: "20s",
						Passes:   4,
						Fails:    5,
					},
				},
			},
			expectedHC: &version2.StreamHealthCheck{
				Enabled:  true,
				Timeout:  "40s",
				Jitter:   "30s",
				Port:     88,
				Interval: "20s",
				Passes:   4,
				Fails:    5,
			},
			expectedMatch: nil,
			msg:           "valid health checks",
		},
		{
			upstreams: []conf_v1alpha1.Upstream{
				{
					Name: "dns-tcp",
					HealthCheck: &conf_v1alpha1.HealthCheck{
						Enabled:  true,
						Timeout:  "40s",
						Jitter:   "30s",
						Port:     88,
						Interval: "20s",
						Passes:   4,
						Fails:    5,
					},
				},
				{
					Name: "dns-tcp-2",
					HealthCheck: &conf_v1alpha1.HealthCheck{
						Enabled:  false,
						Timeout:  "50s",
						Jitter:   "60s",
						Port:     89,
						Interval: "10s",
						Passes:   9,
						Fails:    7,
					},
				},
			},
			expectedHC: &version2.StreamHealthCheck{
				Enabled:  true,
				Timeout:  "40s",
				Jitter:   "30s",
				Port:     88,
				Interval: "20s",
				Passes:   4,
				Fails:    5,
			},
			expectedMatch: nil,
			msg:           "valid 2 health checks",
		},
		{
			upstreams: []conf_v1alpha1.Upstream{
				{
					Name: "dns-tcp",
					Port: 90,
					HealthCheck: &conf_v1alpha1.HealthCheck{
						Enabled: true,
					},
				},
			},
			expectedHC: &version2.StreamHealthCheck{
				Enabled:  true,
				Timeout:  "5s",
				Jitter:   "0s",
				Port:     90,
				Interval: "5s",
				Passes:   1,
				Fails:    1,
			},
			expectedMatch: nil,
			msg:           "return default values for health check",
		},
		{
			upstreams: []conf_v1alpha1.Upstream{
				{
					Name: "dns-tcp",
					Port: 90,
					HealthCheck: &conf_v1alpha1.HealthCheck{
						Enabled: true,
						Match: &conf_v1alpha1.Match{
							Send:   `GET / HTTP/1.0\r\nHost: localhost\r\n\r\n`,
							Expect: "~*200 OK",
						},
					},
				},
			},
			expectedHC: &version2.StreamHealthCheck{
				Enabled:  true,
				Timeout:  "5s",
				Jitter:   "0s",
				Port:     90,
				Interval: "5s",
				Passes:   1,
				Fails:    1,
				Match:    "match_ts_namespace_name_dns-tcp",
			},
			expectedMatch: &version2.Match{
				Name:                "match_ts_namespace_name_dns-tcp",
				Send:                `GET / HTTP/1.0\r\nHost: localhost\r\n\r\n`,
				ExpectRegexModifier: "~*",
				Expect:              "200 OK",
			},
			msg: "health check with match",
		},
	}

	for _, test := range tests {
		hc, match := generateTransportServerHealthCheck(upstreamName, generatedUpsteamName, test.upstreams)
		if diff := cmp.Diff(test.expectedHC, hc); diff != "" {
			t.Errorf("generateTransportServerHealthCheck() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
		if diff := cmp.Diff(test.expectedMatch, match); diff != "" {
			t.Errorf("generateTransportServerHealthCheck() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestGenerateHealthCheckMatch(t *testing.T) {
	tests := []struct {
		match    *conf_v1alpha1.Match
		expected *version2.Match
		msg      string
	}{
		{
			match: &conf_v1alpha1.Match{
				Send:   "",
				Expect: "",
			},
			expected: &version2.Match{
				Name:                "match",
				Send:                "",
				ExpectRegexModifier: "",
				Expect:              "",
			},
			msg: "match with empty fields",
		},
		{
			match: &conf_v1alpha1.Match{
				Send:   "xxx",
				Expect: "yyy",
			},
			expected: &version2.Match{
				Name:                "match",
				Send:                "xxx",
				ExpectRegexModifier: "",
				Expect:              "yyy",
			},
			msg: "match with all fields and no regexp",
		},
		{
			match: &conf_v1alpha1.Match{
				Send:   "xxx",
				Expect: "~yyy",
			},
			expected: &version2.Match{
				Name:                "match",
				Send:                "xxx",
				ExpectRegexModifier: "~",
				Expect:              "yyy",
			},
			msg: "match with all fields and case sensitive regexp",
		},
		{
			match: &conf_v1alpha1.Match{
				Send:   "xxx",
				Expect: "~*yyy",
			},
			expected: &version2.Match{
				Name:                "match",
				Send:                "xxx",
				ExpectRegexModifier: "~*",
				Expect:              "yyy",
			},
			msg: "match with all fields and case insensitive regexp",
		},
	}
	name := "match"

	for _, test := range tests {
		result := generateHealthCheckMatch(test.match, name)
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("generateHealthCheckMatch() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func intPointer(value int) *int {
	return &value
}

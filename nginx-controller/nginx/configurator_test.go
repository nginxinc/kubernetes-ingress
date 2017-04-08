package nginx

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/util/intstr"
)

func TestPathOrDefaultReturnDefault(t *testing.T) {
	path := ""
	expected := "/"
	if pathOrDefault(path) != expected {
		t.Errorf("pathOrDefault(%q) should return %q", path, expected)
	}
}

func TestPathOrDefaultReturnActual(t *testing.T) {
	path := "/path/to/resource"
	if pathOrDefault(path) != path {
		t.Errorf("pathOrDefault(%q) should return %q", path, path)
	}
}

func TestParseRewrites(t *testing.T) {
	serviceName := "coffee-svc"
	serviceNamePart := "serviceName=" + serviceName
	rewritePath := "/beans/"
	rewritePathPart := "rewrite=" + rewritePath
	rewriteService := serviceNamePart + " " + rewritePathPart

	serviceNameActual, rewritePathActual, err := parseRewrites(rewriteService)
	if serviceName != serviceNameActual || rewritePath != rewritePathActual || err != nil {
		t.Errorf("parseRewrites(%s) should return %q, %q, nil; got %q, %q, %v", rewriteService, serviceName, rewritePath, serviceNameActual, rewritePathActual, err)
	}
}

func TestParseRewritesInvalidFormat(t *testing.T) {
	rewriteService := "serviceNamecoffee-svc rewrite=/"

	_, _, err := parseRewrites(rewriteService)
	if err == nil {
		t.Errorf("parseRewrites(%s) should return error, got nil", rewriteService)
	}
}

type generateNginxCfgFixture struct {
	IngressEx *IngressEx
	Servers   []Server
	Upstreams []Upstream
	Pems      map[string]string
}

var testIngressMetadata = api.ObjectMeta{
	Name:      "test",
	Namespace: "default",
	Annotations: map[string]string{
		"nginx.org/proxy-connect-timeout":    "20s",
		"nginx.org/proxy-read-timeout":       "30s",
		"nginx.org/client-max-body-size":     "2m",
		"nginx.org/http2":                    "True",
		"nginx.org/ssl-services":             "svc2",
		"nginx.org/websocket-services":       "svc3",
		"nginx.org/rewrites":                 "serviceName=svc1 rewrite=/;serviceName=svc2 rewrite=/beans/",
		"nginx.org/proxy-buffering":          "True",
		"nginx.org/proxy-buffers":            "8k",
		"nginx.org/proxy-buffer-size":        "16k",
		"nginx.org/proxy-max-temp-file-size": "1024m",
	},
}

func getGenerateNginxCfgEmptyHostFixture() *generateNginxCfgFixture {
	ingEx := &IngressEx{
		Ingress: &extensions.Ingress{
			ObjectMeta: testIngressMetadata,
			Spec: extensions.IngressSpec{
				TLS: []extensions.IngressTLS{extensions.IngressTLS{
					SecretName: "test",
					Hosts:      []string{"test.com"},
				}},
				Rules: []extensions.IngressRule{},
				Backend: &extensions.IngressBackend{
					ServiceName: "svc1",
					ServicePort: intstr.FromInt(8080),
				},
			},
		},
		Secrets: map[string]*api.Secret{
			"test": &api.Secret{
				Data: map[string][]byte{
					api.TLSCertKey:       []byte("cert cert cert"),
					api.TLSPrivateKeyKey: []byte("key key key"),
				},
			},
		},
		Endpoints: map[string][]string{
			"svc18080": []string{"10.1.100.1:8080", "10.1.100.2:8080"},
		},
	}

	pems := map[string]string{
		"": "cert",
	}

	u1 := Upstream{
		Name: "default-test--svc1",
		UpstreamServers: []UpstreamServer{
			UpstreamServer{
				Address: "10.1.100.1",
				Port:    "8080",
			},
			UpstreamServer{
				Address: "10.1.100.2",
				Port:    "8080",
			},
		},
	}

	l1 := Location{
		Path:                 "/",
		Upstream:             u1,
		ProxyConnectTimeout:  "20s",
		ProxyReadTimeout:     "30s",
		ClientMaxBodySize:    "2m",
		Websocket:            false,
		Rewrite:              "/",
		SSL:                  false,
		ProxyBuffering:       true,
		ProxyBuffers:         "8k",
		ProxyBufferSize:      "16k",
		ProxyMaxTempFileSize: "1024m",
	}

	server1 := Server{
		Name:              emptyHost,
		HTTP2:             true,
		SSL:               true,
		SSLCertificate:    "cert",
		SSLCertificateKey: "cert",
		Locations:         []Location{l1},
	}

	return &generateNginxCfgFixture{
		Pems:      pems,
		IngressEx: ingEx,
		Upstreams: []Upstream{u1},
		Servers:   []Server{server1},
	}
}

func getGenerateNginxCfgDefaultFixture() *generateNginxCfgFixture {
	ingEx := &IngressEx{
		Ingress: &extensions.Ingress{
			ObjectMeta: testIngressMetadata,
			Spec: extensions.IngressSpec{
				TLS: []extensions.IngressTLS{extensions.IngressTLS{
					SecretName: "test",
					Hosts:      []string{"test.com"},
				}},
				Rules: []extensions.IngressRule{
					// test.com host
					extensions.IngressRule{
						Host: "test.com",
						IngressRuleValue: extensions.IngressRuleValue{
							HTTP: &extensions.HTTPIngressRuleValue{
								Paths: []extensions.HTTPIngressPath{
									extensions.HTTPIngressPath{
										Path: "/other_service",
										Backend: extensions.IngressBackend{
											ServiceName: "svc2",
											ServicePort: intstr.FromInt(8080),
										},
									},
								},
							},
						},
					},

					// other.com host
					extensions.IngressRule{
						Host: "other.com",
						IngressRuleValue: extensions.IngressRuleValue{
							HTTP: &extensions.HTTPIngressRuleValue{
								Paths: []extensions.HTTPIngressPath{
									extensions.HTTPIngressPath{
										Path: "/",
										Backend: extensions.IngressBackend{
											ServiceName: "svc3",
											ServicePort: intstr.FromInt(8080),
										},
									},
								},
							},
						},
					},
				},
				Backend: &extensions.IngressBackend{
					ServiceName: "svc1",
					ServicePort: intstr.FromInt(8080),
				},
			},
		},
		Secrets: map[string]*api.Secret{
			"test": &api.Secret{
				Data: map[string][]byte{
					api.TLSCertKey:       []byte("cert cert cert"),
					api.TLSPrivateKeyKey: []byte("key key key"),
				},
			},
		},
		Endpoints: map[string][]string{
			"svc18080": []string{"10.1.100.1:8080", "10.1.100.2:8080"},
			"svc28080": []string{"10.1.100.2:8080", "10.1.100.3:8080"},
			"svc38080": []string{"10.1.100.2:8080", "10.1.100.3:8080"},
		},
	}

	pems := map[string]string{
		"test.com": "cert",
	}

	u1 := Upstream{
		Name: "default-test-test.com-svc2",
		UpstreamServers: []UpstreamServer{
			UpstreamServer{
				Address: "10.1.100.2",
				Port:    "8080",
			},
			UpstreamServer{
				Address: "10.1.100.3",
				Port:    "8080",
			},
		},
	}

	u2 := Upstream{
		Name: "default-test--svc1",
		UpstreamServers: []UpstreamServer{
			UpstreamServer{
				Address: "10.1.100.1",
				Port:    "8080",
			},
			UpstreamServer{
				Address: "10.1.100.2",
				Port:    "8080",
			},
		},
	}

	u3 := Upstream{
		Name: "default-test-other.com-svc3",
		UpstreamServers: []UpstreamServer{
			UpstreamServer{
				Address: "10.1.100.2",
				Port:    "8080",
			},
			UpstreamServer{
				Address: "10.1.100.3",
				Port:    "8080",
			},
		},
	}

	l1 := Location{
		Path:                 "/other_service",
		Upstream:             u1,
		ProxyConnectTimeout:  "20s",
		ProxyReadTimeout:     "30s",
		ClientMaxBodySize:    "2m",
		Websocket:            false,
		Rewrite:              "/beans/",
		SSL:                  true,
		ProxyBuffering:       true,
		ProxyBuffers:         "8k",
		ProxyBufferSize:      "16k",
		ProxyMaxTempFileSize: "1024m",
	}

	l2 := Location{
		Path:                 "/",
		Upstream:             u2,
		ProxyConnectTimeout:  "20s",
		ProxyReadTimeout:     "30s",
		ClientMaxBodySize:    "2m",
		Websocket:            false,
		Rewrite:              "/",
		SSL:                  false,
		ProxyBuffering:       true,
		ProxyBuffers:         "8k",
		ProxyBufferSize:      "16k",
		ProxyMaxTempFileSize: "1024m",
	}

	l3 := Location{
		Path:                 "/",
		Upstream:             u3,
		ProxyConnectTimeout:  "20s",
		ProxyReadTimeout:     "30s",
		ClientMaxBodySize:    "2m",
		Websocket:            true,
		SSL:                  false,
		ProxyBuffering:       true,
		ProxyBuffers:         "8k",
		ProxyBufferSize:      "16k",
		ProxyMaxTempFileSize: "1024m",
	}

	server1 := Server{
		Name:              "test.com",
		HTTP2:             true,
		SSL:               true,
		SSLCertificate:    "cert",
		SSLCertificateKey: "cert",
		Locations:         []Location{l2, l1},
	}

	server2 := Server{
		Name:              "other.com",
		HTTP2:             true,
		SSL:               false,
		SSLCertificate:    "",
		SSLCertificateKey: "",
		Locations:         []Location{l3},
	}

	return &generateNginxCfgFixture{
		Pems:      pems,
		IngressEx: ingEx,
		Upstreams: []Upstream{u2, u1, u3},
		Servers:   []Server{server1, server2},
	}
}

func TestConfiguratorGenerateNginxCfg(t *testing.T) {
	assert := assert.New(t)

	nginxConf := &Config{}
	nginx, _ := NewNginxController("nginxConfPath", true)
	cnf := NewConfigurator(nginx, nginxConf)

	fixtures := []*generateNginxCfgFixture{
		getGenerateNginxCfgDefaultFixture(),
		getGenerateNginxCfgEmptyHostFixture(),
	}

	for _, fixture := range fixtures {
		c := cnf.generateNginxCfg(fixture.IngressEx, fixture.Pems)
		if assert.NotNil(c) {
			// Servers test
			if assert.Len(c.Servers, len(fixture.Servers), "Unexpected number of servers in IngressNginxConfig") {
				for si, fixtureServer := range fixture.Servers {
					configServer := c.Servers[si]

					// Do not use assert.Contains(c.Servers, server)
					// because the order of location objects in the server should not matter
					assert.Equal(fixtureServer.Name, configServer.Name, "Unexpected value for Name")
					assert.Equal(fixtureServer.HTTP2, configServer.HTTP2, "Unexpected value for HTTP2")
					assert.Equal(fixtureServer.SSL, configServer.SSL, "Unexpected value for SSL")
					assert.Equal(fixtureServer.SSLCertificate, configServer.SSLCertificate, "Unexpected value for SSLCertificate")
					assert.Equal(fixtureServer.SSLCertificateKey, configServer.SSLCertificateKey, "Unexpected value for SSLCertificateKey")

					// Locations test
					assert.Len(configServer.Locations, len(fixtureServer.Locations), "Unexpected number of locations in Server.Locations")
					for _, location := range fixtureServer.Locations {
						assert.Contains(configServer.Locations, location, "Expected location was not found in Server.Locations")
					}
				}
			}

			// Upstreams test
			assert.Len(c.Upstreams, len(fixture.Upstreams), "Unexpected number of upstreams")
			for _, upstream := range fixture.Upstreams {
				assert.Contains(c.Upstreams, upstream, "Expected upstream config not found in NignxIngressConfig")
			}
		}
	}
}

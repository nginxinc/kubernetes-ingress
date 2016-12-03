package nginx

import (
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

var (
	//
	// Ingress 1
	//
	ingress1Upstream1 = Upstream{
		Name: "default-ing1-one.example.com-svc1",
	}
	ingress1Location1 = Location{
		Path:     "/1",
		Upstream: ingress1Upstream1,
	}
	ingress1Server1 = Server{
		Name:      "one.example.com",
		Locations: []Location{ingress1Location1},
	}
	ingress1Config1 = IngressNginxConfig{
		Upstreams: []Upstream{ingress1Upstream1},
		Server:    ingress1Server1,
	}
	ingress1 = extensions.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:              "ing1",
			Namespace:         "default",
			CreationTimestamp: unversioned.NewTime(time.Now().Add(-2 * time.Hour)),
		},
	}

	//
	// Ingress 2
	//
	ingress2Upstream1 = Upstream{
		Name: "default-ing2-one.example.com-svc1",
	}
	ingress2Upstream2 = Upstream{
		Name: "default-ing2-two.example.com-svc1",
	}
	ingress2Location1 = Location{
		Path:     "/1",
		Upstream: ingress2Upstream1,
	}
	ingress2Location2 = Location{
		Path:     "/2",
		Upstream: ingress2Upstream1,
	}
	ingress2Location3 = Location{
		Path:     "/3",
		Upstream: ingress2Upstream2,
	}
	ingress2Server1 = Server{
		Name:      "one.example.com",
		Locations: []Location{ingress2Location1, ingress2Location2},
	}
	ingress2Server2 = Server{
		Name:      "two.example.com",
		Locations: []Location{ingress2Location3},
	}
	ingress2Config1 = IngressNginxConfig{
		Upstreams: []Upstream{ingress2Upstream1},
		Server:    ingress2Server1,
	}
	ingress2Config2 = IngressNginxConfig{
		Upstreams: []Upstream{ingress2Upstream2},
		Server:    ingress2Server2,
	}
	ingress2 = extensions.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:      "ing2",
			Namespace: "default",
			// this ingress is older than ing1, so it shoud be the base of the merge,
			// which means that ing2 will override conflicting locations, but not server settings
			CreationTimestamp: unversioned.NewTime(time.Now().Add(-4 * time.Hour)),
		},
	}

	//
	// Ingress 3
	//
	ingress3Upstream1 = Upstream{
		Name: "default-ing3-one.example.com-svc1",
	}
	ingress3Location1 = Location{
		Path:     "/1",
		Upstream: ingress3Upstream1,
	}
	ingress3Server1 = Server{
		Name:      "one.example.com",
		Locations: []Location{ingress3Location1},
		// this server introduces additional settings,
		// because the merging process is additive they will be
		// used regardless of age
		SSL:                   true,
		SSLCertificate:        "cert.pem",
		SSLCertificateKey:     "cert.pem",
		HTTP2:                 true,
		HSTS:                  true,
		HSTSMaxAge:            2000,
		HSTSIncludeSubdomains: true,
	}
	ingress3Config1 = IngressNginxConfig{
		Upstreams: []Upstream{ingress3Upstream1},
		Server:    ingress3Server1,
	}
	ingress3 = extensions.Ingress{
		ObjectMeta: api.ObjectMeta{
			Name:              "ing3",
			Namespace:         "default",
			CreationTimestamp: unversioned.NewTime(time.Now().Add(-1 * time.Hour)),
		},
	}
)

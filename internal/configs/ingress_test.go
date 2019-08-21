package configs

import (
	"reflect"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGenerateNginxCfg(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	configParams := NewDefaultConfigParams()

	expected := createExpectedConfigForCafeIngressEx()

	pems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}

	result := generateNginxCfg(&cafeIngressEx, pems, false, configParams, false, false, "")

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateNginxCfg returned \n%v,  but expected \n%v", result, expected)
	}

}

func TestGenerateNginxCfgForJWT(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	cafeIngressEx.Ingress.Annotations["nginx.com/jwt-key"] = "cafe-jwk"
	cafeIngressEx.Ingress.Annotations["nginx.com/jwt-realm"] = "Cafe App"
	cafeIngressEx.Ingress.Annotations["nginx.com/jwt-token"] = "$cookie_auth_token"
	cafeIngressEx.Ingress.Annotations["nginx.com/jwt-login-url"] = "https://login.example.com"
	cafeIngressEx.JWTKey = JWTKey{"cafe-jwk", &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-jwk",
			Namespace: "default",
		},
	}}

	configParams := NewDefaultConfigParams()

	expected := createExpectedConfigForCafeIngressEx()
	expected.Servers[0].JWTAuth = &version1.JWTAuth{
		Key:                  "/etc/nginx/secrets/default-cafe-jwk",
		Realm:                "Cafe App",
		Token:                "$cookie_auth_token",
		RedirectLocationName: "@login_url_default-cafe-ingress",
	}
	expected.Servers[0].JWTRedirectLocations = []version1.JWTRedirectLocation{
		{
			Name:     "@login_url_default-cafe-ingress",
			LoginURL: "https://login.example.com",
		},
	}

	pems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}

	result := generateNginxCfg(&cafeIngressEx, pems, false, configParams, true, false, "/etc/nginx/secrets/default-cafe-jwk")

	if !reflect.DeepEqual(result.Servers[0].JWTAuth, expected.Servers[0].JWTAuth) {
		t.Errorf("generateNginxCfg returned \n%v,  but expected \n%v", result.Servers[0].JWTAuth, expected.Servers[0].JWTAuth)
	}
	if !reflect.DeepEqual(result.Servers[0].JWTRedirectLocations, expected.Servers[0].JWTRedirectLocations) {
		t.Errorf("generateNginxCfg returned \n%v,  but expected \n%v", result.Servers[0].JWTRedirectLocations, expected.Servers[0].JWTRedirectLocations)
	}
}

func TestGenerateNginxCfgWithMissingTLSSecret(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	configParams := NewDefaultConfigParams()
	pems := map[string]string{
		"cafe.example.com": pemFileNameForMissingTLSSecret,
	}

	result := generateNginxCfg(&cafeIngressEx, pems, false, configParams, false, false, "")

	expectedCiphers := "NULL"
	resultCiphers := result.Servers[0].SSLCiphers
	if !reflect.DeepEqual(resultCiphers, expectedCiphers) {
		t.Errorf("generateNginxCfg returned SSLCiphers %v,  but expected %v", resultCiphers, expectedCiphers)
	}
}

func TestGenerateNginxCfgWithWildcardTLSSecret(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	configParams := NewDefaultConfigParams()
	pems := map[string]string{
		"cafe.example.com": pemFileNameForWildcardTLSSecret,
	}

	result := generateNginxCfg(&cafeIngressEx, pems, false, configParams, false, false, "")

	resultServer := result.Servers[0]
	if !reflect.DeepEqual(resultServer.SSLCertificate, pemFileNameForWildcardTLSSecret) {
		t.Errorf("generateNginxCfg returned SSLCertificate %v,  but expected %v", resultServer.SSLCertificate, pemFileNameForWildcardTLSSecret)
	}
	if !reflect.DeepEqual(resultServer.SSLCertificateKey, pemFileNameForWildcardTLSSecret) {
		t.Errorf("generateNginxCfg returned SSLCertificateKey %v,  but expected %v", resultServer.SSLCertificateKey, pemFileNameForWildcardTLSSecret)
	}
}

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

func createExpectedConfigForCafeIngressEx() version1.IngressNginxConfig {
	coffeeUpstream := version1.Upstream{
		Name:             "default-cafe-ingress-cafe.example.com-coffee-svc-80",
		LBMethod:         "random two least_conn",
		UpstreamZoneSize: "256k",
		UpstreamServers: []version1.UpstreamServer{
			{
				Address:     "10.0.0.1",
				Port:        "80",
				MaxFails:    1,
				MaxConns:    0,
				FailTimeout: "10s",
			},
		},
	}
	teaUpstream := version1.Upstream{
		Name:             "default-cafe-ingress-cafe.example.com-tea-svc-80",
		LBMethod:         "random two least_conn",
		UpstreamZoneSize: "256k",
		UpstreamServers: []version1.UpstreamServer{
			{
				Address:     "10.0.0.2",
				Port:        "80",
				MaxFails:    1,
				MaxConns:    0,
				FailTimeout: "10s",
			},
		},
	}
	expected := version1.IngressNginxConfig{
		Upstreams: []version1.Upstream{
			coffeeUpstream,
			teaUpstream,
		},
		Servers: []version1.Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []version1.Location{
					{
						Path:                "/coffee",
						Upstream:            coffeeUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
					},
					{
						Path:                "/tea",
						Upstream:            teaUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
					},
				},
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
				StatusZone:        "cafe.example.com",
				HSTSMaxAge:        2592000,
				Ports:             []int{80},
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				HealthChecks:      make(map[string]version1.HealthCheck),
			},
		},
		Ingress: version1.Ingress{
			Name:      "cafe-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
	}
	return expected
}

func createCafeIngressEx() IngressEx {
	cafeIngress := v1beta1.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: v1beta1.IngressBackend{
										ServiceName: "coffee-svc",
										ServicePort: intstr.FromString("80"),
									},
								},
								{
									Path: "/tea",
									Backend: v1beta1.IngressBackend{
										ServiceName: "tea-svc",
										ServicePort: intstr.FromString("80"),
									},
								},
							},
						},
					},
				},
			},
		},
	}
	cafeIngressEx := IngressEx{
		Ingress: &cafeIngress,
		TLSSecrets: map[string]*v1.Secret{
			"cafe-secret": {},
		},
		Endpoints: map[string][]string{
			"coffee-svc80": {"10.0.0.1:80"},
			"tea-svc80":    {"10.0.0.2:80"},
		},
		ExternalNameSvcs: map[string]bool{},
	}
	return cafeIngressEx
}

func TestGenerateNginxCfgForMergeableIngresses(t *testing.T) {
	mergeableIngresses := createMergeableCafeIngress()
	expected := createExpectedConfigForMergeableCafeIngress()

	masterPems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}
	minionJwtKeyFileNames := make(map[string]string)
	configParams := NewDefaultConfigParams()

	result := generateNginxCfgForMergeableIngresses(mergeableIngresses, masterPems, "", minionJwtKeyFileNames, configParams, false, false)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateNginxCfgForMergeableIngresses returned \n%v,  but expected \n%v", result, expected)
	}
}

func TestGenerateNginxCfgForMergeableIngressesForJWT(t *testing.T) {
	mergeableIngresses := createMergeableCafeIngress()
	mergeableIngresses.Master.Ingress.Annotations["nginx.com/jwt-key"] = "cafe-jwk"
	mergeableIngresses.Master.Ingress.Annotations["nginx.com/jwt-realm"] = "Cafe"
	mergeableIngresses.Master.Ingress.Annotations["nginx.com/jwt-token"] = "$cookie_auth_token"
	mergeableIngresses.Master.Ingress.Annotations["nginx.com/jwt-login-url"] = "https://login.example.com"
	mergeableIngresses.Master.JWTKey = JWTKey{
		"cafe-jwk",
		&v1.Secret{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "cafe-jwk",
				Namespace: "default",
			},
		},
	}

	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-key"] = "coffee-jwk"
	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-realm"] = "Coffee"
	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-token"] = "$cookie_auth_token_coffee"
	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-login-url"] = "https://login.cofee.example.com"
	mergeableIngresses.Minions[0].JWTKey = JWTKey{
		"coffee-jwk",
		&v1.Secret{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "coffee-jwk",
				Namespace: "default",
			},
		},
	}

	expected := createExpectedConfigForMergeableCafeIngress()
	expected.Servers[0].JWTAuth = &version1.JWTAuth{
		Key:                  "/etc/nginx/secrets/default-cafe-jwk",
		Realm:                "Cafe",
		Token:                "$cookie_auth_token",
		RedirectLocationName: "@login_url_default-cafe-ingress-master",
	}
	expected.Servers[0].Locations[0].JWTAuth = &version1.JWTAuth{
		Key:                  "/etc/nginx/secrets/default-coffee-jwk",
		Realm:                "Coffee",
		Token:                "$cookie_auth_token_coffee",
		RedirectLocationName: "@login_url_default-cafe-ingress-coffee-minion",
	}
	expected.Servers[0].JWTRedirectLocations = []version1.JWTRedirectLocation{
		{
			Name:     "@login_url_default-cafe-ingress-master",
			LoginURL: "https://login.example.com",
		},
		{
			Name:     "@login_url_default-cafe-ingress-coffee-minion",
			LoginURL: "https://login.cofee.example.com",
		},
	}

	masterPems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}
	minionJwtKeyFileNames := make(map[string]string)
	minionJwtKeyFileNames[objectMetaToFileName(&mergeableIngresses.Minions[0].Ingress.ObjectMeta)] = "/etc/nginx/secrets/default-coffee-jwk"
	configParams := NewDefaultConfigParams()
	isPlus := true

	result := generateNginxCfgForMergeableIngresses(mergeableIngresses, masterPems, "/etc/nginx/secrets/default-cafe-jwk", minionJwtKeyFileNames, configParams, isPlus, false)

	if !reflect.DeepEqual(result.Servers[0].JWTAuth, expected.Servers[0].JWTAuth) {
		t.Errorf("generateNginxCfgForMergeableIngresses returned \n%v,  but expected \n%v", result.Servers[0].JWTAuth, expected.Servers[0].JWTAuth)
	}
	if !reflect.DeepEqual(result.Servers[0].Locations[0].JWTAuth, expected.Servers[0].Locations[0].JWTAuth) {
		t.Errorf("generateNginxCfgForMergeableIngresses returned \n%v,  but expected \n%v", result.Servers[0].Locations[0].JWTAuth, expected.Servers[0].Locations[0].JWTAuth)
	}
	if !reflect.DeepEqual(result.Servers[0].JWTRedirectLocations, expected.Servers[0].JWTRedirectLocations) {
		t.Errorf("generateNginxCfgForMergeableIngresses returned \n%v,  but expected \n%v", result.Servers[0].JWTRedirectLocations, expected.Servers[0].JWTRedirectLocations)
	}
}

func createMergeableCafeIngress() *MergeableIngresses {
	master := v1beta1.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
		Spec: v1beta1.IngressSpec{
			TLS: []v1beta1.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{ // HTTP must not be nil for Master
							Paths: []v1beta1.HTTPIngressPath{},
						},
					},
				},
			},
		},
	}

	coffeeMinion := v1beta1.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-coffee-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: v1beta1.IngressBackend{
										ServiceName: "coffee-svc",
										ServicePort: intstr.FromString("80"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	teaMinion := v1beta1.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-tea-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/tea",
									Backend: v1beta1.IngressBackend{
										ServiceName: "tea-svc",
										ServicePort: intstr.FromString("80"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	mergeableIngresses := &MergeableIngresses{
		Master: &IngressEx{
			Ingress: &master,
			TLSSecrets: map[string]*v1.Secret{
				"cafe-secret": {
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "cafe-secret",
						Namespace: "default",
					},
				},
			},
			Endpoints: map[string][]string{
				"coffee-svc80": {"10.0.0.1:80"},
				"tea-svc80":    {"10.0.0.2:80"},
			},
		},
		Minions: []*IngressEx{
			{
				Ingress: &coffeeMinion,
				Endpoints: map[string][]string{
					"coffee-svc80": {"10.0.0.1:80"},
				},
			},
			{
				Ingress: &teaMinion,
				Endpoints: map[string][]string{
					"tea-svc80": {"10.0.0.2:80"},
				},
			}},
	}

	return mergeableIngresses
}

func createExpectedConfigForMergeableCafeIngress() version1.IngressNginxConfig {
	coffeeUpstream := version1.Upstream{
		Name:             "default-cafe-ingress-coffee-minion-cafe.example.com-coffee-svc-80",
		LBMethod:         "random two least_conn",
		UpstreamZoneSize: "256k",
		UpstreamServers: []version1.UpstreamServer{
			{
				Address:     "10.0.0.1",
				Port:        "80",
				MaxFails:    1,
				MaxConns:    0,
				FailTimeout: "10s",
			},
		},
	}
	teaUpstream := version1.Upstream{
		Name:             "default-cafe-ingress-tea-minion-cafe.example.com-tea-svc-80",
		LBMethod:         "random two least_conn",
		UpstreamZoneSize: "256k",
		UpstreamServers: []version1.UpstreamServer{
			{
				Address:     "10.0.0.2",
				Port:        "80",
				MaxFails:    1,
				MaxConns:    0,
				FailTimeout: "10s",
			},
		},
	}
	expected := version1.IngressNginxConfig{
		Upstreams: []version1.Upstream{
			coffeeUpstream,
			teaUpstream,
		},
		Servers: []version1.Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []version1.Location{
					{
						Path:                "/coffee",
						Upstream:            coffeeUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &version1.Ingress{
							Name:      "cafe-ingress-coffee-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"kubernetes.io/ingress.class":      "nginx",
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
					},
					{
						Path:                "/tea",
						Upstream:            teaUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &version1.Ingress{
							Name:      "cafe-ingress-tea-minion",
							Namespace: "default",
							Annotations: map[string]string{
								"kubernetes.io/ingress.class":      "nginx",
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
					},
				},
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
				StatusZone:        "cafe.example.com",
				HSTSMaxAge:        2592000,
				Ports:             []int{80},
				SSLPorts:          []int{443},
				SSLRedirect:       true,
				HealthChecks:      make(map[string]version1.HealthCheck),
			},
		},
		Ingress: version1.Ingress{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
	}

	return expected
}

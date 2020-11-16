package configs

import (
	"reflect"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
)

func TestGenerateNginxCfg(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	configParams := NewDefaultConfigParams()

	expected := createExpectedConfigForCafeIngressEx()

	pems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}

	apRes := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, pems, apRes, false, configParams, false, false, "", &StaticConfigParams{})

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

	apRes := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, pems, apRes, false, configParams, true, false, "/etc/nginx/secrets/default-cafe-jwk", &StaticConfigParams{})

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

	apRes := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, pems, apRes, false, configParams, false, false, "", &StaticConfigParams{})

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

	apRes := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, pems, apRes, false, configParams, false, false, "", &StaticConfigParams{})

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

func TestGenerateIngressPath(t *testing.T) {
	exact := networking.PathTypeExact
	prefix := networking.PathTypePrefix
	impSpec := networking.PathTypeImplementationSpecific
	tests := []struct {
		pathType *networking.PathType
		path     string
		expected string
	}{
		{
			pathType: &exact,
			path:     "/path/to/resource",
			expected: "= /path/to/resource",
		},
		{
			pathType: &prefix,
			path:     "/path/to/resource",
			expected: "/path/to/resource",
		},
		{
			pathType: &impSpec,
			path:     "/path/to/resource",
			expected: "/path/to/resource",
		},
		{
			pathType: nil,
			path:     "/path/to/resource",
			expected: "/path/to/resource",
		},
	}
	for _, test := range tests {
		result := generateIngressPath(test.path, test.pathType)
		if result != test.expected {
			t.Errorf("generateIngressPath(%v, %v) returned %v, but expected %v", test.path, test.pathType, result, test.expected)
		}
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
						ServiceName:         "coffee-svc",
						Upstream:            coffeeUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						ProxySSLName:        "coffee-svc.default.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
						Upstream:            teaUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						ProxySSLName:        "tea-svc.default.svc",
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
	cafeIngress := networking.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: networking.IngressSpec{
			TLS: []networking.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []networking.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: networking.IngressBackend{
										ServiceName: "coffee-svc",
										ServicePort: intstr.FromString("80"),
									},
								},
								{
									Path: "/tea",
									Backend: networking.IngressBackend{
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
		ValidHosts: map[string]bool{
			"cafe.example.com": true,
		},
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

	masterApRes := make(map[string]string)
	result := generateNginxCfgForMergeableIngresses(mergeableIngresses, masterPems, masterApRes, "", minionJwtKeyFileNames, configParams, false, false, &StaticConfigParams{})

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateNginxCfgForMergeableIngresses returned \n%v,  but expected \n%v", result, expected)
	}
}

func TestGenerateNginxConfigForCrossNamespaceMergeableIngresses(t *testing.T) {
	mergeableIngresses := createMergeableCafeIngress()
	// change the namespaces of the minions to be coffee and tea
	for i, m := range mergeableIngresses.Minions {
		if strings.Contains(m.Ingress.Name, "coffee") {
			mergeableIngresses.Minions[i].Ingress.Namespace = "coffee"
		} else {
			mergeableIngresses.Minions[i].Ingress.Namespace = "tea"
		}
	}

	expected := createExpectedConfigForCrossNamespaceMergeableCafeIngress()
	masterPems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}
	minionJwtKeyFileNames := make(map[string]string)
	configParams := NewDefaultConfigParams()

	emptyApResources := make(map[string]string)
	result := generateNginxCfgForMergeableIngresses(mergeableIngresses, masterPems, emptyApResources, "", minionJwtKeyFileNames, configParams, false, false, &StaticConfigParams{})

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

	masterApRes := make(map[string]string)
	result := generateNginxCfgForMergeableIngresses(mergeableIngresses, masterPems, masterApRes, "/etc/nginx/secrets/default-cafe-jwk", minionJwtKeyFileNames, configParams, isPlus, false, &StaticConfigParams{})

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
	master := networking.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
		Spec: networking.IngressSpec{
			TLS: []networking.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []networking.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{ // HTTP must not be nil for Master
							Paths: []networking.HTTPIngressPath{},
						},
					},
				},
			},
		},
	}

	coffeeMinion := networking.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-coffee-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: networking.IngressBackend{
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

	teaMinion := networking.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-tea-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: "/tea",
									Backend: networking.IngressBackend{
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
			ValidHosts: map[string]bool{
				"cafe.example.com": true,
			},
		},
		Minions: []*IngressEx{
			{
				Ingress: &coffeeMinion,
				Endpoints: map[string][]string{
					"coffee-svc80": {"10.0.0.1:80"},
				},
				ValidHosts: map[string]bool{
					"cafe.example.com": true,
				},
				ValidMinionPaths: map[string]bool{
					"/coffee": true,
				},
			},
			{
				Ingress: &teaMinion,
				Endpoints: map[string][]string{
					"tea-svc80": {"10.0.0.2:80"},
				},
				ValidHosts: map[string]bool{
					"cafe.example.com": true,
				},
				ValidMinionPaths: map[string]bool{
					"/tea": true,
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
						ServiceName:         "coffee-svc",
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
						ProxySSLName: "coffee-svc.default.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
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
						ProxySSLName: "tea-svc.default.svc",
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

func createExpectedConfigForCrossNamespaceMergeableCafeIngress() version1.IngressNginxConfig {
	coffeeUpstream := version1.Upstream{
		Name:             "coffee-cafe-ingress-coffee-minion-cafe.example.com-coffee-svc-80",
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
		Name:             "tea-cafe-ingress-tea-minion-cafe.example.com-tea-svc-80",
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
						ServiceName:         "coffee-svc",
						Upstream:            coffeeUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &version1.Ingress{
							Name:      "cafe-ingress-coffee-minion",
							Namespace: "coffee",
							Annotations: map[string]string{
								"kubernetes.io/ingress.class":      "nginx",
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "coffee-svc.coffee.svc",
					},
					{
						Path:                "/tea",
						ServiceName:         "tea-svc",
						Upstream:            teaUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ProxySendTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						MinionIngress: &version1.Ingress{
							Name:      "cafe-ingress-tea-minion",
							Namespace: "tea",
							Annotations: map[string]string{
								"kubernetes.io/ingress.class":      "nginx",
								"nginx.org/mergeable-ingress-type": "minion",
							},
						},
						ProxySSLName: "tea-svc.tea.svc",
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

func TestGenerateNginxCfgForSpiffe(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	configParams := NewDefaultConfigParams()

	expected := createExpectedConfigForCafeIngressEx()
	expected.SpiffeClientCerts = true
	for i := range expected.Servers[0].Locations {
		expected.Servers[0].Locations[i].SSL = true
	}

	pems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}

	apResources := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, pems, apResources, false, configParams, false, false, "", &StaticConfigParams{NginxServiceMesh: true})

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateNginxCfg returned \n%v,  but expected \n%v", result, expected)
	}
}

func TestGenerateNginxCfgForInternalRoute(t *testing.T) {
	internalRouteAnnotation := "nsm.nginx.com/internal-route"
	cafeIngressEx := createCafeIngressEx()
	cafeIngressEx.Ingress.Annotations[internalRouteAnnotation] = "true"
	configParams := NewDefaultConfigParams()

	expected := createExpectedConfigForCafeIngressEx()
	expected.Servers[0].SpiffeCerts = true
	expected.Ingress.Annotations[internalRouteAnnotation] = "true"

	pems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}

	apResources := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, pems, apResources, false, configParams, false, false, "", &StaticConfigParams{NginxServiceMesh: true, EnableInternalRoutes: true})

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("generateNginxCfg returned \n%+v,  but expected \n%+v", result, expected)
	}
}

func TestIsSSLEnabled(t *testing.T) {
	type testCase struct {
		IsSSLService,
		SpiffeServerCerts,
		NginxServiceMesh,
		Expected bool
	}
	var testCases = []testCase{
		{
			IsSSLService:      false,
			SpiffeServerCerts: false,
			NginxServiceMesh:  false,
			Expected:          false,
		},
		{
			IsSSLService:      false,
			SpiffeServerCerts: true,
			NginxServiceMesh:  true,
			Expected:          false,
		},
		{
			IsSSLService:      false,
			SpiffeServerCerts: false,
			NginxServiceMesh:  true,
			Expected:          true,
		},
		{
			IsSSLService:      false,
			SpiffeServerCerts: true,
			NginxServiceMesh:  false,
			Expected:          false,
		},
		{
			IsSSLService:      true,
			SpiffeServerCerts: true,
			NginxServiceMesh:  true,
			Expected:          true,
		},
		{
			IsSSLService:      true,
			SpiffeServerCerts: false,
			NginxServiceMesh:  true,
			Expected:          true,
		},
		{
			IsSSLService:      true,
			SpiffeServerCerts: true,
			NginxServiceMesh:  false,
			Expected:          true,
		},
		{
			IsSSLService:      true,
			SpiffeServerCerts: false,
			NginxServiceMesh:  false,
			Expected:          true,
		},
	}
	for i, tc := range testCases {
		actual := isSSLEnabled(tc.IsSSLService, ConfigParams{SpiffeServerCerts: tc.SpiffeServerCerts}, &StaticConfigParams{NginxServiceMesh: tc.NginxServiceMesh})
		if actual != tc.Expected {
			t.Errorf("isSSLEnabled returned %v but expected %v for the case %v", actual, tc.Expected, i)
		}
	}
}

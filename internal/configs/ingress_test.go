package configs

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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

	apRes := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, apRes, false, configParams, false, false, &StaticConfigParams{}, false)

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateNginxCfg() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestGenerateNginxCfgForJWT(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	cafeIngressEx.Ingress.Annotations["nginx.com/jwt-key"] = "cafe-jwk"
	cafeIngressEx.Ingress.Annotations["nginx.com/jwt-realm"] = "Cafe App"
	cafeIngressEx.Ingress.Annotations["nginx.com/jwt-token"] = "$cookie_auth_token"
	cafeIngressEx.Ingress.Annotations["nginx.com/jwt-login-url"] = "https://login.example.com"
	cafeIngressEx.SecretRefs["cafe-jwk"] = &SecretReference{
		Type: "nginx.org/jwk",
		Path: "/etc/nginx/secrets/default-cafe-jwk",
	}

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

	apRes := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, apRes, false, configParams, true, false, &StaticConfigParams{}, false)

	if !reflect.DeepEqual(result.Servers[0].JWTAuth, expected.Servers[0].JWTAuth) {
		t.Errorf("generateNginxCfg returned \n%v,  but expected \n%v", result.Servers[0].JWTAuth, expected.Servers[0].JWTAuth)
	}
	if !reflect.DeepEqual(result.Servers[0].JWTRedirectLocations, expected.Servers[0].JWTRedirectLocations) {
		t.Errorf("generateNginxCfg returned \n%v,  but expected \n%v", result.Servers[0].JWTRedirectLocations, expected.Servers[0].JWTRedirectLocations)
	}
}

func TestGenerateNginxCfgWithMissingTLSSecret(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	cafeIngressEx.SecretRefs["cafe-secret"].Error = errors.New("secret doesn't exist")
	configParams := NewDefaultConfigParams()

	apRes := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, apRes, false, configParams, false, false, &StaticConfigParams{}, false)

	expectedCiphers := "NULL"
	resultCiphers := result.Servers[0].SSLCiphers
	if !reflect.DeepEqual(resultCiphers, expectedCiphers) {
		t.Errorf("generateNginxCfg returned SSLCiphers %v,  but expected %v", resultCiphers, expectedCiphers)
	}
}

func TestGenerateNginxCfgWithWildcardTLSSecret(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	cafeIngressEx.Ingress.Spec.TLS[0].SecretName = ""
	configParams := NewDefaultConfigParams()

	apRes := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, apRes, false, configParams, false, false, &StaticConfigParams{}, true)

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
		Endpoints: map[string][]string{
			"coffee-svc80": {"10.0.0.1:80"},
			"tea-svc80":    {"10.0.0.2:80"},
		},
		ExternalNameSvcs: map[string]bool{},
		ValidHosts: map[string]bool{
			"cafe.example.com": true,
		},
		SecretRefs: map[string]*SecretReference{
			"cafe-secret": {
				Type: v1.SecretTypeTLS,
				Path: "/etc/nginx/secrets/default-cafe-secret",
			},
		},
	}
	return cafeIngressEx
}

func TestGenerateNginxCfgForMergeableIngresses(t *testing.T) {
	mergeableIngresses := createMergeableCafeIngress()
	expected := createExpectedConfigForMergeableCafeIngress()

	configParams := NewDefaultConfigParams()

	masterApRes := make(map[string]string)
	result := generateNginxCfgForMergeableIngresses(mergeableIngresses, masterApRes, configParams, false, false, &StaticConfigParams{}, false)

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateNginxCfgForMergeableIngresses() returned unexpected result (-want +got):\n%s", diff)
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
	configParams := NewDefaultConfigParams()

	emptyApResources := make(map[string]string)
	result := generateNginxCfgForMergeableIngresses(mergeableIngresses, emptyApResources, configParams, false, false, &StaticConfigParams{}, false)

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateNginxCfgForMergeableIngresses() returned unexpected result (-want +got):\n%s", diff)
	}
}

func TestGenerateNginxCfgForMergeableIngressesForJWT(t *testing.T) {
	mergeableIngresses := createMergeableCafeIngress()
	mergeableIngresses.Master.Ingress.Annotations["nginx.com/jwt-key"] = "cafe-jwk"
	mergeableIngresses.Master.Ingress.Annotations["nginx.com/jwt-realm"] = "Cafe"
	mergeableIngresses.Master.Ingress.Annotations["nginx.com/jwt-token"] = "$cookie_auth_token"
	mergeableIngresses.Master.Ingress.Annotations["nginx.com/jwt-login-url"] = "https://login.example.com"
	mergeableIngresses.Master.SecretRefs["cafe-jwk"] = &SecretReference{
		Type: SecretTypeJWK,
		Path: "/etc/nginx/secrets/default-cafe-jwk",
	}

	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-key"] = "coffee-jwk"
	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-realm"] = "Coffee"
	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-token"] = "$cookie_auth_token_coffee"
	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-login-url"] = "https://login.cofee.example.com"
	mergeableIngresses.Minions[0].SecretRefs["coffee-jwk"] = &SecretReference{
		Type: SecretTypeJWK,
		Path: "/etc/nginx/secrets/default-coffee-jwk",
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

	minionJwtKeyFileNames := make(map[string]string)
	minionJwtKeyFileNames[objectMetaToFileName(&mergeableIngresses.Minions[0].Ingress.ObjectMeta)] = "/etc/nginx/secrets/default-coffee-jwk"
	configParams := NewDefaultConfigParams()
	isPlus := true

	masterApRes := make(map[string]string)
	result := generateNginxCfgForMergeableIngresses(mergeableIngresses, masterApRes, configParams, isPlus, false, &StaticConfigParams{}, false)

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
			Endpoints: map[string][]string{
				"coffee-svc80": {"10.0.0.1:80"},
				"tea-svc80":    {"10.0.0.2:80"},
			},
			ValidHosts: map[string]bool{
				"cafe.example.com": true,
			},
			SecretRefs: map[string]*SecretReference{
				"cafe-secret": {
					Type:  v1.SecretTypeTLS,
					Path:  "/etc/nginx/secrets/default-cafe-secret",
					Error: nil,
				},
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
				SecretRefs: map[string]*SecretReference{},
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
				SecretRefs: map[string]*SecretReference{},
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

	apResources := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, apResources, false, configParams, false, false,
		&StaticConfigParams{NginxServiceMesh: true}, false)

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateNginxCfg() returned unexpected result (-want +got):\n%s", diff)
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

	apResources := make(map[string]string)
	result := generateNginxCfg(&cafeIngressEx, apResources, false, configParams, false, false,
		&StaticConfigParams{NginxServiceMesh: true, EnableInternalRoutes: true}, false)

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("generateNginxCfg() returned unexpected result (-want +got):\n%s", diff)
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

func TestAddSSLConfig(t *testing.T) {
	tests := []struct {
		host              string
		tls               []networking.IngressTLS
		secretRefs        map[string]*SecretReference
		isWildcardEnabled bool
		expected          version1.Server
		msg               string
	}{
		{
			host: "some.example.com",
			tls: []networking.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			secretRefs: map[string]*SecretReference{
				"cafe-secret": {
					Type: v1.SecretTypeTLS,
					Path: "/etc/nginx/secrets/default-cafe-secret",
				},
			},
			isWildcardEnabled: false,
			expected:          version1.Server{},
			msg:               "TLS termination for different host",
		},
		{
			host: "cafe.example.com",
			tls: []networking.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			secretRefs: map[string]*SecretReference{
				"cafe-secret": {
					Type: v1.SecretTypeTLS,
					Path: "/etc/nginx/secrets/default-cafe-secret",
				},
			},
			isWildcardEnabled: false,
			expected: version1.Server{
				SSL:               true,
				SSLCertificate:    "/etc/nginx/secrets/default-cafe-secret",
				SSLCertificateKey: "/etc/nginx/secrets/default-cafe-secret",
			},
			msg: "TLS termination",
		},
		{
			host: "cafe.example.com",
			tls: []networking.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			secretRefs: map[string]*SecretReference{
				"cafe-secret": {
					Error: errors.New("invalid secret"),
				},
			},
			isWildcardEnabled: false,
			expected: version1.Server{
				SSL:               true,
				SSLCertificate:    pemFileNameForMissingTLSSecret,
				SSLCertificateKey: pemFileNameForMissingTLSSecret,
				SSLCiphers:        "NULL",
			},
			msg: "invalid secret",
		},
		{
			host: "cafe.example.com",
			tls: []networking.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			secretRefs: map[string]*SecretReference{
				"cafe-secret": {
					Type: SecretTypeCA,
					Path: "/etc/nginx/secrets/default-cafe-secret",
				},
			},
			isWildcardEnabled: false,
			expected: version1.Server{
				SSL:               true,
				SSLCertificate:    pemFileNameForMissingTLSSecret,
				SSLCertificateKey: pemFileNameForMissingTLSSecret,
				SSLCiphers:        "NULL",
			},
			msg: "secret of wrong type",
		},
		{
			host: "cafe.example.com",
			tls: []networking.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "",
				},
			},
			isWildcardEnabled: true,
			expected: version1.Server{
				SSL:               true,
				SSLCertificate:    pemFileNameForWildcardTLSSecret,
				SSLCertificateKey: pemFileNameForWildcardTLSSecret,
			},
			msg: "no secret name with wildcard enabled",
		},
		{
			host: "cafe.example.com",
			tls: []networking.IngressTLS{
				{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "",
				},
			},
			isWildcardEnabled: false,
			expected: version1.Server{
				SSL:               true,
				SSLCertificate:    pemFileNameForMissingTLSSecret,
				SSLCertificateKey: pemFileNameForMissingTLSSecret,
				SSLCiphers:        "NULL",
			},
			msg: "no secret name with wildcard disabled",
		},
	}

	for _, test := range tests {
		var result version1.Server

		addSSLConfig(&result, test.host, test.tls, test.secretRefs, test.isWildcardEnabled)

		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("addSSLConfig() '%s' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestGenerateJWTConfig(t *testing.T) {
	tests := []struct {
		secretRefs               map[string]*SecretReference
		cfgParams                *ConfigParams
		redirectLocationName     string
		expectedJWTAuth          *version1.JWTAuth
		expectedRedirectLocation *version1.JWTRedirectLocation
		msg                      string
	}{
		{
			secretRefs: map[string]*SecretReference{
				"cafe-jwk": {
					Type: SecretTypeJWK,
					Path: "/etc/nginx/secrets/default-cafe-jwk",
				},
			},
			cfgParams: &ConfigParams{
				JWTKey:   "cafe-jwk",
				JWTRealm: "cafe",
				JWTToken: "$http_token",
			},
			redirectLocationName: "@loc",
			expectedJWTAuth: &version1.JWTAuth{
				Key:   "/etc/nginx/secrets/default-cafe-jwk",
				Realm: "cafe",
				Token: "$http_token",
			},
			expectedRedirectLocation: nil,
			msg:                      "normal case",
		},
		{
			secretRefs: map[string]*SecretReference{
				"cafe-jwk": {
					Type: SecretTypeJWK,
					Path: "/etc/nginx/secrets/default-cafe-jwk",
				},
			},
			cfgParams: &ConfigParams{
				JWTKey:      "cafe-jwk",
				JWTRealm:    "cafe",
				JWTToken:    "$http_token",
				JWTLoginURL: "http://cafe.example.com/login",
			},
			redirectLocationName: "@loc",
			expectedJWTAuth: &version1.JWTAuth{
				Key:                  "/etc/nginx/secrets/default-cafe-jwk",
				Realm:                "cafe",
				Token:                "$http_token",
				RedirectLocationName: "@loc",
			},
			expectedRedirectLocation: &version1.JWTRedirectLocation{
				Name:     "@loc",
				LoginURL: "http://cafe.example.com/login",
			},
			msg: "normal case with login url",
		},
		{
			secretRefs: map[string]*SecretReference{
				"cafe-jwk": {
					Path:  "/etc/nginx/secrets/default-cafe-jwk",
					Error: errors.New("invalid secret"),
				},
			},
			cfgParams: &ConfigParams{
				JWTKey:   "cafe-jwk",
				JWTRealm: "cafe",
				JWTToken: "$http_token",
			},
			redirectLocationName: "@loc",
			expectedJWTAuth: &version1.JWTAuth{
				Key:   "/etc/nginx/secrets/default-cafe-jwk",
				Realm: "cafe",
				Token: "$http_token",
			},
			expectedRedirectLocation: nil,
			msg:                      "invalid secret",
		},
		{
			secretRefs: map[string]*SecretReference{
				"cafe-jwk": {
					Type: SecretTypeCA,
					Path: "/etc/nginx/secrets/default-cafe-jwk",
				},
			},
			cfgParams: &ConfigParams{
				JWTKey:   "cafe-jwk",
				JWTRealm: "cafe",
				JWTToken: "$http_token",
			},
			redirectLocationName: "@loc",
			expectedJWTAuth: &version1.JWTAuth{
				Key:   "/etc/nginx/secrets/default-cafe-jwk",
				Realm: "cafe",
				Token: "$http_token",
			},
			expectedRedirectLocation: nil,
			msg:                      "secret of wrong type",
		},
	}

	for _, test := range tests {
		jwtAuth, redirectLocation := generateJWTConfig(test.secretRefs, test.cfgParams, test.redirectLocationName)

		if diff := cmp.Diff(test.expectedJWTAuth, jwtAuth); diff != "" {
			t.Errorf("generateJWTConfig() '%s' mismatch for jwtAuth (-want +got):\n%s", test.msg, diff)
		}
		if diff := cmp.Diff(test.expectedRedirectLocation, redirectLocation); diff != "" {
			t.Errorf("generateJWTConfig() '%s' mismatch for redirectLocation (-want +got):\n%s", test.msg, diff)
		}
	}
}

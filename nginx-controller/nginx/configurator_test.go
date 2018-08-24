package nginx

import (
	"net/http"
	"reflect"
	"sort"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/nginx-controller/nginx/plus"

	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

func TestParseStickyService(t *testing.T) {
	serviceName := "coffee-svc"
	serviceNamePart := "serviceName=" + serviceName
	stickyCookie := "srv_id expires=1h domain=.example.com path=/"
	stickyService := serviceNamePart + " " + stickyCookie

	serviceNameActual, stickyCookieActual, err := parseStickyService(stickyService)
	if serviceName != serviceNameActual || stickyCookie != stickyCookieActual || err != nil {
		t.Errorf("parseStickyService(%s) should return %q, %q, nil; got %q, %q, %v", stickyService, serviceName, stickyCookie, serviceNameActual, stickyCookieActual, err)
	}
}

func TestParseStickyServiceInvalidFormat(t *testing.T) {
	stickyService := "serviceNamecoffee-svc srv_id expires=1h domain=.example.com path=/"

	_, _, err := parseStickyService(stickyService)
	if err == nil {
		t.Errorf("parseStickyService(%s) should return error, got nil", stickyService)
	}
}

func TestFilterMasterAnnotations(t *testing.T) {
	masterAnnotations := map[string]string{
		"nginx.org/rewrites":                "serviceName=service1 rewrite=rewrite1",
		"nginx.org/ssl-services":            "service1",
		"nginx.org/hsts":                    "True",
		"nginx.org/hsts-max-age":            "2700000",
		"nginx.org/hsts-include-subdomains": "True",
	}
	removedAnnotations := filterMasterAnnotations(masterAnnotations)

	expectedfilteredMasterAnnotations := map[string]string{
		"nginx.org/hsts":                    "True",
		"nginx.org/hsts-max-age":            "2700000",
		"nginx.org/hsts-include-subdomains": "True",
	}
	expectedRemovedAnnotations := []string{
		"nginx.org/rewrites",
		"nginx.org/ssl-services",
	}

	sort.Strings(removedAnnotations)
	sort.Strings(expectedRemovedAnnotations)

	if !reflect.DeepEqual(expectedfilteredMasterAnnotations, masterAnnotations) {
		t.Errorf("filterMasterAnnotations returned %v, but expected %v", masterAnnotations, expectedfilteredMasterAnnotations)
	}
	if !reflect.DeepEqual(expectedRemovedAnnotations, removedAnnotations) {
		t.Errorf("filterMasterAnnotations returned %v, but expected %v", removedAnnotations, expectedRemovedAnnotations)
	}
}

func TestFilterMinionAnnotations(t *testing.T) {
	minionAnnotations := map[string]string{
		"nginx.org/rewrites":                "serviceName=service1 rewrite=rewrite1",
		"nginx.org/ssl-services":            "service1",
		"nginx.org/hsts":                    "True",
		"nginx.org/hsts-max-age":            "2700000",
		"nginx.org/hsts-include-subdomains": "True",
	}
	removedAnnotations := filterMinionAnnotations(minionAnnotations)

	expectedfilteredMinionAnnotations := map[string]string{
		"nginx.org/rewrites":     "serviceName=service1 rewrite=rewrite1",
		"nginx.org/ssl-services": "service1",
	}
	expectedRemovedAnnotations := []string{
		"nginx.org/hsts",
		"nginx.org/hsts-max-age",
		"nginx.org/hsts-include-subdomains",
	}

	sort.Strings(removedAnnotations)
	sort.Strings(expectedRemovedAnnotations)

	if !reflect.DeepEqual(expectedfilteredMinionAnnotations, minionAnnotations) {
		t.Errorf("filterMinionAnnotations returned %v, but expected %v", minionAnnotations, expectedfilteredMinionAnnotations)
	}
	if !reflect.DeepEqual(expectedRemovedAnnotations, removedAnnotations) {
		t.Errorf("filterMinionAnnotations returned %v, but expected %v", removedAnnotations, expectedRemovedAnnotations)
	}
}

func TestMergeMasterAnnotationsIntoMinion(t *testing.T) {
	masterAnnotations := map[string]string{
		"nginx.org/proxy-buffering":       "True",
		"nginx.org/proxy-buffers":         "2",
		"nginx.org/proxy-buffer-size":     "8k",
		"nginx.org/hsts":                  "True",
		"nginx.org/hsts-max-age":          "2700000",
		"nginx.org/proxy-connect-timeout": "50s",
		"nginx.com/jwt-token":             "$cookie_auth_token",
	}
	minionAnnotations := map[string]string{
		"nginx.org/client-max-body-size":  "2m",
		"nginx.org/proxy-connect-timeout": "20s",
	}
	mergeMasterAnnotationsIntoMinion(minionAnnotations, masterAnnotations)

	expectedMergedAnnotations := map[string]string{
		"nginx.org/proxy-buffering":       "True",
		"nginx.org/proxy-buffers":         "2",
		"nginx.org/proxy-buffer-size":     "8k",
		"nginx.org/client-max-body-size":  "2m",
		"nginx.org/proxy-connect-timeout": "20s",
	}
	if !reflect.DeepEqual(expectedMergedAnnotations, minionAnnotations) {
		t.Errorf("mergeMasterAnnotationsIntoMinion returned %v, but expected %v", minionAnnotations, expectedMergedAnnotations)
	}
}

func createCafeIngressEx() IngressEx {
	cafeIngress := extensions.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: extensions.IngressSpec{
			TLS: []extensions.IngressTLS{
				extensions.IngressTLS{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []extensions.IngressRule{
				extensions.IngressRule{
					Host: "cafe.example.com",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								extensions.HTTPIngressPath{
									Path: "/coffee",
									Backend: extensions.IngressBackend{
										ServiceName: "coffee-svc",
										ServicePort: intstr.FromString("80"),
									},
								},
								extensions.HTTPIngressPath{
									Path: "/tea",
									Backend: extensions.IngressBackend{
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
		TLSSecrets: map[string]*api_v1.Secret{
			"cafe-secret": &api_v1.Secret{},
		},
		Endpoints: map[string][]string{
			"coffee-svc80": []string{"10.0.0.1:80"},
			"tea-svc80":    []string{"10.0.0.2:80"},
		},
	}
	return cafeIngressEx
}

func createExpectedConfigForCafeIngressEx() IngressNginxConfig {
	coffeeUpstream := Upstream{
		Name:     "default-cafe-ingress-cafe.example.com-coffee-svc-80",
		LBMethod: "least_conn",
		UpstreamServers: []UpstreamServer{
			{
				Address:     "10.0.0.1",
				Port:        "80",
				MaxFails:    1,
				FailTimeout: "10s",
			},
		},
	}
	teaUpstream := Upstream{
		Name:     "default-cafe-ingress-cafe.example.com-tea-svc-80",
		LBMethod: "least_conn",
		UpstreamServers: []UpstreamServer{
			{
				Address:     "10.0.0.2",
				Port:        "80",
				MaxFails:    1,
				FailTimeout: "10s",
			},
		},
	}
	expected := IngressNginxConfig{
		Upstreams: []Upstream{
			coffeeUpstream,
			teaUpstream,
		},
		Servers: []Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []Location{
					{
						Path:                "/coffee",
						Upstream:            coffeeUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
					},
					{
						Path:                "/tea",
						Upstream:            teaUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
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
				HealthChecks:      make(map[string]HealthCheck),
			},
		},
	}
	return expected
}

func createMergeableCafeIngress() *MergeableIngresses {
	master := extensions.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
		Spec: extensions.IngressSpec{
			TLS: []extensions.IngressTLS{
				extensions.IngressTLS{
					Hosts:      []string{"cafe.example.com"},
					SecretName: "cafe-secret",
				},
			},
			Rules: []extensions.IngressRule{
				extensions.IngressRule{
					Host: "cafe.example.com",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{ // HTTP must not be nil for Master
							Paths: []extensions.HTTPIngressPath{},
						},
					},
				},
			},
		},
	}

	coffeeMinion := extensions.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-coffee-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: extensions.IngressSpec{
			Rules: []extensions.IngressRule{
				extensions.IngressRule{
					Host: "cafe.example.com",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								extensions.HTTPIngressPath{
									Path: "/coffee",
									Backend: extensions.IngressBackend{
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

	teaMinion := extensions.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-tea-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: extensions.IngressSpec{
			Rules: []extensions.IngressRule{
				extensions.IngressRule{
					Host: "cafe.example.com",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								extensions.HTTPIngressPath{
									Path: "/tea",
									Backend: extensions.IngressBackend{
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
			TLSSecrets: map[string]*api_v1.Secret{
				"cafe-secret": &api_v1.Secret{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "cafe-secret",
						Namespace: "default",
					},
				},
			},
			Endpoints: map[string][]string{
				"coffee-svc80": []string{"10.0.0.1:80"},
				"tea-svc80":    []string{"10.0.0.2:80"},
			},
		},
		Minions: []*IngressEx{
			&IngressEx{
				Ingress: &coffeeMinion,
				Endpoints: map[string][]string{
					"coffee-svc80": []string{"10.0.0.1:80"},
				},
			},
			&IngressEx{
				Ingress: &teaMinion,
				Endpoints: map[string][]string{
					"tea-svc80": []string{"10.0.0.2:80"},
				},
			}},
	}

	return mergeableIngresses
}

func createExpectedConfigForMergeableCafeIngress() IngressNginxConfig {
	coffeeUpstream := Upstream{
		Name:     "default-cafe-ingress-coffee-minion-cafe.example.com-coffee-svc-80",
		LBMethod: "least_conn",
		UpstreamServers: []UpstreamServer{
			{
				Address:     "10.0.0.1",
				Port:        "80",
				MaxFails:    1,
				FailTimeout: "10s",
			},
		},
	}
	teaUpstream := Upstream{
		Name:     "default-cafe-ingress-tea-minion-cafe.example.com-tea-svc-80",
		LBMethod: "least_conn",
		UpstreamServers: []UpstreamServer{
			{
				Address:     "10.0.0.2",
				Port:        "80",
				MaxFails:    1,
				FailTimeout: "10s",
			},
		},
	}
	expected := IngressNginxConfig{
		Upstreams: []Upstream{
			coffeeUpstream,
			teaUpstream,
		},
		Servers: []Server{
			{
				Name:         "cafe.example.com",
				ServerTokens: "on",
				Locations: []Location{
					{
						Path:                "/coffee",
						Upstream:            coffeeUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						IngressResource:     "default-cafe-ingress-coffee-minion",
					},
					{
						Path:                "/tea",
						Upstream:            teaUpstream,
						ProxyConnectTimeout: "60s",
						ProxyReadTimeout:    "60s",
						ClientMaxBodySize:   "1m",
						ProxyBuffering:      true,
						IngressResource:     "default-cafe-ingress-tea-minion",
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
				HealthChecks:      make(map[string]HealthCheck),
				IngressResource:   "default-cafe-ingress-master",
			},
		},
	}
	return expected

}

func createTestConfigurator() (*Configurator, error) {
	templateExecutor, err := NewTemplateExecutor("templates/nginx-plus.tmpl", "templates/nginx-plus.ingress.tmpl", true, true, 8080)
	if err != nil {
		return nil, err
	}
	ngxc := NewNginxController("/etc/nginx", true)
	apiCtrl, err := plus.NewNginxAPIController(&http.Client{}, "", true)
	if err != nil {
		return nil, err
	}
	return NewConfigurator(ngxc, NewDefaultConfig(), apiCtrl, templateExecutor), nil
}

func createTestConfiguratorInvalidIngressTemplate() (*Configurator, error) {
	templateExecutor, err := NewTemplateExecutor("templates/nginx-plus.tmpl", "templates/nginx-plus.ingress.tmpl", true, true, 8080)
	if err != nil {
		return nil, err
	}
	invalidIngressTemplate := "{{.Upstreams.This.Field.Does.Not.Exist}}"
	if err := templateExecutor.UpdateIngressTemplate(&invalidIngressTemplate); err != nil {
		return nil, err
	}
	ngxc := NewNginxController("/etc/nginx", true)
	apiCtrl, _ := plus.NewNginxAPIController(&http.Client{}, "", true)
	return NewConfigurator(ngxc, NewDefaultConfig(), apiCtrl, templateExecutor), nil
}

func TestGenerateNginxCfg(t *testing.T) {
	cafeIngressEx := createCafeIngressEx()
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}
	expected := createExpectedConfigForCafeIngressEx()

	pems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}

	result := cnf.generateNginxCfg(&cafeIngressEx, pems, "", false)

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
	cafeIngressEx.JWTKey = &api_v1.Secret{}

	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}
	expected := createExpectedConfigForCafeIngressEx()
	expected.Servers[0].JWTAuth = &JWTAuth{
		Key:                  "/etc/nginx/secrets/default-cafe-jwk",
		Realm:                "Cafe App",
		Token:                "$cookie_auth_token",
		RedirectLocationName: "@login_url_default-cafe-ingress",
	}
	expected.Servers[0].JWTRedirectLocations = []JWTRedirectLocation{
		{
			Name:     "@login_url_default-cafe-ingress",
			LoginURL: "https://login.example.com",
		},
	}

	pems := map[string]string{
		"cafe.example.com": "/etc/nginx/secrets/default-cafe-secret",
	}

	result := cnf.generateNginxCfg(&cafeIngressEx, pems, "/etc/nginx/secrets/default-cafe-jwk", false)

	if !reflect.DeepEqual(result.Servers[0].JWTAuth, expected.Servers[0].JWTAuth) {
		t.Errorf("generateNginxCfg returned \n%v,  but expected \n%v", result.Servers[0].JWTAuth, expected.Servers[0].JWTAuth)
	}
	if !reflect.DeepEqual(result.Servers[0].JWTRedirectLocations, expected.Servers[0].JWTRedirectLocations) {
		t.Errorf("generateNginxCfg returned \n%v,  but expected \n%v", result.Servers[0].JWTRedirectLocations, expected.Servers[0].JWTRedirectLocations)
	}
}

func TestGenerateNginxCfgForMergeableIngresses(t *testing.T) {
	mergeableIngresses := createMergeableCafeIngress()
	expected := createExpectedConfigForMergeableCafeIngress()

	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	result := cnf.generateNginxCfgForMergeableIngresses(mergeableIngresses)

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
	mergeableIngresses.Master.JWTKey = &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-jwk",
			Namespace: "default",
		},
	}

	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-key"] = "coffee-jwk"
	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-realm"] = "Coffee"
	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-token"] = "$cookie_auth_token_coffee"
	mergeableIngresses.Minions[0].Ingress.Annotations["nginx.com/jwt-login-url"] = "https://login.cofee.example.com"
	mergeableIngresses.Minions[0].JWTKey = &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee-jwk",
			Namespace: "default",
		},
	}

	expected := createExpectedConfigForMergeableCafeIngress()
	expected.Servers[0].JWTAuth = &JWTAuth{
		Key:                  "/etc/nginx/secrets/default-cafe-jwk",
		Realm:                "Cafe",
		Token:                "$cookie_auth_token",
		RedirectLocationName: "@login_url_default-cafe-ingress-master",
	}
	expected.Servers[0].Locations[0].JWTAuth = &JWTAuth{
		Key:                  "/etc/nginx/secrets/default-coffee-jwk",
		Realm:                "Coffee",
		Token:                "$cookie_auth_token_coffee",
		RedirectLocationName: "@login_url_default-cafe-ingress-coffee-minion",
	}
	expected.Servers[0].JWTRedirectLocations = []JWTRedirectLocation{
		{
			Name:     "@login_url_default-cafe-ingress-master",
			LoginURL: "https://login.example.com",
		},
		{
			Name:     "@login_url_default-cafe-ingress-coffee-minion",
			LoginURL: "https://login.cofee.example.com",
		},
	}

	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	result := cnf.generateNginxCfgForMergeableIngresses(mergeableIngresses)

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

func TestAddOrUpdateIngress(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}
	ingress := createCafeIngressEx()
	err = cnf.AddOrUpdateIngress(&ingress)
	if err != nil {
		t.Errorf("AddOrUpdateIngress returned:  \n%v, but expected: \n%v", err, nil)
	}

	cnfHasIngress := cnf.HasIngress(ingress.Ingress)
	if !cnfHasIngress {
		t.Errorf("AddOrUpdateIngress didn't add ingress successfully. HasIngress returned %v, expected %v", cnfHasIngress, true)
	}
}

func TestAddOrUpdateMergeableIngress(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}
	mergeableIngess := createMergeableCafeIngress()
	err = cnf.AddOrUpdateMergeableIngress(mergeableIngess)
	if err != nil {
		t.Errorf("AddOrUpdateMergeableIngress returned \n%v, expected \n%v", err, nil)
	}

	cnfHasMergeableIngress := cnf.HasIngress(mergeableIngess.Master.Ingress)
	if !cnfHasMergeableIngress {
		t.Errorf("AddOrUpdateMergeableIngress didn't add mergeable ingress successfully. HasIngress returned %v, expected %v", cnfHasMergeableIngress, true)
	}
}

func TestAddOrUpdateIngressFailsWithInvalidIngressTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()
	err = cnf.AddOrUpdateIngress(&ingress)
	if err == nil {
		t.Errorf("AddOrUpdateIngressFailsWithInvalidTemplate returned \n%v,  but expected \n%v", nil, "template execution error")
	}
}

func TestAddOrUpdateMergeableIngressFailsWithInvalidIngressTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngess := createMergeableCafeIngress()
	err = cnf.AddOrUpdateMergeableIngress(mergeableIngess)
	if err == nil {
		t.Errorf("AddOrUpdateMergeableIngress returned \n%v, but expected \n%v", nil, "template execution error")
	}
}

func TestUpdateEndpoints(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()
	err = cnf.UpdateEndpoints(&ingress)
	if err != nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", err, nil)
	}

	// test with OSS Configurator
	cnf.nginxAPI = nil
	err = cnf.UpdateEndpoints(&ingress)
	if err != nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", err, nil)
	}
}

func TestUpdateEndpointsMergeableIngress(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngress := createMergeableCafeIngress()
	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngress)
	if err != nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", err, nil)
	}

	// test with OSS Configurator
	cnf.nginxAPI = nil
	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngress)
	if err != nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", err, nil)
	}
}

func TestUpdateEndpointsFailsWithInvalidTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()
	err = cnf.UpdateEndpoints(&ingress)
	if err == nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", nil, "template execution error")
	}
}

func TestUpdateEndpointsMergeableIngressFailsWithInvalidTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngress := createMergeableCafeIngress()
	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngress)
	if err == nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", nil, "template execution error")
	}
}

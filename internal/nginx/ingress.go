package nginx

import (
	api_v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
)

// IngressEx holds an Ingress along with Secrets and Endpoints of the services
// that are referenced in this Ingress
type IngressEx struct {
	Ingress      *extensions.Ingress
	TLSSecrets   map[string]*api_v1.Secret
	JWTKey       *api_v1.Secret
	Endpoints    map[string][]string
	HealthChecks map[string]*api_v1.Probe
}

type MergeableIngresses struct {
	Master  *IngressEx
	Minions []*IngressEx
}

var masterBlacklist = map[string]bool{
	"nginx.org/rewrites":                      true,
	"nginx.org/ssl-services":                  true,
	"nginx.org/grpc-services":                 true,
	"nginx.org/websocket-services":            true,
	"nginx.com/sticky-cookie-services":        true,
	"nginx.com/health-checks":                 true,
	"nginx.com/health-checks-mandatory":       true,
	"nginx.com/health-checks-mandatory-queue": true,
}

var minionBlacklist = map[string]bool{
	"nginx.org/proxy-hide-headers":       true,
	"nginx.org/proxy-pass-headers":       true,
	"nginx.org/redirect-to-https":        true,
	"ingress.kubernetes.io/ssl-redirect": true,
	"nginx.org/hsts":                     true,
	"nginx.org/hsts-max-age":             true,
	"nginx.org/hsts-include-subdomains":  true,
	"nginx.org/server-tokens":            true,
	"nginx.org/listen-ports":             true,
	"nginx.org/listen-ports-ssl":         true,
	"nginx.org/server-snippets":          true,
}

var minionInheritanceList = map[string]bool{
	"nginx.org/proxy-connect-timeout":    true,
	"nginx.org/proxy-read-timeout":       true,
	"nginx.org/client-max-body-size":     true,
	"nginx.org/proxy-buffering":          true,
	"nginx.org/proxy-buffers":            true,
	"nginx.org/proxy-buffer-size":        true,
	"nginx.org/proxy-max-temp-file-size": true,
	"nginx.org/location-snippets":        true,
	"nginx.org/lb-method":                true,
	"nginx.org/keepalive":                true,
	"nginx.org/max-fails":                true,
	"nginx.org/fail-timeout":             true,
}

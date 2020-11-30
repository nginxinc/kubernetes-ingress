module github.com/nginxinc/kubernetes-ingress

go 1.15

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/google/go-cmp v0.5.2
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/nginxinc/nginx-plus-go-client v0.8.0
	github.com/nginxinc/nginx-prometheus-exporter v0.8.1-0.20201130150826-7c45b2334b3c
	github.com/prometheus/client_golang v1.8.0
	github.com/prometheus/common v0.15.0 // indirect
	github.com/spiffe/go-spiffe v1.1.0
	golang.org/x/sys v0.0.0-20201130072748-111129e158e2 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.3
	k8s.io/code-generator v0.19.3
	sigs.k8s.io/controller-tools v0.4.1
)

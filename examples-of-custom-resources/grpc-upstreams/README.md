# gRPC support

To support a gRPC application using Virtual server resources with NGINX Ingress controllers, you need to add the **type: grpc** field to an upstream.
The protocol defaults to http if left unset.

## Prerequisites

* HTTP/2 must be enabled. See `http2` ConfigMap key in the [ConfigMap](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/configmap-resource/#listeners)


## Example

In the following example we load balance three applications, one of which is using gRPC:
```yaml
apiVersion: k8s.nginx.org/v1
kind: VirtualServer
metadata:
  name: cafe
spec:
  host: cafe.example.com
  tls:
    secret: cafe-secret
  upstreams:
  - name: grpc
    service: grpc-svc
    port: 80
    type: grpc
  - name: tea
    service: tea-svc
    port: 80
  - name: coffee
    service: coffee-svc
    port: 80
  routes:
  - path: /grpc
    action:
      pass: grpc
  - path: /tea
    action:
      pass: tea
  - path: /coffee
    action:
      pass: coffee
```
*grpc-svc* is a service for the gRPC application. 

# gRPC support

To support a gRPC application using VirtualServer resources with NGINX Ingress Controller, you need to add the **type:
grpc** field to an upstream. The protocol defaults to http if left unset.

## Prerequisites

- HTTP/2 must be enabled. See `http2` ConfigMap key in the
  [ConfigMap](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/configmap-resource/#listeners)

- VirtualServer and VirtualServerRoute resources for gRPC applications must include TLS termination.

## Example

```yaml
apiVersion: k8s.nginx.org/v1
kind: VirtualServer
metadata:
  name: grpc-vs
spec:
  host: grpc.example.com
  tls:
    secret: grpc-secret
  upstreams:
  - name: grpc1
    service: grpc-svc
    port: 50051
    type: grpc
  routes:
  - path: /helloworld.Greeter
    action:
      pass: grpc1
```

*grpc-svc* is a service for the gRPC application. The service becomes available at the `/helloworld.Greeter` path. Note
how we used the **type: grpc** field in the Upstream configuration.

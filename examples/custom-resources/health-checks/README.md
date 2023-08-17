# Support for Active Health Checks

NGINX Plus supports [active health
checks](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-health-check/#active-health-checks). To use active
health checks in the Ingress Controller:

1. Define health checks ([HTTP Readiness
   Probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#define-readiness-probes))
   in the templates of your application pods.
2. Enable heath checks in the VirtualServer resource for your application. For the full list of configurable parameters,
   see the
   [docs](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#upstreamhealthcheck).

## Example

In the following example we enable active health checks in the cafe VirtualServer for the tea-svc service:

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
  - name: tea
    service: tea-svc
    port: 80
    healthCheck:
      enable: true
      path: /healthz
      interval: 20s
      jitter: 3s
      keep_alive: 120s
      fails: 5
      passes: 5
      port: 8080
      tls:
        enable: true
      connect-timeout: 10s
      read-timeout: 10s
      send-timeout: 10s
      headers:
      - name: Host
        value: my.service
      statusMatch: "! 500"
  routes:
  - path: /tea
    action:
      pass: tea
```

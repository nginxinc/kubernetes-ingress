# Wildcard TLS Certificate

The wildcard TLS certificate simplifies the configuration of TLS termination if you need to use the same TLS certificate
in multiple Ingress and VirtualServer resources from various namespaces. Typically, such a certificate is for a
subdomain (for example, `*.example.com`), while the hosts in the Ingress and VirtualServer resources include that
subdomain (for example, `foo.example.com`, `bar.example.com`).

## Example

### Prerequisites

Start the Ingress Controller with the `-wildcard-tls-secret` [command-line
argument](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-wildcard-tls-secret)
set to a TLS secret with a wildcard cert/key. For example:

```yaml
-wildcard-tls-secret=nginx-ingress/wildcard-tls-secret
```

**Note**: the Ingress Controller supports only one wildcard TLS secret.

### Configuring TLS Termination

In the example below we configure TLS termination for an Ingress for the host `foo.example.com` and a VirtualServer for
the host `bar.example.com`:

`foo` Ingress from the namespace `foo`:

 ```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: foo
  namespace: foo
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - foo.example.com
  rules:
  - host: foo.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: foo-service
            port:
              number: 80
 ```

`bar` VirtualServer from the namespace `bar`:

```yaml
apiVersion: k8s.nginx.org/v1
kind: VirtualServer
metadata:
  name: bar
  namespace: bar
spec:
  host: bar.example.com
  tls:
    secret: ""
  upstreams:
  - name: bar
    service: bar-service
    port: 80
  routes:
  - path: /
    action:
      pass: bar
```

Because we don't reference any TLS secret in the resources above -- there is no `secret` field in the `tls` section of
the Ingress resource and the `secret` field is empty in the VirtualServer -- NGINX will use the wildcard secret
specified in the `-wildcard-tls-secret` command-line argument.

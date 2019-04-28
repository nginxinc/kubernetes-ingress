# Wildcard TLS Certificate

The wildcard TLS certificate simplifies the configuration of TLS termination if you need to use the same TLS certificate in multiple Ingress resources from various namespaces. Typically, such a certificate is for a subdomain (for example, `*.example.com`), while the hosts in the Ingress resources include that subdomain (for example, `foo.example.com`, `bar.example.com`).

## Example

### Prerequisites

Start the Ingress Controller with the `-wildcard-tls-secret` [command-line argument](../../docs/cli-arguments.md) set to a TLS secret with a wildcard cert/key. For example:

```yaml
-wildcard-tls-secret=nginx-ingress/wildlcard-tls-secret
```

**Note**: the Ingress Controller supports only one wildcard TLS secret.

### Configuring TLS Termination

In the example below we configure TLS termination for two Ingress resources for the hosts `foo.example.com` and `bar.example.com` respectively:

`foo-ingress` from the namespace `foo-namespace`:

 ```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: foo-ingress
  namespace: foo-namespace
  annotations:
    kubernetes.io/ingress.class: "nginx"
spec:
  tls:
  - hosts:
    - foo.example.com
  rules:
  - host: foo.example.com
    http:
      paths:
      - path: /
        backend:
          serviceName: foo-service
          servicePort: 80
 ```

`bar-ingress` from the namespace `bar-namespace`:

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: bar-ingress
  namespace: bar-namespace
  annotations:
    kubernetes.io/ingress.class: "nginx"
spec:
  tls:
  - hosts:
    - bar.example.com
  rules:
  - host: bar.example.com
    http:
      paths:
      - path: /
        backend:
          serviceName: bar-service
          servicePort: 80
```

Because we don't reference any TLS secret in the `tls` section (there is no `secretName` field) in both Ingress resources, NGINX will use the wildcard secret specified in the `-wildcard-tls-secret` command-line argument.

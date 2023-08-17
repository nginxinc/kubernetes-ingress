# Rewrites Support

You can configure NGINX to rewrite the URI of a request before sending it to the application. For example, `/tea/green`
can be rewritten to `/green`.

## Syntax

To configure URI rewriting you need to add the **nginx.org/rewrites** annotation to your Ingress resource definition.
The annotation syntax is as follows:

```yaml
nginx.org/rewrites: "serviceName=service1 rewrite=rewrite1[;serviceName=service2 rewrite=rewrite2;...]"
```

## Example

In the following example we load balance two applications that require URI rewriting:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/rewrites: "serviceName=tea-svc rewrite=/;serviceName=coffee-svc rewrite=/beans/"
spec:
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea/
        pathType: Prefix
        backend:
          service:
            name: tea-svc
            port:
              number: 80
      - path: /coffee/
        pathType: Prefix
        backend:
          service:
            name: coffee-svc
            port:
              number: 80
```

Below are the examples of how the URI of requests to the *tea-svc* are rewritten (Note that the `/tea` requests are
redirected to `/tea/`).

- `/tea/` -> `/`
- `/tea/abc` -> `/abc`

Below are the examples of how the URI of requests to the *coffee-svc* are rewritten (Note that the `/coffee` requests
are redirected to `/coffee/`).

- `/coffee/` -> `/beans/`
- `/coffee/abc` -> `/beans/abc`

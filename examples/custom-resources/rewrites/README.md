# Rewrites Support

You can configure NGINX to rewrite the URI of a request before sending it to the application. For example, `/tea/green`
can be rewritten to `/green`.

To configure URI rewriting you need to use the
[ActionProxy](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#action-proxy)
of the [VirtualServer or
VirtualServerRoute](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/).

## Example with a Prefix Path

In the following example we load balance two applications that require URI rewriting using prefix-based URI matching:

```yaml
apiVersion: k8s.nginx.org/v1
kind: VirtualServer
metadata:
  name: cafe
spec:
  host: cafe.example.com
  upstreams:
  - name: tea
    service: tea-svc
    port: 80
  - name: coffee
    service: coffee-svc
    port: 80
  routes:
  - path: /tea/
    action:
      proxy:
        upstream: tea
        rewritePath: /
  - path: /coffee
    action:
      proxy:
        upstream: coffee
        rewritePath: /beans
```

Below are the examples of how the URI of requests to the *tea-svc* are rewritten (Note that the `/tea` requests are
redirected to `/tea/`).

- `/tea/` -> `/`
- `/tea/abc` -> `/abc`

Below are the examples of how the URI of requests to the *coffee-svc* are rewritten.

- `/coffee` -> `/beans`
- `/coffee/` -> `/beans/`
- `/coffee/abc` -> `/beans/abc`

## Example with Regular Expressions

If the route path is a regular expression instead of a prefix or an exact match, the `rewritePath` can include capture
groups with `$1-9`, for example:

```yaml
apiVersion: k8s.nginx.org/v1
kind: VirtualServer
metadata:
  name: cafe
spec:
  host: cafe.example.com
  upstreams:
  - name: tea
    service: tea-svc
    port: 80
  routes:
  - path: ~ /tea/?(.*)
    action:
      proxy:
        upstream: tea
        rewritePath: /$1
```

Note the capture group in the path `(.*)` is used in the rewritePath `/$1`. This is needed in order to pass the rest of
the request URI (after `/tea`).

Below are the examples of how the URI of requests to the *tea-svc* are rewritten.

- `/tea` -> `/`
- `/tea/` -> `/`
- `/tea/abc` -> `/abc`

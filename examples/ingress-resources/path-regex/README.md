# Support for path regular expressions

NGINX and NGINX Plus support regular expression modifiers for [location](https://nginx.org/en/docs/http/ngx_http_core_module.html#location) directive.

The NGINX Ingress Controller provides the following annotations for configuring regular expression support:

* Optional: ```nginx.com/path-regex: "case_sensitive"``` -- specifies a preceding regex modifier to be case sensitive (`~*`).
* Optional: ```nginx.com/path-regex: "case_insensitive"``` -- specifies a preceding regex modifier to be case sensitive (`~`).
* Optional: ```nginx.com/path-regex: "exact"``` -- specifies exact match preceding modifier (`=`).

## Example 1: Case Sensitive RegEx

In the following example you enable path regex annotation ``nginx.com/path-regex`` and set its value to `case_sensitive`.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/path-regex: "case_sensitive"
spec:
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea/[A-Z0-9]
        backend:
          serviceName: tea-svc
          servicePort: 80
      - path: /coffee/[A-Z0-9]
        backend:
          serviceName: coffee-svc
          servicePort: 80
```

Corresponding NGINX config file snippet:

```bash
...

  location ~ "^/tea/[A-Z0-9]" {

    set $service "tea-svc";
    status_zone "tea-svc";

...

  location ~ "^/coffee/[A-Z0-9]" {

    set $service "coffee-svc";
    status_zone "coffee-svc";

...
```

## Example 2: Case Insensitive RegEx

In the following example you enable path regex annotation ``nginx.com/path-regex`` and set its value to `case_insensitive`.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/path-regex: "case_insensitive"
spec:
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea/[A-Z0-9]
        backend:
          serviceName: tea-svc
          servicePort: 80
      - path: /coffee/[A-Z0-9]
        backend:
          serviceName: coffee-svc
          servicePort: 80
```

Corresponding NGINX config file snippet:

```bash
...

  location ~* "^/tea/[A-Z0-9]" {

    set $service "tea-svc";
    status_zone "tea-svc";

...

  location ~* "^/coffee/[A-Z0-9]" {

    set $service "coffee-svc";
    status_zone "coffee-svc";

...
```

## Example 3: Exact RegEx

In the following example you enable path regex annotation ``nginx.com/path-regex`` and set its value to `exact` match.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/path-regex: "exact"
spec:
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea/
        backend:
          serviceName: tea-svc
          servicePort: 80
      - path: /coffee/
        backend:
          serviceName: coffee-svc
          servicePort: 80
```

Corresponding NGINX config file snippet:

```bash
...

  location = "/tea" {

    set $service "tea-svc";
    status_zone "tea-svc";

...

  location = "/coffee" {

    set $service "coffee-svc";
    status_zone "coffee-svc";
...
```

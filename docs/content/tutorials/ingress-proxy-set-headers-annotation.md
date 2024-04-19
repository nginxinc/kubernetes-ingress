---
docs: DOCS-1451
doctypes:
- concept
title: Ingresses Custom Headers Using Proxy-Set-Headers Annotation
toc: true
weight: 1800
---

This document describes how to customize Ingress and Mergeable Ingress types with proxy-set-headers annotations.
## Customizing NGINX Ingress Controller with Proxy-Set-Headers Annotations


## Standard Ingress Type

In this example, you will use the `nginx.org/proxy-set-headers` annotations to set custom proxy headers.

Start by modifying `cafe-ingress.yaml` metadata to add the annotation section and configure
the ``nginx.org/proxy-set-headers`` annotation.

`cafe-ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/proxy-set-headers: "X-Forwarded-ABC: master, ABC"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea
        pathType: Prefix
        backend:
          service:
            name: tea-svc
            port:
              number: 80
      - path: /coffee
        pathType: Prefix
        backend:
          service:
            name: coffee-svc
            port:
              number: 80
```

Create the Ingress 

```shell
kubectl create -f cafe-ingress.yaml
```

This will add the custom proxy headers to the NGINX config in the ``tea`` and ``coffee`` locations:

```nginx
...

 location /coffee {
  ...
  proxy_set_header X-Forwarded-Master "master";
  proxy_set_header ABC $http_abc;
  ...

...
location /tea {
  ...
  proxy_set_header X-Forwarded-Master "master";
  proxy_set_header ABC $http_abc;
  ...

...

```

## Mergeable Ingress Type

This section explains how to deploy and configure Mergeable Ingress Type.

First, you will deploy a Master Ingress and two Minion Ingresses. Then, you will configure them with `proxy-set-headers` annotations.

Create a Master Ingress.

`cafe-master.yaml`

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress-master
  annotations:
    nginx.org/mergeable-ingress-type: "master"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
  rules:
  - host: cafe.example.com
```

```shell
kubectl create -f cafe-master.yaml
```

Verify the Master Ingress was created

```shell
kubectl get ingress cafe-ingress-master

NAME                  CLASS   HOSTS              ADDRESS   PORTS     AGE
cafe-ingress-master   nginx   cafe.example.com             80, 443   8s
```

Create the first Ingress Minion.

`tea-minion.yaml`

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress-tea-minion
  annotations:
    nginx.org/mergeable-ingress-type: "minion"
spec:
  ingressClassName: nginx
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea
        pathType: Prefix
        backend:
          service:
            name: tea-svc
            port:
              number: 80
```

```shell
kubectl create -f tea-minion.yaml

ingress.networking.k8s.io/cafe-ingress-tea-minion created
```

Verify the Minion was created:

```shell
kubectl get ingress cafe-ingress-tea-minion

NAME                      CLASS   HOSTS              ADDRESS   PORTS   AGE
cafe-ingress-tea-minion   nginx   cafe.example.com             80      10s
```

Create the second Ingress Minion.

`tea-minion.yaml`

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress-tea-minion
  annotations:
    nginx.org/mergeable-ingress-type: "minion"
spec:
  ingressClassName: nginx
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea
        pathType: Prefix
        backend:
          service:
            name: tea-svc
            port:
              number: 80
```

```shell
kubectl create -f tea-minion.yaml

ingress.networking.k8s.io/cafe-ingress-tea-minion created
```

Verify the Minion Ingress was created:

```shell
kubectl get ingress cafe-ingress-tea-minion

NAME                      CLASS   HOSTS              ADDRESS   PORTS   AGE
cafe-ingress-tea-minion   nginx   cafe.example.com             80      5m21s
```

You created a Master Ingress and two Minion Ingresses. Minion Ingresses are defined with two paths: `/tea` and `/coffee`.

In the following steps, you will be adding custom headers and values using the `proxy-set-headers` annotation to each location as well as master.

Update the Master Ingress:

- add `proxy-set-headers` annotation a custom headers and a custom value eg. `X-Forwarded-ABC` with the value `master`


`cafe-master.yaml`

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress-master
  annotations:
    nginx.org/mergeable-ingress-type: "master"
    nginx.org/proxy-set-headers: "X-Forwaded-ABC: master"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
  rules:
  - host: cafe.example.com
```

Create the Ingress

```shell
kubectl create -f cafe-master.yaml
```

Verify the Master Ingress was created

```shell
kubectl get ingress cafe-ingress-master

NAME                  CLASS   HOSTS              ADDRESS   PORTS     AGE
cafe-ingress-master   nginx   cafe.example.com             80, 443   8s
```

Note that the `proxy-set-headers` annotation in master applies to both paths defined.
The annotation will show up in both locations and if one minion has the same header name as master, it will override it.

See [examples](https://github.com/nginxinc/kubernetes-ingress/blob/main/examples/ingress-resources/proxy-set-headers/README.md) for more information

Follow the steps below to add a different custom proxy header on the First Minion Ingress.

Update the Minion Ingress `Tea`:

- add `proxy-set-headers` annotation a header with a custom value eg. `X-Forwarded-Tea` with the value `value1` and different header `abc123` with the default value

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress-tea-minion
  annotations:
    nginx.org/mergeable-ingress-type: "minion"
    nginx.org/proxy-set-headers: "X-Forwaded-Tea: value1, abc123"
spec:
  ingressClassName: nginx
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea
        pathType: Prefix
        backend:
          service:
            name: tea-svc
            port:
              number: 80
```

Apply the changes:

```shell
kubectl apply -f tea-minion.yaml
```

Looking at the updated list of annotations, we can see the new `proxy-set-headers` annotation was added.

It adds the headers `X-Forwaded-Tea` with the value of `value1` and the header `abc123` with a default value in the path `/tea` alongside master.
Updated path (location) in the NGINX config file: 

```nginx
...
...
location /tea {
  ...
  
  proxy_set_header X-Forwarded-ABC "master";
  proxy_set_header X-Forwarded-ABC "value1";
  proxy_set_header abc123 $http_abc123;
  
  ...
...
```

Note that the `proxy-set-headers` annotation applies only to paths defined on the corresponding Minion Ingress.
The paths defined in the second Minion (`coffee`) are not modified.

Follow the steps below to add a different custom proxy header on the Second Minion Ingress.

Update the Minion Ingress `Coffee`:

- add `proxy-set-headers` annotation using a single header and value eg. `X-Forwarded-Coffee` with a custom value

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress-coffee-minion
  annotations:
    nginx.org/mergeable-ingress-type: "minion"
    nginx.org/proxy-set-headers: "X-Forwarded-Coffee: mocha"
spec:
  ingressClassName: nginx
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /coffee
        pathType: Prefix
        backend:
          service:
            name: coffee-svc
            port:
              number: 80
```

Apply changes to the Minion Ingress:

```shell
kubectl apply -f coffee-minion.yaml

ingress.networking.k8s.io/cafe-ingress-coffee-minion created
```

The new annotation `nginx.org/proxy-set-headers` was added.

It adds the headers `X-Forwaded-Coffee` with the value of `mocha` in the path `/coffee` alongside master. 
Updated path (location) in the NGINX config file: 

```nginx
...
...
location /coffee {
  ...

  proxy_set_header X-Forwarded-ABC "master";
  proxy_set_header X-Forwarded-Coffee "mocha";
  
  ...
...

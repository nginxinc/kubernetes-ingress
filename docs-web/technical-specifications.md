# NGINX Ingress Controller Technical Specifications

## Supported NGINX Ingress Controller Versions

We advise users to run the most recent release of the NGINX Ingress Controller, and we issue software updates to the most recent release. We provide technical support for F5 customers who are using the most recent version of the NGINX Ingress Controller, and any version released within two years of the current release.

## Supported Kubernetes Versions

We explicitly test the NGINX Ingress Controller on a range of Kubernetes platforms at each release, and the [release notes](/nginx-ingress-controller/releases) list which platforms were tested. We will provide technical support for the NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and which passes the [Kubernetes conformance tests](https://www.cncf.io/certification/software-conformance/).

## Supported Docker Images

We provide the following Docker images, which include NGINX/NGINX Plus bundled with the Ingress Controller binary.

### Images with NGINX

All images include NGINX 1.21.0.
The supported architecture is x86-64.

```eval_rst
.. list-table::
    :header-rows: 1

    * - Name
      - Base image
      - Third-party modules
      - DockerHub image
    * - Debian-based image
      - ``nginx:1.21.0``, which is based on ``debian:buster-slim``
      -
      - ``nginx/nginx-ingress:1.11.3``
    * - Alpine-based image
      - ``nginx:1.21.0-alpine``, which is based on ``alpine:3.13``
      -
      - ``nginx/nginx-ingress:1.11.3-alpine``
    * - Debian-based image with Opentracing
      - ``nginx:1.21.0``, which is based on ``debian:buster-slim``
      - OpenTracing API for C++ 1.5.1, NGINX plugin for OpenTracing, C++ OpenTracing binding for Jaeger 0.4.2
      -
    * - Ubi-based image
      - ``registry.access.redhat.com/ubi8/ubi:8.3``
      -
      - ``nginx/nginx-ingress:1.11.3-ubi``
```

### Images with NGINX Plus

All images include NGINX Plus R23.
The supported architecture is x86-64.

NGINX Plus images are not available through DockerHub.

```eval_rst
.. list-table::
    :header-rows: 1

    * - Name
      - Base image
      - Third-party modules
    * - Alpine-based image
      - ``alpine:3.13``
      -
    * - Debian-based image
      - ``debian:buster-slim``
      -
    * - Debian-based image with Opentracing
      - ``debian:buster-slim``
      - NGINX Plus OpenTracing module, C++ OpenTracing binding for Jaeger 0.4.2
    * - Ubi-based image
      - ``registry.access.redhat.com/ubi8/ubi:8.3``
      -
    * - Debian-based image with App Protect
      - ``debian:buster-slim``
      - NGINX Plus App Protect module
    * - Ubi-based image with App Protect
      - ``registry.access.redhat.com/ubi7/ubi``
      - NGINX Plus App Protect module
```

### Custom Images

You can customize an existing Dockerfile or use it as a reference to create a new one, which is necessary for the following cases:

* Choosing a different base image.
* Installing additional NGINX modules.

## Supported Helm Versions

The Ingress Controller supports installation via Helm 3.0+.

## Recommended Hardware

See the [Sizing guide](https://www.nginx.com/resources/datasheets/nginx-ingress-controller-kubernetes-sizing-guide/) for recommendations.

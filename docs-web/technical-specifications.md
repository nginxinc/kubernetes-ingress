# NGINX Ingress Controller Technical Specifications

## Supported NGINX Ingress Controller Versions

We advise users to run the most recent release of the NGINX Ingress Controller, and we issue software updates to the most recent release. We provide technical support for F5 customers who are using the most recent version of the NGINX Ingress Controller, and any version released within two years of the current release.

## Supported Kubernetes Versions

We explicitly test the NGINX Ingress Controller on a range of Kubernetes platforms at each release, and the [release notes](/nginx-ingress-controller/releases) list which platforms were tested. We will provide technical support for the NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and which passes the [Kubernetes conformance tests](https://www.cncf.io/certification/software-conformance/).

## Supported Docker Images

We provide the following Docker images, which include NGINX/NGINX Plus bundled with the Ingress Controller binary.

### Images with NGINX

All images include NGINX 1.23.2.
The supported architectures are amd64, arm64, ppc64le and s390x.

```eval_rst
.. list-table::
    :header-rows: 1

    * - Name
      - Base image
      - Third-party modules
      - DockerHub image
    * - Debian-based image
      - ``nginx:1.23.2``, which is based on ``debian:bullseye-slim``
      -
      - ``nginx/nginx-ingress:1.12.5``
    * - Alpine-based image
      - ``nginx:1.23.2-alpine``, which is based on ``alpine:3.15``
      -
      - ``nginx/nginx-ingress:1.12.5-alpine``
    * - Debian-based image with Opentracing
      - ``nginx:1.23.2``, which is based on ``debian:bullseye-slim``
      - NGINX OpenTracing module, OpenTracing library, OpenTracing tracers for Jaeger, Zipkin and Datadog
      -
    * - Ubi-based image
      - ``redhat/ubi8``
      -
      - ``nginx/nginx-ingress:1.12.5-ubi``
```

### Images with NGINX Plus

All images include NGINX Plus R27.
The supported architecture is amd64.

NGINX Plus images are not available through DockerHub.

```eval_rst
.. list-table::
    :header-rows: 1

    * - Name
      - Base image
      - Third-party modules
    * - Alpine-based image
      - ``alpine:3.15``
      -
    * - Debian-based image
      - ``debian:bullseye-slim``
      -
    * - Debian-based image with Opentracing
      - ``debian:bullseye-slim``
      - NGINX Plus OpenTracing module, OpenTracing tracers for Jaeger, Zipkin and Datadog
    * - Ubi-based image
      - ``redhat/ubi8``
      -
    * - Debian-based image with App Protect
      - ``debian:buster-slim``
      - NGINX Plus App Protect module
    * - Ubi-based image with App Protect
      - ``redhat/ubi8``
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

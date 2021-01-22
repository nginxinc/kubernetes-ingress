# NGINX Ingress Controller Technical Specifications

## Supported Kubernetes Versions

The NGINX Ingress Controller has been verified to run on the following Kubernetes versions:
* Kubernetes 1.15-1.19

## Supported Docker Images

We provide the following Docker images, which include NGINX/NGINX Plus bundled with the Ingress Controller binary. 

### Images with NGINX 

All images include NGINX 1.19.6.
The supported architecture is x86-64.

```eval_rst
.. list-table::
    :header-rows: 1

    * - Name
      - Dockerfile*
      - Base image
      - Third-party modules
      - DockerHub image
    * - Debian-based image
      - ``Dockerfile``
      - ``nginx:1.19.6``, which is based on ``debian:buster-slim``
      - 
      - ``nginx/nginx-ingress:1.10.0``
    * - Alpine-based image
      - ``DockerfileForAlpine``
      - ``nginx:1.19.6-alpine``, which is based on ``alpine:3.10``
      - 
      - ``nginx/nginx-ingress:1.10.0-alpine``
    * - Debian-based image with Opentracing
      - ``DockerfileWithOpentracing``
      - ``nginx:1.19.6``, which is based on ``debian:buster-slim``
      - OpenTracing API for C++ 1.5.1, NGINX plugin for OpenTracing, C++ OpenTracing binding for Jaeger 0.4.2 
      - 
    * - Ubi-based image
      - ``openshift/Dockerfile``
      - ``registry.access.redhat.com/ubi8/ubi:8.3``
      - 
      - ``nginx/nginx-ingress:1.10.0-ubi``
```
\* -- Dockerfile paths are relative to the ``build`` folder of the Ingress Controller git repo.

### Images with NGINX Plus

All images include NGINX Plus R23.
The supported architecture is x86-64.

NGINX Plus images are not available through DockerHub.

```eval_rst
.. list-table::
    :header-rows: 1

    * - Name
      - Dockerfile*
      - Base image
      - Third-party modules
    * - Debian-based image
      - ``DockerfileForPlus``
      - ``debian:buster-slim``
      - 
    * - Debian-based image with Opentracing
      - ``DockerfileWithOpentracingForPlus``
      - ``debian:buster-slim``
      - NGINX Plus OpenTracing module, C++ OpenTracing binding for Jaeger 0.4.2 
    * - Ubi-based image
      - ``openshift/DockerfileForPlus``
      - ``registry.access.redhat.com/ubi8/ubi:8.3``
      - 
    * - Debian-based image with App Protect
      - ``appprotect/DockerfileWithAppProtectForPlus``
      - ``debian:buster-slim``
      - NGINX Plus App Protect module
    * - Ubi-based image with App Protect
      - ``appprotect/DockerfileWithAppProtectForPlusForOpenShift``
      - ``registry.access.redhat.com/ubi7/ubi``
      - NGINX Plus App Protect module   
```

\* -- Dockerfile paths are relative to the ``build`` folder of the Ingress Controller git repo.

### Custom Images

You can customize an existing Dockerfile or use it as a reference to create a new one, which is necessary for the following cases:

* Choosing a different base image.
* Installing additional NGINX modules.

## Supported Helm Versions

The Ingress Controller supports installation via Helm 3.0+.

## Recommended Hardware

See the [Sizing guide](https://www.nginx.com/resources/datasheets/nginx-ingress-controller-kubernetes-sizing-guide/) for recommendations.

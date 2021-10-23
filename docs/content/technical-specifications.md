---
title: Technical Specifications
description:
weight: 2000
doctypes: ["concept"]
toc: true
---


## Supported NGINX Ingress Controller Versions

We advise users to run the most recent release of the NGINX Ingress Controller, and we issue software updates to the most recent release. We provide technical support for F5 customers who are using the most recent version of the NGINX Ingress Controller, and any version released within two years of the current release.

## Supported Kubernetes Versions

We explicitly test the NGINX Ingress Controller on a range of Kubernetes platforms at each release, and the [release notes](/nginx-ingress-controller/releases) list which platforms were tested. We will provide technical support for the NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and which passes the [Kubernetes conformance tests](https://www.cncf.io/certification/software-conformance/).

## Supported Docker Images

We provide the following Docker images, which include NGINX/NGINX Plus bundled with the Ingress Controller binary.

### Images with NGINX

All images include NGINX 1.21.3.

{{% table %}}
|Name | Base image | Third-party modules | DockerHub image | Architectures |
| ---| ---| ---| --- | --- |
|Alpine-based image | ``nginx:1.21.3-alpine``, which is based on ``alpine:3.14`` |  | ``nginx/nginx-ingress:2.0.2-alpine`` | arm/v7, arm64, amd64, ppc64le, s390x |
|Debian-based image | ``nginx:1.21.3``, which is based on ``debian:buster-slim`` |  | ``nginx/nginx-ingress:2.0.2`` | arm/v7, arm64, amd64, ppc64le, s390x |
|Debian-based image with Opentracing | ``nginx:1.21.3``, which is based on ``debian:buster-slim`` | NGINX OpenTracing module, OpenTracing library, OpenTracing tracers for Jaeger, Zipkin and Datadog | ``nginx/nginx-ingress:2.0.2-ot`` | arm/v7, arm64, amd64, ppc64le, s390x |
|Ubi-based image | ``redhat/ubi8-minimal`` |  | ``nginx/nginx-ingress:2.0.2-ubi`` | arm64, amd64 |
{{% /table %}}

### Images with NGINX Plus

NGINX Plus images include NGINX Plus R25.
The supported architecture is x86-64.

NGINX Plus images are available through the F5 Container registry `private-registry.nginx.com` - see [Using the NGINX IC Plus JWT token in a Docker Config Secret](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret) and [Pulling the NGINX Ingress Controller image](/nginx-ingress-controller/installation/pulling-ingress-controller-image).

{{% table %}}
|Name | Base image | Third-party modules | F5 Container Registry Image |
| ---| ---| --- | --- |
|Alpine-based image | ``alpine:3.14`` | NGINX Plus JavaScript module | `nginx-ic/nginx-plus-ingress:2.0.2-alpine` |
|Debian-based image | ``debian:buster-slim`` | NGINX Plus JavaScript module | `nginx-ic/nginx-plus-ingress:2.0.2` |
|Debian-based image with Opentracing | ``debian:buster-slim`` | NGINX Plus OpenTracing module, OpenTracing tracers for Jaeger, Zipkin and Datadog; NGINX Plus JavaScript module | `nginx-ic/nginx-plus-ingress:2.0.2-ot` |
|Debian-based image with App Protect | ``debian:buster-slim`` | NGINX Plus App Protect module; NGINX Plus JavaScript module | `nginx-ic-nap/nginx-plus-ingress:2.0.2` |
|Ubi-based image | ``redhat/ubi8-minimal`` | NGINX Plus JavaScript module | `nginx-ic/nginx-plus-ingress:2.0.2-ubi` |
|Ubi-based image with App Protect | ``registry.access.redhat.com/ubi7/ubi`` | NGINX Plus App Protect module; NGINX Plus JavaScript module | `nginx-ic-nap/nginx-plus-ingress:2.0.2-ubi` |
{{% /table %}}

We also provide NGINX Plus images through the AWS Marketplace. Please see [Using the AWS Marketplace Ingress Controller Image](/nginx-ingress-controller/installation/using-aws-marketplace-image.md) for details on how to set up the required IAM resources in your EKS cluster.

{{% table %}}
|Name | Base image | Third-party modules | AWS Marketplace Link |
| ---| ---| --- | --- |
|Debian-based image | ``debian:buster-slim`` | NGINX Plus JavaScript module | [NGINX Ingress Controller](https://aws.amazon.com/marketplace/pp/prodview-fx3faxl7zqeau) |
|Debian-based image with App Protect | ``debian:buster-slim`` | NGINX Plus App Protect module; NGINX Plus JavaScript module | [NGINX Ingress Controller with NGINX App Protect](https://aws.amazon.com/marketplace/pp/prodview-vnrnxbf6u3nra) |
{{% /table %}}

### Custom Images

You can customize an existing Dockerfile or use it as a reference to create a new one, which is necessary for the following cases:

* Choosing a different base image.
* Installing additional NGINX modules.

## Supported Helm Versions

The Ingress Controller supports installation via Helm 3.0+.

## Recommended Hardware

See the [Sizing guide](https://www.nginx.com/resources/datasheets/nginx-ingress-controller-kubernetes-sizing-guide/) for recommendations.

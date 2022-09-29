---
title: Technical Specifications
description: "NGINX Ingress Controller Technical Specifications."
weight: 2000
doctypes: ["concept"]
toc: true
docs: "DOCS-617"
---


## Supported NGINX Ingress Controller Versions

We advise users to run the most recent release of the NGINX Ingress Controller, and we issue software updates to the most recent release. We provide technical support for F5 customers who are using the most recent version of the NGINX Ingress Controller, and any version released within two years of the current release.

Additionally, the current release version is 2.x which is compatible with the Kubernetes Ingress v1 API. Therefore Kubernetes 1.19 and later.
The 1.12 release supports the Ingress v1beta1 API and continues to receive security fixes to support those unable to upgrade to Kubernetes 1.19 or later. The v1beta1 Ingress API was deprecated with Kubernetes release 1.19 and removed with the Kubernetes 1.22 release.

## Supported Kubernetes Versions

We explicitly test the NGINX Ingress Controller (NIC) on a range of Kubernetes platforms at each release, and the [release notes](/nginx-ingress-controller/releases) list which platforms were tested. We will provide technical support for the NGINX Ingress Controller (NIC) on any Kubernetes platform that is currently supported by its provider and which passes the [Kubernetes conformance tests](https://www.cncf.io/certification/software-conformance/).

{{% table %}}
| NIC Version | Supported Kubernetes Version | NIC Helm Chart Version | NIC Operator Version | NGINX / NGINX Plus version |
| --- | --- | --- | --- | --- |
| 2.4.0 | 1.25 - 1.19 | 0.15.0 | 1.2.0 | 1.23.1 / R27 |
| 2.3.1 | 1.24 - 1.19 | 0.14.1 | 1.1.0 | 1.23.1 / R27 |
| 2.2.2 | 1.23 - 1.19 | 0.13.2 | 1.0.0 | 1.21.6 / R26 |
| 2.1.2 | 1.23 - 1.19 | 0.12.1 | 0.5.1 | 1.21.6 / R26 |
| 2.0.3 | 1.22 - 1.19 | 0.11.3 | 0.4.0 | 1.21.3 / R25 |
| 1.12.4 | 1.21 - 1.16 | 0.10.4 | 0.3.0 | 1.21.6 / R26 |
| 1.11.3 | 1.20 - 1.16 | 0.9.0 | 0.2.0 | 1.21.0 / R23 P1 |
| 1.10.1 | 1.19 - 1.16 | 0.8.0 | 0.1.0 | 1.19.8 / R23 |
| 1.9.1 | 1.18 - 1.16 | 0.7.1 | 0.0.7 | 1.19.3 / R22 |
| 1.8.1 |  | 0.6.0 | 0.0.6 | 1.19.2 / R22 |
| 1.7.2 |  | 0.5.1 | 0.0.4 | 1.19.0 / R22 |
| 1.6.3 |  | 0.4.3 | -- | 1.17.9 / R21 |

## Supported Docker Images

We provide the following Docker images, which include NGINX/NGINX Plus bundled with the Ingress Controller binary.

### Images with NGINX

All images include NGINX 1.23.1.

{{% table %}}
|Name | Base image | Third-party modules | DockerHub image | Architectures |
| ---| ---| ---| --- | --- |
|Alpine-based image | ``nginx:1.23.1-alpine``, which is based on ``alpine:3.16`` | NGINX OpenTracing module, OpenTracing library, OpenTracing tracers for Jaeger, Zipkin and Datadog | ``nginx/nginx-ingress:2.4.0-alpine`` | arm/v7, arm64, amd64, ppc64le, s390x |
|Debian-based image | ``nginx:1.23.1``, which is based on ``debian:bullseye-slim`` | NGINX OpenTracing module, OpenTracing library, OpenTracing tracers for Jaeger, Zipkin and Datadog | ``nginx/nginx-ingress:2.4.0`` | arm/v7, arm64, amd64, ppc64le, s390x |
|Ubi-based image | ``redhat/ubi8`` |  | ``nginx/nginx-ingress:2.4.0-ubi`` | arm64, amd64, s390x |
{{% /table %}}

### Images with NGINX Plus

NGINX Plus images include NGINX Plus R27.

NGINX Plus images are available through the F5 Container registry `private-registry.nginx.com` - see [Using the NGINX IC Plus JWT token in a Docker Config Secret](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret) and [Pulling the NGINX Ingress Controller image](/nginx-ingress-controller/installation/pulling-ingress-controller-image).

{{% table %}}
|Name | Base image | Third-party modules | F5 Container Registry Image | Architectures |
| ---| ---| --- | --- | --- |
|Alpine-based image | ``alpine:3.16`` | NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic/nginx-plus-ingress:2.4.0-alpine` | arm64, amd64 |
|Debian-based image | ``debian:bullseye-slim`` | NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic/nginx-plus-ingress:2.4.0` | arm64, amd64 |
|Debian-based image with App Protect WAF | ``debian:buster-slim`` | NGINX Plus App Protect WAF, JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic-nap/nginx-plus-ingress:2.4.0` | amd64 |
|Debian-based image with App Protect DoS | ``debian:bullseye-slim`` | NGINX Plus App Protect DoS, JavaScript module and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic-dos/nginx-plus-ingress:2.4.0` | amd64 |
|Debian-based image with App Protect WAF and DoS | ``debian:buster-slim`` | NGINX Plus App Protect WAF, DoS, JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic-nap-dos/nginx-plus-ingress:2.4.0` | amd64 |
|Ubi-based image | ``redhat/ubi8`` | NGINX Plus JavaScript module | `nginx-ic/nginx-plus-ingress:2.4.0-ubi` | arm64, amd64, s390x |
|Ubi-based image with App Protect WAF | ``redhat/ubi8`` | NGINX Plus App Protect WAF and JavaScript modules | `nginx-ic-nap/nginx-plus-ingress:2.4.0-ubi` | amd64 |
|Ubi-based image with App Protect DoS | ``redhat/ubi8`` | NGINX Plus App Protect DoS and JavaScript modules | `nginx-ic-dos/nginx-plus-ingress:2.4.0-ubi` | amd64 |
|Ubi-based image with App Protect WAF and DoS | ``redhat/ubi8`` | NGINX Plus App Protect WAF, DoS and JavaScript modules | `nginx-ic-nap-dos/nginx-plus-ingress:2.4.0-ubi` | amd64 |
{{% /table %}}

We also provide NGINX Plus images through the AWS Marketplace. Please see [Using the AWS Marketplace Ingress Controller Image](/nginx-ingress-controller/installation/using-aws-marketplace-image/) for details on how to set up the required IAM resources in your EKS cluster.

{{% table %}}
|Name | Base image | Third-party modules | AWS Marketplace Link |
| ---| ---| --- | --- |
|Debian-based image | ``debian:bullseye-slim`` | NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | [NGINX Ingress Controller](https://aws.amazon.com/marketplace/pp/prodview-fx3faxl7zqeau) |
|Debian-based image with App Protect | ``debian:buster-slim`` | NGINX Plus App Protect, JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | [NGINX Ingress Controller with NGINX App Protect](https://aws.amazon.com/marketplace/pp/prodview-vnrnxbf6u3nra) |
{{% /table %}}

### Custom Images

You can customize an existing Dockerfile or use it as a reference to create a new one, which is necessary for the following cases:

* Choosing a different base image.
* Installing additional NGINX modules.

## Supported Helm Versions

The Ingress Controller supports installation via Helm 3.0+.

## Recommended Hardware

See the [Sizing guide](https://www.nginx.com/resources/datasheets/nginx-ingress-controller-kubernetes-sizing-guide/) for recommendations.

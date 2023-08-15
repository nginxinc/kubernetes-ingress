---
title: Technical Specifications
description: "NGINX Ingress Controller Technical Specifications."
weight: 2200
doctypes: ["concept"]
toc: true
docs: "DOCS-617"
---


## Supported NGINX Ingress Controller Versions

We advise users to run the most recent release of the NGINX Ingress Controller, and we issue software updates to the most recent release. We provide technical support for F5 customers who are using the most recent version of the NGINX Ingress Controller, and any version released within two years of the current release.

The 3.0.0 release supports `discovery.k8s.io/v1` API version of EndpointSlice, available from Kubernetes 1.21 onwards.
The 2.4.2 release is compatible with the Kubernetes Ingress v1 API. Therefore Kubernetes 1.19 and later.
The 1.12 release supports the Ingress v1beta1 API and continues to receive security fixes to support those unable to upgrade to Kubernetes 1.19 or later. The v1beta1 Ingress API was deprecated with Kubernetes release 1.19 and removed with the Kubernetes 1.22 release.

## Supported Kubernetes Versions

We explicitly test the NGINX Ingress Controller (NIC) on a range of Kubernetes platforms at each release, and the [release notes](/nginx-ingress-controller/releases) list which platforms were tested. We will provide technical support for the NGINX Ingress Controller (NIC) on any Kubernetes platform that is currently supported by its provider and which passes the [Kubernetes conformance tests](https://www.cncf.io/certification/software-conformance/).

{{% table %}}
| NIC Version | Supported Kubernetes Version | NIC Helm Chart Version | NIC Operator Version | NGINX / NGINX Plus version |
| --- | --- | --- | --- | --- |
| 3.2.0 | 1.27 - 1.22 | 0.18.0 | 1.5.0 | 1.25.1 / R29 |
| 3.1.1 | 1.26 - 1.22 | 0.17.1 | 1.4.2 | 1.23.4 / R29 |
| 3.0.2 | 1.26 - 1.21 | 0.16.2 | 1.3.1 | 1.23.3 / R28 |
| 2.4.2 | 1.25 - 1.19 | 0.15.2 | 1.2.1 | 1.23.2 / R28 |
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
{{% /table %}}

## Supported Docker Images

We provide the following Docker images, which include NGINX/NGINX Plus bundled with the Ingress Controller binary.

### Images with NGINX

All images include NGINX 1.25.1.

{{% table %}}
|Name | Base image | Third-party modules | DockerHub image | Architectures |
| ---| ---| ---| --- | --- |
|Alpine-based image | ``nginx:1.25.1-alpine``, which is based on ``alpine:3.17`` | NGINX OpenTracing module, OpenTracing library, OpenTracing tracers for Jaeger, Zipkin and Datadog | ``nginx/nginx-ingress:3.2.0-alpine`` | arm/v7, arm64, amd64, ppc64le, s390x |
|Debian-based image | ``nginx:1.25.1``, which is based on ``debian:12-slim`` | NGINX OpenTracing module, OpenTracing library, OpenTracing tracers for Jaeger, Zipkin and Datadog | ``nginx/nginx-ingress:3.2.0`` | arm/v7, arm64, amd64, ppc64le, s390x |
|Ubi-based image | ``nginxcontrib/nginx:1.25.1-ubi``, which is based on ``redhat/ubi9-minimal`` |  | ``nginx/nginx-ingress:3.2.0-ubi`` | arm64, amd64, ppc64le, s390x |
{{% /table %}}

### Images with NGINX Plus

NGINX Plus images include NGINX Plus R29.

NGINX Plus images are available through the F5 Container registry `private-registry.nginx.com` - see [Using the NGINX IC Plus JWT token in a Docker Config Secret](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret) and [Pulling the NGINX Ingress Controller image](/nginx-ingress-controller/installation/pulling-ingress-controller-image).

{{% table %}}
|Name | Base image | Third-party modules | F5 Container Registry Image | Architectures |
| ---| ---| --- | --- | --- |
|Alpine-based image | ``alpine:3.18`` | NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic/nginx-plus-ingress:3.2.0-alpine` | arm64, amd64 |
|Alpine-based image with FIPS inside | ``alpine:3.18`` | NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog, FIPS module and OpenSSL configuration | `nginx-ic/nginx-plus-ingress:3.2.0-alpine-fips` | arm64, amd64 |
|Debian-based image | ``debian:12-slim`` | NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic/nginx-plus-ingress:3.2.0` | arm64, amd64 |
|Debian-based image with NGINX App Protect WAF | ``debian:11-slim`` | NGINX App Protect WAF, NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic-nap/nginx-plus-ingress:3.2.0` | amd64 |
|Debian-based image with NGINX App Protect DoS | ``debian:11-slim`` | NGINX App Protect DoS, NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic-dos/nginx-plus-ingress:3.2.0` | amd64 |
|Debian-based image with NGINX App Protect WAF and DoS | ``debian:11-slim`` | NGINX App Protect WAF and DoS, NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | `nginx-ic-nap-dos/nginx-plus-ingress:3.2.0` | amd64 |
|Ubi-based image | ``redhat/ubi9-minimal`` | NGINX Plus JavaScript module | `nginx-ic/nginx-plus-ingress:3.2.0-ubi` | arm64, amd64, s390x |
|Ubi-based image with NGINX App Protect WAF | ``redhat/ubi8`` | NGINX App Protect WAF and NGINX Plus JavaScript module | `nginx-ic-nap/nginx-plus-ingress:3.2.0-ubi` | amd64 |
|Ubi-based image with NGINX App Protect DoS | ``redhat/ubi8`` | NGINX App Protect DoS and NGINX Plus JavaScript module | `nginx-ic-dos/nginx-plus-ingress:3.2.0-ubi` | amd64 |
|Ubi-based image with NGINX App Protect WAF and DoS | ``redhat/ubi8`` | NGINX App Protect WAF and DoS, NGINX Plus JavaScript module | `nginx-ic-nap-dos/nginx-plus-ingress:3.2.0-ubi` | amd64 |
{{% /table %}}

We also provide NGINX Plus images through the AWS Marketplace. Please see [Using the AWS Marketplace Ingress Controller Image](/nginx-ingress-controller/installation/using-aws-marketplace-image/) for details on how to set up the required IAM resources in your EKS cluster.

{{% table %}}
|Name | Base image | Third-party modules | AWS Marketplace Link | Architectures |
| ---| ---| --- | --- | --- |
|Debian-based image | ``debian:12-slim`` | NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | [F5 NGINX Ingress Controller](https://aws.amazon.com/marketplace/pp/prodview-fx3faxl7zqeau) | amd64 |
|Debian-based image with NGINX App Protect WAF | ``debian:11-slim`` | NGINX App Protect WAF, NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | [F5 NGINX Ingress Controller with F5 NGINX App Protect WAF](https://aws.amazon.com/marketplace/pp/prodview-vnrnxbf6u3nra) | amd64 |
|Debian-based image with NGINX App Protect DoS | ``debian:11-slim`` | NGINX App Protect DoS, NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | [F5 NGINX Ingress Controller with F5 NGINX App Protect WAF and DoS](https://aws.amazon.com/marketplace/pp/prodview-yltaqwzwrnhco) | amd64 |
|Debian-based image with NGINX App Protect WAF and DoS | ``debian:11-slim`` | NGINX App Protect WAF and DoS, NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | [F5 NGINX Ingress Controller with F5 NGINX App Protect DoS](https://aws.amazon.com/marketplace/pp/prodview-sghjw2csktega) | amd64 |
{{% /table %}}

We also provide NGINX Plus images through the Google Cloud Marketplace. Please see [Using GCP Marketplace Ingress Controller](/nginx-ingress-controller/installation/using-gcp-marketplace-package/) for details on how use them.

{{% table %}}
|Name | Base image | Third-party modules | GCP Marketplace Link | Architectures |
| ---| ---| --- | --- | --- |
|Debian-based image | ``debian:11-slim`` | NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | [F5 NGINX Ingress Controller](https://console.cloud.google.com/marketplace/product/f5-7626-networks-public/nginx-ingress-plus) | amd64 |
|Debian-based image with NGINX App Protect DoS | ``debian:11-slim`` | NGINX App Protect DoS, NGINX Plus JavaScript and OpenTracing modules, OpenTracing tracers for Jaeger, Zipkin and Datadog | [F5 NGINX Ingress Controller w/ F5 NGINX App Protect DoS](https://console.cloud.google.com/marketplace/product/f5-7626-networks-public/nginx-ingress-plus-dos) | amd64 |
{{% /table %}}

### Custom Images

You can customize an existing Dockerfile or use it as a reference to create a new one, which is necessary for the following cases:

- Choosing a different base image.
- Installing additional NGINX modules.

## Supported Helm Versions

The Ingress Controller supports installation via Helm 3.0+.

## Recommended Hardware

See the [Sizing guide](https://www.nginx.com/resources/datasheets/nginx-ingress-controller-kubernetes-sizing-guide/) for recommendations.

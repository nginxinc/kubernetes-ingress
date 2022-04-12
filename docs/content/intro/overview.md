---
title: Overview
description: Overview of the NGINX Ingress Controller
weight: 100
doctypes: [""]
toc: true
docs: "DOCS-612"
---



The NGINX Ingress Controller an implementation of a Kubernetes Ingress Controller for NGINX and NGINX Plus.

## What is the Ingress?

The Ingress is a Kubernetes resource that lets you configure an HTTP load balancer for applications running on Kubernetes, represented by one or more [Services](https://kubernetes.io/docs/concepts/services-networking/service/). Such a load balancer is necessary to deliver those applications to clients outside of the Kubernetes cluster.

The Ingress resource supports the following features:
* **Content-based routing**:
    * *Host-based routing*. For example, routing requests with the host header `foo.example.com` to one group of services and the host header `bar.example.com` to another group.
    * *Path-based routing*. For example, routing requests with the URI that starts with `/serviceA` to service A and requests with the URI that starts with `/serviceB` to service B.
* **TLS/SSL termination** for each hostname, such as `foo.example.com`.

See the [Ingress User Guide](https://kubernetes.io/docs/user-guide/ingress/) to learn more about the Ingress resource.

## What is the Ingress Controller?

The Ingress Controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources. The load balancer can be a software load balancer running in the cluster or a hardware or cloud load balancer running externally. Different load balancers require different Ingress Controller implementations.

In the case of NGINX, the Ingress Controller is deployed in a pod along with the load balancer.

## NGINX Ingress Controller

NGINX Ingress Controller works with both NGINX and NGINX Plus and supports the standard Ingress features - content-based routing and TLS/SSL termination.

Additionally, several NGINX and NGINX Plus features are available as extensions to the Ingress resource via annotations and the ConfigMap resource. In addition to HTTP, NGINX Ingress Controller supports load balancing Websocket, gRPC, TCP and UDP applications. See [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource) and [Annotations](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations) docs to learn more about the supported features and customization options.

As an alternative to the Ingress, NGINX Ingress Controller supports the VirtualServer and VirtualServerRoute resources. They enable use cases not supported with the Ingress resource, such as traffic splitting and advanced content-based routing. See [VirtualServer and VirtualServerRoute Resources doc](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources).

TCP, UDP and TLS Passthrough load balancing is also supported. See the [TransportServer resource doc](/nginx-ingress-controller/configuration/transportserver-resource/).

---
title: "Glossary"
description:
weight: 10000
menu:
  docs:
    parent: NGINX Ingress Controller
docs: "DOCS-1446"
---

{{<custom-styles>}}

<style>
h2 {
  border-top: 1px solid #ccc;
  padding-top:20px;
}
</style>

## Ingress {#ingress}

_Ingress_ refers to an _Ingress Resource_, a Kubernetes API object which allows access to [Services](https://kubernetes.io/docs/concepts/services-networking/service/) within a cluster. They are managed by an [Ingress Controller]({{< relref "glossary.md#ingress-controller">}}).

_Ingress_ resources enable the following functionality:

- **Load balancing**, extended through the use of Services
- **Content-based routing**, using hosts and paths
- **TLS/SSL termination**, based on hostnames

For additional information, please read the official [Kubernetes Ingress Documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/).

## Ingress Controller {#ingress-controller}

*Ingress Controllers* are applications within a Kubernetes cluster that enable [Ingress]({{< relref "glossary.md#ingress">}}) resources to function. They are not automatically deployed with a Kubernetes cluster, and can vary in implementation based on intended use, such as load balancing algorithms for Ingress resources.

[How NGINX Ingress Controller is Designed]({{< relref "overview/design">}}) explains the technical details of the F5 NGINX Ingress Controller.

---
docs: DOCS-606
doctypes:
- ''
title: Running Multiple NGINX Ingress Controllers
toc: true
weight: 400
---

This document describes how to run multiple NGINX Ingress Controllers.

It explains the following topics:

- Ingress class concept.
- How to run NGINX Ingress Controller in the same cluster with another Ingress Controller, such as an Ingress Controller for a cloud HTTP load balancer, and prevent any conflicts between the Ingress Controllers.
- How to run multiple NGINX Ingress Controllers.

{{< note >}}In this document we refer to the [Ingress]({{< relref "/configuration/ingress-resources/basic-configuration.md" >}}), [VirtualServer]({{< relref "/configuration/virtualserver-and-virtualserverroute-resources.md#virtualserver-specification" >}}), [VirtualServerRoute]({{< relref "/configuration/virtualserver-and-virtualserverroute-resources.md#virtualserverroute-specification" >}}), and [TransportServer]({{< relref "/configuration/transportserver-resource.md" >}}) resources as "configuration resources".{{< /note >}}

## Ingress class

Thanks to the Ingress class concept, multiple Ingress Controllers can coexist in one cluster. The Ingress class has the following characteristics:

- Every Ingress Controller must only handle Ingress resources for its particular class.
- Ingress resources need to have the `ingressClassName` field set to the value of the class of the Ingress Controller the user wants to use.
- VirtualServer, VirtualServerRoute, Policy, and TransportServer resources need to have the `ingressClassName` field set to the value of the class of the Ingress Controller the user wants to use.

### Configuring Ingress class

The default Ingress class of NGINX Ingress Controller is `nginx`, which means that it only handles configuration resources with the Ingress class set to `nginx`. You can customize the class through the `-ingress-class` command-line argument.

{{< note >}}- If the class of an Ingress resource is not set, Kubernetes will set it to the class of the default Ingress Controller. To make the Ingress Controller the default one, the `ingressclass.kubernetes.io/is-default-class` property must be set on the IngressClass resource. To learn more, see Step 3 *Create an IngressClass resource* of the [Create Common Resources]({{< relref "installation/installing-nic/installation-with-manifests.md#create-common-resources" >}}) section.
- For VirtualServer, VirtualServerRoute, Policy and TransportServer resources, NGINX Ingress Controller will always handle resources with an empty class.{{< /note >}}

## Running NGINX Ingress Controller and another Ingress Controller

It is possible to run NGINX Ingress Controller and an Ingress Controller for another load balancer in the same cluster. This is often the case if you create your cluster through a cloud provider's managed Kubernetes service that by default might include the Ingress Controller for the HTTP load balancer of the cloud provider, and you want to use NGINX Ingress Controller.

To make sure that NGINX Ingress Controller handles specific configuration resources, update those resources with the class set to `nginx` (or the Ingress class value that you configured in NGINX Ingress Controller).

## Running multiple NGINX Ingress Controllers

When running NGINX Ingress Controller, you have the following options with regards to which configuration resources it handles:

- Cluster-wide Ingress Controller (default): NGINX Ingress Controller handles configuration resources created in any namespace of the cluster. As NGINX is a high-performance load balancer capable of serving many applications at the same time, this option is used by default in our installation manifests and Helm chart.
- Defined-namespace Ingress Controller: You can configure the Ingress Controller to handle configuration resources only from particular namespaces, which is controlled through the `-watch-namespace` command-line argument. This can be useful if you want to use different NGINX Ingress Controllers for different applications, both in terms of isolation and/or operation.
- Ingress Controller for Specific Ingress Class: This option works in conjunction with either of the options above. You can further customize which configuration resources are handled by the Ingress Controller by configuring the class of the Ingress Controller and using that class in your configuration resources. The [Configuring Ingress Class](#configuring-ingress-class) section above explains where.

These options allow you to run multiple NGINX Ingress Controllers, each handling a different set of configuration resources.

{{< see-also >}}[Command-line arguments]({{< relref "configuration/global-configuration/command-line-arguments" >}}){{< /see-also >}}

{{< note >}}All the mentioned command-line arguments are also available as parameters in the [Helm chart]({{< relref "installation/installing-nic/installation-with-helm" >}}).{{< /note >}}

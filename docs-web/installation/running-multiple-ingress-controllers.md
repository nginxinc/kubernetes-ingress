# Running Multiple Ingress Controllers

This document explains the following topics:
* Ingress class concept.
* How to run NGINX Ingress Controller in the same cluster with another Ingress Controller, such as an Ingress Controller for a cloud HTTP load balancer, and prevent any conflicts between the Ingress Controllers.
* How to run multiple NGINX Ingress Controllers.

## Ingress Class

The smooth coexistence of multiple Ingress Controllers in one cluster is provided by the Ingress class concept, which mandates the following:
* Every Ingress Controller must only handle Ingress resources for its particular class. 
* Ingress resources should be annotated with the `kubernetes.io/ingress.class` annotation set to the value, which corresponds to the class of the Ingress Controller the user wants to use. 
* Virtual Server and Virtual Server Route resources should have the `class` field set to the value, which corresponds to the class of the Ingress Controller the user wants to use.

### Configuring Ingress Class

The default Ingress class of NGINX Ingress Controller is `nginx`, which means that it only handles Ingress resources with the `kubernetes.io/ingress.class` annotation set to `nginx`. You can customize the class through the `-ingress-class` command-line argument.

**Note**: By default, if the `kubernetes.io/ingress.class` annotation is not set in an Ingress resource, the Ingress Controller will handle the resource. This is controlled via the `-use-ingress-class-only` argument.

## Running NGINX Ingress Controller and Another Ingress Controller

It is possible to run NGINX Ingress Controller and an Ingress Controller for another load balancer in the same cluster. This is often the case if you create your cluster through a cloud provider managed Kubernetes service that by default might include the Ingress Controller for the HTTP load balancer of the cloud provider, and you want to use NGINX Ingress Controller.

To make sure that NGINX Ingress Controller handles particular Ingress resources, annotate those Ingress resources with the `kubernetes.io/ingress.class` set to `nginx` or the value that you configured.


## Running Multiple NGINX Ingress Controllers

When running NGINX Ingress Controller, you have the following options with regards to which Ingress resources it handles:
* **Cluster-wide Ingress Controller (default)**. The Ingress Controller handles Ingress resources created in any namespace of the cluster. As NGINX is a high-performance load balancer capable of serving many applications at the same time, this option is used by default in our installation manifests and Helm chart.
* **Single-namespace Ingress Controller**. You can configure the Ingress Controller to handle Ingress resources only from a particular namespace, which is controlled through the `-watch-namespace` command-line argument. This can be useful if you want to use different NGINX Ingress Controllers for different applications, both in terms of isolation and/or operation.
* **Ingress Controller for Specific Ingress Class**. This option works in conjunction with either of the options above. You can further customize which Ingress resources are handled by the Ingress Controller by configuring the class of the Ingress Controller and using that class in your Ingress resources. See the section [Configuring Ingress Class](#configuring-ingress-class).

Considering the options above, you can run multiple NGINX Ingress Controllers, each handling a different set of Ingress resources.

## See Also

* [Command-line arguments](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments)

**Note**: all mentioned command-line arguments are also available as the parameters in the [Helm chart](/nginx-ingress-controller/installation/installation-with-helm).


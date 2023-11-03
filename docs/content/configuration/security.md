---
title: Security
description: "NGINX Ingress Controller security recommendations."
weight: 1500
doctypes: [""]
toc: true
docs: "DOCS-597"
---


The security of the Ingress Controller is paramount to the success of our Users, however, the Ingress Controller is deployed by a User in their environment, and as such, the User takes responsibility
for securing a deployment of the Ingress Controller.
We strongly recommend every User read and understand the following security concerns.

## Kubernetes

We recommend the Kubernetes [guide to securing a cluster](https://kubernetes.io/docs/tasks/administer-cluster/securing-a-cluster/).
In addition, the following relating more specifically to Ingress Controller.

### RBAC and Service Account

The Ingress Controller is deployed within a Kubernetes environment, this environment must be secured.
Kubernetes uses [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) to control the resources and operations available to different types of users.
The Ingress Controller requires a service account which is configured using RBAC.
We strongly recommend using the RBAC configuration provided in our deployment configurations.
It is configured with the least amount of privilege required for the Ingress Controller to work.

We strongly recommend inspecting the RBAC configuration in the deployment file or Helm chart to understand what access the Ingress Controller service account has and to which resources.
For example, by default the service account has access to all Secret resources in the cluster.

### Certificates and Privacy Keys

Secrets are required by the Ingress Controller for some configurations.
[Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) are stored by Kubernetes unencrypted by default.
We strongly recommend configuring Kubernetes to store these Secrets encrypted at rest.
Kubernetes has [documentation](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/) on how to configure this.

## Ingress Controller

### Recommended Secure Defaults

We recommend the following for the most secure configuration:

- If Prometheus metrics are [enabled](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-prometheus-metrics),
   we recommend [configuring HTTPS](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-prometheus-tls-secret) for Prometheus.

### Snippets

Snippets allow you to insert raw NGINX config into different contexts of NGINX configuration and are supported for [Ingress](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-snippets/), [VirtualServer/VirtualServerRoute](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#using-snippets), and [TransportServer](/nginx-ingress-controller/configuration/transportserver-resource/#using-snippets) resources. Additionally, the [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource#snippets-and-custom-templates) resource configures snippets globally.

Snippets are disabled by default. To use snippets, set the [`enable-snippets`](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments#cmdoption-enable-snippets) command-line argument. Note that for the ConfigMap resource, snippets are always enabled.

### Configure root filesystem as read-only
>
> **Note**: This feature is available for both the NGINX and NGINX Plus editions. NGINX AppProtect WAF and NGINX AppProtect DoS are not yet supported by this feature.

The F5 Nginx Ingress Controller (NIC) has various protections against attacks, such as running the service as non-root to avoid changes to files. An additional industry best practice is having root filesystems set as read-only so that the attack surface is further reduced by limiting changes to binaries and libraries.

Currently, we do not set read-only root filesystem as default. Instead, this is an opt-in feature available on the [Helm Chart](/nginx-ingress-controller/installation/installation-with-helm/#configuration)
via `controller.readOnlyRootFilesystem`.

If you prefer to use manifests instead of Helm, you can use the following manifest to enable this feature:

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/read-only-fs/deploy.yaml
```

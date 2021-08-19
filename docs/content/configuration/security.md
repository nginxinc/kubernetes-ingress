---
title: Security
description: 
weight: 1500
doctypes: [""]
toc: true
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
We strongly recommend using the [RBAC configuration](https://github.com/nginxinc/kubernetes-ingress/blob/master/deployments/rbac/rbac.yaml) provided in our standard deployment configuration. 
It is configured with the least amount of privilege required for the Ingress Controller to work.

We strongly recommend inspecting the RBAC configuration (for [manifests installation](https://github.com/nginxinc/kubernetes-ingress/blob/master/deployments/rbac/rbac.yaml) 
or for [helm](https://github.com/nginxinc/kubernetes-ingress/blob/master/deployments/helm-chart/templates/rbac.yaml)) 
to understand what access the Ingress Controller service account has and to which resources. 
For example, by default the service account has access to all Secret resources in the cluster.

### Certificates and Privacy Keys
Secrets are required by the Ingress Controller for some configurations. 
[Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) are stored by Kubernetes unencrypted by default. 
We strongly recommend configuring Kubernetes to store these Secrets encrypted at rest. 
Kubernetes has [documentation](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/) on how to configure this.

## Ingress Controller

### Recommended Secure Defaults
We recommend the following for the most secure configuration:
 * If Prometheus metrics are [enabled](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-prometheus-metrics), 
   we recommend [configuring HTTPS](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-prometheus-tls-secret) for Prometheus.

### Snippets
[Snippets](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-snippets/)
Snippets will be disabled by default in the future. 
Be sure to understand the implications of configurations you provide through the Snippets capability.

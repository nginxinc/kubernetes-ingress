---
docs: DOCS-597
doctypes:
- ''
title: Security recommendations
toc: true
weight: 1500
---

NGINX Ingress Controller follows Kubernetes best practices. A good starting point is the official Kubernetes documentation for [Securing a Cluster](https://kubernetes.io/docs/tasks/administer-cluster/securing-a-cluster/). 

This page outlines configuration specific to NGINX Ingress Controller you may require, including links to examples in the [GitHub repository](https://github.com/nginxinc/kubernetes-ingress/tree/release-3.5).

## RBAC and Service Accounts

Kubernetes uses [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) to control the resources and operations available to different types of users. 

NGINX Ingress Controller requires RBAC to configure a [ServiceUser](https://kubernetes.io/docs/concepts/security/service-accounts/#default-service-accounts), and provides least privilege access in its standard deployment configurations:

- [Helm](https://github.com/nginxinc/kubernetes-ingress/blob/v3.5.0/deployments/rbac/rbac.yam)
- [Manifests](https://github.com/nginxinc/kubernetes-ingress/blob/v3.5.0/deployments/rbac/rbac.yaml)

By default, the ServiceAccount has access to all Secret resources in the cluster.

## Secrets

[Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) are required by NGINX Ingress Controller for certificates and privacy keys, which Kubernetes stores unencrypted by default. We recommend following the [Kubernetes documentation](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/) to store these Secrets using at-rest encryption.

## Prometheus

If Prometheus metrics are [enabled]({{< relref "/logging-and-monitoring/prometheus.md" >}}), we recommend [using HTTPS]({{< relref "configuration/global-configuration/command-line-arguments.md#cmdoption-prometheus-tls-secret" >}}).

## Configure root filesystem as read-only

{{< caution >}}
 This feature is not compatible with [NGINX App Protect WAF](https://docs.nginx.com/nginx-app-protect-waf/) or [NGINX App Protect DoS](https://docs.nginx.com/nginx-app-protect-dos/).
{{< /caution >}}

NGINX Ingress Controller is designed to be resilient against attacks in various ways, such as running the service as non-root to avoid changes to files. We recommend setting filesystems to read-only so that the attack surface is further reduced by limiting changes to binaries and libraries.

This is not enabled by default, but can be enabled with **Helm** using the [**controller.readOnlyRootFilesystem**]({{< relref "installation/installing-nic/installation-with-helm.md#configuration" >}}) argument.

For **Manifests**, uncomment the following sections of the deployment:

- `readOnlyRootFilesystem: true`
- The entire **volumeMounts** section
- The entire **initContainers** section

The block below shows the code you will look for:

```yaml
#      volumes:
#      - name: nginx-etc
#        emptyDir: {}
#      - name: nginx-cache
#        emptyDir: {}
#      - name: nginx-lib
#        emptyDir: {}
#      - name: nginx-log
#        emptyDir: {}
.
.
.
#          readOnlyRootFilesystem: true
.
.
.
#        volumeMounts:
#        - mountPath: /etc/nginx
#          name: nginx-etc
#        - mountPath: /var/cache/nginx
#          name: nginx-cache
#        - mountPath: /var/lib/nginx
#          name: nginx-lib
#        - mountPath: /var/log/nginx
#          name: nginx-log
```

## Snippets

Snippets allow you to insert raw NGINX config into different contexts of NGINX configuration and are supported for [Ingress](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-snippets/), [VirtualServer/VirtualServerRoute](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#using-snippets), and [TransportServer](/nginx-ingress-controller/configuration/transportserver-resource/#using-snippets) resources. Additionally, the [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource#snippets-and-custom-templates) resource configures snippets globally.

Snippets are disabled by default. To use snippets, set the [`enable-snippets`](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments#cmdoption-enable-snippets) command-line argument. Note that for the ConfigMap resource, snippets are always enabled.
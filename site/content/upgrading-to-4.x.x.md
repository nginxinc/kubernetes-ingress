---
description: Guide on upgrading from v3.4.x and above to v4.x.x
docs: DOCS-000
doctypes:
- concept
title: Upgrading from v3.4.x and above to v4.x.x
toc: true
weight: 2100
noindex: true
---

{{< note >}}
If you are using a version of NGINX Ingress Controller prior to `v3.4` (Helm Chart `v1.1.x`) please upgrade to v3.4 first due to breaking changes.
{{</ note >}}
1. If using NGINX Ingress Controller resources with an `apiVersion` of `k8s.nginx.org/v1alpha1`, prior to upgrading to `v4.x` (Helm Chart `v2.x`) , **please ensure** that they are updated to `apiVersion: k8s.nginx.org/v1`
{{< important >}}
If a resource of `kind: GlobalConfiguration`, `kind: Policy` or `kind: TransportServer` is deployed as `apiVersion: k8s.nginx.org/v1alpha1`, it will be **deleted** during the upgrade process
{{</ important >}}

    Below is an example of how a Policy resource would change. Please ensure the same is done for GlobalConfiguration and TransportServer. 

    {{<tabs name="resource-version-update">}}


{{%tab name="Before"%}}

```yaml
apiVersion: k8s.nginx.org/v1alpha1
kind: Policy
metadata:
  name: rate-limit-policy
spec:
  rateLimit:
    rate: 1r/s
    key: ${binary_remote_addr}
    zoneSize: 10M
```


{{% /tab %}}

{{%tab name="After"%}}
```yaml
apiVersion: k8s.nginx.org/v1
kind: Policy
metadata:
  name: rate-limit-policy
spec:
  rateLimit:
    rate: 1r/s
    key: ${binary_remote_addr}
    zoneSize: 10M
```
{{% /tab %}}

    {{</tabs>}}

1. Read the [Create License Secret]({{< relref "installation/installing-nic/create-license-secret">}}) topic to set up your NGINX Plus license.
1. Usage reporting through the Cluster Connector is no longer required, and is now native to NGINX. 
1. If your NGINX Ingress Controller installation is in an "air-gapped" environment, [usage reports can be sent to NGINX Instance Manager]({{< relref "installation/installing-nic/create-license-secret/#nim">}}). 
1. Configure Structured Logging

    {{<tabs name="structured logging">}}

{{%tab name="Helm"%}}

The previous Helm value `controller.logLevel` has been changed from an integer to a string. Options include: trace, debug, info, warning, error and fatal.

Logs can also be rendered in different formats using the `controller.logFormat` key. Options include: glog, json and text. This only applies to NGINX Ingress Controller logs, not NGINX logs.

```yaml
controller:
    logLevel: info
    logFormat: json 
```
{{% /tab %}}

{{%tab name="Manifests"%}}

The command line argument `-v` has been replaced with `-log-level`. It has been changed from an integer to a string. Options include: trace, debug, info, warning, error and fatal.

Logs can also be rendered in different formats using the `-log-format` command line argument. Options include: glog, json and text. This only applies to NGINX Ingress Controller logs, not NGINX logs.

The command line argument `-logtostderr` has been deprecated.

```yaml
args:
    - -log-level=info
    - -log-format=json
```

{{% /tab %}}

    {{</tabs>}}

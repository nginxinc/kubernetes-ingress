---
title: Using with F5 BIG-IP
description: |
  Learn how to use NGINX Ingress Controller with F5 IngressLink to configure your F5 BIG-IP device.
weight: 1800
doctypes: ["concept"]
toc: true
---


F5 IngressLink is the integration between NGINX Ingress Controller and [F5 Container Ingress Services](https://clouddocs.f5.com/containers/v2/) (CIS) that configures an F5 BIG-IP device as a load balancer for NGINX Ingress Controller pods.

## Configuration

### 1. Install the Ingress Controller with the Integration Enabled

This step depends on how you install the Ingress Controller: using [Manifests](/nginx-ingress-controller/installation/installation-with-manifests) or the [Helm chart](/nginx-ingress-controller/installation/installation-with-helm).

#### Manifests Installation

1. Create a service for the Ingress Controller pods for ports 80 and 443. For example:
    ```yaml
    apiVersion: v1
    kind: Service
    metadata:
      name: nginx-ingress-ingresslink
      namespace: nginx-ingress
      labels:
        app: ingresslink
    spec:
      ports:
      - port: 80
        targetPort: 80
        protocol: TCP
        name: http
      - port: 443
        targetPort: 443
        protocol: TCP
        name: https
      selector:
        app: nginx-ingress
    ```
    Note the label `app: ingresslink`. We will use it in the Step 2.
1. In the [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource), enable the PROXY protocol, which the BIG-IP system will use to pass the client IP and port information to NGINX. For the  `set-real-ip-from` key, use the subnet of the IP, which the BIG-IP system uses to send traffic to NGINX:
    ```yaml
    proxy-protocol: "True"
    real-ip-header: "proxy_protocol"
    set-real-ip-from: "0.0.0.0/0"
    ```
1. Deploy the Ingress Controller with additional [command-line arguments](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments):
    ```yaml
    args:
    - -ingresslink=nginx-ingress
    - -report-ingress-status
    . . .
    ```
    where `ingresslink` references the name of the IngressLink resource from Step 2, and `report-ingress-status` enables [reporting Ingress statuses](/nginx-ingress-controller/configuration/global-configuration/reporting-resources-status#ingress-resources).

#### Helm Installation

Install a helm release with the following values that replicate the Manifest installation above:
```yaml
controller:
  config:
    entries:
      proxy-protocol: "True"
      real-ip-header: "proxy_protocol"
      set-real-ip-from: "0.0.0.0/0"
  reportIngressStatus:
    ingressLink: nginx-ingress
  service:
    type: ClusterIP
    externalTrafficPolicy: Cluster
    extraLabels:
      app: ingresslink
```
We will use the values for the parameters `ingressLink` and `extraLabels` in Step 2. For the  `set-real-ip-from` key, use the subnet of the IP, which the BIG-IP system uses to send traffic to NGINX.

### 2. Configure CIS

To enable the integration, the F5 CIS must be deployed in the cluster and configured to support the integration. Follow the instructions on the [CIS documentation portal](https://clouddocs.f5.com/containers/latest/userguide/ingresslink/#configuring-ingresslink). Ensure that:
* The name of the IngressLink resource is the same as in Step 1 -- `nginx-ingress`.
* The selector in the IngressLink resource is the same as the Service labels configured in Step 1 -- `app: ingresslink`.
* The IngressLink must belong to the same namespace as the Ingress Controller pod -- `nginx-ingress` or the namespace used for installing the Helm chart.

---
title: "Product Telemetry"
description: "Learn why NGINX Ingress Controller collects telemetry, and understand how and what it gathers."
weight: 500
toc: true
---

## Overview

NGINX Ingress Controller collects product telemetry data to allow its developers to understand how it's deployed and configured by users.
This data is used to triage development work, prioritizing features and functionality that will benefit the most people.


Product telemetry is enabled by default, collected once every 24 hours. It's then sent to a service managed by F5 over HTTPS.

{{< note >}}
If you would prefer to avoid sending any telemetry data, you can [opt-out](#opt-out) when installing NGINX Ingress Controller.
{{< /note >}}

## Data Collected

These are the data points collected and reported by NGINX Ingress Controller:
- **Project Name** The name of the software, which will be labelled `NIC`.
- **Project Version** NGINX Ingress Controller version.
- **Project Architecture** The architecture of the kubernetes environment. (e.g. amd64, arm64, etc...)
- **Cluster ID** A unique identifier of the kubernetes cluster that the NGINX Ingress Controller is deployed to.
- **Cluster Version** The version of the Kubernetes cluster.
- **Cluster Platform** The platform that the kubernetes cluster is operating on. (e.g. eks, aks,  etc...)
- **Cluster Node Count** The number of nodes in the cluster that NGINX Ingress Controller is deployed to.
- **Installation ID** Used to identify a unique installation of NGINX Ingress Controller.
- **VirtualServers** The number of VirtualServer resources managed by NGINX Ingress Controller.
- **VirtualServerRoutes** The number of VirtualServerRoute resources managed by NGINX Ingress Controller.
- **TransportServers** The number of TransportServer resources managed by NGINX Ingress Controller.
- **Replicas** Number of Deployment replicas, or Daemonset instances.

## Opt out

Product telemetry can be disabled when installing NGINX Ingress Controller.

### Helm


When installing or upgrading NGINX Ingress Controller with Helm, set the `controller.telemetry.enable` option to `false`
This can be set directly in the `values.yaml` file, or using the `--set` option

```shell
helm upgrade --install ... --set controller.telemetry.enable=false
```

### Manifests

When installing NGINX Ingress Controller with Manifests, set the `-enable-telemetry-reporting` flag to `false`

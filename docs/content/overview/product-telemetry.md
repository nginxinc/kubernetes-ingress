---
title: "Product Telemetry"
description: "Learn how and why NGINX Ingress Controller collects product telemetry data."
weight: 500
toc: true
---

## Overview

The NGINX Ingress Controller collects product telemetry data to allow its developers to better understand the way our users deploy and configure our product.
The goal is to use this data to triage development work and prioritize features and functionality that will provide the most value to our users.

## About Product Telemetry

Product telemetry is collected by default.
Data is collected once every 24 hours, and is sent to a service managed by F5 over HTTPS.
Personally identifiable information (PII) is **not** collected. 

> [!NOTE]  
> If you prefer to not have data collected, you can [opt-out](#opt-out) when installing NGINX Ingress Controller.

## Data Collected

Below is a list of current set data points that are collected and reported by the NGINX Ingress Controller:
- **Project Name** This is the name of the product. In this case `NIC`
- **Project Version** The version of the NGINX Ingress Controller.
- **Project Architecture** The architecture of the kubernetes environment. (e.g. amd64, arm64, etc...)
- **Cluster ID** A unique identifier of the kubernetes cluster that the NGINX Ingress Controller is deployed to.
- **Cluster Platform** The platform that the kubernetes cluster is operating on. (e.g. eks, aks,  etc...)
- **Installation ID** Used to identify a unique installation of the NGINX Ingress Controller.
- **VirtualServers** A count of the number of VirtualServer resources managed by the NGINX Ingress Controller.
- **VirtualServerRoutes** A count of the number of VirtualServerRoute resources managed by the NGINX Ingress Controller.
- **TransportServers** A count of the number of TransportServer resources managed by the NGINX Ingress Controller.

## Opt out

Collection and reporting of product telemetry can be switched off when installing the NGINX Ingress Controller.

### Helm

When installing/upgrading with Helm, set the `controller.telemetry.enable` option to `false`
This can be set directly in the `values.yaml` file, or using the `--set` option

```shell
helm upgrade --install ... --set controller.telemetry.enable=false
```

### Manifest

When installing with Manifest, set the `-enable-telemetry-reporting` flag to `false`

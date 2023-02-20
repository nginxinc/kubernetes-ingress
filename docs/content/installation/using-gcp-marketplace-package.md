---
title: Using GCP Marketplace Ingress Controller
description: "This document will walk you through the steps needed to deploy the NGINX Ingress Controller through the GCP Marketplace."
weight: 2400
doctypes: [""]
toc: true
docs: "DOCS-000"
---

This document will walk you through the steps needed to deploy and configure the NGINX Ingress Controller through the GCP Marketplace.

## Product overview.

The F5 NGINX Ingress Controller is an implementation of a Kubernetes Ingress Controller for NGINX and NGINX Plus.

Basic features include:
* Host-based routing. For example, routing requests with the host header foo.example.com to one group of services and the host header bar.example.com to another group.
* Path-based routing. For example, routing requests with the URI that starts with /serviceA to service A and requests with the URI that starts with /serviceB to service B.
* TLS/SSL termination for each hostname, such as foo.example.com.

## One-time setup

To quickly get the NGINX Ingress Controller up and running, follow our [Installation with Manifests](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/) guide.
This details how to locally build an image of the NGINX Ingress Controller as well as the necessary CustomResourceDefinitions (CRDs) required.

## Installation

### Install NGINX Ingress Controller to an existing GKE cluster

1. Deploy NGINX Ingress Controller from GUI

   Open [Google Cloud Console](https://console.cloud.google.com/) and navigate to **Kubernetes Engine > Applications**

2. Click on **DEPLOY FROM MARKETPLACE**, and search for *NGINX Ingress Controller*

   <!-- TODO Add an image of KIC options in marketplace once listing are approved -->
   > **NOTE**: Please verify that you select a *Premium Edition* controller image that has been published by *NGINX, Inc.*, and not a third-party.

   Choose the appropriate *NGINX Ingress Controller* image, and click **CONFIGURE**

3. Install to an existing cluster

   > **NOTE**: Click on **OR SELECT AN EXISTING CLUSTER** if you see the **CREATE NEW CLUSTER** button.

   {{< img title="Install to existing GKE cluster" src="./img/gke-existing-cluster.png" >}}

   Choose an **Existing Kubernetes Cluster** to target from the options provided.

   The *default* namespace will be selected automatically, but you can choose to **Create a namespace** and enter the **New namespace name**.

   The **App instance name** will be used as a prefix for all resources created by the deployment, and must be unique within the selected namespace. A default value will be generated and can be used as-is, or changed.

   Recommended options for NGINX Ingress Controller will be pre-selected, but can be adjusted as necessary.

   Click **DEPLOY** to launch NGINX Ingress Controller installation process.

   {{< img title="Install to existing GKE cluster" src="./img/gke-ingress-controller-application.png" >}}

   You can find the NGINX Ingress Controller *application* by navigating back to **Kubernetes Engine > Applications**

### Install NGINX Ingress Controller to a new GKE cluster

As an alternative to using an existing GKE cluster, GCP Marketplace can create a small zonal cluster for you during the installation process. Please note that you will need to resize the cluster to provide enough vCPU for NGINX Ingress Controller and your other applications.

1. Find NGINX Ingress Controller on the marketplace

   Open [Google Cloud Console](https://console.cloud.google.com/) and navigate to **Marketplace**, then search for *NGINX Ingress Controller*

   <!-- TODO Add an image of KIC options in marketplace once listing are approved -->
   > **NOTE**: Please verify that you select a *Premium Edition* controller image that has been published by *NGINX, Inc.*, and not a third-party.

   Choose the appropriate *NGINX Ingress Controller* image, and click **CONFIGURE**

2. Configure the new GKE cluster

   Choose the **zone**, **network**, and **subnetwork** that is appropriate for your new cluster from the options provided.

   {{< img title="Create a new GKE cluster" src="./img/gke-create-cluster.png" >}}

   Click **CREATE NEW CLUSTER**.

   {{< img title="New cluster creation in progress" src="./img/gke-creating-cluster.png" >}}

   After a short delay the cluster will be ready for step 3.

3. Complete the NGINX Ingress Controller deployment options

   {{< img title="Complete new GKE cluster installation" src="./img/gke-install-to-new-cluster.png" >}}

   The *default* namespace will be selected automatically, but you can choose to **Create a namespace** and enter the **New namespace name**.

   The **App instance name** will be used as a prefix for all resources created by the deployment, and must be unique within the selected namespace. A default value will be generated and can be used as-is, or changed.

   Recommended options for NGINX Ingress Controller will be pre-selected, but can be adjusted as necessary.

   Click **DEPLOY** to launch NGINX Ingress Controller installation process.

   {{< img title="Install to existing GKE cluster" src="./img/gke-ingress-controller-application.png" >}}

   You can find the NGINX Ingress Controller *application* by navigating back to **Kubernetes Engine > Applications**

## Configuration

The GCP Marketplace will deploy the NGINX Ingress Controller with a default configuration and an empty *ConfigMap*. These resources will all be named `<app-instance-name>-nginx-ingress`, where `<app-instance-name>` matches the value you provided during the [installation](#installation) step.

For example, if NGINX Ingress Controller was deployed to namespace `nginx-ingress` and with an **App instance name** of `nginx-ingress-plus` (as used in the examples above), the ConfigMap can be viewed with `kubectl`:

```
$ kubectl get configmap -n nginx-ingress nginx-ingress-plus-nginx-ingress -o yaml
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","data":null,"kind":"ConfigMap","metadata":{"annotations":{},"labels":{"app.kubernetes.io/instance":"nginx-ingress-plus","app.kubernetes.io/managed-by":"Helm","app.kubernetes.io/name":"nginx-ingress-plus-nginx-ingress","helm.sh/chart":"nginx-ingress-0.16.2"},"name":"nginx-ingress-plus-nginx-ingress","namespace":"nginx-ingress","ownerReferences":[{"apiVersion":"app.k8s.io/v1beta1","blockOwnerDeletion":true,"kind":"Application","name":"nginx-ingress-plus","uid":"5cbbebd8-df13-4001-bd65-9467405d9a9d"}]}}
  creationTimestamp: "2022-08-25T01:03:10Z"
  labels:
    app.kubernetes.io/instance: nginx-ingress-plus
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: nginx-ingress-plus-nginx-ingress
    helm.sh/chart: nginx-ingress-0.16.2
  name: nginx-ingress-plus-nginx-ingress
  namespace: nginx-ingress
  ownerReferences:
  - apiVersion: app.k8s.io/v1beta1
    blockOwnerDeletion: true
    kind: Application
    name: nginx-ingress-plus
    uid: 5cbbebd8-df13-4001-bd65-9467405d9a9d
  resourceVersion: "147519"
  uid: 3fa33891-7a30-4004-91bd-bd5d652e34a9
```

See the [Configuration](https://docs.nginx.com/nginx-ingress-controller/configuration/) documentation to modify the resources.

## Basic Usage

To setup a basic application that uses the NGINX Ingress Controller, see our [basic configuration](https://github.com/nginxinc/kubernetes-ingress/tree/main/examples/custom-resources/basic-configuration) example page.

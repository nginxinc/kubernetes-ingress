---
title: Installation with NGINX App Protect Dos
description: "This document provides an overview of the steps required to use NGINX App Protect Dos with your NGINX Ingress Controller deployment."
weight: 1800
doctypes: [""]
toc: true
docs: "DOCS-583"
---

> **Note**: The NGINX Kubernetes Ingress Controller integration with NGINX App Protect requires the use of NGINX Plus.

This document provides an overview of the steps required to use NGINX App Protect DoS with your NGINX Ingress Controller deployment. You can visit the linked documents to find additional information and instructions.

## Prerequisites

1. Make sure you have access to the Ingress Controller image:
    * For NGINX Plus Ingress Controller, see [here](/nginx-ingress-controller/installation/pulling-ingress-controller-image) for details on how to pull the image from the F5 Docker registry.
    * To pull from the F5 Container registry in your Kubernetes cluster, configure a docker registry secret using your JWT token from the MyF5 portal by following the instructions from [here](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret).
    * It is also possible to build your own image and push it to your private Docker registry by following the instructions from [here](/nginx-ingress-controller/installation/building-ingress-controller-image).
2. Clone the Ingress Controller repo:
    ```
    $ git clone https://github.com/nginxinc/kubernetes-ingress.git --branch v2.2.0
    $ cd kubernetes-ingress
    ```

## Create the namespace and service account

```bash
  kubectl apply -f common/ns-and-sa.yaml
```

## Install the App Protect DoS Arbitrator

- Deploy the app protect dos arbitrator
    ```bash
    kubectl apply -f deployment/appprotect-dos-arb.yaml
    kubectl apply -f service/appprotect-dos-arb-svc.yaml
    ```

## Build the Docker Image

Take the steps below to create the Docker image that you'll use to deploy NGINX Ingress Controller with App Protect DoS in Kubernetes.

- [Build the NGINX Ingress Controller image](/nginx-ingress-controller/installation/building-ingress-controller-image).

  When running the `make` command to build the image, be sure to use the `debian-image-dos-plus` target. For example:

    ```bash
    make debian-image-dos-plus PREFIX=<your Docker registry domain>/nginx-plus-ingress
    ```

- [Push the image to your local Docker registry](/nginx-ingress-controller/installation/building-ingress-controller-image/#building-the-image-and-pushing-it-to-the-private-registry).

## Install the Ingress Controller

Take the steps below to set up and deploy the NGINX Ingress Controller and App Protect DoS module in your Kubernetes cluster.

1. [Configure role-based access control (RBAC)](/nginx-ingress-controller/installation/installation-with-manifests/#1-configure-rbac).

   > **Important**: You must have an admin role to configure RBAC in your Kubernetes cluster.

2. [Create the common Kubernetes resources](/nginx-ingress-controller/installation/installation-with-manifests/#2-create-common-resources).
3. Enable the App Protect Dos module by adding the `enable-app-protect-dos` [cli argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-app-protect-dos) to your Deployment or DaemonSet file.
4. [Deploy the Ingress Controller](/nginx-ingress-controller/installation/installation-with-manifests/#3-deploy-the-ingress-controller).

For more information, see the [Configuration guide](/nginx-ingress-controller/app-protect-dos/configuration),the [NGINX Ingress Controller with App Protect DoS example for Ingress](https://github.com/nginxinc/kubernetes-ingress/tree/v2.2.0/examples/appprotect-dos) and the [NGINX Ingress Controller with App Protect DoS example for VirtualServer](https://github.com/nginxinc/kubernetes-ingress/tree/v2.2.0/examples/custom-resources/dos).

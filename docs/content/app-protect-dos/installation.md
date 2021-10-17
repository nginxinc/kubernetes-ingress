---
title: Installation with NGINX App Protect Dos
description:
weight: 1800
doctypes: [""]
toc: true
---

> **Note**: The NGINX Kubernetes Ingress Controller integration with NGINX App Protect requires the use of NGINX Plus.

This document provides an overview of the steps required to use NGINX App Protect Dos with your NGINX Ingress Controller deployment. You can visit the linked documents to find additional information and instructions.

## Install the app-protect-dos-arb

- Deploy the app protect dos arbitrator
    ```bash
    kubectl apply -f deployment/appprotect-dos-arb.yaml
    kubectl apply -f service/appprotect-dos-arb-svc.yaml
    ```

## Build the Docker Image

Take the steps below to create the Docker image that you'll use to deploy NGINX Ingress Controller with App Protect Dos in Kubernetes.

- [Build the NGINX Ingress Controller image](/nginx-ingress-controller/installation/building-ingress-controller-image).

  When running the `make` command to build the image, be sure to use the `debian-image-dos-plus` target. For example:

    ```bash
    make debian-image-dos-plus PREFIX=<your Docker registry domain>/nginx-plus-ingress
    ```

- [Push the image to your local Docker registry](/nginx-ingress-controller/installation/building-ingress-controller-image/#building-the-image-and-pushing-it-to-the-private-registry).

## Install the Ingress Controller

Take the steps below to set up and deploy the NGINX Ingress Controller and App Protect Dos module in your Kubernetes cluster.

1. [Configure role-based access control (RBAC)](/nginx-ingress-controller/installation/installation-with-manifests/#configure-rbac).

   > **Important**: You must have an admin role to configure RBAC in your Kubernetes cluster.

2. [Create the common Kubernetes resources](/nginx-ingress-controller/installation/installation-with-manifests/#create-common-resources).
3. Enable the App Protect Dos module by adding the `enable-app-protect-dos` [cli argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-app-protect-dos) to your Deployment or DaemonSet file.
5. [Deploy the Ingress Controller](/nginx-ingress-controller/installation/installation-with-manifests/#deploy-the-ingress-controller).

For more information, see the [Configuration guide](/nginx-ingress-controller/app-protect-dos/configuration) and the [NGINX Ingress Controller with App Protect Dos examples on GitHub](https://github.com/nginxinc/kubernetes-ingress/tree/v1.12.0/examples/appprotect-dos).

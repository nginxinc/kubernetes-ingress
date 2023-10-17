---
title: Installation with NGINX App Protect DoS
description: "This document provides an overview of the steps required to use NGINX App Protect DoS with your NGINX Ingress Controller deployment."
weight: 1800
doctypes: [""]
toc: true
docs: "DOCS-583"
---

> **Note**: The F5 NGINX Kubernetes Ingress Controller integration with F5 NGINX App Protect DoS requires the use of F5 NGINX Plus.

This document provides an overview of the steps required to use NGINX App Protect DoS with your NGINX Ingress Controller deployment. You can visit the linked documents to find additional information and instructions.

## Prerequisites

1. Make sure you have access to the Ingress Controller image:
    - For NGINX Plus Ingress Controller, see [here](/nginx-ingress-controller/installation/pulling-ingress-controller-image) for details on how to pull the image from the F5 Docker registry.
    - To pull from the F5 Container registry in your Kubernetes cluster, configure a docker registry secret using your JWT token from the MyF5 portal by following the instructions from [here](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret).
    - It is also possible to build your own image and push it to your private Docker registry by following the instructions from [here](/nginx-ingress-controller/installation/building-ingress-controller-image).
2. Clone the Ingress Controller repo:

    ```
    git clone https://github.com/nginxinc/kubernetes-ingress.git --branch v3.3.1
    cd kubernetes-ingress/deployments
    ```

## Install the App Protect DoS Arbitrator

### Helm Chart

The App Protect DoS Arbitrator can be installed using the [NGINX App Protect DoS Helm Chart](https://github.com/nginxinc/nap-dos-arbitrator-helm-chart).
If you have the NGINX Helm Repository already added, you can install the App Protect DoS Arbitrator by running the following command:

```console
helm install my-release-dos nginx-stable/nginx-appprotect-dos-arbitrator
```

### YAML Manifests

Alternatively, you can install the App Protect DoS Arbitrator using the YAML manifests provided in the Ingress Controller repo.

- Create the namespace and service account

```console
  kubectl apply -f common/ns-and-sa.yaml
```

- Deploy the app protect dos arbitrator

    ```console
    kubectl apply -f deployment/appprotect-dos-arb.yaml
    kubectl apply -f service/appprotect-dos-arb-svc.yaml
    ```

## Build the Docker Image

Take the steps below to create the Docker image that you'll use to deploy NGINX Ingress Controller with App Protect DoS in Kubernetes.

- [Build the NGINX Ingress Controller image](/nginx-ingress-controller/installation/building-ingress-controller-image).

  When running the `make` command to build the image, be sure to use the `debian-image-dos-plus` target. For example:

    ```console
    make debian-image-dos-plus PREFIX=<your Docker registry domain>/nginx-plus-ingress
    ```

    Alternatively, if you want to run on an [OpenShift](https://www.openshift.com/) cluster, use the `ubi-image-dos-plus` target.

    If you want to include the App Protect WAF module in the image, you can use the `debian-image-nap-dos-plus` target or the `ubi-image-nap-dos-plus` target for OpenShift.

- [Push the image to your local Docker registry](/nginx-ingress-controller/installation/building-ingress-controller-image/#building-the-image-and-pushing-it-to-the-private-registry).

## Install the Ingress Controller

Take the steps below to set up and deploy the NGINX Ingress Controller and App Protect DoS module in your Kubernetes cluster.

1. [Configure role-based access control (RBAC)](/nginx-ingress-controller/installation/installation-with-manifests/#1-configure-rbac).

   > **Important**: You must have an admin role to configure RBAC in your Kubernetes cluster.

2. [Create the common Kubernetes resources](/nginx-ingress-controller/installation/installation-with-manifests/#2-create-common-resources).
3. Enable the App Protect Dos module by adding the `enable-app-protect-dos` [cli argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-app-protect-dos) to your Deployment or DaemonSet file.
4. [Deploy the Ingress Controller](/nginx-ingress-controller/installation/installation-with-manifests/#3-deploy-the-ingress-controller).

For more information, see the [Configuration guide](/nginx-ingress-controller/app-protect-dos/configuration),the [NGINX Ingress Controller with App Protect DoS example for VirtualServer](https://github.com/nginxinc/kubernetes-ingress/tree/v3.3.1/examples/custom-resources/app-protect-dos) and the [NGINX Ingress Controller with App Protect DoS example for Ingress](https://github.com/nginxinc/kubernetes-ingress/tree/v3.3.1/examples/ingress-resources/app-protect-dos).

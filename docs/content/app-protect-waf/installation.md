---
title: Installation with NGINX App Protect WAF
description: "This document provides an overview of the steps required to use NGINX App Protect WAF with your NGINX Ingress Controller deployment."
weight: 1800
doctypes: [""]
toc: true
docs: "DOCS-579"
aliases: ["/app-protect/installation/"]
---

> **Note**: The NGINX Kubernetes Ingress Controller integration with NGINX App Protect WAF requires the use of NGINX Plus.

This document provides an overview of the steps required to use NGINX App Protect WAF with your NGINX Ingress Controller deployment. You can visit the linked documents to find additional information and instructions.

You can also [install the Ingress Controller with App Protect WAF by using Helm](/nginx-ingress-controller/installation/installation-with-helm/). Use the `controller.appprotect.*` parameters of the chart.

## Prerequisites

1. Make sure you have access to the Ingress Controller image:
    * For NGINX Plus Ingress Controller, see [here](/nginx-ingress-controller/installation/pulling-ingress-controller-image) for details on how to pull the image from the F5 Docker registry.
    * To pull from the F5 Container registry in your Kubernetes cluster, configure a docker registry secret using your JWT token from the MyF5 portal by following the instructions from [here](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret).
    * It is also possible to build your own image and push it to your private Docker registry by following the instructions from [here](/nginx-ingress-controller/installation/building-ingress-controller-image).
2. Clone the Ingress Controller repo:
    ```
    $ git clone https://github.com/nginxinc/kubernetes-ingress.git --branch v2.4.0
    $ cd kubernetes-ingress
    ```

## Build the Docker Image

Take the steps below to create the Docker image that you'll use to deploy NGINX Ingress Controller with App Protect in Kubernetes.

- [Build the NGINX Ingress Controller image](/nginx-ingress-controller/installation/building-ingress-controller-image).

    When running the `make` command to build the image, be sure to use the `debian-image-nap-plus` target. For example:

    ```bash
    make debian-image-nap-plus PREFIX=<your Docker registry domain>/nginx-plus-ingress
    ```
    Alternatively, if you want to run on an [OpenShift](https://www.openshift.com/) cluster, you can use the `ubi-image-nap-plus` target.

    If you want to include the App Protect DoS module in the image, you can use the `debian-image-nap-dos-plus` target or the `ubi-image-nap-dos-plus` target for OpenShift.

    If you intend to use [external references](https://docs.nginx.com/nginx-app-protect/configuration/#external-references) in NGINX App Protect WAF policies, you may want to provide a custom CA certificate to authenticate with the hosting server.
    In order to do so, place the `*.crt` file in the build folder and uncomment the lines that follow this comment:
    `#Uncomment the lines below if you want to install a custom CA certificate`

     > **Note**: [External References](/nginx-app-protect/configuration-guide/configuration/#external-references) in the Ingress Controller are deprecated and will not be supported in future releases.

    **Note**: In the event of a patch version of NGINX Plus being [released](/nginx/releases/), make sure to rebuild your image to get the latest version. The Dockerfile will use the latest available version of the [Attack Signatures](/nginx-app-protect/configuration/#attack-signatures) and [Threat Campaigns](/nginx-app-protect/configuration/#threat-campaigns) packages at the time of build. If your system is caching the Docker layers and not updating the packages, add `DOCKER_BUILD_OPTIONS="--no-cache"` to the `make` command.

- [Push the image to your local Docker registry](/nginx-ingress-controller/installation/building-ingress-controller-image/#building-the-image-and-pushing-it-to-the-private-registry).

## Install the Ingress Controller

Take the steps below to set up and deploy the NGINX Ingress Controller and App Protect WAF module in your Kubernetes cluster.

1. [Configure role-based access control (RBAC)](/nginx-ingress-controller/installation/installation-with-manifests/#1-configure-rbac).

    > **Important**: You must have an admin role to configure RBAC in your Kubernetes cluster.

2. [Create the common Kubernetes resources](/nginx-ingress-controller/installation/installation-with-manifests/#2-create-common-resources).
3. Enable the App Protect WAF module by adding the `enable-app-protect` [cli argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-app-protect) to your Deployment or DaemonSet file.
4. [Deploy the Ingress Controller](/nginx-ingress-controller/installation/installation-with-manifests/#3-deploy-the-ingress-controller).

For more information, see the [Configuration guide](/nginx-ingress-controller/app-protect/configuration) and the NGINX Ingress Controller with App Protect example resources on GitHub [for VirtualServer resources](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/custom-resources/app-protect-waf) and [for Ingress resources](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/ingress-resources/app-protect-waf).

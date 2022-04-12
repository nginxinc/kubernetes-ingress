---
title: Installation with NGINX App Protect
description: "This document provides an overview of the steps required to use NGINX App Protect with your NGINX Ingress Controller deployment."
weight: 1800
doctypes: [""]
toc: true
docs: "DOCS-579"
---

> **Note**: The NGINX Kubernetes Ingress Controller integration with NGINX App Protect requires the use of NGINX Plus.

This document provides an overview of the steps required to use NGINX App Protect with your NGINX Ingress Controller deployment. You can visit the linked documents to find additional information and instructions.

You can also [install the Ingress Controller with App Protect by using Helm](/nginx-ingress-controller/installation/installation-with-helm/). Use the `controller.appprotect.*` parameters of the chart.

## Using the Docker Images from the F5 Container registry

Take the steps below to either configure a Docker Registry secret in your Kubernetes cluster, or to pull the Docker image that you'll use to deploy NGINX Ingress Controller with App Protect in Kubernetes.

- Follow the steps to [use the JWT token to create a docker registry secret](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret).

- Alternatively, follow the steps to [Pull the NGINX Ingress Controller image](/nginx-ingress-controller/installation/pulling-ingress-controller-image).

    For NGINX Plus Ingress Controller with App Protect, pull from `private-registry.nginx.com/nginx-ic-nap/nginx-plus-ingress`:
   ```
   $ docker pull private-registry.nginx.com/nginx-ic-nap/nginx-plus-ingress:2.2.0
   ```
    `2.2.0` will pull down the Debian based image. The other available image tag is `2.2.0-ubi` for the UBI based image.

- Use the docker registry API to list the available image tags for the repository.

   To list the available image tags for the repository, you can use the Docker registry API, e.g.:
   ```
   $ curl https://private-registry.nginx.com/v2/nginx-ic-nap/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic-nap/nginx-plus-ingress",
    "tags": [
        "2.2.0-ubi",
        "2.2.0"
    ]
    }
   ```

## Build the Docker Image

Take the steps below to create the Docker image that you'll use to deploy NGINX Ingress Controller with App Protect in Kubernetes.

- [Build the NGINX Ingress Controller image](/nginx-ingress-controller/installation/building-ingress-controller-image).

    When running the `make` command to build the image, be sure to use the `debian-image-nap-plus` target. For example:

    ```bash
    make debian-image-nap-plus PREFIX=<your Docker registry domain>/nginx-plus-ingress
    ```
    Alternatively, if you want to run on an [OpenShift](https://www.openshift.com/) cluster, you can use the `ubi-image-nap-plus` target.

    If you intend to use [external references](https://docs.nginx.com/nginx-app-protect/configuration/#external-references) in NGINX App Protect policies, you may want to provide a custom CA certificate to authenticate with the hosting server.
    In order to do so, place the `*.crt` file in the build folder and uncomment the lines that follow this comment:
    `#Uncomment the lines below if you want to install a custom CA certificate`

    **Note**: In the event of a patch version of NGINX Plus being [released](/nginx/releases/), make sure to rebuild your image to get the latest version. The Dockerfile will use the latest available version of the [Attack Signatures](/nginx-app-protect/configuration/#attack-signatures) and [Threat Campaigns](/nginx-app-protect/configuration/#threat-campaigns) packages at the time of build. If your system is caching the Docker layers and not updating the packages, add `DOCKER_BUILD_OPTIONS="--no-cache"` to the `make` command.

- [Push the image to your local Docker registry](/nginx-ingress-controller/installation/building-ingress-controller-image/#building-the-image-and-pushing-it-to-the-private-registry).

## Install the Ingress Controller

Take the steps below to set up and deploy the NGINX Ingress Controller and App Protect module in your Kubernetes cluster.

1. [Configure role-based access control (RBAC)](/nginx-ingress-controller/installation/installation-with-manifests/#1-configure-rbac).

    > **Important**: You must have an admin role to configure RBAC in your Kubernetes cluster.

2. [Create the common Kubernetes resources](/nginx-ingress-controller/installation/installation-with-manifests/#2-create-common-resources).
3. Enable the App Protect module by adding the `enable-app-protect` [cli argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-app-protect) to your Deployment or DaemonSet file.
4. [Deploy the Ingress Controller](/nginx-ingress-controller/installation/installation-with-manifests/#3-deploy-the-ingress-controller).

For more information, see the [Configuration guide](/nginx-ingress-controller/app-protect/configuration) and the [NGINX Ingress Controller with App Protect examples on GitHub](https://github.com/nginxinc/kubernetes-ingress/tree/v2.2.0/examples/appprotect).

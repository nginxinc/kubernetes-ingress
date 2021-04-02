# Install NGINX Ingress Controller with App Protect

> **Note**: The NGINX Kubernetes Ingress Controller integration with NGINX App Protect requires the use of NGINX Plus.

This document provides an overview of the steps required to use NGINX App Protect with your NGINX Ingress Controller deployment. You can visit the linked documents to find additional information and instructions.

You can also [install the Ingress Controller with App Protect by using Helm](/nginx-ingress-controller/installation/installation-with-helm/). Use the `controller.appprotect.*` parameters of the chart.

## Build the Docker Image

Take the steps below to create the Docker image that you'll use to deploy NGINX Ingress Controller with App Protect in Kubernetes.

- [Build the NGINX Ingress Controller image](/nginx-ingress-controller/installation/building-ingress-controller-image).

    When running the `make` command to build the image, be sure to use the `debian-image-nap-plus` target. For example:

    ```bash
    make debian-image-nap-plus PREFIX=<your Docker registry domain>/nginx-plus-ingress
    ```
    Alternatively, if you want to run on an [OpenShift](https://www.openshift.com/) cluster, you can use the `openshift-image-nap-plus` target.

    If you intend to use [external references](https://docs.nginx.com/nginx-app-protect/configuration/#external-references) in NGINX App Protect policies, you may want to provide a custom CA certificate to authenticate with the hosting server.
    In order to do so, place the `*.crt` file in the build folder and uncomment the lines that follow this comment:
    `#Uncomment the lines below if you want to install a custom CA certificate`

- [Push the image to your local Docker registry](/nginx-ingress-controller/installation/building-ingress-controller-image/#building-the-image-and-pushing-it-to-the-private-registry).

## Install the Ingress Controller

Take the steps below to set up and deploy the NGINX Ingress Controller and App Protect module in your Kubernetes cluster.

1. [Configure role-based access control (RBAC)](/nginx-ingress-controller/installation/installation-with-manifests/#configure-rbac).

    > **Important**: You must have an admin role to configure RBAC in your Kubernetes cluster.

2. [Create the common Kubernetes resources](/nginx-ingress-controller/installation/installation-with-manifests/#create-common-resources).
3. Enable the App Protect module by adding the `enable-app-protect` [cli argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-app-protect) to your Deployment or DaemonSet file.
4. [Deploy the Ingress Controller](/nginx-ingress-controller/installation/installation-with-manifests/#deploy-the-ingress-controller).

For more information, see the [Configuration guide](/nginx-ingress-controller/app-protect/configuration) and the [NGINX Ingress Controller with App Protect examples on GitHub](https://github.com/nginxinc/kubernetes-ingress/tree/v1.11.0/examples/appprotect).

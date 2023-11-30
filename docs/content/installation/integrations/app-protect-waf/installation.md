---
title: Building NGINX Ingress Controller with NGINX App Protect WAF
description: "This document explains how to build a F5 NGINX Ingress Controller image with F5 NGINX App Protect WAF from source code."
weight: 1800
doctypes: [""]
toc: true
docs: "DOCS-579"
aliases: ["/app-protect/installation/"]
---

{{< custom-styles >}}

{{<call-out "tip" "Pre-built image alternatives" >}}If you'd rather not build your own NGINX Ingress Controller image, see the [pre-built image options](#pre-built-images) at the end of this guide.{{</call-out>}}

## Before you start

- To use NGINX App Protect WAF with NGINX Ingress Controller, you must have NGINX Plus.

## Prepare the environment

Get your system ready for building and pushing the NGINX Ingress Controller image with NGINX App Protect WAF.

1. Sign in to your private registry. Replace `<my-docker-registry>` with the path to your own private registry.

    ```shell
    docker login <my-docker-registry>
    ```

1. Clone the NGINX Ingress Controller repository:

    ```shell
    git clone https://github.com/nginxinc/kubernetes-ingress.git --branch v3.3.2
    cd kubernetes-ingress
    ```

---

## Build the image

Follow these steps to build the NGINX Controller Image with NGINX App Protect WAF.

1. Place your NGINX Plus license files (_nginx-repo.crt_ and _nginx-repo.key_) in the project's root folder. To verify they're in place, run:

    ```shell
    ls nginx-repo.*
    ```

    You should see:

    ```shell
    nginx-repo.crt  nginx-repo.key
    ```

2. Build the image. Replace `<makefile target>` with your chosen build option and `<my-docker-registry>` with your private registry's path. Refer to the [Makefile targets](#makefile-targets) table below for the list of build options.

    ```shell
    make <makefile target> PREFIX=<my-docker-registry>/nginx-plus-ingress TARGET=download
    ```

    For example, to build a Debian-based image with NGINX Plus and NGINX App Protect DoS, run:

    ```shell
    make debian-image-dos-plus PREFIX=<my-docker-registry>/nginx-plus-ingress TARGET=download
    ```

     **What to expect**: The image is built and tagged with a version number, which is derived from the `VERSION` variable in the [_Makefile_]({{< relref "installation/building-nginx-ingress-controller.md#makefile-details" >}}). This version number is used for tracking and deployment purposes.

{{<note>}} In the event a patch of NGINX Plus is released, make sure to rebuild your image to get the latest version. If your system is caching the Docker layers and not updating the packages, add `DOCKER_BUILD_OPTIONS="--pull --no-cache"` to the make command. {{</note>}}

### Makefile targets {#makefile-targets}

{{<bootstrap-table "table table-striped table-bordered">}}
| Makefile Target           | Description                                                       | Compatible Systems  |
|---------------------------|-------------------------------------------------------------------|---------------------|
| **debian-image-nap-plus** | Builds a Debian-based image with NGINX Plus and the [NGINX App Protect WAF](/nginx-app-protect-waf/) module. | Debian  |
| **debian-image-nap-dos-plus** | Builds a Debian-based image with NGINX Plus, [NGINX App Protect WAF](/nginx-app-protect-waf/), and [NGINX App Protect DoS](/nginx-app-protect-dos/) | Debian  |
| **ubi-image-nap-plus**    | Builds a UBI-based image with NGINX Plus and the [NGINX App Protect WAF](/nginx-app-protect-waf/) module. | OpenShift |
| **ubi-image-nap-dos-plus** | Builds a UBNI-based image with NGINX Plus, [NGINX App Protect WAF](/nginx-app-protect-waf/), and [NGINX App Protect DoS](/nginx-app-protect-dos/). | OpenShift |
{{</bootstrap-table>}}

<br>

{{<see-also>}} For the complete list of _Makefile_ targets and customizable variables, see the [Building NGINX Ingress Controller]({{< relref "installation/building-nginx-ingress-controller.md#makefile-details" >}}) guide. {{</see-also>}}

If you intend to use [external references](/nginx-app-protect-waf/configuration/#external-references) in NGINX App Protect WAF policies, you may want to provide a custom CA certificate to authenticate with the hosting server. 

To do so, place the `*.crt` file in the build folder and uncomment the lines following this comment:
`#Uncomment the lines below if you want to install a custom CA certificate`

{{<warning>}} External references are deprecated in NGINX Ingress Controller and will not be supported in future releases. {{</warning>}}

---

## Push the image to your private registry

Once you've successfully built the NGINX Ingress Controller image with NGINX App Protect WAF, the next step is to upload it to your private Docker registry. This makes the image available for deployment to your Kubernetes cluster.

To upload the image, run the following command. If you're using a custom tag, add `TAG=your-tag` to the end of the command. Replace `<my-docker-registry>` with your private registry's path.

```shell
make push PREFIX=<my-docker-registry>/nginx-plus-ingress
```

---

## Set up role-based access control (RBAC) {#set-up-rbac}

{{< include "rbac/set-up-rbac.md" >}}

---

## Create common resources {#create-common-resources}

{{< include "installation/create-common-resources.md" >}}

---

## Deploy NGINX Ingress Controller {#deploy-ingress-controller}

{{< include "installation/deploy-controller.md" >}}

### Using a Deployment

{{< include "installation/manifests/deployment.md" >}}

### Using a DaemonSet

{{< include "installation/manifests/daemonset.md" >}}

---

## Enable NGINX App Protect WAF module

To enable the NGINX App Protect DoS Module:

- Add the `enable-app-protect` [command-line argument]({{< relref "configuration/global-configuration/command-line-arguments.md#cmdoption-enable-app-protect" >}}) to your Deployment or DaemonSet file.

---

## Confirm NGINX Ingress Controller is running

{{< include "installation/manifests/verify-pods-are-running.md" >}}

For more information, see the [Configuration guide]({{< relref "installation/integrations/app-protect-waf/configuration.md" >}}),the [NGINX Ingress Controller with App Protect WAF example for VirtualServer](https://github.com/nginxinc/kubernetes-ingress/tree/v3.3.2/examples/custom-resources/app-protect-waf) and the [NGINX Ingress Controller with App Protect WAF example for Ingress](https://github.com/nginxinc/kubernetes-ingress/tree/v3.3.2/examples/ingress-resources/app-protect-waf).

---

## Alternatives to building your own image {#pre-built-images}

If you prefer not to build your own NGINX Ingress Controller image, you can use pre-built images. Here are your options:

- Download the image using your NGINX Ingress Controller subscription certificate and key. See the [Getting the F5 Registry NGINX Ingress Controller Image]({{< relref "installation/nic-images/pulling-ingress-controller-image.md" >}}) guide.
- Use your NGINX Ingress Controller subscription JWT token to get the image: Instructions are in [Getting the NGINX Ingress Controller Image with JWT]({{< relref "installation/nic-images/using-the-jwt-token-docker-secret.md" >}}).
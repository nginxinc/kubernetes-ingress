---
title: Pulling the Ingress Controller Image
description: 
weight: 1600
doctypes: [""]
toc: true
---

This document explains how to pull an NGINX Plus Ingress Controller image from the F5 Docker registry. Please see [here](/nginx-ingress-controller/installation/building-ingress-controller-image) for information on how to build an Ingress Controller image using the source code and your NGINX Plus license certificate and key. Note that for NGINX Ingress Controller based on NGINX OSS, we provide the image through [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/).

## Prerequisites

Before you can pull the image, make sure that the following software is installed on your machine:
* [Docker](https://www.docker.com/products/docker) v18.09+
* For NGINX Plus, you must have the NGINX Plus Ingress Controller license -- the certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`).

## Pulling the Image using Docker and Pushing It to the Private Registry

1. First, configure the Docker environment to use certificate-based client-server authentication with the F5 Container registry - `docker-registry.nginx.com`. 
   To do so in a Linux based environment, create a `docker-registry.nginx.com` directory under `/etc/docker/certs.d` and create a certificate `client.cert` (using `nginx-repo.crt` - please note that the certificate MUST have the `.cert` suffix, not `.crt`) and a key `client.key` (using `nginx-repo.key`). See [this document](https://docs.docker.com/engine/security/certificates/) for more details.

   ```
   # mkdir /etc/docker/certs.d/docker-registry.nginx.com
   # cp nginx-repo.crt /etc/docker/certs.d/docker-registry.nginx.com/client.cert
   # cp nginx-repo.key /etc/docker/certs.d/docker-registry.nginx.com/client.key
   ```

    > **Note**: The preceding example is operating-system specific and is for illustrative purposes only. You should consult your operating system documentation for creating an os-provided bundled certificate chain. For example, to configure this for Docker Desktop for Mac or Docker Desktop for Windows, see [this document](https://docs.docker.com/docker-for-mac/#add-client-certificates) or [this document](https://docs.docker.com/docker-for-windows/#how-do-i-add-client-certificates) for more details.

2. Use docker to pull the required image from `docker-registry.nginx.com`.
   For NGINX Plus Ingress Controller, pull from `docker-registry.nginx.com/nginx-ic/nginx-plus-ingress`:
   ```
   $ docker pull docker-registry.nginx.com/nginx-ic/nginx-plus-ingress:1.12.0
   ```

   `1.12.0` will pull down the Debian based image. Other available image tags are `1.12.0-alpine` for the Alpine based image, `1.12.0-ot` for the Debian based image with OpenTracing, and `1.12.0-ubi` for the UBI based image.
   
   For NGINX Plus Ingress Controller with App Protect, pull from `docker-registry.nginx.com/nginx-ic-nap/nginx-plus-ingress`:
   ```
   $ docker pull docker-registry.nginx.com/nginx-ic-nap/nginx-plus-ingress:1.12.0
   ```
   
   `1.12.0` will pull down the Debian based image. The other available image tag is `1.12.0-ubi` for the UBI based image.
   
   To list the available image tags for the repositories, you can use the Docker registry API, e.g.:
   ```
   $ curl https://docker-registry.nginx.com/v2/nginx-ic/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic/nginx-plus-ingress",
    "tags": [
        "1.12.0-alpine",
        "1.12.0-ot",
        "1.12.0-ubi",
        "1.12.0"
    ]
    }

   $ curl https://docker-registry.nginx.com/v2/nginx-ic-nap/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic-nap/nginx-plus-ingress",
    "tags": [
        "1.12.0-ubi",
        "1.12.0"
    ]
    }
   ```

3. Tag and push the image to your private registry.
   Make sure to run the `docker login` command first to log in to the registry.
   ```
   $ docker tag docker-registry.nginx.com/nginx-ic/nginx-plus-ingress:1.12.0 <my-docker-registry>/nginx-ic/nginx-plus-ingress:1.12.0
   $ docker push <my-docker-registry>/nginx-ic/nginx-plus-ingress:1.12.0
   ```
   
   or for NGINX App Protect enabled image
   ```
   $ docker tag docker-registry.nginx.com/nginx-ic-nap/nginx-plus-ingress:1.12.0 <my-docker-registry>/nginx-ic-nap/nginx-plus-ingress:1.12.0
   $ docker push <my-docker-registry>/nginx-ic-nap/nginx-plus-ingress:1.12.0
   ```

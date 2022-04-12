---
title: Pulling the Ingress Controller Image
description: "This document explains how to pull an NGINX Plus Ingress Controller image from the F5 Docker registry."
weight: 1700
doctypes: [""]
toc: true
docs: "DOCS-605"
---

This document explains how to pull an NGINX Plus Ingress Controller image from the F5 Docker registry using your NGINX Ingress Controller subscription certificate and key. **Please note that an NGINX Plus subscription certificate and key will not work with the F5 Docker registry.** You can also get the image using the following alternate methods:

* [Install using a JWT token in a Docker Config Secret]({{< relref "using-the-jwt-token-docker-secret" >}})
* [Build the Ingress Controller image]({{< relref "building-ingress-controller-image" >}}) using the source code from the GitHub repository and your NGINX Plus subscription certificate and key.
* For NGINX Ingress Controller based on NGINX OSS, you can pull the [nginx/nginx-ingress image](https://hub.docker.com/r/nginx/nginx-ingress/) from DockerHub.

## Prerequisites

Before you can pull the image, make sure that the following software is installed on your machine:
* [Docker](https://www.docker.com/products/docker) v18.09+
* For NGINX Ingress Controller, you must have the NGINX Ingress Controller subscription -- download the NGINX Plus Ingress Controller (per instance) certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`) from [MyF5](https://my.f5.com).

## Pulling the Image using Docker and Pushing It to the Private Registry

1. First, configure the Docker environment to use certificate-based client-server authentication with the F5 Container registry - `private-registry.nginx.com`.
   To do so in a Linux based environment, create a `private-registry.nginx.com` directory under `/etc/docker/certs.d` and create a certificate `client.cert` (using `nginx-repo.crt` - please note that the certificate MUST have the `.cert` suffix, not `.crt`) and a key `client.key` (using `nginx-repo.key`). See  the [Docker Engine Security documentation](https://docs.docker.com/engine/security/certificates/) for more details.

   ```
   # mkdir /etc/docker/certs.d/private-registry.nginx.com
   # cp nginx-repo.crt /etc/docker/certs.d/private-registry.nginx.com/client.cert
   # cp nginx-repo.key /etc/docker/certs.d/private-registry.nginx.com/client.key
   ```

    > **Note**: The preceding example is operating-system specific and is for illustrative purposes only. You should consult your operating system documentation for creating an os-provided bundled certificate chain. For example, to configure this for Docker Desktop for Mac or Docker Desktop for Windows, see the [Docker for Mac documentation](https://docs.docker.com/docker-for-mac/#add-client-certificates) or [Docker for Windows documentation](https://docs.docker.com/docker-for-windows/#how-do-i-add-client-certificates) for more details.

2. Use Docker to pull the required image from `private-registry.nginx.com`. Choose the image from the available images listed in the [tech specs guide]({{< relref "technical-specifications#images-with-nginx-plus" >}}).
   For NGINX Plus Ingress Controller, pull from `private-registry.nginx.com/nginx-ic/nginx-plus-ingress`. For example:
   ```
   $ docker pull private-registry.nginx.com/nginx-ic/nginx-plus-ingress:2.2.0
   ```

   For NGINX Plus Ingress Controller with App Protect, pull from `private-registry.nginx.com/nginx-ic-nap/nginx-plus-ingress`. For example:
   ```
   $ docker pull private-registry.nginx.com/nginx-ic-nap/nginx-plus-ingress:2.2.0
   ```

   To list the available image tags for the repositories, you can also use the Docker registry API. For example:
   ```
   $ curl https://private-registry.nginx.com/v2/nginx-ic/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic/nginx-plus-ingress",
    "tags": [
        "2.2.0-alpine",
        "2.2.0-ot",
        "2.2.0-ubi",
        "2.2.0"
    ]
    }

   $ curl https://private-registry.nginx.com/v2/nginx-ic-nap/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic-nap/nginx-plus-ingress",
    "tags": [
        "2.2.0-ubi",
        "2.2.0"
    ]
    }
   ```

3. Tag and push the image to your private registry.

   - Make sure to run the `docker login <my-docker-registry>` command first to log in to the registry.
   - Replace `<my-docker-registry>` in the examples below with the correct path to your private Docker registry.

   ```
   $ docker tag private-registry.nginx.com/nginx-ic/nginx-plus-ingress:2.2.0 <my-docker-registry>/nginx-ic/nginx-plus-ingress:2.2.0
   $ docker push <my-docker-registry>/nginx-ic/nginx-plus-ingress:2.2.0
   ```

   or for NGINX App Protect enabled image
   ```
   $ docker tag private-registry.nginx.com/nginx-ic-nap/nginx-plus-ingress:2.2.0 <my-docker-registry>/nginx-ic-nap/nginx-plus-ingress:2.2.0
   $ docker push <my-docker-registry>/nginx-ic-nap/nginx-plus-ingress:2.2.0
   ```

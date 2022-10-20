---
title: Using the NGINX IC Plus JWT token in a Docker Config Secret
description: "This document explains how to use the NGINX Plus Ingress Controller image from the F5 Docker registry in your Kubernetes cluster by using your NGINX Ingress Controller subscription JWT token."
weight: 1600
doctypes: [""]
toc: true
docs: "DOCS-608"
---

This document explains how to use the NGINX Plus Ingress Controller image from the F5 Docker registry in your Kubernetes cluster by using your NGINX Ingress Controller subscription JWT token. **Please note that an NGINX Plus subscription certificate and key will not work with the F5 Docker registry.** You can also get the image using alternative methods:

* You can use Docker to pull an Ingress Controller image with NGINX Plus and push it to your private registry by following the instructions from [here](/nginx-ingress-controller/installation/pulling-ingress-controller-image).
* Please see [here](/nginx-ingress-controller/installation/building-ingress-controller-image) for information on how to build an Ingress Controller image using the source code from this repository and your NGINX Plus subscription certificate and key.
* Note that for NGINX Ingress Controller based on NGINX OSS, we provide the image through [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/).

## Prerequisites

* For NGINX Ingress Controller, you must have the NGINX Ingress Controller subscription -- download the NGINX Plus Ingress Controller (per instance) JWT access token from [MyF5](https://myf5.com).
* To list the available image tags using the Docker registry API, you will also need to download the NGINX Plus Ingress Controller (per instance) certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`) from [MyF5](https://myf5.com).

## Using the JWT token in a Docker Config Secret

1. Create a `docker-registry` secret on the cluster using the JWT token as the username, and `none` for password (password is unused).  The name of the docker server is `private-registry.nginx.com`. Optionally namespace the secret.

	```
    kubectl create secret docker-registry regcred --docker-server=private-registry.nginx.com --docker-username=<JWT Token> --docker-password=none [-n nginx-ingress]
    ```

2. Confirm the details of the created secret by running:

	```
    kubectl get secret regcred --output=yaml
    ```

3. This secret can now be added to a deployment spec, or to a service account to apply to all deployments for a given SA spec. See the official documentation [here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-pod-that-uses-your-secret) and [here](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account) for more details.

4. Update the deployment spec with the chosen image path. Choose the image from the available images listed [here](/nginx-ingress-controller/technical-specifications/#images-with-nginx-plus).

5. You can use the certificate and key from the MyF5 portal and the Docker registry API to list the available image tags for the repositories, e.g.:
   ```
   $ curl https://private-registry.nginx.com/v2/nginx-ic/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic/nginx-plus-ingress",
    "tags": [
        "2.4.0-alpine",
        "2.4.0-ubi",
        "2.4.0"
    ]
    }

   $ curl https://private-registry.nginx.com/v2/nginx-ic-nap/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic-nap/nginx-plus-ingress",
    "tags": [
        "2.4.0-ubi",
        "2.4.0"
    ]
    }

   $ curl https://private-registry.nginx.com/v2/nginx-ic-dos/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic-dos/nginx-plus-ingress",
    "tags": [
        "2.4.0-ubi",
        "2.4.0"
    ]
    }
   ```

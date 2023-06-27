---
title: Using NGINX Ingress Controller Plus JWT token in a Docker Config Secret
description: "This document explains how to use the NGINX Plus Ingress Controller image from the F5 Docker registry in your Kubernetes cluster by using an NGINX Ingress Controller subscription JWT token."
weight: 1600
doctypes: [""]
toc: true
---

## Overview

This document explains how to use the NGINX Plus Ingress Controller image from the F5 Docker registry in your Kubernetes cluster by using an NGINX Ingress Controller subscription JWT token.

**Note that an NGINX Plus subscription certificate and key will not work with the F5 Docker registry.**

You can also get the image using alternative methods:

* You can use Docker to pull an NGINX Ingress Controller image with NGINX Plus and push it to your private registry by following the ["Pulling the Ingress Controller Image"](https://docs.nginx.com/nginx-ingress-controller/installation/pulling-ingress-controller-image/) documentation.
* You can also build an NGINX Ingress Controller image by following the ["Information on how to build an Ingress Controller image"](https://docs.nginx.com/nginx-ingress-controller/installation/building-ingress-controller-image/) documentation.

If you would like an NGINX Ingress Controller image using NGINX open source, we provide the image through [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/).

## Before You Begin

You will need the following information from [MyF5](https://my.f5.com) for these steps:

* A JWT Access Token (Per instance) for NGINX Ingress Controller from an active NGINX Ingress Controller subscription.
* The certificate (`nginx-repo.crt`) and key (`nginx-repo.key`) for each NGINX Ingress Controller instance, used to list the available image tags from the Docker registry API.

## Prepare NGINX Ingress Controller

1. Choose your desired [NGINX Ingress Controller Image](https://docs.nginx.com/nginx-ingress-controller/technical-specifications/#images-with-nginx-plus).
1. Log into the [MyF5 Portal](https://myf5.com/), navigate to your subscription details, and download the relevant .cert, .key and .JWT files.
1. Create a Kubernetes secret using the JWT token. You should use `cat` to view the contents of the JWT token and store the output for use in later steps.
1. Ensure there are no additional characters or extra whiespace that might have been accidently added. This will break authorization and prevent the NGINX Ingress Controller image from being downloaded.
1. Modify your deployment (manifest or helm) to use the Kubernetes secret created in step three.
1. Deploy NGINX Ingress Controller into your Kubernetes cluster and verify successful installation.

## Using the JWT token in a Docker Config Secret

1. Create a kubernetes `docker-registry` secret type on the cluster, using the JWT token as the username and `none` for password (Password is unused).  The name of the docker server is `private-registry.nginx.com`.

	```bash
    kubectl create secret docker-registry regcred --docker-server=private-registry.nginx.com --docker-username=<JWT Token> --docker-password=none [-n nginx-ingress]
    ```
   It is important that the `--docker-username=<JWT Token>` contains the contents of the token and is not pointing to the token itself. Ensure that when you copy the contents of the JWT token, there are no additional characters or extra whitepaces. This can invalidate the token and cause 401 errors when trying to authenticate to the registry.

1. Confirm the details of the created secret by running:

	```bash
    kubectl get secret regcred --output=yaml
    ```

3. You can now add this secret to a deployment spec or to a service account to apply to all deployments for a given SA spec. See the [Create a Pod that uses your Secret](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-pod-that-uses-your-secret) and [Add ImagePullSecrets to a service account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#add-imagepullsecrets-to-a-service-account) documentation for more details.

If you want to install NGINX Ingress Controller using the charts method, the following is an example of using the command line to pass the required arguments using the `set` parameter.

5. You can use the certificate and key from the MyF5 portal and the Docker registry API to list the available image tags for the repositories, e.g.:
   ```
   $ curl https://private-registry.nginx.com/v2/nginx-ic/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic/nginx-plus-ingress",
    "tags": [
        "3.1.1-alpine",
        "3.1.1-ubi",
        "3.1.1"
    ]
    }

   $ curl https://private-registry.nginx.com/v2/nginx-ic-nap/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic-nap/nginx-plus-ingress",
    "tags": [
        "3.1.1-ubi",
        "3.1.1"
    ]
    }

   $ curl https://private-registry.nginx.com/v2/nginx-ic-dos/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic-dos/nginx-plus-ingress",
    "tags": [
        "3.1.1-ubi",
        "3.1.1"
    ]
    }
```

## Pulling an Image for Local Use

If you need to pull the image for local use to then push to a different container registry, here is the command:

```bash
docker login private-registry.nginx.com --username=<output_of_jwt_token> --password=none
```

Replace the contents of `<output_of_jwt_token>` with the contents of the `jwt token` itself.
Once you have successfully pulled the image, you can then tag it as needed.

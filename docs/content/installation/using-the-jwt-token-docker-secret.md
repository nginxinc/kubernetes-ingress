---
title: Using the NGINX IC Plus JWT token in a Docker Config Secret
description: "This document explains how to use the NGINX Plus Ingress Controller image from the F5 Docker registry in your Kubernetes cluster by using your NGINX Ingress Controller subscription JWT token."
weight: 1600
doctypes: [""]
toc: true
docs: "DOCS-608"
---

This document explains how to use the NGINX Plus Ingress Controller image from the F5 Docker registry in your Kubernetes cluster by using your NGINX Ingress Controller subscription JWT token. **Please note that an NGINX Plus subscription certificate and key will not work with the F5 Docker registry.** You can also get the image using alternative methods:

* You can use Docker to pull an Ingress Controller image with NGINX Plus and push it to your private registry by following the [Pulling the Ingress Controller Image]({{< relref "/installation/pulling-ingress-controller-image.md" >}}) documentation.
* Please see the [information on how to build an Ingress Controller image]({{< relref "/installation/building-ingress-controller-image.md" >}}) using the source code from this repository and your NGINX Plus subscription certificate and key.
* Note that for NGINX Ingress Controller based on NGINX OSS, we provide the image through [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/).

## Prerequisites

* For NGINX Ingress Controller, you must have the NGINX Ingress Controller subscription -- download the NGINX Plus Ingress Controller (per instance) JWT access token from [MyF5](https://my.f5.com).
* To list the available image tags using the Docker registry API, you will also need to download the NGINX Plus Ingress Controller (per instance) certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`) from [MyF5](https://my.f5.com).

## Order of steps to pull NGINX Ingress controller from F5  registry

1. Decide on what NGINX Ingress controller image you want to use. [NGINX Ingress controller images](https://docs.nginx.com/nginx-ingress-controller/technical-specifications/#images-with-nginx-plus "Available NGINX Ingress controller images")
2. Log into MyF5 portal. [MyF5 portal login](https://myf5.com/ "MyF5 portal login"). Navigate to your subscription details, and locate your .cert, .key and .JWT file to download.
3. Download the JWT token from the `MyF5` portal.
4. Create kubernetes secret using the JWT token that is provided in the MyF5 portal.
   You can `cat` the contents of the JWT token and then store the output to use in the following steps. Make sure that there are no additional characters or extra whiespace that might have been accidently added. This will break authorization and prevent the NGINX Ingress controller image from being downloaded successfully.
5. Modify your deployment (manifest or helm) to use the kubernetes secret created in step 4.
6. Deploy NGINX Ingress controller into your kubernetes cluster and verify successful installation.


## Using the JWT token in a Docker Config Secret

1. Create a kubernetes `docker-registry` secret type, on the cluster using the JWT token as the username and `none` for password (password is unused).  The name of the docker server is `private-registry.nginx.com`.

	```
    kubectl create secret docker-registry regcred --docker-server=private-registry.nginx.com --docker-username=<JWT Token> --docker-password=none [-n nginx-ingress]
    ```
   In the above command, it is important that the `--docker-username=<JWT Token>` contains the contents of the token and is not pointing to the token itself. Ensure that when you copy the contents of the JWT token, there are no additional characters or extra whitepaces. This can invalidate the token and cause 401 errors when trying to authenticate to the registry.

2. Confirm the details of the created secret by running:

	```bash
    kubectl get secret regcred --output=yaml
    ```

3. We are now going to use our newly created kubernetes secret in our `helm` and `manifest` deployments.


### Manifest deployment

[Installling with Manfiets](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/)

Manifest deployment example:

```yaml
spec:
  serviceAccountName: nginx-ingress
  imagePullSecrets:
  - name: regcred
  automountServiceAccountToken: true
  securityContext:
    seccompProfile:
      type: RuntimeDefault
#        fsGroup: 101 #nginx
  containers:
  - image: private-registry.nginx.com/nginx-ic/nginx-plus-ingress:3.1.1
    imagePullPolicy: IfNotPresent
    name: nginx-plus-ingress
```

Notice `imagePullSecrets` and `containers.image` lines to represent our kubernetes secret as well as the registry and version of the NGINX Ingress controller we are going to deploy.

### Helm install method

If you are using `helm` to install, you can install using two methods. First is the `helm` sources method. [Helm sources install](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/#managing-the-chart-via-sources "helm source install")

1. Clone the nginxinc/kubernetes-ingress repository
2. Change directory into deployents/helm-chart of the recently cloned repo.
3. Modify the `values.yaml` file.

We want to set a few lines for NGINX Plus Ingress controller to be deployed.

1. Change to `nginxplus`
2. Specify NGINX Ingress controller image to use.
3. Specify a `imagePullSecretName` to pull from the private registry.


### Ensure nginxplus is set to `true`
```yaml
## Deploys the Ingress Controller for NGINX Plus.
nginxplus: true
```

### Specify the image to be used for deployment

```yaml
image:
  ## The image repository of the Ingress Controller.
  repository: private-registry.nginx.com/nginx-ic/nginx-plus-ingress

  ## The tag of the
  tag: 3.1.1
```

### Specify the `imagePullSecrets`

```yaml
  serviceAccount:
    ## The annotations of the service account of the Ingress Controller pods.
    annotations: {}

    ## The name of the service account of the Ingress Controller pods. Used for RBAC.
    ## Autogenerated if not set or set to "".
    # name: nginx-ingress

    ## The name of the secret containing docker registry credentials.
    ## Secret must exist in the same namespace as the helm release.
    imagePullSecretName: regcred
```

### Using the `helm` charts method:

This will install `NGINX Ingress controller` using the charts method, by defining specific settings using `set` on the command line.

```bash
helm install my-release -n nginx-ingress oci://ghcr.io/nginxinc/charts/nginx-ingress --version 0.17.1 --set controller.image.repository=private-registry.nginx.com/nginx-ic/nginx-plus-ingress --set controller.image.tag=3.1.1 --set controller.nginxplus=true --set controller.serviceAccount.imagePullSecretName=regcred
```

### Verify that NGINX Ingress controller was installed successfull

TODO: Add install output of successful install



## Checking the validation that the .crts/key and .jwt are able to successfully authenticate to the repo to pull NGINX Ingress controller images:

You can also use the certificate and key from the MyF5 portal and the Docker registry API to list the available image tags for the repositories, e.g.:

```bash
   $ curl https://private-registry.nginx.com/v2/nginx-ic/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic/nginx-plus-ingress",
    "tags": [
        "3.1.0-alpine",
        "3.1.0-ubi",
        "3.1.0"
    ]
    }

   $ curl https://private-registry.nginx.com/v2/nginx-ic-nap/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic-nap/nginx-plus-ingress",
    "tags": [
        "3.1.0-ubi",
        "3.1.0"
    ]
    }

   $ curl https://private-registry.nginx.com/v2/nginx-ic-dos/nginx-plus-ingress/tags/list --key <path-to-client.key> --cert <path-to-client.cert> | jq
   {
    "name": "nginx-ic-dos/nginx-plus-ingress",
    "tags": [
        "3.1.0-ubi",
        "3.1.0"
    ]
    }
```

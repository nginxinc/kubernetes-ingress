# Building the Ingress Controller Image

This document explains how to build an Ingress Controller image. Note that for NGINX, we provide the image though [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/). For NGINX Plus, you need to build the image.

## Prerequisites

Before you can build the image, make sure that the following software is installed on your machine:
* [Docker](https://www.docker.com/products/docker) v18.09+
* [GNU Make](https://www.gnu.org/software/make/)
* [git](https://git-scm.com/)
* [OpenSSL](https://www.openssl.org/), optionally, if you would like to generate a self-signed certificate and a key for the default server.
* For NGINX Plus, you must have the NGINX Plus license -- the certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`).

Although the Ingress Controller is written in golang, golang is not required, you have the option to build the Ingress Controller in a Docker container.

## Building the Image and Pushing It to the Private Registry

We build the image using the make utility and the provided `Makefile`. Let’s create the Ingress Controller binary, build an image and push the image to the private registry.

**Note**: If you have a local golang environment, you can remove `TARGET=container` from the `make` commands to speed up the build.

1. Make sure to run the `docker login` command first to log in to the registry.

   If you’re using Google Container Registry, make sure you’re logged into the gcloud tool by running the `gcloud auth login` and `gcloud auth configure-docker` commands.

1. Clone the Ingress Controller repo:
    ```
    $ git clone https://github.com/nginxinc/kubernetes-ingress/
    $ git checkout v1.11.0
    ```

1. Build the image:
    * For **NGINX**:
      ```
      $ make debian-image PREFIX=myregistry.example.com/nginx-ingress TARGET=container
      ```
      or if you wish to use alpine
      ```
      $ make alpine-image PREFIX=myregistry.example.com/nginx-ingress TARGET=container
      ```
      `myregistry.example.com/nginx-ingress` defines the repo in your private registry where the image will be pushed. Substitute that value with the repo in your private registry.

      As a result, the image **myregistry.example.com/nginx-ingress:1.11.0** is built. Note that the tag `1.11.0` comes from the `VERSION` variable, defined in the Makefile.

    * For **NGINX Plus**, first, make sure that the certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`) of your license are located in the root of the project:
      ```
      $ ls nginx-repo.*
      nginx-repo.crt  nginx-repo.key
      ```
      Then run:
      ```
      $ make debian-image-plus PREFIX=myregistry.example.com/nginx-plus-ingress TARGET=container
      ```
      `myregistry.example.com/nginx-plus-ingress` defines the repo in your private registry where the image will be pushed. Substitute that value with the repo in your private registry.

      As a result, the image **myregistry.example.com/nginx-plus-ingress:1.11.0** is built. Note that the tag `1.11.0` comes from the `VERSION` variable, defined in the Makefile.

1. Push the image:
    ```
    $ make push PREFIX=myregistry.example.com/nginx-ingress
    ```
    Note: If you're using a different tag, append `TAG=your-tag` to the command above.

Next you will find the details about available Makefile targets and variables.

### Makefile Targets

You can see a list of all the targets by running `make` without any target or `make help`

Below you can find some of the most useful targets in the **Makefile**:
* **build**: creates the Ingress Controller binary using the local golang environment (ignored when `TARGET` is `container`).
* **debian-image**: for building a debian-based image with NGINX.
* **alpine-image**: for building an alpine-based image with NGINX.
* **debian-image-plus**: for building a debian-based image with NGINX Plus.
* **debian-image-nap-plus**: for building a debian-based image with NGINX Plus and the [appprotect](/nginx-app-protect/) module.
* **debian-image-opentracing**: for building a debian-based image with NGINX, [opentracing](https://github.com/opentracing-contrib/nginx-opentracing) module and the [Jaeger](https://www.jaegertracing.io/) tracer.
* **debian-image-opentracing-plus**: for building a debian-based image with NGINX Plus, [opentracing](https://github.com/opentracing-contrib/nginx-opentracing) module and the [Jaeger](https://www.jaegertracing.io/) tracer.
* **openshift-image**: for building an ubi-based image with NGINX for [Openshift](https://www.openshift.com/) clusters.
* **openshift-image-plus**: for building an ubi-based image with NGINX Plus for [Openshift](https://www.openshift.com/) clusters.
* **openshift-image-nap-plus**: for building an ubi-based image with NGINX Plus and the [appprotect](/nginx-app-protect/) module for [Openshift](https://www.openshift.com/) clusters.
Note: You need to store your RHEL organization and activation keys in a file named `rhel_license` in the project root. Example:
  ```bash
  RHEL_ORGANIZATION=1111111
  RHEL_ACTIVATION_KEY=your-key
  ```

A few other useful targets:
* **push**: pushes the image to the Docker registry specified in `PREFIX` and `TAG` variables.
* **all**: executes test `test`, `lint`, `verify-codegen`, `update-crds` and `debian-image`. If one of the targets fails, the execution process stops, reporting an error.
* **test**: runs unit tests.
* **certificate-and-key**: The Ingress Controller requires a certificate and a key for the default HTTP/HTTPS server. You can reference them in a TLS Secret in a command-line argument to the Ingress Controller. As an alternative, you can add a file in the PEM format with your certificate and key to the image as `/etc/nginx/secrets/default`. Optionally, you can generate a self-signed certificate and a key using this target. Note that you must add the `ADD` instruction in the Dockerfile to copy the cert and the key to the image.

### Makefile Variables

The **Makefile** contains the following main variables for you to customize (either by changing the Makefile or by overriding the variables in the make command):
* **PREFIX** -- the name of the image. The default is `nginx/nginx-ingress`.
* **VERSION** -- the current version of the Ingress Controller.
* **TAG** -- the tag added to the image. It's set to the value of the `VERSION` variable by default.
* **DOCKER_BUILD_OPTIONS** -- the [options](https://docs.docker.com/engine/reference/commandline/build/#options) for the `docker build` command. For example, `--pull`.
* **TARGET** -- By default, the Ingress Controller is compiled locally using a `local` golang environment. If you want to compile the Ingress Controller using your local golang environment, make sure that the Ingress Controller repo is in your `$GOPATH`. To compile the Ingress Controller using the Docker [golang](https://hub.docker.com/_/golang/) container, specify `TARGET=container`.

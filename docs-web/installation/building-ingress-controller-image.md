# Building the Ingress Controller Image

This document explains how to build an Ingress Controller image. Note that for NGINX, we provide the image though [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/). For NGINX Plus, you need to build the image.

## Prerequisites

Before you can build the image, make sure that the following software is installed on your machine:
* [Docker](https://www.docker.com/products/docker) v18.09+
* [GNU Make](https://www.gnu.org/software/make/)
* [git](https://git-scm.com/)
* [OpenSSL](https://www.openssl.org/), optionally, if you would like to generate a self-signed certificate and a key for the default server.
* For NGINX Plus, you must have the NGINX Plus license -- the certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`).

Although the Ingress controller is written in golang, golang is not required, as the Ingress controller binary will be built in a Docker container.

## Building the Image and Pushing It to the Private Registry

We build the image using the make utility and the provided `Makefile`. Let’s create the controller binary, build an image and push the image to the private registry.

1. Make sure to run the `docker login` command first to login to the registry. If you’re using Google Container Registry, you don’t need to use the docker command to login -- make sure you’re logged into the gcloud tool (using the `gcloud auth login` and `gcloud auth configure-docker` commands).

1. Clone the Ingress controller repo:
    ```
    $ git clone https://github.com/nginxinc/kubernetes-ingress/
    ```

1. Build the image:
    * For **NGINX**:
      ```
      $ make debian-image PREFIX=myregistry.example.com/nginx-ingress
      ```
      or if you wish to use alpine
      ```
      $ make alpine-image PREFIX=myregistry.example.com/nginx-ingress
      ```
      `myregistry.example.com/nginx-ingress` defines the repo in your private registry where the image will be pushed. Substitute that value with the repo in your private registry.

      As a result, the image **myregistry.example.com/nginx-ingress:edge** is built. Note that the tag `edge` comes from the `VERSION` variable, defined in the Makefile.

    * For **NGINX Plus**, first, make sure that the certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`) of your license are located in the root of the project:
      ```
      $ ls nginx-repo.*
      nginx-repo.crt  nginx-repo.key
      ```
      Then run:
      ```
      $ make debian-image-plus PREFIX=myregistry.example.com/nginx-plus-ingress
      ```
      `myregistry.example.com/nginx-plus-ingress` defines the repo in your private registry where the image will be pushed. Substitute that value with the repo in your private registry.

      As a result, the image **myregistry.example.com/nginx-plus-ingress:edge** is built. Note that the tag `edge` comes from the `VERSION` variable, defined in the Makefile.

1. Push the image:
    ```
    $ make push PREFIX=myregistry.example.com/nginx-ingress
    ```
    Note: If you're using a different tag, append `TAG=yourtag` to the command above.

Next you will find the details about available Makefile targets and variables.

### Makefile Targets

You can see a list of all the targets by running `make` without any target or `make help`

Below you can find some of the most useful targets in the **Makefile**:
* **build**: creates the controller binary using local golang environment (ignored when `TARGET` is `container`).
* **debian-image**: for building a debian-based image with NGINX.
* **alpine-image**: for building an alpine-based image with NGINX.
* **debian-image-plus**: for building an debian-based image with NGINX Plus.
* **debian-image-opentracing**: for building a debian-based image with NGINX, [opentracing](https://github.com/opentracing-contrib/nginx-opentracing) module and the [Jaeger](https://www.jaegertracing.io/) tracer.
* **debian-image-opentracing-plus**: for building a debian-based image with NGINX Plus, [opentracing](https://github.com/opentracing-contrib/nginx-opentracing) module and the [Jaeger](https://www.jaegertracing.io/) tracer.
* **openshift-image**: for building an ubi-based image with NGINX for [Openshift](https://www.openshift.com/) clusters.
* **openshift-image-plus**: for building an ubi-based image with NGINX Plus for [Openshift](https://www.openshift.com/) clusters.
* **debian-image-nap-plus**: for building a debian-based image with NGINX Plus and the [appprotect](/nginx-app-protect/) module.
Note: You need to place a file named `rhel_license` containing Your Organization and Activation key in the project root. Example:
  ```bash
  RHEL_ORGANIZATION=1111111
  RHEL_ACTIVATION_KEY=your-key
  ```

A few other useful targets:
* **push**: pushes the image to the Docker registry specified in `PREFIX` and `TAG` variables.
* **all**: executes test `test`, `lint`, `verify-codegen`, `update-crds` and `debian-image`. If one of the targets fails, the execution process stops, reporting an error.
* **test**: runs unit tests.
* **certificate-and-key**: The Ingress controller requires a certificate and a key for the default HTTP/HTTPS server. You can reference them in a TLS Secret in a command-line argument to the Ingress controller. As an alternative, you can add a file in the PEM format with your certificate and key to the image as `/etc/nginx/secrets/default`. Optionally, you can generate a self-signed certificate and a key using this target. Note that you must add the `ADD` instruction in the Dockerfile to copy the cert and the key to the image.

### Makefile Variables

The **Makefile** contains the following main variables for you to customize (either by changing the Makefile or by overriding the variables in the make command):
* **PREFIX** -- the name of the image. The default is `nginx/nginx-ingress`.
* **VERSION** -- the current version of the controller.
* **TAG** -- the tag added to the image. It's set to the value of the `VERSION` variable by default.
* **DOCKER_BUILD_OPTIONS** -- the [options](https://docs.docker.com/engine/reference/commandline/build/#options) for the `docker build` command. For example, `--pull`.
* **TARGET** -- By default, the Ingress Controller locally is compiled locally using a `local` golang environment. If you want to compile the controller using your local golang environment make sure that the Ingress controller repo is in your `$GOPATH`. To compile the Ingress Controller using Docker the [golang](https://hub.docker.com/_/golang/) container, specify `TARGET=container`.

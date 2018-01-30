# NGINX Ingress Controller

This is an implementation of a Kubernetes Ingress controller for NGINX and NGINX Plus, which provides HTTP load balancing for applications your deploy in your Kubernetes cluster. You can find more details on what an Ingress controller is on the [main page](https://github.com/nginxinc/kubernetes-ingress).

## How to Use the Controller

To find examples on how to deploy, configure and use the Ingress controller, please see [the examples folder](../examples). The examples require the Docker image of the controller to be available to your Kubernetes cluster. We provide such an image though [DockerHub](https://hub.docker.com/r/nginxdemos/nginx-ingress/) for NGINX. If you are using NGINX Plus, you need to build the image.

There are other cases when you need to build your own image. For example if you want to customize the controller, either by changing the NGINX configuration templates and/or modifying the controller logic. Please read the next section for instructions on how to build an image.

## How to Build the Controller Image

### Prerequisites

Before you can build the image, make sure that the following software is installed on your machine:
* [Docker](https://www.docker.com/products/docker)
* [GNU Make](https://www.gnu.org/software/make/)
* [OpenSSL](https://www.openssl.org/), optionally, if you would like to generate a self-signed certificate and a key for the default server.

Additionally, for NGINX Plus, you must have the NGINX Plus license. If you don't have one, you can sign up for a [free 30-day trial](https://www.nginx.com/free-trial-request/).  Put the certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`) of your license inside this folder.

### Building the image

We build the image using the make utility. The **Makefile** we provide has the following targets:
* **test**: runs unit tests.
* **nginx-ingress**: creates the controller binary.
* **container**: builds a Docker image.
* **push**: pushes the image to the private Docker registry.
* **all** (the default target): executes the four targets above in the order listed. If one of the targets fails, the execution process stops, reporting an error.

The **Makefile** contains the following main variables, which you should customize (either by changing the Makefile or by overriding the variables in the make command):
* **PREFIX** -- the name of the image. For example, `nginx-ingress`
* **VERSION** -- the current version of the controller.
* **TAG** -- the tag added to the image. It's set to the value of the `VERSION` variable by default.
* **PUSH_TO_GCR**. If you’re running your Kubernetes in GCE and using Google Container Registry, make sure that `PUSH_TO_GCR = 1`. This means using the `gcloud docker push` command to push the image, which is convenient when pushing images to GCR. By default, the variable is unset and the regular `docker push` command is used to push the image to the registry.
* **DOCKERFILE** -- the path to a Dockerfile. We provide three Dockerfiles:
  1. `Dockerfile`, for building a debian-based image with NGINX. It's used by default.
  1. `DockerfileForAlpine`, for building an alpine-based image with NGINX.
  1. `DockerfileForPlus`, for building an ubuntu-based image with NGINX Plus.
* **GENERATE_DEFAULT_CERT_AND_KEY** - The Ingress controller requires a certificate and a key for the default HTTP/HTTPS server. You can reference them in a TLS Secret in a command-line argument to the Ingress controller. As an alternative, you can add a file in the PEM format with your certificate and key to the image as `/etc/nginx/secrets/default`. Optionally, you can generate a self-signed certificate and a key during the build process. Set `GENERATE_DEFAULT_CERT_AND_KEY` to `1` to generate a certificate and a key in the `default.pem` file. Note that you must add the `ADD` instruction in the Dockerfile to copy the cert and the key to the image. The default value of `GENERATE_DEFAULT_CERT_AND_KEY` is `0`. 
* **DOCKER_BUILD_OPTIONS** -- the [options](https://docs.docker.com/engine/reference/commandline/build/#options) for the `docker build` command. For example, `--pull`.

Let’s create the controller binary, build an image and push the image to the private registry. Make sure to run the `docker login` command first to login to the registry. If you’re using Google Container Registry, as we are in our example here, you don’t need to use the docker command to login. However, make sure you’re logged into the gcloud tool (using the `gcloud auth login` command).

In this folder we run the following commands in the shell:

For NGINX:
```
$ make clean
$ make PREFIX=gcr.io/my-k8s-project/nginx-ingress TAG=latest PUSH_TO_GCR=1
```

Where **my-k8s-project** is the name of the GCE project where we run our Kubernetes cluster. As the result, the image -- **gcr.io/my-k8s-project/nginx-ingress:latest** --  is built and pushed to the registry.

For NGINX Plus:
```
$ make clean
$ make DOCKERFILE=DockerfileForPlus PREFIX=gcr.io/my-k8s-project/nginx-plus-ingress TAG=latest PUSH_TO_GCR=1
```

By default, to compile the controller we use the [golang](https://hub.docker.com/_/golang/) container that we run as part of the building process. If you want to compile the controller using your local golang environment, specify `BUILD_IN_CONTAINER=0` when you run the make command.

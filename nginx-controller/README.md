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
* [git](https://git-scm.com/)
* [OpenSSL](https://www.openssl.org/), optionally, if you would like to generate a self-signed certificate and a key for the default server.
* For NGINX Plus, you must have the NGINX Plus license -- the certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`). If you don't have one, you can sign up for a [free 30-day trial](https://www.nginx.com/free-trial-request/).

Although the Ingress controller is written in golang, golang is not required, as the Ingress controller binary will be built in a Docker container.

### Building the Image and Pushing It to the Private Registry

We build the image using the make utility and the provided `Makefile`. Let’s create the controller binary, build an image and push the image to the private registry. 

1. Make sure to run the `docker login` command first to login to the registry. If you’re using Google Container Registry, you don’t need to use the docker command to login -- make sure you’re logged into the gcloud tool (using the `gcloud auth login` command) and set the variable `PUSH_TO_GCR=1` when running the make command.

1. Clone the Ingress controller repo and change your folder to `nginx-controller`: 
    ```
    $ git clone https://github.com/nginxinc/kubernetes-ingress/
    $ cd kubernetes-ingress/nginx-controller
    ```

1. Build the image:
    * For NGINX:
      ```
      $ make clean
      $ make PREFIX=myregistry.example.com/nginx-ingress
      ```
      `myregistry.example.com/nginx-ingress` defines the repo in your private registry where the image will be pushed. Substitute that value with the repo in your private registry.
      
      As the result, the image **myregistry.example.com/nginx-ingress:1.1.1** is built and pushed to the registry. Note that the tag `1.1.1` comes from the `VERSION` variable, defined in the Makefile.

    * For NGINX Plus, first, make sure that the certificate (`nginx-repo.crt`) and the key (`nginx-repo.key`) of your license are located in the `nginx-controller` folder:
      ```
      $ ls nginx-repo.*
      nginx-repo.crt  nginx-repo.key
      ```
      Then run:
      ```
      $ make clean
      $ make DOCKERFILE=DockerfileForPlus PREFIX=myregistry.example.com/nginx-plus-ingress
      ```
      `myregistry.example.com/nginx-plus-ingress` defines the repo in your private registry where the image will be pushed. Substitute that value with the repo in your private registry.
      
      As the result, the image **myregistry.example.com/nginx-plus-ingress:1.1.1** is built and pushed to the registry. Note that the tag `1.1.1` comes from the `VERSION` variable, defined in the Makefile.

Next you will find the details about available Makefile targets and variables.

### Makefile Targets

The **Makefile** we provide has the following targets:
* **test**: runs unit tests.
* **nginx-ingress**: creates the controller binary.
* **container**: builds a Docker image.
* **push**: pushes the image to the private Docker registry.
* **all** (the default target): executes the four targets above in the order listed. If one of the targets fails, the execution process stops, reporting an error.

### Makefile Variables

The **Makefile** contains the following main variables for you to customize (either by changing the Makefile or by overriding the variables in the make command):
* **PREFIX** -- the name of the image. The default is `nginxdemos/nginx-ingress`.
* **VERSION** -- the current version of the controller.
* **TAG** -- the tag added to the image. It's set to the value of the `VERSION` variable by default.
* **PUSH_TO_GCR**. If you’re running your Kubernetes in GCE and using Google Container Registry, make sure that `PUSH_TO_GCR = 1`. This means using the `gcloud docker push` command to push the image, which is convenient when pushing images to GCR. By default, the variable is unset and the regular `docker push` command is used to push the image to the registry.
* **DOCKERFILE** -- the path to a Dockerfile. We provide three Dockerfiles:
  1. `Dockerfile`, for building a debian-based image with NGINX. It's used by default.
  1. `DockerfileForAlpine`, for building an alpine-based image with NGINX.
  1. `DockerfileForPlus`, for building an debian-based image with NGINX Plus.
* **GENERATE_DEFAULT_CERT_AND_KEY** - The Ingress controller requires a certificate and a key for the default HTTP/HTTPS server. You can reference them in a TLS Secret in a command-line argument to the Ingress controller. As an alternative, you can add a file in the PEM format with your certificate and key to the image as `/etc/nginx/secrets/default`. Optionally, you can generate a self-signed certificate and a key during the build process. Set `GENERATE_DEFAULT_CERT_AND_KEY` to `1` to generate a certificate and a key in the `default.pem` file. Note that you must add the `ADD` instruction in the Dockerfile to copy the cert and the key to the image. The default value of `GENERATE_DEFAULT_CERT_AND_KEY` is `0`. 
* **DOCKER_BUILD_OPTIONS** -- the [options](https://docs.docker.com/engine/reference/commandline/build/#options) for the `docker build` command. For example, `--pull`.
* **BUILD_IN_CONTAINER** -- By default, to compile the controller we use the [golang](https://hub.docker.com/_/golang/) container that we run as part of the building process. If you want to compile the controller using your local golang environment:
  1. Make sure that the Ingress controller repo is in your `$GOPATH`.
  1. Specify `BUILD_IN_CONTAINER=0` when you run the make command.



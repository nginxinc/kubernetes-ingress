
[![Build Status](https://travis-ci.org/nginxinc/kubernetes-ingress.svg?branch=master)](https://travis-ci.org/nginxinc/kubernetes-ingress)  [![FOSSA Status](https://app.fossa.io/api/projects/custom%2B1062%2Fgithub.com%2Fnginxinc%2Fkubernetes-ingress.svg?type=shield)](https://app.fossa.io/projects/custom%2B1062%2Fgithub.com%2Fnginxinc%2Fkubernetes-ingress?ref=badge_shield)  [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)  [![Go Report Card](https://goreportcard.com/badge/github.com/nginxinc/kubernetes-ingress)](https://goreportcard.com/report/github.com/nginxinc/kubernetes-ingress)

# NGINX Ingress Controller

This repo provides an implementation of an Ingress controller for NGINX and NGINX Plus. 

**Note**: this project is different from the NGINX Ingress controller in [kubernetes/ingress-nginx](https://github.com/kubernetes/ingress-nginx) repo. See [this doc](docs/nginx-ingress-controllers.md) to find out about the key differences.

## What is the Ingress?

The Ingress is a Kubernetes resource that lets you configure an HTTP load balancer for applications running on Kubernetes, represented by one or more [Services](https://kubernetes.io/docs/concepts/services-networking/service/). Such a load balancer is necessary to deliver those applications to clients outside of the Kubernetes cluster.

The Ingress resource supports the following features:
* **Content-based routing**:
    * *Host-based routing*. For example, routing requests with the host header `foo.example.com` to one group of services and the host header `bar.example.com` to another group.
    * *Path-based routing*. For example, routing requests with the URI that starts with `/serviceA` to service A and requests with the URI that starts with `/serviceB` to service B.
* **TLS/SSL termination** for each hostname, such as `foo.example.com`.

See the [Ingress User Guide](http://kubernetes.io/docs/user-guide/ingress/) to learn more about the Ingress resource.

## What is the Ingress Controller?

The Ingress controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources. The load balancer can be a software load balancer running in the cluster or a hardware or cloud load balancer running externally. Different load balancers require different Ingress controller implementations. 

In the case of NGINX, the Ingress controller is deployed in a pod along with the load balancer.

## NGINX Ingress Controller

NGINX Ingress controller works with both NGINX and NGINX Plus and supports the standard Ingress features - content-based routing and TLS/SSL termination.

Additionally, several NGINX and NGINX Plus features are available as extensions to the Ingress resource via annotations and the ConfigMap resource. In addition to HTTP, NGINX Ingress controller supports load balancing Websocket, gRPC, TCP and UDP applications. See [ConfigMap and Annotations doc](docs/configmap-and-annotations.md) to learn more about the supported features and customization options.

As an alternative to the Ingress, NGINX Ingress controller supports the VirtualServer and VirtualServerRoute resources. They enable use cases not supported with the Ingress resource, such as traffic splitting and advanced content-based routing. See [VirtualServer and VirtualServerRoute Resources doc](docs/virtualserver-and-virtualserverroute.md).

Read [this doc](docs/nginx-plus.md) to learn more about NGINX Ingress controller with NGINX Plus.

## Getting Started

1. Install the NGINX Ingress controller using the Kubernetes [manifests](deployments) or the [helm chart](deployments/helm-chart).
1. Configure load balancing for a simple web application:
    * Use the Ingress resource. See the [Cafe example](examples/complete-example).
    * Or the VirtualServer resource. See the [Basic configuration](examples-of-custom-resources/basic-configuration) example.
1. See additional configuration [examples](examples).
1. Learn more about all available configuration and customization in the [docs](docs).


## NGINX Ingress Controller Releases

We publish Ingress controller releases on GitHub. See our [releases page](https://github.com/nginxinc/kubernetes-ingress/releases).

The latest stable release is [1.5.7](https://github.com/nginxinc/kubernetes-ingress/releases/tag/v1.5.7). For production use, we recommend that you choose the latest stable release.  As an alternative, you can choose the *edge* version built from the [latest commit](https://github.com/nginxinc/kubernetes-ingress/commits/master) from the master branch. The edge version is useful for experimenting with new features that are not yet published in a stable release.

To use the Ingress controller, you need to have access to:
* An Ingress controller image.
* Installation manifests or a Helm chart.
* Documentation and examples.

It is important that the versions of those things above match. 

The table below summarizes the options regarding the images, manifests, helm chart, documentation and examples and gives your links to the correct versions:

| Version | Description |  Image for NGINX | Image for NGINX Plus | Installation Manifests and Helm Chart | Documentation and Examples |
| ------- | ----------- | --------------- | -------------------- | ---------------------------------------| -------------------------- |
| Latest stable release | For production use | `nginx/nginx-ingress:1.5.7`, `nginx/nginx-ingress:1.5.7-alpine` from [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/) or [build your own image](https://github.com/nginxinc/kubernetes-ingress/tree/v1.5.7/build). | [Build your own image](https://github.com/nginxinc/kubernetes-ingress/tree/v1.5.7/build). | [Manifests](https://github.com/nginxinc/kubernetes-ingress/tree/v1.5.7/deployments). [Helm chart](https://github.com/nginxinc/kubernetes-ingress/tree/v1.5.7/deployments/helm-chart). | [Documentation](https://github.com/nginxinc/kubernetes-ingress/tree/v1.5.7/docs). [Examples](https://github.com/nginxinc/kubernetes-ingress/tree/v1.5.7/examples). |
| Edge | For testing and experimenting | `nginx/nginx-ingress:edge`, `nginx/nginx-ingress:edge-alpine` from [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/) or [build your own image](https://github.com/nginxinc/kubernetes-ingress/tree/master/build). | [Build your own image](https://github.com/nginxinc/kubernetes-ingress/tree/master/build). | [Manifests](https://github.com/nginxinc/kubernetes-ingress/tree/master/deployments). [Helm chart](https://github.com/nginxinc/kubernetes-ingress/tree/master/deployments/helm-chart). | [Documentation](https://github.com/nginxinc/kubernetes-ingress/tree/master/docs). [Examples](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples). |

## Contacts

We’d like to hear your feedback! If you have any suggestions or experience issues with our Ingress controller, please create an issue or send a pull request on Github.
You can contact us directly via [kubernetes@nginx.com](mailto:kubernetes@nginx.com).

## Contributing

If you'd like to contribute to the project, please read our [Contributing guide](CONTRIBUTING.md).

## Support 

For NGINX Plus customers NGINX Ingress controller (when used with NGINX Plus) is covered 
by the support contract.

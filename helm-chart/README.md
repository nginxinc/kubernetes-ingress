# NGINX Ingress Controller Helm Chart

## Introduction

This chart deploys the NGINX Ingress controller in your Kubernetes cluster. 

## Prerequisites

  - Kubernetes 1.6+.
  - If you’d like to use NGINX Plus, you need to build your own Docker image and push it to your private registry by following the instructions from [here](../nginx-controller/README.md).

## Installing the Chart

To install the chart with the release name my-release:

For NGINX: 
```console
$ git clone git@github.com:nginxinc/kubernetes-ingress.git
$ helm install --name my-release kubernetes-ingress/helm-chart/
```

For NGINX Plus:
```console
$ git clone git@github.com:nginxinc/kubernetes-ingress.git
$ helm install --name my-release -f kubernetes-ingress/helm-chart/values-plus.yaml kubernetes-ingress/helm-chart/
```

The command deploys the Ingress controller in your Kubernetes cluster in the default configuration. The configuration section lists the parameters that can be configured during installation.

> **Tip**: List all releases using helm list

## Uninstalling the Chart

To uninstall/delete the my-release

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following tables lists the configurable parameters of the NGINX Ingress controller  chart and their default values.

Parameter | Description | Default
--- | --- | ---
`controller.name` | The name of the Ingress controller daemon set or deployment. | nginx-ingress
`controller.kind` | The kind of the Ingress controller installation - deployment or daemonset. | deployment
`controller.nginxplus` | Should NGINX Plus be deployed. | false
`controller.hostNetwork` | If the nginx deployment / daemonset should run on the host's network namespace. | false
`controller.image.repository` | The image repository of the Ingress controller. | nginx/nginx-ingress
`controller.image.tag` | The tag of the Ingress controller image. | 1.1.1
`controller.image.pullPolicy` | The pull policy for the Ingress controller image. | IfNotPresent
`controller.config.entries` | The entries of the ConfigMap for customizing NGINX configuration. | { }
`controller.defaultTLS.cert` | The TLS certificate for the default HTTPS server. | None
`controller.defaultTLS.key` | The  TLS key for the default HTTPS server. | None
`controller.nodeselector` | The node selectorlabels for pod assignment for the Ingress controller pods. | { }
`controller.terminationGracePeriodSeconds` | The termination grace period of the Ingress controller pod. | 30
`controller.tolerations` | The tolerations required for the IBM Cloud Platform installation. | None



## Limitations

This is a preview version of our helm chart. It has limitations including support for cloud installations (except for the IBM Cloud Platform) and RBAC.  This version is not suitable for using in production environments.




# NGINX App Protect DoS Arbitrator Helm Chart

## Introduction

This chart deploys the NGINX App Protect DoS Arbitrator in your Kubernetes cluster.

## Prerequisites

  - A [Kubernetes Version Supported by the Ingress Controller](https://docs.nginx.com/nginx-ingress-controller/technical-specifications/#supported-kubernetes-versions)
  - Helm 3.0+.
  - Git.

## Getting the Chart Sources

This step is required if you're installing the chart using its sources. Additionally, the step is also required for managing the custom resource definitions (CRDs), which the Ingress Controller requires by default, or for upgrading/deleting the CRDs.

1. Clone the Ingress Controller repo:
    ```console
    $ git clone https://github.com/nginxinc/kubernetes-ingress.git --branch v2.4.1
    ```
2. Change your working directory to /deployments/helm-chart-dos-arbitrator:
    ```console
    $ cd kubernetes-ingress/deployments/helm-chart-dos-arbitrator
    ```

## Adding the Helm Repository

This step is required if you're installing the chart via the helm repository.

```console
$ helm repo add nginx-stable https://helm.nginx.com/stable
$ helm repo update
```

## Installing the Chart

### Installing via Helm Repository

To install the chart with the release name my-release-dos (my-release-dos is the name that you choose):

```console
$ helm install my-release-dos nginx-stable/nginx-appprotect-dos-arbitrator
```


### Installing Using Chart Sources

To install the chart with the release name my-release-dos (my-release-dos is the name that you choose):

```console
$ helm install my-release-dos .
```

The command deploys the App Protect DoS Arbitrator in your Kubernetes cluster in the default configuration. The configuration section lists the parameters that can be configured during installation.

## Upgrading the Chart

### Upgrading the Release

To upgrade the release `my-release-dos`:

#### Upgrade Using Chart Sources:

```console
$ helm upgrade my-release-dos .
```

#### Upgrade via Helm Repository:

```console
$ helm upgrade my-release-dos nginx-stable/nginx-appprotect-dos-arbitrator
```

## Uninstalling the Chart

### Uninstalling the Release

To uninstall/delete the release `my-release-dos`:

```console
$ helm uninstall my-release-dos
```

The command removes all the Kubernetes components associated with the release and deletes the release.

## Configuration

The following tables lists the configurable parameters of the NGINX App Protect DoS Arbitrator chart and their default values.

Parameter | Description | Default
--- | --- | ---
`arbitrator.resources` | The resources of the Arbitrator pods. | limits:<br>cpu: 500m<br>memory: 128Mi
`arbitrator.image.repository` | The image repository of the Arbitrator image. | docker-registry.nginx.com/nap-dos/app_protect_dos_arb
`arbitrator.image.tag` | The tag of the Arbitrator image. | 1.1.0
`arbitrator.image.pullPolicy` | The pull policy for the Arbitrator image. | IfNotPresent

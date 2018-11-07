# NGINX Ingress Controller Helm Chart

## Introduction

This chart deploys the NGINX Ingress controller in your Kubernetes cluster.

## Prerequisites

  - Kubernetes 1.6+.
  - Helm 2.8.x+.
  - Git.
  - If you’d like to use NGINX Plus:
    - Build an Ingress controller image with NGINX Plus and push it to your private registry by following the instructions from [here](../../build/README.md).
    - Update the `controller.image.repository` field of the `values-plus.yaml` accordingly.

## Installing the Chart

1. Clone the Ingress controller repo:
    ```
    $ git clone https://github.com/nginxinc/kubernetes-ingress/
    ```
2. Change your working directory to /deployments/helm-chart:
    ```
    $ cd kubernetes-ingress/deployments/helm-chart
    ```
3. To install the chart with the release name my-release (my-release is the name that you choose):

    For NGINX:
    ```
    $ helm install --name my-release .
    ```

    For NGINX Plus:
    ```
    $ helm install --name my-release -f values-plus.yaml .
    ```

    The command deploys the Ingress controller in your Kubernetes cluster in the default configuration. The configuration section lists the parameters that can be configured during installation.

    When deploying the Ingress controller, make sure to use your own TLS certificate and key for the default server rather than the default pre-generated ones. Read the [Configuration](#Configuration) section below to see how to configure a TLS certificate and key for the default server. Note that the default server returns the Not Found page with the 404 status code for all requests for domains for which there are no Ingress rules defined.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the release `my-release`

```console
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following tables lists the configurable parameters of the NGINX Ingress controller chart and their default values.

Parameter | Description | Default
--- | --- | ---
`controller.name` | The name of the Ingress controller daemon set or deployment. | nginx-ingress
`controller.kind` | The kind of the Ingress controller installation - deployment or daemonset. | deployment
`controller.nginxplus` | Deploys the Ingress controller for NGINX Plus. | false
`controller.hostNetwork` | Enables the Ingress controller pods to use the host's network namespace. | false
`controller.nginxDebug` | Enables debugging for NGINX. Uses the `nginx-debug` binary. Requires `error-log-level: debug` in the ConfigMap via `controller.config.entries`. | false
`controller.image.repository` | The image repository of the Ingress controller. | nginx/nginx-ingress
`controller.image.tag` | The tag of the Ingress controller image. | edge
`controller.image.pullPolicy` | The pull policy for the Ingress controller image. | IfNotPresent
`controller.config.entries` | The entries of the ConfigMap for customizing NGINX configuration. | { }
`controller.defaultTLS.cert` | The base64-encoded TLS certificate for the default HTTPS server. If not specified, a pre-generated self-signed certificate is used. **Note:** It is recommended that you specify your own certificate. | A pre-generated self-signed certificate.
`controller.defaultTLS.key` | The base64-encoded TLS key for the default HTTPS server. **Note:** If not specified, a pre-generated key is used. It is recommended that you specify your own key. | A pre-generated key.
`controller.defaultTLS.secret` | The secret with a TLS certificate and key for the default HTTPS server. The value must follow the following format: `<namespace>/<name>`. Used as an alternative to specifiying a certifcate and key using `controller.defaultTLS.cert` and `controller.defaultTLS.key` parameters. | None
`controller.nodeSelector` | The node selector for pod assignment for the Ingress controller pods. | { }
`controller.terminationGracePeriodSeconds` | The termination grace period of the Ingress controller pod. | 30
`controller.tolerations` | The tolerations required for the IBM Cloud Private installation. | None
`controller.replicaCount` | The number of replicas of the Ingress controller deployment. | 1
`controller.service.create` | Creates a service to expose the Ingress controller pods. | true
`controller.service.type` | The type of service to create for the Ingress controller. | LoadBalancer
`controller.service.externalTrafficPolicy` | The externalTrafficPolicy of the service. The value Local preserves the client source IP. | Local
`controller.service.annotations` | The annotations of the Ingress controller service. | { }
`controller.service.loadBalancerIP` | The static IP address for the load balancer. Requires `controller.service.type` set to `LoadBalancer`.  | None
`controller.service.externalIPs` | The list of external IPs for the Ingress controller service. | []
`controller.serviceAccount.name` | The serviceAccountName of the Ingress controller pods. Used for RBAC. | nginx-ingress
`controller.serviceAccount.imagePullSecrets.name` | The name of the secret containing docker registry credentials. | None
`controller.ingressClass` | A class of the Ingress controller. The Ingress controller only processes Ingress resources that belong to its class - i.e. have the annotation `"kubernetes.io/ingress.class"` equal to the class. Additionally, the Ingress controller processes Ingress resources that do not have that annotation which can be disabled by setting the "-use-ingress-class-only" flag. | nginx
`controller.useIngressClassOnly` | Ignore Ingress resources without the `"kubernetes.io/ingress.class"` annotation. | false
`controller.watchNamespace` | Namespace to watch for Ingress resources. By default the Ingress controller watches all namespaces. | ""
`controller.healthStatus` | Add a location "/nginx-health" to the default server. The location responds with the 200 status code for any request. Useful for external health-checking of the Ingress controller. | false
`controller.nginxStatus.enable` | Enable the NGINX stub_status, or the NGINX Plus API. | true
`controller.nginxStatus.port` | Set the port where the NGINX stub_status or the NGINX Plus API is exposed. | 8080
`controller.nginxStatus.allowCidrs` | Whitelist IPv4 IP/CIDR blocks to allow access to NGINX stub_status or the NGINX Plus API. Separate multiple IP/CIDR by commas. | 127.0.0.1
`controller.reportIngressStatus.enable` | Update the address field in the status of Ingresses resources with an external address of the Ingress controller. You must also specify the source of the external address either through an external service via `controller.reportIngressStatus.externalService` or the `external-status-address` entry in the ConfigMap via `controller.config.entries`. **Note:** `controller.config.entries.external-status-address` takes precedence if both are set. | true
`controller.reportIngressStatus.externalService` | Specifies the name of the service with the type LoadBalancer through which the Ingress controller is exposed externally. The external address of the service is used when reporting the status of Ingress resources. `controller.reportIngressStatus.enable` must be set to `true`. | nginx-ingress
`controller.reportIngressStatus.enableLeaderElection` | Enable Leader election to avoid multiple replicas of the controller reporting the status of Ingress resources. `controller.reportIngressStatus.enable` must be set to `true`. | true
`rbac.create` | Configures RBAC. | true
`prometheues.create` | Deploys a Prometheus exporter container within the Ingress controller pod. Requires NGINX status enabled via `controller.nginxStatus.enable`. Note: the exporter will use the port specified by `controller.nginxStatus.port`.| false
`prometheus.port` | Configures the port to scrape the metrics. | 9113
`prometheus.image.repository` | The image repository of the Prometheus exporter. | nginx/nginx-prometheus-exporter
`prometheus.image.tag` | The tag of the Prometheus exporter image. | 0.1.0
`prometheus.image.pullPolicy` | The pull policy for the Prometheus exporter image. | IfNotPresent

Example:
```
$ cd kubernetes-ingress/helm-chart
$ helm install --name my-release . --set controller.replicaCount=5
```

## Notes
* The values-icp.yaml file is used for deploying the Ingress controller on IBM Cloud Private. See the [blog post](https://www.nginx.com/blog/nginx-ingress-controller-ibm-cloud-private/) for more details.

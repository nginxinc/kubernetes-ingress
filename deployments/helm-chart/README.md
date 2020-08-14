# NGINX Ingress Controller Helm Chart

## Introduction

This chart deploys the NGINX Ingress controller in your Kubernetes cluster.

## Prerequisites

  - A [Kubernetes Version Supported by the Ingress Controller](https://docs.nginx.com/nginx-ingress-controller/technical-specifications/#supported-kubernetes-versions)
  - Helm 2.16+ or 3.0+.
  - Git.
  - If you’d like to use NGINX Plus:
    - Build an Ingress controller image with NGINX Plus and push it to your private registry by following the instructions from [here](../../build/README.md).
    - Update the `controller.image.repository` field of the `values-plus.yaml` accordingly.

## Getting the Chart Sources

This step is required if you're installing the chart using its sources. Additionally, the step is also required for managing the custom resource definitions (CRDs), which the Ingress Controller requires by default: upgrading/deleting the CRDs, or installing the CRDs for Helm 2.x.

1. Clone the Ingress controller repo:
    ```console
    $ git clone https://github.com/nginxinc/kubernetes-ingress/
    $ git checkout v1.8.1
    ```
2. Change your working directory to /deployments/helm-chart:
    ```console
    $ cd kubernetes-ingress/deployments/helm-chart
    ```

## Adding the Helm Repository

This step is required if you're installing the chart via the helm repository.

```console
$ helm repo add nginx-stable https://helm.nginx.com/stable
$ helm repo update
```

## Installing the Chart

### Installing the CRDs

**Note**: If you're using Kubernetes 1.14, make sure to add `--validate=false` to the `kubectl create` command below. Otherwise, you will get an error validating data:
```
ValidationError(CustomResourceDefinition.spec): unknown field "preserveUnknownFields" in io.k8s.apiextensions-apiserver.pkg.apis.api extensions.v1beta1.CustomResourceDefinitionSpec
```

By default, the Ingress Controller requires a number of custom resource definitions (CRDs) installed in the cluster. Helm 3.x client will install those CRDs. If you're using a Helm 2.x client, you need to install the CRDs via `kubectl`:

```console
$ kubectl create -f crds/
```

If you do not use the custom resources that require those CRDs (which corresponds to `controller.enableCustomResources` set to `false` and `controller.appprotect.enable` set to `false`), you can skip the installation of the CRDs. For Helm 2.x, no action is needed, as it does not install the CRDs. For Helm 3.x, specify `--skip-crds` for the helm install command.

### Installing via Helm Repository

To install the chart with the release name my-release (my-release is the name that you choose):

* Using Helm 3.x client:

    For NGINX:
    ```console
    $ helm install my-release nginx-stable/nginx-ingress
    ```

    For NGINX Plus: (assuming you have pushed the Ingress controller image `nginx-plus-ingress` to your private registry `myregistry.example.com`)
    ```console
    $ helm install my-release nginx-stable/nginx-ingress --set controller.image.repository=myregistry.example.com/nginx-plus-ingress --set controller.nginxplus=true
    ```

* Using Helm 2.x client:

    For NGINX:
    ```console
    $ helm install --name my-release nginx-stable/nginx-ingress
    ```

    For NGINX Plus: (assuming you have pushed the Ingress controller image `nginx-plus-ingress` to your private registry `myregistry.example.com`)
    ```console
    $ helm install --name my-release nginx-stable/nginx-ingress --set controller.image.repository=myregistry.example.com/nginx-plus-ingress --set controller.nginxplus=true
    ```

### Installing Using Chart Sources

To install the chart with the release name my-release (my-release is the name that you choose):

* Using Helm 3.x client:

    For NGINX:
    ```console
    $ helm install my-release .
    ```

    For NGINX Plus:
    ```console
    $ helm install my-release -f values-plus.yaml .
    ```

* Using Helm 2.x client:

    For NGINX:
    ```console
    $ helm install --name my-release .
    ```

    For NGINX Plus:
    ```console
    $ helm install --name my-release -f values-plus.yaml .
    ```

    The command deploys the Ingress controller in your Kubernetes cluster in the default configuration. The configuration section lists the parameters that can be configured during installation.

    When deploying the Ingress controller, make sure to use your own TLS certificate and key for the default server rather than the default pre-generated ones. Read the [Configuration](#Configuration) section below to see how to configure a TLS certificate and key for the default server. Note that the default server returns the Not Found page with the 404 status code for all requests for domains for which there are no Ingress rules defined.

> **Tip**: List all releases using `helm list`

## Upgrading the Chart

### Upgrading the CRDs

**Note**: If you're using Kubernetes 1.14, make sure to add `--validate=false` to the `kubectl apply` command below.

Helm does not upgrade the CRDs during a release upgrade. Before you upgrade a release, run the following command to upgrade the CRDs:

```console
$ kubectl apply -f crds/
```
> **Note**: The following warning is expected and can be ignored: `Warning: kubectl apply should be used on resource created by either kubectl create --save-config or kubectl apply`.

> **Note**: Make sure to check the [release notes](https://www.github.com/nginxinc/kubernetes-ingress/releases) for a new release for any special upgrade procedures.

### Upgrading the Release

To upgrade the release `my-release`:

#### Upgrade Using Chart Sources:

```console
$ helm upgrade my-release .
```

#### Upgrade via Helm Repository:

```console
$ helm upgrade my-release nginx-stable/nginx-ingress
```

## Uninstalling the Chart

### Uninstalling the Release

To uninstall/delete the release `my-release`:

* Using Helm 3.x client:

    ```console
    $ helm uninstall my-release
    ```

* Using Helm 2.x client:

    ```console
    $ helm delete --purge my-release
    ```

The command removes all the Kubernetes components associated with the release and deletes the release.

### Uninstalling the CRDs

Uninstalling the release does not remove the CRDs. To remove the CRDs, run:

```console
$ kubectl delete -f crds/
```
> **Note**: This command will delete all the corresponding custom resources in your cluster across all namespaces. Please ensure there are no custom resources that you want to keep and there are no other Ingress Controller releases running in the cluster.

## Running Multiple Ingress Controllers

If you are running multiple Ingress Controller releases in your cluster with enabled custom resources, the releases will share a single version of the CRDs. As a result, make sure that the Ingress Controller versions match the version of the CRDs. Additionally, when uninstalling a release, ensure that you don’t remove the CRDs until there are no other Ingress Controller releases running in the cluster.

See [running multiple ingress controllers](https://docs.nginx.com/nginx-ingress-controller/installation/running-multiple-ingress-controllers/) for more details.

## Configuration

The following tables lists the configurable parameters of the NGINX Ingress controller chart and their default values.

Parameter | Description | Default
--- | --- | ---
`controller.name` | The name of the Ingress controller daemonset or deployment. | Autogenerated
`controller.kind` | The kind of the Ingress controller installation - deployment or daemonset. | deployment
`controller.nginxplus` | Deploys the Ingress controller for NGINX Plus. | false
`controller.nginxReloadTimeout` | The timeout in milliseconds which the Ingress Controller will wait for a successful NGINX reload after a change or at the initial start. The default is 4000 (or 20000 if `controller.appprotect.enable` is true). If set to 0, the default value will be used. | 0
`controller.hostNetwork` | Enables the Ingress controller pods to use the host's network namespace. | false
`controller.nginxDebug` | Enables debugging for NGINX. Uses the `nginx-debug` binary. Requires `error-log-level: debug` in the ConfigMap via `controller.config.entries`. | false
`controller.logLevel` | The log level of the Ingress Controller. | 1
`controller.image.repository` | The image repository of the Ingress controller. | nginx/nginx-ingress
`controller.image.tag` | The tag of the Ingress controller image. | 1.8.1
`controller.image.pullPolicy` | The pull policy for the Ingress controller image. | IfNotPresent
`controller.config.name` | The name of the ConfigMap used by the Ingress controller. | Autogenerated
`controller.config.annotations` | The annotations of the Ingress controller configmap. | {}
`controller.config.entries` | The entries of the ConfigMap for customizing NGINX configuration. | {}
`controller.customPorts` | A list of custom ports to expose on the NGINX ingress controller pod. Follows the conventional Kubernetes yaml syntax for container ports. | []
`controller.defaultTLS.cert` | The base64-encoded TLS certificate for the default HTTPS server. If not specified, a pre-generated self-signed certificate is used. **Note:** It is recommended that you specify your own certificate. | A pre-generated self-signed certificate.
`controller.defaultTLS.key` | The base64-encoded TLS key for the default HTTPS server. **Note:** If not specified, a pre-generated key is used. It is recommended that you specify your own key. | A pre-generated key.
`controller.defaultTLS.secret` | The secret with a TLS certificate and key for the default HTTPS server. The value must follow the following format: `<namespace>/<name>`. Used as an alternative to specifying a certificate and key using `controller.defaultTLS.cert` and `controller.defaultTLS.key` parameters. | None
`controller.wildcardTLS.cert` | The base64-encoded TLS certificate for every Ingress host that has TLS enabled but no secret specified. If the parameter is not set, for such Ingress hosts NGINX will break any attempt to establish a TLS connection. | None
`controller.wildcardTLS.key` | The base64-encoded TLS key for every Ingress host that has TLS enabled but no secret specified. If the parameter is not set, for such Ingress hosts NGINX will break any attempt to establish a TLS connection. | None
`controller.wildcardTLS.secret` | The secret with a TLS certificate and key for every Ingress host that has TLS enabled but no secret specified. The value must follow the following format: `<namespace>/<name>`. Used as an alternative to specifying a certificate and key using `controller.wildcardTLS.cert` and `controller.wildcardTLS.key` parameters. | None
`controller.nodeSelector` | The node selector for pod assignment for the Ingress controller pods. | {}
`controller.terminationGracePeriodSeconds` | The termination grace period of the Ingress controller pod. | 30
`controller.tolerations` | The tolerations of the Ingress controller pods. | []
`controller.affinity` | The affinity of the Ingress controller pods. | {}
`controller.volumes` | The volumes of the Ingress controller pods. | []
`controller.volumeMounts` | The volumeMounts of the Ingress controller pods. | []
`controller.resources` | The resources of the Ingress controller pods. | {}
`controller.replicaCount` | The number of replicas of the Ingress controller deployment. | 1
`controller.ingressClass` | A class of the Ingress controller. The Ingress controller only processes Ingress resources that belong to its class - i.e. have the annotation `"kubernetes.io/ingress.class"` or the `"ingressClassName"` field in VirtualServer/VirtualServerRoute equal to the class. Additionally, the Ingress controller processes Ingress resources that do not have that annotation which can be disabled by setting the "-use-ingress-class-only" flag. | nginx
`controller.useIngressClassOnly` | Ignore Ingress resources without the `"kubernetes.io/ingress.class"` annotation or the `"ingressClassName"` field in VirtualServer/VirtualServerRoute. | false
`controller.watchNamespace` | Namespace to watch for Ingress resources. By default the Ingress controller watches all namespaces. | ""
`controller.enableCustomResources` | Enable the custom resources. | true
`controller.enableTLSPassthrough` | Enable TLS Passthrough on port 443. Requires `controller.enableCustomResources`. | false
`controller.globalConfiguration.create` | Creates the GlobalConfiguration custom resource. Requires `controller.enableCustomResources`. | false
`controller.globalConfiguration.spec` | The spec of the GlobalConfiguration for defining the global configuration parameters of the Ingress Controller. | {}
`controller.enableSnippets` | Enable custom NGINX configuration snippets in VirtualServer and VirtualServerRoute resources. | false
`controller.healthStatus` | Add a location "/nginx-health" to the default server. The location responds with the 200 status code for any request. Useful for external health-checking of the Ingress controller. | false
`controller.healthStatusURI` | Sets the URI of health status location in the default server. Requires `controller.healthStatus`. | "/nginx-health"
`controller.nginxStatus.enable` | Enable the NGINX stub_status, or the NGINX Plus API. | true
`controller.nginxStatus.port` | Set the port where the NGINX stub_status or the NGINX Plus API is exposed. | 8080
`controller.nginxStatus.allowCidrs` | Whitelist IPv4 IP/CIDR blocks to allow access to NGINX stub_status or the NGINX Plus API. Separate multiple IP/CIDR by commas. | 127.0.0.1
`controller.priorityClassName` | The PriorityClass of the Ingress controller pods. | None
`controller.service.create` | Creates a service to expose the Ingress controller pods. | true
`controller.service.type` | The type of service to create for the Ingress controller. | LoadBalancer
`controller.service.externalTrafficPolicy` | The externalTrafficPolicy of the service. The value Local preserves the client source IP. | Local
`controller.service.annotations` | The annotations of the Ingress controller service. | {}
`controller.service.loadBalancerIP` | The static IP address for the load balancer. Requires `controller.service.type` set to `LoadBalancer`. The cloud provider must support this feature. | ""
`controller.service.externalIPs` | The list of external IPs for the Ingress controller service. | []
`controller.service.loadBalancerSourceRanges` | The IP ranges (CIDR) that are allowed to access the load balancer. Requires `controller.service.type` set to `LoadBalancer`. The cloud provider must support this feature. | []
`controller.service.name` | The name of the service. | Autogenerated
`controller.service.customPorts` | A list of custom ports to expose through the Ingress controller service. Follows the conventional Kubernetes yaml syntax for service ports. | []
`controller.service.httpPort.enable` | Enables the HTTP port for the Ingress controller service. | true
`controller.service.httpPort.port` | The HTTP port of the Ingress controller service. | 80
`controller.service.httpPort.nodePort` | The custom NodePort for the HTTP port. Requires `controller.service.type` set to `NodePort`. | ""
`controller.service.httpPort.targetPort` | The target port of the HTTP port of the Ingress controller service. | 80
`controller.service.httpsPort.enable` | Enables the HTTPS port for the Ingress controller service. | true
`controller.service.httpsPort.port` | The HTTPS port of the Ingress controller service. | 443
`controller.service.httpsPort.nodePort` | The custom NodePort for the HTTPS port. Requires `controller.service.type` set to `NodePort`.  | ""
`controller.service.httpsPort.targetPort` | The target port of the HTTPS port of the Ingress controller service. | 443
`controller.serviceAccount.name` | The name of the service account of the Ingress controller pods. Used for RBAC. | Autogenerated
`controller.serviceAccount.imagePullSecrets` | The names of the secrets containing docker registry credentials. | []
`controller.reportIngressStatus.enable` | Update the address field in the status of Ingresses resources with an external address of the Ingress controller. You must also specify the source of the external address either through an external service via `controller.reportIngressStatus.externalService` or the `external-status-address` entry in the ConfigMap via `controller.config.entries`. **Note:** `controller.config.entries.external-status-address` takes precedence if both are set. | true
`controller.reportIngressStatus.externalService` | Specifies the name of the service with the type LoadBalancer through which the Ingress controller is exposed externally. The external address of the service is used when reporting the status of Ingress resources. `controller.reportIngressStatus.enable` must be set to `true`. The default is autogenerated and enabled when `controller.service.create` is set to `true` and `controller.service.type` is set to `LoadBalancer`. | Autogenerated
`controller.reportIngressStatus.enableLeaderElection` | Enable Leader election to avoid multiple replicas of the controller reporting the status of Ingress resources. `controller.reportIngressStatus.enable` must be set to `true`. | true
`controller.reportIngressStatus.leaderElectionLockName` | Specifies the name of the ConfigMap, within the same namespace as the controller, used as the lock for leader election. controller.reportIngressStatus.enableLeaderElection must be set to true. | Autogenerated
`controller.reportIngressStatus.annotations` | The annotations of the leader election configmap. | {}
`controller.pod.annotations` | The annotations of the Ingress Controller pod. | {}
`controller.appprotect.enable` | Enables the App Protect module in the Ingress Controller. | false
`controller.readyStatus.enable` | Enables the readiness endpoint `"/nginx-ready"`. The endpoint returns a success code when NGINX has loaded all the config after the startup. This also configures a readiness probe for the Ingress Controller pods that uses the readiness endpoint. | true
`controller.readyStatus.port` | The HTTP port for the readiness endpoint. | 8081
`rbac.create` | Configures RBAC. | true
`prometheus.create` | Expose NGINX or NGINX Plus metrics in the Prometheus format. | false
`prometheus.port` | Configures the port to scrape the metrics. | 9113


## Notes
* The values-icp.yaml file is used for deploying the Ingress controller on IBM Cloud Private. See the [blog post](https://www.nginx.com/blog/nginx-ingress-controller-ibm-cloud-private/) for more details.

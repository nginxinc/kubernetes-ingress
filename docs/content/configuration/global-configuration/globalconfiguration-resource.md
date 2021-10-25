---
title: GlobalConfiguration Resource

description: 
weight: 2000
doctypes: [""]
toc: true
---


The GlobalConfiguration resource allows you to define the global configuration parameters of the Ingress Controller. The resource is implemented as a [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

The resource supports configuring listeners for TCP and UDP load balancing. Listeners are required by [TransportServer resources](/nginx-ingress-controller/configuration/transportserver-resource).

> **Feature Status**: The GlobalConfiguration resource is available as a preview feature[^1]: We might introduce some backward-incompatible changes to the resource definition. The feature is disabled by default. To enable it, set the [enable-preview-policies](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-preview-policies) command-line argument of the Ingress Controller.

## Prerequisites

When [installing](/nginx-ingress-controller/installation/installation-with-manifests) the Ingress Controller, you need to reference a GlobalConfiguration resource in the [`-global-configuration`](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments#cmdoption-global-configuration) command-line argument. The Ingress Controller only needs one GlobalConfiguration resource.

## GlobalConfiguration Specification

The GlobalConfiguration resource defines the global configuration parameters of the Ingress Controller. Below is an example:
```yaml
apiVersion: k8s.nginx.org/v1alpha1
kind: GlobalConfiguration 
metadata:
  name: nginx-configuration
  namespace: nginx-ingress
spec:
  listeners:
  - name: dns-udp
    port: 5353
    protocol: UDP
  - name: dns-tcp
    port: 5353
    protocol: TCP
``` 

{{% table %}} 
|Field | Description | Type | Required | 
| ---| ---| ---| --- | 
|``listeners`` | A list of listeners. | [[]listener](#listener) | No | 
{{% /table %}} 

### Listener

The listener defines a listener (a combination of a protocol and a port) that NGINX will use to accept traffic for a [TransportServer](/nginx-ingress-controller/configuration/transportserver-resource):
```yaml
name: dns-tcp
port: 5353
protocol: TCP
```

{{% table %}} 
|Field | Description | Type | Required | 
| ---| ---| ---| --- | 
|``name`` | The name of the listener. Must be a valid DNS label as defined in RFC 1035. For example, ``hello`` and ``listener-123`` are valid. The name must be unique among all listeners. The name ``tls-passthrough`` is reserved for the built-in TLS Passthrough listener and cannot be used. | ``string`` | Yes | 
|``port`` | The port of the listener. The port must fall into the range ``1..65535`` with the following exceptions: ``80``, ``443``, the [status port](/nginx-ingress-controller/logging-and-monitoring/status-page), the [Prometheus metrics port](/nginx-ingress-controller/logging-and-monitoring/prometheus). Among all listeners, only a single combination of a port-protocol is allowed. | ``int`` | Yes | 
|``protocol`` | The protocol of the listener. Supported values: ``TCP`` and ``UDP``. | ``string`` | Yes | 
{{% /table %}} 

## Using GlobalConfiguration 

You can use the usual `kubectl` commands to work with a GlobalConfiguration resource.

For example, the following command creates a GlobalConfiguration resource defined in `global-configuration.yaml` with the name `nginx-configuration`:
```
$ kubectl apply -f global-configuration.yaml
globalconfiguration.k8s.nginx.org/nginx-configuration created
```

Assuming the namespace of the resource is `nginx-ingress`, you can get the resource by running:
```
$ kubectl get globalconfiguration nginx-configuration -n nginx-ingress
NAME                  AGE
nginx-configuration   13s
```

In the kubectl get and similar commands, you can also use the short name `gc` instead of `globalconfiguration`.

### Validation

Two types of validation are available for the GlobalConfiguration resource:
* *Structural validation* by the `kubectl` and Kubernetes API server.
* *Comprehensive validation* by the Ingress Controller.

#### Structural Validation

The custom resource definition for the GlobalConfiguration includes structural OpenAPI schema which describes the type of every field of the resource.

If you try to create (or update) a resource that violates the structural schema (for example, you use a string value for the port field of a listener), `kubectl` and Kubernetes API server will reject such a resource:
* Example of `kubectl` validation:
    ```
    $ kubectl apply -f global-configuration.yaml
    error: error validating "global-configuration.yaml": error validating data: ValidationError(GlobalConfiguration.spec.listeners[0].port): invalid type for org.nginx.k8s.v1alpha1.GlobalConfiguration.spec.listeners.port: got "string", expected "integer"; if you choose to ignore these errors, turn validation off with --validate=false
    ```
* Example of Kubernetes API server validation:
    ```
    $ kubectl apply -f global-configuration.yaml --validate=false
      The GlobalConfiguration "nginx-configuration" is invalid: []: Invalid value: map[string]interface {}{ ... }: validation failure list:
      spec.listeners.port in body must be of type integer: "string"
    ```

If a resource is not rejected (it doesn't violate the structural schema), the Ingress Controller will validate it further.

#### Comprehensive Validation

The Ingress Controller validates the fields of a GlobalConfiguration resource. If a resource is invalid, the Ingress Controller will not use it. Consider the following two cases:
1. When the Ingress Controller pod starts, if the GlobalConfiguration resource is invalid, the Ingress Controller will fail to start and exit with an error.
1. When the Ingress Controller is running, if the GlobalConfiguration resource becomes invalid, the Ingress Controller will ignore the new version. It will report an error and continue to use the previous version. When the resource becomes valid again, the Ingress Controller will start using it. 

**Note**: If a GlobalConfiguration is deleted while the Ingress Controller is running, the controller will keep using the previous version of the resource.

You can check if the Ingress Controller successfully applied the configuration for a GlobalConfiguration. For our  `nginx-configuration` GlobalConfiguration, we can run:
```
$ kubectl describe gc nginx-configuration -n nginx-ingress
. . .
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Normal   Updated   11s   nginx-ingress-controller  GlobalConfiguration nginx-ingress/nginx-configuration was updated
```
Note how the events section includes a Normal event with the Updated reason that informs us that the configuration was successfully applied.

If you create an invalid resource, the Ingress Controller will reject it and emit a Rejected event. For example, if you create a GlobalConfiguration `nginx-configuration` with two or more listeners that have the same protocol UDP and port 53, you will get:
```
$ kubectl describe gc nginx-configuration -n nginx-ingress
. . .
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Normal   Updated   55s   nginx-ingress-controller  GlobalConfiguration nginx-ingress/nginx-configuration was updated
  Warning  Rejected  6s    nginx-ingress-controller  GlobalConfiguration nginx-ingress/nginx-configuration is invalid and was rejected: spec.listeners: Duplicate value: "Duplicated port/protocol combination 53/UDP"
```
Note how the events section includes a Warning event with the Rejected reason.

## Footnotes

[^1]: Capabilities labeled in preview status are fully supported.

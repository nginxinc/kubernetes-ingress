# TransportServer Resource

The TransportServer resource allows you to configure TCP, UDP, and TLS Passthrough load balancing. The resource is implemented as a [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

This document is the reference documentation for the TransportServer resource. To see additional examples of using the resource for specific use cases, go to the [examples-of-custom-resources](https://github.com/nginxinc/kubernetes-ingress/blob/v1.7.0-rc1/examples-of-custom-resources) folder in our GitHub repo.

> **Feature Status**: The TransportServer resource is available as a preview feature: it is suitable for experimenting and testing; however, it must be used with caution in production environments. Additionally, while the feature is in preview, we might introduce some backward-incompatible changes to the resource specification in the next releases.

## Contents

- [TransportServer Resource](#transportserver-resource)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [TransportServer Specification](#transportserver-specification)
    - [Listener](#listener)
    - [Upstream](#upstream)
    - [UpstreamParameters](#upstreamparameters)
    - [Action](#action)
  - [Using TransportServer](#using-transportserver)
    - [Validation](#validation)
      - [Structural Validation](#structural-validation)
      - [Comprehensive Validation](#comprehensive-validation)
  - [Customization via ConfigMap](#customization-via-configmap)
  - [Limitations](#limitations)

## Prerequisites

* For TCP and UDP, the TransportServer resource must be used in conjunction with the [GlobalConfiguration resource](/nginx-ingress-controller/configuration/global-configuration/globalconfiguration-resource), which must be created separately.
* For TLS Passthrough, make sure to enable the [`-enable-tls-passthrough`](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments#cmdoption-enable-tls-passthrough) command-line argument of the Ingress Controller.

## TransportServer Specification

The TransportServer resource defines load balancing configuration for TCP, UDP, or TLS Passthrough traffic. Below are a few examples:

* TCP load balancing:
  ```yaml
  apiVersion: k8s.nginx.org/v1alpha1
  kind: TransportServer
  metadata:
    name: dns-tcp
  spec:
    listener:
      name: dns-tcp 
      protocol: TCP
    upstreams:
    - name: dns-app
      service: dns-service
      port: 5353
    action:
      pass: dns-app
  ```
* UDP load balancing:
  ```yaml
  apiVersion: k8s.nginx.org/v1alpha1
  kind: TransportServer
  metadata:
    name: dns-udp
  spec:
    listener:
      name: dns-udp 
      protocol: UDP
    upstreams:
    - name: dns-app
      service: dns-service
      port: 5353
    upstreamParameters:
      udpRequests: 1
      udpResponses: 1
    action:
      pass: dns-app
  ```
* TLS passthrough load balancing:
  ```yaml
  apiVersion: k8s.nginx.org/v1alpha1
  kind: TransportServer
  metadata:
    name: secure-app
  spec:
    listener:
      name: tls-passthrough
      protocol: TLS_PASSTHROUGH
    host: app.example.com
    upstreams:
    - name: secure-app
      service: secure-app
      port: 8443
    action:
      pass: secure-app
  ```

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``listener``
     - The listener on NGINX that will accept incoming connections/datagrams.
     - `listener <#listener>`_
     - Yes
   * - ``host``
     - The host (domain name) of the server. Must be a valid subdomain as defined in RFC 1123, such as ``my-app`` or ``hello.example.com``. Wildcard domains like ``*.example.com`` are not allowed. Required for TLS Passthrough load balancing.
     - ``string``
     - No*
   * - ``upstreams``
     - A list of upstreams.
     - `[]upstream <#upstream>`_
     - Yes
   * - ``upstreamParameters``
     - The upstream parameters.
     - `upstreamParameters <#upstreamparameters>`_
     - No
   * - ``action``
     - The action to perform for a client connection/datagram.
     - `action <#action>`_
     - Yes
```

\* -- Required for TLS Passthrough load balancing.

### Listener

The listener field references a listener that NGINX will use to accept incoming traffic for the TransportServer. For TCP and UDP, the listener must be defined in the [GlobalConfiguration resource](/nginx-ingress-controller/configuration/global-configuration/globalconfiguration-resource). When referencing a listener, both the name and the protocol must match. For TLS Passthrough, use the built-in listener with the name `tls-passthrough` and the protocol `TLS_PASSTHROUGH`. 

An example:
```yaml
listener:
  name: dns-udp 
  protocol: UDP 
```

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``name``
     - The name of the listener.
     - ``string``
     - Yes
   * - ``protocol``
     - The protocol of the listener.
     - ``string``
     - Yes 
```

### Upstream

The upstream defines a destination for the TransportServer. For example:
```yaml
name: secure-app 
service: secure-app
port: 8443
```

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``name``
     - The name of the upstream. Must be a valid DNS label as defined in RFC 1035. For example, ``hello`` and ``upstream-123`` are valid. The name must be unique among all upstreams of the resource.
     - ``string``
     - Yes
   * - ``service``
     - The name of a `service <https://kubernetes.io/docs/concepts/services-networking/service/>`_. The service must belong to the same namespace as the resource. If the service doesn't exist, NGINX will assume the service has zero endpoints and close client connections/ignore datagrams.
     - ``string``
     - Yes
   * - ``port``
     - The port of the service. If the service doesn't define that port, NGINX will assume the service has zero endpoints and close client connections/ignore datagrams. The port must fall into the range ``1..65535``.
     - ``int``
     - Yes
```

### UpstreamParameters

The upstream parameters define various parameters for the upstreams. For now, only UDP-related parameters are supported:
```yaml
upstreamParameters:
  udpRequests: 1
  udpResponses: 1
```

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``udpRequests``
     - The number of datagrams, after receiving which, the next datagram from the same client starts a new session. See the `proxy_requests <https://nginx.org/en/docs/stream/ngx_stream_proxy_module.html#proxy_requests>`_ directive. The default is ``0``.
     - ``int``
     - No
   * - ``udpResponses``
     - The number of datagrams expected from the proxied server in response to a client datagram. See the `proxy_responses <https://nginx.org/en/docs/stream/ngx_stream_proxy_module.html#proxy_responses>`_ directive. By default, the number of datagrams is not limited.
     - ``int``
     - No 
```

### Action

The action defines an action to perform for a client connection/datagram.

In the example below, client connections/datagrams are passed to an upstream `dns-app`:
```yaml
action:
  pass: dns-app
```

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``pass``
     - Passes connections/datagrams to an upstream. The upstream with that name must be defined in the resource.
     - ``string``
     - Yes 
```

## Using TransportServer 

You can use the usual `kubectl` commands to work with TransportServer resources, similar to Ingress resources.

For example, the following command creates a TransportServer resource defined in `transport-server-passthrough.yaml` with the name `secure-app`:
```
$ kubectl apply -f transport-server-passthrough.yaml
transportserver.k8s.nginx.org/secure-app created
```

You can get the resource by running:
```
$ kubectl get transportserver secure-app
NAME         AGE
secure-app   46sm
```

In the kubectl get and similar commands, you can also use the short name `ts` instead of `transportserver`.

### Validation

Two types of validation are available for the TransportServer resource:
* *Structural validation* by the `kubectl` and Kubernetes API server.
* *Comprehensive validation* by the Ingress Controller.

#### Structural Validation

The custom resource definition for the TransportServer includes structural OpenAPI schema which describes the type of every field of the resource.

If you try to create (or update) a resource that violates the structural schema (for example, you use a string value for the port field of an upstream), `kubectl` and Kubernetes API server will reject such a resource:
* Example of `kubectl` validation:
    ```
    $ kubectl apply -f transport-server-passthrough.yaml
      error: error validating "transport-server-passthrough.yaml": error validating data: ValidationError(TransportServer.spec.upstreams[0].port): invalid type for org.nginx.k8s.v1alpha1.TransportServer.spec.upstreams.port: got "string", expected "integer"; if you choose to ignore these errors, turn validation off with --validate=false
    ```
* Example of Kubernetes API server validation:
    ```
    $ kubectl apply -f transport-server-passthrough.yaml --validate=false
      The TransportServer "secure-app" is invalid: []: Invalid value: map[string]interface {}{ ... }: validation failure list:
      spec.upstreams.port in body must be of type integer: "string"
    ```

If a resource is not rejected (it doesn't violate the structural schema), the Ingress Controller will validate it further.

#### Comprehensive Validation

The Ingress Controller validates the fields of a TransportServer resource. If a resource is invalid, the Ingress Controller will reject it: the resource will continue to exist in the cluster, but the Ingress Controller will ignore it.

You can check if the Ingress Controller successfully applied the configuration for a TransportServer. For our example `secure-app` TransportServer, we can run:
```
$ kubectl describe ts secure-app
. . .
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  3s    nginx-ingress-controller  Configuration for default/secure-app was added or updated
```
Note how the events section includes a Normal event with the AddedOrUpdated reason that informs us that the configuration was successfully applied.

If you create an invalid resource, the Ingress Controller will reject it and emit a Rejected event. For example, if you create a TransportServer `secure-app` with a pass action that references a non-existing upstream, you will get  :
```
$ kubectl describe ts secure-app
. . .
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Warning  Rejected  2s    nginx-ingress-controller  TransportServer default/secure-app is invalid and was rejected: spec.action.pass: Not found: "some-app"
```
Note how the events section includes a Warning event with the Rejected reason.

**Note**: If you make an existing resource invalid, the Ingress Controller will reject it and remove the corresponding configuration from NGINX.

## Customization via ConfigMap

The [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource) keys (except for `stream-snippets` and `stream-log-format`) do not affect TransportServer resources.

## Limitations

As of Release 1.7, the TransportServer resource is a preview feature. Currently, it comes with the following limitations:
* When using TLS Passthrough, it is not possible to configure [Proxy Protocol](https://github.com/nginxinc/kubernetes-ingress/tree/v1.7.0-rc1/examples/proxy-protocol) for port 443 both for regular HTTPS and TLS Passthrough traffic.
* If multiple TCP (or UDP) TransportServers reference the same listener, only one of them will receive the traffic. Moreover, until there is only one TransportServer, NGINX will fail to reload. If this happens, the IC will report a warning event with the `AddedOrUpdatedWithError` reason for the resource, which caused the problem, and also report the error in the logs.
* If multiple TLS Passthrough TransportServers have the same hostname, only one of them will receive the traffic. If this happens, the IC will report a warning in the logs like `host "app.example.com" is used by more than one TransportServers`.

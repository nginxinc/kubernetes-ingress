---
title: Dos Protected Resource

description: 
weight: 1800
doctypes: [""]
toc: true
---


The DosProtectedResource allows you to specify App Protect Dos configuration as a Kubernetes resource that can then be referenced by your [Ingress](/nginx-ingress-controller/configuration/ingress-resources/basic-configuration) and [VirtualServer and VirtualServerRoute](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/) resources.

The resource is implemented as a [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

> **Feature Status**: DOS is available as a preview feature: it is suitable for experimenting and testing; however, it must be used with caution in production environments. Additionally, while the feature is in preview status, we might introduce some backward-incompatible changes to the resource specification in the next releases. The feature is disabled by default. To enable it, set the [enable-preview-policies](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-enable-preview-policies) command-line argument of the Ingress Controller.

> Note: This feature is only available in NGINX Plus with AppProtectDos.

> Note: The feature is implemented using the NGINX Plus [NGINX App Protect Dos Module](https://docs.nginx.com/nginx-app-protect-dos/configuration/).


## Dos Protected Resource Specification

Below is an example of a dos protected resource. It defines it's own configuration and references to policy configuration and to log configuration:
```yaml
apiVersion: appprotectdos.f5.com/v1beta1
kind: DosProtectedResource
metadata:
  name: dos-protected
spec:
  enable: true
  name: "my-dos"
  apDosMonitor: 
    uri: "webapp.example.com"
  apDosPolicy: "dospolicy"
  dosSecurityLog:
    enable: true
    apDosLogConf: "doslogconf"
    dosLogDest: "syslog-svc.default.svc.cluster.local:514"

```

{{% table %}}
|Field | Description | Type | Required |
| ---| ---| ---| --- |
|``enable`` | Enables NGINX App Protect Dos. | ``bool`` | Yes |
|``name`` | Name of the protected object, max of 63 characters. | ``string`` | No |
|``apDosMonitor.uri`` | The destination to the desired protected object. [App Protect Dos monitor](#dosprotectedresourceapdosmonitor) Default value: None, URL will be extracted from the first request which arrives and taken from "Host" header or from destination ip+port. | ``string`` | No |
|``apDosMonitor.protocol`` | Determines if the server listens on http1 / http2 / grpc. [App Protect Dos monitor](#dosprotectedresourceapdosmonitor) Default value: http1. | ``enum`` | No |
|``apDosMonitor.timeout`` | Determines how long (in seconds) should NGINX App Protect DoS wait for a response. [App Protect Dos monitor](#dosprotectedresourceapdosmonitor) Default value: 10 seconds for http1/http2 and 5 seconds for grpc. | ``int64`` | No |
|``apDosPolicy`` | The [App Protect Dos policy](#dosprotectedresourceapdospolicy) of the dos. Accepts an optional namespace. | ``string`` | No |
|``dosSecurityLog.enable`` | Enables security log. | ``bool`` | No |
|``dosSecurityLog.apDosLogConf`` | The [App Protect Dos log conf](/nginx-ingress-controller/app-protect-dos/configuration/#app-protect-dos-logs) resource. Accepts an optional namespace. | ``string`` | No |
|``dosSecurityLog.dosLogDest`` | The log destination for the security log. Accepted variables are ``syslog:server=<ip-address | localhost | dns-name>:<port>``, ``stderr``, ``<absolute path to file>``. Default is ``"syslog:server=127.0.0.1:514"``. | ``string`` | No |
{{% /table %}}

### DosProtectedResource.apDosPolicy

The `apDosPolicy` is a reference to the policy configuration defined as an `ApDosPolicy`.

### DosProtectedResource.apDosMonitor

This is how NGINX App Protect DoS monitors the stress level of the protected object. The monitor requests are sent from localhost (127.0.0.1).

### Applying Policies

You can apply policies to both VirtualServer and VirtualServerRoute resources. For example:
  * VirtualServer:
    ```yaml
    apiVersion: k8s.nginx.org/v1
    kind: VirtualServer
    metadata:
      name: cafe
      namespace: cafe
    spec:
      host: cafe.example.com
      dos: "default/dos-protected" # virtual server dos configuration
      upstreams:
      - name: coffee
        service: coffee-svc
        port: 80
      routes:
      - path: /tea
        dos: "other/other-dos-protected" # route dos configuration
        route: tea/tea
      - path: /coffee
        action:
          pass: coffee
      ```

      For VirtualServer, you can apply a policy:
      - to all routes (spec dos)
      - to a specific route (route dos)

      Route dos configuration override spec dos configuration.
  
  * VirtualServerRoute, which is referenced by the VirtualServer above:
    ```yaml
    apiVersion: k8s.nginx.org/v1
    kind: VirtualServerRoute
    metadata:
      name: tea
      namespace: tea
    spec:
      host: cafe.example.com
      upstreams:
      - name: tea
        service: tea-svc
        port: 80
      subroutes:
      - path: /tea
        dos: "default/dos-protected"
        action:
          pass: tea
    ```

    For VirtualServerRoute, you can apply dos configuration to a subroute (subroute policies).

### Invalid Dos Protected Resources

NGINX will treat a dos protected resource as invalid if one of the following conditions is met:
* The dos protected resource doesn't pass the [comprehensive validation](#comprehensive-validation).
* The dos protected resource isn't present in the cluster.

### Validation

Two types of validation are available for the Dos Protected resource:
* *Structural validation*, done by `kubectl` and the Kubernetes API server.
* *Comprehensive validation*, done by the Ingress Controller.

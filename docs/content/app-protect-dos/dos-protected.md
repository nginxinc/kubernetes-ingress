---
title: DoS Protected Resource

description: "Dos Protected Resource Specification"
weight: 1800
doctypes: [""]
toc: true
docs: "DOCS-581"
---

> Note: This feature is only available in NGINX Plus with AppProtectDos.

> Note: The feature is implemented using the NGINX Plus [NGINX App Protect Dos Module](/nginx-app-protect-dos/deployment-guide/learn-about-deployment/).


## DoS Protected Resource Specification

Below is an example of a dos protected resource.
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

```

{{% table %}}
|Field | Description | Type | Required |
| ---| ---| ---| --- |
|``enable`` | Enables NGINX App Protect DoS. | ``bool`` | No |
|``name`` | Name of the protected object, max of 63 characters. | ``string`` | No |
|``apDosMonitor.uri`` | The destination to the desired protected object. [App Protect DoS monitor](#dosprotectedresourceapdosmonitor) Default value: None, URL will be extracted from the first request which arrives and taken from "Host" header or from destination ip+port. | ``string`` | No |
|``apDosMonitor.protocol`` | Determines if the server listens on http1 / http2 / grpc / websocket. [App Protect DoS monitor](#dosprotectedresourceapdosmonitor) Default value: http1. | ``enum`` | No |
|``apDosMonitor.timeout`` | Determines how long (in seconds) should NGINX App Protect DoS wait for a response. [App Protect DoS monitor](#dosprotectedresourceapdosmonitor) Default value: 10 seconds for http1/http2 and 5 seconds for grpc. | ``int64`` | No |
|``apDosPolicy`` | The [App Protect DoS policy](#dosprotectedresourceapdospolicy) of the dos. Accepts an optional namespace. | ``string`` | No |
|``dosSecurityLog.enable`` | Enables security log. | ``bool`` | No |
|``dosSecurityLog.apDosLogConf`` | The [App Protect DoS log conf](/nginx-ingress-controller/app-protect-dos/configuration/#app-protect-dos-logs) resource. Accepts an optional namespace. | ``string`` | No |
|``dosSecurityLog.dosLogDest`` | The log destination for the security log. Accepted variables are ``syslog:server=<ip-address | localhost | dns-name>:<port>``, ``stderr``, ``<absolute path to file>``. Default is ``"syslog:server=127.0.0.1:514"``. | ``string`` | No |
{{% /table %}}

### DosProtectedResource.apDosPolicy

The `apDosPolicy` is a reference (qualified identifier in the format `namespace/name`) to the policy configuration defined as an `ApDosPolicy`.

### DosProtectedResource.apDosMonitor

This is how NGINX App Protect DoS monitors the stress level of the protected object. The monitor requests are sent from localhost (127.0.0.1).

### Invalid DoS Protected Resources

NGINX will treat a dos protected resource as invalid if one of the following conditions is met:
* The dos protected resource doesn't pass the [comprehensive validation](#comprehensive-validation).
* The dos protected resource isn't present in the cluster.

### Validation

Two types of validation are available for the dos protected resource:
* *Structural validation*, done by `kubectl` and the Kubernetes API server.
* *Comprehensive validation*, done by the Ingress Controller.

#### Structural Validation

The custom resource definition for the dos protected resource includes a structural OpenAPI schema, which describes the type of every field of the resource.

If you try to create (or update) a resource that violates the structural schema -- for example, the resource uses a string value instead of a bool in the `enable` field -- `kubectl` and the Kubernetes API server will reject the resource.
* Example of `kubectl` validation:
    ```
    $ kubectl apply -f apdos-protected.yaml
    error: error validating "examples/app-protect-dos/apdos-protected.yaml": error validating data: ValidationError(DosProtectedResource.spec.enable): invalid type for com.f5.appprotectdos.v1beta1.DosProtectedResource.spec.enable: got "string", expected "boolean"; if you choose to ignore these errors, turn validation off with --validate=false
    ```
* Example of Kubernetes API server validation:
    ```
    $ kubectl apply -f access-control-policy-allow.yaml --validate=false
    The DosProtectedResource "dos-protected" is invalid: spec.enable: Invalid value: "string": spec.enable in body must be of type boolean: "string"
    ```

If a resource passes structural validation, then the Ingress Controller's comprehensive validation runs.


#### Comprehensive Validation

The Ingress Controller validates the fields of a dos protected resource. If a resource is invalid, the Ingress Controller will reject it. The resource will continue to exist in the cluster, but the Ingress Controller will ignore it.

You can use `kubectl` to check if the Ingress Controller successfully applied a dos protected resource configuration. For our example `dos-protected` dos protected resource, we can run:
```
$ kubectl describe dosprotectedresource dos-protected
. . .
Events:
  Type    Reason          Age                From                      Message
  ----    ------          ----               ----                      -------
  Normal  AddedOrUpdated  12s (x2 over 18h)  nginx-ingress-controller  Configuration for default/dos-protected was added or updated
```
Note how the events section includes a Normal event with the AddedOrUpdated reason that informs us that the configuration was successfully applied.

If you create an invalid resource, the Ingress Controller will reject it and emit a Rejected event. For example, if you create a dos protected resource `dos-protected` with an invalid URI `bad` in the `dosSecurityLog/dosLogDest` field, you will get:
```
$ kubectl describe policy webapp-policy
. . .
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Warning  Rejected  2s    nginx-ingress-controller  error validating DosProtectedResource: dos-protected invalid field: dosSecurityLog/dosLogDest err: invalid log destination: bad, must follow format: <ip-address | localhost | dns name>:<port> or stderr
```
Note how the events section includes a Warning event with the Rejected reason.

**Note**: If you make an existing resource invalid, the Ingress Controller will reject it.

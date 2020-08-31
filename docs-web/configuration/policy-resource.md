# Policy Resource

The Policy resource allows you to configure features like access control and rate-limiting, which you can add to your [VirtualServer and VirtualServerRoute resources](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/)..

The resource is implemented as a [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

This document is the reference documentation for the Policy resource. An example of a Policy for access control is available in our [GitHub repo](https://github.com/nginxinc/kubernetes-ingress/blob/master/examples-of-custom-resources/access-control).

> **Feature Status**: The Policy resource is available as a preview feature: it is suitable for experimenting and testing; however, it must be used with caution in production environments. Additionally, while the feature is in preview, we might introduce some backward-incompatible changes to the resource specification in the next releases.

## Contents

- [Policy Resource](#policy-resource)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Policy Specification](#policy-specification)
    - [AccessControl](#accesscontrol)
      - [AccessControl Merging Behavior](#accesscontrol-merging-behavior)
    - [RateLimit](#ratelimit)
      - [RateLimit Merging Behavior](#ratelimit-merging-behavior)
  - [Using Policy](#using-policy)
    - [Validation](#validation)
      - [Structural Validation](#structural-validation)
      - [Comprehensive Validation](#comprehensive-validation)

## Prerequisites

Policies work together with [VirtualServer and VirtualServerRoute resources](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/), which you need to create separately. 

## Policy Specification

Below is an example of a policy that allows access for clients from the subnet `10.0.0.0/8` and denies access for any other clients:
```yaml
apiVersion: k8s.nginx.org/v1alpha1
kind: Policy 
metadata:
  name: allow-localhost
spec:
  accessControl:
    allow:
    - 10.0.0.0/8
```

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``accessControl``
     - The access control policy based on the client IP address.
     - `accessControl <#accesscontrol>`_
     - No*
   * - ``rateLimit``
     - The rate limit policy controls the rate of processing requests per a defined key.
     - `rateLimit <#ratelimit>`_
     - No*
```

\* A policy must include exactly one policy.

### AccessControl

The access control policy configures NGINX to deny or allow requests from clients with the specified IP addresses/subnets.

For example, the following policy allows access for clients from the subnet `10.0.0.0/8` and denies access for any other clients: 
```yaml
accessControl:
  allow:
  - 10.0.0.0/8
```

In contrast, the policy below does the opposite: denies access for clients from `10.0.0.0/8` and allows access for any other clients:
```yaml
accessControl:
  deny:
  - 10.0.0.0/8
```

> Note: The feature is implemented using the NGINX [ngx_http_access_module](http://nginx.org/en/docs/http/ngx_http_access_module.html). The Ingress Controller access control policy supports either allow or deny rules, but not both (as the module does).

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``allow``
     - Allows access for the specified networks or addresses. For example, ``192.168.1.1`` or ``10.1.1.0/16``.
     - ``[]string``
     - No*
   * - ``deny``
     - Denies access for the specified networks or addresses. For example, ``192.168.1.1`` or ``10.1.1.0/16``.
     - ``[]string``
     - No*
```
\* an accessControl must include either `allow` or `deny`.

#### AccessControl Merging Behavior

A VirtualServer/VirtualServerRoute can reference multiple access control policies. For example, here we reference two policies, each with configured allow lists:
```yaml
policies:
- name: allow-policy-one
- name: allow-policy-two
```
When you reference more than one access control policy, the Ingress Controller will merge the contents into a single allow list or a single deny list.  

Referencing both allow and deny policies, as shown in the example below, is not supported. If both allow and deny lists are referenced, the Ingress Controller uses just the allow list policies. 
```yaml
- name: deny-policy
- name: allow-policy-one
- name: allow-policy-two
```

### RateLimit

The rate limit policy configures NGINX to limit the processing rate of requests.

For example, the following policy will limit all subsequent requests coming from a single IP address once a rate of 10 requests per second is exceeded:
```yaml
rateLimit:
  rate: 10r/s
  zoneSize: 10M
  key: ${binary_remote_addr}
```

> Note: The feature is implemented using the NGINX [ngx_http_limit_req_module](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html).

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``rate``
     - The rate of requests permitted. The rate is specified in requests per second (r/s) or requests per minute (r/m).
     - ``string``
     - Yes
   * - ``key``
     - The key which the rate limit is applied. Can contain text, variables, or a combination of them. Variables must be surrounded by ``${}``. For example: ``${$binary_remote_addr}``. Accepted variables are ``$binary_remote_addr``, ``$request_uri``, ``$url``, ``$http_``, ``$args``, ``$arg_``, ``$cookie_``.
     - ``string``
     - Yes
   * - ``zoneSize``
     - Size of the shared memory zone. Only positive values are allowed. Allowed suffixes are ``k`` or ``m``, if none are present ``k`` is assumed.
     - ``string``
     - Yes
   * - ``delay``
     - The delay parameter specifies a limit at which excessive requests become delayed. If not set all excessive requests are delayed.
     - ``int``
     - No*
   * - ``noDelay``
     - Disables the delaying of excessive requests while requests are being limited. Overrides ``delay`` if both are set.
     - ``bool``
     - No*
   * - ``burst``
     - Excessive requests are delayed until their number exceeds the ``burst`` size in which case the request is terminated with an error.
     - ``int``
     - No*
   * - ``dryRun``
     - Enables the dry run mode. In this mode, requests processing rate is not limited, however, in the shared memory zone, the number of excessive requests is accounted as usual.
     - ``bool``
     - No*
   * - ``logLevel``
     - Sets the desired logging level for cases when the server refuses to process requests due to rate exceeding, or delays request processing. Allowed values are ``info``, ``notice``, ``warn`` or ``error``. Default is ``error``.
     - ``string``
     - No*
   * - ``rejectCode``
     - Sets the status code to return in response to rejected requests. Must fall into the range ``400..600``. Default is ``503``.
     - ``string``
     - No*
```

> For each policy referenced in a VirtualServer and/or its VirtualServerRoutes, the Ingress Controller will generate a singe rate limiting zone defined by the [`limit_req_zone`](http://nginx.org/en/docs/http/ngx_http_limit_req_module.html#limit_req_zone) directive. If two VirtualServer resources reference the same policy, the Ingress Controller will generate two different rate limiting zones, one zone per VirtualServer.

#### RateLimit Merging Behavior
A VirtualServer/VirtualServerRoute can reference multiple rate limit policies. For example, here we reference two policies:
```yaml
policies:
- name: rate-limit-policy-one
- name: rate-limit-policy-two
```

When you reference more than one rate limit policy, the Ingress Controller will configure NGINX to use all referenced rate limits. However, all referenced policy will use the `dryRun`, `logLevel` and `rejectCode` parameters from the first reference (`rate-limit-policy-one` in the example above).

## Using Policy

You can use the usual `kubectl` commands to work with Policy resources, just as with built-in Kubernetes resources.

For example, the following command creates a Policy resource defined in `access-control-policy-allow.yaml` with the name `webapp-policy`:
```
$ kubectl apply -f access-control-policy-allow.yaml
policy.k8s.nginx.org/webapp-policy configured
```

You can get the resource by running:
```
$ kubectl get policy webapp-policy
NAME            AGE
webapp-policy   27m
```

For `kubectl get` and similar commands, you can also use the short name `pol` instead of `policy`.

### Validation

Two types of validation are available for the Policy resource:
* *Structural validation*, done by `kubectl` and the Kubernetes API server.
* *Comprehensive validation*, done by the Ingress Controller.

#### Structural Validation

The custom resource definition for the Policy includes a structural OpenAPI schema, which describes the type of every field of the resource.

If you try to create (or update) a resource that violates the structural schema -- for example, the resource uses a string value instead of an array of strings in the `allow` field -- `kubectl` and the Kubernetes API server will reject the resource.
* Example of `kubectl` validation:
    ```
    $ kubectl apply -f access-control-policy-allow.yaml
    error: error validating "access-control-policy-allow.yaml": error validating data: ValidationError(Policy.spec.accessControl.allow): invalid type for org.nginx.k8s.v1alpha1.Policy.spec.accessControl.allow: got "string", expected "array"; if you choose to ignore these errors, turn validation off with --validate=false
    ```
* Example of Kubernetes API server validation:
    ```
    $ kubectl apply -f access-control-policy-allow.yaml --validate=false
    The Policy "webapp-policy" is invalid: spec.accessControl.allow: Invalid value: "string": spec.accessControl.allow in body must be of type array: "string"
    ```

If a resource passes structural validation, then the Ingress Controller's comprehensive validation runs.

#### Comprehensive Validation

The Ingress Controller validates the fields of a Policy resource. If a resource is invalid, the Ingress Controller will reject it. The resource will continue to exist in the cluster, but the Ingress Controller will ignore it.

You can use `kubectl` to check whether or not the Ingress Controller successfully applied a Policy configuration. For our example `webapp-policy` Policy, we can run:
```
$ kubectl describe pol webapp-policy
. . .
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  11s   nginx-ingress-controller  Policy default/webapp-policy was added or updated
```
Note how the events section includes a Normal event with the AddedOrUpdated reason that informs us that the configuration was successfully applied.

If you create an invalid resource, the Ingress Controller will reject it and emit a Rejected event. For example, if you create a Policy `webapp-policy` with an invalid IP `10.0.0.` in the `allow` field, you will get:
```
$ kubectl describe policy webapp-policy
. . .
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Warning  Rejected  7s    nginx-ingress-controller  Policy default/webapp-policy is invalid and was rejected: spec.accessControl.allow[0]: Invalid value: "10.0.0.": must be a CIDR or IP
```
Note how the events section includes a Warning event with the Rejected reason.

**Note**: If you make an existing resource invalid, the Ingress Controller will reject it.

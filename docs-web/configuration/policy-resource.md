# Policy Resource

The Policy resource allows you to configure features like access control and rate-limiting, which you can add to your [VirtualServer and VirtualServerRoute resources](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/).

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
    - [JWT](#jwt)
      - [JWT Merging Behavior](#jwt-merging-behavior)
    - [IngressMTLS](#ingressmtls)
      - [IngressMTLS Merging Behavior](#ingressmtls-merging-behavior)
    - [EgressMTLS](#egressmtls)
      - [EgressMTLS Merging Behavior](#egressmtls-merging-behavior)
  - [Using Policy](#using-policy)
    - [Applying Policies](#applying-policies)
    - [Invalid Policies](#invalid-policies)
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
   * - ``jwt``
     - The JWT policy configures NGINX Plus to authenticate client requests using JSON Web Tokens.
     - `jwt <#jwt>`_
     - No*
   * - ``ingressMTLS``
     - The IngressMTLS policy configures client certificate verification.
     - `ingressMTLS <#ingressmtls>`_
     - No*
   * - ``egressMTLS``
     - The EgressMTLS policy configures upstreams authentication and certificate verification.
     - `egressMTLS <#egressmtls>`_
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
policies:
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
     - The key to which the rate limit is applied. Can contain text, variables, or a combination of them. Variables must be surrounded by ``${}``. For example: ``${binary_remote_addr}``. Accepted variables are ``$binary_remote_addr``, ``$request_uri``, ``$url``, ``$http_``, ``$args``, ``$arg_``, ``$cookie_``.
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
     - Excessive requests are delayed until their number exceeds the ``burst`` size, in which case the request is terminated with an error.
     - ``int``
     - No*
   * - ``dryRun``
     - Enables the dry run mode. In this mode, the rate limit is not actually applied, but the the number of excessive requests is accounted as usual in the shared memory zone.
     - ``bool``
     - No*
   * - ``logLevel``
     - Sets the desired logging level for cases when the server refuses to process requests due to rate exceeding, or delays request processing. Allowed values are ``info``, ``notice``, ``warn`` or ``error``. Default is ``error``.
     - ``string``
     - No*
   * - ``rejectCode``
     - Sets the status code to return in response to rejected requests. Must fall into the range ``400..599``. Default is ``503``.
     - ``string``
     - No*
```

> For each policy referenced in a VirtualServer and/or its VirtualServerRoutes, the Ingress Controller will generate a single rate limiting zone defined by the [`limit_req_zone`](http://nginx.org/en/docs/http/ngx_http_limit_req_module.html#limit_req_zone) directive. If two VirtualServer resources reference the same policy, the Ingress Controller will generate two different rate limiting zones, one zone per VirtualServer.

#### RateLimit Merging Behavior
A VirtualServer/VirtualServerRoute can reference multiple rate limit policies. For example, here we reference two policies:
```yaml
policies:
- name: rate-limit-policy-one
- name: rate-limit-policy-two
```

When you reference more than one rate limit policy, the Ingress Controller will configure NGINX to use all referenced rate limits. When you define multiple policies, each additional policy inherits the `dryRun`, `logLevel`, and `rejectCode` parameters from the first policy referenced (`rate-limit-policy-one`, in the example above).

### JWT

> Note: This feature is only available in NGINX Plus.

The JWT policy configures NGINX Plus to authenticate client requests using JSON Web Tokens.

For example, the following policy will reject all requests that do not include a valid JWT in the HTTP header `token`:
```yaml
jwt:
  secret: jwk-secret
  realm: "My API"
  token: $http_token
```

You can pass the JWT claims and JOSE headers to the upstream servers. For example:
```yaml
action:
  proxy:
    upstream: webapp
    requestHeaders:
      set:
      - name: user 
        value: ${jwt_claim_user}
      - name: alg
        value: ${jwt_header_alg} 
```
We use the `requestHeaders` of the [Action.Proxy](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#action-proxy) to set the values of two headers that NGINX will pass to the upstream servers. 

The value of the `${jwt_claim_user}` variable is the `user` claim of a JWT. For other claims, use `${jwt_claim_name}`, where `name` is the name of the claim. Note that nested claims and claims that include a period (`.`) are not supported. Similarly, use `${jwt_header_name}` where `name` is the name of a header. In our example, we use the `alg` header. 
 

> Note: The feature is implemented using the NGINX Plus [ngx_http_auth_jwt_module](https://nginx.org/en/docs/http/ngx_http_auth_jwt_module.html).

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``secret``
     - The name of the Kubernetes secret that stores the JWK. It must be in the same namespace as the Policy resource. The JWK must be stored in the secret under the key ``jwk``, otherwise the secret will be rejected as invalid.
     - ``string``
     - Yes
   * - ``realm``
     - The realm of the JWT.
     - ``string``
     - Yes
   * - ``token``
     - The token specifies a variable that contains the JSON Web Token. By default the JWT is passed in the ``Authorization`` header as a Bearer Token. JWT may be also passed as a cookie or a part of a query string, for example: ``$cookie_auth_token``. Accepted variables are ``$http_``, ``$arg_``, ``$cookie_``.
     - ``string``
     - No
```

#### JWT Merging Behavior

A VirtualServer/VirtualServerRoute can reference multiple JWT policies. However, only one can be applied. Every subsequent reference will be ignored. For example, here we reference two policies:
```yaml
policies:
- name: jwt-policy-one
- name: jwt-policy-two
```
In this example the Ingress Controller will use the configuration from the first policy reference `jwt-policy-one`, and ignores `jwt-policy-two`.

### IngressMTLS

The IngressMTLS policy configures client certificate verification.

For example, the following policy will verify a client certificate using the CA certificate specified in the `ingress-mtls-secret`:
```yaml
ingressMTLS:
  clientCertSecret: ingress-mtls-secret
  verifyClient: "on"
  verifyDepth: 1
```

A VirtualServer that references an IngressMTLS policy must:
* Enable [TLS termination](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#virtualserver-tls).
* Reference the policy in the VirtualServer [`spec`](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#virtualserver-specification). It is not allowed to reference an IngressMTLS policy in a [`route `](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#virtualserver-route) or in a VirtualServerRoute [`subroute`](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#virtualserverroute-subroute).

If the conditions above are not met, NGINX will send the `500` status code to clients.

You can pass the client certificate details, including the certificate, to the upstream servers. For example:
```yaml
action:
  proxy:
    upstream: webapp
    requestHeaders:
      set:
      - name: client-cert-subj-dn
        value: ${ssl_client_s_dn} # subject DN
      - name: client-cert
        value: ${ssl_client_escaped_cert} # client certificate in the PEM format (urlencoded) 
```
We use the `requestHeaders` of the [Action.Proxy](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#action-proxy) to set the values of the two headers that NGINX will pass to the upstream servers. See the [list of embedded variables](https://nginx.org/en/docs/http/ngx_http_ssl_module.html#variables) that are supported by the `ngx_http_ssl_module`, which you can use to pass the client certificate details.

> Note: The feature is implemented using the NGINX [ngx_http_ssl_module](https://nginx.org/en/docs/http/ngx_http_ssl_module.html).

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``clientCertSecret``
     - The name of the Kubernetes secret that stores the CA certificate. It must be in the same namespace as the Policy resource. The certificate must be stored in the secret under the key ``ca.crt``, otherwise the secret will be rejected as invalid.
     - ``string``
     - Yes
   * - ``verifyClient``
     - Verification for the client. Possible values are ``"on"``, ``"off"``, ``"optional"``, ``"optional_no_ca"``. The default is ``"on"``.
     - ``string``
     - No
   * - ``verifyDepth``
     - Sets the verification depth in the client certificates chain. The default is ``1``.
     - ``int``
     - No
```

#### IngressMTLS Merging Behavior

A VirtualServer can reference only a single IngressMTLS policy. Every subsequent reference will be ignored. For example, here we reference two policies:
```yaml
policies:
- name: ingress-mtls-policy-one
- name: ingress-mtls-policy-two
```
In this example the Ingress Controller will use the configuration from the first policy reference `ingress-mtls-policy-one`, and ignores `ingress-mtls-policy-two`.

### EgressMTLS

The EgressMTLS policy configures upstreams authentication and certificate verification.

For example, the following policy will use `egress-mtls-secret` to authenticate with the upstream application and `egress-trusted-ca-secret` to verify the certificate of the application:
```yaml
egressMTLS:
  tlsSecret: egress-mtls-secret
  trustedCertSecret: egress-trusted-ca-secret
  verifyServer: on
  verifyDepth: 2
```

> Note: The feature is implemented using the NGINX [ngx_http_proxy_module](https://nginx.org/en/docs/http/ngx_http_proxy_module.html).

```eval_rst
.. list-table::
   :header-rows: 1

   * - Field
     - Description
     - Type
     - Required
   * - ``tlsSecret``
     - The name of the Kubernetes secret that stores the TLS certificate and key. It must be in the same namespace as the Policy resource. The certificate must be stored in the secret under the key ``tls.crt`` and ``tls.key``, otherwise the secret will be rejected as invalid.
     - ``string``
     - No
   * - ``trustedCertSecret``
     - The name of the Kubernetes secret that stores the CA certificate. It must be in the same namespace as the Policy resource. The certificate must be stored in the secret under the key ``ca.crt``, otherwise the secret will be rejected as invalid.
     - ``string``
     - No
   * - ``verifyServer``
     - Verification of the upstream HTTPS server certificate.
     - ``bool`
     - No
   * - ``verifyDepth``
     - Sets the verification depth in the client certificates chain. The default is ``1``.
     - ``int``
     - No
   * - ``sessionReuse``
     - Enables reuse of SSL sessions to the upstreams. The default is ``true``.
     - ``bool`
     - No
   * - ``serverName``
     - Enables passing of the server name through ``Server Name Indication`` extension.
     - ``bool`
     - No
   * - ``sslName``
     - Allows overriding the server name used to verify the certificate of the upstream HTTPS server.
     - ``string`
     - No
   * - ``ciphers``
     - Specifies the enabled ciphers for requests to an upstream HTTPS server. The default is ``DEFAULT``.
     - ``string`
     - No
   * - ``protocols``
     - Specifies the protocols for requests to an upstream HTTPS server. The default is ``TLSv1 TLSv1.1 TLSv1.2``.
     - ``string`
     - No
```

#### EgressMTLS Merging Behavior

A VirtualServer/VirtualServerRoute can reference only a single EgressMTLS policy. Every subsequent reference will be ignored. For example, here we reference two policies:
```yaml
policies:
- name: egress-mtls-policy-one
- name: egress-mtls-policy-two
```
In this example the Ingress Controller will use the configuration from the first policy reference `egress-mtls-policy-one`, and ignores `egress-mtls-policy-two`.

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
      tls:
        secret: cafe-secret
      policies: # spec policies
      - policy1
      upstreams:
      - name: coffee 
        service: coffee-svc
        port: 80
      routes:
      - path: /tea
        policies: # route policies
        - policy2
        route: tea/tea
      - path: /coffee 
        policies: # route policies
        - policy3
        action:
          pass: coffee
      ```

      For VirtualServer, you can apply a policy:
      - to all routes (spec policies)
      - to a specific route (route policies)

      Route policies of the *same type* override spec policies. In the example above, if the type of the policies `policy-1` and `policy-3` is `accessControl`, then for requests to `cafe.example.com/coffee`, NGINX will apply `policy-3`.

      The overriding is enforced by NGINX: the spec policies are implemented in the `server` context of the config, and the route policies are implemented in the `location` context. As a result, the route policies of the same type win.
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
      subroutes: # subroute policies
      - path: /tea 
        policies:
        - policy4
        action:
          pass: tea
    ```
    
    For VirtualServerRoute, you can apply a policy to a subroute (subroute policies).

    Subroute policies of the same type override spec policies. In the example above, if the type of the policies `policy-1` (in the VirtualServer) and `policy-4` is `accessControl`, then for requests to `cafe.example.com/tea`, NGINX will apply `policy-4`. As with the VirtualServer, the overriding is enforced by NGINX.

    Subroute policies always override route policies no matter the types. For example, the policy `policy-2` in the VirtualServer route will be ignored for the subroute `/tea`, because the subroute has its own policies (in our case, only one policy `policy4`). If the subroute didn't have any policies, then the `policy-2` would be applied. This overriding is enforced by the Ingress Controller -- the `location` context for the subroute will either have route policies or subroute policies, but not both.

### Invalid Policies

NGINX will treat a policy as invalid if one of the following conditions is met:
* The policy doesn't pass the [comprehensive validation](#comprehensive-validation).
* The policy isn't present in the cluster.
* The policy doesn't meet its type-specific requirements. For example, an `ingressMTLS` policy requires TLS termination enabled in the VirtualServer. 


For an invalid policy, NGINX returns the 500 status code for client requests with the following rules:
* If a policy is referenced in a VirtualServer `route` or a VirtualServerRoute `subroute`, then NGINX will return the 500 status code for requests for the URIs of that route/subroute.
* If a policy is referenced in the VirtualServer `spec`, then NGINX will return the 500 status code for requests for all URIs of that VirtualServer.

If a policy is invalid, the VirtualServer or VirtualServerRoute will have the [status](/nginx-ingress-controller/configuration/global-configuration/reporting-resources-status#virtualserver-and-virtualserverroute-resources) with the state `Warning` and the message explaining why the policy wasn't considered invalid.

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

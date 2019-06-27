# VirtualServer and VirtualServerRoute Resources

The VirtualServer and VirtualServerRoute resources are new load balancing configuration, introduced in release 1.5 as an alternative to the Ingress resource. The resources enable use cases not supported with the Ingress resource, such as traffic splitting and advanced content-based routing. The resources are implemented as [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

This document is the reference documentation for the resources. To see additional examples of using the resources for specific use cases, go to the [examples-of-custom-resources](../examples-of-custom-resources) folder.

**Feature Status**: The VirtualServer and VirtualServerRoute resources are available as a preview feature: it is suitable for experimenting and testing; however, it must be used with caution in production environments. Additionally, while the feature is in preview, we might introduce some backward-incompatible changes to the resources specification in the next releases.

## Contents
- [VirtualServer and VirtualServerRoute Resources](#VirtualServer-and-VirtualServerRoute-Resources)
  - [Contents](#Contents)
  - [Prerequisites](#Prerequisites)
  - [VirtualServer Specification](#VirtualServer-Specification)
    - [VirtualServer.TLS](#VirtualServerTLS)
    - [VirtualServer.Route](#VirtualServerRoute)
  - [VirtualServerRoute Specification](#VirtualServerRoute-Specification)
    - [VirtualServerRoute.Subroute](#VirtualServerRouteSubroute)
  - [Common Parts of the VirtualServer and VirtualServerRoute](#Common-Parts-of-the-VirtualServer-and-VirtualServerRoute)
    - [Upstream](#Upstream)
    - [Split](#Split)
    - [Rules](#Rules)
    - [Condition](#Condition)
    - [Match](#Match)
  - [Using VirtualServer and VirtualServerRoute](#Using-VirtualServer-and-VirtualServerRoute)
    - [Validation](#Validation)
  - [Customization via ConfigMap](#Customization-via-ConfigMap)

## Prerequisites

The VirtualServer and VirtualServerRoute resources are disabled by default. Make sure to follow Step 1.4 of the [installation](installation.md) doc during the installation process to enable the resources.

## VirtualServer Specification

The VirtualServer resource defines load balancing configuration for a domain name, such as `example.com`. Below is an example of such configuration:
```yaml
apiVersion: k8s.nginx.org/v1alpha1
kind: VirtualServer
metadata:
  name: cafe
spec:
  host: cafe.example.com
  tls:
    secret: cafe-secret
  upstreams:
  - name: tea
    service: tea-svc
    port: 80
  - name: coffee
    service: coffee-svc
    port: 80
  routes:
  - path: /tea
    upstream: tea
  - path: /coffee
    upstream: coffee
```

| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `host` | The host (domain name) of the server. Must be a valid subdomain as defined in RFC 1123, such as `my-app` or `hello.example.com`. Wildcard domains like `*.example.com` are not allowed. | `string` | Yes |
| `tls` | The TLS termination configuration. | [`tls`](#VirtualServerTLS) | No |
| `upstreams` | A list of upstreams. | [`[]upstream`](#Upstream) | No |
| `routes` | A list of routes. | [`[]route`](#VirtualServerRoute) | No |

### VirtualServer.TLS

The tls field defines TLS configuration for a VirtualServer. For example:
```yaml
secret: cafe-secret
```

| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `secret` | The name of a secret with a TLS certificate and key. The secret must belong to the same namespace as the VirtualServer. The secret must contain keys named `tls.crt` and `tls.key` that contain the certificate and private key as described [here](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls). If the secret doesn't exist, NGINX will break any attempt to establish a TLS connection to the host of the VirtualServer. | `string` | Yes |


### VirtualServer.Route

The route defines rules for routing requests to one or multiple upstreams. For example:
```yaml
path: /tea
upstream: tea
```

| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `path` | The path of the route. NGINX will match it against the URI of a request. The path must start with `/` and must not include any whitespace characters, `{`, `}` or `;`. For example, `/`, `/path` are valid. The path must be unique among the paths of all routes of the VirtualServer. | `string` | Yes |
| `upstream` | The name of an upstream. The upstream with that name must be defined in the VirtualServer. | `string` | No* |
| `splits` | The splits configuration for traffic splitting. Must include at least 2 splits. | [`[]split`](#Split) | No* |
| `rules` | The rules configuration for advanced content-based routing. |[`rules`](#Rules) | No* |
| `route` | The name of a VirtualServerRoute resource that defines this route. If the VirtualServerRoute belongs to a different namespace than the VirtualServer, you need to include the namespace. For example, `tea-namespace/tea`. | `string` | No* |

\* -- a route must include exactly one of the following: `upstream`, `splits`, `rules` or `route`.


## VirtualServerRoute Specification

The VirtualServerRoute resource defines a route for a VirtualServer. It can consist of one or multiple subroutes. The VirtualServerRoute is an alternative to [Mergeable Ingress types](../examples/mergeable-ingress-types/README.md).

In the example below, the VirtualServer `cafe` from the namespace `cafe-ns` defines a route with the path `/coffee`, which is further defined in the VirtualServerRoute `coffee` from the namespace `coffee-ns`.

VirtualServer:
```yaml
apiVersion: k8s.nginx.org/v1alpha1
kind: VirtualServer
metadata:
  name: cafe
  namespace: cafe-ns
spec:
  host: cafe.example.com
  upstreams:
  - name: tea
    service: tea-svc
    port: 80
  routes:
  - path: /tea
    upstream: tea
  - path: /coffee
    route: coffee-ns/coffee
```

VirtualServerRoute:
```yaml
apiVersion: k8s.nginx.org/v1alpha1
kind: VirtualServerRoute
metadata:
  name: coffee
  namespace: coffee-ns
spec:
  host: cafe.example.com
  upstreams:
  - name: latte
    service: latte-svc
    port: 80
  - name: espresso
    service: espresso-svc
    port: 80
  subroutes:
  - path: /coffee/latte
    upstream: latte
  - path: /coffee/espresso
    upstream: espresso
```

Note that each subroute must have a `path` that starts with the same prefix (here `/coffee`), which is defined in the route of the VirtualServer. Additionally, the `host` in the VirtualServerRoute must be the same as the `host` of the VirtualServer.

| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `host` | The host (domain name) of the server. Must be a valid subdomain as defined in RFC 1123, such as `my-app` or `hello.example.com`. Wildcard domains like `*.example.com` are not allowed. Must be the same as the `host` of the VirtualServer that references this resource. | `string` | Yes |
| `upstreams` | A list of upstreams. | [`[]upstream`](#Upstream) | No |
| `subroutes` | A list of subroutes. | [`[]subroute`](#VirtualServerRouteSubroute) | No |

### VirtualServerRoute.Subroute

The subroute defines rules for routing requests to one or multiple upstreams. For example:
```yaml
path: /coffee
upstream: coffee
```

| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `path` | The path of the subroute. NGINX will match it against the URI of a request. The path must start with the same path as the path of the route of the VirtualServer that references this resource. It must not include any whitespace characters, `{`, `}` or `;`. The path must be unique among the paths of all subroutes of the VirtualServerRoute. | `string` | Yes |
| `upstream` | The name of an upstream. The upstream with that name must be defined in the VirtualServerRoute. | `string` | No* |
| `splits` | The splits configuration for traffic splitting. Must include at least 2 splits. | [`[]splits`](#Split) | No* |
| `rules` | The rules configuration advanced content-based routing. |[`rules`](#Rules) | No* |

\* -- a subroute must include exactly one of the following: `upstream`, `splits` or `rules`.

## Common Parts of the VirtualServer and VirtualServerRoute

### Upstream

The upstream defines a destination for the routing configuration. For example:
```yaml
name: tea
service: tea-svc
port: 80
lb-method: round_robin
fail-timeout: 10s
max-fails: 1
keepalive: 32
connect-timeout: 30s
read-timeout: 30s
send-timeout: 30s
```

| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `name` | The name of the upstream. Must be a valid DNS label as defined in RFC 1035. For example, `hello` and `upstream-123` are valid. The name must be unique among all upstreams of the resource. | `string` | Yes |
| `service` | The name of a [service](https://kubernetes.io/docs/concepts/services-networking/service/). The service must belong to the same namespace as the resource. If the service doesn't exist, NGINX will assume the service has zero endpoints and return a `502` response for requests for this upstream. | `string` | Yes |
| `port` | The port of the service. If the service doesn't define that port, NGINX will assume the service has zero endpoints and return a `502` response for requests for this upstream. The port must fall into the range `1..65553`. | `uint16` | Yes |
| `lb-method` | The load [balancing method](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-load-balancer/#choosing-a-load-balancing-method). To use the round-robin method, specify `round_robin`. The default is specified in the `lb-method` ConfigMap key. | `string` | No |
| `fail-timeout` | The time during which the specified number of unsuccessful attempts to communicate with an upstream server should happen to consider the server unavailable. See the [fail_timeout](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#fail_timeout) parameter of the server directive. The default is set in the `fail-timeout` ConfigMap key. | `string` | No |
| `max-fails` | The number of unsuccessful attempts to communicate with an upstream server that should happen in the duration set by the `fail-timeout` to consider the server unavailable. See the [max_fails](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#max_fails) parameter of the server directive. The default is set in the `max-fails` ConfgMap key. | `int` | No |
| `keepalive` | Configures the cache for connections to upstream servers. See the [keepalive](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#keepalive) directive. The default is set in the `keepalive` ConfigMap key. | `int` | No
`connect-timeout` | The timeout for establishing a connection with an upstream server. See the [proxy_connect_timeout](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_connect_timeout) directive. The default is specified in the `proxy-connect-timeout` ConfigMap key. | `string` | No
`read-timeout` | The timeout for reading a response from an upstream server. See the [proxy_read_timeout](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_read_timeout) directive.  The default is specified in the `proxy-read-timeout` ConfigMap key. | `string` | No
`send-timeout` | The timeout for transmitting a request to an upstream server. See the [proxy_send_timeout](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_send_timeout) directive. The default is `60s`. | `string` | No

### Split

The split defines a weight for an upstream as part of the splits configuration.

In the example below NGINX routes 80% of requests to the upstream `coffee-v1` and the remaining 20% to `coffee-v2`:
```yaml
splits:
- weight: 80
  upstream: coffee-v1
- weight: 20
  upstream: coffee-v2
```

| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `weight` | The weight of an upstream. Must fall into the range `1..99`. The sum of the weights of all splits must be equal to `100`. | `int` | Yes |
| `upstream` | The name of an upstream. Must be defined in the resource. | `string` | Yes |

### Rules

The rules defines a set of content-based routing rules in a route or subroute.

In the example below, NGINX routes requests with the path `/coffee` to different upstreams based on the value of the cookie `user`:
* `user=john` -> `coffee-future`
* `user=bob` -> `coffee-deprecated`
* If the cookie is not set or not equal to either `john` or `bob`, NGINX routes to `coffee-stable`

```yaml
path: /coffee
rules:
  conditions:
  - cookie: user
  matches:
  - values:
    - john
    upstream: coffee-future
  - values:
    - bob
    upstream: coffee-deprecated
  defaultUpstream: coffee-stable
```

In the next example, NGINX routes requests based on the value of the built-in [`$request_method` variable](http://nginx.org/en/docs/http/ngx_http_core_module.html#var_request_method), which represents the HTTP method of a request:
* all POST requests -> `coffee-post`
* all non-POST requests -> `coffee`

```yaml
path: /coffee
rules:
  conditions:
  - variable: $request_method
  matches:
  - values:
    - POST
    upstream: coffee-post
  defaultUpstream: coffee
```

| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `conditions` | A list of conditions. Must include at least 1 condition. | [`[]condition`](#Condition) | Yes |
| `matches` | A list of matches. Must include at least 1 match. | [`[]match`](#Match) | Yes |
| `defaultUpstream` | The name of the default upstream. NGINX will route requests to the default upstream if it cannot find a successful match in matches. The upstream must be defined in the resource. | `string` | Yes |

### Condition

The condition defines a condition in rules.

| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `header` | The name of a header. Must consist of alphanumeric characters or `-`. | `string` | No* |
| `cookie` | The name of a cookie. Must consist of alphanumeric characters or `_`. | `string` | No* |
| `argument` | The name of an argument. Must consist of alphanumeric characters or `_`. | `string` | No* |
| `variable` | The name of an NGINX variable. Must start with `$`. See the list of the supported variables below the table. | `string` | No* |

\* -- a condition must include exactly one of the following: `header`, `cookie`, `argument` or `variable`.

Supported NGINX variables: `$args`, `$http2`, `$https`, `$remote_addr`, `$remote_port`, `$query_string`, `$request`, `$request_body`, `$request_uri`, `$request_method`, `$scheme`. Find the documentation for each variable [here](https://nginx.org/en/docs/varindex.html).

### Match

The match defines a match that corresponds to conditions.


| Field | Description | Type | Required |
| ----- | ----------- | ---- | -------- |
| `values` | A list of matched values. Must include a value for each condition defined in the rules. How to define a value is shown below the table. | `[]string` | Yes |
| `upstream` | The name of an upstream. Must be defined in the resource. | `string` | Yes |

The value supports two kinds of matching:
* *Case-insensitive string comparison*. For example:
  * `john` -- case-insensitive matching that succeeds for strings, such as `john`, `John`, `JOHN`.
  * `!john` -- negation of the case-incentive matching for john that succeeds for strings, such as `bob`, `anything`, `''` (empty string).
* *Matching with a regular expression*. Note that NGINX supports regular expressions compatible with those used by the Perl programming language (PCRE). For example:
  * `~^yes` -- a case-sensitive regular expression that matches any string that starts with `yes`. For example: `yes`, `yes123`.
  * `!~^yes` -- negation of the previous regular expression that succeeds for strings like `YES`, `Yes123`, `noyes`. (The negation mechanism is not part of the PCRE syntax).
  * `~*no$` -- a case-insensitive regular expression that matches any string that ends with `no`. For example: `no`, `123no`, `123NO`.

**Note**: a value must not include any unescaped double quotes (`"`) and must not end with an unescaped backslash (`\`). For example, the following are invalid values: `some"value`, `somevalue\`.

## Using VirtualServer and VirtualServerRoute

You can use the usual `kubectl` commands to work with VirtualServer and VirtualServerRoute resources, similar to Ingress resources.

For example, the following command creates a VirtualServer resource defined in `cafe-virtual-server.yaml` with the name `cafe`:
```
$ kubectl apply -f cafe-virtual-server.yaml
virtualserver.k8s.nginx.org "cafe" created
```

You can get the resource by running:
```
$ kubectl get virtualserver cafe
NAME      AGE
cafe      3m
```

In the kubectl get and similar commands, you can also use the short name `vs` instead of `virtualserver`.

Working with VirtualServerRoute resources is analogous. In the kubectl commands, use `virtualserverroute` or the short name `vsr`.

### Validation

The Ingress Controller validates VirtualServer and VirtualServerRoute resources. If a resource is invalid, the Ingress Controller will reject it.

You can check if the Ingress Controller successfully applied the configuration for a VirtualServer. For our example `cafe` VirtualServer, we can run:
```
$ kubectl describe vs cafe
. . .
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  16s   nginx-ingress-controller  Configuration for default/cafe was added or updated
```
Note how the events section includes a Normal event with the AddedOrUpdated reason that informs us that the configuration was successfully applied.

If you create an invalid resource, the Ingress Controller will reject it and emit a Rejected event. For example, if you create a VirtualServer `cafe` with an empty `host` field, you will get:
```
$ kubectl describe vs cafe
. . .
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Warning  Rejected  2s    nginx-ingress-controller  VirtualServer default/cafe is invalid and was rejected: spec.host: Required value
```
Note how the events section includes a Warning event with the Rejected reason.

The Ingress Controller validates VirtualServerRoute resources in a similar way.

**Note**: If you make an existing resource invalid, the Ingress Controller will reject it and remove the corresponding configuration from NGINX.

## Customization via ConfigMap

You can customize the NGINX configuration for VirtualServer and VirtualServerRoutes resources using the [ConfigMap](configmap-and-annotations.md). Most of the ConfigMap keys are supported, with the following exceptions:
* `proxy-hide-headers`
* `proxy-pass-headers`
* `hsts`
* `hsts-max-age`
* `hsts-include-subdomains`
* `hsts-behind-proxy`

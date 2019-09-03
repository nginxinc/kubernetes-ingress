# ConfigMap and Annotations

The Ingress resource only allows you to use basic NGINX features -- host and path-based routing and TLS termination. Thus, advanced features like rewriting the request URI or inserting additional response headers are not available.

In addition to using advanced features, often it is necessary to customize or fine tune NGINX behavior. For example, set the number of worker processes or customize the access log format.

Special **ConfigMap** resource and **Annotations** applied to an Ingress resource allow you to:
* Use advanced NGINX features.
* Customize NGINX behavior.

This document describes how to use the ConfigMap resource and Annotations and what features and customization options are available.

## Using ConfigMap

1. Our [installation instructions](installation.md) deploy an empty ConfigMap while the [default installation manifests](../deployments) specify it in the command-line arguments of the Ingress controller. However, if you customized the manifests, to use ConfigMap, make sure to specify the ConfigMap resource to use through the [command-line arguments](cli-arguments.md) of the Ingress controller.

1. Create a ConfigMap file with the name *nginx-config.yaml* and set the values
that make sense for your setup:
    ```yaml
    kind: ConfigMap
    apiVersion: v1
    metadata:
      name: nginx-config
      namespace: nginx-ingress
    data:
      proxy-connect-timeout: "10s"
      proxy-read-timeout: "10s"
      client-max-body-size: "2m"
    ```
    See the section [Summary of ConfigMap and Annotations](#Summary-of-ConfigMap-and-Annotations) for the explanation of the available ConfigMap keys (such as `proxy-connect-timeout` in this example).

1. Create a new (or update the existing) ConfigMap resource:
    ```
    $ kubectl apply -f nginx-config.yaml
    ```
    The NGINX configuration will be updated.

## Using Annotations

Here is an example of using annotations to customize the configuration for a particular Ingress resource:
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: cafe-ingress-with-annotations
  annotations:
    nginx.org/proxy-connect-timeout: "30s"
    nginx.org/proxy-read-timeout: "20s"
    nginx.org/client-max-body-size: "4m"
    nginx.org/server-snippets: |
      location / {
        return 302 /coffee;
      }
spec:
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea
        backend:
          serviceName: tea-svc
          servicePort: 80
      - path: /coffee
        backend:
          serviceName: coffee-svc
          servicePort: 80
```
**Note**: Annotations take precedence over the ConfigMap.

## Summary of ConfigMap and Annotations


**Note**: The annotations that start with `nginx.com` are only supported with NGINX Plus.

### Ingress Controller (Not Related to NGINX Configuration)

| Annotation | ConfigMap Key | Description | Default | Example |
| ---------- | -------------- | ----------- | ------- | ------- |
| `kubernetes.io/ingress.class` | N/A | Specifies which Ingress controller must handle the Ingress resource. Set to `nginx` to make NGINX Ingress controller handle it. | N/A | [Multiple Ingress controllers](../examples/multiple-ingress-controllers). |
| N/A | `external-status-address` | Sets the address to be reported in the status of Ingress resources. Requires the `-report-status` command-line argument. Overrides the `-external-service` argument. | N/A | [Report Ingress Status](report-ingress-status.md). |

### General Customization

| Annotation | ConfigMap Key | Description | Default | Example |
| ---------- | -------------- | ----------- | ------- | ------- |
| `nginx.org/proxy-connect-timeout` | `proxy-connect-timeout` | Sets the value of the [proxy_connect_timeout](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_connect_timeout) and [grpc_connect_timeout](http://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_connect_timeout) directive. | `60s` | |
| `nginx.org/proxy-read-timeout` | `proxy-read-timeout` | Sets the value of the [proxy_read_timeout](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_read_timeout) and [grpc_read_timeout](http://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_read_timeout) directive. | `60s` | |
| `nginx.org/proxy-send-timeout` | `proxy-send-timeout` | Sets the value of the [proxy_send_timeout](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_send_timeout) and [grpc_send_timeout](http://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_send_timeout) directive. | `60s` | |
| `nginx.org/client-max-body-size` | `client-max-body-size` | Sets the value of the [client_max_body_size](http://nginx.org/en/docs/http/ngx_http_core_module.html#client_max_body_size) directive. | `1m` | |
| `nginx.org/proxy-buffering` | `proxy-buffering` | Enables or disables [buffering of responses](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffering) from the proxied server. | `True` | |
| `nginx.org/proxy-buffers` | `proxy-buffers` | Sets the value of the [proxy_buffers](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffers) directive. | Depends on the platform. | |
| `nginx.org/proxy-buffer-size` | `proxy-buffer-size` | Sets the value of the [proxy_buffer_size](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffer_size) and [grpc_buffer_size](http://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_buffer_size) directives. | Depends on the platform. | |
| `nginx.org/proxy-max-temp-file-size` | `proxy-max-temp-file-size` | Sets the value of the  [proxy_max_temp_file_size](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_max_temp_file_size) directive. | `1024m` | |
| N/A | `set-real-ip-from` | Sets the value of the [set_real_ip_from](http://nginx.org/en/docs/http/ngx_http_realip_module.html#set_real_ip_from) directive. | N/A | |
| N/A | `real-ip-header` | Sets the value of the [real_ip_header](http://nginx.org/en/docs/http/ngx_http_realip_module.html#real_ip_header) directive. | `X-Real-IP`| |
| N/A | `real-ip-recursive` | Enables or disables the [real_ip_recursive](http://nginx.org/en/docs/http/ngx_http_realip_module.html#real_ip_recursive) directive. | `False`| |
| `nginx.org/server-tokens` | `server-tokens` | Enables or disables the [server_tokens](http://nginx.org/en/docs/http/ngx_http_core_module.html#server_tokens) directive. Additionally, with the NGINX Plus, you can specify a custom string value, including the empty string value, which disables the emission of the “Server” field. | `True`| |
| N/A | `worker-processes` | Sets the value of the [worker_processes](http://nginx.org/en/docs/ngx_core_module.html#worker_processes) directive. | `auto` | |
| N/A | `worker-rlimit-nofile` | Sets the value of the [worker_rlimit_nofile](http://nginx.org/en/docs/ngx_core_module.html#worker_rlimit_nofile) directive. | N/A | |
| N/A | `worker-connections` | Sets the value of the [worker_connections](http://nginx.org/en/docs/ngx_core_module.html#worker_connections) directive. | `1024` | |
| N/A | `worker-cpu-affinity` | Sets the value of the [worker_cpu_affinity](http://nginx.org/en/docs/ngx_core_module.html#worker_cpu_affinity) directive. | N/A | |
| N/A | `worker-shutdown-timeout` | Sets the value of the [worker_shutdown_timeout](http://nginx.org/en/docs/ngx_core_module.html#worker_shutdown_timeout) directive. | N/A | |
| N/A | `server-names-hash-bucket-size` | Sets the value of the [server_names_hash_bucket_size](http://nginx.org/en/docs/http/ngx_http_core_module.html#server_names_hash_bucket_size) directive. | `256` | |
| N/A | `server-names-hash-max-size` | Sets the value of the [server_names_hash_max_size](http://nginx.org/en/docs/http/ngx_http_core_module.html#server_names_hash_max_size) directive. | `1024` | |
| N/A | `resolver-addresses` | Sets the value of the [resolver](http://nginx.org/en/docs/http/ngx_http_core_module.html#resolver) addresses. Note: If you use a DNS name (ex., `kube-dns.kube-system.svc.cluster.local`) as a resolver address, NGINX Plus will resolve it using the system resolver during the start and on every configuration reload. As a consequence, If the name cannot be resolved or the DNS server doesn't respond, NGINX Plus will fail to start or reload. To avoid this, consider using only IP addresses as resolver addresses. Supported in NGINX Plus only. | N/A | [Support for Type ExternalName Services](../examples/externalname-services). |
| N/A | `resolver-ipv6` | Enables IPv6 resolution in the resolver. Supported in NGINX Plus only. | `True` | [Support for Type ExternalName Services](../examples/externalname-services). |
| N/A | `resolver-valid` | Sets the time NGINX caches the resolved DNS records. Supported in NGINX Plus only. | TTL value of a DNS record | [Support for Type ExternalName Services](../examples/externalname-services). |
| N/A | `resolver-timeout` | Sets the [resolver_timeout](http://nginx.org/en/docs/http/ngx_http_core_module.html#resolver_timeout) for name resolution. Supported in NGINX Plus only.  | `30s` | [Support for Type ExternalName Services](../examples/externalname-services). |
| N/A | `keepalive-timeout` | Sets the value of the [keepalive_timeout](http://nginx.org/en/docs/http/ngx_http_core_module.html#keepalive_timeout) directive. | `65s` | |
| N/A | `keepalive-requests` | Sets the value of the [keepalive_requests](http://nginx.org/en/docs/http/ngx_http_core_module.html#keepalive_requests) directive. | `100` | |
| N/A | `variables-hash-bucket-size` | Sets the value of the [variables_hash_bucket_size](http://nginx.org/en/docs/http/ngx_http_core_module.html#variables_hash_bucket_size) directive. | `256` | |
| N/A | `variables-hash-max-size` | Sets the value of the [variables-hash-max-size](http://nginx.org/en/docs/http/ngx_http_core_module.html#variables_hash_max_size) directive. | `1024` | |

### Logging

| Annotation | ConfigMap Key | Description | Default | Example |
| ---------- | -------------- | ----------- | ------- | ------- |
| N/A | `error-log-level` | Sets the global [error log level](http://nginx.org/en/docs/ngx_core_module.html#error_log) for NGINX.  | `notice` | |
| N/A | `access-log-off` | Disables the [access log](http://nginx.org/en/docs/http/ngx_http_log_module.html#access_log). | `False` | |
| N/A | `log-format` | Sets the custom [log format](http://nginx.org/en/docs/http/ngx_http_log_module.html#log_format).  | See the [template file](../internal/configs/version1/nginx.tmpl) for the access log. | |
| N/A | `stream-log-format` | Sets the custom [log format](http://nginx.org/en/docs/stream/ngx_stream_log_module.html#log_format) for TCP/UDP load balancing.  | See the [template file](../internal/configs/version1/nginx.tmpl). | |

### Request URI/Header Manipulation

| Annotation | ConfigMap Key | Description | Default | Example |
| ---------- | -------------- | ----------- | ------- | ------- |
| `nginx.org/proxy-hide-headers` | `proxy-hide-headers` | Sets the value of one or more  [proxy_hide_header](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_hide_header) directives. Example: `"nginx.org/proxy-hide-headers": "header-a,header-b"` | N/A | |
| `nginx.org/proxy-pass-headers` | `proxy-pass-headers` | Sets the value of one or more   [proxy_pass_header](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass_header) directives. Example: `"nginx.org/proxy-pass-headers": "header-a,header-b"` | N/A | |
| `nginx.org/rewrites` | N/A | Configures URI rewriting. | N/A | [Rewrites Support](../examples/rewrites). |

### Auth and SSL/TLS

| Annotation | ConfigMap Key | Description | Default | Example |
| ---------- | -------------- | ----------- | ------- | ------- |
| `nginx.org/redirect-to-https` | `redirect-to-https` | Sets the 301 redirect rule based on the value of the `http_x_forwarded_proto` header on the server block to force incoming traffic to be over HTTPS. Useful when terminating SSL in a load balancer in front of the Ingress controller — see [115](https://github.com/nginxinc/kubernetes-ingress/issues/115) | `False` | |
| `ingress.kubernetes.io/ssl-redirect` | `ssl-redirect` | Sets an unconditional 301 redirect rule for all incoming HTTP traffic to force incoming traffic over HTTPS. | `True` | |
| `nginx.org/hsts` | `hsts` | Enables [HTTP Strict Transport Security (HSTS)](https://www.nginx.com/blog/http-strict-transport-security-hsts-and-nginx/): the HSTS header is added to the responses from backends. The `preload` directive is included in the header. | `False` | |
| `nginx.org/hsts-max-age` | `hsts-max-age` | Sets the value of the `max-age` directive of the HSTS header. | `2592000` (1 month) |
| `nginx.org/hsts-include-subdomains` | `hsts-include-subdomains` | Adds the `includeSubDomains` directive to the HSTS header. | `False`| |
| `nginx.org/hsts-behind-proxy` | `hsts-behind-proxy` | Enables HSTS based on the value of the `http_x_forwarded_proto` request header. Should only be used when TLS termination is configured in a load balancer (proxy) in front of the Ingress Controller. Note: to control redirection from HTTP to HTTPS configure the `nginx.org/redirect-to-https` annotation. | `False` | |
| N/A | `ssl-protocols` | Sets the value of the [ssl_protocols](http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_protocols) directive. | `TLSv1 TLSv1.1 TLSv1.2`| |
| N/A | `ssl-prefer-server-ciphers` | Enables or disables the [ssl_prefer_server_ciphers](http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_prefer_server_ciphers) directive. | `False`| |
| N/A | `ssl-ciphers` | Sets the value of the [ssl_ciphers](http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_ciphers) directive. | `HIGH:!aNULL:!MD5`| |
| N/A | `ssl-dhparam-file` | Sets the content of the dhparam file. The controller will create the file and set the value of the [ssl_dhparam](http://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_dhparam) directive with the path of the file.  | N/A | |
| `nginx.com/jwt-key` | N/A |  Specifies a Secret resource with keys for validating JSON Web Tokens (JWTs). | N/A | [Support for JSON Web Tokens (JWTs)](../examples/jwt). |
| `nginx.com/jwt-realm` | N/A | Specifies a realm. | N/A | [Support for JSON Web Tokens (JWTs)](../examples/jwt). |
| `nginx.com/jwt-token` | N/A | Specifies a variable that contains JSON Web Token. | By default, a JWT is expected in the `Authorization` header as a Bearer Token. | [Support for JSON Web Tokens (JWTs)](../examples/jwt). |
| `nginx.com/jwt-login-url` | N/A | Specifies a URL to which a client is redirected in case of an invalid or missing JWT. | N/A | [Support for JSON Web Tokens (JWTs)](../examples/jwt). |

### Listeners

| Annotation | ConfigMap Key | Description | Default | Example |
| ---------- | -------------- | ----------- | ------- | ------- |
| N/A | `http2` | Enables HTTP/2 in servers with SSL enabled. | `False` |
| `nginx.org/listen-ports` | N/A | Configures HTTP ports that NGINX will listen on. | `[80]` | |
| `nginx.org/listen-ports-ssl` | N/A | Configures HTTPS ports that NGINX will listen on. | `[443]` | |
| N/A | `proxy-protocol` | Enables PROXY Protocol for incoming connections. | `False` | [Proxy Protocol](../examples/proxy-protocol). |

### Backend Services (Upstreams)

| Annotation | ConfigMap Key | Description | Default | Example |
| ---------- | -------------- | ----------- | ------- | ------- |
| `nginx.org/lb-method` | `lb-method` | Sets the [load balancing method](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-load-balancer/#choosing-a-load-balancing-method). To use the round-robin method, specify `"round_robin"`. | `"random two least_conn"` | |
| `nginx.org/ssl-services` | N/A | Enables HTTPS or gRPC over SSL when connecting to the endpoints of services. | N/A | [SSL Services Support](../examples/ssl-services). |
| `nginx.org/grpc-services` | N/A | Enables gRPC for services. Note: requires HTTP/2 (see `http2` ConfigMap key); only works for Ingresses with TLS termination enabled. | N/A | [GRPC Services Support](../examples/grpc-services).|
| `nginx.org/websocket-services` | N/A | Enables WebSocket for services. | N/A | [WebSocket support](../examples/websocket). |
| `nginx.org/max-fails` | `max-fails` | Sets the value of the [max_fails](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#max_fails) parameter of the `server` directive. | `1` | |
| `nginx.org/max-conns` | N\A | Sets the value of the [max_conns](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#max_conns) parameter of the `server` directive. | `0` | |
| `nginx.org/upstream-zone-size` | `upstream-zone-size` | Sets the size of the shared memory [zone](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#zone) for upstreams. For NGINX, the special value 0 disables the shared memory zones. For NGINX Plus, shared memory zones are required and cannot be disabled. The special value 0 will be ignored. | `256K` | |
| `nginx.org/fail-timeout` | `fail-timeout` | Sets the value of the [fail_timeout](https://nginx.org/en/docs/http/ngx_http_upstream_module.html#fail_timeout) parameter of the `server` directive. | `10s` | |
| `nginx.com/sticky-cookie-services` | N/A | Configures session persistence. | N/A | [Session Persistence](../examples/session-persistence). |
| `nginx.org/keepalive` | `keepalive` | Sets the value of the [keepalive](http://nginx.org/en/docs/http/ngx_http_upstream_module.html#keepalive) directive. Note that `proxy_set_header Connection "";` is added to the generated configuration when the value > 0. | `0` | |
| `nginx.com/health-checks` | N/A | Enables active health checks. | `False` | [Support for Active Health Checks](../examples/health-checks). |
| `nginx.com/health-checks-mandatory` | N/A | Configures active health checks as mandatory. | `False` | [Support for Active Health Checks](../examples/health-checks). |
| `nginx.com/health-checks-mandatory-queue` | N/A | When active health checks are mandatory, configures a queue for temporary storing incoming requests during the time when NGINX Plus is checking the health of the endpoints after a configuration reload. | `0` | [Support for Active Health Checks](../examples/health-checks). |
| `nginx.com/slow-start` | N/A | Sets the upstream server [slow-start period](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-load-balancer/#server-slow-start). By default, slow-start is activated after a server becomes [available](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-health-check/#passive-health-checks) or [healthy](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-health-check/#active-health-checks). To enable slow-start for newly added servers, configure [mandatory active health checks](../examples/health-checks). | `"0s"` | |


### Snippets and Custom Templates

| Annotation | ConfigMap Key | Description | Default | Example |
| ---------- | -------------- | ----------- | ------- | ------- |
| N/A | `main-snippets` | Sets a custom snippet in main context. | N/A | |
| N/A | `http-snippets` | Sets a custom snippet in http context. | N/A | |
| `nginx.org/location-snippets` | `location-snippets` | Sets a custom snippet in location context. | N/A | |
| `nginx.org/server-snippets` | `server-snippets` | Sets a custom snippet in server context. | N/A | |
| N/A | `stream-snippets` | Sets a custom snippet in stream context. | N/A | [Support for  TCP/UDP Load Balancing](../examples/tcp-udp). |
| N/A | `main-template` | Sets the main NGINX configuration template. | By default the template is read from the file in the container. | [Custom Templates](../examples/custom-templates). |
| N/A | `ingress-template` | Sets the NGINX configuration template for an Ingress resource. | By default the template is read from the file on the container. | [Custom Templates](../examples/custom-templates). |


### Modules
| Annotation | ConfigMap Key | Description | Default | Example |
| ---------- | -------------- | ----------- | ------- | ------- |
| N/A | `opentracing` | Enables [OpenTracing](https://opentracing.io) globally (for all Ingress, VirtualServer and VirtualServerRoute resources). Note: requires the Ingress Controller image with OpenTracing module and a tracer. See the [docs](./opentracing.md) for more information. | `False` | [Support for OpenTracing](../examples/opentracing/README.md). |
| N/A | `opentracing-tracer` | Sets the path to the vendor tracer binary plugin.| N/A | [Support for OpenTracing](../examples/opentracing/README.md). |
| N/A | `opentracing-tracer-config` | Sets the tracer configuration in JSON format.| N/A | [Support for OpenTracing](../examples/opentracing/README.md). |
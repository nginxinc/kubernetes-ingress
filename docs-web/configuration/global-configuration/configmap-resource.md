# ConfigMap Resource

The ConfigMap resources allows you to customize or fine tune NGINX behavior. For example, set the number of worker processes or customize the access log format.

## Using ConfigMap

1. Our [installation instructions](/nginx-ingress-controller/installation/installation-with-manifests) deploy an empty ConfigMap while the default installation manifests specify it in the command-line arguments of the Ingress controller. However, if you customized the manifests, to use ConfigMap, make sure to specify the ConfigMap resource to use through the [command-line arguments](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments) of the Ingress controller.

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
    See the section [Summary of ConfigMap Keys](#summary-of-configmap-keys) for the explanation of the available ConfigMap keys (such as `proxy-connect-timeout` in this example).

1. Create a new (or update the existing) ConfigMap resource:
    ```
    $ kubectl apply -f nginx-config.yaml
    ```
    The NGINX configuration will be updated.

## ConfigMap and Ingress Annotations

Annotations allow you to configure advanced NGINX features and customize or fine tune NGINX behavior.

The ConfigMap applies globally, meaning that it affects every Ingress resource. In contrast, annotations always apply to their Ingress resource. Annotations allow overriding some ConfigMap keys. For example, the `nginx.org/proxy-connect-timeout` annotations overrides the `proxy-connect-timeout` ConfigMap key.

See the doc about [annotations](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations).

## ConfigMap and VirtualServer/VirtualServerRoute Resource

The ConfigMap affects every VirtualServer and VirtualServerRoute resources. However, the fields of those resources allow overriding some ConfigMap keys. For example, the `connect-timeout` field of the `upstream` overrides the `proxy-connect-timeout` ConfigMap key.

See the doc about [VirtualServer and VirtualServerRoute resources](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources).

## Summary of ConfigMap Keys

### Ingress Controller (Not Related to NGINX Configuration)

```eval_rst
.. list-table::
   :header-rows: 1

   * - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``external-status-address``
     - Sets the address to be reported in the status of Ingress resources. Requires the ``-report-status`` command-line argument. Overrides the ``-external-service`` argument.
     - N/A
     - `Report Ingress Status </nginx-ingress-controller/configuration/global-configuration/reporting-resources-status>`_.
```

### General Customization

```eval_rst
.. list-table::
   :header-rows: 1

   * - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``proxy-connect-timeout``
     - Sets the value of the `proxy_connect_timeout <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_connect_timeout>`_ and `grpc_connect_timeout <https://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_connect_timeout>`_ directive.
     - ``60s``
     - 
   * - ``proxy-read-timeout``
     - Sets the value of the `proxy_read_timeout <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_read_timeout>`_ and `grpc_read_timeout <https://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_read_timeout>`_ directive.
     - ``60s``
     - 
   * - ``proxy-send-timeout``
     - Sets the value of the `proxy_send_timeout <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_send_timeout>`_ and `grpc_send_timeout <https://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_send_timeout>`_ directive.
     - ``60s``
     - 
   * - ``client-max-body-size``
     - Sets the value of the `client_max_body_size <https://nginx.org/en/docs/http/ngx_http_core_module.html#client_max_body_size>`_ directive.
     - ``1m``
     - 
   * - ``proxy-buffering``
     - Enables or disables `buffering of responses <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffering>`_ from the proxied server.
     - ``True``
     - 
   * - ``proxy-buffers``
     - Sets the value of the `proxy_buffers <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffers>`_ directive.
     - Depends on the platform.
     - 
   * - ``proxy-buffer-size``
     - Sets the value of the `proxy_buffer_size <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffer_size>`_ and `grpc_buffer_size <https://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_buffer_size>`_ directives.
     - Depends on the platform.
     - 
   * - ``proxy-max-temp-file-size``
     - Sets the value of the  `proxy_max_temp_file_size <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_max_temp_file_size>`_ directive.
     - ``1024m``
     - 
   * - ``set-real-ip-from``
     - Sets the value of the `set_real_ip_from <https://nginx.org/en/docs/http/ngx_http_realip_module.html#set_real_ip_from>`_ directive.
     - N/A
     - 
   * - ``real-ip-header``
     - Sets the value of the `real_ip_header <https://nginx.org/en/docs/http/ngx_http_realip_module.html#real_ip_header>`_ directive.
     - ``X-Real-IP``
     - 
   * - ``real-ip-recursive``
     - Enables or disables the `real_ip_recursive <https://nginx.org/en/docs/http/ngx_http_realip_module.html#real_ip_recursive>`_ directive.
     - ``False``
     - 
   * - ``server-tokens``
     - Enables or disables the `server_tokens <https://nginx.org/en/docs/http/ngx_http_core_module.html#server_tokens>`_ directive. Additionally, with the NGINX Plus, you can specify a custom string value, including the empty string value, which disables the emission of the “Server” field.
     - ``True``
     - 
   * - ``worker-processes``
     - Sets the value of the `worker_processes <https://nginx.org/en/docs/ngx_core_module.html#worker_processes>`_ directive.
     - ``auto``
     - 
   * - ``worker-rlimit-nofile``
     - Sets the value of the `worker_rlimit_nofile <https://nginx.org/en/docs/ngx_core_module.html#worker_rlimit_nofile>`_ directive.
     - N/A
     - 
   * - ``worker-connections``
     - Sets the value of the `worker_connections <https://nginx.org/en/docs/ngx_core_module.html#worker_connections>`_ directive.
     - ``1024``
     - 
   * - ``worker-cpu-affinity``
     - Sets the value of the `worker_cpu_affinity <https://nginx.org/en/docs/ngx_core_module.html#worker_cpu_affinity>`_ directive.
     - N/A
     - 
   * - ``worker-shutdown-timeout``
     - Sets the value of the `worker_shutdown_timeout <https://nginx.org/en/docs/ngx_core_module.html#worker_shutdown_timeout>`_ directive.
     - N/A
     - 
   * - ``server-names-hash-bucket-size``
     - Sets the value of the `server_names_hash_bucket_size <https://nginx.org/en/docs/http/ngx_http_core_module.html#server_names_hash_bucket_size>`_ directive.
     - ``256``
     - 
   * - ``server-names-hash-max-size``
     - Sets the value of the `server_names_hash_max_size <https://nginx.org/en/docs/http/ngx_http_core_module.html#server_names_hash_max_size>`_ directive.
     - ``1024``
     - 
   * - ``resolver-addresses``
     - Sets the value of the `resolver <https://nginx.org/en/docs/http/ngx_http_core_module.html#resolver>`_ addresses. Note: If you use a DNS name (ex., ``kube-dns.kube-system.svc.cluster.local``\ ) as a resolver address, NGINX Plus will resolve it using the system resolver during the start and on every configuration reload. As a consequence, If the name cannot be resolved or the DNS server doesn't respond, NGINX Plus will fail to start or reload. To avoid this, consider using only IP addresses as resolver addresses. Supported in NGINX Plus only.
     - N/A
     - `Support for Type ExternalName Services <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/externalname-services>`_.
   * - ``resolver-ipv6``
     - Enables IPv6 resolution in the resolver. Supported in NGINX Plus only.
     - ``True``
     - `Support for Type ExternalName Services <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/externalname-services>`_.
   * - ``resolver-valid``
     - Sets the time NGINX caches the resolved DNS records. Supported in NGINX Plus only.
     - TTL value of a DNS record
     - `Support for Type ExternalName Services <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/externalname-services>`_.
   * - ``resolver-timeout``
     - Sets the `resolver_timeout <https://nginx.org/en/docs/http/ngx_http_core_module.html#resolver_timeout>`_ for name resolution. Supported in NGINX Plus only.
     - ``30s``
     - `Support for Type ExternalName Services <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/externalname-services>`_.
   * - ``keepalive-timeout``
     - Sets the value of the `keepalive_timeout <https://nginx.org/en/docs/http/ngx_http_core_module.html#keepalive_timeout>`_ directive.
     - ``65s``
     - 
   * - ``keepalive-requests``
     - Sets the value of the `keepalive_requests <https://nginx.org/en/docs/http/ngx_http_core_module.html#keepalive_requests>`_ directive.
     - ``100``
     - 
   * - ``variables-hash-bucket-size``
     - Sets the value of the `variables_hash_bucket_size <https://nginx.org/en/docs/http/ngx_http_core_module.html#variables_hash_bucket_size>`_ directive.
     - ``256``
     - 
   * - ``variables-hash-max-size``
     - Sets the value of the `variables-hash-max-size <https://nginx.org/en/docs/http/ngx_http_core_module.html#variables_hash_max_size>`_ directive.
     - ``1024``
     - 
```

### Logging

```eval_rst
.. list-table::
   :header-rows: 1

   * - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``error-log-level``
     - Sets the global `error log level <https://nginx.org/en/docs/ngx_core_module.html#error_log>`_ for NGINX.
     - ``notice``
     - 
   * - ``access-log-off``
     - Disables the `access log <https://nginx.org/en/docs/http/ngx_http_log_module.html#access_log>`_.
     - ``False``
     - 
   * - ``default-server-access-log-off``
     - Disables the `access log <https://nginx.org/en/docs/http/ngx_http_log_module.html#access_log>`_ for the default server. If access log is disabled globally (``access-log-off: "True"``), then the default server access log is always disabled.
     - ``False``
     - 
   * - ``log-format``
     - Sets the custom `log format <https://nginx.org/en/docs/http/ngx_http_log_module.html#log_format>`_ for HTTP and HTTPS traffic. For convenience, it is possible to define the log format across multiple lines (each line separated by ``\n``). In that case, the Ingress Controller will replace every ``\n`` character with a space character. All ``'`` characters must be escaped.
     - See the `template file <https://github.com/nginxinc/kubernetes-ingress/blob/master/internal/configs/version1/nginx.tmpl>`_ for the access log.
     - 
   * - ``log-format-escaping``
     - Sets the characters escaping for the variables of the log format. Supported values: ``json`` (JSON escaping), ``default`` (the default escaping) ``none`` (disables escaping).
     - ``default``
     -
   * - ``stream-log-format``
     - Sets the custom `log format <https://nginx.org/en/docs/stream/ngx_stream_log_module.html#log_format>`_ for TCP, UDP, and TLS Passthrough traffic. For convenience, it is possible to define the log format across multiple lines (each line separated by ``\n``). In that case, the Ingress Controller will replace every ``\n`` character with a space character. All ``'`` characters must be escaped.
     - See the `template file <https://github.com/nginxinc/kubernetes-ingress/blob/master/internal/configs/version1/nginx.tmpl>`_.
     - 
   * - ``stream-log-format-escaping``
     - Sets the characters escaping for the variables of the stream log format. Supported values: ``json`` (JSON escaping), ``default`` (the default escaping) ``none`` (disables escaping).
     - ``default``
     - 
```

### Request URI/Header Manipulation

```eval_rst
.. list-table::
   :header-rows: 1

   * - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``proxy-hide-headers``
     - Sets the value of one or more  `proxy_hide_header <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_hide_header>`_ directives. Example: ``"nginx.org/proxy-hide-headers": "header-a,header-b"``
     - N/A
     - 
   * - ``proxy-pass-headers``
     - Sets the value of one or more   `proxy_pass_header <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass_header>`_ directives. Example: ``"nginx.org/proxy-pass-headers": "header-a,header-b"``
     - N/A
     - 
```

### Auth and SSL/TLS

```eval_rst
.. list-table::
   :header-rows: 1

   * - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``redirect-to-https``
     - Sets the 301 redirect rule based on the value of the ``http_x_forwarded_proto`` header on the server block to force incoming traffic to be over HTTPS. Useful when terminating SSL in a load balancer in front of the Ingress controller — see `115 <https://github.com/nginxinc/kubernetes-ingress/issues/115>`_
     - ``False``
     - 
   * - ``ssl-redirect``
     - Sets an unconditional 301 redirect rule for all incoming HTTP traffic to force incoming traffic over HTTPS.
     - ``True``
     - 
   * - ``hsts``
     - Enables `HTTP Strict Transport Security (HSTS) <https://www.nginx.com/blog/http-strict-transport-security-hsts-and-nginx/>`_\ : the HSTS header is added to the responses from backends. The ``preload`` directive is included in the header.
     - ``False``
     - 
   * - ``hsts-max-age``
     - Sets the value of the ``max-age`` directive of the HSTS header.
     - ``2592000`` (1 month)
     - 
   * - ``hsts-include-subdomains``
     - Adds the ``includeSubDomains`` directive to the HSTS header.
     - ``False``
     - 
   * - ``hsts-behind-proxy``
     - Enables HSTS based on the value of the ``http_x_forwarded_proto`` request header. Should only be used when TLS termination is configured in a load balancer (proxy) in front of the Ingress Controller. Note: to control redirection from HTTP to HTTPS configure the ``nginx.org/redirect-to-https`` annotation.
     - ``False``
     - 
   * - ``ssl-protocols``
     - Sets the value of the `ssl_protocols <https://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_protocols>`_ directive.
     - ``TLSv1 TLSv1.1 TLSv1.2``
     - 
   * - ``ssl-prefer-server-ciphers``
     - Enables or disables the `ssl_prefer_server_ciphers <https://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_prefer_server_ciphers>`_ directive.
     - ``False``
     - 
   * - ``ssl-ciphers``
     - Sets the value of the `ssl_ciphers <https://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_ciphers>`_ directive.
     - ``HIGH:!aNULL:!MD5``
     - 
   * - ``ssl-dhparam-file``
     - Sets the content of the dhparam file. The controller will create the file and set the value of the `ssl_dhparam <https://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_dhparam>`_ directive with the path of the file.
     - N/A
     - 
```

### Listeners

```eval_rst
.. list-table::
   :header-rows: 1

   * - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``http2``
     - Enables HTTP/2 in servers with SSL enabled.
     - ``False``
     - 
   * - ``proxy-protocol``
     - Enables PROXY Protocol for incoming connections.
     - ``False``
     - `Proxy Protocol <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/proxy-protocol>`_.
```

### Backend Services (Upstreams)

```eval_rst
.. list-table::
   :header-rows: 1

   * - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``lb-method``
     - Sets the `load balancing method <https://docs.nginx.com/nginx/admin-guide/load-balancer/http-load-balancer/#choosing-a-load-balancing-method>`_. To use the round-robin method, specify ``"round_robin"``.
     - ``"random two least_conn"``
     - 
   * - ``max-fails``
     - Sets the value of the `max_fails <https://nginx.org/en/docs/http/ngx_http_upstream_module.html#max_fails>`_ parameter of the ``server`` directive.
     - ``1``
     - 
   * - ``upstream-zone-size``
     - Sets the size of the shared memory `zone <https://nginx.org/en/docs/http/ngx_http_upstream_module.html#zone>`_ for upstreams. For NGINX, the special value 0 disables the shared memory zones. For NGINX Plus, shared memory zones are required and cannot be disabled. The special value 0 will be ignored.
     - ``256K``
     - 
   * - ``fail-timeout``
     - Sets the value of the `fail_timeout <https://nginx.org/en/docs/http/ngx_http_upstream_module.html#fail_timeout>`_ parameter of the ``server`` directive.
     - ``10s``
     - 
   * - ``keepalive``
     - Sets the value of the `keepalive <https://nginx.org/en/docs/http/ngx_http_upstream_module.html#keepalive>`_ directive. Note that ``proxy_set_header Connection "";`` is added to the generated configuration when the value > 0.
     - ``0``
     - 
```

### Snippets and Custom Templates

```eval_rst
.. list-table::
   :header-rows: 1

   * - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``main-snippets``
     - Sets a custom snippet in main context.
     - N/A
     - 
   * - ``http-snippets``
     - Sets a custom snippet in http context.
     - N/A
     - 
   * - ``location-snippets``
     - Sets a custom snippet in location context.
     - N/A
     - 
   * - ``server-snippets``
     - Sets a custom snippet in server context.
     - N/A
     - 
   * - ``stream-snippets``
     - Sets a custom snippet in stream context.
     - N/A
     - `Support for  TCP/UDP Load Balancing <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/tcp-udp>`_.
   * - ``main-template``
     - Sets the main NGINX configuration template.
     - By default the template is read from the file in the container.
     - `Custom Templates </nginx-ingress-controller/configuration/global-configuration/custom-templates>`_.
   * - ``ingress-template``
     - Sets the NGINX configuration template for an Ingress resource.
     - By default the template is read from the file on the container.
     - `Custom Templates </nginx-ingress-controller/configuration/global-configuration/custom-templates>`_.
   * - ``virtualserver-template``
     - Sets the NGINX configuration template for an VirtualServer resource.
     - By default the template is read from the file on the container.
     - `Custom Templates </nginx-ingress-controller/configuration/global-configuration/custom-templates>`_.
```

### Modules

```eval_rst
.. list-table::
   :header-rows: 1

   * - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``opentracing``
     - Enables `OpenTracing <https://opentracing.io>`_ globally (for all Ingress, VirtualServer and VirtualServerRoute resources). Note: requires the Ingress Controller image with OpenTracing module and a tracer. See the `docs </nginx-ingress-controller/third-party-modules/opentracing>`_ for more information.
     - ``False``
     - `Support for OpenTracing <https://github.com/nginxinc/kubernetes-ingress/blob/master/examples/opentracing/README.md>`_.
   * - ``opentracing-tracer``
     - Sets the path to the vendor tracer binary plugin.
     - N/A
     - `Support for OpenTracing <https://github.com/nginxinc/kubernetes-ingress/blob/master/examples/opentracing/README.md>`_.
   * - ``opentracing-tracer-config``
     - Sets the tracer configuration in JSON format.
     - N/A
     - `Support for OpenTracing <https://github.com/nginxinc/kubernetes-ingress/blob/master/examples/opentracing/README.md>`_.
   * - ``app-protect-cookie-seed``
     - Sets the ``app_protect_cookie_seed`` `global directive </nginx-app-protect/configuration/#global-directives>`_.
     - Random automatically generated string
     - 
   * - ``app-protect-failure-mode-action``
     - Sets the ``app_protect_failure_mode_action`` `global directive </nginx-app-protect/configuration/#global-directives>`_.
     - ``pass``
     - 
   * - ``app-protect-cpu-thresholds``
     - Sets the ``app_protect_cpu_thresholds`` `global directive </nginx-app-protect/configuration/#global-directives>`_.
     - ``high=100 low=100``
     - 
   * - ``app-protect-physical-memory-util-thresholds``
     - Sets the ``app_protect_physical_memory_util_thresholds`` `global directive </nginx-app-protect/configuration/#global-directives>`_.
     - ``high=100 low=100``
     - 
```

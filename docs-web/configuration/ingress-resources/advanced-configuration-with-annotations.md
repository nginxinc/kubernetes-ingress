# Advanced Configuration with Annotations

The Ingress resource only allows you to use basic NGINX features -- host and path-based routing and TLS termination. Thus, advanced features like rewriting the request URI or inserting additional response headers are not available. 

In addition to using advanced features, often it is necessary to customize or fine tune NGINX behavior. For example, set the value of connection timeouts.

Annotations applied to an Ingress resource allow you to use advanced NGINX features and customize/fine tune NGINX behavior for that Ingress resource. 

Customization and fine-tuning is also available through the [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource). Annotations take precedence over the ConfigMap.

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

## Summary of Annotations

The table below summarizes the available annotations. 

**Note**: The annotations that start with `nginx.com` are only supported with NGINX Plus.

### Ingress Controller (Not Related to NGINX Configuration)

```eval_rst
.. list-table::
   :header-rows: 1

   * - Annotation
     - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``kubernetes.io/ingress.class``
     - N/A
     - Specifies which Ingress controller must handle the Ingress resource. Set to ``nginx`` to make NGINX Ingress controller handle it.
     - N/A
     - `Multiple Ingress controllers </nginx-ingress-controller/installation/running-multiple-ingress-controllers>`_.
```

### General Customization

```eval_rst
.. list-table::
   :header-rows: 1

   * - Annotation
     - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``nginx.org/proxy-connect-timeout``
     - ``proxy-connect-timeout``
     - Sets the value of the `proxy_connect_timeout <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_connect_timeout>`_ and `grpc_connect_timeout <https://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_connect_timeout>`_ directive.
     - ``60s``
     - 
   * - ``nginx.org/proxy-read-timeout``
     - ``proxy-read-timeout``
     - Sets the value of the `proxy_read_timeout <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_read_timeout>`_ and `grpc_read_timeout <https://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_read_timeout>`_ directive.
     - ``60s``
     - 
   * - ``nginx.org/proxy-send-timeout``
     - ``proxy-send-timeout``
     - Sets the value of the `proxy_send_timeout <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_send_timeout>`_ and `grpc_send_timeout <https://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_send_timeout>`_ directive.
     - ``60s``
     - 
   * - ``nginx.org/client-max-body-size``
     - ``client-max-body-size``
     - Sets the value of the `client_max_body_size <https://nginx.org/en/docs/http/ngx_http_core_module.html#client_max_body_size>`_ directive.
     - ``1m``
     - 
   * - ``nginx.org/proxy-buffering``
     - ``proxy-buffering``
     - Enables or disables `buffering of responses <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffering>`_ from the proxied server.
     - ``True``
     - 
   * - ``nginx.org/proxy-buffers``
     - ``proxy-buffers``
     - Sets the value of the `proxy_buffers <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffers>`_ directive.
     - Depends on the platform.
     - 
   * - ``nginx.org/proxy-buffer-size``
     - ``proxy-buffer-size``
     - Sets the value of the `proxy_buffer_size <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_buffer_size>`_ and `grpc_buffer_size <https://nginx.org/en/docs/http/ngx_http_grpc_module.html#grpc_buffer_size>`_ directives.
     - Depends on the platform.
     - 
   * - ``nginx.org/proxy-max-temp-file-size``
     - ``proxy-max-temp-file-size``
     - Sets the value of the  `proxy_max_temp_file_size <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_max_temp_file_size>`_ directive.
     - ``1024m``
     - 
   * - ``nginx.org/server-tokens``
     - ``server-tokens``
     - Enables or disables the `server_tokens <https://nginx.org/en/docs/http/ngx_http_core_module.html#server_tokens>`_ directive. Additionally, with the NGINX Plus, you can specify a custom string value, including the empty string value, which disables the emission of the “Server” field.
     - ``True``
     - 
```

### Request URI/Header Manipulation

```eval_rst
.. list-table::
   :header-rows: 1

   * - Annotation
     - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``nginx.org/proxy-hide-headers``
     - ``proxy-hide-headers``
     - Sets the value of one or more  `proxy_hide_header <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_hide_header>`_ directives. Example: ``"nginx.org/proxy-hide-headers": "header-a,header-b"``
     - N/A
     - 
   * - ``nginx.org/proxy-pass-headers``
     - ``proxy-pass-headers``
     - Sets the value of one or more   `proxy_pass_header <https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_pass_header>`_ directives. Example: ``"nginx.org/proxy-pass-headers": "header-a,header-b"``
     - N/A
     - 
   * - ``nginx.org/rewrites``
     - N/A
     - Configures URI rewriting.
     - N/A
     - `Rewrites Support <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/rewrites>`_.
```

### Auth and SSL/TLS

```eval_rst
.. list-table::
   :header-rows: 1

   * - Annotation
     - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``nginx.org/redirect-to-https``
     - ``redirect-to-https``
     - Sets the 301 redirect rule based on the value of the ``http_x_forwarded_proto`` header on the server block to force incoming traffic to be over HTTPS. Useful when terminating SSL in a load balancer in front of the Ingress controller — see `115 <https://github.com/nginxinc/kubernetes-ingress/issues/115>`_
     - ``False``
     - 
   * - ``ingress.kubernetes.io/ssl-redirect``
     - ``ssl-redirect``
     - Sets an unconditional 301 redirect rule for all incoming HTTP traffic to force incoming traffic over HTTPS.
     - ``True``
     - 
   * - ``nginx.org/hsts``
     - ``hsts``
     - Enables `HTTP Strict Transport Security (HSTS) <https://www.nginx.com/blog/http-strict-transport-security-hsts-and-nginx/>`_\ : the HSTS header is added to the responses from backends. The ``preload`` directive is included in the header.
     - ``False``
     - 
   * - ``nginx.org/hsts-max-age``
     - ``hsts-max-age``
     - Sets the value of the ``max-age`` directive of the HSTS header.
     - ``2592000`` (1 month)
     - 
   * - ``nginx.org/hsts-include-subdomains``
     - ``hsts-include-subdomains``
     - Adds the ``includeSubDomains`` directive to the HSTS header.
     - ``False``
     - 
   * - ``nginx.org/hsts-behind-proxy``
     - ``hsts-behind-proxy``
     - Enables HSTS based on the value of the ``http_x_forwarded_proto`` request header. Should only be used when TLS termination is configured in a load balancer (proxy) in front of the Ingress Controller. Note: to control redirection from HTTP to HTTPS configure the ``nginx.org/redirect-to-https`` annotation.
     - ``False``
     - 
   * - ``nginx.com/jwt-key``
     - N/A
     - Specifies a Secret resource with keys for validating JSON Web Tokens (JWTs).
     - N/A
     - `Support for JSON Web Tokens (JWTs) <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/jwt>`_.
   * - ``nginx.com/jwt-realm``
     - N/A
     - Specifies a realm.
     - N/A
     - `Support for JSON Web Tokens (JWTs) <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/jwt>`_.
   * - ``nginx.com/jwt-token``
     - N/A
     - Specifies a variable that contains JSON Web Token.
     - By default, a JWT is expected in the ``Authorization`` header as a Bearer Token.
     - `Support for JSON Web Tokens (JWTs) <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/jwt>`_.
   * - ``nginx.com/jwt-login-url``
     - N/A
     - Specifies a URL to which a client is redirected in case of an invalid or missing JWT.
     - N/A
     - `Support for JSON Web Tokens (JWTs) <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/jwt>`_.
```

### Listeners

```eval_rst
.. list-table::
   :header-rows: 1

   * - Annotation
     - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``nginx.org/listen-ports``
     - N/A
     - Configures HTTP ports that NGINX will listen on.
     - ``[80]``
     - 
   * - ``nginx.org/listen-ports-ssl``
     - N/A
     - Configures HTTPS ports that NGINX will listen on.
     - ``[443]``
     - 
```

### Backend Services (Upstreams)

```eval_rst
.. list-table::
   :header-rows: 1

   * - Annotation
     - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``nginx.org/lb-method``
     - ``lb-method``
     - Sets the `load balancing method <https://docs.nginx.com/nginx/admin-guide/load-balancer/http-load-balancer/#choosing-a-load-balancing-method>`_. To use the round-robin method, specify ``"round_robin"``.
     - ``"random two least_conn"``
     - 
   * - ``nginx.org/ssl-services``
     - N/A
     - Enables HTTPS or gRPC over SSL when connecting to the endpoints of services.
     - N/A
     - `SSL Services Support <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/ssl-services>`_.
   * - ``nginx.org/grpc-services``
     - N/A
     - Enables gRPC for services. Note: requires HTTP/2 (see ``http2`` ConfigMap key); only works for Ingresses with TLS termination enabled.
     - N/A
     - `GRPC Services Support <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/grpc-services>`_.
   * - ``nginx.org/websocket-services``
     - N/A
     - Enables WebSocket for services.
     - N/A
     - `WebSocket support <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/websocket>`_.
   * - ``nginx.org/max-fails``
     - ``max-fails``
     - Sets the value of the `max_fails <https://nginx.org/en/docs/http/ngx_http_upstream_module.html#max_fails>`_ parameter of the ``server`` directive.
     - ``1``
     - 
   * - ``nginx.org/max-conns``
     - N\A
     - Sets the value of the `max_conns <https://nginx.org/en/docs/http/ngx_http_upstream_module.html#max_conns>`_ parameter of the ``server`` directive.
     - ``0``
     - 
   * - ``nginx.org/upstream-zone-size``
     - ``upstream-zone-size``
     - Sets the size of the shared memory `zone <https://nginx.org/en/docs/http/ngx_http_upstream_module.html#zone>`_ for upstreams. For NGINX, the special value 0 disables the shared memory zones. For NGINX Plus, shared memory zones are required and cannot be disabled. The special value 0 will be ignored.
     - ``256K``
     - 
   * - ``nginx.org/fail-timeout``
     - ``fail-timeout``
     - Sets the value of the `fail_timeout <https://nginx.org/en/docs/http/ngx_http_upstream_module.html#fail_timeout>`_ parameter of the ``server`` directive.
     - ``10s``
     - 
   * - ``nginx.com/sticky-cookie-services``
     - N/A
     - Configures session persistence.
     - N/A
     - `Session Persistence <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/session-persistence>`_.
   * - ``nginx.org/keepalive``
     - ``keepalive``
     - Sets the value of the `keepalive <https://nginx.org/en/docs/http/ngx_http_upstream_module.html#keepalive>`_ directive. Note that ``proxy_set_header Connection "";`` is added to the generated configuration when the value > 0.
     - ``0``
     - 
   * - ``nginx.com/health-checks``
     - N/A
     - Enables active health checks.
     - ``False``
     - `Support for Active Health Checks <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/health-checks>`_.
   * - ``nginx.com/health-checks-mandatory``
     - N/A
     - Configures active health checks as mandatory.
     - ``False``
     - `Support for Active Health Checks <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/health-checks>`_.
   * - ``nginx.com/health-checks-mandatory-queue``
     - N/A
     - When active health checks are mandatory, configures a queue for temporary storing incoming requests during the time when NGINX Plus is checking the health of the endpoints after a configuration reload.
     - ``0``
     - `Support for Active Health Checks <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/health-checks>`_.
   * - ``nginx.com/slow-start``
     - N/A
     - Sets the upstream server `slow-start period <https://docs.nginx.com/nginx/admin-guide/load-balancer/http-load-balancer/#server-slow-start>`_. By default, slow-start is activated after a server becomes `available <https://docs.nginx.com/nginx/admin-guide/load-balancer/http-health-check/#passive-health-checks>`_ or `healthy <https://docs.nginx.com/nginx/admin-guide/load-balancer/http-health-check/#active-health-checks>`_. To enable slow-start for newly added servers, configure `mandatory active health checks <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/health-checks>`_.
     - ``"0s"``
     - 
```

### Snippets and Custom Templates

```eval_rst
.. list-table::
   :header-rows: 1

   * - Annotation
     - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``nginx.org/location-snippets``
     - ``location-snippets``
     - Sets a custom snippet in location context.
     - N/A
     - 
   * - ``nginx.org/server-snippets``
     - ``server-snippets``
     - Sets a custom snippet in server context.
     - N/A
     - 
```
    
### App Protect

**Note**: The App Protect annotations only work if App Protect module is [installed](/nginx-ingress-controller/app-protect/installation/).

```eval_rst
.. list-table::
   :header-rows: 1

   * - Annotation
     - ConfigMap Key
     - Description
     - Default
     - Example
   * - ``appprotect.f5.com/app-protect-policy``
     - N/A
     - The name of the App Protect Policy for the Ingress Resource. Format is ``namespace/name``. If no namespace is specified, the same namespace of the Ingress Resource is used. If not specified but ``appprotect.f5.com/app-protect-enable`` is true, a default policy id applied. If the referenced policy resource does not exist, or policy is invalid, this annotation will be ignored, and the default policy will be applied. 
     - N/A
     - `Example for App Protect <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/appprotect>`_.
   * - ``appprotect.f5.com/app-protect-enable``
     - N/A
     - Enable App Protect for the Ingress Resource.
     - ``False``
     - `Example for App Protect <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/appprotect>`_.
   * - ``appprotect.f5.com/app-protect-security-log-enable``
     - N/A
     - Enable the `security log </nginx-app-protect/troubleshooting/#app-protect-security-log>`_ for App Protect.
     - ``False``
     - `Example for App Protect <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/appprotect>`_.
   * - ``appprotect.f5.com/app-protect-security-log``
     - N/A
     - The App Protect log configuration for the Ingress Resource. Format is ``namespace/name``. If no namespace is specified, the same namespace as the Ingress Resource is used. If not specified the  default is used which is:  filter: ``illegal``, format: ``default``
     - N/A
     - `Example for App Protect <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/appprotect>`_.
   * - ``appprotect.f5.com/app-protect-security-log-destination``
     - N/A
     - The destination of the security log. For more information check the `DESTINATION argument </nginx-app-protect/troubleshooting/#app-protect-security-log>`_. 
     - ``syslog:server=localhost:514``
     - `Example for App Protect <https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/appprotect>`_.
```

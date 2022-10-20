---
title: NGINX Ingress Controller with NGINX Plus
description: "This document explains the key characteristics that NGINX Plus brings on top of NGINX into the NGINX Ingress Controller."
weight: 400
doctypes: ["concept"]
toc: true
docs: "DOCS-611"
aliases:
  - /nginx-plus/
---


NGINX Ingress Controller works with both [NGINX](https://nginx.org/) and [NGINX Plus](https://www.nginx.com/products/nginx/) -- a commercial closed source version of NGINX that comes with additional features and support.

Below are the key characteristics that NGINX Plus brings on top of NGINX into the NGINX Ingress Controller.

## Additional features

* *Real-time metrics* A number metrics about how NGINX Plus and applications are performing are available through the API or a [built-in dashboard](https://docs.nginx.com/nginx-ingress-controller/logging-and-monitoring/status-page/). Optionally, the metrics can be exported to [Prometheus](https://docs.nginx.com/nginx-ingress-controller/logging-and-monitoring/prometheus/).
* *Additional load balancing methods*. The following additional methods are available: `least_time` and `random two least_time` and their derivatives. See the [documentation](https://nginx.org/en/docs/http/ngx_http_upstream_module.html) for the complete list of load balancing methods.
* *Session persistence* The *sticky cookie* method is available. See the [Session Persistence for VirtualServer Resources example](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/custom-resources/session-persistence) and the [Session Persistence for Ingress Resources example](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/ingress-resources/session-persistence).
* *Active health checks*. See the [Support for Active Health Checks for VirtualServer Resources example](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/custom-resources/health-checks) and the [Support for Active Health Checks for Ingress Resources example](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/ingress-resources/health-checks).
* *JWT validation*. See the [Support for JSON Web Tokens for VirtualServer Resources example (JWTs)](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/custom-resources/jwt) and the [Support for JSON Web Tokens for Ingress Resources example (JWTs)](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/ingress-resources/jwt).

See the [VirtualServer](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources.md), [Policy](https://docs.nginx.com/nginx-ingress-controller/configuration/policy-resource.md) and [TransportServer](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources.md) docs  for a comprehensive guide of the NGINX Plus features available by using our custom resources

For the complete list of available NGINX Plus features available for Ingress resources, see the [ConfigMap](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/configmap-resource/) and [Annotations](https://docs.nginx.com/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations/) docs. Note that such features are configured through annotations that start with `nginx.com`, for example, `nginx.com/health-checks`.

## Dynamic reconfiguration

Every time the number of pods of services you expose via an Ingress resource changes, the Ingress Controller updates the configuration of the load balancer to reflect those changes. For NGINX, the configuration file must be changed and the configuration subsequently reloaded. For NGINX Plus, the dynamic reconfiguration is utilized, which allows NGINX Plus to be updated on-the-fly without reloading the configuration. This prevents increase of memory usage during reloads, especially with a high volume of client requests, as well as increased memory usage when load balancing applications with long-lived connections (WebSocket, applications with file uploading/downloading or streaming).

## Commercial support

Support from NGINX Inc is available for NGINX Plus Ingress Controller.

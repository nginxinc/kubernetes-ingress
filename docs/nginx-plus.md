# NGINX Ingress Controller with NGINX Plus

NGINX Ingress controller works with both [NGINX](https://nginx.org/) and [NGINX Plus](https://www.nginx.com/products/nginx/) -- a commercial closed source version of NGINX that comes with additional features and support. 

Below are the key characteristics that NGINX Plus brings on top of NGINX into the NGINX Ingress controller:
* **Additional features**
    * *Real-time metrics* A number metrics about how NGINX Plus and applications are performing are available through the API or a [built-in dashboard](installation.md#5-access-the-live-activity-monitoring-dashboard--stub_status-page). Optionally, the metrics can be exported to [Prometheus](installation.md#support-for-prometheus-monitoring).
    * *Additional load balancing methods*. The following additional methods are available: `least_time` and `random two least_time` and their derivatives. See the [documentation](https://nginx.org/en/docs/http/ngx_http_upstream_module.html) for the complete list of load balancing methods.
    * *Session persistence* The *sticky cookie* method is available. See the [Session Persistence](../examples/session-persistence) example.
    * *Active health checks*. See the [Support for Active Health Checks](../examples/health-checks) example.
    * *JWT validation*. See the [Support for JSON Web Tokens (JWTs)](../examples/jwt) example.
    
    See [ConfigMap and Annotations](configmap-and-annotations.md) doc for the complete list of available NGINX Plus features. Note that such features are configured through annotations that start with `nginx.com`, for example, `nginx.com/health-checks`.
* **Dynamic reconfiguration** Every time the number of pods of services you expose via an Ingress resource changes, the Ingress controller updates the configuration of the load balancer to reflect those changes. For NGINX, the configuration file must be changed and the configuration subsequently reloaded. For NGINX Plus, the dynamic reconfiguration is utilized, which allows NGINX Plus to be updated on-the-fly without reloading the configuration. This prevents increase of memory usage during reloads, especially with a high volume of client requests, as well as increased memory usage when load balancing applications with long-lived connections (WebSocket, applications with file uploading/downloading or streaming).
* **Commercial support** Support from NGINX Inc is available for NGINX Plus Ingress controller.

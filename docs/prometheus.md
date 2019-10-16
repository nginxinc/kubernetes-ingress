# Monitoring the Ingress Controller Using Prometheus

The Ingress Controller exposes a number of metrics in the [Prometheus](https://prometheus.io/) format. Those include NGINX/NGINX Plus and the Ingress Controller metrics.

## Enabling Metrics
To enable Prometheus metrics, follow the [Support For Prometheus Monitoring](./installation.md#support-for-prometheus-monitoring) section of the installation doc. Once enabled, the metrics will be available via the configured endpoint.

## Available Metrics
The Ingress Controller exports the following metrics:

* NGINX/NGINX Plus metrics. Please see this [doc](https://github.com/nginxinc/nginx-prometheus-exporter#exported-metrics) to find more information about the exported metrics.

* Ingress Controller metrics
  * `controller_nginx_reloads_total`. Number of successful NGINX reloads.
  * `controller_nginx_reload_errors_total`. Number of unsuccessful NGINX reloads.
  * `controller_nginx_last_reload_status`. Status of the last NGINX reload, 0 meaning down and 1 up.
  * `controller_nginx_last_reload_milliseconds`. Duration in milliseconds of the last NGINX reload.
  * `controller_ingress_resources_total`. Number of handled Ingress resources. This metric includes the label type, that groups the Ingress resources by their type (regular, [minion or master](./../examples/mergeable-ingress-types)). **Note**: The metric doesn't count minions without a master.
  * `controller_virtualserver_resources_total`. Number of handled VirtualServer resources.
  * `controller_virtualserverroute_resources_total`. Number of handled VirtualServerRoute resources. **Note**: The metric counts only VirtualServerRoutes that have a reference from a VirtualServer.

**Note**: all metrics have the namespace nginx_ingress. For example, nginx_ingress_controller_nginx_reloads_total.

# Prometheus

The Ingress Controller exposes a number of metrics in the [Prometheus](https://prometheus.io/) format. Those include NGINX/NGINX Plus and the Ingress Controller metrics.

## Enabling Metrics

If you're using *Kubernetes manifests* (Deployment or DaemonSet) to install the Ingress Controller, to enable Prometheus metrics:
1. Run the Ingress controller with the `-enable-prometheus-metrics` [command-line argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments). As a result, the Ingress Controller will expose NGINX or NGINX Plus metrics in the Prometheus format via the path `/metrics` on port `9113` (customizable via the `-prometheus-metrics-listen-port` command-line argument).
1. Add the Prometheus port to the list of the ports of the Ingress Controller container in the template of the Ingress Controller pod:
    ```yaml
    - name: prometheus
      containerPort: 9113
    ```
1. Make Prometheus aware of the Ingress Controller targets by adding the following annotations to the template of the Ingress Controller pod (note: this assumes your Prometheus is configured to discover targets by analyzing the annotations of pods):
    ```yaml
    annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9113"
    ```

If you're using *Helm* to install the Ingress Controller, to enable Prometheus metrics, configure the `prometheus.*` parameters of the Helm chart. See the [Installation with Helm](/nginx-ingress-controller/installation/installation-with-helm) doc.

## Available Metrics
The Ingress Controller exports the following metrics:

* NGINX/NGINX Plus metrics. Please see this [doc](https://github.com/nginxinc/nginx-prometheus-exporter#exported-metrics) to find more information about the exported metrics.

* Ingress Controller metrics
  * `controller_nginx_reloads_total`. Number of successful NGINX reloads.
  * `controller_nginx_reload_errors_total`. Number of unsuccessful NGINX reloads.
  * `controller_nginx_last_reload_status`. Status of the last NGINX reload, 0 meaning down and 1 up.
  * `controller_nginx_last_reload_milliseconds`. Duration in milliseconds of the last NGINX reload.
  * `controller_nginx_worker_processes_total`. Number of NGINX worker processes. This metric includes the constant label generation with 2 possible values `old` or `current`. 
  * `controller_ingress_resources_total`. Number of handled Ingress resources. This metric includes the label type, that groups the Ingress resources by their type (regular, [minion or master](/nginx-ingress-controller/configuration/ingress-resources/cross-namespace-configuration)). **Note**: The metric doesn't count minions without a master.
  * `controller_virtualserver_resources_total`. Number of handled VirtualServer resources.
  * `controller_virtualserverroute_resources_total`. Number of handled VirtualServerRoute resources. **Note**: The metric counts only VirtualServerRoutes that have a reference from a VirtualServer.

**Note**: all metrics have the namespace nginx_ingress. For example, nginx_ingress_controller_nginx_reloads_total.

**Note**: all metrics include the label `class`, which is set to the class of the Ingress Controller. The class is configured via the `-ingress-class` command-line argument.

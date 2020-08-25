import pytest
import requests

from kubernetes.client import V1ContainerPort

from suite.resources_utils import wait_until_all_pods_are_ready, ensure_connection


@pytest.fixture(scope="class")
def enable_exporter_port(cli_arguments, kube_apis,
                         ingress_controller_prerequisites, ingress_controller) -> None:
    """
    Set containerPort for Prometheus Exporter.

    :param cli_arguments: context
    :param kube_apis: client apis
    :param ingress_controller_prerequisites
    :param ingress_controller: IC name
    :return:
    """
    namespace = ingress_controller_prerequisites.namespace
    port = V1ContainerPort(9113, None, None, "prometheus", 'TCP')
    print("------------------------- Enable 9113 port in IC -----------------------------------")
    body = kube_apis.apps_v1_api.read_namespaced_deployment(ingress_controller, namespace)
    body.spec.template.spec.containers[0].ports.append(port)

    if cli_arguments['deployment-type'] == 'deployment':
        kube_apis.apps_v1_api.patch_namespaced_deployment(ingress_controller, namespace, body)
    else:
        kube_apis.apps_v1_api.patch_namespaced_daemon_set(ingress_controller, namespace, body)
    wait_until_all_pods_are_ready(kube_apis.v1, namespace)


@pytest.mark.ingresses
@pytest.mark.smoke
@pytest.mark.parametrize('ingress_controller, expected_metrics',
                         [
                             pytest.param({"extra_args": ["-enable-prometheus-metrics"]},
                                          ['nginx_ingress_controller_nginx_reload_errors_total{class="nginx"} 0',
                                           'nginx_ingress_controller_ingress_resources_total{class="nginx",type="master"} 0',
                                           'nginx_ingress_controller_ingress_resources_total{class="nginx",type="minion"} 0',
                                           'nginx_ingress_controller_ingress_resources_total{class="nginx",type="regular"} 0',
                                           'nginx_ingress_controller_nginx_last_reload_milliseconds',
                                           'nginx_ingress_controller_nginx_last_reload_status{class="nginx"} 1',
                                           'nginx_ingress_controller_nginx_reload_errors_total{class="nginx"} 0',
                                           'nginx_ingress_controller_nginx_reloads_total{class="nginx",reason="endpoints"}',
                                           'nginx_ingress_controller_nginx_reloads_total{class="nginx",reason="other"}'])
                          ],
                         indirect=["ingress_controller"])
class TestPrometheusExporter:
    def test_metrics(self, ingress_controller_endpoint, ingress_controller, enable_exporter_port, expected_metrics):
        req_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
        ensure_connection(req_url, 200)
        resp = requests.get(req_url)
        assert resp.status_code == 200, f"Expected 200 code for /metrics but got {resp.status_code}"
        resp_content = resp.content.decode('utf-8')
        for item in expected_metrics:
            assert item in resp_content

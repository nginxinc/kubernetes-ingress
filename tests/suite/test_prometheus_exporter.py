import pytest
import requests

from kubernetes.client import V1ContainerPort

from suite.resources_utils import (
    ensure_connection_to_public_endpoint,
    create_items_from_yaml,
    create_example_app,
    delete_common_app,
    delete_items_from_yaml,
    wait_until_all_pods_are_ready,
    ensure_response_from_backend,
    wait_before_test,
    wait_until_all_pods_are_ready,
    ensure_connection,
    delete_secret,
    create_secret_from_yaml,
)
from suite.yaml_utils import get_first_ingress_host_from_yaml
from settings import TEST_DATA


class IngressSetup:
    """
    Encapsulate the Smoke Example details.

    Attributes:
        public_endpoint (PublicEndpoint):
        ingress_host (str):
    """

    def __init__(self, req_url, ingress_host):
        self.req_url = req_url
        self.ingress_host = ingress_host


@pytest.fixture(scope="class")
def enable_exporter_port(
    cli_arguments, kube_apis, ingress_controller_prerequisites, ingress_controller
) -> None:
    """
    Set containerPort for Prometheus Exporter.

    :param cli_arguments: context
    :param kube_apis: client apis
    :param ingress_controller_prerequisites
    :param ingress_controller: IC name
    :return:
    """
    namespace = ingress_controller_prerequisites.namespace
    port = V1ContainerPort(9113, None, None, "prometheus", "TCP")
    print("------------------------- Enable 9113 port in IC -----------------------------------")
    body = kube_apis.apps_v1_api.read_namespaced_deployment(ingress_controller, namespace)
    body.spec.template.spec.containers[0].ports.append(port)

    if cli_arguments["deployment-type"] == "deployment":
        kube_apis.apps_v1_api.patch_namespaced_deployment(ingress_controller, namespace, body)
    else:
        kube_apis.apps_v1_api.patch_namespaced_daemon_set(ingress_controller, namespace, body)
    wait_until_all_pods_are_ready(kube_apis.v1, namespace)


@pytest.fixture(scope="class")
def ingress_setup(request, kube_apis, ingress_controller_endpoint, test_namespace) -> IngressSetup:
    print("------------------------- Deploy Ingress Example -----------------------------------")
    secret_name = create_secret_from_yaml(
        kube_apis.v1, test_namespace, f"{TEST_DATA}/smoke/smoke-secret.yaml"
    )
    create_items_from_yaml(
        kube_apis, f"{TEST_DATA}/smoke/standard/smoke-ingress.yaml", test_namespace
    )
    ingress_host = get_first_ingress_host_from_yaml(
        f"{TEST_DATA}/smoke/standard/smoke-ingress.yaml"
    )
    create_example_app(kube_apis, "simple", test_namespace)
    wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)
    ensure_connection_to_public_endpoint(
        ingress_controller_endpoint.public_ip,
        ingress_controller_endpoint.port,
        ingress_controller_endpoint.port_ssl,
    )
    req_url = f"https://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.port_ssl}/backend1"

    def fin():
        print("Clean up simple app")
        delete_common_app(kube_apis, "simple", test_namespace)
        delete_items_from_yaml(
            kube_apis, f"{TEST_DATA}/smoke/standard/smoke-ingress.yaml", test_namespace
        )
        delete_secret(kube_apis.v1, secret_name, test_namespace)

    request.addfinalizer(fin)

    return IngressSetup(req_url, ingress_host)


@pytest.mark.ingresses
@pytest.mark.smoke
class TestPrometheusExporter:
    @pytest.mark.parametrize(
        "ingress_controller, expected_metrics",
        [
            pytest.param(
                {"extra_args": ["-enable-prometheus-metrics"]},
                [
                    'nginx_ingress_controller_nginx_reload_errors_total{class="nginx"} 0',
                    'nginx_ingress_controller_ingress_resources_total{class="nginx",type="master"} 0',
                    'nginx_ingress_controller_ingress_resources_total{class="nginx",type="minion"} 0',
                    'nginx_ingress_controller_ingress_resources_total{class="nginx",type="regular"} 1',
                    "nginx_ingress_controller_nginx_last_reload_milliseconds",
                    'nginx_ingress_controller_nginx_last_reload_status{class="nginx"} 1',
                    'nginx_ingress_controller_nginx_reload_errors_total{class="nginx"} 0',
                    'nginx_ingress_controller_nginx_reloads_total{class="nginx",reason="endpoints"}',
                    'nginx_ingress_controller_nginx_reloads_total{class="nginx",reason="other"}',
                    'nginx_ingress_controller_workqueue_depth{class="nginx",name="taskQueue"}',
                    'nginx_ingress_controller_workqueue_queue_duration_seconds_bucket{class="nginx",name="taskQueue",le=',
                    'nginx_ingress_controller_workqueue_queue_duration_seconds_sum{class="nginx",name="taskQueue"}',
                    'nginx_ingress_controller_workqueue_queue_duration_seconds_count{class="nginx",name="taskQueue"}',
                ],
            )
        ],
        indirect=["ingress_controller"],
    )
    def test_metrics(
        self,
        ingress_controller_endpoint,
        ingress_controller,
        enable_exporter_port,
        expected_metrics,
        ingress_setup,
    ):
        resp = requests.get(ingress_setup.req_url, headers={"host": ingress_setup.ingress_host}, verify=False)
        assert resp.status_code == 200
        req_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
        ensure_connection(req_url, 200)
        resp = requests.get(req_url)
        assert resp.status_code == 200, f"Expected 200 code for /metrics but got {resp.status_code}"
        resp_content = resp.content.decode("utf-8")
        for item in expected_metrics:
            assert item in resp_content

    @pytest.mark.parametrize(
        "ingress_controller, expected_metrics",
        [
            pytest.param(
                {"extra_args": ["-enable-prometheus-metrics", "-enable-latency-metrics"]},
                [
                    'nginx_ingress_controller_upstream_server_response_latency_ms_bucket{class="nginx",code="200",pod_name=',
                    'nginx_ingress_controller_upstream_server_response_latency_ms_sum{class="nginx",code="200",pod_name=',
                    'nginx_ingress_controller_upstream_server_response_latency_ms_count{class="nginx",code="200",pod_name=',
                    'nginx_ingress_controller_ingress_resources_total{class="nginx",type="regular"} 1',
                ],
            )
        ],
        indirect=["ingress_controller"],
    )
    def test_latency_metrics(
        self,
        ingress_controller_endpoint,
        ingress_controller,
        enable_exporter_port,
        expected_metrics,
        ingress_setup,
    ):
        resp = requests.get(ingress_setup.req_url, headers={"host": ingress_setup.ingress_host}, verify=False)
        assert resp.status_code == 200
        req_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
        ensure_connection(req_url, 200)
        resp = requests.get(req_url)
        assert resp.status_code == 200, f"Expected 200 code for /metrics but got {resp.status_code}"
        resp_content = resp.content.decode("utf-8")
        for item in expected_metrics:
            assert item in resp_content


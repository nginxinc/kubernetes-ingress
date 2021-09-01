import requests
import pytest
import json

from suite.fixtures import PublicEndpoint
from suite.resources_utils import (
    create_secret_from_yaml,
    delete_secret,
    ensure_connection_to_public_endpoint,
    create_items_from_yaml,
    delete_items_from_yaml,
    create_example_app,
    delete_common_app,
    wait_until_all_pods_are_ready,
    ensure_response_from_backend,
    get_test_file_name,
    get_last_reload_time,
    write_to_json,
    scale_deployment,
    get_pods_amount,
    wait_before_test,
    get_reload_count
)
from suite.yaml_utils import get_first_ingress_host_from_yaml
from settings import TEST_DATA

paths = ["backend1", "backend2"]
reload_times = {}


class SmokeSetup:
    """
    Encapsulate the Smoke Example details.

    Attributes:
        public_endpoint (PublicEndpoint):
        ingress_host (str):
    """

    def __init__(self, public_endpoint: PublicEndpoint, ingress_host):
        self.public_endpoint = public_endpoint
        self.ingress_host = ingress_host


@pytest.fixture(scope="class", params=["standard", "mergeable"])
def smoke_setup(
    request, kube_apis, ingress_controller_endpoint, ingress_controller, test_namespace
) -> SmokeSetup:
    print("------------------------- Deploy Smoke Example -----------------------------------")
    secret_name = create_secret_from_yaml(
        kube_apis.v1, test_namespace, f"{TEST_DATA}/smoke/smoke-secret.yaml"
    )
    create_items_from_yaml(
        kube_apis, f"{TEST_DATA}/smoke/{request.param}/smoke-ingress.yaml", test_namespace
    )
    ingress_host = get_first_ingress_host_from_yaml(
        f"{TEST_DATA}/smoke/{request.param}/smoke-ingress.yaml"
    )
    create_example_app(kube_apis, "simple", test_namespace)
    wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)
    ensure_connection_to_public_endpoint(
        ingress_controller_endpoint.public_ip,
        ingress_controller_endpoint.port,
        ingress_controller_endpoint.port_ssl,
    )

    def fin():
        print("Clean up the Smoke Application:")
        delete_common_app(kube_apis, "simple", test_namespace)
        delete_items_from_yaml(
            kube_apis, f"{TEST_DATA}/smoke/{request.param}/smoke-ingress.yaml", test_namespace
        )
        delete_secret(kube_apis.v1, secret_name, test_namespace)
        write_to_json(
            f"reload-{get_test_file_name(request.node.fspath)}.json",
            reload_times
        )

    request.addfinalizer(fin)

    return SmokeSetup(ingress_controller_endpoint, ingress_host)


@pytest.mark.smoke
@pytest.mark.ingresses
class TestSmoke:
    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param({"extra_args": ["-enable-prometheus-metrics"]}, id="one-additional-cli-args"),
            pytest.param(
                {"extra_args": ["-nginx-debug", "-health-status=true", "-enable-prometheus-metrics"]},
                id="some-additional-cli-args",
            ),
        ],
        indirect=True,
    )
    @pytest.mark.parametrize("path", paths)
    def test_response_code_200_and_server_name(
        self, request, ingress_controller, smoke_setup, path
    ):
        req_url = f"https://{smoke_setup.public_endpoint.public_ip}:{smoke_setup.public_endpoint.port_ssl}/{path}"
        metrics_url = f"http://{smoke_setup.public_endpoint.public_ip}:{smoke_setup.public_endpoint.metrics_port}/metrics"
        ensure_response_from_backend(req_url, smoke_setup.ingress_host)
        resp = requests.get(req_url, headers={"host": smoke_setup.ingress_host}, verify=False)
        reload_ms = get_last_reload_time(metrics_url, "nginx")
        print(f"last reload duration: {reload_ms} ms")
        reload_times[f"{request.node.name}"] = f"last reload duration: {reload_ms} ms"
        assert resp.status_code == 200
        assert f"Server name: {path}" in resp.text

    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param({"extra_args": ["-enable-prometheus-metrics"]}, id="one-additional-cli-args"),
        ],
        indirect=True,
    )
    def test_reload_count_after_start(
            self, kube_apis, smoke_setup, ingress_controller_prerequisites
    ):
        ns = ingress_controller_prerequisites.namespace

        scale_deployment(kube_apis.v1, kube_apis.apps_v1_api, "nginx-ingress", ns, 0)
        while get_pods_amount(kube_apis.v1, ns) is not 0:
            print(f"Number of replicas not 0, retrying...")
            wait_before_test()
        num = scale_deployment(kube_apis.v1, kube_apis.apps_v1_api, "nginx-ingress", ns, 1)
        assert num is None

        metrics_url = f"http://{smoke_setup.public_endpoint.public_ip}:{smoke_setup.public_endpoint.metrics_port}/metrics"
        count = get_reload_count(metrics_url)

        assert count == 1

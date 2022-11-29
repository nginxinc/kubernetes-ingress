import pytest
import requests
from settings import TEST_DATA
from suite.utils.resources_utils import (
    create_secret_from_yaml,
    delete_secret,
    ensure_response_from_backend,
    wait_before_test,
)
from suite.utils.yaml_utils import get_first_host_from_yaml


@pytest.mark.vs
@pytest.mark.skip_for_nginx_oss
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {"type": "complete", "extra_args": [f"-enable-custom-resources", f"-enable-service-insight"]},
            {"example": "virtual-server", "app_type": "simple"},
        )
    ],
    indirect=True,
)
class TestHealthCheckVsHttp:
    def test_responses_svc_insight_http(
        self, request, kube_apis, crd_ingress_controller, virtual_server_setup, ingress_controller_endpoint
    ):
        """test responses from service insight endpoint with http"""
        vs_source = f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml"
        host = get_first_host_from_yaml(vs_source)
        req_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.service_insight_port}/probe/{host}"
        ensure_response_from_backend(req_url, virtual_server_setup.vs_host)
        resp = requests.get(req_url)
        assert resp.status_code == 200, f"Expected 200 code for /probe/{host} but got {resp.status_code}"
        print(resp.json())


@pytest.fixture(scope="class")
def https_secret_setup(request, kube_apis, test_namespace):
    print("------------------------- Deploy Secret -----------------------------------")
    secret_name = create_secret_from_yaml(kube_apis.v1, "nginx-ingress", f"{TEST_DATA}/service-insight/secret.yaml")

    def fin():
        delete_secret(kube_apis.v1, secret_name, "nginx-ingress")

    request.addfinalizer(fin)


@pytest.mark.vs
@pytest.mark.skip_for_nginx_oss
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    f"-enable-custom-resources",
                    f"-enable-service-insight",
                    f"-service-insight-tls-secret=nginx-ingress/test-secret",
                ],
            },
            {"example": "virtual-server", "app_type": "simple"},
        )
    ],
    indirect=True,
)
class TestHealthCheckVsHttps:
    def test_responses_svc_insight_https(
        self,
        request,
        kube_apis,
        https_secret_setup,
        ingress_controller_endpoint,
        crd_ingress_controller,
        virtual_server_setup,
    ):
        """test responses from service insight endpoint with https"""
        vs_source = f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml"
        host = get_first_host_from_yaml(vs_source)
        req_url = f"https://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.service_insight_port}/probe/{host}"
        ensure_response_from_backend(req_url, virtual_server_setup.vs_host)
        resp = requests.get(req_url, verify=False)
        assert resp.status_code == 200, f"Expected 200 code for /probe/{host} but got {resp.status_code}"
        print(resp.json())

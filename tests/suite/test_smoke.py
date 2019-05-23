import requests
import pytest

from suite.fixtures import PublicEndpoint
from suite.resources_utils import create_secret_from_yaml, delete_secret, \
    ensure_connection_to_public_endpoint, create_items_from_yaml, \
    delete_items_from_yaml, create_common_app, delete_common_app, \
    wait_until_all_pods_are_ready, wait_before_test, ensure_response_from_backend
from suite.yaml_utils import get_first_ingress_host_from_yaml
from settings import TEST_DATA

paths = ["backend1", "backend2"]


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
def smoke_setup(request, kube_apis, ingress_controller_endpoint, ingress_controller, test_namespace) -> SmokeSetup:
    print("------------------------- Deploy Smoke Example -----------------------------------")
    secret_name = create_secret_from_yaml(kube_apis.v1, test_namespace, f"{TEST_DATA}/smoke/smoke-secret.yaml")
    create_items_from_yaml(kube_apis, f"{TEST_DATA}/smoke/{request.param}/smoke-ingress.yaml", test_namespace)
    ingress_host = get_first_ingress_host_from_yaml(f"{TEST_DATA}/smoke/{request.param}/smoke-ingress.yaml")
    common_app = create_common_app(kube_apis.v1, kube_apis.extensions_v1_beta1, test_namespace)
    wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)
    ensure_connection_to_public_endpoint(ingress_controller_endpoint.public_ip,
                                         ingress_controller_endpoint.port,
                                         ingress_controller_endpoint.port_ssl)

    def fin():
        print("Clean up the Smoke Application:")
        delete_common_app(kube_apis.v1, kube_apis.extensions_v1_beta1, common_app, test_namespace)
        delete_items_from_yaml(kube_apis, f"{TEST_DATA}/smoke/{request.param}/smoke-ingress.yaml",
                               test_namespace)
        delete_secret(kube_apis.v1, secret_name, test_namespace)

    request.addfinalizer(fin)

    return SmokeSetup(ingress_controller_endpoint, ingress_host)


@pytest.mark.smoke
class TestSmoke:
    @pytest.mark.parametrize("path", paths)
    def test_response_code_200_and_server_name(self, smoke_setup, path):
        req_url = f"https://{smoke_setup.public_endpoint.public_ip}:{smoke_setup.public_endpoint.port_ssl}/{path}"
        ensure_response_from_backend(req_url, smoke_setup.ingress_host)
        resp = requests.get(req_url, headers={"host": smoke_setup.ingress_host}, verify=False)
        assert resp.status_code == 200
        assert f"Server name: {path}" in resp.text

import pytest
import requests
from settings import DEPLOYMENTS, TEST_DATA
from suite.utils.custom_assertions import wait_and_assert_status_code
from suite.utils.custom_resources_utils import create_crd_from_yaml, delete_crd
from suite.utils.resources_utils import (
    create_service_from_yaml,
    delete_service,
    patch_rbac,
    read_service,
    replace_service,
    wait_before_test,
)
from suite.utils.vs_vsr_resources_utils import (
    create_virtual_server_from_yaml,
    delete_virtual_server,
    patch_virtual_server_from_yaml,
)
from suite.utils.yaml_utils import get_first_host_from_yaml, get_name_from_yaml, get_paths_from_vs_yaml


@pytest.mark.vs
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
class TestHealtcheckVS:
    def test_responses_after_setup(
        self, request, kube_apis, crd_ingress_controller, virtual_server_setup, ingress_controller_endpoint
    ):
        print("\nStep 1: initial check")
        wait_before_test()
        wait_and_assert_status_code(200, virtual_server_setup.backend_1_url, virtual_server_setup.vs_host)
        wait_and_assert_status_code(200, virtual_server_setup.backend_2_url, virtual_server_setup.vs_host)

        vs_source = f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml"
        host = get_first_host_from_yaml(vs_source)
        req_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.service_insight_port}/probe/{host}"
        resp = requests.get(req_url)
        assert resp.status_code == 200, f"Expected 200 code for /probe/{host} but got {resp.status_code}"
        print(resp.json())

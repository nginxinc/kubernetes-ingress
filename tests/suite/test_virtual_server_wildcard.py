import time

import pytest
from settings import TEST_DATA
from suite.utils.custom_assertions import wait_and_assert_status_code
from suite.utils.custom_resources_utils import read_custom_resource
from suite.utils.resources_utils import wait_before_test
from suite.utils.vs_vsr_resources_utils import create_virtual_server_from_yaml, delete_virtual_server


@pytest.mark.vs
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {"type": "complete", "extra_args": [f"-enable-custom-resources"]},
            {"example": "virtual-server", "app_type": "simple"},
        )
    ],
    indirect=True,
)
class TestVirtualServerWildcard:
    def test_vs_status(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        wait_and_assert_status_code(200, virtual_server_setup.backend_1_url, virtual_server_setup.vs_host)
        wait_and_assert_status_code(200, virtual_server_setup.backend_2_url, virtual_server_setup.vs_host)
        wait_and_assert_status_code(404, virtual_server_setup.backend_1_url, "test.example.com")
        wait_and_assert_status_code(404, virtual_server_setup.backend_2_url, "test.example.com")

        # create virtual server with wildcard hostname
        retry = 0
        manifest_vs_wc = f"{TEST_DATA}/virtual-server-wildcard/virtual-server-wildcard.yaml"
        vs_wc_name = create_virtual_server_from_yaml(
            kube_apis.custom_objects, manifest_vs_wc, virtual_server_setup.namespace
        )
        wait_before_test()
        response = {}
        while ("status" not in response) and (retry <= 60):
            print("Waiting for VS status update...")
            time.sleep(1)
            retry += 1
            response = read_custom_resource(
                kube_apis.custom_objects,
                virtual_server_setup.namespace,
                "virtualservers",
                vs_wc_name,
            )

        assert response["status"]["reason"] == "AddedOrUpdated" and response["status"]["state"] == "Valid"
        wait_and_assert_status_code(200, virtual_server_setup.backend_1_url, "test.example.com")
        wait_and_assert_status_code(200, virtual_server_setup.backend_2_url, "test.example.com")
        wait_and_assert_status_code(404, virtual_server_setup.backend_1_url, "test.xexample.com")
        wait_and_assert_status_code(404, virtual_server_setup.backend_2_url, "test.xexample.com")

        delete_virtual_server(kube_apis.custom_objects, vs_wc_name, virtual_server_setup.namespace)

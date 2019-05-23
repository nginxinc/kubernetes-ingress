import pytest

import requests

from settings import TEST_DATA
from suite.custom_resources_utils import patch_virtual_server_from_yaml
from suite.resources_utils import wait_before_test, ensure_response_from_backend


def execute_assertions(resp_1, resp_2, resp_3):
    assert resp_1.status_code == 200
    assert "Server name: backend1-" in resp_1.text
    assert resp_2.status_code == 200
    assert "Server name: backend3-" in resp_2.text
    assert resp_3.status_code == 200
    assert "Server name: backend4-" in resp_3.text


@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-advanced-routing", "app_type": "advanced-routing"})],
                         indirect=True)
class TestAdvancedRouting:
    def test_flow_with_header(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        ensure_response_from_backend(virtual_server_setup.backend_1_url, virtual_server_setup.vs_host)

        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host, "x-version": "future"})
        resp_2 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host, "x-version": "deprecated"})
        resp_3 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host, "x-version-invalid": "deprecated"})
        execute_assertions(resp_1, resp_2, resp_3)

    def test_flow_with_argument(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-advanced-routing/virtual-server-argument.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)

        resp_1 = requests.get(virtual_server_setup.backend_1_url + "?arg1=v1",
                              headers={"host": virtual_server_setup.vs_host})
        resp_2 = requests.get(virtual_server_setup.backend_1_url + "?arg1=v2",
                              headers={"host": virtual_server_setup.vs_host})
        resp_3 = requests.get(virtual_server_setup.backend_1_url + "?argument1=v1",
                              headers={"host": virtual_server_setup.vs_host})
        execute_assertions(resp_1, resp_2, resp_3)

    def test_flow_with_cookie(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-advanced-routing/virtual-server-cookie.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)

        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host}, cookies={"user": "some"})
        resp_2 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host}, cookies={"user": "bad"})
        resp_3 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host}, cookies={"user": "anonymous"})
        execute_assertions(resp_1, resp_2, resp_3)

    def test_flow_with_variable(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-advanced-routing/virtual-server-variable.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)

        resp_1 = requests.get(virtual_server_setup.backend_1_url, headers={"host": virtual_server_setup.vs_host})
        resp_2 = requests.post(virtual_server_setup.backend_1_url, headers={"host": virtual_server_setup.vs_host})
        resp_3 = requests.put(virtual_server_setup.backend_1_url, headers={"host": virtual_server_setup.vs_host})
        execute_assertions(resp_1, resp_2, resp_3)

    def test_flow_with_complex_conditions(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-advanced-routing/virtual-server-complex.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)

        resp_1 = requests.get(virtual_server_setup.backend_1_url + "?arg1=v1",
                              headers={"host": virtual_server_setup.vs_host,
                                       "x-version": "future"}, cookies={"user": "some"})
        resp_2 = requests.post(virtual_server_setup.backend_1_url + "?arg1=v2",
                               headers={"host": virtual_server_setup.vs_host,
                                        "x-version": "deprecated"}, cookies={"user": "bad"})
        resp_3 = requests.get(virtual_server_setup.backend_1_url + "?arg1=v2",
                              headers={"host": virtual_server_setup.vs_host,
                                       "x-version": "deprecated"}, cookies={"user": "bad"})
        execute_assertions(resp_1, resp_2, resp_3)

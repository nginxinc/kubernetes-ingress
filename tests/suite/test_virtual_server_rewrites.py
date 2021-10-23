import pytest
import requests
from kubernetes.client.rest import ApiException
from suite.resources_utils import wait_before_test
from suite.vs_vsr_resources_utils import (
    patch_virtual_server_from_yaml,
)
from suite.custom_resources_utils import (
    read_custom_resource,
)
from settings import TEST_DATA

@pytest.mark.vs
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {"type": "complete", "extra_args": [f"-enable-custom-resources"],},
            {"example": "virtual-server-status", "app_type": "simple",},
        )
    ],
    indirect=True,
)
@pytest.mark.rewrite
class TestVirtualServerRewrites:

    def patch_standard_vs(self, kube_apis, virtual_server_setup) -> None:
        """
        Deploy standard virtual-server
        """
        patch_src = f"{TEST_DATA}/virtual-server-rewrites/standard/virtual-server.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )

    def test_prefix_rewrite(
        self, kube_apis, crd_ingress_controller, virtual_server_setup,
    ):
        """
        Test VirtualServer URI rewrite using prefix
        """
        patch_src = f"{TEST_DATA}/virtual-server-rewrites/virtual-server-rewrite-prefix.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()
        resp1 = requests.get(virtual_server_setup.backend_1_url+"/",
                    headers={"host": virtual_server_setup.vs_host})
        resp2 = requests.get(virtual_server_setup.backend_1_url+"/abc",
                    headers={"host": virtual_server_setup.vs_host})
        resp3 = requests.get(virtual_server_setup.backend_2_url,
                    headers={"host": virtual_server_setup.vs_host})
        resp4 = requests.get(virtual_server_setup.backend_2_url+"/",
                    headers={"host": virtual_server_setup.vs_host})    
        resp5 = requests.get(virtual_server_setup.backend_2_url+"/abc",
                    headers={"host": virtual_server_setup.vs_host})          
        self.patch_standard_vs(kube_apis, virtual_server_setup)

        assert ("URI: /\nRequest" in resp1.text
        and "URI: /abc\nRequest" in resp2.text
        and "URI: /backend2_1\nRequest" in resp3.text
        and "URI: /backend2_1/\nRequest" in resp4.text
        and "URI: /backend2_1/abc\nRequest" in resp5.text)

    def test_regex_rewrite(
        self, kube_apis, crd_ingress_controller, virtual_server_setup,
    ):
        """
        Test VirtualServer URI rewrite using regex
        """
        patch_src = f"{TEST_DATA}/virtual-server-rewrites/virtual-server-rewrite-regex.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()
        resp1 = requests.get(virtual_server_setup.backend_1_url,
                    headers={"host": virtual_server_setup.vs_host})
        resp2 = requests.get(virtual_server_setup.backend_1_url+"/",
                    headers={"host": virtual_server_setup.vs_host})
        resp3 = requests.get(virtual_server_setup.backend_2_url+"/abc",
                    headers={"host": virtual_server_setup.vs_host})
        self.patch_standard_vs(kube_apis, virtual_server_setup)

        assert ("URI: /\nRequest" in resp1.text
        and "URI: /\nRequest" in resp2.text
        and "URI: /abc\nRequest" in resp3.text)

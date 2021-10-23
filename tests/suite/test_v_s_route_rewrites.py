import pytest
import requests
from kubernetes.client.rest import ApiException
from suite.resources_utils import wait_before_test
from suite.custom_resources_utils import (
    read_custom_resource,
)
from suite.vs_vsr_resources_utils import (
    patch_virtual_server_from_yaml,
    patch_v_s_route_from_yaml,
    delete_virtual_server,
    create_virtual_server_from_yaml,
)
from settings import TEST_DATA

@pytest.mark.vsr
@pytest.mark.parametrize(
    "crd_ingress_controller, v_s_route_setup",
    [
        (
            {"type": "complete", "extra_args": [f"-enable-custom-resources"],},
            {"example": "virtual-server-route"},
        )
    ],
    indirect=True,
)
@pytest.mark.rewrite
class TestVirtualServerRouteRewrite:
    def patch_standard_vsr(self, kube_apis, v_s_route_setup) -> None:
        """
        Function to revert vsr deployments to valid state
        """
        patch_src_m = f"{TEST_DATA}/virtual-server-route/route-multiple.yaml"
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            patch_src_m,
            v_s_route_setup.route_m.namespace,
        )
        wait_before_test()
        patch_src_s = f"{TEST_DATA}/virtual-server-route/route-single.yaml"
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_s.name,
            patch_src_s,
            v_s_route_setup.route_s.namespace,
        )
        wait_before_test()

    def test_prefix_rewrite(
        self, kube_apis, crd_ingress_controller, v_s_route_setup, v_s_route_app_setup
    ):
        """
        Test VirtualServerRoute URI rewrite using prefix
        """
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        patch_src_m = f"{TEST_DATA}/virtual-server-route-rewrites/route-multiple-prefix-regex.yaml"
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            patch_src_m,
            v_s_route_setup.route_m.namespace,
        )
        wait_before_test()
        patch_src_s = f"{TEST_DATA}/virtual-server-route-rewrites/route-single-prefix.yaml"
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_s.name,
            patch_src_s,
            v_s_route_setup.route_s.namespace,
        )
        wait_before_test()
        resp1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}/",
                              headers={"host": v_s_route_setup.vs_host})
        resp2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}/abc",
                              headers={"host": v_s_route_setup.vs_host})
        resp3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp4 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}/",
                              headers={"host": v_s_route_setup.vs_host})
        resp5 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}/abc",
                              headers={"host": v_s_route_setup.vs_host})
        self.patch_standard_vsr(kube_apis, v_s_route_setup)

        assert ( "URI: /\nRequest" in resp1.text
        and "URI: /abc\nRequest" in resp2.text
        and "URI: /backend2_1\nRequest" in resp3.text
        and "URI: /backend2_1/\nRequest" in resp4.text
        and "URI: /backend2_1/abc\nRequest" in resp5.text
        )

    def test_regex_rewrite(
        self, kube_apis, crd_ingress_controller, v_s_route_setup, v_s_route_app_setup
    ):
        """
        Test VirtualServerRoute URI rewrite using regex
        """
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        patch_src_m = f"{TEST_DATA}/virtual-server-route-rewrites/route-multiple-prefix-regex.yaml"
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            patch_src_m,
            v_s_route_setup.route_m.namespace,
        )

        wait_before_test()
        resp1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}/",
                              headers={"host": v_s_route_setup.vs_host})                   
        resp3 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}/abc",
                              headers={"host": v_s_route_setup.vs_host})

        self.patch_standard_vsr(kube_apis, v_s_route_setup)

        assert ( "URI: /\nRequest" in resp1.text
        and "URI: /\nRequest" in resp2.text
        and "URI: /abc\nRequest" in resp3.text
        )
    
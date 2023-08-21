import pytest
import requests
from settings import TEST_DATA
from suite.utils.custom_resources_utils import create_gc_from_yaml, delete_gc
from suite.utils.resources_utils import create_secret_from_yaml, delete_secret, wait_before_test
from suite.utils.vs_vsr_resources_utils import patch_virtual_server_from_yaml


@pytest.mark.test
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    f"-global-configuration=nginx-ingress/nginx-configuration",
                    f"-enable-leader-election=false",
                ],
            },
            {
                "example": "virtual-server-custom-listeners",
                "app_type": "simple",
            },
        )
    ],
    indirect=True,
)
class TestVirtualServerCustomListeners:
    def restore_default_vs(self, kube_apis, virtual_server_setup) -> None:
        """
        Function to revert vs deployment to valid state
        """
        patch_src = f"{TEST_DATA}/virtual-server-status/standard/virtual-server.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()

    def test_custom_listeners(self, kube_apis, crd_ingress_controller, virtual_server_setup) -> None:
        print("\nStep 1: Create GC resource")
        secret_name = create_secret_from_yaml(
            kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/virtual-server-tls/tls-secret.yaml"
        )
        global_config_file = f"{TEST_DATA}/virtual-server-custom-listeners/global-configuration.yaml"
        gc_resource = create_gc_from_yaml(kube_apis.custom_objects, global_config_file, "nginx-ingress")
        patch_src = f"{TEST_DATA}/virtual-server-custom-listeners/virtual-server.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()
        print(virtual_server_setup.backend_1_url_custom_ssl)
        resp1 = requests.get(
            virtual_server_setup.backend_1_url_custom_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        print(virtual_server_setup.backend_1_url_custom)
        resp2 = requests.get(
            virtual_server_setup.backend_1_url_custom,
            headers={"host": virtual_server_setup.vs_host},
        )
        print(resp1.status_code)
        print(resp1.text)
        print(resp2.status_code)
        print(resp2.text)

        print(virtual_server_setup.backend_1_url_ssl)
        resp3 = requests.get(
            virtual_server_setup.backend_1_url_ssl,
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )
        print(virtual_server_setup.backend_1_url)
        resp4 = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )
        print(resp3.status_code)
        print(resp3.text)
        print(resp4.status_code)
        print(resp4.text)

        delete_secret(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        delete_gc(kube_apis.custom_objects, gc_resource, "nginx-ingress")
        self.restore_default_vs(kube_apis, virtual_server_setup)

        assert resp1.status_code == 200 and resp2.status_code == 200
        assert resp3.status_code == 404 and resp4.status_code == 404

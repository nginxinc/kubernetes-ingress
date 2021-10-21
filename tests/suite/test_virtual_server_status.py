import pytest
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
            {"type": "complete", "extra_args": [f"-enable-custom-resources", f"-enable-leader-election=false"],},
            {"example": "virtual-server-status", "app_type": "simple",},
        )
    ],
    indirect=True,
)
class TestVirtualServerStatus:

    def patch_valid_vs(self, kube_apis, virtual_server_setup) -> None:
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

    @pytest.mark.smoke
    def test_status_valid(
        self, kube_apis, crd_ingress_controller, virtual_server_setup,
    ):
        """
        Test VirtualServer status with a valid fields in yaml
        """
        response = read_custom_resource(
            kube_apis.custom_objects,
            virtual_server_setup.namespace,
            "virtualservers",
            virtual_server_setup.vs_name,
        )
        assert (
            response["status"]
            and response["status"]["reason"] == "AddedOrUpdated"
            and response["status"]["state"] == "Valid"
        )

    def test_status_invalid(
        self, kube_apis, crd_ingress_controller, virtual_server_setup,
    ):
        """
        Test VirtualServer status with a invalid path pattern
        """
        patch_src = f"{TEST_DATA}/virtual-server-status/invalid-state.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()
        response = read_custom_resource(
            kube_apis.custom_objects,
            virtual_server_setup.namespace,
            "virtualservers",
            virtual_server_setup.vs_name,
        )
        self.patch_valid_vs(kube_apis, virtual_server_setup)
        assert (
            response["status"]
            and response["status"]["reason"] == "Rejected"
            and response["status"]["state"] == "Invalid"
        )

    @pytest.mark.skip_for_nginx_oss
    def test_status_warning(
        self, kube_apis, crd_ingress_controller, virtual_server_setup,
    ):
        """
        Test VirtualServer status with conflicting Upstream fields
        Only for N+ since Slow-start isn
        """
        patch_src = f"{TEST_DATA}/virtual-server-status/warning-state.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()
        response = read_custom_resource(
            kube_apis.custom_objects,
            virtual_server_setup.namespace,
            "virtualservers",
            virtual_server_setup.vs_name,
        )
        self.patch_valid_vs(kube_apis, virtual_server_setup)
        assert (
            response["status"]
            and response["status"]["reason"] == "AddedOrUpdatedWithWarning"
            and response["status"]["state"] == "Warning"
        )

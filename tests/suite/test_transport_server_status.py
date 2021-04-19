import pytest

from suite.resources_utils import wait_before_test
from suite.custom_resources_utils import (
    read_ts,
    patch_ts,
)
from settings import TEST_DATA


@pytest.mark.ts
@pytest.mark.parametrize(
    "crd_ingress_controller, transport_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args":
                    [
                        "-global-configuration=nginx-ingress/nginx-configuration",
                        "-enable-leader-election=false"
                    ]
            },
            {"example": "transport-server-status", "app_type": "simple"},
        )
    ],
    indirect=True,
)
class TestTransportServerStatus:

    def restore_ts(self, kube_apis, transport_server_setup) -> None:
        """
        Function to revert a TransportServer resource to a valid state.
        """
        patch_src = f"{TEST_DATA}/transport-server-status/standard/transport-server.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )

    @pytest.mark.smoke
    def test_status_valid(
        self, kube_apis, crd_ingress_controller, transport_server_setup,
    ):
        """
        Test TransportServer status with valid fields in yaml.
        """
        response = read_ts(
            kube_apis.custom_objects,
            transport_server_setup.namespace,
            transport_server_setup.name,
        )
        assert (
            response["status"]
            and response["status"]["reason"] == "AddedOrUpdated"
            and response["status"]["state"] == "Valid"
        )

    def test_status_warning(
        self, kube_apis, crd_ingress_controller, transport_server_setup,
    ):
        """
        Test TransportServer status with a missing listener.
        """
        patch_src = f"{TEST_DATA}/transport-server-status/rejected-warning.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )
        wait_before_test()
        response = read_ts(
            kube_apis.custom_objects,
            transport_server_setup.namespace,
            transport_server_setup.name,
        )
        self.restore_ts(kube_apis, transport_server_setup)
        assert (
            response["status"]
            and response["status"]["reason"] == "Rejected"
            and response["status"]["state"] == "Warning"
        )

    def test_status_invalid(
        self, kube_apis, crd_ingress_controller, transport_server_setup,
    ):
        """
        Test TransportServer status with an invalid protocol.
        """
        patch_src = f"{TEST_DATA}/transport-server-status/rejected-invalid.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )
        wait_before_test()
        response = read_ts(
            kube_apis.custom_objects,
            transport_server_setup.namespace,
            transport_server_setup.name,
        )
        self.restore_ts(kube_apis, transport_server_setup)
        assert (
            response["status"]
            and response["status"]["reason"] == "Rejected"
            and response["status"]["state"] == "Invalid"
        )

import pytest
from settings import TEST_DATA
from suite.utils.custom_resources_utils import patch_gc_from_yaml, patch_ts_from_yaml, read_ts
from suite.utils.resources_utils import get_events_for_object, wait_before_test


@pytest.mark.ts
@pytest.mark.parametrize(
    "crd_ingress_controller, transport_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    "-global-configuration=nginx-ingress/nginx-configuration",
                    "-enable-leader-election=false",
                    "-enable-prometheus-metrics=true",
                ],
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
        patch_ts_from_yaml(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )

    @pytest.mark.smoke
    def test_status_valid(
        self,
        kube_apis,
        crd_ingress_controller,
        transport_server_setup,
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
        self,
        kube_apis,
        crd_ingress_controller,
        transport_server_setup,
    ):
        """
        Test TransportServer status with a missing listener.
        """
        patch_src = f"{TEST_DATA}/transport-server-status/rejected-warning.yaml"
        patch_ts_from_yaml(
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
            and "Listener invalid-listener doesn't exist" in response["status"]["message"]
        )

    def test_status_invalid(
        self,
        kube_apis,
        crd_ingress_controller,
        transport_server_setup,
    ):
        """
        Test TransportServer status with an invalid protocol.
        """
        patch_src = f"{TEST_DATA}/transport-server-status/rejected-invalid.yaml"
        patch_ts_from_yaml(
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
            and 'spec.listener.protocol: Invalid value: "invalid-protocol": must specify a valid protocol. '
            "Accepted values: HTTP,TCP,UDP" in response["status"]["message"]
        )

    def test_valid_status_invalid_udp_listener(self, kube_apis, crd_ingress_controller, transport_server_setup):
        """
        Test TransportServer status with another listener invalid.
        """
        global_config_file = (
            f"{TEST_DATA}/transport-server-status/standard/global-configuration-invalid-preceding-udp.yaml"
        )
        gc_resource = patch_gc_from_yaml(
            kube_apis.custom_objects, "nginx-configuration", global_config_file, "nginx-ingress"
        )
        wait_before_test()
        patch_src = f"{TEST_DATA}/transport-server-status/standard/transport-server.yaml"
        patch_ts_from_yaml(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )
        response = read_ts(
            kube_apis.custom_objects,
            transport_server_setup.namespace,
            transport_server_setup.name,
        )
        self.restore_ts(kube_apis, transport_server_setup)
        assert (
            response["status"]
            and response["status"]["reason"] == "AddedOrUpdated"
            and response["status"]["state"] == "Valid"
        )

        gc_events = get_events_for_object(kube_apis.v1, "nginx-ingress", "nginx-configuration")
        gc_event_latest = gc_events[-1]
        print(gc_event_latest)
        assert (
            gc_event_latest.reason == "AddedOrUpdatedWithError"
            and gc_event_latest.type == "Warning"
            and "GlobalConfiguration nginx-ingress/nginx-configuration is updated with errors: "
            "spec.listeners[0].port: Forbidden: Listener dns-udp: port 9113 is forbidden" in gc_event_latest.message
        )

    def test_valid_status_invalid_tcp_listener(self, kube_apis, crd_ingress_controller, transport_server_setup):
        """
        Test TransportServer status with listener using invalid port.
        """
        global_config_file = f"{TEST_DATA}/transport-server-status/standard/global-configuration-invalid-tcp.yaml"
        gc_resource = patch_gc_from_yaml(
            kube_apis.custom_objects, "nginx-configuration", global_config_file, "nginx-ingress"
        )
        wait_before_test()
        patch_src = f"{TEST_DATA}/transport-server-status/standard/transport-server.yaml"
        patch_ts_from_yaml(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )
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
            and "Listener dns-tcp doesn't exist" in response["status"]["message"]
        )
        gc_events = get_events_for_object(kube_apis.v1, "nginx-ingress", "nginx-configuration")
        gc_event_latest = gc_events[-1]
        print(gc_event_latest)
        assert (
            gc_event_latest.reason == "AddedOrUpdatedWithError"
            and gc_event_latest.type == "Warning"
            and "GlobalConfiguration nginx-ingress/nginx-configuration is updated with errors: "
            "spec.listeners[1].port: Forbidden: Listener dns-tcp: port 9113 is forbidden" in gc_event_latest.message
        )

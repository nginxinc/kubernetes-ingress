import pytest
from settings import TEST_DATA
from suite.utils.resources_utils import (
    create_license,
    ensure_connection_to_public_endpoint,
    get_events_for_object,
    get_first_pod_name,
    get_reload_count,
    is_secret_present,
    replace_configmap_from_yaml,
    wait_before_test,
)


def assert_event(event_list, event_type, reason, message_substring):
    """
    Assert that an event with specific type, reason, and message substring exists.

    :param event_list: List of events
    :param event_type: 'Normal' or 'Warning'
    :param reason: Event reason
    :param message_substring: Substring expected in the event message
    """
    for event in event_list:
        if event.type == event_type and event.reason == reason and message_substring in event.message:
            return
    assert (
        False
    ), f"Expected event with type '{event_type}', reason '{reason}', and message containing '{message_substring}' not found."


@pytest.mark.skip_for_nginx_oss
@pytest.mark.ingresses
@pytest.mark.smoke
class TestMGMTConfigMap:
    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param(
                {"extra_args": ["-enable-prometheus-metrics"]},
            )
        ],
        indirect=["ingress_controller"],
    )
    def test_mgmt_configmap_events(
        self,
        cli_arguments,
        kube_apis,
        ingress_controller_prerequisites,
        ingress_controller,
        ingress_controller_endpoint,
    ):
        ensure_connection_to_public_endpoint(
            ingress_controller_endpoint.public_ip,
            ingress_controller_endpoint.port,
            ingress_controller_endpoint.port_ssl,
        )
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        metrics_url = (
            f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
        )

        print("Step 1: get reload count")
        reload_count = get_reload_count(metrics_url)

        wait_before_test(1)
        print(f"Step 1a: initial reload count is {reload_count}")

        print("Step 2: create duplicate existing secret with new name")
        license_name = create_license(
            kube_apis.v1,
            ingress_controller_prerequisites.namespace,
            cli_arguments["plus-jwt"],
            license_token_name="license-token-changed",
        )
        assert is_secret_present(kube_apis.v1, license_name, ingress_controller_prerequisites.namespace)

        print("Step 3: update the ConfigMap/license-token-secret-name to the new secret")
        replace_configmap_from_yaml(
            kube_apis.v1,
            "nginx-config-mgmt",
            ingress_controller_prerequisites.namespace,
            f"{TEST_DATA}/mgmt-configmap-keys/plus-token-name-keys.yaml",
        )

        wait_before_test()

        print("Step 4: check reload count has incremented")
        new_reload_count = get_reload_count(metrics_url)
        print(f"Step 4a: new reload count is {new_reload_count}")
        assert new_reload_count > reload_count

        print("Step 5: check pod for SecretUpdated event")
        events = get_events_for_object(
            kube_apis.v1,
            ingress_controller_prerequisites.namespace,
            ic_pod_name,
        )

        # Assert that the 'SecretUpdated' event is present
        assert_event(
            events,
            "Normal",
            "SecretUpdated",
            f"the special Secret {ingress_controller_prerequisites.namespace}/{license_name} was updated",
        )

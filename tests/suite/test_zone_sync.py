import re

import pytest
from settings import TEST_DATA
from suite.utils.resources_utils import (
    create_license,
    create_secret_from_yaml,
    ensure_connection_to_public_endpoint,
    get_events_for_object,
    get_first_pod_name,
    get_nginx_template_conf,
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


@pytest.mark.zonesync
@pytest.mark.skip_for_nginx_oss
@pytest.mark.ingresses
@pytest.mark.smoke
class TestZoneSyncLifecycle:
    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param(
                {"extra_args": ["-enable-prometheus-metrics"]},
            )
        ],
        indirect=["ingress_controller"],
    )
    def test_nic_starts_without_zonesync(
        self,
        cli_arguments,
        kube_apis,
        ingress_controller_prerequisites,
        ingress_controller,
        ingress_controller_endpoint,
    ):
        """
        Test:
        1. NIC starts without zone-sync configured in the `nginx-config`
        2. Apply config map with zone-sync disabled - no zone sync created, no headless service created.
        """
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

        wait_before_test(5)
        print(f"Step 1a: initial reload count is {reload_count}")

        configmap_name = "nginx-config"

        print("Step 2: update the ConfigMap nginx-config - set zone-sync: false")
        replace_configmap_from_yaml(
            kube_apis.v1,
            configmap_name,
            ingress_controller_prerequisites.namespace,
            f"{TEST_DATA}/zone-sync/configmap-with-zonesync-disabled.yaml",
        )

        wait_before_test(3)

        print("Step 4: check reload count has incremented")
        new_reload_count = get_reload_count(metrics_url)

        print(f"Step 4a: new reload count is {new_reload_count}")
        assert new_reload_count > reload_count

        print("Step 5: check pod for ConfigMap updated event")
        config_events = get_events_for_object(
            kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name
        )

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # TODO:
        # 1. assert headless service is NOT creates
        # 2. assert zone-sync config NOT present in nginx.conf

    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param(
                {"extra_args": ["-enable-prometheus-metrics"]},
            )
        ],
        indirect=["ingress_controller"],
    )
    def test_apply_minimal_default_zonesync_config(
        self,
        cli_arguments,
        kube_apis,
        ingress_controller_prerequisites,
        ingress_controller,
        ingress_controller_endpoint,
    ):
        """
        Test:
        1. NIC starts with zone-sync not configured in the `nginx-config`.
        2. Minimal `zone-sync` config is applied. Zone sync and headless service is created.
        """
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

        wait_before_test(3)
        print(f"Step 1a: initial reload count is {reload_count}")

        configmap_name = "nginx-config"

        print("Step 2: update the ConfigMap nginx-config - set zone-sync minimal configuration")
        replace_configmap_from_yaml(
            kube_apis.v1,
            configmap_name,
            ingress_controller_prerequisites.namespace,
            f"{TEST_DATA}/zone-sync/configmap-with-zonesync-minimal.yaml",
        )

        wait_before_test(3)

        print("Step 4: check reload count has incremented")
        new_reload_count = get_reload_count(metrics_url)

        print(f"Step 4a: new reload count is {new_reload_count}")
        assert new_reload_count > reload_count

        print("Step 5: check pod for ConfigMap updated event")
        pod_events = get_events_for_object(
            kube_apis.v1,
            ingress_controller_prerequisites.namespace,
            ic_pod_name,
        )

        # Assert that the 'ConfigMapUpdated' event is present
        config_events = get_events_for_object(
            kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name
        )

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

    #     # TODO:
    #     # 1. assert headless service is created
    #     # 2. assert zone-sync config present in nginx.conf

    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param(
                {"extra_args": ["-enable-prometheus-metrics"]},
            )
        ],
        indirect=["ingress_controller"],
    )
    def test_enable_customized_zonesync_resolver_valid(
        self,
        cli_arguments,
        kube_apis,
        ingress_controller_prerequisites,
        ingress_controller,
        ingress_controller_endpoint,
    ):
        """
        Test:
        1. NIC starts with zone-sync not configured in the `nginx-config`
        2. Apply zone-sync config enabled, custom port, resolver valid time 
        """
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

        wait_before_test(3)
        print(f"Step 1a: initial reload count is {reload_count}")

        configmap_name = "nginx-config"

        print("Step 2: update the ConfigMap nginx-config - enable zone-sync with custom resolver time")
        replace_configmap_from_yaml(
            kube_apis.v1,
            configmap_name,
            ingress_controller_prerequisites.namespace,
            f"{TEST_DATA}/zone-sync/configmap-with-zonesync-resolver-valid.yaml",
        )

        wait_before_test(3)

        print("Step 4: check reload count has incremented")
        new_reload_count = get_reload_count(metrics_url)

        print(f"Step 4a: new reload count is {new_reload_count}")
        assert new_reload_count > reload_count

        print("Step 5: check pod for ConfigMap updated event")
        # Assert that the 'ConfigMapUpdated' event is present
        config_events = get_events_for_object(
            kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name
        )

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # TODO:
        # 1. assert headless service is created
        # 2. assert zone-sync config present in nginx.conf with custom resolver time

    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param(
                {"extra_args": ["-enable-prometheus-metrics"]},
            )
        ],
        indirect=["ingress_controller"],
    )
    def test_update_zonesync_port(
        self,
        cli_arguments,
        kube_apis,
        ingress_controller_prerequisites,
        ingress_controller,
        ingress_controller_endpoint,
    ):
        """
        Test:
        1. that NIC starts with zone-sync not configured in the `nginx-config`
        2. 
        """
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

        configmap_name = "nginx-config"

        print("Step 2: update the ConfigMap nginx-config - set zone-sync-port to custom value")
        replace_configmap_from_yaml(
            kube_apis.v1,
            configmap_name,
            ingress_controller_prerequisites.namespace,
            f"{TEST_DATA}/zone-sync/configmap-with-zonesync-minimal-changed-port.yaml",
        )

        wait_before_test(3)

        print("Step 4: check reload count has incremented")
        new_reload_count = get_reload_count(metrics_url)

        print(f"Step 4a: new reload count is {new_reload_count}")
        assert new_reload_count > reload_count

        print("Step 5: check pod for ConfigMap updated event")
        config_events = get_events_for_object(
            kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name
        )

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error", # TODO: Verify if the message is correct
        )

        print("Step 6: Update the ConfigMap nginx-config - re-configure zone-sync port back to default port")
        replace_configmap_from_yaml(
            kube_apis.v1,
            configmap_name,
            ingress_controller_prerequisites.namespace,
            f"{TEST_DATA}/zone-sync/configmap-with-zonesync-minimal.yaml",
        )

        wait_before_test(3)

        print("Step 7: check reload count has incremented")
        new_reload_count = get_reload_count(metrics_url)

        print(f"Step 7a: new reload count is {new_reload_count}")
        assert new_reload_count > reload_count

        print("Step 8: check pod for ConfigMap updated event")
        config_events = get_events_for_object(
            kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name
        )

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param(
                {"extra_args": ["-enable-prometheus-metrics"]},
            )
        ],
        indirect=["ingress_controller"],
    )
    def test_deactivate_zonesync(
        self,
        cli_arguments,
        kube_apis,
        ingress_controller_prerequisites,
        ingress_controller,
        ingress_controller_endpoint,
    ):
        """
        Test:
        1. NIC starts with zone-sync not configured in the `nginx-config`
        2. 
        """
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

        configmap_name = "nginx-config"

        print("Step 2: update the ConfigMap nginx-config - set zone-sync-port default")
        replace_configmap_from_yaml(
            kube_apis.v1,
            configmap_name,
            ingress_controller_prerequisites.namespace,
            f"{TEST_DATA}/zone-sync/configmap-with-zonesync-minimal.yaml",
        )

        wait_before_test(1)

        print("Step 4: check reload count has incremented")
        new_reload_count = get_reload_count(metrics_url)

        print(f"Step 4a: new reload count is {new_reload_count}")
        assert new_reload_count > reload_count

        print("Step 5: check pod for ConfigMap updated event")
        config_events = get_events_for_object(
            kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name
        )

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # Disable zone-sync
        print("Step 6: update the ConfigMap nginx-config - remove zone-sync entries")
        replace_configmap_from_yaml(
            kube_apis.v1,
            configmap_name,
            ingress_controller_prerequisites.namespace,
            f"{TEST_DATA}/zone-sync/configmap-without-zonesync.yaml",
        )

        wait_before_test(1)

        print("Step 7: check reload count has incremented")
        new_reload_count = get_reload_count(metrics_url)

        print(f"Step 7a: new reload count is {new_reload_count}")
        assert new_reload_count > reload_count

        print("Step 8: check pod for ConfigMap updated event")
        # Assert that the 'ConfigMapUpdated' event is present
        config_events = get_events_for_object(
            kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name
        )

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # assert no headless service in place

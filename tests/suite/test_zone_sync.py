import pytest
from settings import TEST_DATA
from suite.utils.resources_utils import (
    ensure_connection_to_public_endpoint,
    get_events_for_object,
    get_first_pod_name,
    get_reload_count,
    replace_configmap_from_yaml,
    wait_before_test,
    get_nginx_template_conf,
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


def assert_zonesync_enabled(nginx_config, port="12345"):
    """
    Assert that zone_sync exists in nginx.conf

    :param nginx_config: NGINX config file `nginx.config`
    """
    #assert "zone_sync;" in nginx_config
    assert f"zone_sync_server nginx-headless.nginx.svc.cluster.local:'{port}'" in nginx_config


def assert_zonesync_disabled(nginx_config):
    """
    Assert that zone_sync doesn't exist in nginx.conf

    :param nginx_config: NGINX config file `nginx.config`
    """
    #assert "zone_sync;" not in nginx_config
    assert "zone_sync_server nginx-headless.nginx.svc.cluster.local" not in nginx_config


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
        get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        metrics_url = (
            f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
        )

        # Verify zone_sync not present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_disabled(nginx_config)

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
        config_events = get_events_for_object(kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name)

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # Verify zone_sync not present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_disabled(nginx_config)


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
        2. Apply the minimal `zone-sync` config.
        3. Verify zone-sync, headless service, and nginx.config zone_sync entry created.
        """
        ensure_connection_to_public_endpoint(
            ingress_controller_endpoint.public_ip,
            ingress_controller_endpoint.port,
            ingress_controller_endpoint.port_ssl,
        )
        get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        metrics_url = (
            f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
        )

        # Verify zone_sync not present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_disabled(nginx_config)

        print("Step 1: get reload count")
        reload_count = get_reload_count(metrics_url)

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
        # Assert that the 'ConfigMapUpdated' event is present
        config_events = get_events_for_object(kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name)

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # Verify zone_sync present in nginx.conf
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_enabled(nginx_config)


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
        2. Apply zone-sync config enabled, custom port, and resolver time
        3. Verify zone-sync, headless service, and nginx.config zone_sync entry created.
        """
        ensure_connection_to_public_endpoint(
            ingress_controller_endpoint.public_ip,
            ingress_controller_endpoint.port,
            ingress_controller_endpoint.port_ssl,
        )
        get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        metrics_url = (
            f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
        )

        # Verify zone_sync not present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_disabled(nginx_config)

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
        config_events = get_events_for_object(kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name)

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_enabled(nginx_config)


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
        1. NIC starts with zone-sync not configured in the `nginx-config`
        2. Apply zone-sync config enabled and custom port
        3. Verify zone-sync, headless service, and nginx.config zone_sync entry created.
        4. Apply default minimal zone-sync config
        5. Verify zone-sync, headless service, and nginx.config zone_sync entry is updated.
        """
        ensure_connection_to_public_endpoint(
            ingress_controller_endpoint.public_ip,
            ingress_controller_endpoint.port,
            ingress_controller_endpoint.port_ssl,
        )
        get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        metrics_url = (
            f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
        )

        # Verify zone_sync not present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_disabled(nginx_config)

        print("Step 1: get reload count")
        reload_count = get_reload_count(metrics_url)

        wait_before_test(3)
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
        config_events = get_events_for_object(kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name)

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # Verify zone_sync present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_enabled(nginx_config, port="34100")

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
        config_events = get_events_for_object(kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name)

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # Verify zone_sync present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_enabled(nginx_config)


    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param(
                {"extra_args": ["-enable-prometheus-metrics"]},
            )
        ],
        indirect=["ingress_controller"],
    )
    def test_zonesync_enabled_and_disabled(
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
        get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        metrics_url = (
            f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
        )

        # Verify zone_sync not present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_disabled(nginx_config)

        print("Step 1: get reload count")
        reload_count = get_reload_count(metrics_url)

        wait_before_test(5)
        print(f"Step 1a: initial reload count is {reload_count}")

        configmap_name = "nginx-config"

        print("Step 2: update the ConfigMap nginx-config - set zone-sync default - minimal")
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
        config_events = get_events_for_object(kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name)

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # Verify zone_sync not present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_enabled(nginx_config)

        print("Step 6: update the ConfigMap nginx-config - set zone-sync disabled")
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
        config_events = get_events_for_object(kube_apis.v1, ingress_controller_prerequisites.namespace, configmap_name)

        assert_event(
            config_events,
            "Normal",
            "Updated",
            f"ConfigMap {ingress_controller_prerequisites.namespace}/{configmap_name} updated without error",
        )

        # Verify zone_sync not present in nginx.conf
        wait_before_test(3)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        assert_zonesync_disabled(nginx_config)

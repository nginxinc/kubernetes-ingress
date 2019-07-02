import requests
import pytest

from settings import TEST_DATA, DEPLOYMENTS
from suite.custom_resources_utils import get_vs_nginx_template_conf, patch_virtual_server_from_yaml, \
    patch_virtual_server, generate_item_with_upstream_option
from suite.resources_utils import get_first_pod_name, wait_before_test, replace_configmap_from_yaml, get_events, \
    create_configmap_from_yaml_with_overriden_key, replace_configmap


def assert_response_codes(resp_1, resp_2, code=200):
    assert resp_1.status_code == code
    assert resp_2.status_code == code


def get_event_count(event_text, events_list) -> int:
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            return events_list[i].count
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def assert_event_count_increased(event_text, count, events_list):
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            assert events_list[i].count > count
            return
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def assert_event(event_text, events_list):
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            return
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def assert_event_starts_with_text_and_contains_errors(event_text, events_list, fields_list):
    for i in range(len(events_list) - 1, -1, -1):
        if str(events_list[i].message).startswith(event_text):
            for field_error in fields_list:
                assert field_error in events_list[i].message
            return
    pytest.fail(f"Failed to find the event starting with \"{event_text}\" in the list. Exiting...")


def assert_template_config_does_not_exist(response):
    assert "No such file or directory" in response


@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-upstream-options", "app_type": "simple"})],
                         indirect=True)
class TestVirtualServerUpstreamOptions:
    def test_nginx_config_defaults(self, kube_apis, ingress_controller_prerequisites,
                                   crd_ingress_controller, virtual_server_setup):
        print("Case 1: no ConfigMap key, no options in VS")
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            virtual_server_setup.namespace,
                                            virtual_server_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)

        assert "random two least_conn;" in config
        assert "ip_hash;" not in config
        assert "hash " not in config
        assert "least_time " not in config

        assert "max_fails=1 fail_timeout=10s;" in config

    @pytest.mark.parametrize('option, option_value, expected_string', [
        ("lb-method", "least_conn", "least_conn;"),
        ("lb-method", "ip_hash", "ip_hash;"),
        ("max-fails", 8, "max_fails=8 "),
        ("fail-timeout", "13s", "fail_timeout=13s;")
    ])
    def test_when_option_in_v_s_only(self, kube_apis, ingress_controller_prerequisites,
                                     crd_ingress_controller, virtual_server_setup,
                                     option, option_value, expected_string):
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"Configuration for {text} was added or updated"
        events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)
        initial_count = get_event_count(vs_event_text, events_vs)
        print(f"Case 2: no ConfigMap {option}, {option} specified in VS")
        new_body = generate_item_with_upstream_option(
            f"{TEST_DATA}/virtual-server-upstream-options/standard/virtual-server.yaml",
            option, option_value)
        patch_virtual_server(kube_apis.custom_objects,
                             virtual_server_setup.vs_name, virtual_server_setup.namespace, new_body)
        wait_before_test(1)
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            virtual_server_setup.namespace,
                                            virtual_server_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host})
        resp_2 = requests.get(virtual_server_setup.backend_2_url,
                              headers={"host": virtual_server_setup.vs_host})
        vs_events = get_events(kube_apis.v1, virtual_server_setup.namespace)

        assert_event_count_increased(vs_event_text, initial_count, vs_events)
        assert expected_string in config
        assert_response_codes(resp_1, resp_2)

    @pytest.mark.parametrize('option, option_value, expected_string, unexpected_string', [
        ("lb-method", "round_robin", [], ["ip_hash;", "least_conn;", "random ", "hash", "least_time "]),
        ("max-fails", "28", ["max_fails=28 "], ["max_fails=1 "]),
        ("fail-timeout", "23s", ["fail_timeout=23s;"], ["fail_timeout=10s;"])
    ])
    def test_when_option_in_config_map_only(self, kube_apis, ingress_controller_prerequisites,
                                            crd_ingress_controller, virtual_server_setup,
                                            option, option_value, expected_string, unexpected_string):
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"Configuration for {text} was updated"
        print(f"Case 3: {option} specified in ConfigMap, no {option} in VS")
        patch_virtual_server_from_yaml(kube_apis.custom_objects, virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-upstream-options/standard/virtual-server.yaml",
                                       virtual_server_setup.namespace)
        config_map_name = ingress_controller_prerequisites.config_map["metadata"]["name"]
        new_configmap = create_configmap_from_yaml_with_overriden_key(
            f"{DEPLOYMENTS}/common/nginx-config.yaml", option, option_value)
        replace_configmap(kube_apis.v1, config_map_name,
                          ingress_controller_prerequisites.namespace,
                          new_configmap)
        wait_before_test(1)
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            virtual_server_setup.namespace,
                                            virtual_server_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host})
        resp_2 = requests.get(virtual_server_setup.backend_2_url,
                              headers={"host": virtual_server_setup.vs_host})
        vs_events = get_events(kube_apis.v1, virtual_server_setup.namespace)

        assert_event(vs_event_text, vs_events)
        for _ in expected_string:
            assert _ in config
        for _ in unexpected_string:
            assert _ not in config
        assert_response_codes(resp_1, resp_2)

    @pytest.mark.parametrize('option, option_value, expected_string, unexpected_string', [
        ("lb-method", "least_conn", ["least_conn;"], ["ip_hash;", "random ", "hash", "least_time "]),
        ("max-fails", 12, ["max_fails=12 "], ["max_fails=1 ", "max_fails=3 "]),
        ("fail-timeout", "1m", ["fail_timeout=1m;"], ["fail_timeout=10s;", "fail_timeout=33s;"])
    ])
    def test_v_s_overrides_config_map(self, kube_apis, ingress_controller_prerequisites,
                                      crd_ingress_controller, virtual_server_setup,
                                      option, option_value, expected_string, unexpected_string):
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"Configuration for {text} was added or updated"
        events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)
        initial_count = get_event_count(vs_event_text, events_vs)
        print(f"Case 4: {option} in ConfigMap, {option} specified in VS")
        new_body = generate_item_with_upstream_option(
            f"{TEST_DATA}/virtual-server-upstream-options/standard/virtual-server.yaml",
            option, option_value)
        patch_virtual_server(kube_apis.custom_objects,
                             virtual_server_setup.vs_name, virtual_server_setup.namespace, new_body)
        config_map_name = ingress_controller_prerequisites.config_map["metadata"]["name"]
        replace_configmap_from_yaml(kube_apis.v1, config_map_name,
                                    ingress_controller_prerequisites.namespace,
                                    f"{TEST_DATA}/virtual-server-upstream-options/configmap-with-keys.yaml")
        wait_before_test(1)
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            virtual_server_setup.namespace,
                                            virtual_server_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host})
        resp_2 = requests.get(virtual_server_setup.backend_2_url,
                              headers={"host": virtual_server_setup.vs_host})
        vs_events = get_events(kube_apis.v1, virtual_server_setup.namespace)

        assert_event_count_increased(vs_event_text, initial_count, vs_events)
        for _ in expected_string:
            assert _ in config
        for _ in unexpected_string:
            assert _ not in config
        assert_response_codes(resp_1, resp_2)


@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-upstream-options", "app_type": "simple"})],
                         indirect=True)
class TestVirtualServerUpstreamOptionValidation:
    def test_event_message_and_config(self, kube_apis, ingress_controller_prerequisites,
                                      crd_ingress_controller, virtual_server_setup):
        invalid_fields = ["upstreams[0].lb-method", "upstreams[0].fail-timeout",
                          "upstreams[0].max-fails"]
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"VirtualServer {text} is invalid and was rejected: "
        vs_file = f"{TEST_DATA}/virtual-server-upstream-options/virtual-server-with-invalid-keys.yaml"
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       vs_file,
                                       virtual_server_setup.namespace)
        wait_before_test(2)
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        response = get_vs_nginx_template_conf(kube_apis.v1,
                                              virtual_server_setup.namespace,
                                              virtual_server_setup.vs_name,
                                              ic_pod_name,
                                              ingress_controller_prerequisites.namespace)
        vs_events = get_events(kube_apis.v1, virtual_server_setup.namespace)

        assert_event_starts_with_text_and_contains_errors(vs_event_text, vs_events, invalid_fields)
        assert_template_config_does_not_exist(response)

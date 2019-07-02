import requests
import pytest

from settings import TEST_DATA, DEPLOYMENTS
from suite.custom_resources_utils import get_vs_nginx_template_conf, patch_v_s_route_from_yaml, patch_v_s_route, \
    generate_item_with_upstream_option
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


@pytest.mark.parametrize('crd_ingress_controller, v_s_route_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-route-upstream-options"})],
                         indirect=True)
class TestVSRouteUpstreamOptions:
    def test_nginx_config_upstreams_defaults(self, kube_apis, ingress_controller_prerequisites,
                                             crd_ingress_controller, v_s_route_setup, v_s_route_app_setup):
        print("Case 1: no ConfigMap keys, no options in VS")
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            v_s_route_setup.namespace,
                                            v_s_route_setup.vs_name,
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
    def test_when_option_in_v_s_r_only(self, kube_apis,
                                       ingress_controller_prerequisites,
                                       crd_ingress_controller,
                                       v_s_route_setup, v_s_route_app_setup,
                                       option, option_value, expected_string):
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        text_s = f"{v_s_route_setup.route_s.namespace}/{v_s_route_setup.route_s.name}"
        text_m = f"{v_s_route_setup.route_m.namespace}/{v_s_route_setup.route_m.name}"
        vsr_s_event_text = f"Configuration for {text_s} was added or updated"
        vsr_m_event_text = f"Configuration for {text_m} was added or updated"
        events_ns_m = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_s = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        initial_count_vsr_m = get_event_count(vsr_m_event_text, events_ns_m)
        initial_count_vsr_s = get_event_count(vsr_s_event_text, events_ns_s)
        print(f"Case 2: no {option} in ConfigMap, {option} specified in VSR")
        new_body_m = generate_item_with_upstream_option(
            f"{TEST_DATA}/virtual-server-route-upstream-options/route-multiple.yaml",
            option, option_value)
        new_body_s = generate_item_with_upstream_option(
            f"{TEST_DATA}/virtual-server-route-upstream-options/route-single.yaml",
            option, option_value)
        patch_v_s_route(kube_apis.custom_objects,
                        v_s_route_setup.route_m.name, v_s_route_setup.route_m.namespace, new_body_m)
        patch_v_s_route(kube_apis.custom_objects,
                        v_s_route_setup.route_s.name, v_s_route_setup.route_s.namespace, new_body_s)
        wait_before_test(1)
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            v_s_route_setup.namespace,
                                            v_s_route_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        vsr_s_events = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        vsr_m_events = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)

        assert_event_count_increased(vsr_m_event_text, initial_count_vsr_m, vsr_m_events)
        assert_event_count_increased(vsr_s_event_text, initial_count_vsr_s, vsr_s_events)
        assert expected_string in config
        assert_response_codes(resp_1, resp_2)

    @pytest.mark.parametrize('option, option_value, expected_string, unexpected_string', [
        ("lb-method", "round_robin", [], ["ip_hash;", "least_conn;", "random ", "hash", "least_time "]),
        ("max-fails", "28", ["max_fails=28 "], ["max_fails=1 "]),
        ("fail-timeout", "23s", ["fail_timeout=23s;"], ["fail_timeout=10s;"])
    ])
    def test_when_option_in_config_map_only(self, kube_apis,
                                            ingress_controller_prerequisites,
                                            crd_ingress_controller,
                                            v_s_route_setup, v_s_route_app_setup,
                                            option, option_value, expected_string, unexpected_string):
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        text_s = f"{v_s_route_setup.route_s.namespace}/{v_s_route_setup.route_s.name}"
        text_m = f"{v_s_route_setup.route_m.namespace}/{v_s_route_setup.route_m.name}"
        vsr_s_event_text = f"Configuration for {text_s} was updated"
        vsr_m_event_text = f"Configuration for {text_m} was updated"
        print(f"Case 3: {option} specified in ConfigMap, no {option} in VS")
        patch_v_s_route_from_yaml(kube_apis.custom_objects, v_s_route_setup.route_m.name,
                                  f"{TEST_DATA}/virtual-server-route-upstream-options/route-multiple.yaml",
                                  v_s_route_setup.route_m.namespace)
        patch_v_s_route_from_yaml(kube_apis.custom_objects, v_s_route_setup.route_s.name,
                                  f"{TEST_DATA}/virtual-server-route-upstream-options/route-single.yaml",
                                  v_s_route_setup.route_s.namespace)
        config_map_name = ingress_controller_prerequisites.config_map["metadata"]["name"]
        new_configmap = create_configmap_from_yaml_with_overriden_key(
            f"{DEPLOYMENTS}/common/nginx-config.yaml", option, option_value)
        replace_configmap(kube_apis.v1, config_map_name,
                          ingress_controller_prerequisites.namespace,
                          new_configmap)
        wait_before_test(1)
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            v_s_route_setup.namespace,
                                            v_s_route_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        vsr_s_events = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        vsr_m_events = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)

        assert_event(vsr_m_event_text, vsr_m_events)
        assert_event(vsr_s_event_text, vsr_s_events)
        for _ in expected_string:
            assert _ in config
        for _ in unexpected_string:
            assert _ not in config
        assert_response_codes(resp_1, resp_2)

    @pytest.mark.parametrize('option, option_value, expected_string, unexpected_string', [
        ("lb-method", "least_conn", ["least_conn;"], ["ip_hash;", "random ", "hash", "least_time "]),
        ("max-fails", 12, ["max_fails=12 "], ["max_fails=1 "]),
        ("fail-timeout", "1m", ["fail_timeout=1m;"], ["fail_timeout=10s;"])
    ])
    def test_v_s_r_overrides_config_map(self, kube_apis,
                                        ingress_controller_prerequisites,
                                        crd_ingress_controller, v_s_route_setup, v_s_route_app_setup,
                                        option, option_value, expected_string, unexpected_string):
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        text_s = f"{v_s_route_setup.route_s.namespace}/{v_s_route_setup.route_s.name}"
        text_m = f"{v_s_route_setup.route_m.namespace}/{v_s_route_setup.route_m.name}"
        vsr_s_event_text = f"Configuration for {text_s} was added or updated"
        vsr_m_event_text = f"Configuration for {text_m} was added or updated"
        events_ns_m = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_s = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        initial_count_vsr_m = get_event_count(vsr_m_event_text, events_ns_m)
        initial_count_vsr_s = get_event_count(vsr_s_event_text, events_ns_s)
        print(f"Case 4: {option} specified in ConfigMap, {option} specified in VS")
        new_body_m = generate_item_with_upstream_option(
            f"{TEST_DATA}/virtual-server-route-upstream-options/route-multiple.yaml",
            option, option_value)
        new_body_s = generate_item_with_upstream_option(
            f"{TEST_DATA}/virtual-server-route-upstream-options/route-single.yaml",
            option, option_value)
        patch_v_s_route(kube_apis.custom_objects,
                        v_s_route_setup.route_m.name, v_s_route_setup.route_m.namespace, new_body_m)
        patch_v_s_route(kube_apis.custom_objects,
                        v_s_route_setup.route_s.name, v_s_route_setup.route_s.namespace, new_body_s)
        config_map_name = ingress_controller_prerequisites.config_map["metadata"]["name"]
        replace_configmap_from_yaml(kube_apis.v1, config_map_name,
                                    ingress_controller_prerequisites.namespace,
                                    f"{TEST_DATA}/virtual-server-route-upstream-options/configmap-with-keys.yaml")
        wait_before_test(1)
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            v_s_route_setup.namespace,
                                            v_s_route_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        vsr_s_events = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        vsr_m_events = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)

        assert_event_count_increased(vsr_m_event_text, initial_count_vsr_m, vsr_m_events)
        assert_event_count_increased(vsr_s_event_text, initial_count_vsr_s, vsr_s_events)
        for _ in expected_string:
            assert _ in config
        for _ in unexpected_string:
            assert _ not in config
        assert_response_codes(resp_1, resp_2)


@pytest.mark.parametrize('crd_ingress_controller, v_s_route_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-route-upstream-options"})],
                         indirect=True)
class TestVSRouteUpstreamOptionsValidation:
    def test_event_message_and_config(self, kube_apis, ingress_controller_prerequisites,
                                      crd_ingress_controller, v_s_route_setup):
        invalid_fields_s = ["upstreams[0].lb-method", "upstreams[0].fail-timeout",
                            "upstreams[0].max-fails"]
        invalid_fields_m = ["upstreams[0].lb-method", "upstreams[0].fail-timeout",
                            "upstreams[0].max-fails",
                            "upstreams[1].lb-method", "upstreams[1].fail-timeout",
                            "upstreams[1].max-fails"]
        text_s = f"{v_s_route_setup.route_s.namespace}/{v_s_route_setup.route_s.name}"
        text_m = f"{v_s_route_setup.route_m.namespace}/{v_s_route_setup.route_m.name}"
        vsr_s_event_text = f"VirtualServerRoute {text_s} is invalid and was rejected: "
        vsr_m_event_text = f"VirtualServerRoute {text_m} is invalid and was rejected: "
        patch_v_s_route_from_yaml(kube_apis.custom_objects,
                                  v_s_route_setup.route_s.name,
                                  f"{TEST_DATA}/virtual-server-route-upstream-options/route-single-invalid-keys.yaml",
                                  v_s_route_setup.route_s.namespace)
        patch_v_s_route_from_yaml(kube_apis.custom_objects,
                                  v_s_route_setup.route_m.name,
                                  f"{TEST_DATA}/virtual-server-route-upstream-options/route-multiple-invalid-keys.yaml",
                                  v_s_route_setup.route_m.namespace)
        wait_before_test(2)
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            v_s_route_setup.namespace,
                                            v_s_route_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        vsr_s_events = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        vsr_m_events = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)

        assert_event_starts_with_text_and_contains_errors(vsr_s_event_text, vsr_s_events, invalid_fields_s)
        assert_event_starts_with_text_and_contains_errors(vsr_m_event_text, vsr_m_events, invalid_fields_m)
        assert "upstream" not in config

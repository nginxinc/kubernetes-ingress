import requests
import pytest

from settings import TEST_DATA
from suite.custom_resources_utils import get_vs_nginx_template_conf, patch_virtual_server_from_yaml, \
    patch_virtual_server, generate_item_with_upstream_options
from suite.resources_utils import get_first_pod_name, wait_before_test, replace_configmap_from_yaml, get_events


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

        assert "proxy_connect_timeout 60s;" in config
        assert "proxy_read_timeout 60s;" in config
        assert "proxy_send_timeout 60s;" in config

        assert "max_fails=1 fail_timeout=10s max_conns=0;" in config

        assert "keepalive" not in config
        assert 'proxy_set_header Connection "";' not in config

    @pytest.mark.parametrize('options, expected_strings', [
        ({"lb-method": "least_conn", "max-fails": 8,
          "fail-timeout": "13s", "connect-timeout": "55s", "read-timeout": "1s", "send-timeout": "1h",
          "keepalive": 54},
         ["least_conn;", "max_fails=8 ",
          "fail_timeout=13s ", "max_conns=0;", "proxy_connect_timeout 55s;", "proxy_read_timeout 1s;", "proxy_send_timeout 1h;",
          "keepalive 54;", 'proxy_set_header Connection "";']),
        ({"lb-method": "ip_hash", "connect-timeout": "75", "read-timeout": "15", "send-timeout": "1h"},
         ["ip_hash;", "proxy_connect_timeout 75;", "proxy_read_timeout 15;", "proxy_send_timeout 1h;"]),
        ({"connect-timeout": "1m", "read-timeout": "1m", "send-timeout": "1s"},
         ["proxy_connect_timeout 1m;", "proxy_read_timeout 1m;", "proxy_send_timeout 1s;"], )
    ])
    def test_when_option_in_v_s_only(self, kube_apis, ingress_controller_prerequisites,
                                     crd_ingress_controller, virtual_server_setup,
                                     options, expected_strings):
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"Configuration for {text} was added or updated"
        events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)
        initial_count = get_event_count(vs_event_text, events_vs)
        print(f"Case 2: no key in ConfigMap , option specified in VS")
        new_body = generate_item_with_upstream_options(
            f"{TEST_DATA}/virtual-server-upstream-options/standard/virtual-server.yaml",
            options)
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
        for _ in expected_strings:
            assert _ in config
        assert_response_codes(resp_1, resp_2)

    @pytest.mark.parametrize('config_map_file, expected_strings, unexpected_strings', [
        (f"{TEST_DATA}/virtual-server-upstream-options/configmap-with-keys.yaml",
         ["max_fails=3 ", "fail_timeout=33s ", "max_conns=0;",
          "proxy_connect_timeout 44s;", "proxy_read_timeout 22s;", "proxy_send_timeout 55s;",
          "keepalive 1024;", 'proxy_set_header Connection "";'],
         ["ip_hash;", "least_conn;", "random ", "hash", "least_time ",
          "max_fails=1 ", "fail_timeout=10s ", "max_conns=1s;",
          "proxy_connect_timeout 60s;", "proxy_read_timeout 60s;", "proxy_send_timeout 60s;"]),
    ])
    def test_when_option_in_config_map_only(self, kube_apis, ingress_controller_prerequisites,
                                            crd_ingress_controller, virtual_server_setup,
                                            config_map_file, expected_strings, unexpected_strings):
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"Configuration for {text} was updated"
        print(f"Case 3: key specified in ConfigMap, no option in VS")
        patch_virtual_server_from_yaml(kube_apis.custom_objects, virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-upstream-options/standard/virtual-server.yaml",
                                       virtual_server_setup.namespace)
        config_map_name = ingress_controller_prerequisites.config_map["metadata"]["name"]
        replace_configmap_from_yaml(kube_apis.v1, config_map_name,
                                    ingress_controller_prerequisites.namespace,
                                    config_map_file)
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
        for _ in expected_strings:
            assert _ in config
        for _ in unexpected_strings:
            assert _ not in config
        assert_response_codes(resp_1, resp_2)

    @pytest.mark.parametrize('options, expected_strings, unexpected_strings', [
        ({"lb-method": "least_conn", "max-fails": 12,
          "fail-timeout": "1m", "connect-timeout": "1m", "read-timeout": "77s", "send-timeout": "23s",
          "keepalive": 48},
         ["least_conn;", "max_fails=12 ",
          "fail_timeout=1m ", "max_conns=0;", "proxy_connect_timeout 1m;", "proxy_read_timeout 77s;", "proxy_send_timeout 23s;",
          "keepalive 48;", 'proxy_set_header Connection "";'],
         ["ip_hash;", "random ", "hash", "least_time ", "max_fails=1 ",
          "fail_timeout=10s ", "max_conns=33s;", "proxy_connect_timeout 44s;", "proxy_read_timeout 22s;", "proxy_send_timeout 55s;",
          "keepalive 1024;"])
    ])
    def test_v_s_overrides_config_map(self, kube_apis, ingress_controller_prerequisites,
                                      crd_ingress_controller, virtual_server_setup,
                                      options, expected_strings, unexpected_strings):
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"Configuration for {text} was added or updated"
        events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)
        initial_count = get_event_count(vs_event_text, events_vs)
        print(f"Case 4: key in ConfigMap, option specified in VS")
        new_body = generate_item_with_upstream_options(
            f"{TEST_DATA}/virtual-server-upstream-options/standard/virtual-server.yaml",
            options)
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
        for _ in expected_strings:
            assert _ in config
        for _ in unexpected_strings:
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
                          "upstreams[0].max-fails", "upstreams[0].connect-timeout",
                          "upstreams[0].read-timeout", "upstreams[0].send-timeout",
                          "upstreams[0].keepalive"]
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

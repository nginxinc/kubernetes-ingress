import requests
import pytest

from settings import TEST_DATA
from suite.custom_resources_utils import get_vs_nginx_template_conf, patch_virtual_server_from_yaml
from suite.resources_utils import get_first_pod_name, wait_before_test, get_events


def assert_response_codes(resp_1, resp_2, code_1=200, code_2=200):
    assert resp_1.status_code == code_1
    assert resp_2.status_code == code_2


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


def assert_no_new_events(old_list, new_list):
    assert len(old_list) == len(new_list), "expected: lists are the same"
    for i in range(len(new_list) - 1, -1, -1):
        if old_list[i].count != new_list[i].count:
            pytest.fail(f"Expected: no new events. There is a new event found:\"{new_list[i].message}\". Exiting...")


@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-upstream-tls", "app_type": "secure"})],
                         indirect=True)
class TestVirtualServerUpstreamTls:
    def test_responses_and_config_after_setup(self, kube_apis, ingress_controller_prerequisites,
                                              crd_ingress_controller, virtual_server_setup):
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

        proxy_host = f"vs_{virtual_server_setup.namespace}_{virtual_server_setup.vs_name}"
        assert f'proxy_pass https://{proxy_host}_backend1' not in config
        assert f'proxy_pass https://{proxy_host}_backend2' in config
        assert_response_codes(resp_1, resp_2)

    def test_event_after_setup(self, kube_apis, ingress_controller_prerequisites,
                               crd_ingress_controller, virtual_server_setup):
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"Configuration for {text} was added or updated"
        events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_event(vs_event_text, events_vs)

    def test_invalid_value_rejection(self, kube_apis, ingress_controller_prerequisites,
                                     crd_ingress_controller, virtual_server_setup):
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        initial_events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-upstream-tls/virtual-server-invalid.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            virtual_server_setup.namespace,
                                            virtual_server_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host})
        resp_2 = requests.get(virtual_server_setup.backend_2_url,
                              headers={"host": virtual_server_setup.vs_host})
        new_events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)

        proxy_host = f"vs_{virtual_server_setup.namespace}_{virtual_server_setup.vs_name}"
        assert f'proxy_pass https://{proxy_host}_backend1' not in config
        assert f'proxy_pass https://{proxy_host}_backend2' in config
        assert_response_codes(resp_1, resp_2)
        assert_no_new_events(initial_events_vs, new_events_vs)

    def test_responses_and_config_after_disable_tls(self, kube_apis, ingress_controller_prerequisites,
                                                    crd_ingress_controller, virtual_server_setup):
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"Configuration for {text} was added or updated"
        initial_events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)
        initial_count = get_event_count(vs_event_text, initial_events_vs)
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-upstream-tls/virtual-server-disable-tls.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            virtual_server_setup.namespace,
                                            virtual_server_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host})
        resp_2 = requests.get(virtual_server_setup.backend_2_url,
                              headers={"host": virtual_server_setup.vs_host})
        new_events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)

        assert 'proxy_pass https://' not in config
        assert_response_codes(resp_1, resp_2, 200, 400)
        assert_event_count_increased(vs_event_text, initial_count, new_events_vs)

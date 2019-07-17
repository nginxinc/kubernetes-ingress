import requests
import pytest

from settings import TEST_DATA
from suite.custom_resources_utils import get_vs_nginx_template_conf, patch_v_s_route_from_yaml
from suite.resources_utils import create_items_from_yaml, get_first_pod_name, \
    delete_items_from_yaml, wait_until_all_pods_are_ready, wait_before_test, get_events


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


@pytest.fixture(scope="class")
def v_s_route_secure_app_setup(request, kube_apis, v_s_route_setup) -> None:
    """
    Prepare a secure example app for Virtual Server Route.

    1st namespace with backend1-svc and backend3-svc and deployment
    and 2nd namespace with https backend2-svc and deployment.

    :param request: internal pytest fixture
    :param kube_apis: client apis
    :param v_s_route_setup:
    :return:
    """
    print("---------------------- Deploy a VS Route Example Application ----------------------------")
    create_items_from_yaml(kube_apis,
                           f"{TEST_DATA}/common/app/vsr/secure/multiple.yaml", v_s_route_setup.route_m.namespace)

    create_items_from_yaml(kube_apis,
                           f"{TEST_DATA}/common/app/vsr/secure/single.yaml", v_s_route_setup.route_s.namespace)

    wait_until_all_pods_are_ready(kube_apis.v1, v_s_route_setup.route_m.namespace)
    wait_until_all_pods_are_ready(kube_apis.v1, v_s_route_setup.route_s.namespace)

    def fin():
        print("Clean up the Application:")
        delete_items_from_yaml(kube_apis,
                               f"{TEST_DATA}/common/app/vsr/secure/multiple.yaml",
                               v_s_route_setup.route_m.namespace)
        delete_items_from_yaml(kube_apis,
                               f"{TEST_DATA}/common/app/vsr/secure/single.yaml",
                               v_s_route_setup.route_s.namespace)

    request.addfinalizer(fin)


@pytest.mark.parametrize('crd_ingress_controller, v_s_route_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-route-upstream-tls"})],
                         indirect=True)
class TestVSRouteUpstreamOptions:
    def test_responses_and_config_after_setup(self, kube_apis, ingress_controller_prerequisites,
                                              crd_ingress_controller, v_s_route_setup, v_s_route_secure_app_setup):
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            v_s_route_setup.namespace,
                                            v_s_route_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        vs_line = f"vs_{v_s_route_setup.namespace}_{v_s_route_setup.vs_name}"
        proxy_host_s = f"{vs_line}_vsr_{v_s_route_setup.route_s.namespace}_{v_s_route_setup.route_s.name}"
        proxy_host_m = f"{vs_line}_vsr_{v_s_route_setup.route_m.namespace}_{v_s_route_setup.route_m.name}"
        assert f'proxy_pass https://{proxy_host_m}' not in config
        assert f'proxy_pass https://{proxy_host_s}' in config
        assert_response_codes(resp_1, resp_2)

    def test_events_after_setup(self, kube_apis, ingress_controller_prerequisites,
                               crd_ingress_controller, v_s_route_setup, v_s_route_secure_app_setup):
        text_s = f"{v_s_route_setup.route_s.namespace}/{v_s_route_setup.route_s.name}"
        text_m = f"{v_s_route_setup.route_m.namespace}/{v_s_route_setup.route_m.name}"
        text_vs = f"{v_s_route_setup.namespace}/{v_s_route_setup.vs_name}"
        vsr_s_event_text = f"Configuration for {text_s} was added or updated"
        vsr_m_event_text = f"Configuration for {text_m} was added or updated"
        vs_event_text = f"Configuration for {text_vs} was added or updated"
        events_ns_m = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_s = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_event(vsr_s_event_text, events_ns_s)
        assert_event(vsr_m_event_text, events_ns_m)
        assert_event(vs_event_text, events_ns_m)

    def test_invalid_value_rejection(self, kube_apis, ingress_controller_prerequisites,
                                     crd_ingress_controller,
                                     v_s_route_setup, v_s_route_secure_app_setup):
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        initial_events_ns_m = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        initial_events_ns_s = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        patch_v_s_route_from_yaml(kube_apis.custom_objects,
                                  v_s_route_setup.route_s.name,
                                  f"{TEST_DATA}/virtual-server-route-upstream-tls/route-single-invalid.yaml",
                                  v_s_route_setup.route_s.namespace)
        wait_before_test(1)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            v_s_route_setup.namespace,
                                            v_s_route_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        new_events_ns_m = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        new_events_ns_s = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)

        vs_line = f"vs_{v_s_route_setup.namespace}_{v_s_route_setup.vs_name}"
        proxy_host_s = f"{vs_line}_vsr_{v_s_route_setup.route_s.namespace}_{v_s_route_setup.route_s.name}"
        proxy_host_m = f"{vs_line}_vsr_{v_s_route_setup.route_m.namespace}_{v_s_route_setup.route_m.name}"
        assert f'proxy_pass https://{proxy_host_m}' not in config
        assert f'proxy_pass https://{proxy_host_s}' in config
        assert_response_codes(resp_1, resp_2)
        assert_no_new_events(initial_events_ns_m, new_events_ns_m)
        assert_no_new_events(initial_events_ns_s, new_events_ns_s)

    def test_responses_and_config_after_disable_tls(self, kube_apis, ingress_controller_prerequisites,
                                                    crd_ingress_controller,
                                                    v_s_route_setup, v_s_route_secure_app_setup):
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        text_s = f"{v_s_route_setup.route_s.namespace}/{v_s_route_setup.route_s.name}"
        text_m = f"{v_s_route_setup.route_m.namespace}/{v_s_route_setup.route_m.name}"
        text_vs = f"{v_s_route_setup.namespace}/{v_s_route_setup.vs_name}"
        vsr_s_event_text = f"Configuration for {text_s} was added or updated"
        vsr_m_event_text = f"Configuration for {text_m} was added or updated"
        vs_event_text = f"Configuration for {text_vs} was added or updated"
        initial_events_ns_m = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        initial_events_ns_s = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        initial_count_vsr_m = get_event_count(vsr_m_event_text, initial_events_ns_m)
        initial_count_vsr_s = get_event_count(vsr_s_event_text, initial_events_ns_s)
        initial_count_vs = get_event_count(vs_event_text, initial_events_ns_m)
        patch_v_s_route_from_yaml(kube_apis.custom_objects,
                                  v_s_route_setup.route_s.name,
                                  f"{TEST_DATA}/virtual-server-route-upstream-tls/route-single-disable-tls.yaml",
                                  v_s_route_setup.route_s.namespace)
        wait_before_test(1)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            v_s_route_setup.namespace,
                                            v_s_route_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        new_events_ns_m = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        new_events_ns_s = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)

        assert 'proxy_pass https://' not in config
        assert_response_codes(resp_1, resp_2, 200, 400)
        assert_event_count_increased(vsr_m_event_text, initial_count_vsr_m, new_events_ns_m)
        assert_event_count_increased(vs_event_text, initial_count_vs, new_events_ns_m)
        assert_event_count_increased(vsr_s_event_text, initial_count_vsr_s, new_events_ns_s)

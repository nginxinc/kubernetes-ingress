import pytest
import requests

from settings import TEST_DATA
from suite.custom_assertions import assert_vs_conf_not_exists, assert_vs_conf_exists
from suite.vs_vsr_resources_utils import patch_virtual_server_from_yaml,\
    delete_virtual_server, create_virtual_server_from_yaml
from suite.resources_utils import wait_before_test, get_events, get_first_pod_name, get_pods_amount


def assert_reject_events_emitted(virtual_server_setup, new_list, previous_list, expected_amount):
    item_name = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
    text_invalid = f"VirtualServer {item_name} was rejected with error"
    new_event = new_list[len(new_list) - 1]
    assert len(new_list) - len(previous_list) == expected_amount
    assert text_invalid in new_event.message


def assert_event_count_increased_in_list(virtual_server_setup, new_list, previous_list):
    item_name = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
    text_valid = f"Configuration for {item_name} was added or updated"
    for i in range(len(previous_list)-1, 0, -1):
        if text_valid in previous_list[i].message:
            assert new_list[i].count - previous_list[i].count == 1, "We expect the counter to increase"


def assert_response_200(virtual_server_setup):
    resp = requests.get(virtual_server_setup.backend_1_url, headers={"host": virtual_server_setup.vs_host})
    assert resp.status_code == 200
    resp = requests.get(virtual_server_setup.backend_2_url, headers={"host": virtual_server_setup.vs_host})
    assert resp.status_code == 200


def assert_response_404(virtual_server_setup):
    resp = requests.get(virtual_server_setup.backend_1_url, headers={"host": virtual_server_setup.vs_host})
    assert resp.status_code == 404
    resp = requests.get(virtual_server_setup.backend_2_url, headers={"host": virtual_server_setup.vs_host})
    assert resp.status_code == 404


@pytest.mark.vs
@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-validation", "app_type": "advanced-routing"})],
                         indirect=True)
class TestVirtualServerValidation:
    def test_virtual_server_behavior(self,
                                     kube_apis, cli_arguments,
                                     ingress_controller_prerequisites,
                                     crd_ingress_controller,
                                     virtual_server_setup):
        ic_pods_amount = get_pods_amount(kube_apis.v1, ingress_controller_prerequisites.namespace)
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)

        print("Step 1: initial check")
        step_1_list = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_vs_conf_exists(kube_apis, ic_pod_name, ingress_controller_prerequisites.namespace,
                              virtual_server_setup)
        assert_response_200(virtual_server_setup)

        print("Step 2: make a valid VirtualServer invalid and check")
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-validation/virtual-server-invalid-cookie.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)
        step_2_list = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_reject_events_emitted(virtual_server_setup, step_2_list, step_1_list, ic_pods_amount)
        assert_vs_conf_not_exists(kube_apis, ic_pod_name, ingress_controller_prerequisites.namespace,
                                  virtual_server_setup)
        assert_response_404(virtual_server_setup)

        print("Step 3: update an invalid VirtualServer with another invalid and check")
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-validation/virtual-server-no-default-action.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)
        step_3_list = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_reject_events_emitted(virtual_server_setup, step_3_list, step_2_list, ic_pods_amount)
        assert_vs_conf_not_exists(kube_apis, ic_pod_name, ingress_controller_prerequisites.namespace,
                                  virtual_server_setup)
        assert_response_404(virtual_server_setup)

        print("Step 4: make an invalid VirtualServer valid and check")
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-validation/standard/virtual-server.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)
        step_4_list = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_vs_conf_exists(kube_apis, ic_pod_name, ingress_controller_prerequisites.namespace,
                              virtual_server_setup)
        assert_event_count_increased_in_list(virtual_server_setup, step_4_list, step_3_list)
        assert_response_200(virtual_server_setup)

        print("Step 5: delete VS and then create an invalid and check")
        delete_virtual_server(kube_apis.custom_objects, virtual_server_setup.vs_name, virtual_server_setup.namespace)
        create_virtual_server_from_yaml(kube_apis.custom_objects,
                                        f"{TEST_DATA}/virtual-server-validation/virtual-server-invalid-cookie.yaml",
                                        virtual_server_setup.namespace)
        wait_before_test(1)
        step_5_list = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_reject_events_emitted(virtual_server_setup, step_5_list, step_4_list, ic_pods_amount)
        assert_vs_conf_not_exists(kube_apis, ic_pod_name, ingress_controller_prerequisites.namespace,
                                  virtual_server_setup)
        assert_response_404(virtual_server_setup)

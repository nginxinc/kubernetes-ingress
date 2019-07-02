import requests
import pytest

from settings import TEST_DATA
from suite.custom_resources_utils import create_virtual_server_from_yaml, \
    delete_virtual_server, create_v_s_route_from_yaml, delete_v_s_route, get_vs_nginx_template_conf, \
    patch_v_s_route_from_yaml
from suite.fixtures import PublicEndpoint
from suite.resources_utils import wait_until_all_pods_are_ready, \
    create_namespace_with_name_from_yaml, delete_namespace, \
    delete_deployment, delete_service, get_first_pod_name, get_events, \
    wait_before_test, read_service, replace_service,\
    create_service_with_name, create_deployment_with_name
from suite.yaml_utils import get_first_vs_host_from_yaml, \
    get_route_namespace_from_vs_yaml, get_paths_from_vsr_yaml


@pytest.fixture(scope="class")
def example_app_setup(request, kube_apis, v_s_route_setup) -> None:
    """
    Prepare an example app.

    1st namespace with backend1-svc and backend3-svc and deployment and 2nd namespace with backend2-svc and deployment.

    :param request: internal pytest fixture
    :param kube_apis: client apis
    :param v_s_route_setup:
    :return:
    """
    print("---------------------- Deploy a VS Route Example Application ----------------------------")
    svc_one = create_service_with_name(kube_apis.v1, v_s_route_setup.route_m.namespace, "backend1-svc")
    svc_three = create_service_with_name(kube_apis.v1, v_s_route_setup.route_m.namespace, "backend3-svc")
    deployment_one = create_deployment_with_name(kube_apis.apps_v1_api,
                                                 v_s_route_setup.route_m.namespace,
                                                 "backend1")
    deployment_three = create_deployment_with_name(kube_apis.apps_v1_api,
                                                   v_s_route_setup.route_m.namespace,
                                                   "backend3")

    svc_two = create_service_with_name(kube_apis.v1, v_s_route_setup.route_s.namespace, "backend2-svc")
    deployment_two = create_deployment_with_name(kube_apis.apps_v1_api,
                                                 v_s_route_setup.route_s.namespace, "backend2")

    wait_until_all_pods_are_ready(kube_apis.v1, v_s_route_setup.route_m.namespace)
    wait_until_all_pods_are_ready(kube_apis.v1, v_s_route_setup.route_s.namespace)
    # wait_before_test(2)

    def fin():
        print("Clean up the Application:")
        delete_deployment(kube_apis.apps_v1_api, deployment_one, v_s_route_setup.route_m.namespace)
        delete_service(kube_apis.v1, svc_one, v_s_route_setup.route_m.namespace)
        delete_deployment(kube_apis.apps_v1_api, deployment_three, v_s_route_setup.route_m.namespace)
        delete_service(kube_apis.v1, svc_three, v_s_route_setup.route_m.namespace)
        delete_deployment(kube_apis.apps_v1_api, deployment_two, v_s_route_setup.route_s.namespace)
        delete_service(kube_apis.v1, svc_two, v_s_route_setup.route_s.namespace)

    request.addfinalizer(fin)


class VirtualServerRoute:
    """
    Encapsulate  Virtual Server Route details.

    Attributes:
        namespace (str):
        name (str):
        paths ([]):
    """

    def __init__(self, namespace, name, paths):
        self.namespace = namespace
        self.name = name
        self.paths = paths


class VirtualServerRouteSetup:
    """
    Encapsulate Virtual Server Example details.

    Attributes:
        public_endpoint (PublicEndpoint):
        namespace (str):
        vs_host (str):
        vs_name (str):
        route_m (VirtualServerRoute): route with multiple subroutes
        route_s (VirtualServerRoute): route with single subroute
    """

    def __init__(self, public_endpoint: PublicEndpoint,
                 namespace, vs_host, vs_name, route_m: VirtualServerRoute, route_s: VirtualServerRoute):
        self.public_endpoint = public_endpoint
        self.namespace = namespace
        self.vs_host = vs_host
        self.vs_name = vs_name
        self.route_m = route_m
        self.route_s = route_s


@pytest.fixture(scope="class")
def v_s_route_setup(request,
                    kube_apis,
                    crd_ingress_controller,
                    ingress_controller_endpoint) -> VirtualServerRouteSetup:
    """
    Prepare Virtual Server Route Example.

    1st namespace with VS and 1st addressed VSR and 2nd namespace with second addressed VSR.

    :param request: internal pytest fixture
    :param kube_apis: client apis
    :param crd_ingress_controller:
    :param ingress_controller_endpoint:

    :return: VirtualServerRouteSetup
    """
    vs_routes_ns = get_route_namespace_from_vs_yaml(f"{TEST_DATA}/virtual-server-route/standard/virtual-server.yaml")
    ns_1 = create_namespace_with_name_from_yaml(kube_apis.v1,
                                                vs_routes_ns[0],
                                                f"{TEST_DATA}/common/ns.yaml")
    ns_2 = create_namespace_with_name_from_yaml(kube_apis.v1,
                                                vs_routes_ns[1],
                                                f"{TEST_DATA}/common/ns.yaml")
    print("------------------------- Deploy Virtual Server -----------------------------------")
    vs_name = create_virtual_server_from_yaml(kube_apis.custom_objects,
                                              f"{TEST_DATA}/virtual-server-route/standard/virtual-server.yaml",
                                              ns_1)
    vs_host = get_first_vs_host_from_yaml(f"{TEST_DATA}/virtual-server-route/standard/virtual-server.yaml")

    print("------------------------- Deploy Virtual Server Route -----------------------------------")
    vsr_m_name = create_v_s_route_from_yaml(kube_apis.custom_objects,
                                            f"{TEST_DATA}/virtual-server-route/route-multiple.yaml",
                                            ns_1)
    vsr_m_paths = get_paths_from_vsr_yaml(f"{TEST_DATA}/virtual-server-route/route-multiple.yaml")
    route_m = VirtualServerRoute(ns_1, vsr_m_name, vsr_m_paths)

    vsr_s_name = create_v_s_route_from_yaml(kube_apis.custom_objects,
                                            f"{TEST_DATA}/virtual-server-route/route-single.yaml",
                                            ns_2)
    vsr_s_paths = get_paths_from_vsr_yaml(f"{TEST_DATA}/virtual-server-route/route-single.yaml")
    route_s = VirtualServerRoute(ns_2, vsr_s_name, vsr_s_paths)

    def fin():
        print("Clean up the Virtual Server Route:")
        delete_v_s_route(kube_apis.custom_objects, vsr_m_name, ns_1)
        delete_v_s_route(kube_apis.custom_objects, vsr_s_name, ns_2)
        print("Clean up Virtual Server:")
        delete_virtual_server(kube_apis.custom_objects, vs_name, ns_1)
        print("Delete test namespaces")
        delete_namespace(kube_apis.v1, ns_1)
        delete_namespace(kube_apis.v1, ns_2)

    request.addfinalizer(fin)

    return VirtualServerRouteSetup(ingress_controller_endpoint,
                                   ns_1, vs_host, vs_name, route_m, route_s)


def assert_responses_and_server_name(resp_1, resp_2, resp_3):
    assert resp_1.status_code == 200
    assert "Server name: backend1-" in resp_1.text
    assert resp_2.status_code == 200
    assert "Server name: backend3-" in resp_2.text
    assert resp_3.status_code == 200
    assert "Server name: backend2-" in resp_3.text


def assert_locations_in_config(config, paths):
    for path in paths:
        assert f"location {path}" in config


def assert_locations_not_in_config(config, paths):
    assert "No such file or directory" not in config
    for path in paths:
        assert f"location {path}" not in config


def assert_event_and_count(event_text, count, events_list):
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            assert events_list[i].count == count
            return
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


def assert_event_and_get_count(event_text, events_list) -> int:
    for i in range(len(events_list) - 1, -1, -1):
        if event_text in events_list[i].message:
            return events_list[i].count
    pytest.fail(f"Failed to find the event \"{event_text}\" in the list. Exiting...")


@pytest.mark.parametrize('crd_ingress_controller',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]})],
                         indirect=True)
class TestVirtualServerRoute:
    def test_responses_and_events_in_flow(self, kube_apis,
                                          ingress_controller_prerequisites,
                                          crd_ingress_controller,
                                          v_s_route_setup,
                                          example_app_setup):
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vs_name = f"{v_s_route_setup.namespace}/{v_s_route_setup.vs_name}"
        vsr_1_name = f"{v_s_route_setup.namespace}/{v_s_route_setup.route_m.name}"
        vsr_2_name = f"{v_s_route_setup.route_s.namespace}/{v_s_route_setup.route_s.name}"
        vsr_1_event_text = f"Configuration for {vsr_1_name} was added or updated"
        vs_event_text = f"Configuration for {vs_name} was added or updated"
        vsr_2_event_text = f"Configuration for {vsr_2_name} was added or updated"
        initial_config = get_vs_nginx_template_conf(kube_apis.v1,
                                                    v_s_route_setup.namespace,
                                                    v_s_route_setup.vs_name,
                                                    ic_pod_name,
                                                    ingress_controller_prerequisites.namespace)

        print("\nStep 1: initial check")
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        events_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_responses_and_server_name(resp_1, resp_2, resp_3)
        assert_locations_in_config(initial_config, v_s_route_setup.route_m.paths)
        assert_locations_in_config(initial_config, v_s_route_setup.route_s.paths)
        initial_count_vsr_1 = assert_event_and_get_count(vsr_1_event_text, events_ns_1)
        initial_count_vs = assert_event_and_get_count(vs_event_text, events_ns_1)
        initial_count_vsr_2 = assert_event_and_get_count(vsr_2_event_text, events_ns_2)

        print("\nStep 2: update multiple VSRoute and check")
        patch_v_s_route_from_yaml(kube_apis.custom_objects,
                                  v_s_route_setup.route_m.name,
                                  f"{TEST_DATA}/virtual-server-route/route-multiple-updated.yaml",
                                  v_s_route_setup.route_m.namespace)
        new_vsr_paths = get_paths_from_vsr_yaml(f"{TEST_DATA}/virtual-server-route/route-multiple-updated.yaml")
        wait_before_test(1)
        resp_1 = requests.get(f"{req_url}{new_vsr_paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{new_vsr_paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert_responses_and_server_name(resp_1, resp_2, resp_3)
        events_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_event_and_count(vsr_1_event_text, initial_count_vsr_1 + 1, events_ns_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 1, events_ns_1)
        # 2nd VSRoute gets an event about update too
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 1, events_ns_2)

        print("\nStep 3: restore VSRoute and check")
        patch_v_s_route_from_yaml(kube_apis.custom_objects,
                                  v_s_route_setup.route_m.name,
                                  f"{TEST_DATA}/virtual-server-route/route-multiple.yaml",
                                  v_s_route_setup.route_m.namespace)
        wait_before_test(1)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert_responses_and_server_name(resp_1, resp_2, resp_3)
        events_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_event_and_count(vsr_1_event_text, initial_count_vsr_1 + 2, events_ns_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 2, events_ns_1)
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 2, events_ns_2)

        print("\nStep 4: update one backend service port and check")
        svc_1 = read_service(kube_apis.v1, "backend1-svc", v_s_route_setup.route_m.namespace)
        svc_1.spec.ports[0].port = 8080
        replace_service(kube_apis.v1, "backend1-svc", v_s_route_setup.route_m.namespace, svc_1)
        wait_before_test(1)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert resp_1.status_code == 502
        assert resp_2.status_code == 200
        events_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_event_and_count(vsr_1_event_text, initial_count_vsr_1 + 3, events_ns_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 3, events_ns_1)
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 3, events_ns_2)

        print("\nStep 5: restore backend service and check")
        svc_1 = read_service(kube_apis.v1, "backend1-svc", v_s_route_setup.route_m.namespace)
        svc_1.spec.ports[0].port = 80
        replace_service(kube_apis.v1, "backend1-svc", v_s_route_setup.route_m.namespace, svc_1)
        wait_before_test(1)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert resp_1.status_code == 200
        assert resp_2.status_code == 200
        events_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_event_and_count(vsr_1_event_text, initial_count_vsr_1 + 4, events_ns_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 4, events_ns_1)
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 4, events_ns_2)

        print("\nStep 6: remove VSRoute and check")
        delete_v_s_route(kube_apis.custom_objects, v_s_route_setup.route_m.name, v_s_route_setup.namespace)
        wait_before_test(1)
        new_config = get_vs_nginx_template_conf(kube_apis.v1,
                                                v_s_route_setup.namespace,
                                                v_s_route_setup.vs_name,
                                                ic_pod_name,
                                                ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert resp_1.status_code == 404
        assert resp_2.status_code == 404
        assert resp_3.status_code == 200
        events_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_locations_not_in_config(new_config, v_s_route_setup.route_m.paths)
        assert_event_and_count(vsr_1_event_text, initial_count_vsr_1 + 4, events_ns_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 5, events_ns_1)
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 5, events_ns_2)

        print("\nStep 7: restore VSRoute and check")
        create_v_s_route_from_yaml(kube_apis.custom_objects,
                                   f"{TEST_DATA}/virtual-server-route/route-multiple.yaml",
                                   v_s_route_setup.namespace)
        wait_before_test(1)
        new_config = get_vs_nginx_template_conf(kube_apis.v1,
                                                v_s_route_setup.namespace,
                                                v_s_route_setup.vs_name,
                                                ic_pod_name,
                                                ingress_controller_prerequisites.namespace)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert_responses_and_server_name(resp_1, resp_2, resp_3)
        events_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_locations_in_config(new_config, v_s_route_setup.route_m.paths)
        assert_event_and_count(vsr_1_event_text, 1, events_ns_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 6, events_ns_1)
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 6, events_ns_2)

        print("\nStep 8: remove one backend service and check")
        delete_service(kube_apis.v1, "backend1-svc", v_s_route_setup.route_m.namespace)
        wait_before_test(1)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert resp_1.status_code == 502
        assert resp_2.status_code == 200
        assert resp_3.status_code == 200
        events_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_event_and_count(vsr_1_event_text, 2, events_ns_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 7, events_ns_1)
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 7, events_ns_2)

        print("\nStep 9: restore backend service and check")
        create_service_with_name(kube_apis.v1, v_s_route_setup.route_m.namespace, "backend1-svc")
        wait_before_test(1)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert_responses_and_server_name(resp_1, resp_2, resp_3)
        events_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        events_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_event_and_count(vsr_1_event_text, 3, events_ns_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 8, events_ns_1)
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 8, events_ns_2)

        print("\nStep 10: remove VS and check")
        delete_virtual_server(kube_apis.custom_objects, v_s_route_setup.vs_name, v_s_route_setup.namespace)
        wait_before_test(1)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert resp_1.status_code == 404
        assert resp_2.status_code == 404
        assert resp_3.status_code == 404
        list0_list_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        list0_list_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_event_and_count(vsr_1_event_text, 3, list0_list_ns_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 8, list0_list_ns_1)
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 8, list0_list_ns_2)

        print("\nStep 11: restore VS and check")
        create_virtual_server_from_yaml(kube_apis.custom_objects,
                                        f"{TEST_DATA}/virtual-server-route/standard/virtual-server.yaml",
                                        v_s_route_setup.namespace)
        wait_before_test(1)
        resp_1 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_2 = requests.get(f"{req_url}{v_s_route_setup.route_m.paths[1]}",
                              headers={"host": v_s_route_setup.vs_host})
        resp_3 = requests.get(f"{req_url}{v_s_route_setup.route_s.paths[0]}",
                              headers={"host": v_s_route_setup.vs_host})
        assert_responses_and_server_name(resp_1, resp_2, resp_3)
        list1_list_ns_1 = get_events(kube_apis.v1, v_s_route_setup.route_m.namespace)
        list1_list_ns_2 = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_event_and_count(vsr_1_event_text, 4, list1_list_ns_1)
        assert_event_and_count(vs_event_text, 1, list1_list_ns_1)
        assert_event_and_count(vsr_2_event_text, initial_count_vsr_2 + 9, list1_list_ns_2)


@pytest.mark.parametrize('crd_ingress_controller',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]})],
                         indirect=True)
class TestVirtualServerRouteValidation:
    def test_vsr_without_vs(self, kube_apis,
                            ingress_controller_prerequisites,
                            crd_ingress_controller,
                            v_s_route_setup,
                            test_namespace):
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        vsr_name = create_v_s_route_from_yaml(kube_apis.custom_objects,
                                              f"{TEST_DATA}/virtual-server-route/route-orphan.yaml",
                                              test_namespace)
        vsr_paths = get_paths_from_vsr_yaml(f"{TEST_DATA}/virtual-server-route/route-orphan.yaml")
        wait_before_test(1)
        new_config = get_vs_nginx_template_conf(kube_apis.v1,
                                                v_s_route_setup.namespace,
                                                v_s_route_setup.vs_name,
                                                ic_pod_name,
                                                ingress_controller_prerequisites.namespace)
        new_list_ns_3 = get_events(kube_apis.v1, test_namespace)
        assert_locations_not_in_config(new_config, vsr_paths)
        assert_event_and_count(f"No VirtualServer references VirtualServerRoute {test_namespace}/{vsr_name}",
                               1,
                               new_list_ns_3)

    @pytest.mark.parametrize("route_yaml", [f"{TEST_DATA}/virtual-server-route/route-single-invalid-host.yaml",
                                            f"{TEST_DATA}/virtual-server-route/route-single-duplicate-path.yaml"])
    def test_make_existing_vsr_invalid(self, kube_apis,
                                       ingress_controller_prerequisites,
                                       crd_ingress_controller,
                                       v_s_route_setup,
                                       route_yaml):
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        patch_v_s_route_from_yaml(kube_apis.custom_objects,
                                  v_s_route_setup.route_s.name,
                                  route_yaml,
                                  v_s_route_setup.route_s.namespace)
        wait_before_test(1)
        new_config = get_vs_nginx_template_conf(kube_apis.v1,
                                                v_s_route_setup.namespace,
                                                v_s_route_setup.vs_name,
                                                ic_pod_name,
                                                ingress_controller_prerequisites.namespace)
        new_vs_events = get_events(kube_apis.v1, v_s_route_setup.namespace)
        new_vsr_events = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_locations_not_in_config(new_config, v_s_route_setup.route_s.paths)
        text = f"{v_s_route_setup.route_s.namespace}/{v_s_route_setup.route_s.name}"
        assert_event_and_count(f"Ignored VirtualServerRoute {text}",
                               1,
                               new_vs_events)
        assert_event_and_count(f"Ignored by VirtualServer {v_s_route_setup.namespace}/{v_s_route_setup.vs_name}",
                               1,
                               new_vsr_events)


@pytest.mark.parametrize('crd_ingress_controller',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]})],
                         indirect=True)
class TestCreateInvalidVirtualServerRoute:
    def test_create_invalid_vsr(self, kube_apis,
                                ingress_controller_prerequisites,
                                crd_ingress_controller,
                                v_s_route_setup):
        route_yaml = f"{TEST_DATA}/virtual-server-route/route-single-duplicate-path.yaml"
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        text = f"{v_s_route_setup.route_s.namespace}/{v_s_route_setup.route_s.name}"
        vs_event_text = f"Ignored VirtualServerRoute {text}: spec.subroutes[1].path: Duplicate value: \"/backend2\""
        vsr_event_text = f"Ignored by VirtualServer {v_s_route_setup.namespace}/{v_s_route_setup.vs_name}"
        delete_v_s_route(kube_apis.custom_objects,
                         v_s_route_setup.route_s.name,
                         v_s_route_setup.route_s.namespace)

        create_v_s_route_from_yaml(kube_apis.custom_objects,
                                   route_yaml,
                                   v_s_route_setup.route_s.namespace)
        wait_before_test(1)
        new_config = get_vs_nginx_template_conf(kube_apis.v1,
                                                v_s_route_setup.namespace,
                                                v_s_route_setup.vs_name,
                                                ic_pod_name,
                                                ingress_controller_prerequisites.namespace)
        new_vs_events = get_events(kube_apis.v1, v_s_route_setup.namespace)
        new_vsr_events = get_events(kube_apis.v1, v_s_route_setup.route_s.namespace)
        assert_locations_not_in_config(new_config, v_s_route_setup.route_s.paths)
        assert_event_and_count(vs_event_text,
                               1,
                               new_vs_events)
        assert_event_and_count(vsr_event_text,
                               1,
                               new_vsr_events)

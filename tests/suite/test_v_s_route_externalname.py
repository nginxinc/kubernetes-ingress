import requests
import pytest

from settings import TEST_DATA
from suite.custom_resources_utils import create_virtual_server_from_yaml, \
    create_v_s_route_from_yaml, get_vs_nginx_template_conf
from suite.fixtures import VirtualServerRoute, PublicEndpoint
from suite.resources_utils import get_first_pod_name, get_events, \
    wait_before_test, replace_configmap_from_yaml, create_service_from_yaml, \
    delete_namespace, create_namespace_with_name_from_yaml, read_service, replace_service, replace_configmap
from suite.yaml_utils import get_paths_from_vsr_yaml, get_route_namespace_from_vs_yaml, get_first_vs_host_from_yaml, \
    get_external_host_from_service_yaml


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


class ReducedVirtualServerRouteSetup:
    """
    Encapsulate Virtual Server Example details.

    Attributes:
        public_endpoint (PublicEndpoint):
        namespace (str):
        vs_host (str):
        vs_name (str):
        route (VirtualServerRoute): route with single subroute
    """

    def __init__(self, public_endpoint: PublicEndpoint,
                 namespace, vs_host, vs_name, route: VirtualServerRoute, external_svc, external_host):
        self.public_endpoint = public_endpoint
        self.namespace = namespace
        self.vs_host = vs_host
        self.vs_name = vs_name
        self.route = route
        self.external_svc = external_svc
        self.external_host = external_host


@pytest.fixture(scope="class")
def vsr_externalname_setup(request, kube_apis,
                           ingress_controller_prerequisites,
                           ingress_controller_endpoint) -> ReducedVirtualServerRouteSetup:
    """
    Prepare an example app for Virtual Server Route.

    1st namespace with externalName svc and VS+VSR.

    :param request: internal pytest fixture
    :param kube_apis: client apis
    :param ingress_controller_endpoint:
    :param ingress_controller_prerequisites:
    :return:
    """
    vs_routes_ns = get_route_namespace_from_vs_yaml(
        f"{TEST_DATA}/{request.param['example']}/standard/virtual-server.yaml")
    ns_1 = create_namespace_with_name_from_yaml(kube_apis.v1,
                                                vs_routes_ns[0],
                                                f"{TEST_DATA}/common/ns.yaml")
    print("------------------------- Deploy Virtual Server -----------------------------------")
    vs_name = create_virtual_server_from_yaml(kube_apis.custom_objects,
                                              f"{TEST_DATA}/{request.param['example']}/standard/virtual-server.yaml",
                                              ns_1)
    vs_host = get_first_vs_host_from_yaml(f"{TEST_DATA}/{request.param['example']}/standard/virtual-server.yaml")

    print("------------------------- Deploy Virtual Server Route -----------------------------------")
    vsr_name = create_v_s_route_from_yaml(kube_apis.custom_objects,
                                          f"{TEST_DATA}/{request.param['example']}/route-single.yaml",
                                          ns_1)
    vsr_paths = get_paths_from_vsr_yaml(f"{TEST_DATA}/{request.param['example']}/route-single.yaml")
    route = VirtualServerRoute(ns_1, vsr_name, vsr_paths)

    print("---------------------- Deploy ExternalName service and update ConfigMap ----------------------------")
    config_map_name = ingress_controller_prerequisites.config_map["metadata"]["name"]
    replace_configmap_from_yaml(kube_apis.v1, config_map_name,
                                ingress_controller_prerequisites.namespace,
                                f"{TEST_DATA}/{request.param['example']}/nginx-config.yaml")
    external_svc_src = f"{TEST_DATA}/{request.param['example']}/externalname-svc.yaml"
    external_svc_name = create_service_from_yaml(kube_apis.v1, route.namespace, external_svc_src)
    external_svc_host = get_external_host_from_service_yaml(external_svc_src)
    wait_before_test(2)

    def fin():
        print("Delete test namespace")
        delete_namespace(kube_apis.v1, ns_1)

    request.addfinalizer(fin)

    return ReducedVirtualServerRouteSetup(ingress_controller_endpoint,
                                          ns_1, vs_host, vs_name, route, external_svc_name, external_svc_host)


@pytest.mark.skip_for_nginx_oss
@pytest.mark.parametrize('crd_ingress_controller, vsr_externalname_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-route-externalname"})],
                         indirect=True)
class TestVSRWithExternalNameService:
    def test_responses(self, kube_apis,
                       crd_ingress_controller,
                       vsr_externalname_setup):
        req_url = f"http://{vsr_externalname_setup.public_endpoint.public_ip}:" \
            f"{vsr_externalname_setup.public_endpoint.port}"
        resp = requests.get(f"{req_url}{vsr_externalname_setup.route.paths[0]}",
                            headers={"host": vsr_externalname_setup.vs_host})
        assert resp.status_code == 502

    def test_template_config(self, kube_apis,
                             ingress_controller_prerequisites,
                             crd_ingress_controller,
                             vsr_externalname_setup):
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        initial_config = get_vs_nginx_template_conf(kube_apis.v1,
                                                    vsr_externalname_setup.namespace,
                                                    vsr_externalname_setup.vs_name,
                                                    ic_pod_name,
                                                    ingress_controller_prerequisites.namespace)

        line = f"zone vs_{vsr_externalname_setup.namespace}_{vsr_externalname_setup.vs_name}" \
            f"_vsr_{vsr_externalname_setup.route.namespace}_{vsr_externalname_setup.route.name}_ext-backend 256k;"
        assert line in initial_config
        assert "random two least_conn;" in initial_config
        assert f"server {vsr_externalname_setup.external_host}:80 max_fails=1 fail_timeout=10s max_conns=0 resolve;"\
               in initial_config

    def test_events_flows(self, kube_apis,
                          ingress_controller_prerequisites,
                          crd_ingress_controller,
                          vsr_externalname_setup):
        text_vsr = f"{vsr_externalname_setup.route.namespace}/{vsr_externalname_setup.route.name}"
        text_vs = f"{vsr_externalname_setup.namespace}/{vsr_externalname_setup.vs_name}"
        vsr_event_text = f"Configuration for {text_vsr} was added or updated"
        vs_event_text = f"Configuration for {text_vs} was added or updated"
        vs_event_update_text = f"Configuration for {text_vs} was updated"
        initial_events = get_events(kube_apis.v1, vsr_externalname_setup.route.namespace)
        initial_count_vsr = assert_event_and_get_count(vsr_event_text, initial_events)
        initial_count_vs = assert_event_and_get_count(vs_event_text, initial_events)
        initial_count_vs_up = assert_event_and_get_count(vs_event_update_text, initial_events)

        print("Step 1: Update external host in externalName service")
        external_svc = read_service(kube_apis.v1, vsr_externalname_setup.external_svc, vsr_externalname_setup.namespace)
        external_svc.spec.external_name = "demo.nginx.com"
        replace_service(kube_apis.v1,
                        vsr_externalname_setup.external_svc, vsr_externalname_setup.namespace, external_svc)
        wait_before_test(1)

        events_step_1 = get_events(kube_apis.v1, vsr_externalname_setup.route.namespace)
        assert_event_and_count(vsr_event_text, initial_count_vsr + 1, events_step_1)
        assert_event_and_count(vs_event_text, initial_count_vs + 1, events_step_1)
        assert_event_and_count(vs_event_update_text, initial_count_vs_up, events_step_1)

        print("Step 2: Remove resolver from ConfigMap to trigger an error")
        vsr_event_warning_text = f"Configuration for {text_vsr} was updated with warning(s):"
        config_map_name = ingress_controller_prerequisites.config_map["metadata"]["name"]
        replace_configmap(kube_apis.v1, config_map_name,
                          ingress_controller_prerequisites.namespace,
                          ingress_controller_prerequisites.config_map)
        wait_before_test(1)

        events_step_2 = get_events(kube_apis.v1, vsr_externalname_setup.route.namespace)
        assert_event_and_count(vsr_event_warning_text, 1, events_step_2)
        assert_event_and_count(vs_event_update_text, initial_count_vs_up + 1, events_step_2)

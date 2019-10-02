import pytest
import requests

from settings import TEST_DATA
from suite.custom_resources_utils import get_vs_nginx_template_conf
from suite.resources_utils import replace_configmap_from_yaml, \
    ensure_connection_to_public_endpoint, replace_configmap, create_service_from_yaml, get_first_pod_name, get_events, \
    read_service, replace_service, wait_before_test
from suite.yaml_utils import get_external_host_from_service_yaml


class ExternalNameSetup:
    """Encapsulate ExternalName example details.

    Attributes:
        ic_pod_name:
        external_host: external service host
    """
    def __init__(self, ic_pod_name, external_svc, external_host):
        self.ic_pod_name = ic_pod_name
        self.external_svc = external_svc
        self.external_host = external_host


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


@pytest.fixture(scope="class")
def vs_externalname_setup(request,
                          kube_apis,
                          ingress_controller_prerequisites,
                          virtual_server_setup) -> ExternalNameSetup:
    print("------------------------- Prepare ExternalName Setup -----------------------------------")
    external_svc_src = f"{TEST_DATA}/virtual-server-externalname/externalname-svc.yaml"
    external_svc_host = get_external_host_from_service_yaml(external_svc_src)
    config_map_name = ingress_controller_prerequisites.config_map["metadata"]["name"]
    replace_configmap_from_yaml(kube_apis.v1, config_map_name,
                                ingress_controller_prerequisites.namespace,
                                f"{TEST_DATA}/virtual-server-externalname/nginx-config.yaml")
    external_svc = create_service_from_yaml(kube_apis.v1, virtual_server_setup.namespace, external_svc_src)
    wait_before_test(1)
    ensure_connection_to_public_endpoint(virtual_server_setup.public_endpoint.public_ip,
                                         virtual_server_setup.public_endpoint.port,
                                         virtual_server_setup.public_endpoint.port_ssl)
    ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)

    def fin():
        print("Clean up ExternalName Setup:")
        replace_configmap(kube_apis.v1, config_map_name,
                          ingress_controller_prerequisites.namespace,
                          ingress_controller_prerequisites.config_map)

    request.addfinalizer(fin)

    return ExternalNameSetup(ic_pod_name, external_svc, external_svc_host)


@pytest.mark.skip_for_nginx_oss
@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-externalname", "app_type": "simple"})],
                         indirect=True)
class TestVSWithExternalNameService:
    def test_response(self, kube_apis, crd_ingress_controller, virtual_server_setup, vs_externalname_setup):
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 502

    def test_template_config(self, kube_apis, ingress_controller_prerequisites,
                             crd_ingress_controller,
                             virtual_server_setup, vs_externalname_setup):
        result_conf = get_vs_nginx_template_conf(kube_apis.v1,
                                                 virtual_server_setup.namespace,
                                                 virtual_server_setup.vs_name,
                                                 vs_externalname_setup.ic_pod_name,
                                                 ingress_controller_prerequisites.namespace)
        line = f"zone vs_{virtual_server_setup.namespace}_{virtual_server_setup.vs_name}_backend1 256k;"
        assert line in result_conf
        assert "random two least_conn;" in result_conf
        assert f"server {vs_externalname_setup.external_host}:80 max_fails=1 fail_timeout=10s max_conns=0 resolve;"\
               in result_conf

    def test_events_flows(self, kube_apis, ingress_controller_prerequisites,
                          crd_ingress_controller, virtual_server_setup, vs_externalname_setup):
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        vs_event_text = f"Configuration for {text} was added or updated"
        vs_event_update_text = f"Configuration for {text} was updated"
        events_vs = get_events(kube_apis.v1, virtual_server_setup.namespace)
        initial_count = assert_event_and_get_count(vs_event_text, events_vs)
        initial_count_up = assert_event_and_get_count(vs_event_update_text, events_vs)

        print("Step 1: Update external host in externalName service")
        external_svc = read_service(kube_apis.v1, vs_externalname_setup.external_svc, virtual_server_setup.namespace)
        external_svc.spec.external_name = "demo.nginx.com"
        replace_service(kube_apis.v1, vs_externalname_setup.external_svc, virtual_server_setup.namespace, external_svc)
        wait_before_test(1)

        events_step_1 = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_event_and_count(vs_event_text, initial_count + 1, events_step_1)
        assert_event_and_count(vs_event_update_text, initial_count_up, events_step_1)

        print("Step 2: Remove resolver from ConfigMap to trigger an error")
        config_map_name = ingress_controller_prerequisites.config_map["metadata"]["name"]
        vs_event_warning_text = f"Configuration for {text} was updated with warning(s):"
        replace_configmap(kube_apis.v1, config_map_name,
                          ingress_controller_prerequisites.namespace,
                          ingress_controller_prerequisites.config_map)
        wait_before_test(1)

        events_step_2 = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_event_and_count(vs_event_warning_text, 1, events_step_2)
        assert_event_and_count(vs_event_update_text, initial_count_up, events_step_2)

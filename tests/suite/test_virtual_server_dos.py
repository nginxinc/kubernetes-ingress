import requests
import pytest
import time

from settings import TEST_DATA, DEPLOYMENTS
from suite.custom_resources_utils import (
    create_dos_logconf_from_yaml,
    create_dos_policy_from_yaml,
    create_dos_protected_from_yaml,
    delete_dos_policy,
    delete_dos_logconf,
    delete_dos_protected,
    get_vs_nginx_template_conf,
    create_virtual_server_from_yaml,
    delete_virtual_server,
)
from suite.resources_utils import (
    wait_before_test,
    create_example_app,
    wait_until_all_pods_are_ready,
    create_items_from_yaml,
    delete_items_from_yaml,
    delete_common_app,
    ensure_connection_to_public_endpoint,
    create_ingress_with_dos_annotations,
    ensure_response_from_backend,
    get_ingress_nginx_template_conf,
    get_first_pod_name,
    get_file_contents,
    get_service_endpoint,
    get_test_file_name,
    write_to_json,
)
from suite.yaml_utils import (
    get_first_ingress_host_from_yaml,
    get_first_host_from_yaml,
    get_paths_from_vs_yaml,
)

# src_ing_yaml = f"{TEST_DATA}/dos/dos-ingress.yaml"
valid_resp_addr = "Server address:"
valid_resp_name = "Server name:"
invalid_resp_title = "Request Rejected"
invalid_resp_body = "The requested URL was rejected. Please consult with your administrator."
reload_times = {}

class VirtualServerSetupDos:
    def __init__(self, public_endpoint, namespace, vs_host, vs_name, vs_paths):
        self.public_endpoint = public_endpoint
        self.namespace = namespace
        self.vs_host = vs_host
        self.vs_name = vs_name
        self.backend_1_url = (
            f"http://{public_endpoint.public_ip}:{public_endpoint.port}{vs_paths[0]}"
        )

@pytest.fixture(scope="class")
def virtual_server_setup_dos(
    request, kube_apis, ingress_controller_endpoint, test_namespace
) -> VirtualServerSetupDos:
    print(
        "------------------------- Deploy Virtual Server Example -----------------------------------"
    )
    vs_source = f"{TEST_DATA}/virtual-server-dos/virtual-server.yaml"
    vs_name = create_virtual_server_from_yaml(kube_apis.custom_objects, vs_source, test_namespace)
    vs_host = get_first_host_from_yaml(vs_source)
    vs_paths = get_paths_from_vs_yaml(vs_source)
    if request.param["app_type"]:
        create_example_app(kube_apis, request.param["app_type"], test_namespace)
        wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)

    def fin():
        print("Clean up Virtual Server Example:")
        delete_virtual_server(kube_apis.custom_objects, vs_name, test_namespace)
        if request.param["app_type"]:
            delete_common_app(kube_apis, request.param["app_type"], test_namespace)

    request.addfinalizer(fin)

    return VirtualServerSetupDos(
        ingress_controller_endpoint, test_namespace, vs_host, vs_name, vs_paths
    )


class DosSetup:
    """
    Encapsulate the example details.
    Attributes:
        req_url (str):
    """
    def __init__(self, req_url):
        self.req_url = req_url

@pytest.fixture(scope="class")
def dos_setup(
    request, kube_apis, ingress_controller_endpoint, test_namespace
) -> DosSetup:
    """
    Deploy simple application and all the DOS resources under test in one namespace.

    :param request: pytest fixture
    :param kube_apis: client apis
    :param ingress_controller_endpoint: public endpoint
    :param test_namespace:
    :return: BackendSetup
    """
    req_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.port}/"

    print("------------------------- Deploy webapp -----------------------------")
    src_webapp_yaml = f"{TEST_DATA}/virtual-server-dos/webapp.yaml"
    create_items_from_yaml(kube_apis, src_webapp_yaml, test_namespace)

    print("------------------------- Deploy logconf -----------------------------")
    src_log_yaml = f"{TEST_DATA}/virtual-server-dos/dos-logconf.yaml"
    log_name = create_dos_logconf_from_yaml(kube_apis.custom_objects, src_log_yaml, test_namespace)

    print(f"------------------------- Deploy dataguard-alarm appolicy ---------------------------")
    src_pol_yaml = f"{TEST_DATA}/virtual-server-dos/dos-policy.yaml"
    pol_name = create_dos_policy_from_yaml(kube_apis.custom_objects, src_pol_yaml, test_namespace)

    print(f"------------------------- Deploy protected resource ---------------------------")
    src_protected_yaml = f"{TEST_DATA}/virtual-server-dos/dos-protected.yaml"
    protected_name = create_dos_protected_from_yaml(kube_apis.custom_objects, src_protected_yaml, test_namespace)

    def fin():
        print("Clean up:")
        delete_dos_policy(kube_apis.custom_objects, pol_name, test_namespace)
        delete_dos_logconf(kube_apis.custom_objects, log_name, test_namespace)
        delete_dos_protected(kube_apis.custom_objects, protected_name, test_namespace)
        delete_items_from_yaml(kube_apis, src_webapp_yaml, test_namespace)

    request.addfinalizer(fin)

    return DosSetup(req_url)


@pytest.mark.dos
@pytest.mark.parametrize('crd_ingress_controller_with_dos, virtual_server_setup_dos',
                         [({"type": "complete", "extra_args": [
                             f"-enable-custom-resources",
                             f"-enable-app-protect-dos",
                             f"-v=3",
                         ]},
                           {"example": "virtual-server-dos", "app_type": "simple"})],
                         indirect=True)
class TestDos:

    def test_responses_after_setup(self, kube_apis, crd_ingress_controller_with_dos, dos_setup, virtual_server_setup_dos ):
        print("\nStep 1: initial check")
        resp = requests.get(virtual_server_setup_dos.backend_1_url,
                            headers={"host": virtual_server_setup_dos.vs_host})
        assert resp.status_code == 200


    def test_dos_vs_logs(
        self,
        kube_apis,
        crd_ingress_controller_with_dos,
        virtual_server_setup_dos,
        dos_setup,
        test_namespace,
    ):
        """
        Test app protect logs appear in syslog after sending request to dos enabled route
        """
        src_syslog_yaml = f"{TEST_DATA}/virtual-server-dos/syslog.yaml"
        log_loc = f"/var/log/messages"

        create_items_from_yaml(kube_apis, src_syslog_yaml, test_namespace)

        print("----------------------- Get syslog pod name ----------------------")
        syslog_pod = ""
        for thing in kube_apis.v1.list_namespaced_pod(test_namespace).items:
            if "syslog" in thing.metadata.name:
                syslog_pod = thing.metadata.name

        assert "syslog" in syslog_pod

        print("----------------------- Send request ----------------------")
        ensure_response_from_backend(virtual_server_setup_dos.backend_1_url, virtual_server_setup_dos.vs_host)

        response = requests.get(virtual_server_setup_dos.backend_1_url, headers={"host": virtual_server_setup_dos.vs_host})
        print(response.text)

        wait_before_test(20)

        print("----------------------- Check Logs ----------------------")
        print(f'log_loc {log_loc} syslog_pod {syslog_pod} test_namespace {test_namespace}')
        log_contents = get_file_contents(kube_apis.v1, log_loc, syslog_pod, test_namespace)

        delete_items_from_yaml(kube_apis, src_syslog_yaml, test_namespace)

        wait_before_test(10)

        assert 'product="app-protect-dos"' in log_contents
        assert f'vs_name="{test_namespace}/dos-protected/name"' in log_contents
        assert 'bad_actor' in log_contents


    def test_vs_with_dos_config(self, kube_apis, crd_ingress_controller_with_dos, dos_setup, virtual_server_setup_dos, test_namespace):
        """
        Test to verify Dos annotations in nginx config
        """
        conf_annotations = [
            f"app_protect_dos_enable on;",
            f"app_protect_dos_security_log_enable on;",
            f"app_protect_dos_monitor \"webapp.example.com\";",
            f"app_protect_dos_name \"{test_namespace}/dos-protected/name\";",
        ]

        print("\n confirm response for standard request")
        resp = requests.get(virtual_server_setup_dos.backend_1_url,
                            headers={"host": virtual_server_setup_dos.vs_host})
        assert resp.status_code == 200

        pod_name = get_first_pod_name(kube_apis.v1, "nginx-ingress")

        result_conf = get_vs_nginx_template_conf(kube_apis.v1,
                                            virtual_server_setup_dos.namespace,
                                            virtual_server_setup_dos.vs_name,
                                            pod_name,
                                            "nginx-ingress")

        for _ in conf_annotations:
            assert _ in result_conf

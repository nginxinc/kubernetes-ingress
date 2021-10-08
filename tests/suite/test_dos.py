import requests
import pytest

from settings import TEST_DATA, DEPLOYMENTS
from suite.custom_resources_utils import (
    create_dos_logconf_from_yaml,
    create_dos_policy_from_yaml,
    delete_dos_policy,
    delete_dos_logconf,
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
from suite.yaml_utils import get_first_ingress_host_from_yaml

src_ing_yaml = f"{TEST_DATA}/dos/dos-ingress.yaml"
valid_resp_addr = "Server address:"
valid_resp_name = "Server name:"
invalid_resp_title = "Request Rejected"
invalid_resp_body = "The requested URL was rejected. Please consult with your administrator."
reload_times = {}


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
    print("------------------------- Deploy simple backend application -------------------------")
    create_example_app(kube_apis, "simple", test_namespace)
    req_url = f"https://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.port_ssl}/"
    wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)
    ensure_connection_to_public_endpoint(
        ingress_controller_endpoint.public_ip,
        ingress_controller_endpoint.port,
        ingress_controller_endpoint.port_ssl,
    )

    print("------------------------- Deploy Secret -----------------------------")
    src_sec_yaml = f"{TEST_DATA}/dos/dos-secret.yaml"
    create_items_from_yaml(kube_apis, src_sec_yaml, test_namespace)

    print("------------------------- Deploy logconf -----------------------------")
    src_log_yaml = f"{TEST_DATA}/dos/dos-logconf.yaml"
    log_name = create_dos_logconf_from_yaml(kube_apis.custom_objects, src_log_yaml, test_namespace)

    print(f"------------------------- Deploy dataguard-alarm appolicy ---------------------------")
    src_pol_yaml = f"{TEST_DATA}/dos/dos-policy.yaml"
    pol_name = create_dos_policy_from_yaml(kube_apis.custom_objects, src_pol_yaml, test_namespace)

    def fin():
        print("Clean up:")
        delete_dos_policy(kube_apis.custom_objects, pol_name, test_namespace)
        delete_dos_logconf(kube_apis.custom_objects, log_name, test_namespace)
        delete_common_app(kube_apis, "simple", test_namespace)
        delete_items_from_yaml(kube_apis, src_sec_yaml, test_namespace)
        write_to_json(f"reload-{get_test_file_name(request.node.fspath)}.json", reload_times)

    request.addfinalizer(fin)

    return DosSetup(req_url)


@pytest.mark.smoke
@pytest.mark.dos
@pytest.mark.parametrize(
    "crd_ingress_controller_with_dos",
    [
        {
            "extra_args": [
                f"-enable-custom-resources",
                f"-enable-app-protect-dos",
            ]
        }
    ],
    indirect=["crd_ingress_controller_with_dos"],
)
class TestDos:
    def test_ap_nginx_config_entries(
        self, kube_apis, crd_ingress_controller_with_dos, dos_setup, test_namespace
    ):
        """
        Test to verify Dos annotations in nginx config
        """
        conf_annotations = [
            f"app_protect_dos_enable on;",
            f"app_protect_dos_security_log_enable on;",
            f"app_protect_dos_monitor \"dos.example.com\";",
            f"app_protect_dos_name \"dos.example.com\";",
        ]

        create_ingress_with_dos_annotations(
            kube_apis, src_ing_yaml, test_namespace, "True", "True", "514"
        )

        ingress_host = get_first_ingress_host_from_yaml(src_ing_yaml)
        ensure_response_from_backend(dos_setup.req_url, ingress_host, check404=True)

        pod_name = get_first_pod_name(kube_apis.v1, "nginx-ingress")

        result_conf = get_ingress_nginx_template_conf(
            kube_apis.v1, test_namespace, "dos-ingress", pod_name, "nginx-ingress"
        )
        delete_items_from_yaml(kube_apis, src_ing_yaml, test_namespace)

        for _ in conf_annotations:
            assert _ in result_conf

    def test_dos_sec_logs_on(
        self,
        kube_apis,
        ingress_controller_prerequisites,
        crd_ingress_controller_with_dos,
        dos_setup,
        test_namespace,
    ):
        """
        Test corresponding log entries with correct policy (includes setting up a syslog server as defined in syslog.yaml)
        """
        src_syslog_yaml = f"{TEST_DATA}/dos/dos-syslog.yaml"
        log_loc = f"/var/log/messages"

        create_items_from_yaml(kube_apis, src_syslog_yaml, test_namespace)

        syslog_ep = get_service_endpoint(kube_apis, "syslog-svc", test_namespace)

        # items[-1] because syslog pod is last one to spin-up
        syslog_pod = kube_apis.v1.list_namespaced_pod(test_namespace).items[-1].metadata.name

        create_ingress_with_dos_annotations(
            kube_apis, src_ing_yaml, test_namespace, "True", "True", "514"
        )
        ingress_host = get_first_ingress_host_from_yaml(src_ing_yaml)

        print("--------- Run test while DOS module is enabled with correct policy ---------")

        ensure_response_from_backend(dos_setup.req_url, ingress_host, check404=True)

        print("----------------------- Send request ----------------------")
        response = requests.get(
            dos_setup.req_url, headers={"host": "dos.example.com"}, verify=False
        )
        print(response.text)
        wait_before_test(10)

        log_contents = get_file_contents(kube_apis.v1, log_loc, syslog_pod, test_namespace)

        delete_items_from_yaml(kube_apis, src_syslog_yaml, test_namespace)

        assert 'product="app-protect-dos"' in log_contents
        assert 'vs_name="dos.example.com"' in log_contents
        assert 'bad_actor' in log_contents


import requests
import pytest, time

from settings import TEST_DATA, DEPLOYMENTS
from suite.custom_resources_utils import (
    create_ap_logconf_from_yaml,
    create_ap_policy_from_yaml,
    delete_ap_policy,
    delete_ap_logconf,
)
from suite.resources_utils import (
    wait_before_test,
    create_example_app,
    wait_until_all_pods_are_ready,
    create_items_from_yaml,
    delete_items_from_yaml,
    delete_common_app,
    ensure_connection_to_public_endpoint,
    create_ingress_with_ap_annotations,
    ensure_response_from_backend,
    wait_before_test,
    get_events,
)
from suite.yaml_utils import get_first_ingress_host_from_yaml


ap_policies_under_test = ["dataguard-alarm", "file-block", "malformed-block"]
valid_resp_addr = "Server address:"
valid_resp_name = "Server name:"
invalid_resp_title = "Request Rejected"
invalid_resp_body = "The requested URL was rejected. Please consult with your administrator."


class BackendSetup:
    """
    Encapsulate the example details.

    Attributes:
        req_url (str):
        ingress_host (str):
    """

    def __init__(self, req_url, ingress_host):
        self.req_url = req_url
        self.ingress_host = ingress_host


@pytest.fixture(scope="function")
def backend_setup(request, kube_apis, ingress_controller_endpoint, test_namespace) -> BackendSetup:
    """
    Deploy a simple application and AppProtect manifests.

    :param request: pytest fixture
    :param kube_apis: client apis
    :param ingress_controller_endpoint: public endpoint
    :param test_namespace:
    :return: BackendSetup
    """
    policy = request.param["policy"]
    print("------------------------- Deploy backend application -------------------------")
    create_example_app(kube_apis, "simple", test_namespace)
    req_url = f"https://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.port_ssl}/backend1"
    wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)
    ensure_connection_to_public_endpoint(
        ingress_controller_endpoint.public_ip,
        ingress_controller_endpoint.port,
        ingress_controller_endpoint.port_ssl,
    )

    print("------------------------- Deploy Secret -----------------------------")
    src_sec_yaml = f"{TEST_DATA}/appprotect/appprotect-secret.yaml"
    create_items_from_yaml(kube_apis, src_sec_yaml, test_namespace)

    print("------------------------- Deploy logconf -----------------------------")
    src_log_yaml = f"{TEST_DATA}/appprotect/logconf.yaml"
    log_name = create_ap_logconf_from_yaml(kube_apis.custom_objects, src_log_yaml, test_namespace)

    print(f"------------------------- Deploy appolicy: {policy} ---------------------------")
    src_pol_yaml = f"{TEST_DATA}/appprotect/{policy}.yaml"
    pol_name = create_ap_policy_from_yaml(kube_apis.custom_objects, src_pol_yaml, test_namespace)

    print("------------------------- Deploy ingress -----------------------------")
    ingress_host = {}
    src_ing_yaml = f"{TEST_DATA}/appprotect/appprotect-ingress.yaml"
    create_ingress_with_ap_annotations(kube_apis, src_ing_yaml, test_namespace, policy, "True", "True", "127.0.0.1:514")
    ingress_host = get_first_ingress_host_from_yaml(src_ing_yaml)
    wait_before_test()

    def fin():
        print("Clean up:")
        src_ing_yaml = f"{TEST_DATA}/appprotect/appprotect-ingress.yaml"
        delete_items_from_yaml(kube_apis, src_ing_yaml, test_namespace)
        delete_ap_policy(kube_apis.custom_objects, pol_name, test_namespace)
        delete_ap_logconf(kube_apis.custom_objects, log_name, test_namespace)
        delete_common_app(kube_apis, "simple", test_namespace)
        src_sec_yaml = f"{TEST_DATA}/appprotect/appprotect-secret.yaml"
        delete_items_from_yaml(kube_apis, src_sec_yaml, test_namespace)

    request.addfinalizer(fin)

    return BackendSetup(req_url, ingress_host)


@pytest.mark.skip_for_nginx_oss
@pytest.mark.appprotect
@pytest.mark.smoke
@pytest.mark.parametrize(
    "crd_ingress_controller_with_ap",
    [{"extra_args": [f"-enable-custom-resources", f"-enable-app-protect"]}],
    indirect=["crd_ingress_controller_with_ap"],
)
class TestAppProtect:
    @pytest.mark.parametrize("backend_setup", [{"policy": "dataguard-alarm"}], indirect=True)
    def test_responses_dataguard_alarm(
        self, kube_apis, crd_ingress_controller_with_ap, backend_setup, test_namespace
    ):
        """
        Test dataguard-alarm AppProtect policy: Block malicious script in url
        """
        print("------------- Run test for AP policy: dataguard-alarm --------------")
        print(f"Request URL: {backend_setup.req_url} and Host: {backend_setup.ingress_host}")

        wait_before_test(40)
        ensure_response_from_backend(backend_setup.req_url, backend_setup.ingress_host)

        print("----------------------- Send valid request ----------------------")
        resp_valid = requests.get(
            backend_setup.req_url, headers={"host": backend_setup.ingress_host}, verify=False
        )
        print(resp_valid.text)
        assert valid_resp_addr in resp_valid.text
        assert valid_resp_name in resp_valid.text
        assert resp_valid.status_code == 200

        print("---------------------- Send invalid request ---------------------")
        resp_invalid = requests.get(
            backend_setup.req_url + "/<script>",
            headers={"host": backend_setup.ingress_host},
            verify=False,
        )
        print(resp_invalid.text)
        assert invalid_resp_title in resp_invalid.text
        assert invalid_resp_body in resp_invalid.text
        assert resp_invalid.status_code == 200

    @pytest.mark.parametrize("backend_setup", [{"policy": "file-block"}], indirect=True)
    def test_responses_file_block(self, kube_apis, crd_ingress_controller_with_ap, backend_setup, test_namespace):
        """
        Test file-block AppProtect policy: Block executing types e.g. .bat and .exe
        """   
        print("------------- Run test for AP policy: file-block --------------")
        print(f"Request URL: {backend_setup.req_url} and Host: {backend_setup.ingress_host}")

        wait_before_test(40)
        ensure_response_from_backend(backend_setup.req_url, backend_setup.ingress_host)

        print("----------------------- Send valid request ----------------------")
        resp_valid = requests.get(
            backend_setup.req_url, headers={"host": backend_setup.ingress_host}, verify=False
        )
        print(resp_valid.text)
        assert valid_resp_addr in resp_valid.text
        assert valid_resp_name in resp_valid.text
        assert resp_valid.status_code == 200

        print("---------------------- Send invalid request ---------------------")
        resp_invalid = requests.get(
            backend_setup.req_url + "/test.bat",
            headers={"host": backend_setup.ingress_host},
            verify=False,
        )
        print(resp_invalid.text)
        assert invalid_resp_title in resp_invalid.text
        assert invalid_resp_body in resp_invalid.text
        assert resp_invalid.status_code == 200

    @pytest.mark.parametrize("backend_setup", [{"policy": "malformed-block"}], indirect=True)
    def test_responses_malformed_block(
        self, kube_apis, crd_ingress_controller_with_ap, backend_setup, test_namespace
    ):
        """
        Test malformed-block blocking AppProtect policy: Block requests with invalid json or xml body
        """
        print("------------- Run test for AP policy: malformed-block --------------")
        print(f"Request URL: {backend_setup.req_url} and Host: {backend_setup.ingress_host}")

        wait_before_test(40)
        ensure_response_from_backend(backend_setup.req_url, backend_setup.ingress_host)

        print("----------------------- Send valid request with no body ----------------------")
        headers = {"host": backend_setup.ingress_host}
        resp_valid = requests.get(backend_setup.req_url, headers=headers, verify=False)
        print(resp_valid.text)
        assert valid_resp_addr in resp_valid.text
        assert valid_resp_name in resp_valid.text
        assert resp_valid.status_code == 200

        print("----------------------- Send valid request with body ----------------------")
        headers = {
             "Content-Type": "application/json",
             "host": backend_setup.ingress_host}
        resp_valid = requests.post(backend_setup.req_url, headers=headers, data="{}", verify=False)
        print(resp_valid.text)
        assert valid_resp_addr in resp_valid.text
        assert valid_resp_name in resp_valid.text
        assert resp_valid.status_code == 200

        print("---------------------- Send invalid request ---------------------")
        resp_invalid = requests.post(backend_setup.req_url, headers=headers, data="{{}}", verify=False,)
        print(resp_invalid.text)
        assert invalid_resp_title in resp_invalid.text
        assert invalid_resp_body in resp_invalid.text
        assert resp_invalid.status_code == 200

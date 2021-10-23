import requests
import pytest, json

from settings import TEST_DATA, DEPLOYMENTS
from suite.ap_resources_utils import (
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
    get_last_reload_time,
    get_test_file_name,
    write_to_json,
)
from suite.yaml_utils import get_first_ingress_host_from_yaml


ap_policies_under_test = ["dataguard-alarm", "file-block", "malformed-block"]
valid_resp_addr = "Server address:"
valid_resp_name = "Server name:"
invalid_resp_title = "Request Rejected"
invalid_resp_body = "The requested URL was rejected. Please consult with your administrator."
reload_times = {}


class BackendSetup:
    """
    Encapsulate the example details.

    Attributes:
        req_url (str):
        ingress_host (str):
    """

    def __init__(self, req_url, req_url_2, metrics_url, ingress_host):
        self.req_url = req_url
        self.req_url_2 = req_url_2
        self.metrics_url = metrics_url
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
    req_url_2 = f"https://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.port_ssl}/backend2"
    metrics_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
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
    create_ingress_with_ap_annotations(
        kube_apis, src_ing_yaml, test_namespace, policy, "True", "True", "127.0.0.1:514"
    )
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
        write_to_json(
            f"reload-{get_test_file_name(request.node.fspath)}.json",
            reload_times
        )

    request.addfinalizer(fin)

    return BackendSetup(req_url, req_url_2, metrics_url, ingress_host)


@pytest.mark.skip_for_nginx_oss
@pytest.mark.appprotect
@pytest.mark.smoke
@pytest.mark.parametrize(
    "crd_ingress_controller_with_ap",
    [
        {
            "extra_args": [
                f"-enable-custom-resources",
                f"-enable-app-protect",
                f"-enable-prometheus-metrics",
            ]
        }
    ],
    indirect=True,
)
class TestAppProtect:
    @pytest.mark.parametrize("backend_setup", [{"policy": "dataguard-alarm"}], indirect=True)
    def test_responses_dataguard_alarm(
        self, request, kube_apis, crd_ingress_controller_with_ap, backend_setup, test_namespace
    ):
        """
        Test dataguard-alarm AppProtect policy: Block malicious script in url
        """
        print("------------- Run test for AP policy: dataguard-alarm --------------")
        print(f"Request URL: {backend_setup.req_url} and Host: {backend_setup.ingress_host}")

        ensure_response_from_backend(
            backend_setup.req_url, backend_setup.ingress_host, check404=True
        )

        print("----------------------- Send valid request ----------------------")
        resp_valid = requests.get(
            backend_setup.req_url, headers={"host": backend_setup.ingress_host}, verify=False
        )

        print(resp_valid.text)
        reload_ms = get_last_reload_time(backend_setup.metrics_url, "nginx")
        print(f"last reload duration: {reload_ms} ms")
        reload_times[f"{request.node.name}"] = f"last reload duration: {reload_ms} ms"

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
    def test_responses_file_block(
        self, request, kube_apis, crd_ingress_controller_with_ap, backend_setup, test_namespace
    ):
        """
        Test file-block AppProtect policy: Block executing types e.g. .bat and .exe
        """
        print("------------- Run test for AP policy: file-block --------------")
        print(f"Request URL: {backend_setup.req_url} and Host: {backend_setup.ingress_host}")

        ensure_response_from_backend(
            backend_setup.req_url, backend_setup.ingress_host, check404=True
        )

        print("----------------------- Send valid request ----------------------")
        resp_valid = requests.get(
            backend_setup.req_url, headers={"host": backend_setup.ingress_host}, verify=False
        )
        print(resp_valid.text)

        reload_ms = get_last_reload_time(backend_setup.metrics_url, "nginx")
        print(f"last reload duration: {reload_ms} ms")
        reload_times[f"{request.node.name}"] = f"last reload duration: {reload_ms} ms"

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

        ensure_response_from_backend(
            backend_setup.req_url, backend_setup.ingress_host, check404=True
        )

        print("----------------------- Send valid request with no body ----------------------")
        headers = {"host": backend_setup.ingress_host}
        resp_valid = requests.get(backend_setup.req_url, headers=headers, verify=False)
        print(resp_valid.text)
        assert valid_resp_addr in resp_valid.text
        assert valid_resp_name in resp_valid.text
        assert resp_valid.status_code == 200

        print("----------------------- Send valid request with body ----------------------")
        headers = {"Content-Type": "application/json", "host": backend_setup.ingress_host}
        resp_valid = requests.post(backend_setup.req_url, headers=headers, data="{}", verify=False)
        print(resp_valid.text)
        assert valid_resp_addr in resp_valid.text
        assert valid_resp_name in resp_valid.text
        assert resp_valid.status_code == 200

        print("---------------------- Send invalid request ---------------------")
        resp_invalid = requests.post(
            backend_setup.req_url,
            headers=headers,
            data="{{}}",
            verify=False,
        )
        print(resp_invalid.text)
        assert invalid_resp_title in resp_invalid.text
        assert invalid_resp_body in resp_invalid.text
        assert resp_invalid.status_code == 200

    @pytest.mark.parametrize("backend_setup", [{"policy": "csrf"}], indirect=True)
    def test_responses_csrf(
        self,
        kube_apis,
        ingress_controller_endpoint,
        crd_ingress_controller_with_ap,
        backend_setup,
        test_namespace,
    ):
        """
        Test CSRF (Cross Site Request Forgery) AppProtect policy: Block requests with invalid/null/non-https origin-header
        """
        print("------------- Run test for AP policy: CSRF --------------")
        print(f"Request URL without CSRF protection: {backend_setup.req_url}")
        print(f"Request URL with CSRF protection: {backend_setup.req_url_2}")

        ensure_response_from_backend(
            backend_setup.req_url_2, backend_setup.ingress_host, check404=True
        )

        print("----------------------- Send request with http origin header ----------------------")

        headers = {"host": backend_setup.ingress_host, "Origin": "http://appprotect.example.com"}
        resp_valid = requests.post(
            backend_setup.req_url, headers=headers, verify=False, cookies={"flavor": "darkchoco"}
        )
        resp_invalid = requests.post(
            backend_setup.req_url_2, headers=headers, verify=False, cookies={"flavor": "whitechoco"}
        )

        print(resp_valid.text)
        print(resp_invalid.text)

        assert valid_resp_addr in resp_valid.text
        assert valid_resp_name in resp_valid.text
        assert resp_valid.status_code == 200
        assert invalid_resp_title in resp_invalid.text
        assert invalid_resp_body in resp_invalid.text
        assert resp_invalid.status_code == 200

    @pytest.mark.parametrize("backend_setup", [{"policy": "ap-user-def-browser"}], indirect=True)
    def test_responses_user_def_browser(
        self,
        crd_ingress_controller_with_ap,
        backend_setup,
    ):
        """
        Test User defined browser AppProtect policy: Block requests from built-in and user-defined browser based on action in policy.
        """
        print("------------- Run test for AP policy: User Defined Browser --------------")
        print(f"Request URL: {backend_setup.req_url}")

        ensure_response_from_backend(
            backend_setup.req_url, backend_setup.ingress_host, check404=True
        )

        print(
            "----------------------- Send request with User-Agent: browser ----------------------"
        )

        headers_firefox = {
            "host": backend_setup.ingress_host,
            "User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Firefox/59.0",
        }
        resp_firefox = requests.get(backend_setup.req_url, headers=headers_firefox, verify=False)
        headers_chrome = {
            "host": backend_setup.ingress_host,
            "User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Chrome/76.0.3809.100",
        }
        resp_chrome = requests.get(backend_setup.req_url_2, headers=headers_chrome, verify=False)
        headers_safari = {
            "host": backend_setup.ingress_host,
            "User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Safari/537.36",
        }
        resp_safari = requests.get(backend_setup.req_url_2, headers=headers_safari, verify=False)
        headers_custom1 = {
            "host": backend_setup.ingress_host,
            "User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 custombrowser1/0.1",
        }
        resp_custom1 = requests.get(backend_setup.req_url_2, headers=headers_custom1, verify=False)
        headers_custom2 = {
            "host": backend_setup.ingress_host,
            "User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 custombrowser2/0.1",
        }
        resp_custom2 = requests.get(backend_setup.req_url_2, headers=headers_custom2, verify=False)

        assert (
            200
            == resp_firefox.status_code
            == resp_chrome.status_code
            == resp_safari.status_code
            == resp_custom1.status_code
            == resp_custom2.status_code
        )
        assert (
            valid_resp_addr in resp_firefox.text
            and valid_resp_addr in resp_safari.text
            and valid_resp_addr in resp_custom2.text
        )
        assert (
            valid_resp_name in resp_firefox.text
            and valid_resp_name in resp_safari.text
            and valid_resp_name in resp_custom2.text
        )
        assert invalid_resp_title in resp_chrome.text and invalid_resp_title in resp_custom1.text
        assert invalid_resp_body in resp_chrome.text and invalid_resp_body in resp_custom1.text

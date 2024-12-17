import time

import pytest
import requests
from settings import TEST_DATA
from suite.fixtures.fixtures import PublicEndpoint
from suite.utils.resources_utils import (
    are_all_pods_in_ready_state,
    create_example_app,
    create_items_from_yaml,
    delete_common_app,
    delete_items_from_yaml,
    ensure_connection_to_public_endpoint,
    ensure_response_from_backend,
    get_first_pod_name,
    get_ingress_nginx_template_conf,
    get_pod_list,
    scale_deployment,
    wait_before_test,
    wait_until_all_pods_are_ready,
)
from suite.utils.yaml_utils import get_first_ingress_host_from_yaml, get_name_from_yaml


class AnnotationsSetup:
    """Encapsulate Annotations example details.

    Attributes:
        public_endpoint: PublicEndpoint
        ingress_src_file:
        ingress_name:
        ingress_pod_name:
        ingress_host:
        namespace: example namespace
    """

    def __init__(
        self,
        public_endpoint: PublicEndpoint,
        ingress_src_file,
        ingress_name,
        ingress_host,
        ingress_pod_name,
        namespace,
        request_url,
    ):
        self.public_endpoint = public_endpoint
        self.ingress_name = ingress_name
        self.ingress_pod_name = ingress_pod_name
        self.namespace = namespace
        self.ingress_host = ingress_host
        self.ingress_src_file = ingress_src_file
        self.request_url = request_url


@pytest.fixture(scope="class")
def annotations_setup(
    request,
    kube_apis,
    ingress_controller_prerequisites,
    ingress_controller_endpoint,
    ingress_controller,
    test_namespace,
) -> AnnotationsSetup:
    print("------------------------- Deploy Ingress with rate-limit annotations -----------------------------------")
    src = f"{TEST_DATA}/rate-limit/ingress/{request.param}/annotations-rl-ingress.yaml"
    create_items_from_yaml(kube_apis, src, test_namespace)
    ingress_name = get_name_from_yaml(src)
    ingress_host = get_first_ingress_host_from_yaml(src)
    request_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.port}/backend1"

    create_example_app(kube_apis, "simple", test_namespace)
    wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)

    ensure_connection_to_public_endpoint(
        ingress_controller_endpoint.public_ip, ingress_controller_endpoint.port, ingress_controller_endpoint.port_ssl
    )
    ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)

    def fin():
        if request.config.getoption("--skip-fixture-teardown") == "no":
            print("Clean up:")
            delete_common_app(kube_apis, "simple", test_namespace)
            delete_items_from_yaml(kube_apis, src, test_namespace)

    request.addfinalizer(fin)

    return AnnotationsSetup(
        ingress_controller_endpoint,
        src,
        ingress_name,
        ingress_host,
        ic_pod_name,
        test_namespace,
        request_url,
    )


@pytest.mark.annotations
@pytest.mark.parametrize("annotations_setup", ["standard", "mergeable"], indirect=True)
class TestRateLimitIngress:
    def test_ingress_rate_limit(self, kube_apis, annotations_setup, ingress_controller_prerequisites, test_namespace):
        """
        Test if rate-limit applies with 1rps for standard and mergeable ingresses
        """
        ensure_response_from_backend(annotations_setup.request_url, annotations_setup.ingress_host, check404=True)
        print("----------------------- Send request ----------------------")
        counter = []
        t_end = time.perf_counter() + 2  # send requests for 2 seconds
        while time.perf_counter() < t_end:
            resp = requests.get(
                annotations_setup.request_url,
                headers={"host": annotations_setup.ingress_host},
            )
            counter.append(resp.status_code)
        assert (counter.count(200)) <= 2 and (429 in counter)  # check for only 2 200s in the list


@pytest.mark.annotations
@pytest.mark.parametrize("annotations_setup", ["standard-scaled", "mergeable-scaled"], indirect=True)
class TestRateLimitIngressScaled:
    def test_ingress_rate_limit_scaled(
        self, kube_apis, annotations_setup, ingress_controller_prerequisites, test_namespace
    ):
        """
        Test if rate-limit scaling works with standard and mergeable ingresses
        """
        ns = ingress_controller_prerequisites.namespace
        scale_deployment(kube_apis.v1, kube_apis.apps_v1_api, "nginx-ingress", ns, 4)
        count = 0
        while (not are_all_pods_in_ready_state(kube_apis.v1, ns)) and count < 10:
            count += 1
            wait_before_test()

        ic_pods = get_pod_list(kube_apis.v1, ns)
        flag = False
        retries = 0
        while flag is False and retries < 10:
            retries += 1
            wait_before_test()
            for i in range(len(ic_pods)):
                conf = get_ingress_nginx_template_conf(
                    kube_apis.v1,
                    annotations_setup.namespace,
                    annotations_setup.ingress_name,
                    ic_pods[i].metadata.name,
                    ingress_controller_prerequisites.namespace,
                )
                flag = ("rate=10r/s" in conf) or ("rate=13r/s" in conf)

        assert flag
        scale_deployment(kube_apis.v1, kube_apis.apps_v1_api, "nginx-ingress", ns, 1)

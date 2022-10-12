import pytest
from settings import TEST_DATA
from suite.fixtures import PublicEndpoint
from suite.resources_utils import (
    create_example_app,
    create_items_from_yaml,
    create_secret_from_yaml,
    delete_common_app,
    delete_items_from_yaml,
    delete_secret,
    ensure_connection_to_public_endpoint,
    get_first_pod_name,
    get_ingress_nginx_template_conf,
    get_nginx_template_conf,
    wait_before_test,
    wait_until_all_pods_are_ready,
)
from suite.yaml_utils import get_first_ingress_host_from_yaml, get_name_from_yaml

paths = ["backend1", "backend2"]


class DisableIPV6Setup:
    """
    Encapsulate the Disable IPV6 Example details.

    Attributes:
        public_endpoint (PublicEndpoint):
        ingress_name (str):
        ingress_host (str):
        ingress_pod_name (str):
        namespace (str):
    """

    def __init__(self, public_endpoint: PublicEndpoint, ingress_name, ingress_host, ingress_pod_name, namespace):
        self.public_endpoint = public_endpoint
        self.ingress_host = ingress_host
        self.ingress_name = ingress_name
        self.ingress_pod_name = ingress_pod_name
        self.namespace = namespace


@pytest.fixture(scope="class", params=["standard", "mergeable"])
def disable_ipv6_setup(
    request,
    kube_apis,
    ingress_controller_prerequisites,
    ingress_controller_endpoint,
    ingress_controller,
    test_namespace,
) -> DisableIPV6Setup:
    print("------------------------- Deploy Disable IPV6 Example -----------------------------------")
    secret_name = create_secret_from_yaml(
        kube_apis.v1, test_namespace, f"{TEST_DATA}/disable-ipv6-ingress/disable-ipv6-secret.yaml"
    )

    create_items_from_yaml(
        kube_apis, f"{TEST_DATA}/disable-ipv6-ingress/{request.param}/disable-ipv6-ingress.yaml", test_namespace
    )
    ingress_name = get_name_from_yaml(f"{TEST_DATA}/disable-ipv6-ingress/{request.param}/disable-ipv6-ingress.yaml")
    ingress_host = get_first_ingress_host_from_yaml(
        f"{TEST_DATA}/disable-ipv6-ingress/{request.param}/disable-ipv6-ingress.yaml"
    )
    create_example_app(kube_apis, "simple", test_namespace)
    wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)

    ensure_connection_to_public_endpoint(
        ingress_controller_endpoint.public_ip,
        ingress_controller_endpoint.port,
        ingress_controller_endpoint.port_ssl,
    )
    ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)

    def fin():
        print("Clean up the Disable IPV6 Application:")
        delete_common_app(kube_apis, "simple", test_namespace)
        delete_items_from_yaml(
            kube_apis, f"{TEST_DATA}/disable-ipv6-ingress/{request.param}/disable-ipv6-ingress.yaml", test_namespace
        )
        delete_secret(kube_apis.v1, secret_name, test_namespace)

    request.addfinalizer(fin)

    return DisableIPV6Setup(ingress_controller_endpoint, ingress_name, ingress_host, ic_pod_name, test_namespace)


@pytest.mark.ingresses
class TestDisableIPV6:
    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param({"extra_args": ["-disable-ipv6"]}, id="one-additional-cli-args"),
        ],
        indirect=True,
    )
    def test_ipv6_listeners_not_in_config(
        self,
        kube_apis,
        disable_ipv6_setup: DisableIPV6Setup,
        ingress_controller_prerequisites,
    ):
        wait_before_test()
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        upstream_conf = get_ingress_nginx_template_conf(
            kube_apis.v1,
            disable_ipv6_setup.namespace,
            disable_ipv6_setup.ingress_name,
            disable_ipv6_setup.ingress_pod_name,
            ingress_controller_prerequisites.namespace,
        )
        assert "listen [::]:" not in nginx_config
        assert "listen [::]:" not in upstream_conf

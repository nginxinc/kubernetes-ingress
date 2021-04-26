from ssl import SSLError

import pytest

from suite.resources_utils import create_secret_from_yaml, is_secret_present, delete_secret, wait_before_test, \
    ensure_connection, replace_secret
from suite.ssl_utils import get_server_certificate_subject
from settings import TEST_DATA, DEPLOYMENTS


def assert_cn(endpoint, cn):
    host = "random" # any host would work
    subject_dict = get_server_certificate_subject(endpoint.public_ip, host, endpoint.port_ssl)
    assert subject_dict[b'CN'] == cn.encode('ascii')


def assert_unrecognized_name_error(endpoint):
    try:
        host = "random"  # any host would work
        get_server_certificate_subject(endpoint.public_ip, host, endpoint.port_ssl)
        pytest.fail("We expected an SSLError here, but didn't get it or got another error. Exiting...")
    except SSLError as e:
        assert "SSL" in e.library
        assert "TLSV1_UNRECOGNIZED_NAME" in e.reason


secret_path=f"{DEPLOYMENTS}/common/default-server-secret.yaml"
test_data_path=f"{TEST_DATA}/default-server"
invalid_secret_path=f"{test_data_path}/invalid-tls-secret.yaml"
new_secret_path=f"{test_data_path}/new-tls-secret.yaml"
secret_name="default-server-secret"
secret_namespace="nginx-ingress"


@pytest.fixture(scope="class")
def default_server_setup(ingress_controller_endpoint, ingress_controller):
    ensure_connection(f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.port}/")


@pytest.fixture(scope="class")
def secret_setup(request, kube_apis):
    def fin():
        if is_secret_present(kube_apis.v1, secret_name, secret_namespace):
            print("cleaning up secret!")
            delete_secret(kube_apis.v1, secret_name, secret_namespace)
            # restore the original secret created in ingress_controller_prerequisites fixture
            create_secret_from_yaml(kube_apis.v1, secret_namespace, secret_path)

    request.addfinalizer(fin)


@pytest.mark.ingresses
class TestDefaultServer:
    def test_with_default_tls_secret(self, kube_apis, ingress_controller_endpoint, secret_setup, default_server_setup):
        print("Step 1: ensure CN of the default server TLS cert")
        assert_cn(ingress_controller_endpoint, "NGINXIngressController")

        print("Step 2: ensure CN of the default server TLS cert after removing the secret")
        delete_secret(kube_apis.v1, secret_name, secret_namespace)
        wait_before_test(1)
        # Ingress Controller retains the previous valid secret
        assert_cn(ingress_controller_endpoint, "NGINXIngressController")

        print("Step 3: ensure CN of the default TLS cert after creating an updated secret")
        create_secret_from_yaml(kube_apis.v1, secret_namespace, new_secret_path)
        wait_before_test(1)
        assert_cn(ingress_controller_endpoint, "cafe.example.com")

        print("Step 4: ensure CN of the default TLS cert after making the secret invalid")
        replace_secret(kube_apis.v1, secret_name, secret_namespace, invalid_secret_path)
        wait_before_test(1)
        # Ingress Controller retains the previous valid secret
        assert_cn(ingress_controller_endpoint, "cafe.example.com")

        print("Step 5: ensure CN of the default TLS cert after restoring the secret")
        replace_secret(kube_apis.v1, secret_name, secret_namespace, secret_path)
        wait_before_test(1)
        assert_cn(ingress_controller_endpoint, "NGINXIngressController")

    @pytest.mark.parametrize(
        "ingress_controller",
        [
            pytest.param(
                {"extra_args": ["-default-server-tls-secret="]},
            ),
        ],
        indirect=True,
    )
    def test_without_default_tls_secret(self, ingress_controller_endpoint, default_server_setup):
        print("Ensure connection to HTTPS cannot be established")
        assert_unrecognized_name_error(ingress_controller_endpoint)

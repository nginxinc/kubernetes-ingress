import pytest
import requests
from settings import TEST_DATA
from suite.utils.custom_resources_utils import read_custom_resource
from suite.utils.policy_resources_utils import apply_and_wait_for_valid_policy, create_policy_from_yaml, delete_policy
from suite.utils.resources_utils import (
    create_example_app,
    create_items_from_yaml,
    create_secret_from_yaml,
    delete_common_app,
    delete_items_from_yaml,
    delete_secret,
    ensure_connection_to_public_endpoint,
    get_reload_count,
    wait_before_test,
    wait_for_reload,
    wait_until_all_pods_are_ready,
)
from suite.utils.yaml_utils import get_first_ingress_host_from_yaml, get_name_from_yaml

jwk_sec_valid_src = f"{TEST_DATA}/jwt-policy/secret/jwk-secret-valid.yaml"
jwk_sec_valid_opaque_src = f"{TEST_DATA}/jwt-policy/secret/jwk-secret-valid-opaque.yaml"
jwk_sec_invalid_src = f"{TEST_DATA}/jwt-policy/secret/jwk-secret-invalid.yaml"
jwt_pol_valid_src = f"{TEST_DATA}/jwt-policy/policies/jwt-policy-valid.yaml"
jwt_pol_multi_src = f"{TEST_DATA}/jwt-policy/policies/jwt-policy-valid-multi.yaml"
jwt_pol_invalid_src = f"{TEST_DATA}/jwt-policy/policies/jwt-policy-invalid.yaml"
jwt_pol_invalid_sec_src = f"{TEST_DATA}/jwt-policy/policies/jwt-policy-invalid-secret.yaml"
valid_token = f"{TEST_DATA}/jwt-policy/token.jwt"
invalid_token = f"{TEST_DATA}/jwt-policy/invalid-token.jwt"

valid_ingress_variants = [
    pytest.param("standard", id="standard"),
    pytest.param("mergeable-master", id="mergeable-master"),
    pytest.param("mergeable-minion", id="mergeable-minion"),
]


class IngressSetup:
    """Capture the data needed to exercise an Ingress variant."""

    def __init__(self, ingress_src_file, ingress_name, ingress_host, namespace, request_url, metrics_url):
        self.ingress_src_file = ingress_src_file
        self.ingress_name = ingress_name
        self.ingress_host = ingress_host
        self.namespace = namespace
        self.request_url = request_url
        self.metrics_url = metrics_url


def read_jwt_token(token_file):
    """Read a JWT token string from a file."""
    with open(token_file) as f:
        return f.readline().strip()


def deploy_ingress(kube_apis, ingress_controller_endpoint, test_namespace, ingress_variant):
    """Deploy a specific Ingress variant and wait for NGINX reload."""
    src = f"{TEST_DATA}/jwt-policy/ingress/{ingress_variant}/jwt-ingress.yaml"
    metrics_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.metrics_port}/metrics"
    count_before = get_reload_count(metrics_url)

    print(f"------------- Create ingress from {src} --------------")
    create_items_from_yaml(kube_apis, src, test_namespace)

    print("------------- Wait for reload after ingress apply --------------")
    wait_for_reload(metrics_url, count_before)

    ensure_connection_to_public_endpoint(
        ingress_controller_endpoint.public_ip,
        ingress_controller_endpoint.port,
        ingress_controller_endpoint.port_ssl,
    )

    ingress_name = get_name_from_yaml(src)
    ingress_host = get_first_ingress_host_from_yaml(src)
    request_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.port}/backend1"

    return IngressSetup(src, ingress_name, ingress_host, test_namespace, request_url, metrics_url)


@pytest.fixture(scope="class")
def backend_setup(request, kube_apis, test_namespace):
    """Deploy the simple backend once per test class and clean it up afterwards."""
    print("------------- Deploy simple backend app --------------")
    create_example_app(kube_apis, "simple", test_namespace)
    wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)

    def fin():
        if request.config.getoption("--skip-fixture-teardown") == "no":
            print("------------- Delete simple backend app --------------")
            delete_common_app(kube_apis, "simple", test_namespace)

    request.addfinalizer(fin)


@pytest.mark.skip_for_nginx_oss
@pytest.mark.policies
@pytest.mark.policies_jwt
@pytest.mark.parametrize(
    "crd_ingress_controller",
    [
        {
            "type": "complete",
            "extra_args": [
                "-enable-custom-resources",
                "-enable-leader-election=false",
                "-enable-prometheus-metrics",
            ],
        },
    ],
    indirect=True,
)
class TestJWTPoliciesIngress:
    def setup_single_policy(self, kube_apis, test_namespace, token, secret, policy, wait_valid=True):
        """Create JWK secret and JWT policy, returning secret name, policy name, and token value."""
        print("Create jwk secret")
        secret_name = create_secret_from_yaml(kube_apis.v1, test_namespace, secret)

        print("Create jwt policy")
        if wait_valid:
            apply_and_wait_for_valid_policy(kube_apis, test_namespace, policy)
            pol_name = get_name_from_yaml(policy)
        else:
            pol_name = create_policy_from_yaml(kube_apis.custom_objects, policy, test_namespace)
            wait_before_test()

        token_data = read_jwt_token(token)
        return secret_name, pol_name, token_data

    def setup_multiple_policies(self, kube_apis, test_namespace, token, secret, policy_1, policy_2):
        """Create JWK secret and two JWT policies, returning names and token value."""
        print("Create jwk secret")
        secret_name = create_secret_from_yaml(kube_apis.v1, test_namespace, secret)

        print("Create jwt policy #1")
        apply_and_wait_for_valid_policy(kube_apis, test_namespace, policy_1)
        pol_name_1 = get_name_from_yaml(policy_1)

        print("Create jwt policy #2")
        apply_and_wait_for_valid_policy(kube_apis, test_namespace, policy_2)
        pol_name_2 = get_name_from_yaml(policy_2)

        token_data = read_jwt_token(token)
        return secret_name, pol_name_1, pol_name_2, token_data

    @pytest.mark.parametrize("ingress_variant", valid_ingress_variants)
    @pytest.mark.parametrize("token", [valid_token, invalid_token])
    def test_jwt_policy_token(
        self,
        kube_apis,
        crd_ingress_controller,
        backend_setup,
        ingress_controller_endpoint,
        test_namespace,
        ingress_variant,
        token,
    ):
        """
        Test jwt-policy on standard, mergeable-master, and mergeable-minion Ingresses
        with no token, valid token, and invalid token.
        """
        secret_name = pol_name = None
        ingress_setup = None
        try:
            secret_name, pol_name, token_data = self.setup_single_policy(
                kube_apis,
                test_namespace,
                token,
                jwk_sec_valid_src,
                jwt_pol_valid_src,
            )
            ingress_setup = deploy_ingress(kube_apis, ingress_controller_endpoint, test_namespace, ingress_variant)

            resp_no_token = requests.get(
                ingress_setup.request_url,
                headers={"host": ingress_setup.ingress_host},
            )
            print(resp_no_token.status_code)

            resp_with_token = requests.get(
                ingress_setup.request_url,
                headers={"host": ingress_setup.ingress_host, "token": token_data},
            )
            print(resp_with_token.status_code)

            assert resp_no_token.status_code == 401
            assert "401 Authorization Required" in resp_no_token.text

            if token == valid_token:
                assert resp_with_token.status_code == 200
                assert "Request ID:" in resp_with_token.text
            else:
                assert resp_with_token.status_code == 401
                assert "Authorization Required" in resp_with_token.text
        finally:
            if ingress_setup is not None:
                delete_items_from_yaml(kube_apis, ingress_setup.ingress_src_file, test_namespace)
            if pol_name is not None:
                delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
            if secret_name is not None:
                delete_secret(kube_apis.v1, secret_name, test_namespace)

    @pytest.mark.parametrize("jwk_secret", [jwk_sec_valid_src, jwk_sec_valid_opaque_src, jwk_sec_invalid_src])
    def test_jwt_policy_secret(
        self,
        kube_apis,
        crd_ingress_controller,
        backend_setup,
        ingress_controller_endpoint,
        test_namespace,
        jwk_secret,
    ):
        """
        Test jwt-policy on Ingress with a valid nginx.org/jwk secret, a valid Opaque secret,
        and an invalid secret.
        """
        if jwk_secret in (jwk_sec_valid_src, jwk_sec_valid_opaque_src):
            pol = jwt_pol_valid_src
            ingress_variant = "standard"
            wait_valid = True
        elif jwk_secret == jwk_sec_invalid_src:
            pol = jwt_pol_invalid_sec_src
            ingress_variant = "standard-invalid-secret"
            wait_valid = False
        else:
            pytest.fail("Invalid configuration")

        secret_name = pol_name = None
        ingress_setup = None
        try:
            secret_name, pol_name, token_data = self.setup_single_policy(
                kube_apis,
                test_namespace,
                valid_token,
                jwk_secret,
                pol,
                wait_valid=wait_valid,
            )
            ingress_setup = deploy_ingress(kube_apis, ingress_controller_endpoint, test_namespace, ingress_variant)

            resp = requests.get(
                ingress_setup.request_url,
                headers={"host": ingress_setup.ingress_host, "token": token_data},
            )
            print(resp.status_code)

            if jwk_secret in (jwk_sec_valid_src, jwk_sec_valid_opaque_src):
                assert resp.status_code == 200
                assert "Request ID:" in resp.text
            elif jwk_secret == jwk_sec_invalid_src:
                assert resp.status_code == 500
                assert "Internal Server Error" in resp.text
            else:
                pytest.fail("Not a valid case or parameter")
        finally:
            if ingress_setup is not None:
                delete_items_from_yaml(kube_apis, ingress_setup.ingress_src_file, test_namespace)
            if pol_name is not None:
                delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
            if secret_name is not None:
                delete_secret(kube_apis.v1, secret_name, test_namespace)

    @pytest.mark.smoke
    @pytest.mark.parametrize("ingress_variant", valid_ingress_variants)
    @pytest.mark.parametrize("policy", [jwt_pol_valid_src, jwt_pol_invalid_src])
    def test_jwt_policy(
        self,
        kube_apis,
        crd_ingress_controller,
        backend_setup,
        ingress_controller_endpoint,
        test_namespace,
        ingress_variant,
        policy,
    ):
        """
        Test jwt-policy on standard, mergeable-master, and mergeable-minion Ingresses
        with a valid and an invalid policy.
        """
        wait_valid = policy == jwt_pol_valid_src
        target_variant = ingress_variant if wait_valid else f"{ingress_variant}-invalid-pol"

        secret_name = pol_name = None
        ingress_setup = None
        try:
            secret_name, pol_name, token_data = self.setup_single_policy(
                kube_apis,
                test_namespace,
                valid_token,
                jwk_sec_valid_src,
                policy,
                wait_valid=wait_valid,
            )
            policy_info = read_custom_resource(kube_apis.custom_objects, test_namespace, "policies", pol_name)
            if policy == jwt_pol_valid_src:
                assert (
                    policy_info["status"]
                    and policy_info["status"]["reason"] == "AddedOrUpdated"
                    and policy_info["status"]["state"] == "Valid"
                )
            elif policy == jwt_pol_invalid_src:
                assert (
                    policy_info["status"]
                    and policy_info["status"]["reason"] == "Rejected"
                    and policy_info["status"]["state"] == "Invalid"
                )
            else:
                pytest.fail("Invalid configuration")

            ingress_setup = deploy_ingress(kube_apis, ingress_controller_endpoint, test_namespace, target_variant)
            resp = requests.get(
                ingress_setup.request_url,
                headers={"host": ingress_setup.ingress_host, "token": token_data},
            )
            print(resp.status_code)

            if policy == jwt_pol_valid_src:
                assert resp.status_code == 200
                assert "Request ID:" in resp.text
            elif policy == jwt_pol_invalid_src:
                assert resp.status_code == 500
                assert "Internal Server Error" in resp.text
            else:
                pytest.fail("Not a valid case or parameter")
        finally:
            if ingress_setup is not None:
                delete_items_from_yaml(kube_apis, ingress_setup.ingress_src_file, test_namespace)
            if pol_name is not None:
                delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
            if secret_name is not None:
                delete_secret(kube_apis.v1, secret_name, test_namespace)

    def test_jwt_policy_delete_secret(
        self,
        kube_apis,
        crd_ingress_controller,
        backend_setup,
        ingress_controller_endpoint,
        test_namespace,
    ):
        """
        Test if requests result in 500 when the JWK secret is deleted.
        """
        secret_name = pol_name = None
        ingress_setup = None
        try:
            secret_name, pol_name, token_data = self.setup_single_policy(
                kube_apis,
                test_namespace,
                valid_token,
                jwk_sec_valid_src,
                jwt_pol_valid_src,
            )
            ingress_setup = deploy_ingress(kube_apis, ingress_controller_endpoint, test_namespace, "standard")
            headers = {"host": ingress_setup.ingress_host, "token": token_data}

            resp1 = requests.get(ingress_setup.request_url, headers=headers)
            print(resp1.status_code)

            delete_secret(kube_apis.v1, secret_name, test_namespace)
            secret_name = None
            wait_before_test()

            resp2 = requests.get(ingress_setup.request_url, headers=headers)
            print(resp2.status_code)

            assert resp1.status_code == 200
            assert resp2.status_code == 500
        finally:
            if ingress_setup is not None:
                delete_items_from_yaml(kube_apis, ingress_setup.ingress_src_file, test_namespace)
            if pol_name is not None:
                delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
            if secret_name is not None:
                delete_secret(kube_apis.v1, secret_name, test_namespace)

    def test_jwt_policy_delete_policy(
        self,
        kube_apis,
        crd_ingress_controller,
        backend_setup,
        ingress_controller_endpoint,
        test_namespace,
    ):
        """
        Test if requests result in 500 when the JWT policy is deleted.
        """
        secret_name = pol_name = None
        ingress_setup = None
        try:
            secret_name, pol_name, token_data = self.setup_single_policy(
                kube_apis,
                test_namespace,
                valid_token,
                jwk_sec_valid_src,
                jwt_pol_valid_src,
            )
            ingress_setup = deploy_ingress(kube_apis, ingress_controller_endpoint, test_namespace, "standard")
            headers = {"host": ingress_setup.ingress_host, "token": token_data}

            resp1 = requests.get(ingress_setup.request_url, headers=headers)
            print(resp1.status_code)

            delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
            pol_name = None
            wait_before_test()

            resp2 = requests.get(ingress_setup.request_url, headers=headers)
            print(resp2.status_code)

            assert resp1.status_code == 200
            assert resp2.status_code == 500
        finally:
            if ingress_setup is not None:
                delete_items_from_yaml(kube_apis, ingress_setup.ingress_src_file, test_namespace)
            if pol_name is not None:
                delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
            if secret_name is not None:
                delete_secret(kube_apis.v1, secret_name, test_namespace)

    def test_jwt_policy_override(
        self,
        kube_apis,
        crd_ingress_controller,
        backend_setup,
        ingress_controller_endpoint,
        test_namespace,
    ):
        """
        Test if the first JWT policy referenced in nginx.com/policies takes precedence.
        """
        secret_name = pol_name_1 = pol_name_2 = None
        ingress_setup = None
        try:
            secret_name, pol_name_1, pol_name_2, token_data = self.setup_multiple_policies(
                kube_apis,
                test_namespace,
                valid_token,
                jwk_sec_valid_src,
                jwt_pol_valid_src,
                jwt_pol_multi_src,
            )

            print("Apply Ingress with policies in order: jwt-policy-valid-multi, jwt-policy-valid")
            ingress_setup = deploy_ingress(kube_apis, ingress_controller_endpoint, test_namespace, "standard-multi-1")
            headers = {"host": ingress_setup.ingress_host, "token": token_data}

            resp1 = requests.get(ingress_setup.request_url, headers=headers)
            print(resp1.status_code)

            delete_items_from_yaml(kube_apis, ingress_setup.ingress_src_file, test_namespace)
            ingress_setup = None

            print("Apply Ingress with policies in order: jwt-policy-valid, jwt-policy-valid-multi")
            ingress_setup = deploy_ingress(kube_apis, ingress_controller_endpoint, test_namespace, "standard-multi-2")
            resp2 = requests.get(ingress_setup.request_url, headers=headers)
            print(resp2.status_code)

            assert resp1.status_code == 401  # 401 unauthorized, since jwt-policy-valid-multi expects Bearer token
            assert resp2.status_code == 200
        finally:
            if ingress_setup is not None:
                delete_items_from_yaml(kube_apis, ingress_setup.ingress_src_file, test_namespace)
            if pol_name_1 is not None:
                delete_policy(kube_apis.custom_objects, pol_name_1, test_namespace)
            if pol_name_2 is not None:
                delete_policy(kube_apis.custom_objects, pol_name_2, test_namespace)
            if secret_name is not None:
                delete_secret(kube_apis.v1, secret_name, test_namespace)

    def test_jwt_policy_master_vs_minion_override(
        self,
        kube_apis,
        crd_ingress_controller,
        backend_setup,
        ingress_controller_endpoint,
        test_namespace,
    ):
        """
        Test if a JWT policy referenced on a minion Ingress overrides a JWT policy on the master Ingress.
        """
        secret_name = pol_name_1 = pol_name_2 = None
        ingress_setup = None
        try:
            secret_name, pol_name_1, pol_name_2, token_data = self.setup_multiple_policies(
                kube_apis,
                test_namespace,
                valid_token,
                jwk_sec_valid_src,
                jwt_pol_valid_src,
                jwt_pol_multi_src,
            )

            print("Apply mergeable Ingress with jwt-policy-valid on master and jwt-policy-valid-multi on minion")
            ingress_setup = deploy_ingress(
                kube_apis, ingress_controller_endpoint, test_namespace, "mergeable-override-1"
            )
            headers = {"host": ingress_setup.ingress_host, "token": token_data}

            resp1 = requests.get(ingress_setup.request_url, headers=headers)
            print(resp1.status_code)

            delete_items_from_yaml(kube_apis, ingress_setup.ingress_src_file, test_namespace)
            ingress_setup = None

            print("Apply mergeable Ingress with jwt-policy-valid-multi on master and jwt-policy-valid on minion")
            ingress_setup = deploy_ingress(
                kube_apis, ingress_controller_endpoint, test_namespace, "mergeable-override-2"
            )
            resp2 = requests.get(ingress_setup.request_url, headers=headers)
            print(resp2.status_code)

            assert resp1.status_code == 401  # 401 unauthorized, since minion policy expects Bearer token
            assert resp2.status_code == 200
        finally:
            if ingress_setup is not None:
                delete_items_from_yaml(kube_apis, ingress_setup.ingress_src_file, test_namespace)
            if pol_name_1 is not None:
                delete_policy(kube_apis.custom_objects, pol_name_1, test_namespace)
            if pol_name_2 is not None:
                delete_policy(kube_apis.custom_objects, pol_name_2, test_namespace)
            if secret_name is not None:
                delete_secret(kube_apis.v1, secret_name, test_namespace)

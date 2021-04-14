import pytest, requests, time
from kubernetes.client.rest import ApiException
from suite.resources_utils import (
    wait_before_test,
    replace_configmap_from_yaml,
    create_secret_from_yaml,
    delete_secret,
    replace_secret,
)
from suite.ssl_utils import get_server_certificate_subject, create_sni_session
from suite.custom_resources_utils import (
    read_custom_resource,
    delete_virtual_server,
    create_virtual_server_from_yaml,
    patch_virtual_server_from_yaml,
    delete_and_create_vs_from_yaml,
    create_policy_from_yaml,
    delete_policy,
    read_policy,
)
from settings import TEST_DATA, DEPLOYMENTS

std_vs_src = f"{TEST_DATA}/ingress-mtls/standard/virtual-server.yaml"
mtls_sec_valid_src = f"{TEST_DATA}/ingress-mtls/secret/ingress-mtls-secret.yaml"
mtls_sec_invalid_src = f"{TEST_DATA}/ingress-mtls/secret/ingress-mtls-secret.yaml"
tls_sec_valid_src = f"{TEST_DATA}/ingress-mtls/secret/tls-secret.yaml"
mtls_pol_valid_src = f"{TEST_DATA}/ingress-mtls/policies/ingress-mtls.yaml"
mtls_vs_src = f"{TEST_DATA}/ingress-mtls/spec/virtual-server-mtls.yaml"
crt = f"{TEST_DATA}/ingress-mtls/client-auth/client-cert.pem"
key = f"{TEST_DATA}/ingress-mtls/client-auth/client-key.pem"

@pytest.mark.imtls
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [f"-enable-custom-resources", f"-enable-leader-election=false", f"-enable-preview-policies"],
            },
            {"example": "ingress-mtls", "app_type": "simple",},
        )
    ],
    indirect=True,
)
class TestIngressMtlsPolicies:
    def setup_single_policy(self, kube_apis, test_namespace, mtls_secret, tls_secret, policy):
        print(f"Create ingress-mtls secret")
        mtls_secret_name = create_secret_from_yaml(kube_apis.v1, test_namespace, mtls_secret)

        print(f"Create ingress-mtls policy")
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, policy, test_namespace)

        print(f"Create tls secret")
        tls_secret_name = create_secret_from_yaml(kube_apis.v1, test_namespace, tls_secret)

        return mtls_secret_name, tls_secret_name, pol_name

    def test_ingress_mtls_policy_token(
        self, kube_apis, crd_ingress_controller, virtual_server_setup, test_namespace,
    ):
        """
            Test ingress-mtls with no token, valid token and invalid token
        """
        session = create_sni_session()
        mtls_secret, tls_secret, pol_name = self.setup_single_policy(
            kube_apis, test_namespace, mtls_sec_valid_src, tls_sec_valid_src, mtls_pol_valid_src,
        )

        print(f"Patch vs with policy: {mtls_pol_valid_src}")
        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            mtls_vs_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()
        print(virtual_server_setup.vs_host)
        print(get_server_certificate_subject(virtual_server_setup.public_endpoint.public_ip,
                                virtual_server_setup.vs_host,
                                virtual_server_setup.public_endpoint.port_ssl))
        resp1 = session.get(
            virtual_server_setup.backend_1_url_ssl,
            cert = (crt, key),
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects = False,
            verify = False,
        )
        print(resp1.status_code)
        print(resp1.text)
        # wait_before_test(60)
        delete_secret(kube_apis.v1, tls_secret, test_namespace)
        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        delete_secret(kube_apis.v1, mtls_secret, test_namespace)

        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            std_vs_src,
            virtual_server_setup.namespace,
        )

        assert resp1.status_code == 200
        # assert f"400 No required SSL certificate was sent" in resp1.text

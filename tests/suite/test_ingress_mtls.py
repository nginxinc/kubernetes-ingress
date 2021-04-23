import pytest, requests, time
from ssl import SSLError
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
tls_sec_valid_src = f"{TEST_DATA}/ingress-mtls/secret/tls-secret.yaml"
mtls_pol_valid_src = f"{TEST_DATA}/ingress-mtls/policies/ingress-mtls.yaml"
mtls_pol_invalid_src = f"{TEST_DATA}/ingress-mtls/policies/ingress-mtls-invalid.yaml"
mtls_vs_src = f"{TEST_DATA}/ingress-mtls/spec/virtual-server-mtls.yaml"
mtls_vs_invalid_pol_src = f"{TEST_DATA}/ingress-mtls/spec/virtual-server-mtls-invalid-pol.yaml"
crt = f"{TEST_DATA}/ingress-mtls/client-auth/valid/client-cert.pem"
key = f"{TEST_DATA}/ingress-mtls/client-auth/valid/client-key.pem"
invalid_crt = f"{TEST_DATA}/ingress-mtls/client-auth/invalid/client-cert.pem"
invalid_key = f"{TEST_DATA}/ingress-mtls/client-auth/invalid/client-cert.pem"


@pytest.mark.policies
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    f"-enable-custom-resources",
                    f"-enable-leader-election=false",
                    f"-enable-preview-policies",
                ],
            },
            {
                "example": "ingress-mtls",
                "app_type": "simple",
            },
        )
    ],
    indirect=True,
)
class TestIngressMtlsPolicies:
    def setup_policy(self, kube_apis, test_namespace, mtls_secret, tls_secret, policy):
        print(f"Create ingress-mtls secret")
        mtls_secret_name = create_secret_from_yaml(kube_apis.v1, test_namespace, mtls_secret)

        print(f"Create ingress-mtls policy")
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, policy, test_namespace)

        print(f"Create tls secret")
        tls_secret_name = create_secret_from_yaml(kube_apis.v1, test_namespace, tls_secret)
        return mtls_secret_name, tls_secret_name, pol_name
    
    def teardown_policy(self, kube_apis, test_namespace, tls_secret, pol_name, mtls_secret):

        print("Delete policy and related secrets")
        delete_secret(kube_apis.v1, tls_secret, test_namespace)
        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        delete_secret(kube_apis.v1, mtls_secret, test_namespace)

    @pytest.mark.parametrize("policy_src", [mtls_pol_valid_src, mtls_pol_invalid_src])
    def test_ingress_mtls_policy(
        self,
        kube_apis,
        crd_ingress_controller,
        virtual_server_setup,
        test_namespace,
        policy_src,
    ):
        """
        Test ingress-mtls with valid and invalid policy
        """
        session = create_sni_session()
        mtls_secret, tls_secret, pol_name = self.setup_policy(
            kube_apis,
            test_namespace,
            mtls_sec_valid_src,
            tls_sec_valid_src,
            policy_src,
        )

        print(f"Patch vs with policy: {policy_src}")
        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            mtls_vs_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()
        resp = session.get(
            virtual_server_setup.backend_1_url_ssl,
            cert=(crt, key),
            headers={"host": virtual_server_setup.vs_host},
            allow_redirects=False,
            verify=False,
        )

        self.teardown_policy(kube_apis, test_namespace, tls_secret, pol_name, mtls_secret)

        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            std_vs_src,
            virtual_server_setup.namespace,
        )
        if policy_src == mtls_pol_valid_src:
            assert resp.status_code == 200
            assert "Server address:" in resp.text and "Server name:" in resp.text
        elif policy_src == mtls_pol_invalid_src:
            assert resp.status_code == 500
        else:
            pytest.fail(f"Invalid parameter")

    @pytest.mark.test
    @pytest.mark.parametrize("certificate", [(crt, key), "",(invalid_crt, invalid_key)])
    def test_ingress_mtls_policy_cert(
        self,
        kube_apis,
        crd_ingress_controller,
        virtual_server_setup,
        test_namespace,
        certificate,
    ):
        """
        Test ingress-mtls with valid and invalid policy
        """
        session = create_sni_session()
        mtls_secret, tls_secret, pol_name = self.setup_policy(
            kube_apis,
            test_namespace,
            mtls_sec_valid_src,
            tls_sec_valid_src,
            mtls_pol_valid_src,
        )

        print(f"Patch vs with policy: {mtls_pol_valid_src}")
        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            mtls_vs_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()
        try :
            resp = session.get(
                virtual_server_setup.backend_1_url_ssl,
                cert=certificate,
                headers={"host": virtual_server_setup.vs_host},
                allow_redirects=False,
                verify=False,
            )
        except SSLError as e:
            exception = e
            print("SSL Error occured")
        print(resp.text)

        self.teardown_policy(kube_apis, test_namespace, tls_secret, pol_name, mtls_secret)

        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            std_vs_src,
            virtual_server_setup.namespace,
        )
        if certificate == (crt, key):
            assert resp.status_code == 200
            assert "Server address:" in resp.text and "Server name:" in resp.text
        elif certificate == (invalid_crt, invalid_key):
            assert "SSL" in exception.library
        else:
            assert resp.status_code == 400
            assert "400 No required SSL certificate was sent" in resp.text
import pytest, requests
from kubernetes.client.rest import ApiException
from suite.resources_utils import wait_before_test, replace_configmap_from_yaml
from suite.custom_resources_utils import (
    read_crd,
    delete_virtual_server,
    create_virtual_server_from_yaml,
    patch_virtual_server_from_yaml,
    create_policy_from_yaml,
    delete_policy,
    read_policy,
)
from settings import TEST_DATA, DEPLOYMENTS

std_cm_src = f"{DEPLOYMENTS}/common/nginx-config.yaml"
test_cm_src = f"{TEST_DATA}/access-control/configmap/nginx-config.yaml"
std_vs_src = f"{TEST_DATA}/access-control/standard/virtual-server.yaml"
deny_pol_src = f"{TEST_DATA}/access-control/access-control-policy-deny.yaml"
deny_vs_src = f"{TEST_DATA}/access-control/virtual-server-deny.yaml"
allow_pol_src = f"{TEST_DATA}/access-control/access-control-policy-allow.yaml"
allow_vs_src = f"{TEST_DATA}/access-control/virtual-server-allow.yaml"
override_vs_src = f"{TEST_DATA}/access-control/virtual-server-override.yaml"
invalid_pol_src = f"{TEST_DATA}/access-control/access-control-policy-invalid.yaml"
invalid_vs_src = f"{TEST_DATA}/access-control/virtual-server-invalid.yaml"


@pytest.fixture(scope="class")
def config_setup(request, kube_apis, ingress_controller_prerequisites) -> None:
    """
    Replace configmap to add "set-real-ip-from"
    :param request: pytest fixture
    :param kube_apis: client apis
    :param ingress_controller_prerequisites: IC pre-requisites
    """
    print(f"------------- Replace ConfigMap --------------")
    replace_configmap_from_yaml(
        kube_apis.v1,
        ingress_controller_prerequisites.config_map["metadata"]["name"],
        ingress_controller_prerequisites.namespace,
        test_cm_src,
    )

    def fin():
        print(f"------------- Restore ConfigMap --------------")
        replace_configmap_from_yaml(
        kube_apis.v1,
        ingress_controller_prerequisites.config_map["metadata"]["name"],
        ingress_controller_prerequisites.namespace,
        std_cm_src,
    )
    request.addfinalizer(fin)


@pytest.mark.policies
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [f"-enable-custom-resources", f"-enable-leader-election=false"],
            },
            {"example": "access-control", "app_type": "simple",},
        )
    ],
    indirect=True,
)
class TestAccessControlPolicies:
    def restore_default_vs(self, kube_apis, virtual_server_setup) -> None:
        """
        Restore VirtualServer without policy spec
        """
        delete_virtual_server(
            kube_apis.custom_objects, virtual_server_setup.vs_name, virtual_server_setup.namespace
        )
        create_virtual_server_from_yaml(
            kube_apis.custom_objects, std_vs_src, virtual_server_setup.namespace
        )
        wait_before_test()

    @pytest.mark.smoke
    def test_deny_policy(
        self, kube_apis, crd_ingress_controller, virtual_server_setup, test_namespace, config_setup
    ):
        """
        Test if ip (10.0.0.1) block-listing is working: default(no policy) -> deny
        """
        resp = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"Response: {resp.status_code}\n{resp.text}")
        assert resp.status_code == 200

        print(f"Create deny policy")
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, deny_pol_src, test_namespace)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            deny_vs_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        print(f"\nUse IP listed in deny block: 10.0.0.1")
        resp1 = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"Response: {resp1.status_code}\n{resp1.text}")
        print(f"\nUse IP not listed in deny block: 10.0.0.2")
        resp2 = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.2"},
        )
        print(f"Response: {resp2.status_code}\n{resp2.text}")

        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)

        assert (
            resp1.status_code == 403
            and "403 Forbidden" in resp1.text
            and resp2.status_code == 200
            and "Server address:" in resp2.text
        )

    @pytest.mark.smoke
    def test_allow_policy(
        self, kube_apis, crd_ingress_controller, virtual_server_setup, test_namespace, config_setup
    ):
        """
        Test if ip (10.0.0.1) allow-listing is working: default(no policy) -> allow
        """
        resp = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"Response: {resp.status_code}\n{resp.text}")
        assert resp.status_code == 200

        print(f"Create allow policy")
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, allow_pol_src, test_namespace)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            allow_vs_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        print(f"\nUse IP listed in allow block: 10.0.0.1")
        resp1 = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"\nUse IP listed not in allow block: 10.0.0.2")
        print(f"Response: {resp1.status_code}\n{resp1.text}")
        resp2 = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.2"},
        )
        print(f"Response: {resp2.status_code}\n{resp2.text}")

        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)

        assert (
            resp1.status_code == 200
            and "Server address:" in resp1.text
            and resp2.status_code == 403
            and "403 Forbidden" in resp2.text
        )

    def test_override_policy(
        self, kube_apis, crd_ingress_controller, virtual_server_setup, test_namespace, config_setup
    ):
        """
        Test if ip allow-listing overrides block-listing: default(no policy) -> deny and allow
        """
        resp = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"Response: {resp.status_code}\n{resp.text}")
        assert resp.status_code == 200

        print(f"Create deny policy")
        deny_pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, deny_pol_src, test_namespace
        )
        print(f"Create allow policy")
        allow_pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, allow_pol_src, test_namespace
        )
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            override_vs_src,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        print(f"Use IP listed in both deny and allow policies: 10.0.0.1")
        resp = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"Response: {resp.status_code}\n{resp.text}")

        delete_policy(kube_apis.custom_objects, deny_pol_name, test_namespace)
        delete_policy(kube_apis.custom_objects, allow_pol_name, test_namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)

        assert resp.status_code == 200 and "Server address:" in resp.text

    def test_invalid_policy(
        self, kube_apis, crd_ingress_controller, virtual_server_setup, test_namespace, config_setup
    ):
        """
        Test if invalid policy is applied then response is 500
        """
        resp = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"Response: {resp.status_code}\n{resp.text}")
        assert resp.status_code == 200

        print(f"Create invalid policy")
        invalid_pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, invalid_pol_src, test_namespace
        )
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            invalid_vs_src,
            virtual_server_setup.namespace,
        )

        wait_before_test()
        resp = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"Response: {resp.status_code}\n{resp.text}")

        vs_info = read_crd(
            kube_apis.custom_objects,
            virtual_server_setup.namespace,
            "virtualservers",
            virtual_server_setup.vs_name,
        )
        delete_policy(kube_apis.custom_objects, invalid_pol_name, test_namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)

        assert resp.status_code == 500 and "500 Internal Server Error" in resp.text
        assert (
            vs_info["status"]["state"] == "Warning"
            and vs_info["status"]["reason"] == "AddedOrUpdatedWithWarning"
        )

    def test_deleted_policy(
        self, kube_apis, crd_ingress_controller, virtual_server_setup, test_namespace, config_setup
    ):
        """
        Test if valid policy is deleted then response is 500
        """
        resp = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"Response: {resp.status_code}\n{resp.text}")
        assert resp.status_code == 200

        print(f"Create deny policy")
        
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            deny_vs_src,
            virtual_server_setup.namespace,
        )
        pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, deny_pol_src, test_namespace
        )
        wait_before_test()
        vs_info = read_crd(
            kube_apis.custom_objects,
            virtual_server_setup.namespace,
            "virtualservers",
            virtual_server_setup.vs_name,
        )
        assert vs_info["status"]["state"] == "Valid"
        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)

        wait_before_test()
        resp = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "X-Real-IP": "10.0.0.1"},
        )
        print(f"Response: {resp.status_code}\n{resp.text}")

        vs_info = read_crd(
            kube_apis.custom_objects,
            virtual_server_setup.namespace,
            "virtualservers",
            virtual_server_setup.vs_name,
        )
        self.restore_default_vs(kube_apis, virtual_server_setup)

        assert resp.status_code == 500 and "500 Internal Server Error" in resp.text
        assert (
            vs_info["status"]["state"] == "Warning"
            and vs_info["status"]["reason"] == "UpdatedWithWarning"
        )

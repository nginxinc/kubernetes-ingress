import pytest
import requests
import time
from suite.resources_utils import (
    wait_before_test,
)
from suite.custom_resources_utils import (
    read_custom_resource,
    delete_virtual_server,
    create_virtual_server_from_yaml,
    patch_virtual_server_from_yaml,
    create_policy_from_yaml,
    delete_policy,
)
from settings import TEST_DATA

vs_src = f"{TEST_DATA}/policy-ingress-class/virtual-server.yaml"
vs_policy_src = f"{TEST_DATA}/policy-ingress-class/virtual-server-policy.yaml"

policy_src = f"{TEST_DATA}/policy-ingress-class/policy.yaml"
policy_ingress_class_src = f"{TEST_DATA}/policy-ingress-class/policy-ingress-class.yaml"
policy_other_ingress_class_src = f"{TEST_DATA}/policy-ingress-class/policy-other-ingress-class.yaml"


@pytest.mark.sean
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    "-ingress-class=nginx",
                    f"-enable-custom-resources",
                    f"-enable-preview-policies",
                    f"-enable-leader-election=false",
                ],
            },
            {"example": "rate-limit", "app_type": "simple",},
        )
    ],
    indirect=True,
)
class TestRateLimitingPolicies:
    def restore_default_vs(self, kube_apis, virtual_server_setup) -> None:
        """
        Restore VirtualServer without policy spec
        """
        delete_virtual_server(
            kube_apis.custom_objects, virtual_server_setup.vs_name, virtual_server_setup.namespace
        )
        create_virtual_server_from_yaml(
            kube_apis.custom_objects, vs_src, virtual_server_setup.namespace
        )
        wait_before_test()

    @pytest.mark.parametrize("src", [vs_policy_src])
    def test_policy_empty_ingress_class(
        self, kube_apis, crd_ingress_controller, virtual_server_setup, test_namespace, src,
    ):
        """
        Test if rate-limiting policy is working with 1 rps
        """
        print(f"Create rl policy")
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, policy_src, test_namespace)
        print(f"Patch vs with policy: {src}")
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            src,
            virtual_server_setup.namespace,
        )

        wait_before_test()
        policy_info = read_custom_resource(kube_apis.custom_objects, test_namespace, "policies", pol_name)
        occur = []
        t_end = time.perf_counter() + 1
        resp = requests.get(
            virtual_server_setup.backend_1_url, headers={"host": virtual_server_setup.vs_host},
        )
        print(resp.status_code)
        assert resp.status_code == 200
        while time.perf_counter() < t_end:
            resp = requests.get(
                virtual_server_setup.backend_1_url, headers={"host": virtual_server_setup.vs_host},
            )
            occur.append(resp.status_code)
        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        assert (
            policy_info["status"]
            and policy_info["status"]["reason"] == "AddedOrUpdated"
            and policy_info["status"]["state"] == "Valid"
        )
        assert occur.count(200) <= 1

    @pytest.mark.parametrize("src", [vs_policy_src])
    def test_policy_matching_ingress_class(
            self, kube_apis, crd_ingress_controller, virtual_server_setup, test_namespace, src,
    ):
        """
        Test if rate-limiting policy is working with 1 rps
        """
        print(f"Create rl policy")
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, policy_ingress_class_src, test_namespace)
        print(f"Patch vs with policy: {src}")
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            src,
            virtual_server_setup.namespace,
        )

        wait_before_test()
        policy_info = read_custom_resource(kube_apis.custom_objects, test_namespace, "policies", pol_name)
        occur = []
        t_end = time.perf_counter() + 1
        resp = requests.get(
            virtual_server_setup.backend_1_url, headers={"host": virtual_server_setup.vs_host},
        )
        print(resp.status_code)
        assert resp.status_code == 200
        while time.perf_counter() < t_end:
            resp = requests.get(
                virtual_server_setup.backend_1_url, headers={"host": virtual_server_setup.vs_host},
            )
            occur.append(resp.status_code)
        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        assert (
                policy_info["status"]
                and policy_info["status"]["reason"] == "AddedOrUpdated"
                and policy_info["status"]["state"] == "Valid"
        )
        assert occur.count(200) <= 1

    @pytest.mark.parametrize("src", [vs_policy_src])
    def test_policy_non_matching_ingress_class(
            self, kube_apis, crd_ingress_controller, virtual_server_setup, test_namespace, src,
    ):
        """
        Test if rate-limiting policy is working with 1 rps
        """
        print(f"Create rl policy")
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, policy_other_ingress_class_src, test_namespace)
        print(f"Patch vs with policy: {src}")
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            src,
            virtual_server_setup.namespace,
        )

        wait_before_test()
        policy_info = read_custom_resource(kube_apis.custom_objects, test_namespace, "policies", pol_name)

        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        self.restore_default_vs(kube_apis, virtual_server_setup)
        assert (
                "status" not in policy_info
        )


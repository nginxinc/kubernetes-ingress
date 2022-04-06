import pytest, requests, time
from kubernetes.client.rest import ApiException
from suite.resources_utils import wait_before_test, replace_configmap_from_yaml
from suite.custom_resources_utils import (
    read_custom_resource,
)
from suite.vs_vsr_resources_utils import (
    delete_virtual_server,
    create_virtual_server_from_yaml,
    patch_virtual_server_from_yaml,
    patch_v_s_route_from_yaml,
)
from suite.policy_resources_utils import (
    create_policy_from_yaml,
    delete_policy,
    read_policy,
)
from settings import TEST_DATA, DEPLOYMENTS

std_vs_src = f"{TEST_DATA}/virtual-server-route/standard/virtual-server.yaml"
rl_pol_pri_src = f"{TEST_DATA}/rate-limit/policies/rate-limit-primary.yaml"
rl_vsr_pri_src = f"{TEST_DATA}/rate-limit/route-subroute/virtual-server-route-pri-subroute.yaml"
rl_pol_sec_src = f"{TEST_DATA}/rate-limit/policies/rate-limit-secondary.yaml"
rl_vsr_sec_src = f"{TEST_DATA}/rate-limit/route-subroute/virtual-server-route-sec-subroute.yaml"
rl_pol_invalid_src = f"{TEST_DATA}/rate-limit/policies/rate-limit-invalid.yaml"
rl_vsr_invalid_src = (
    f"{TEST_DATA}/rate-limit/route-subroute/virtual-server-route-invalid-subroute.yaml"
)
rl_vsr_override_src = (
    f"{TEST_DATA}/rate-limit/route-subroute/virtual-server-route-override-subroute.yaml"
)
rl_vsr_override_vs_spec_src = (
    f"{TEST_DATA}/rate-limit/route-subroute/virtual-server-vsr-spec-override.yaml"
)
rl_vsr_override_vs_route_src = (
    f"{TEST_DATA}/rate-limit/route-subroute/virtual-server-vsr-route-override.yaml"
)


@pytest.mark.policies
@pytest.mark.parametrize(
    "crd_ingress_controller, v_s_route_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    f"-enable-custom-resources",
                    f"-enable-leader-election=false",
                ],
            },
            {"example": "virtual-server-route"},
        )
    ],
    indirect=True,
)
class TestRateLimitingPoliciesVsr:
    def restore_default_vsr(self, kube_apis, v_s_route_setup) -> None:
        """
        Function to revert vsr deployments to valid state
        """
        patch_src_m = f"{TEST_DATA}/virtual-server-route/route-multiple.yaml"
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            patch_src_m,
            v_s_route_setup.route_m.namespace,
        )
        wait_before_test()

    @pytest.mark.smoke
    @pytest.mark.parametrize("src", [rl_vsr_pri_src])
    def test_rl_policy_1rs_vsr(
        self,
        kube_apis,
        crd_ingress_controller,
        v_s_route_app_setup,
        v_s_route_setup,
        test_namespace,
        src,
    ):
        """
        Test if rate-limiting policy is working with ~1 rps in vsr:subroute
        """

        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        print(f"Create rl policy")
        pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, rl_pol_pri_src, v_s_route_setup.route_m.namespace
        )
        print(f"Patch vsr with policy: {src}")
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            src,
            v_s_route_setup.route_m.namespace,
        )

        wait_before_test()
        policy_info = read_custom_resource(
            kube_apis.custom_objects, v_s_route_setup.route_m.namespace, "policies", pol_name
        )
        occur = []
        t_end = time.perf_counter() + 1
        resp = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host},
        )
        print(resp.status_code)
        assert resp.status_code == 200
        while time.perf_counter() < t_end:
            resp = requests.get(
                f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                headers={"host": v_s_route_setup.vs_host},
            )
            occur.append(resp.status_code)
        delete_policy(kube_apis.custom_objects, pol_name, v_s_route_setup.route_m.namespace)
        self.restore_default_vsr(kube_apis, v_s_route_setup)
        assert (
            policy_info["status"]
            and policy_info["status"]["reason"] == "AddedOrUpdated"
            and policy_info["status"]["state"] == "Valid"
        )
        assert occur.count(200) <= 1

    @pytest.mark.parametrize("src", [rl_vsr_sec_src])
    def test_rl_policy_5rs_vsr(
        self,
        kube_apis,
        crd_ingress_controller,
        v_s_route_app_setup,
        v_s_route_setup,
        test_namespace,
        src,
    ):
        """
        Test if rate-limiting policy is working with ~5 rps in vsr:subroute
        """
        rate_sec = 5
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        print(f"Create rl policy")
        pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, rl_pol_sec_src, v_s_route_setup.route_m.namespace
        )
        print(f"Patch vsr with policy: {src}")
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            src,
            v_s_route_setup.route_m.namespace,
        )

        wait_before_test()
        policy_info = read_custom_resource(
            kube_apis.custom_objects, v_s_route_setup.route_m.namespace, "policies", pol_name
        )
        occur = []
        t_end = time.perf_counter() + 1
        resp = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host},
        )
        print(resp.status_code)
        assert resp.status_code == 200
        while time.perf_counter() < t_end:
            resp = requests.get(
                f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                headers={"host": v_s_route_setup.vs_host},
            )
            occur.append(resp.status_code)
        delete_policy(kube_apis.custom_objects, pol_name, v_s_route_setup.route_m.namespace)
        self.restore_default_vsr(kube_apis, v_s_route_setup)
        assert (
            policy_info["status"]
            and policy_info["status"]["reason"] == "AddedOrUpdated"
            and policy_info["status"]["state"] == "Valid"
        )
        assert rate_sec >= occur.count(200) >= (rate_sec - 2)

    @pytest.mark.parametrize("src", [rl_vsr_override_src])
    def test_rl_policy_override_vsr(
        self,
        kube_apis,
        crd_ingress_controller,
        v_s_route_app_setup,
        v_s_route_setup,
        test_namespace,
        src,
    ):
        """
        Test if rate-limiting policy with lower rps is used when multiple policies are listed in vsr:subroute
        And test if the order of policies in vsr:subroute has no effect
        """

        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        print(f"Create rl policy: 1rps")
        pol_name_pri = create_policy_from_yaml(
            kube_apis.custom_objects, rl_pol_pri_src, v_s_route_setup.route_m.namespace
        )
        print(f"Create rl policy: 5rps")
        pol_name_sec = create_policy_from_yaml(
            kube_apis.custom_objects, rl_pol_sec_src, v_s_route_setup.route_m.namespace
        )
        print(f"Patch vsr with policy: {src}")
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            src,
            v_s_route_setup.route_m.namespace,
        )
        wait_before_test()
        occur = []
        t_end = time.perf_counter() + 1
        resp = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host},
        )
        print(resp.status_code)
        assert resp.status_code == 200
        while time.perf_counter() < t_end:
            resp = requests.get(
                f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                headers={"host": v_s_route_setup.vs_host},
            )
            occur.append(resp.status_code)
        delete_policy(kube_apis.custom_objects, pol_name_pri, v_s_route_setup.route_m.namespace)
        delete_policy(kube_apis.custom_objects, pol_name_sec, v_s_route_setup.route_m.namespace)
        self.restore_default_vsr(kube_apis, v_s_route_setup)
        assert occur.count(200) <= 1

    @pytest.mark.parametrize("src", [rl_vsr_pri_src])
    def test_rl_policy_deleted_vsr(
        self,
        kube_apis,
        crd_ingress_controller,
        v_s_route_app_setup,
        v_s_route_setup,
        test_namespace,
        src,
    ):
        """
        Test if deleting a policy results in 500
        """
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        print(f"Create rl policy")
        pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, rl_pol_pri_src, v_s_route_setup.route_m.namespace
        )
        print(f"Patch vsr with policy: {src}")
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            src,
            v_s_route_setup.route_m.namespace,
        )
        wait_before_test()
        resp = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host},
        )
        assert resp.status_code == 200
        print(resp.status_code)
        delete_policy(kube_apis.custom_objects, pol_name, v_s_route_setup.route_m.namespace)
        resp = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host},
        )
        self.restore_default_vsr(kube_apis, v_s_route_setup)
        assert resp.status_code == 500

    @pytest.mark.parametrize("src", [rl_vsr_invalid_src])
    def test_rl_policy_invalid_vsr(
        self,
        kube_apis,
        crd_ingress_controller,
        v_s_route_app_setup,
        v_s_route_setup,
        test_namespace,
        src,
    ):
        """
        Test if using an invalid policy in vsr:subroute results in 500
        """
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        print(f"Create rl policy")
        invalid_pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, rl_pol_invalid_src, v_s_route_setup.route_m.namespace
        )
        print(f"Patch vsr with policy: {src}")
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            src,
            v_s_route_setup.route_m.namespace,
        )

        wait_before_test()
        policy_info = read_custom_resource(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.namespace,
            "policies",
            invalid_pol_name,
        )
        resp = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host},
        )
        print(resp.status_code)
        delete_policy(kube_apis.custom_objects, invalid_pol_name, v_s_route_setup.route_m.namespace)
        self.restore_default_vsr(kube_apis, v_s_route_setup)
        assert (
            policy_info["status"]
            and policy_info["status"]["reason"] == "Rejected"
            and policy_info["status"]["state"] == "Invalid"
        )
        assert resp.status_code == 500

    @pytest.mark.parametrize("src", [rl_vsr_override_vs_spec_src, rl_vsr_override_vs_route_src])
    def test_override_vs_vsr(
        self,
        kube_apis,
        crd_ingress_controller,
        v_s_route_app_setup,
        test_namespace,
        v_s_route_setup,
        src,
    ):
        """
        Test if vsr subroute policy overrides vs spec policy 
        And vsr subroute policy overrides vs route policy
        """
        rate_sec = 5
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"

        # policy for virtualserver
        print(f"Create rl policy: 1rps")
        pol_name_vs = create_policy_from_yaml(
            kube_apis.custom_objects, rl_pol_pri_src, v_s_route_setup.route_m.namespace
        )
        # policy for virtualserverroute
        print(f"Create rl policy: 5rps")
        pol_name_vsr = create_policy_from_yaml(
            kube_apis.custom_objects, rl_pol_sec_src, v_s_route_setup.route_m.namespace
        )

        # patch vsr with 5rps policy
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            rl_vsr_sec_src,
            v_s_route_setup.route_m.namespace,
        )
        # patch vs with 1rps policy
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects, v_s_route_setup.vs_name, src, v_s_route_setup.namespace
        )
        wait_before_test()
        occur = []
        t_end = time.perf_counter() + 1
        resp = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host},
        )
        print(resp.status_code)
        assert resp.status_code == 200
        while time.perf_counter() < t_end:
            resp = requests.get(
                f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                headers={"host": v_s_route_setup.vs_host},
            )
            occur.append(resp.status_code)

        delete_policy(kube_apis.custom_objects, pol_name_vs, v_s_route_setup.route_m.namespace)
        delete_policy(kube_apis.custom_objects, pol_name_vsr, v_s_route_setup.route_m.namespace)
        self.restore_default_vsr(kube_apis, v_s_route_setup)
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects, v_s_route_setup.vs_name, std_vs_src, v_s_route_setup.namespace
        )
        assert rate_sec >= occur.count(200) >= (rate_sec - 2)

import time
from unittest import mock

import pytest
import requests
from settings import TEST_DATA
from suite.utils.policy_resources_utils import create_policy_from_yaml, delete_policy
from suite.utils.resources_utils import replace_configmap_from_yaml, wait_before_test
from suite.utils.vs_vsr_resources_utils import patch_v_s_route_from_yaml

std_vsr_src = f"{TEST_DATA}/virtual-server-route/route-multiple.yaml"
jwt_pol_valid_src = f"{TEST_DATA}/jwt-policy-jwksuri/policies/jwt-policy-valid.yaml"
jwt_pol_invalid_src = f"{TEST_DATA}/jwt-policy-jwksuri/policies/jwt-policy-invalid.yaml"
jwt_vsr_subroute_src = f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server-route/virtual-server-route-policy-subroute.yaml"
jwt_vsr_invalid_pol_subroute_src = (
    f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server-route/virtual-server-route-invalid-policy-subroute.yaml"
)
jwt_cm_src = f"{TEST_DATA}/jwt-policy-jwksuri/configmap/nginx-config.yaml"
ad_tenant = "dd3dfd2f-6a3b-40d1-9be0-bf8327d81c50"
client_id = "8a172a83-a630-41a4-9ca6-1e5ef03cd7e7"


def get_token(request):
    """
    get jwt token from azure ad endpoint
    """
    data = {
        "client_id": f"{client_id}",
        "scope": ".default",
        "client_secret": request.config.getoption("--ad-secret"),
        "grant_type": "client_credentials",
    }
    ad_response = requests.get(
        f"https://login.microsoftonline.com/{ad_tenant}/oauth2/token",
        data=data,
        timeout=5,
        headers={"User-Agent": "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:47.0) Gecko/20100101 Chrome/76.0.3809.100"},
    )

    if ad_response.status_code == 200:
        return ad_response.json()["access_token"]
    pytest.fail("Unable to request Azure token endpoint")


@pytest.mark.skip_for_nginx_oss
@pytest.mark.policies
@pytest.mark.skip(reason="issues with IdP communication")
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
            {
                "example": "virtual-server-route",
            },
        )
    ],
    indirect=True,
)
class TestJWTPoliciesVSRJwksuri:
    @pytest.mark.parametrize("jwt_virtual_server_route", [jwt_vsr_subroute_src])
    def test_jwt_policy_jwksuri(
        self,
        request,
        kube_apis,
        ingress_controller_prerequisites,
        crd_ingress_controller,
        v_s_route_app_setup,
        v_s_route_setup,
        test_namespace,
        jwt_virtual_server_route,
    ):
        """
        Test jwt-policy in Virtual Server Route with keys fetched form Azure
        """
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        replace_configmap_from_yaml(
            kube_apis.v1,
            ingress_controller_prerequisites.config_map["metadata"]["name"],
            ingress_controller_prerequisites.namespace,
            jwt_cm_src,
        )
        pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, jwt_pol_valid_src, v_s_route_setup.route_m.namespace
        )
        wait_before_test()

        print(f"Patch vsr with policy: {jwt_virtual_server_route}")
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            jwt_virtual_server_route,
            v_s_route_setup.route_m.namespace,
        )
        resp_no_token = mock.Mock()
        resp_no_token.status_code == 502
        counter = 0

        while resp_no_token.status_code != 401 and counter < 20:
            resp_no_token = requests.get(
                f"{req_url}{v_s_route_setup.route_m.paths[0]}",
                headers={"host": v_s_route_setup.vs_host},
            )
            wait_before_test()
            counter += 1

        token = get_token(request)

        resp_valid_token = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host, "token": token},
        )

        delete_policy(kube_apis.custom_objects, pol_name, v_s_route_setup.route_m.namespace)
        wait_before_test()

        resp_pol_deleted = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host, "token": token},
        )

        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            std_vsr_src,
            v_s_route_setup.route_m.namespace,
        )

        assert resp_no_token.status_code == 401 and f"Authorization Required" in resp_no_token.text
        assert resp_valid_token.status_code == 200 and f"Request ID:" in resp_valid_token.text
        assert resp_pol_deleted.status_code == 500 and f"Internal Server Error" in resp_pol_deleted.text

    @pytest.mark.parametrize("jwt_virtual_server_route", [jwt_vsr_invalid_pol_subroute_src])
    def test_jwt_invalid_policy_jwksuri(
        self,
        request,
        kube_apis,
        ingress_controller_prerequisites,
        crd_ingress_controller,
        v_s_route_app_setup,
        v_s_route_setup,
        test_namespace,
        jwt_virtual_server_route,
    ):
        """
        Test invalid jwt-policy in Virtual Server Route with keys fetched form Azure
        """
        req_url = f"http://{v_s_route_setup.public_endpoint.public_ip}:{v_s_route_setup.public_endpoint.port}"
        replace_configmap_from_yaml(
            kube_apis.v1,
            ingress_controller_prerequisites.config_map["metadata"]["name"],
            ingress_controller_prerequisites.namespace,
            jwt_cm_src,
        )
        pol_name = create_policy_from_yaml(
            kube_apis.custom_objects, jwt_pol_invalid_src, v_s_route_setup.route_m.namespace
        )
        wait_before_test()

        print(f"Patch vsr with policy: {jwt_virtual_server_route}")
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            jwt_virtual_server_route,
            v_s_route_setup.route_m.namespace,
        )
        wait_before_test()

        resp1 = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host},
        )

        token = get_token(request)

        resp2 = requests.get(
            f"{req_url}{v_s_route_setup.route_m.paths[0]}",
            headers={"host": v_s_route_setup.vs_host, "token": token},
        )

        delete_policy(kube_apis.custom_objects, pol_name, v_s_route_setup.route_m.namespace)
        patch_v_s_route_from_yaml(
            kube_apis.custom_objects,
            v_s_route_setup.route_m.name,
            std_vsr_src,
            v_s_route_setup.route_m.namespace,
        )

        assert resp1.status_code == 500 and f"Internal Server Error" in resp1.text
        assert resp2.status_code == 500 and f"Internal Server Error" in resp2.text

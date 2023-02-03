import time
from unittest import mock

import pytest
import requests
from settings import TEST_DATA
from suite.utils.policy_resources_utils import create_policy_from_yaml, delete_policy
from suite.utils.resources_utils import replace_configmap_from_yaml, wait_before_test
from suite.utils.vs_vsr_resources_utils import delete_and_create_vs_from_yaml, patch_v_s_route_from_yaml

std_vs_src = f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml"
jwt_pol_valid_src = f"{TEST_DATA}/jwt-policy-jwksuri/policies/jwt-policy-valid.yaml"
jwt_vs_spec_src = f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-policy-spec.yaml"
jwt_vs_route_src = f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-policy-route.yaml"
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
    ad_response = requests.post(f"https://login.microsoftonline.com/{ad_tenant}/oauth2/token", data=data)

    if ad_response.status_code == 200:
        return ad_response.json()["access_token"]
    else:
        pytest.fail("Unable to request Azure token endpoint")


@pytest.mark.skip_for_nginx_oss
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
                ],
            },
            {
                "example": "virtual-server",
                "app_type": "simple",
            },
        )
    ],
    indirect=True,
)
class TestJWTPoliciesVsJwksuri:
    @pytest.mark.parametrize("jwt_virtual_server", [jwt_vs_spec_src, jwt_vs_route_src])
    def test_jwt_policy_jwksuri(
        self,
        request,
        kube_apis,
        ingress_controller_prerequisites,
        crd_ingress_controller,
        virtual_server_setup,
        test_namespace,
        jwt_virtual_server,
    ):
        """
        Test jwt-policy in Virtual Server (spec and route) with keys fetched form Azure
        """
        replace_configmap_from_yaml(
            kube_apis.v1,
            ingress_controller_prerequisites.config_map["metadata"]["name"],
            ingress_controller_prerequisites.namespace,
            jwt_cm_src,
        )
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, jwt_pol_valid_src, test_namespace)
        wait_before_test()

        print(f"Patch vs with policy: {jwt_virtual_server}")
        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            jwt_virtual_server,
            virtual_server_setup.namespace,
        )
        resp1 = mock.Mock()
        resp1.status_code == 502
        counter = 0

        while resp1.status_code != 401 and counter < 3:
            resp1 = requests.get(
                virtual_server_setup.backend_1_url,
                headers={"host": virtual_server_setup.vs_host},
            )
            wait_before_test()
            counter = +1

        token = get_token(request)

        resp2 = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "token": token},
        )

        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            std_vs_src,
            virtual_server_setup.namespace,
        )

        assert resp1.status_code == 401 and f"Authorization Required" in resp1.text
        assert resp2.status_code == 200 and f"Request ID:" in resp2.text

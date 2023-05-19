import time
from unittest import mock

import pytest
import requests
from settings import TEST_DATA
from suite.utils.policy_resources_utils import create_policy_from_yaml, delete_policy
from suite.utils.resources_utils import replace_configmap_from_yaml, wait_before_test
from suite.utils.vs_vsr_resources_utils import (
    create_virtual_server_from_yaml,
    delete_and_create_vs_from_yaml,
    delete_virtual_server,
    patch_v_s_route_from_yaml,
)

std_vs_src = f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml"
jwt_pol_valid_src = f"{TEST_DATA}/jwt-policy-jwksuri/policies/jwt-policy-valid.yaml"
jwt_pol_invalid_src = f"{TEST_DATA}/jwt-policy-jwksuri/policies/jwt-policy-invalid.yaml"
jwt_vs_spec_src = f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-policy-spec.yaml"
jwt_vs_route_src = f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-policy-route.yaml"
jwt_spec_and_route_src = f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-policy-spec-and-route.yaml"
jwt_vs_route_subpath_src = f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-policy-route-subpath.yaml"
jwt_vs_route_subpath_diff_host_src = (
    f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-policy-route-subpath-diff-host.yaml"
)
jwt_vs_invalid_pol_spec_src = f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-invalid-policy-spec.yaml"
jwt_vs_invalid_pol_route_src = f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-invalid-policy-route.yaml"
jwt_vs_invalid_pol_route_subpath_src = (
    f"{TEST_DATA}/jwt-policy-jwksuri/virtual-server/virtual-server-invalid-policy-route-subpath.yaml"
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
@pytest.mark.skip(reason="issues with IdP communication")
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
    @pytest.mark.parametrize("jwt_virtual_server", [jwt_vs_spec_src, jwt_vs_route_src, jwt_spec_and_route_src])
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
        Test jwt-policy in Virtual Server (spec, route and both at the same time) with keys fetched form Azure
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
        resp_no_token = mock.Mock()
        resp_no_token.status_code == 502
        counter = 0

        while resp_no_token.status_code != 401 and counter < 20:
            resp_no_token = requests.get(
                virtual_server_setup.backend_1_url,
                headers={"host": virtual_server_setup.vs_host},
            )
            wait_before_test()
            counter += 1

        token = get_token(request)

        resp_valid_token = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "token": token},
            timeout=5,
        )

        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        wait_before_test()

        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            std_vs_src,
            virtual_server_setup.namespace,
        )

        assert resp_no_token.status_code == 401 and f"Authorization Required" in resp_no_token.text
        assert resp_valid_token.status_code == 200 and f"Request ID:" in resp_valid_token.text

    @pytest.mark.parametrize("jwt_virtual_server", [jwt_vs_invalid_pol_spec_src, jwt_vs_invalid_pol_route_src])
    def test_jwt_invalid_policy_jwksuri(
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
        Test invalid jwt-policy in Virtual Server (spec and route) with keys fetched form Azure
        """
        replace_configmap_from_yaml(
            kube_apis.v1,
            ingress_controller_prerequisites.config_map["metadata"]["name"],
            ingress_controller_prerequisites.namespace,
            jwt_cm_src,
        )
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, jwt_pol_invalid_src, test_namespace)
        wait_before_test()

        print(f"Patch vs with policy: {jwt_virtual_server}")
        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            jwt_virtual_server,
            virtual_server_setup.namespace,
        )
        wait_before_test()

        resp1 = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )

        token = get_token(request)

        resp2 = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host, "token": token},
            timeout=5,
        )

        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            std_vs_src,
            virtual_server_setup.namespace,
        )

        assert resp1.status_code == 500 and f"Internal Server Error" in resp1.text
        assert resp2.status_code == 500 and f"Internal Server Error" in resp2.text

    @pytest.mark.parametrize("jwt_virtual_server", [jwt_vs_route_subpath_src])
    def test_jwt_policy_subroute_jwksuri(
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
        Test jwt-policy in Virtual Server using subpaths with keys fetched form Azure
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
        resp_no_token = mock.Mock()
        resp_no_token.status_code == 502
        counter = 0

        while resp_no_token.status_code != 401 and counter < 20:
            resp_no_token = requests.get(
                virtual_server_setup.backend_1_url + "/subpath1",
                headers={"host": virtual_server_setup.vs_host},
            )
            wait_before_test()
            counter += 1

        token = get_token(request)

        resp_valid_token = requests.get(
            virtual_server_setup.backend_1_url + "/subpath1",
            headers={"host": virtual_server_setup.vs_host, "token": token},
            timeout=5,
        )

        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        wait_before_test()

        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            std_vs_src,
            virtual_server_setup.namespace,
        )

        assert resp_no_token.status_code == 401 and f"Authorization Required" in resp_no_token.text
        assert resp_valid_token.status_code == 200 and f"Request ID:" in resp_valid_token.text

    def test_jwt_policy_subroute_jwksuri_multiple_vs(
        self,
        request,
        kube_apis,
        ingress_controller_prerequisites,
        crd_ingress_controller,
        virtual_server_setup,
        test_namespace,
    ):
        """
        Test jwt-policy applied to two Virtual Servers with different hosts and the same subpaths
        """
        replace_configmap_from_yaml(
            kube_apis.v1,
            ingress_controller_prerequisites.config_map["metadata"]["name"],
            ingress_controller_prerequisites.namespace,
            jwt_cm_src,
        )
        pol_name = create_policy_from_yaml(kube_apis.custom_objects, jwt_pol_valid_src, test_namespace)
        wait_before_test()

        print(f"Patch first vs with policy: {jwt_vs_route_subpath_src}")
        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            jwt_vs_route_subpath_src,
            virtual_server_setup.namespace,
        )

        print(f"Create second vs with policy: {jwt_vs_route_subpath_diff_host_src}")
        create_virtual_server_from_yaml(
            kube_apis.custom_objects,
            jwt_vs_route_subpath_diff_host_src,
            virtual_server_setup.namespace,
        )

        wait_before_test()

        resp_1_no_token = mock.Mock()
        resp_1_no_token.status_code == 502

        resp_2_no_token = mock.Mock()
        resp_2_no_token.status_code == 502
        counter = 0

        while resp_1_no_token.status_code != 401 and counter < 20:
            resp_1_no_token = requests.get(
                virtual_server_setup.backend_1_url + "/subpath1",
                headers={"host": virtual_server_setup.vs_host},
            )
            wait_before_test()
            counter += 1

        counter = 0

        while resp_2_no_token.status_code != 401 and counter < 20:
            resp_2_no_token = requests.get(
                virtual_server_setup.backend_1_url + "/subpath1",
                headers={"host": "virtual-server-2.example.com"},
            )
            wait_before_test()
            counter += 1

        token = get_token(request)

        resp_1_valid_token = requests.get(
            virtual_server_setup.backend_1_url + "/subpath1",
            headers={"host": virtual_server_setup.vs_host, "token": token},
            timeout=5,
        )

        resp_2_valid_token = requests.get(
            virtual_server_setup.backend_1_url + "/subpath1",
            headers={"host": "virtual-server-2.example.com", "token": token},
            timeout=5,
        )

        delete_policy(kube_apis.custom_objects, pol_name, test_namespace)
        wait_before_test()

        delete_and_create_vs_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            std_vs_src,
            virtual_server_setup.namespace,
        )

        delete_virtual_server(
            kube_apis.custom_objects,
            "virtual-server-2",
            virtual_server_setup.namespace,
        )

        assert resp_1_no_token.status_code == 401 and f"Authorization Required" in resp_1_no_token.text
        assert resp_1_valid_token.status_code == 200 and f"Request ID:" in resp_1_valid_token.text

        assert resp_2_no_token.status_code == 401 and f"Authorization Required" in resp_2_no_token.text
        assert resp_2_valid_token.status_code == 200 and f"Request ID:" in resp_2_valid_token.text

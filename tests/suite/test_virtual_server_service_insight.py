from unittest import mock

import pytest
import requests
from settings import TEST_DATA
from suite.utils.resources_utils import (
    create_secret_from_yaml,
    delete_secret,
    ensure_response_from_backend,
    patch_deployment_from_yaml,
    wait_before_test,
)
from suite.utils.yaml_utils import get_first_host_from_yaml


@pytest.mark.vs
@pytest.mark.skip_for_nginx_oss
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {"type": "complete", "extra_args": [f"-enable-custom-resources", f"-enable-service-insight"]},
            {"example": "virtual-server", "app_type": "simple"},
        )
    ],
    indirect=True,
)
class TestVirtualServerServiceInsightHTTP:
    def test_responses_svc_insight_http(
        self, request, kube_apis, crd_ingress_controller, virtual_server_setup, ingress_controller_endpoint
    ):
        """test responses from service insight endpoint with http"""
        retry = 0
        resp = mock.Mock()
        resp.json.return_value = {}
        resp.status_code == 502
        vs_source = f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml"
        host = get_first_host_from_yaml(vs_source)
        req_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.service_insight_port}/probe/{host}"
        ensure_response_from_backend(req_url, virtual_server_setup.vs_host)
        while (resp.json() != {"Total": 3, "Up": 3, "Unhealthy": 0}) and retry < 5:
            resp = requests.get(req_url)
            wait_before_test()
            retry = retry + 1

        assert resp.status_code == 200, f"Expected 200 code for /probe/{host} but got {resp.status_code}"
        assert resp.json() == {"Total": 3, "Up": 3, "Unhealthy": 0}


@pytest.fixture(scope="class")
def https_secret_setup(request, kube_apis, test_namespace):
    print("------------------------- Deploy Secret -----------------------------------")
    secret_name = create_secret_from_yaml(kube_apis.v1, "nginx-ingress", f"{TEST_DATA}/service-insight/secret.yaml")

    def fin():
        delete_secret(kube_apis.v1, secret_name, "nginx-ingress")

    request.addfinalizer(fin)


@pytest.mark.vs
@pytest.mark.skip_for_nginx_oss
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    f"-enable-custom-resources",
                    f"-enable-service-insight",
                    f"-service-insight-tls-secret=nginx-ingress/test-secret",
                ],
            },
            {"example": "virtual-server", "app_type": "simple"},
        )
    ],
    indirect=True,
)
class TestVirtualServerServiceInsightHTTPS:
    def test_responses_svc_insight_https(
        self,
        request,
        kube_apis,
        https_secret_setup,
        ingress_controller_endpoint,
        crd_ingress_controller,
        virtual_server_setup,
    ):
        """test responses from service insight endpoint with https"""
        retry = 0
        resp = mock.Mock()
        resp.json.return_value = {}
        resp.status_code == 502
        vs_source = f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml"
        host = get_first_host_from_yaml(vs_source)
        req_url = f"https://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.service_insight_port}/probe/{host}"
        ensure_response_from_backend(req_url, virtual_server_setup.vs_host)
        while (resp.json() != {"Total": 3, "Up": 3, "Unhealthy": 0}) and retry < 5:
            resp = requests.get(req_url, verify=False)
            wait_before_test()
            retry = retry + 1
        assert resp.status_code == 200, f"Expected 200 code for /probe/{host} but got {resp.status_code}"
        assert resp.json() == {"Total": 3, "Up": 3, "Unhealthy": 0}

    def test_responses_svc_insight_update_pods(
        self,
        request,
        kube_apis,
        https_secret_setup,
        ingress_controller_endpoint,
        test_namespace,
        crd_ingress_controller,
        virtual_server_setup,
    ):
        """test responses from service insight endpoint with https and update number of replicas"""
        retry = 0
        resp = mock.Mock()
        resp.json.return_value = {}
        resp.status_code == 502
        vs_source = f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml"
        host = get_first_host_from_yaml(vs_source)
        req_url = f"https://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.service_insight_port}/probe/{host}"
        ensure_response_from_backend(req_url, virtual_server_setup.vs_host)

        # patch backend1 deployment with 5 replicas
        patch_deployment_from_yaml(kube_apis.apps_v1_api, test_namespace, f"{TEST_DATA}/service-insight/app.yaml")
        ensure_response_from_backend(req_url, virtual_server_setup.vs_host)
        while (resp.json() != {"Total": 6, "Up": 6, "Unhealthy": 0}) and retry < 5:
            resp = requests.get(req_url, verify=False)
            wait_before_test()
            retry = retry + 1
        assert resp.status_code == 200, f"Expected 200 code for /probe/{host} but got {resp.status_code}"
        assert resp.json() == {"Total": 6, "Up": 6, "Unhealthy": 0}

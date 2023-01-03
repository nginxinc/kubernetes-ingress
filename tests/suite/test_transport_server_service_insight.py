from unittest import mock

import pytest
import requests
from settings import TEST_DATA
from suite.utils.resources_utils import (
    create_secret_from_yaml,
    delete_secret,
    ensure_response_from_backend,
    get_ts_nginx_template_conf,
    patch_deployment_from_yaml,
    wait_before_test,
)
from suite.utils.yaml_utils import get_name_from_yaml


@pytest.mark.ts
@pytest.mark.skip_from_nginx_oss
@pytest.mark.parametrize(
    "crd_ingress_controller, transport_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    f"-global-configuration=nginx-ingress/nginx-configuration",
                    f"-enable-leader-election=false",
                    f"-enable-snippets",
                    f"-enable-custom-resources",
                    f"-enable-service-insight",
                ],
            },
            {"example": "transport-server-status"},
        )
    ],
    indirect=True,
)
class TestHealthCheckTransportServer:
    def test_responses_svc_insight_http(
        self,
        request,
        kube_apis,
        crd_ingress_controller,
        transport_server_setup,
        ingress_controller_prerequisites,
        ingress_controller_endpoint,
    ):
        """Test responses from service insight endpoint with http"""
        retry = 0
        resp = mock.Mock()
        resp.json.return_value = {}
        resp.status_code == 502
        ts_source = f"{TEST_DATA}/transport-server-tcp-load-balance/standard/service_deployment.yaml"
        name = get_name_from_yaml(ts_source)
        req_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.service_insight_port}/probe/ts/{name}"
        ensure_response_from_backend(req_url, transport_server_setup.ts_host)
        while (resp.json() != {"Total": 3, "Up": 3, "Unhealthy": 0}) and retry < 5:
            resp = requests.get(req_url)
            wait_before_test()
            retry = +1

        assert resp.status_code == 200, f"Expected 200 code for /probe/ts/{name} but got {resp.status_code}"
        assert resp.json() == {"Total": 1, "Up": 1, "Unhealthy": 0}

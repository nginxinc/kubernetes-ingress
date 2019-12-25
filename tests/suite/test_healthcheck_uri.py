import pytest
import requests

from suite.resources_utils import ensure_connection


@pytest.mark.ingresses
@pytest.mark.parametrize('ingress_controller, expected_responses',
                         [
                             pytest.param({"extra_args": ["-health-status=true",
                                                          "-health-status-uri=/something-va(l)id/blabla"]},
                                          {"/something-va(l)id/blabla": 200, "/nginx-health": 404},
                                          id="custom-health-status-uri"),
                             pytest.param({"extra_args": ["-health-status=true"]},
                                          {"/something-va(l)id/blabla": 404, "/nginx-health": 200},
                                          id="default-health-status-uri"),
                             pytest.param({"extra_args": ["-health-status=false"]},
                                          {"/something-va(l)id/blabla": 404, "/nginx-health": 404},
                                          id="disable-health-status")
                          ],
                         indirect=["ingress_controller"])
class TestHealthStatusURI:
    def test_response_code(self, ingress_controller_endpoint, ingress_controller, expected_responses):
        for uri in expected_responses:
            req_url = f"http://{ingress_controller_endpoint.public_ip}:{ingress_controller_endpoint.port}{uri}"
            ensure_connection(req_url, expected_responses[uri])
            resp = requests.get(req_url)
            assert resp.status_code == expected_responses[uri],\
                f"Expected {expected_responses[uri]} code for {uri} but got {resp.status_code}"

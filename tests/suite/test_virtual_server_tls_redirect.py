import pytest
import requests

from settings import TEST_DATA
from suite.custom_resources_utils import patch_virtual_server_from_yaml
from suite.resources_utils import wait_before_test


@pytest.mark.vs
@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-tls-redirect", "app_type": "simple"})],
                         indirect=True)
class TestVirtualServerTLSRedirect:
    def test_tls_redirect_defaults(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-tls-redirect/virtual-server-default-redirect.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)

        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host},
                              allow_redirects=False)
        resp_2 = requests.get(virtual_server_setup.backend_2_url,
                              headers={"host": virtual_server_setup.vs_host},
                              allow_redirects=False)
        assert resp_1.status_code == 301, "Expected: a redirect for scheme=http"
        assert resp_2.status_code == 301, "Expected: a redirect for scheme=http"

        resp_3 = requests.get(virtual_server_setup.backend_1_url_ssl,
                              headers={"host": virtual_server_setup.vs_host},
                              allow_redirects=False, verify=False)
        resp_4 = requests.get(virtual_server_setup.backend_2_url_ssl,
                              headers={"host": virtual_server_setup.vs_host},
                              allow_redirects=False, verify=False)
        assert resp_3.status_code == 200, "Expected: no redirect for scheme=https"
        assert resp_4.status_code == 200, "Expected: no redirect for scheme=https"

    def test_tls_redirect_based_on_header(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-tls-redirect/virtual-server-header-redirect.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)

        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host, "x-forwarded-proto": "http"},
                              allow_redirects=False)
        resp_2 = requests.get(virtual_server_setup.backend_2_url,
                              headers={"host": virtual_server_setup.vs_host, "x-forwarded-proto": "http"},
                              allow_redirects=False)
        assert resp_1.status_code == 308, "Expected: a redirect for x-forwarded-proto=http"
        assert resp_2.status_code == 308, "Expected: a redirect for x-forwarded-proto=http"

        resp_3 = requests.get(virtual_server_setup.backend_1_url_ssl,
                              headers={"host": virtual_server_setup.vs_host, "x-forwarded-proto": "http"},
                              allow_redirects=False, verify=False)
        resp_4 = requests.get(virtual_server_setup.backend_2_url_ssl,
                              headers={"host": virtual_server_setup.vs_host, "x-forwarded-proto": "http"},
                              allow_redirects=False, verify=False)
        assert resp_3.status_code == 308, "Expected: a redirect for x-forwarded-proto=http"
        assert resp_4.status_code == 308, "Expected: a redirect for x-forwarded-proto=http"

        resp_5 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host, "x-forwarded-proto": "https"},
                              allow_redirects=False)
        resp_6 = requests.get(virtual_server_setup.backend_2_url,
                              headers={"host": virtual_server_setup.vs_host, "x-forwarded-proto": "https"},
                              allow_redirects=False)
        assert resp_5.status_code == 200, "Expected: no redirect for x-forwarded-proto=https"
        assert resp_6.status_code == 200, "Expected: no redirect for x-forwarded-proto=https"

        resp_7 = requests.get(virtual_server_setup.backend_1_url_ssl,
                              headers={"host": virtual_server_setup.vs_host, "x-forwarded-proto": "https"},
                              allow_redirects=False, verify=False)
        resp_8 = requests.get(virtual_server_setup.backend_2_url_ssl,
                              headers={"host": virtual_server_setup.vs_host, "x-forwarded-proto": "https"},
                              allow_redirects=False, verify=False)
        assert resp_7.status_code == 200, "Expected: no redirect for x-forwarded-proto=https"
        assert resp_8.status_code == 200, "Expected: no redirect for x-forwarded-proto=https"

    def test_tls_redirect_based_on_scheme(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-tls-redirect/virtual-server-scheme-redirect.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)

        resp_1 = requests.get(virtual_server_setup.backend_1_url,
                              headers={"host": virtual_server_setup.vs_host},
                              allow_redirects=False)
        resp_2 = requests.get(virtual_server_setup.backend_2_url,
                              headers={"host": virtual_server_setup.vs_host},
                              allow_redirects=False)
        assert resp_1.status_code == 302, "Expected: a redirect for scheme=http"
        assert resp_2.status_code == 302, "Expected: a redirect for scheme=http"

        resp_3 = requests.get(virtual_server_setup.backend_1_url_ssl,
                              headers={"host": virtual_server_setup.vs_host},
                              allow_redirects=False, verify=False)
        resp_4 = requests.get(virtual_server_setup.backend_2_url_ssl,
                              headers={"host": virtual_server_setup.vs_host},
                              allow_redirects=False, verify=False)
        assert resp_3.status_code == 200, "Expected: no redirect for scheme=https"
        assert resp_4.status_code == 200, "Expected: no redirect for scheme=https"

import requests
import pytest

from settings import TEST_DATA
from suite.custom_assertions import assert_event_starts_with_text_and_contains_errors
from suite.resources_utils import wait_before_test, is_secret_present, get_events
from suite.vs_vsr_resources_utils import patch_virtual_server_from_yaml
from suite.yaml_utils import get_secret_name_from_vs_yaml


@pytest.mark.vs
@pytest.mark.smoke
@pytest.mark.parametrize('crd_ingress_controller, create_certmanager, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources", f"-enable-cert-manager"]},
                           {"issuer_name": "self-signed"},
                           {"example": "virtual-server-certmanager", "app_type": "simple"})],
                         indirect=True)
class TestCertManagerVirtualServer:
    def test_responses_after_setup(self, kube_apis, crd_ingress_controller, create_certmanager, virtual_server_setup):
        print("\nStep 1: Verify secret exists")
        wait_before_test(10)
        secret_name = get_secret_name_from_vs_yaml(f"{TEST_DATA}/virtual-server-certmanager/standard/virtual-server.yaml")
        sec = is_secret_present(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        assert sec is True
        print("\nStep 2: verify connectivity")
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200


@pytest.mark.vs
@pytest.mark.smoke
@pytest.mark.parametrize('crd_ingress_controller, create_certmanager, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources", f"-enable-cert-manager"]},
                           {"issuer_name": "ca-issuer"},
                           {"example": "virtual-server-certmanager", "app_type": "simple"})],
                         indirect=True)
class TestCertManagerVirtualServerCA:
    def test_responses_after_setup(self, kube_apis, crd_ingress_controller, create_certmanager, virtual_server_setup):
        vs_src = f"{TEST_DATA}/virtual-server-certmanager/virtual-server-updated.yaml"
        print("\nStep 1: Verify no secret exists with bad issuer name")
        secret_name = get_secret_name_from_vs_yaml(f"{TEST_DATA}/virtual-server-certmanager/standard/virtual-server.yaml")
        sec = is_secret_present(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        assert sec is False
        patch_virtual_server_from_yaml(kube_apis.custom_objects, virtual_server_setup.vs_name, vs_src,
                                      virtual_server_setup.namespace)
        print("\nStep 2: Verify secret exists with updated issuer name")
        secret_name = get_secret_name_from_vs_yaml(f"{TEST_DATA}/virtual-server-certmanager/virtual-server-updated.yaml")
        sec = is_secret_present(kube_apis.v1, secret_name, virtual_server_setup.namespace)
        retry = 0
        while not sec and retry <= 30:
            sec = is_secret_present(kube_apis.v1, secret_name, virtual_server_setup.namespace)
            retry += 1
            wait_before_test(5)
            print(f"Secret not found yet, retrying... #{retry}")
        print("\nStep 3: verify connectivity")
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

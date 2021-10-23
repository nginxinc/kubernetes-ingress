import requests
import pytest

from settings import TEST_DATA, DEPLOYMENTS
from suite.custom_assertions import wait_and_assert_status_code
from suite.custom_resources_utils import delete_crd, create_crd_from_yaml
from suite.vs_vsr_resources_utils import (
    delete_virtual_server,
    create_virtual_server_from_yaml,
    patch_virtual_server_from_yaml,
)
from suite.resources_utils import patch_rbac, replace_service, read_service, \
    wait_before_test, delete_service, create_service_from_yaml
from suite.yaml_utils import get_paths_from_vs_yaml, get_first_host_from_yaml, get_name_from_yaml


@pytest.mark.vs
@pytest.mark.smoke
@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server", "app_type": "simple"})],
                         indirect=True)
class TestVirtualServer:
    def test_responses_after_setup(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        print("\nStep 1: initial check")
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

    def test_responses_after_virtual_server_update(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        print("Step 2: update host and paths in the VS and check")
        patch_virtual_server_from_yaml(kube_apis.custom_objects, virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server/standard/virtual-server-updated.yaml",
                                       virtual_server_setup.namespace)
        new_paths = get_paths_from_vs_yaml(f"{TEST_DATA}/virtual-server/standard/virtual-server-updated.yaml")
        new_backend_1_url = f"http://{virtual_server_setup.public_endpoint.public_ip}" \
            f":{virtual_server_setup.public_endpoint.port}/{new_paths[0]}"
        new_backend_2_url = f"http://{virtual_server_setup.public_endpoint.public_ip}" \
            f":{virtual_server_setup.public_endpoint.port}/{new_paths[1]}"
        new_host = get_first_host_from_yaml(f"{TEST_DATA}/virtual-server/standard/virtual-server-updated.yaml")
        wait_before_test(1)
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 404
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 404
        resp = requests.get(new_backend_1_url, headers={"host": new_host})
        assert resp.status_code == 200
        resp = requests.get(new_backend_2_url, headers={"host": new_host})
        assert resp.status_code == 200

        print("Step 3: restore VS and check")
        patch_virtual_server_from_yaml(kube_apis.custom_objects, virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test(1)
        resp = requests.get(new_backend_1_url, headers={"host": new_host})
        assert resp.status_code == 404
        resp = requests.get(new_backend_2_url, headers={"host": new_host})
        assert resp.status_code == 404
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

    def test_responses_after_backend_update(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        print("Step 4: update one backend service port and check")
        the_service = read_service(kube_apis.v1, "backend1-svc", virtual_server_setup.namespace)
        the_service.spec.ports[0].port = 8080
        replace_service(kube_apis.v1, "backend1-svc", virtual_server_setup.namespace, the_service)
        wait_before_test(1)
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 502
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

        print("Step 5: restore BE and check")
        the_service = read_service(kube_apis.v1, "backend1-svc", virtual_server_setup.namespace)
        the_service.spec.ports[0].port = 80
        replace_service(kube_apis.v1, "backend1-svc", virtual_server_setup.namespace, the_service)
        wait_before_test(1)
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

    def test_responses_after_virtual_server_removal(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        print("\nStep 6: delete VS and check")
        delete_virtual_server(kube_apis.custom_objects, virtual_server_setup.vs_name, virtual_server_setup.namespace)
        wait_before_test(1)
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 404
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 404

        print("Step 7: restore VS and check")
        create_virtual_server_from_yaml(kube_apis.custom_objects,
                                        f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml",
                                        virtual_server_setup.namespace)
        wait_before_test(1)
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

    def test_responses_after_backend_service_removal(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        print("\nStep 8: remove one backend service and check")
        delete_service(kube_apis.v1, "backend1-svc", virtual_server_setup.namespace)
        wait_before_test(1)
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 502
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

        print("\nStep 9: restore backend service and check")
        create_service_from_yaml(kube_apis.v1, virtual_server_setup.namespace, f"{TEST_DATA}/common/backend1-svc.yaml")
        wait_before_test(3)
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

    def test_responses_after_rbac_misconfiguration_on_the_fly(self, kube_apis, crd_ingress_controller,
                                                              virtual_server_setup):
        print("Step 10: remove virtualservers from the ClusterRole and check")
        patch_rbac(kube_apis.rbac_v1, f"{TEST_DATA}/virtual-server/rbac-without-vs.yaml")
        wait_before_test(1)
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

        print("Step 11: restore ClusterRole and check")
        patch_rbac(kube_apis.rbac_v1, f"{DEPLOYMENTS}/rbac/rbac.yaml")
        wait_before_test(1)
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 200

    def test_responses_after_crd_removal_on_the_fly(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        print("\nStep 12: remove CRD and check")
        crd_name = get_name_from_yaml(f"{DEPLOYMENTS}/common/crds/k8s.nginx.org_virtualservers.yaml")
        delete_crd(kube_apis.api_extensions_v1, crd_name)
        wait_and_assert_status_code(404, virtual_server_setup.backend_1_url, virtual_server_setup.vs_host)
        wait_and_assert_status_code(404, virtual_server_setup.backend_2_url, virtual_server_setup.vs_host)

        print("Step 13: restore CRD and VS and check")
        create_crd_from_yaml(kube_apis.api_extensions_v1, crd_name,
                             f"{DEPLOYMENTS}/common/crds/k8s.nginx.org_virtualservers.yaml")
        wait_before_test(1)
        create_virtual_server_from_yaml(kube_apis.custom_objects,
                                        f"{TEST_DATA}/virtual-server/standard/virtual-server.yaml",
                                        virtual_server_setup.namespace)
        wait_and_assert_status_code(200, virtual_server_setup.backend_1_url, virtual_server_setup.vs_host)
        wait_and_assert_status_code(200, virtual_server_setup.backend_2_url, virtual_server_setup.vs_host)


@pytest.mark.vs
@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "rbac-without-vs", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server", "app_type": "simple"})],
                         indirect=True)
class TestVirtualServerInitialRBACMisconfiguration:
    @pytest.mark.skip(reason="issues with ingressClass")
    def test_responses_after_rbac_misconfiguration(self, kube_apis, crd_ingress_controller, virtual_server_setup):
        print("\nStep 1: rbac misconfiguration from the very start")
        resp = requests.get(virtual_server_setup.backend_1_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 404
        resp = requests.get(virtual_server_setup.backend_2_url,
                            headers={"host": virtual_server_setup.vs_host})
        assert resp.status_code == 404

        print("Step 2: configure RBAC and check")
        patch_rbac(kube_apis.rbac_v1, f"{DEPLOYMENTS}/rbac/rbac.yaml")
        wait_and_assert_status_code(200, virtual_server_setup.backend_1_url, virtual_server_setup.vs_host)
        wait_and_assert_status_code(200, virtual_server_setup.backend_2_url, virtual_server_setup.vs_host)

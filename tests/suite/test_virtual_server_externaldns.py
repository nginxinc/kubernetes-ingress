import pytest

from settings import TEST_DATA
from suite.custom_resources_utils import is_dnsendpoint_present
from suite.resources_utils import wait_before_test
from suite.vs_vsr_resources_utils import patch_virtual_server_from_yaml
from suite.yaml_utils import get_first_host_from_yaml, get_namespace_from_yaml


@pytest.mark.vs
@pytest.mark.smoke
@pytest.mark.parametrize('crd_ingress_controller_with_ed, create_externaldns, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources", f"-enable-external-dns"]},
                           {}, {"example": "virtual-server-external-dns", "app_type": "simple"})],
                         indirect=True)
class TestExternalDNSVirtualServer:
    def test_responses_after_setup(self, kube_apis, crd_ingress_controller_with_ed, create_externaldns, virtual_server_setup):
        print("\nStep 1: Verify DNSEndpoint exists")
        dns_name = get_first_host_from_yaml(f"{TEST_DATA}/virtual-server-external-dns/standard/virtual-server.yaml")
        retry = 0
        dep = is_dnsendpoint_present(kube_apis.custom_objects, dns_name, virtual_server_setup.namespace) 
        while dep == False and retry <= 60:
            dep = is_dnsendpoint_present(kube_apis.custom_objects, dns_name, virtual_server_setup.namespace)
            retry += 1
            wait_before_test(1)
            print(f"DNSEndpoint not created, retrying... #{retry}")
        print("\nStep 2: Verify external-dns picked up the record")
        pod_ns = get_namespace_from_yaml(f"{TEST_DATA}/virtual-server-external-dns/external-dns.yaml")
        print(f"\nPod namespace: {pod_ns}")
        pods = kube_apis.v1.list_namespaced_pod(pod_ns)
        print(f"\nPods in namespace: {pods}")
        pod_name = kube_apis.v1.list_namespaced_pod(pod_ns).items[0].metadata.name
        print(f"\nPod name: {pod_name}")
        log_contents = kube_apis.v1.read_namespaced_pod_log(pod_name, pod_ns)
        retry = 0
        while "CREATE: virtual-server.example.com 0 IN A" not in log_contents and retry <= 30:
            log_contents = kube_apis.v1.read_namespaced_pod_log(pod_name, pod_ns)
            retry += 1
            wait_before_test(1)
            print(f"External DNS not updated, retrying... #{retry}")

    def test_update_to_ed_in_vs(self, kube_apis, crd_ingress_controller_with_ed, create_externaldns, virtual_server_setup):
        print("\nStep 1: Update VirtualServer")
        dns_name = get_first_host_from_yaml(f"{TEST_DATA}/virtual-server-external-dns/virtual-server-updated.yaml")
        patch_src = f"{TEST_DATA}/virtual-server-external-dns/virtual-server-updated.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )
        retry = 0
        dep = is_dnsendpoint_present(kube_apis.custom_objects, dns_name, virtual_server_setup.namespace) 
        while dep == False and retry <= 60:
            dep = is_dnsendpoint_present(kube_apis.custom_objects, dns_name, virtual_server_setup.namespace)
            retry += 1
            wait_before_test(1)
            print(f"DNSEndpoint not created, retrying... #{retry}")
        print("\nStep 2: Verify external-dns picked up the update")
        pod_ns = get_namespace_from_yaml(f"{TEST_DATA}/virtual-server-external-dns/external-dns.yaml")
        pod_name = kube_apis.v1.list_namespaced_pod(pod_ns).items[0].metadata.name
        log_contents = kube_apis.v1.read_namespaced_pod_log(pod_name, pod_ns)
        retry = 0
        while "UPSERT: virtual-server.example.com 180 IN A" not in log_contents and retry <= 30:
            log_contents = kube_apis.v1.read_namespaced_pod_log(pod_name, pod_ns)
            retry += 1
            wait_before_test(1)
            print(f"External DNS not updated, retrying... #{retry}")

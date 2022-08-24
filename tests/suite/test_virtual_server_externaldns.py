import pytest
from settings import TEST_DATA
from suite.custom_assertions import assert_event
from suite.custom_resources_utils import is_dnsendpoint_present
from suite.resources_utils import get_events, wait_before_test
from suite.vs_vsr_resources_utils import patch_virtual_server_from_yaml
from suite.yaml_utils import get_name_from_yaml, get_namespace_from_yaml

VS_YAML = f"{TEST_DATA}/virtual-server-external-dns/standard/virtual-server.yaml"


@pytest.mark.vs
@pytest.mark.smoke
@pytest.mark.parametrize(
    "crd_ingress_controller_with_ed, create_externaldns, virtual_server_setup",
    [
        (
            {"type": "complete", "extra_args": [f"-enable-custom-resources", f"-enable-external-dns"]},
            {},
            {"example": "virtual-server-external-dns", "app_type": "simple"},
        )
    ],
    indirect=True,
)
class TestExternalDNSVirtualServer:
    def test_responses_after_setup(
        self, kube_apis, crd_ingress_controller_with_ed, create_externaldns, virtual_server_setup
    ):
        print("\nStep 1: Verify DNSEndpoint exists")
        dns_ep_name = get_name_from_yaml(VS_YAML)
        retry = 0
        dep = is_dnsendpoint_present(kube_apis.custom_objects, dns_ep_name, virtual_server_setup.namespace)
        while dep == False and retry <= 60:
            dep = is_dnsendpoint_present(kube_apis.custom_objects, dns_ep_name, virtual_server_setup.namespace)
            retry += 1
            wait_before_test(1)
            print(f"DNSEndpoint not created, retrying... #{retry}")
        assert dep is True
        print("\nStep 2: Verify external-dns picked up the record")
        pod_ns = get_namespace_from_yaml(f"{TEST_DATA}/virtual-server-external-dns/external-dns.yaml")
        pod_name = kube_apis.v1.list_namespaced_pod(pod_ns).items[0].metadata.name
        log_contents = kube_apis.v1.read_namespaced_pod_log(pod_name, pod_ns)
        wanted_string = "CREATE: virtual-server.example.com 0 IN A"
        retry = 0
        while wanted_string not in log_contents and retry <= 60:
            log_contents = kube_apis.v1.read_namespaced_pod_log(pod_name, pod_ns)
            retry += 1
            wait_before_test(1)
            print(f"External DNS not updated, retrying... #{retry}")
        assert wanted_string in log_contents

    def test_update_to_ed_in_vs(
        self, kube_apis, crd_ingress_controller_with_ed, create_externaldns, virtual_server_setup
    ):
        print("\nStep 1: Update VirtualServer")
        patch_src = f"{TEST_DATA}/virtual-server-external-dns/virtual-server-updated.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )
        print("\nStep 2: Verify the DNSEndpoint was updated")
        vs_event_update_text = "Successfully updated DNSEndpoint"
        wait_before_test(5)
        events = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_event(vs_event_update_text, events)

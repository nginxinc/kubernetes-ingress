import pytest, ssl
import requests
from pprint import pprint
from suite.fixtures import PublicEndpoint
from suite.resources_utils import (
    wait_before_test,
    create_items_from_yaml,
    delete_items_from_yaml,
    wait_until_all_pods_are_ready,
    get_first_pod_name,
)
from suite.custom_resources_utils import (
    read_ts,
    delete_ts,
    create_ts_from_yaml,

)
from suite.vs_vsr_resources_utils import (
    read_vs,
    create_virtual_server_from_yaml,
    delete_virtual_server,
)
from suite.yaml_utils import get_first_host_from_yaml
from suite.ssl_utils import get_server_certificate_subject, create_sni_session
from settings import TEST_DATA

class TransportServerTlsSetup:
    """
    Encapsulate Transport Server details.

    Attributes:
        public_endpoint (object):
        ts_resource (dict):
        name (str):
        namespace (str):
        ts_host (str):
    """

    def __init__(self, public_endpoint: PublicEndpoint, ts_resource, name, namespace, ts_host):
        self.public_endpoint = public_endpoint
        self.ts_resource = ts_resource
        self.name = name
        self.namespace = namespace
        self.ts_host = ts_host


@pytest.fixture(scope="class")
def transport_server_tls_passthrough_setup(
    request, kube_apis, test_namespace, ingress_controller_endpoint
) -> TransportServerTlsSetup:
    """
    Prepare Transport Server Example.

    :param request: internal pytest fixture to parametrize this method
    :param kube_apis: client apis
    :param test_namespace: namespace for test resources
    :param ingress_controller_endpoint: ip and port information
    :return TransportServerTlsSetup:
    """
    print(
        "------------------------- Deploy Transport Server with tls passthrough -----------------------------------"
    )
    # deploy secure_app
    secure_app_file = f"{TEST_DATA}/{request.param['example']}/standard/secure-app.yaml"
    create_items_from_yaml(kube_apis, secure_app_file, test_namespace)

    # deploy transport server
    transport_server_std_src = f"{TEST_DATA}/{request.param['example']}/standard/transport-server.yaml"
    ts_resource = create_ts_from_yaml(
        kube_apis.custom_objects, transport_server_std_src, test_namespace
    )
    ts_host = get_first_host_from_yaml(transport_server_std_src)
    wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)

    def fin():
        print("Clean up TransportServer and app:")
        delete_ts(kube_apis.custom_objects, ts_resource, test_namespace)
        delete_items_from_yaml(kube_apis, secure_app_file, test_namespace)

    request.addfinalizer(fin)

    return TransportServerTlsSetup(
        ingress_controller_endpoint,
        ts_resource,
        ts_resource["metadata"]["name"],
        test_namespace,
        ts_host,
    )


@pytest.mark.ts
@pytest.mark.parametrize(
    "crd_ingress_controller, transport_server_tls_passthrough_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    "-enable-leader-election=false",
                    "-enable-tls-passthrough=true",
                ],
            },
            {"example": "transport-server-tls-passthrough"},
        )
    ],
    indirect=True,
)
class TestTransportServerTlsPassthrough:
    def restore_ts(self, kube_apis, transport_server_tls_passthrough_setup) -> None:
        """
        Function to create std TS resource
        """
        ts_std_src = f"{TEST_DATA}/transport-server-tls-passthrough/standard/transport-server.yaml"
        ts_std_res = create_ts_from_yaml(
                        kube_apis.custom_objects,
                        ts_std_src,
                        transport_server_tls_passthrough_setup.namespace,
                    )
        wait_before_test(1)
        pprint(ts_std_res)

    @pytest.mark.smoke
    def test_tls_passthrough(
        self,
        kube_apis,
        crd_ingress_controller,
        transport_server_tls_passthrough_setup,
        test_namespace,
    ):
        """
            Test TransportServer TLS passthrough on https port.
        """
        session = create_sni_session()
        req_url = (
            f"https://{transport_server_tls_passthrough_setup.public_endpoint.public_ip}:"
            f"{transport_server_tls_passthrough_setup.public_endpoint.port_ssl}"
        )
        wait_before_test()
        resp = session.get(
            req_url,
            headers={"host": transport_server_tls_passthrough_setup.ts_host},
            verify=False,
        )
        assert resp.status_code == 200
        assert f"hello from pod {get_first_pod_name(kube_apis.v1, test_namespace)}" in resp.text
    
    def test_tls_passthrough_host_collision_ts(
        self,
        kube_apis,
        crd_ingress_controller,
        transport_server_tls_passthrough_setup,
        test_namespace,
    ):
        """
            Test host collision handling in TransportServer with another TransportServer.
        """
        print("Step 1: Create second TS with same host")
        ts_src_same_host = (
            f"{TEST_DATA}/transport-server-tls-passthrough/transport-server-same-host.yaml"
        )
        ts_same_host = create_ts_from_yaml(
            kube_apis.custom_objects, ts_src_same_host, test_namespace
        )
        wait_before_test()
        response = read_ts(
            kube_apis.custom_objects, test_namespace, ts_same_host["metadata"]["name"]
        )
        assert (
            response["status"]["reason"] == "Rejected"
            and response["status"]["message"] == "Host is taken by another resource"
        )

        print("Step 2: Delete TS taking up the host")
        delete_ts(
            kube_apis.custom_objects,
            transport_server_tls_passthrough_setup.ts_resource,
            test_namespace,
        )
        wait_before_test(1)
        response = read_ts(
            kube_apis.custom_objects, test_namespace, ts_same_host["metadata"]["name"]
        )
        assert (
            response["status"]["reason"] == "AddedOrUpdated"
            and response["status"]["state"] == "Valid"
        )
        print("Step 3: Delete second TS and re-create standard one")
        delete_ts(
            kube_apis.custom_objects,
            ts_same_host,
            test_namespace
        )
        self.restore_ts(kube_apis, transport_server_tls_passthrough_setup)
        response = read_ts(
            kube_apis.custom_objects, test_namespace, transport_server_tls_passthrough_setup.name
        )
        assert (
            response["status"]["reason"] == "AddedOrUpdated"
            and response["status"]["state"] == "Valid"
        )

    def test_tls_passthrough_host_collision_vs(
        self,
        kube_apis,
        crd_ingress_controller,
        transport_server_tls_passthrough_setup,
        test_namespace,
    ):
        """
            Test host collision handling in TransportServer with VirtualServer.
        """
        print("Step 1: Create VirtualServer with same host")
        vs_src_same_host = (
            f"{TEST_DATA}/transport-server-tls-passthrough/virtual-server-same-host.yaml"
        )
        vs_same_host_name = create_virtual_server_from_yaml(
            kube_apis.custom_objects, vs_src_same_host, test_namespace
        )
        wait_before_test(1)
        response = read_vs(kube_apis.custom_objects, test_namespace, vs_same_host_name)
        delete_virtual_server(kube_apis.custom_objects, vs_same_host_name, test_namespace)

        assert (
            response["status"]["reason"] == "Rejected"
            and response["status"]["message"] == "Host is taken by another resource"
        )

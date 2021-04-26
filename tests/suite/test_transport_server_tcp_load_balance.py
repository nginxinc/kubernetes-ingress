import pytest
import re
import socket
import time

from suite.resources_utils import (
    wait_before_test,
    get_ts_nginx_template_conf,
    scale_deployment
)
from suite.custom_resources_utils import (
    patch_ts,
    read_ts,
    delete_ts,
    create_ts_from_yaml,
)
from settings import TEST_DATA

@pytest.mark.ts
@pytest.mark.parametrize(
    "crd_ingress_controller, transport_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args":
                    [
                        "-global-configuration=nginx-ingress/nginx-configuration",
                        "-enable-leader-election=false"
                    ]
            },
            {"example": "transport-server-tcp-load-balance"},
        )
    ],
    indirect=True,
)
class TestTransportServerTcpLoadBalance:

    def restore_ts(self, kube_apis, transport_server_setup) -> None:
        """
        Function to revert a TransportServer resource to a valid state.
        """
        patch_src = f"{TEST_DATA}/transport-server-tcp-load-balance/standard/transport-server.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )
        wait_before_test()

    def test_number_of_replicas(
        self, kube_apis, crd_ingress_controller, transport_server_setup, ingress_controller_prerequisites
    ):
        """
        The load balancing of TCP should result in 4 servers to match the 4 replicas of a service.
        """
        original = scale_deployment(kube_apis.apps_v1_api, "tcp-service", transport_server_setup.namespace, 4)
        wait_before_test()

        result_conf = get_ts_nginx_template_conf(
            kube_apis.v1,
            transport_server_setup.namespace,
            transport_server_setup.name,
            transport_server_setup.ingress_pod_name,
            ingress_controller_prerequisites.namespace
        )

        pattern = 'server .*;'
        num_servers = len(re.findall(pattern, result_conf))

        assert num_servers is 4

        scale_deployment(kube_apis.apps_v1_api, "tcp-service", transport_server_setup.namespace, original)
        wait_before_test()

    def test_tcp_request_load_balanced(
            self, kube_apis, crd_ingress_controller, transport_server_setup, ingress_controller_prerequisites
    ):
        """
        Requests to the load balanced TCP service should result in responses from 3 different endpoints.
        """
        wait_before_test()
        port = transport_server_setup.public_endpoint.tcp_server_port
        host = transport_server_setup.public_endpoint.public_ip

        print(f"sending tcp requests to: {host}:{port}")

        endpoints = {}
        for i in range(20):
            client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            client.connect((host, port))
            client.sendall(b'connect')
            response = client.recv(4096)
            endpoint = response.decode()
            print(f' req number {i}; response: {endpoint}')
            if endpoint not in endpoints:
                endpoints[endpoint] = 1
            else:
                endpoints[endpoint] = endpoints[endpoint] + 1
            client.close()

        assert len(endpoints) is 3

        result_conf = get_ts_nginx_template_conf(
            kube_apis.v1,
            transport_server_setup.namespace,
            transport_server_setup.name,
            transport_server_setup.ingress_pod_name,
            ingress_controller_prerequisites.namespace
        )

        pattern = 'server .*;'
        servers = re.findall(pattern, result_conf)
        for key in endpoints.keys():
            found = False
            for server in servers:
                if key in server:
                    found = True
            assert found

    def test_tcp_request_load_balanced_multiple(
            self, kube_apis, crd_ingress_controller, transport_server_setup
    ):
        """
        Requests to the load balanced TCP service should result in responses from 3 different endpoints.
        """
        port = transport_server_setup.public_endpoint.tcp_server_port
        host = transport_server_setup.public_endpoint.public_ip

        # Step 1, confirm load balancing is working.
        print(f"sending tcp requests to: {host}:{port}")
        client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        client.connect((host, port))
        client.sendall(b'connect')
        response = client.recv(4096)
        endpoint = response.decode()
        print(f'response: {endpoint}')
        client.close()
        assert endpoint is not ""

        # Step 2, add a second TransportServer with the same port and confirm te collision
        transport_server_file = f"{TEST_DATA}/transport-server-tcp-load-balance/second-transport-server.yaml"
        ts_resource = create_ts_from_yaml(
            kube_apis.custom_objects, transport_server_file, transport_server_setup.namespace
        )
        wait_before_test()

        second_ts_name = ts_resource['metadata']['name']
        response = read_ts(
            kube_apis.custom_objects,
            transport_server_setup.namespace,
            second_ts_name,
        )
        assert (
                response["status"]
                and response["status"]["reason"] == "Rejected"
                and response["status"]["state"] == "Warning"
                and response["status"]["message"] == "Listener tcp-server is taken by another resource"
        )

        # Step 3, remove the default TransportServer with the same port
        delete_ts(kube_apis.custom_objects, transport_server_setup.resource, transport_server_setup.namespace)

        wait_before_test()
        response = read_ts(
            kube_apis.custom_objects,
            transport_server_setup.namespace,
            second_ts_name,
        )
        assert (
                response["status"]
                and response["status"]["reason"] == "AddedOrUpdated"
                and response["status"]["state"] == "Valid"
        )

        # Step 4, confirm load balancing is still working.
        print(f"sending tcp requests to: {host}:{port}")
        client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        client.connect((host, port))
        client.sendall(b'connect')
        response = client.recv(4096)
        endpoint = response.decode()
        print(f'response: {endpoint}')
        client.close()
        assert endpoint is not ""

        # cleanup
        delete_ts(kube_apis.custom_objects, ts_resource, transport_server_setup.namespace)
        transport_server_file = f"{TEST_DATA}/transport-server-tcp-load-balance/standard/transport-server.yaml"
        create_ts_from_yaml(
            kube_apis.custom_objects, transport_server_file, transport_server_setup.namespace
        )
        wait_before_test()

    def test_tcp_request_load_balanced_wrong_port(
            self, kube_apis, crd_ingress_controller, transport_server_setup
    ):
        """
        Requests to the load balanced TCP service should result in responses from 3 different endpoints.
        """

        patch_src = f"{TEST_DATA}/transport-server-tcp-load-balance/wrong-port-transport-server.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )

        wait_before_test()

        port = transport_server_setup.public_endpoint.tcp_server_port
        host = transport_server_setup.public_endpoint.public_ip

        print(f"sending tcp requests to: {host}:{port}")
        for i in range(3):
            try:
                client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                client.connect((host, port))
                client.sendall(b'connect')
            except ConnectionResetError as E:
                print("The expected exception occurred:", E)

        self.restore_ts(kube_apis, transport_server_setup)

    def test_tcp_request_load_balanced_missing_service(
            self, kube_apis, crd_ingress_controller, transport_server_setup
    ):
        """
        Requests to the load balanced TCP service should result in responses from 3 different endpoints.
        """

        patch_src = f"{TEST_DATA}/transport-server-tcp-load-balance/missing-service-transport-server.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )

        wait_before_test()

        port = transport_server_setup.public_endpoint.tcp_server_port
        host = transport_server_setup.public_endpoint.public_ip

        print(f"sending tcp requests to: {host}:{port}")
        for i in range(3):
            try:
                client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                client.connect((host, port))
                client.sendall(b'connect')
            except ConnectionResetError as E:
                print("The expected exception occurred:", E)

        self.restore_ts(kube_apis, transport_server_setup)

    def make_holding_connection(self, host, port):
        print(f"sending tcp requests to: {host}:{port}")
        client = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        client.connect((host, port))
        client.sendall(b'hold')
        response = client.recv(4096)
        endpoint = response.decode()
        print(f'response: {endpoint}')
        return client

    def test_tcp_request_max_connections(
            self, kube_apis, crd_ingress_controller, transport_server_setup, ingress_controller_prerequisites
    ):
        """
        The config, maxConns, should limit the number of open TCP connections.
        3 replicas of max 2 connections is 6, so making the 7th connection will fail.
        """

        # step 1 - set max connections to 2 with 1 replica
        patch_src = f"{TEST_DATA}/transport-server-tcp-load-balance/max-connections-transport-server.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )
        wait_before_test()

        result_conf = get_ts_nginx_template_conf(
            kube_apis.v1,
            transport_server_setup.namespace,
            transport_server_setup.name,
            transport_server_setup.ingress_pod_name,
            ingress_controller_prerequisites.namespace
        )

        pattern = 'max_conns=2'
        configs = re.findall(pattern, result_conf)

        assert len(configs) is 3

        # step 2 - make the number of allowed connections
        port = transport_server_setup.public_endpoint.tcp_server_port
        host = transport_server_setup.public_endpoint.public_ip

        clients = []
        for i in range(6):
            c = self.make_holding_connection(host, port)
            clients.append(c)

        # step 3 - assert the next connection fails
        try:
            c = self.make_holding_connection(host, port)
            # making a connection should fail and throw an exception
            assert c is None
        except ConnectionResetError as E:
            print("The expected exception occurred:", E)

        for c in clients:
            c.close()

        # step 4 - revert to config with no max connections
        patch_src = f"{TEST_DATA}/transport-server-tcp-load-balance/standard/transport-server.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )
        wait_before_test()

        # step 5 - confirm making lots of connections doesn't cause an error
        clients = []
        for i in range(24):
            c = self.make_holding_connection(host, port)
            clients.append(c)

        for c in clients:
            c.close()

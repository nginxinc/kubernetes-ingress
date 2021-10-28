import grpc
import pytest
from kubernetes.client.rest import ApiException

from settings import TEST_DATA, DEPLOYMENTS
from suite.custom_assertions import assert_event_starts_with_text_and_contains_errors
from suite.custom_resources_utils import read_custom_resource
from suite.grpc.helloworld_pb2 import HelloRequest
from suite.grpc.helloworld_pb2_grpc import GreeterStub
from suite.resources_utils import create_example_app, wait_until_all_pods_are_ready, \
    delete_common_app, create_secret_from_yaml, replace_configmap_from_yaml, \
    delete_items_from_yaml, get_first_pod_name, get_events
from suite.ssl_utils import get_certificate
from suite.vs_vsr_resources_utils import get_vs_nginx_template_conf, \
    patch_virtual_server_from_yaml
from suite.resources_utils import wait_before_test


@pytest.fixture(scope="function")
def backend_setup(request, kube_apis, ingress_controller_prerequisites, test_namespace):
    """
    Replace the ConfigMap and deploy the secret.

    :param request: pytest fixture
    :param kube_apis: client apis
    :param test_namespace:
    """
    app_name = request.param.get("app_type")
    try:
        print("------------------------- Replace ConfigMap with HTTP2 -------------------------")
        cm_source = f"{TEST_DATA}/virtual-server-grpc/nginx-config.yaml"
        replace_configmap_from_yaml(kube_apis.v1, 
                                    ingress_controller_prerequisites.config_map['metadata']['name'],
                                    ingress_controller_prerequisites.namespace,
                                    cm_source)
        print("------------------------- Deploy Secret -----------------------------")
        src_sec_yaml = f"{TEST_DATA}/virtual-server-grpc/tls-secret.yaml"
        create_secret_from_yaml(kube_apis.v1, test_namespace, src_sec_yaml)
        print("------------------------- Deploy App -----------------------------")
        app_name = request.param.get("app_type")
        create_example_app(kube_apis, app_name, test_namespace)
        wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)
    except Exception as ex:
        print("Failed to complete setup, cleaning up..")
        delete_items_from_yaml(kube_apis, src_sec_yaml, test_namespace)
        replace_configmap_from_yaml(kube_apis.v1,
                        ingress_controller_prerequisites.config_map['metadata']['name'],
                        ingress_controller_prerequisites.namespace,
                        f"{DEPLOYMENTS}/common/nginx-config.yaml")
        delete_common_app(kube_apis, app_name, test_namespace)
        pytest.fail(f"VS GRPC setup failed")

    def fin():
        print("Clean up:")
        delete_items_from_yaml(kube_apis, src_sec_yaml, test_namespace)
        replace_configmap_from_yaml(kube_apis.v1,
                        ingress_controller_prerequisites.config_map['metadata']['name'],
                        ingress_controller_prerequisites.namespace,
                        f"{DEPLOYMENTS}/common/nginx-config.yaml")
        delete_common_app(kube_apis, app_name, test_namespace)

    request.addfinalizer(fin)

@pytest.mark.vs
@pytest.mark.smoke
@pytest.mark.parametrize('crd_ingress_controller, virtual_server_setup',
                         [({"type": "complete", "extra_args": [f"-enable-custom-resources"]},
                           {"example": "virtual-server-grpc"})],
                         indirect=True)
class TestVirtualServerGrpc:
 
    def connect_grpc_backend(self, virtual_server_setup) -> None:
        """
        Function to connect to a gRPC enabled backend
        """
        cert = get_certificate(virtual_server_setup.public_endpoint.public_ip,
                               virtual_server_setup.vs_host,
                               virtual_server_setup.public_endpoint.port_ssl)
        target = f'{virtual_server_setup.public_endpoint.public_ip}:{virtual_server_setup.public_endpoint.port_ssl}'
        credentials = grpc.ssl_channel_credentials(root_certificates=cert.encode())
        options = (('grpc.ssl_target_name_override', virtual_server_setup.vs_host),)

        with grpc.secure_channel(target, credentials, options) as channel:
            stub = GreeterStub(channel)
            response = ""
            try:
                response = stub.SayHello(HelloRequest(name=virtual_server_setup.public_endpoint.public_ip))
                print(response)
            except grpc.RpcError as e:
                print(e.details())
                pytest.fail("RPC error was not expected during call, exiting...")

    def patch_valid_vs(self, kube_apis, virtual_server_setup) -> None:
        """
        Function to revert vs deployment to valid state
        """
        patch_src = f"{TEST_DATA}/virtual-server-grpc/standard/virtual-server.yaml"
        patch_virtual_server_from_yaml(
            kube_apis.custom_objects,
            virtual_server_setup.vs_name,
            patch_src,
            virtual_server_setup.namespace,
        )

    @pytest.mark.parametrize("backend_setup", [{"app_type": "grpc-vs"}], indirect=True)
    def test_config_after_setup(self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, 
                                backend_setup, virtual_server_setup):
        print("\nStep 1: assert config")
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            virtual_server_setup.namespace,
                                            virtual_server_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)

        assert "proxy_connect_timeout 60s;" not in config
        assert "proxy_read_timeout 60s;" not in config
        assert "proxy_send_timeout 60s;" not in config

        assert "proxy_set_header Upgrade $http_upgrade;" not in config
        assert "proxy_http_version 1.1;" not in config

        assert "proxy_next_upstream error timeout;" not in config
        assert "proxy_next_upstream_timeout 0s;" not in config
        assert "proxy_next_upstream_tries 0;" not in config

        assert "grpc_connect_timeout 60s;" in config
        assert "grpc_read_timeout 60s;" in config
        assert "grpc_send_timeout 60s;" in config

        assert "grpc_set_header X-Real-IP $remote_addr;" in config
        assert "grpc_set_header X-Forwarded-For $proxy_add_x_forwarded_for;" in config
        assert "grpc_set_header X-Forwarded-Host $host;" in config
        assert "grpc_set_header X-Forwarded-Port $server_port;" in config
        assert "grpc_set_header X-Forwarded-Proto $scheme;" in config

        assert 'grpc_set_header Host "$host";' in config

        assert "grpc_next_upstream error timeout;" in config
        assert "grpc_next_upstream_timeout 0s;" in config
        assert "grpc_next_upstream_tries 0;" in config

        print("\nStep 2: Check connection to app")
        self.connect_grpc_backend(virtual_server_setup)

    @pytest.mark.parametrize("backend_setup", [{"app_type": "grpc-vs"}], indirect=True)
    def test_config_after_enable_tls(self, kube_apis, ingress_controller_prerequisites,
                                     crd_ingress_controller, backend_setup, virtual_server_setup):
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                       virtual_server_setup.vs_name,
                                       f"{TEST_DATA}/virtual-server-grpc/virtual-server-updated.yaml",
                                       virtual_server_setup.namespace)
        wait_before_test()
        config = get_vs_nginx_template_conf(kube_apis.v1,
                                            virtual_server_setup.namespace,
                                            virtual_server_setup.vs_name,
                                            ic_pod_name,
                                            ingress_controller_prerequisites.namespace)
        assert 'grpc_pass grpcs://' in config

    @pytest.mark.parametrize("backend_setup", [{"app_type": "grpc-vs"}], indirect=True)
    def test_validation_flow(self, kube_apis, ingress_controller_prerequisites,
                             crd_ingress_controller, backend_setup, virtual_server_setup):
        print("\nTest 1: Wrong type")
        patch_virtual_server_from_yaml(kube_apis.custom_objects,
                                        virtual_server_setup.vs_name,
                                        f"{TEST_DATA}/virtual-server-grpc/virtual-server-invalid-type.yaml",
                                        virtual_server_setup.namespace)
        wait_before_test()
        text = f"{virtual_server_setup.namespace}/{virtual_server_setup.vs_name}"
        event_text1 = f"VirtualServer {text} was rejected with error:"
        invalid_fields1 = ["spec.upstreams[0].type"]
        vs_events1 = get_events(kube_apis.v1, virtual_server_setup.namespace)
        assert_event_starts_with_text_and_contains_errors(event_text1, vs_events1, invalid_fields1)

        self.patch_valid_vs(kube_apis, virtual_server_setup)
        wait_before_test()

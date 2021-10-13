import grpc
import pytest
from settings import TEST_DATA, DEPLOYMENTS
from suite.ap_resources_utils import (
    create_ap_logconf_from_yaml,
    create_ap_policy_from_yaml,
    delete_ap_policy,
    delete_ap_logconf,
)
from suite.grpc.helloworld_pb2 import HelloRequest
from suite.grpc.helloworld_pb2_grpc import GreeterStub

from suite.resources_utils import (
    wait_before_test,
    create_example_app,
    wait_until_all_pods_are_ready,
    create_items_from_yaml,
    delete_items_from_yaml,
    delete_common_app,
    replace_configmap_from_yaml,
    create_ingress_with_ap_annotations,
    wait_before_test,
    get_file_contents,
    get_service_endpoint,
)
from suite.ssl_utils import get_certificate
from suite.yaml_utils import get_first_ingress_host_from_yaml

log_loc = f"/var/log/messages"
valid_resp_txt = "Hello"
invalid_resp_text = "The request was rejected. Please consult with your administrator."

class BackendSetup:
    """
    Encapsulate the example details.

    Attributes:
        ingress_host (str):
    """

    def __init__(self, ingress_host, ip, port_ssl):
        self.ingress_host = ingress_host
        self.ip = ip
        self.port_ssl = port_ssl


@pytest.fixture(scope="function")
def backend_setup(request, kube_apis, ingress_controller_endpoint, ingress_controller_prerequisites, test_namespace) -> BackendSetup:
    """
    Deploy a simple application and AppProtect manifests.

    :param request: pytest fixture
    :param kube_apis: client apis
    :param ingress_controller_endpoint: public endpoint
    :param test_namespace:
    :return: BackendSetup
    """
    try:
        print("------------------------- Replace ConfigMap with HTTP2 -------------------------")
        replace_configmap_from_yaml(kube_apis.v1,
                                ingress_controller_prerequisites.config_map['metadata']['name'],
                                ingress_controller_prerequisites.namespace,
                                f"{TEST_DATA}/appprotect/grpc/nginx-config.yaml")

        policy = request.param["policy"]
        print("------------------------- Deploy backend application -------------------------")
        create_example_app(kube_apis, "grpc", test_namespace)
        wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)

        print("------------------------- Deploy Secret -----------------------------")
        src_sec_yaml = f"{TEST_DATA}/appprotect/appprotect-secret.yaml"
        create_items_from_yaml(kube_apis, src_sec_yaml, test_namespace)

        print("------------------------- Deploy logconf -----------------------------")
        src_log_yaml = f"{TEST_DATA}/appprotect/logconf.yaml"
        log_name = create_ap_logconf_from_yaml(kube_apis.custom_objects, src_log_yaml, test_namespace)

        print(f"------------------------- Deploy appolicy: {policy} ---------------------------")
        src_pol_yaml = f"{TEST_DATA}/appprotect/grpc/{policy}.yaml"
        pol_name = create_ap_policy_from_yaml(kube_apis.custom_objects, src_pol_yaml, test_namespace)

        print("------------------------- Deploy Syslog -----------------------------")
        src_syslog_yaml = f"{TEST_DATA}/appprotect/syslog.yaml"
        create_items_from_yaml(kube_apis, src_syslog_yaml, test_namespace)
        syslog_ep = get_service_endpoint(kube_apis, "syslog-svc", test_namespace)
        print(syslog_ep)
        print("------------------------- Deploy ingress -----------------------------")
        src_ing_yaml = f"{TEST_DATA}/appprotect/grpc/ingress.yaml"
        create_ingress_with_ap_annotations(kube_apis, src_ing_yaml, test_namespace, policy, "True", "True", f"{syslog_ep}:514")
        ingress_host = get_first_ingress_host_from_yaml(src_ing_yaml)
        wait_before_test(40)
    except Exception as ex:
        print("Failed to complete setup, cleaning up..")
        delete_items_from_yaml(kube_apis, src_syslog_yaml, test_namespace)
        delete_items_from_yaml(kube_apis, src_ing_yaml, test_namespace)
        delete_ap_policy(kube_apis.custom_objects, pol_name, test_namespace)
        delete_ap_logconf(kube_apis.custom_objects, log_name, test_namespace)
        delete_common_app(kube_apis, "grpc", test_namespace)
        delete_items_from_yaml(kube_apis, src_sec_yaml, test_namespace)
        replace_configmap_from_yaml(kube_apis.v1,
                        ingress_controller_prerequisites.config_map['metadata']['name'],
                        ingress_controller_prerequisites.namespace,
                        f"{DEPLOYMENTS}/common/nginx-config.yaml")
        pytest.fail(f"AP GRPC setup failed")

    def fin():
        print("Clean up:")
        delete_items_from_yaml(kube_apis, src_syslog_yaml, test_namespace)
        delete_items_from_yaml(kube_apis, src_ing_yaml, test_namespace)
        delete_ap_policy(kube_apis.custom_objects, pol_name, test_namespace)
        delete_ap_logconf(kube_apis.custom_objects, log_name, test_namespace)
        delete_common_app(kube_apis, "grpc", test_namespace)
        delete_items_from_yaml(kube_apis, src_sec_yaml, test_namespace)
        replace_configmap_from_yaml(kube_apis.v1,
                        ingress_controller_prerequisites.config_map['metadata']['name'],
                        ingress_controller_prerequisites.namespace,
                        f"{DEPLOYMENTS}/common/nginx-config.yaml")

    request.addfinalizer(fin)

    return BackendSetup(ingress_host, ingress_controller_endpoint.public_ip, ingress_controller_endpoint.port_ssl)


@pytest.mark.skip_for_nginx_oss
@pytest.mark.appprotect
@pytest.mark.parametrize(
    "crd_ingress_controller_with_ap",
    [{"extra_args": [f"-enable-custom-resources", f"-enable-app-protect"]}],
    indirect=["crd_ingress_controller_with_ap"],
)
class TestAppProtect:
    @pytest.mark.smoke
    @pytest.mark.parametrize("backend_setup", [{"policy": "grpc-block-sayhello"}], indirect=True)
    def test_responses_grpc_block(
        self, kube_apis, crd_ingress_controller_with_ap, backend_setup, test_namespace
    ):
        """
        Test grpc-block-hello AppProtect policy: Blocks /sayhello gRPC method only
        Client sends request to /sayhello
        """
        syslog_pod = kube_apis.v1.list_namespaced_pod(test_namespace).items[-1].metadata.name

        # we need to get the cert so that it can be used in credentials in grpc.secure_channel to verify itself.
        # without verification, we will not be able to use the channel
        cert = get_certificate(backend_setup.ip, backend_setup.ingress_host, backend_setup.port_ssl)

        target = f'{backend_setup.ip}:{backend_setup.port_ssl}'
        credentials = grpc.ssl_channel_credentials(root_certificates=cert.encode())

        # this option is necessary to set the SNI of a gRPC connection and it only works with grpc.secure_channel.
        # also, the TLS cert for the Ingress must have the CN equal to backend_setup.ingress_host
        options = (('grpc.ssl_target_name_override', backend_setup.ingress_host),)

        with grpc.secure_channel(target, credentials, options) as channel:
            stub = GreeterStub(channel)
            ex = ""
            try:
                stub.SayHello(HelloRequest(name=backend_setup.ip))
                pytest.fail("RPC error was expected during call, exiting...")
            except grpc.RpcError as e:
                # grpc.RpcError is also grpc.Call https://grpc.github.io/grpc/python/grpc.html#client-side-context
                ex = e.details()
                print(ex)
                
        log_contents = get_file_contents(kube_apis.v1, log_loc, syslog_pod, test_namespace)
        assert (
            invalid_resp_text in ex and
            'ASM:attack_type="Directory Indexing"' in log_contents and
            'violations="Illegal gRPC method"' in log_contents and
            'severity="Error"' in log_contents and
            'outcome="REJECTED"' in log_contents
        )

    @pytest.mark.parametrize("backend_setup", [{"policy": "grpc-block-saygoodbye"}], indirect=True)
    def test_responses_grpc_allow(
        self, kube_apis, crd_ingress_controller_with_ap, backend_setup, test_namespace, ingress_controller_endpoint
    ):
        """
        Test grpc-block-goodbye AppProtect policy: Blocks /saygoodbye gRPC method only
        Client sends request to /sayhello thus should pass
        """
        syslog_pod = kube_apis.v1.list_namespaced_pod(test_namespace).items[-1].metadata.name
        cert = get_certificate(backend_setup.ip, backend_setup.ingress_host, backend_setup.port_ssl)

        target = f'{backend_setup.ip}:{backend_setup.port_ssl}'
        credentials = grpc.ssl_channel_credentials(root_certificates=cert.encode())
        options = (('grpc.ssl_target_name_override', backend_setup.ingress_host),)

        with grpc.secure_channel(target, credentials, options) as channel:
            stub = GreeterStub(channel)
            response = ""
            try:
                response = stub.SayHello(HelloRequest(name=backend_setup.ip))
                print(response)
            except grpc.RpcError as e:
                print(e.details())
                pytest.fail("RPC error was not expected during call, exiting...")
                
        log_contents = get_file_contents(kube_apis.v1, log_loc, syslog_pod, test_namespace)
        assert (
            valid_resp_txt in response.message and
            'ASM:attack_type="N/A"' in log_contents and
            'violations="N/A"' in log_contents and
            'severity="Informational"' in log_contents and
            'outcome="PASSED"' in log_contents
        )
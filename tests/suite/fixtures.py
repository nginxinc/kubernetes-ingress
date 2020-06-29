"""Describe project shared pytest fixtures."""

import time
import os
import pytest
import yaml

from kubernetes import config, client
from kubernetes.client import CoreV1Api, ExtensionsV1beta1Api, RbacAuthorizationV1beta1Api, CustomObjectsApi, \
    ApiextensionsV1beta1Api, AppsV1Api
from kubernetes.client.rest import ApiException

from suite.custom_resources_utils import create_virtual_server_from_yaml, \
    delete_virtual_server, create_v_s_route_from_yaml, delete_v_s_route, create_crd_from_yaml, delete_crd
from suite.kube_config_utils import ensure_context_in_config, get_current_context_name
from suite.resources_utils import create_namespace_with_name_from_yaml, delete_namespace, create_ns_and_sa_from_yaml, \
    patch_rbac, create_example_app, wait_until_all_pods_are_ready, delete_common_app, \
    ensure_connection_to_public_endpoint, create_service_with_name, create_deployment_with_name, delete_deployment, \
    delete_service, replace_configmap_from_yaml, delete_testing_namespaces
from suite.resources_utils import create_ingress_controller, delete_ingress_controller, configure_rbac, cleanup_rbac
from suite.resources_utils import create_service_from_yaml, get_service_node_ports, wait_for_public_ip
from suite.resources_utils import create_configmap_from_yaml, create_secret_from_yaml
from suite.yaml_utils import get_first_vs_host_from_yaml, get_paths_from_vs_yaml, get_paths_from_vsr_yaml, \
    get_route_namespace_from_vs_yaml, get_name_from_yaml

from settings import ALLOWED_SERVICE_TYPES, ALLOWED_IC_TYPES, DEPLOYMENTS, TEST_DATA, ALLOWED_DEPLOYMENT_TYPES


class KubeApis:
    """
    Encapsulate all the used kubernetes-client APIs.

    Attributes:
        v1: CoreV1Api
        extensions_v1_beta1: ExtensionsV1beta1Api
        rbac_v1_beta1: RbacAuthorizationV1beta1Api
        api_extensions_v1_beta1: ApiextensionsV1beta1Api
        custom_objects: CustomObjectsApi
    """
    def __init__(self, v1: CoreV1Api,
                 extensions_v1_beta1: ExtensionsV1beta1Api,
                 apps_v1_api: AppsV1Api,
                 rbac_v1_beta1: RbacAuthorizationV1beta1Api,
                 api_extensions_v1_beta1: ApiextensionsV1beta1Api,
                 custom_objects: CustomObjectsApi):
        self.v1 = v1
        self.extensions_v1_beta1 = extensions_v1_beta1
        self.apps_v1_api = apps_v1_api
        self.rbac_v1_beta1 = rbac_v1_beta1
        self.api_extensions_v1_beta1 = api_extensions_v1_beta1
        self.custom_objects = custom_objects


class PublicEndpoint:
    """
    Encapsulate the Public Endpoint info.

    Attributes:
        public_ip (str):
        port (int):
        port_ssl (int):
    """
    def __init__(self, public_ip, port=80, port_ssl=443, api_port=8080, metrics_port=9113):
        self.public_ip = public_ip
        self.port = port
        self.port_ssl = port_ssl
        self.api_port = api_port
        self.metrics_port = metrics_port


class IngressControllerPrerequisites:
    """
    Encapsulate shared items.

    Attributes:
        namespace (str): namespace name
        config_map (str): config_map name
    """
    def __init__(self, config_map, namespace):
        self.namespace = namespace
        self.config_map = config_map


@pytest.fixture(autouse=True)
def print_name() -> None:
    """Print out a current test name."""
    test_name = f"{os.environ.get('PYTEST_CURRENT_TEST').split(':')[2]} :: {os.environ.get('PYTEST_CURRENT_TEST').split(':')[4].split(' ')[0]}"
    print(f"\n============================= {test_name} =============================")


@pytest.fixture(scope="class")
def test_namespace(kube_apis) -> str:
    """
    Create a test namespace.

    :param kube_apis: client apis
    :return: str
    """
    timestamp = round(time.time() * 1000)
    print("------------------------- Create Test Namespace -----------------------------------")
    namespace = create_namespace_with_name_from_yaml(kube_apis.v1,
                                                     f"test-namespace-{str(timestamp)}",
                                                     f"{TEST_DATA}/common/ns.yaml")
    return namespace


@pytest.fixture(scope="session", autouse=True)
def delete_test_namespaces(kube_apis, request) -> None:
    """
    Delete all the testing namespaces.

    Testing namespaces are the ones starting with "test-namespace-"

    :param kube_apis: client apis
    :param request: pytest fixture
    :return: str
    """

    def fin():
        print("------------------------- Delete All Test Namespaces -----------------------------------")
        delete_testing_namespaces(kube_apis.v1)

    request.addfinalizer(fin)


@pytest.fixture(scope="class")
def ingress_controller(cli_arguments, kube_apis, ingress_controller_prerequisites, request) -> str:
    """
    Create Ingress Controller according to the context.

    :param cli_arguments: context
    :param kube_apis: client apis
    :param ingress_controller_prerequisites
    :param request: pytest fixture
    :return: IC name
    """
    namespace = ingress_controller_prerequisites.namespace
    name = "nginx-ingress"
    print("------------------------- Create IC without CRDs -----------------------------------")
    try:
        extra_args = request.param.get('extra_args', None)
        extra_args.append("-enable-custom-resources=false")
    except AttributeError:
        print("IC will start with CRDs disabled and without any additional cli-arguments")
        extra_args = ["-enable-custom-resources=false"]
    try:
        name = create_ingress_controller(kube_apis.v1, kube_apis.apps_v1_api, cli_arguments, namespace, extra_args)
    except ApiException as ex:
        # Finalizer doesn't start if fixture creation was incomplete, ensure clean up here
        print(f"Failed to complete IC fixture: {ex}\nClean up the cluster as much as possible.")
        delete_ingress_controller(kube_apis.apps_v1_api, name, cli_arguments['deployment-type'], namespace)

    def fin():
        print("Delete IC:")
        delete_ingress_controller(kube_apis.apps_v1_api, name, cli_arguments['deployment-type'], namespace)

    request.addfinalizer(fin)

    return name


@pytest.fixture(scope="session")
def ingress_controller_endpoint(cli_arguments, kube_apis, ingress_controller_prerequisites) -> PublicEndpoint:
    """
    Create an entry point for the IC.

    :param cli_arguments: tests context
    :param kube_apis: client apis
    :param ingress_controller_prerequisites: common cluster context
    :return: PublicEndpoint
    """
    print("------------------------- Create Public Endpoint  -----------------------------------")
    namespace = ingress_controller_prerequisites.namespace
    if cli_arguments["service"] == "nodeport":
        public_ip = cli_arguments["node-ip"]
        print(f"The Public IP: {public_ip}")
        service_name = create_service_from_yaml(kube_apis.v1,
                                                namespace,
                                                f"{TEST_DATA}/common/service/nodeport-with-additional-ports.yaml")
        port, port_ssl, api_port, metrics_port = get_service_node_ports(kube_apis.v1, service_name, namespace)
        return PublicEndpoint(public_ip, port, port_ssl, api_port, metrics_port)
    else:
        create_service_from_yaml(kube_apis.v1,
                                 namespace, f"{TEST_DATA}/common/service/loadbalancer-with-additional-ports.yaml")
        public_ip = wait_for_public_ip(kube_apis.v1, namespace)
        print(f"The Public IP: {public_ip}")
        return PublicEndpoint(public_ip)


@pytest.fixture(scope="session")
def ingress_controller_prerequisites(cli_arguments, kube_apis, request) -> IngressControllerPrerequisites:
    """
    Create RBAC, SA, IC namespace and default-secret.

    :param cli_arguments: tests context
    :param kube_apis: client apis
    :param request: pytest fixture
    :return: IngressControllerPrerequisites
    """
    print("------------------------- Create IC Prerequisites  -----------------------------------")
    rbac = configure_rbac(kube_apis.rbac_v1_beta1)
    namespace = create_ns_and_sa_from_yaml(kube_apis.v1, f"{DEPLOYMENTS}/common/ns-and-sa.yaml")
    config_map_yaml = f"{DEPLOYMENTS}/common/nginx-config.yaml"
    create_configmap_from_yaml(kube_apis.v1, namespace, config_map_yaml)
    with open(config_map_yaml) as f:
        config_map = yaml.safe_load(f)
    create_secret_from_yaml(kube_apis.v1, namespace, f"{DEPLOYMENTS}/common/default-server-secret.yaml")

    def fin():
        print("Clean up prerequisites")
        delete_namespace(kube_apis.v1, namespace)
        cleanup_rbac(kube_apis.rbac_v1_beta1, rbac)

    request.addfinalizer(fin)

    return IngressControllerPrerequisites(config_map, namespace)


@pytest.fixture(scope="session")
def kube_apis(cli_arguments) -> KubeApis:
    """
    Set up kubernets-client to operate in cluster.

    :param cli_arguments: a set of command-line arguments
    :return: KubeApis
    """
    context_name = cli_arguments['context']
    kubeconfig = cli_arguments['kubeconfig']
    config.load_kube_config(config_file=kubeconfig, context=context_name, persist_config=False)
    v1 = client.CoreV1Api()
    extensions_v1_beta1 = client.ExtensionsV1beta1Api()
    apps_v1_api = client.AppsV1Api()
    rbac_v1_beta1 = client.RbacAuthorizationV1beta1Api()
    api_extensions_v1_beta1 = client.ApiextensionsV1beta1Api()
    custom_objects = client.CustomObjectsApi()
    return KubeApis(v1, extensions_v1_beta1, apps_v1_api, rbac_v1_beta1, api_extensions_v1_beta1, custom_objects)


@pytest.fixture(scope="session", autouse=True)
def cli_arguments(request) -> {}:
    """
    Verify the CLI arguments.

    :param request: pytest fixture
    :return: {context, image, image-pull-policy, deployment-type, ic-type, service, node-ip, kubeconfig}
    """
    result = {"kubeconfig": request.config.getoption("--kubeconfig")}
    assert result["kubeconfig"] != "", "Empty kubeconfig is not allowed"
    print(f"\nTests will use this kubeconfig: {result['kubeconfig']}")

    result["context"] = request.config.getoption("--context")
    if result["context"] != "":
        ensure_context_in_config(result["kubeconfig"], result["context"])
        print(f"Tests will run against: {result['context']}")
    else:
        result["context"] = get_current_context_name(result["kubeconfig"])
        print(f"Tests will use a current context: {result['context']}")

    result["image"] = request.config.getoption("--image")
    assert result["image"] != "", "Empty image is not allowed"
    print(f"Tests will use the image: {result['image']}")

    result["image-pull-policy"] = request.config.getoption("--image-pull-policy")
    assert result["image-pull-policy"] != "", "Empty image-pull-policy is not allowed"
    print(f"Tests will run with the image-pull-policy: {result['image-pull-policy']}")

    result["deployment-type"] = request.config.getoption("--deployment-type")
    assert result["deployment-type"] in ALLOWED_DEPLOYMENT_TYPES, f"Deployment type {result['deployment-type']} is not allowed"
    print(f"Tests will use the IC deployment of type: {result['deployment-type']}")

    result["ic-type"] = request.config.getoption("--ic-type")
    assert result["ic-type"] in ALLOWED_IC_TYPES, f"IC type {result['ic-type']} is not allowed"
    print(f"Tests will run against the IC of type: {result['ic-type']}")

    result["service"] = request.config.getoption("--service")
    assert result["service"] in ALLOWED_SERVICE_TYPES, f"Service {result['service']} is not allowed"
    print(f"Tests will use Service of this type: {result['service']}")
    if result['service'] == "nodeport":
        node_ip = request.config.getoption("--node-ip", None)
        assert node_ip is not None and node_ip != "", f"Service 'nodeport' requires a node-ip"
        result["node-ip"] = node_ip
        print(f"Tests will use the node-ip: {result['node-ip']}")
    return result


@pytest.fixture(scope="class")
def crd_ingress_controller(cli_arguments, kube_apis, ingress_controller_prerequisites,
                           ingress_controller_endpoint,
                           request) -> None:
    """
    Create an Ingress Controller with CRD enabled.

    :param cli_arguments: pytest context
    :param kube_apis: client apis
    :param ingress_controller_prerequisites
    :param ingress_controller_endpoint:
    :param request: pytest fixture to parametrize this method
        {type: complete|rbac-without-vs, extra_args: }
        'type' type of test pre-configuration
        'extra_args' list of IC cli arguments
    :return:
    """
    namespace = ingress_controller_prerequisites.namespace
    name = "nginx-ingress"
    vs_crd_name = get_name_from_yaml(f"{DEPLOYMENTS}/common/vs-definition.yaml")
    vsr_crd_name = get_name_from_yaml(f"{DEPLOYMENTS}/common/vsr-definition.yaml")
    pol_crd_name = get_name_from_yaml(f"{DEPLOYMENTS}/common/policy-definition.yaml")
    ts_crd_name = get_name_from_yaml(f"{DEPLOYMENTS}/common/ts-definition.yaml")
    gc_crd_name = get_name_from_yaml(f"{DEPLOYMENTS}/common/gc-definition.yaml")

    try:
        print("------------------------- Update ClusterRole -----------------------------------")
        if request.param['type'] == 'rbac-without-vs':
            patch_rbac(kube_apis.rbac_v1_beta1, f"{TEST_DATA}/virtual-server/rbac-without-vs.yaml")
        print("------------------------- Register CRDs -----------------------------------")
        create_crd_from_yaml(kube_apis.api_extensions_v1_beta1, vs_crd_name,
                             f"{DEPLOYMENTS}/common/vs-definition.yaml")
        create_crd_from_yaml(kube_apis.api_extensions_v1_beta1, vsr_crd_name,
                             f"{DEPLOYMENTS}/common/vsr-definition.yaml")
        create_crd_from_yaml(kube_apis.api_extensions_v1_beta1, pol_crd_name,
                             f"{DEPLOYMENTS}/common/policy-definition.yaml")
        create_crd_from_yaml(kube_apis.api_extensions_v1_beta1, ts_crd_name,
                             f"{DEPLOYMENTS}/common/ts-definition.yaml")
        create_crd_from_yaml(kube_apis.api_extensions_v1_beta1, gc_crd_name,
                             f"{DEPLOYMENTS}/common/gc-definition.yaml")
        print("------------------------- Create IC -----------------------------------")
        name = create_ingress_controller(kube_apis.v1, kube_apis.apps_v1_api, cli_arguments, namespace,
                                         request.param.get('extra_args', None))
        ensure_connection_to_public_endpoint(ingress_controller_endpoint.public_ip,
                                             ingress_controller_endpoint.port,
                                             ingress_controller_endpoint.port_ssl)
    except ApiException as ex:
        # Finalizer method doesn't start if fixture creation was incomplete, ensure clean up here
        print(f"Failed to complete CRD IC fixture: {ex}\nClean up the cluster as much as possible.")
        delete_crd(kube_apis.api_extensions_v1_beta1, vs_crd_name)
        delete_crd(kube_apis.api_extensions_v1_beta1, vsr_crd_name)
        delete_crd(kube_apis.api_extensions_v1_beta1, pol_crd_name)
        delete_crd(kube_apis.api_extensions_v1_beta1, ts_crd_name)
        delete_crd(kube_apis.api_extensions_v1_beta1, gc_crd_name)
        print("Restore the ClusterRole:")
        patch_rbac(kube_apis.rbac_v1_beta1, f"{DEPLOYMENTS}/rbac/rbac.yaml")
        print("Remove the IC:")
        delete_ingress_controller(kube_apis.apps_v1_api, name, cli_arguments['deployment-type'], namespace)

    def fin():
        delete_crd(kube_apis.api_extensions_v1_beta1, vs_crd_name)
        delete_crd(kube_apis.api_extensions_v1_beta1, vsr_crd_name)
        delete_crd(kube_apis.api_extensions_v1_beta1, pol_crd_name)
        delete_crd(kube_apis.api_extensions_v1_beta1, ts_crd_name)
        delete_crd(kube_apis.api_extensions_v1_beta1, gc_crd_name)
        print("Restore the ClusterRole:")
        patch_rbac(kube_apis.rbac_v1_beta1, f"{DEPLOYMENTS}/rbac/rbac.yaml")
        print("Remove the IC:")
        delete_ingress_controller(kube_apis.apps_v1_api, name, cli_arguments['deployment-type'], namespace)

    request.addfinalizer(fin)


class VirtualServerSetup:
    """
    Encapsulate  Virtual Server Example details.

    Attributes:
        public_endpoint (PublicEndpoint):
        namespace (str):
        vs_host (str):
        vs_name (str):
        backend_1_url (str):
        backend_2_url (str):
    """
    def __init__(self, public_endpoint: PublicEndpoint, namespace, vs_host, vs_name, vs_paths):
        self.public_endpoint = public_endpoint
        self.namespace = namespace
        self.vs_host = vs_host
        self.vs_name = vs_name
        self.backend_1_url = f"http://{public_endpoint.public_ip}:{public_endpoint.port}{vs_paths[0]}"
        self.backend_2_url = f"http://{public_endpoint.public_ip}:{public_endpoint.port}{vs_paths[1]}"
        self.backend_1_url_ssl = f"https://{public_endpoint.public_ip}:{public_endpoint.port_ssl}{vs_paths[0]}"
        self.backend_2_url_ssl = f"https://{public_endpoint.public_ip}:{public_endpoint.port_ssl}{vs_paths[1]}"


@pytest.fixture(scope="class")
def virtual_server_setup(request, kube_apis, crd_ingress_controller, ingress_controller_endpoint,
                         test_namespace) -> VirtualServerSetup:
    """
    Prepare Virtual Server Example.

    :param request: internal pytest fixture to parametrize this method:
        {example: virtual-server|virtual-server-tls|..., app_type: simple|split|...}
        'example' is a directory name in TEST_DATA,
        'app_type' is a directory name in TEST_DATA/common/app
    :param kube_apis: client apis
    :param crd_ingress_controller:
    :param ingress_controller_endpoint:
    :param test_namespace:
    :return: VirtualServerSetup
    """
    print("------------------------- Deploy Virtual Server Example -----------------------------------")
    vs_source = f"{TEST_DATA}/{request.param['example']}/standard/virtual-server.yaml"
    vs_name = create_virtual_server_from_yaml(kube_apis.custom_objects,
                                              vs_source,
                                              test_namespace)
    vs_host = get_first_vs_host_from_yaml(vs_source)
    vs_paths = get_paths_from_vs_yaml(vs_source)
    if request.param['app_type']:
        create_example_app(kube_apis, request.param['app_type'], test_namespace)
        wait_until_all_pods_are_ready(kube_apis.v1, test_namespace)

    def fin():
        print("Clean up Virtual Server Example:")
        delete_virtual_server(kube_apis.custom_objects, vs_name, test_namespace)
        if request.param['app_type']:
            delete_common_app(kube_apis, request.param['app_type'], test_namespace)

    request.addfinalizer(fin)

    return VirtualServerSetup(ingress_controller_endpoint, test_namespace, vs_host, vs_name, vs_paths)


@pytest.fixture(scope="class")
def v_s_route_app_setup(request, kube_apis, v_s_route_setup) -> None:
    """
    Prepare an example app for Virtual Server Route.

    1st namespace with backend1-svc and backend3-svc and deployment and 2nd namespace with backend2-svc and deployment.

    :param request: internal pytest fixture
    :param kube_apis: client apis
    :param v_s_route_setup:
    :return:
    """
    print("---------------------- Deploy a VS Route Example Application ----------------------------")
    svc_one = create_service_with_name(kube_apis.v1, v_s_route_setup.route_m.namespace, "backend1-svc")
    svc_three = create_service_with_name(kube_apis.v1, v_s_route_setup.route_m.namespace, "backend3-svc")
    deployment_one = create_deployment_with_name(kube_apis.apps_v1_api,
                                                 v_s_route_setup.route_m.namespace,
                                                 "backend1")
    deployment_three = create_deployment_with_name(kube_apis.apps_v1_api,
                                                   v_s_route_setup.route_m.namespace,
                                                   "backend3")

    svc_two = create_service_with_name(kube_apis.v1, v_s_route_setup.route_s.namespace, "backend2-svc")
    deployment_two = create_deployment_with_name(kube_apis.apps_v1_api,
                                                 v_s_route_setup.route_s.namespace, "backend2")

    wait_until_all_pods_are_ready(kube_apis.v1, v_s_route_setup.route_m.namespace)
    wait_until_all_pods_are_ready(kube_apis.v1, v_s_route_setup.route_s.namespace)

    def fin():
        print("Clean up the Application:")
        delete_deployment(kube_apis.apps_v1_api, deployment_one, v_s_route_setup.route_m.namespace)
        delete_service(kube_apis.v1, svc_one, v_s_route_setup.route_m.namespace)
        delete_deployment(kube_apis.apps_v1_api, deployment_three, v_s_route_setup.route_m.namespace)
        delete_service(kube_apis.v1, svc_three, v_s_route_setup.route_m.namespace)
        delete_deployment(kube_apis.apps_v1_api, deployment_two, v_s_route_setup.route_s.namespace)
        delete_service(kube_apis.v1, svc_two, v_s_route_setup.route_s.namespace)

    request.addfinalizer(fin)


class VirtualServerRoute:
    """
    Encapsulate  Virtual Server Route details.

    Attributes:
        namespace (str):
        name (str):
        paths ([]):
    """

    def __init__(self, namespace, name, paths):
        self.namespace = namespace
        self.name = name
        self.paths = paths


class VirtualServerRouteSetup:
    """
    Encapsulate Virtual Server Example details.

    Attributes:
        public_endpoint (PublicEndpoint):
        namespace (str):
        vs_host (str):
        vs_name (str):
        route_m (VirtualServerRoute): route with multiple subroutes
        route_s (VirtualServerRoute): route with single subroute
    """

    def __init__(self, public_endpoint: PublicEndpoint,
                 namespace, vs_host, vs_name, route_m: VirtualServerRoute, route_s: VirtualServerRoute):
        self.public_endpoint = public_endpoint
        self.namespace = namespace
        self.vs_host = vs_host
        self.vs_name = vs_name
        self.route_m = route_m
        self.route_s = route_s


@pytest.fixture(scope="class")
def v_s_route_setup(request,
                    kube_apis,
                    crd_ingress_controller,
                    ingress_controller_endpoint) -> VirtualServerRouteSetup:
    """
    Prepare Virtual Server Route Example.

    1st namespace with VS and 1st addressed VSR and 2nd namespace with second addressed VSR.

    :param request: internal pytest fixture to parametrize this method:
        {example: virtual-server|virtual-server-tls|...}
        'example' is a directory name in TEST_DATA
    :param kube_apis: client apis
    :param crd_ingress_controller:
    :param ingress_controller_endpoint:

    :return: VirtualServerRouteSetup
    """
    vs_routes_ns = get_route_namespace_from_vs_yaml(
        f"{TEST_DATA}/{request.param['example']}/standard/virtual-server.yaml")
    ns_1 = create_namespace_with_name_from_yaml(kube_apis.v1,
                                                vs_routes_ns[0],
                                                f"{TEST_DATA}/common/ns.yaml")
    ns_2 = create_namespace_with_name_from_yaml(kube_apis.v1,
                                                vs_routes_ns[1],
                                                f"{TEST_DATA}/common/ns.yaml")
    print("------------------------- Deploy Virtual Server -----------------------------------")
    vs_name = create_virtual_server_from_yaml(kube_apis.custom_objects,
                                              f"{TEST_DATA}/{request.param['example']}/standard/virtual-server.yaml",
                                              ns_1)
    vs_host = get_first_vs_host_from_yaml(f"{TEST_DATA}/{request.param['example']}/standard/virtual-server.yaml")

    print("------------------------- Deploy Virtual Server Routes -----------------------------------")
    vsr_m_name = create_v_s_route_from_yaml(kube_apis.custom_objects,
                                            f"{TEST_DATA}/{request.param['example']}/route-multiple.yaml",
                                            ns_1)
    vsr_m_paths = get_paths_from_vsr_yaml(f"{TEST_DATA}/{request.param['example']}/route-multiple.yaml")
    route_m = VirtualServerRoute(ns_1, vsr_m_name, vsr_m_paths)

    vsr_s_name = create_v_s_route_from_yaml(kube_apis.custom_objects,
                                            f"{TEST_DATA}/{request.param['example']}/route-single.yaml",
                                            ns_2)
    vsr_s_paths = get_paths_from_vsr_yaml(f"{TEST_DATA}/{request.param['example']}/route-single.yaml")
    route_s = VirtualServerRoute(ns_2, vsr_s_name, vsr_s_paths)

    def fin():
        print("Clean up the Virtual Server Route:")
        delete_v_s_route(kube_apis.custom_objects, vsr_m_name, ns_1)
        delete_v_s_route(kube_apis.custom_objects, vsr_s_name, ns_2)
        print("Clean up Virtual Server:")
        delete_virtual_server(kube_apis.custom_objects, vs_name, ns_1)
        print("Delete test namespaces")
        delete_namespace(kube_apis.v1, ns_1)
        delete_namespace(kube_apis.v1, ns_2)

    request.addfinalizer(fin)

    return VirtualServerRouteSetup(ingress_controller_endpoint,
                                   ns_1, vs_host, vs_name, route_m, route_s)


@pytest.fixture(scope="function")
def restore_configmap(request, kube_apis, ingress_controller_prerequisites, test_namespace) -> None:
    """
    Return ConfigMap to the initial state after the test.

    :param request: internal pytest fixture
    :param kube_apis: client apis
    :param ingress_controller_prerequisites:
    :param test_namespace: str
    :return:
    """

    def fin():
        replace_configmap_from_yaml(kube_apis.v1,
                                    ingress_controller_prerequisites.config_map['metadata']['name'],
                                    ingress_controller_prerequisites.namespace,
                                    f"{DEPLOYMENTS}/common/nginx-config.yaml")

    request.addfinalizer(fin)

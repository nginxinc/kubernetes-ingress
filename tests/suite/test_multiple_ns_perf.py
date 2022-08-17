import time
import pytest
import yaml
import subprocess

from suite.resources_utils import (
    create_namespace_with_name_from_yaml,
    delete_namespace,
    create_ingress_controller,
    delete_ingress_controller,
    create_example_app,
    wait_until_all_pods_are_ready,
    create_secret_from_yaml,
    create_ingress,
    get_test_file_name,
    write_to_json,
)
from suite.custom_resources_utils import (
    get_pod_metrics,
)
watched_namespaces=""
from suite.yaml_utils import get_first_ingress_host_from_yaml
from settings import TEST_DATA

@pytest.fixture(scope="class")
def ingress_ns_setup(
    request,
    kube_apis,
) -> None:
    """
    Create and deploy namespaces, apps and ingresses

    :param request: pytest fixture
    :param kube_apis: client apis
    """
    
    manifest = f"{TEST_DATA}/smoke/standard/smoke-ingress.yaml"
    ns_count = int(request.config.getoption("--ns-count"))
    multi_ns=""
    for i in range(1, ns_count + 1):
        watched_namespace = create_namespace_with_name_from_yaml(
            kube_apis.v1, f"ns-{i}", f"{TEST_DATA}/common/ns.yaml"
        )
        multi_ns = multi_ns+f"{watched_namespace},"
        create_example_app(kube_apis, "simple", watched_namespace)
        secret_name = create_secret_from_yaml(
            kube_apis.v1, watched_namespace, f"{TEST_DATA}/smoke/smoke-secret.yaml"
        )
        with open(manifest) as f:
                doc = yaml.safe_load(f)
                doc["metadata"]["name"] = f"smoke-ingress-{i}"
                doc["spec"]["rules"][0]["host"] = f"smoke-{i}.example.com"
                create_ingress(kube_apis.networking_v1, watched_namespace, doc)
    global watched_namespaces            
    watched_namespaces = multi_ns[:-1]
    for i in range(1, ns_count + 1):
        wait_until_all_pods_are_ready(kube_apis.v1, f"ns-{i}")

    def fin():
        for i in range(1, ns_count + 1):
            delete_namespace(kube_apis.v1, f"ns-{i}")
    request.addfinalizer(fin)

@pytest.mark.multi_ns
class TestMultipleSimpleIngress:
    """Test to output CPU/Memory perf metrics for pods with multiple namespaces"""
    def test_ingress_multi_ns(
        self,
        request,
        kube_apis,
        cli_arguments,
        ingress_ns_setup,
        ingress_controller_prerequisites,
    ):  
        metric_dict = {}
        namespace = ingress_controller_prerequisites.namespace
        extra_args = ["-enable-custom-resources=false", f"-watch-namespace={watched_namespaces}"]
        name = create_ingress_controller(kube_apis.v1, kube_apis.apps_v1_api, cli_arguments, namespace, extra_args)
        metrics = get_pod_metrics(request,namespace)
        metric_dict[f"{request.node.name}+{time.time()}"] = metrics
        write_to_json(
            f"pod-metrics-{get_test_file_name(request.node.fspath)}.json",
            metric_dict
        )
        delete_ingress_controller(kube_apis.apps_v1_api, name, cli_arguments['deployment-type'], namespace)

        assert metrics



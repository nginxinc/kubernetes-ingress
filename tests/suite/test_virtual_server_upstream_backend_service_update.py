import datetime

import pytest
from kubernetes.client.rest import ApiException
from settings import TEST_DATA
from suite.utils.resources_utils import (
    ensure_connection_to_public_endpoint,
    get_first_pod_name,
    scale_deployment,
    wait_before_test,
)
from suite.utils.vs_vsr_resources_utils import get_vs_nginx_template_conf


def restart_deployment(v1_apps, deployment, namespace):
    now = datetime.datetime.utcnow()
    now = str(now.isoformat("T") + "Z")
    body = {"spec": {"template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": now}}}}}
    try:
        v1_apps.patch_namespaced_deployment(deployment, namespace, body, pretty="true")
    except ApiException as e:
        print("Exception when calling AppsV1Api->read_namespaced_deployment_status: %s\n" % e)


@pytest.mark.upstream_rollout
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {"type": "complete", "extra_args": [f"-enable-custom-resources"]},
            {"example": "virtual-server-upstream-options", "app_type": "simple"},
        )
    ],
    indirect=True,
)
class TestVirtualServerUpstreamUpdate:
    def test_nginx_config_defaults(
        self, kube_apis, ingress_controller_prerequisites, crd_ingress_controller, virtual_server_setup
    ):
        scale_deployment(kube_apis.v1, kube_apis.apps_v1_api, "backend1", virtual_server_setup.namespace, 10)
        wait_before_test(3)

        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        ## Get list of endpoints in backend1 service
        resp = kube_apis.v1.list_namespaced_endpoints(virtual_server_setup.namespace)
        end_point_slices = []
        port = resp.items[0].subsets[0].ports[0].port
        for a in resp.items[0].subsets[0].addresses:
            slice = {"ip": a.ip, "port": port}
            end_point_slices.append(slice)

        ## Ensure pods are running
        pods = kube_apis.v1.list_namespaced_pod(virtual_server_setup.namespace, label_selector="app=backend1")
        for p in pods.items:
            for s in end_point_slices:
                if s["ip"] != p.status.pod_ip:
                    continue
                s["pod"] = p.metadata.name
                print(f"{p.metadata.name}/{s['ip']}:{s['port']} - {p.status.phase}")
                if p.status.phase != "Running":
                    wait_before_test(1)
                s["status"] = p.status.phase

        for s in end_point_slices:
            print(s)
            assert f"{s['ip']}:{s['port']}" in config

        ensure_connection_to_public_endpoint(
            virtual_server_setup.public_endpoint.public_ip,
            virtual_server_setup.public_endpoint.port,
            virtual_server_setup.public_endpoint.port_ssl,
        )

        ## restart deployment and wait for deployment to complete
        restart_deployment(kube_apis.apps_v1_api, "backend1", virtual_server_setup.namespace)
        retry = 3600
        count = 0
        for i in range(count, retry):
            ensure_connection_to_public_endpoint(
                virtual_server_setup.public_endpoint.public_ip,
                virtual_server_setup.public_endpoint.port,
                virtual_server_setup.public_endpoint.port_ssl,
            )
            wait_before_test(0.5)

        latest_ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        latest_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            latest_ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )

        ## Get updated list of endpoints in backend1 service
        resp = kube_apis.v1.list_namespaced_endpoints(virtual_server_setup.namespace)
        latest_end_point_slices = []
        port = resp.items[0].subsets[0].ports[0].port
        for a in resp.items[0].subsets[0].addresses:
            slice = {"ip": a.ip, "port": port}
            latest_end_point_slices.append(slice)

        ## Ensure pods are running
        pods = kube_apis.v1.list_namespaced_pod(virtual_server_setup.namespace, label_selector="app=backend1")
        for p in pods.items:
            for s in latest_end_point_slices:
                if s["ip"] != p.status.pod_ip:
                    continue
                s["pod"] = p.metadata.name
                print(f"{p.metadata.name}/{s['ip']}:{s['port']} - {p.status.phase}")
                if p.status.phase != "Running":
                    wait_before_test(1)
                s["status"] = p.status.phase

        for s in latest_end_point_slices:
            print(s)
            assert f"{s['ip']}:{s['port']}" in latest_config

        ensure_connection_to_public_endpoint(
            virtual_server_setup.public_endpoint.public_ip,
            virtual_server_setup.public_endpoint.port,
            virtual_server_setup.public_endpoint.port_ssl,
        )

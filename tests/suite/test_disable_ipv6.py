import pytest
from suite.resources_utils import (
    get_first_pod_name,
    get_nginx_template_conf,
    get_ts_nginx_template_conf,
    wait_before_test,
)
from suite.vs_vsr_resources_utils import get_vs_nginx_template_conf


@pytest.mark.vs
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup, transport_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [
                    "-enable-custom-resources",
                    "-global-configuration=nginx-ingress/nginx-configuration",
                    "-disable-ipv6=true",
                ],
            },
            {"example": "virtual-server-status", "app_type": "simple"},
            {"example": "transport-server-status", "app_type": "simple"},
        )
    ],
    indirect=True,
)
class TestDisableIpv6:
    def test_ipv6_is_disabled(
        self,
        kube_apis,
        ingress_controller_prerequisites,
        crd_ingress_controller,
        virtual_server_setup,
        transport_server_setup,
    ):
        wait_before_test()
        ic_pod_name = get_first_pod_name(kube_apis.v1, ingress_controller_prerequisites.namespace)
        nginx_config = get_nginx_template_conf(kube_apis.v1, ingress_controller_prerequisites.namespace)
        ts_config = get_ts_nginx_template_conf(
            kube_apis.v1,
            transport_server_setup.namespace,
            transport_server_setup.name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )
        vs_config = get_vs_nginx_template_conf(
            kube_apis.v1,
            virtual_server_setup.namespace,
            virtual_server_setup.vs_name,
            ic_pod_name,
            ingress_controller_prerequisites.namespace,
        )
        assert "listen [::]:" not in nginx_config
        assert "listen [::]:" not in vs_config
        assert "listen [::]:" not in ts_config

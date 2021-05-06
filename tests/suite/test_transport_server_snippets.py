import pytest

from suite.resources_utils import (
    wait_before_test,
    get_ts_nginx_template_conf,
)
from suite.custom_resources_utils import (
    patch_ts,
)
from settings import TEST_DATA

@pytest.mark.ts
@pytest.mark.parametrize(
    "crd_ingress_controller, transport_server_setup",
    [
        (
            {   "type": "complete",
                "extra_args":
                    [
                        "-global-configuration=nginx-ingress/nginx-configuration",
                        "-enable-leader-election=false",
                        "-enable-snippets",
                    ]
            },
            {"example": "transport-server-status"},
        )
    ],
    indirect=True,
)
class TestTransportServerSnippets:

    def test_snippets(
        self, kube_apis, crd_ingress_controller, transport_server_setup, ingress_controller_prerequisites
    ):
        """
        Test snippets are present in conf when enabled
        """
        patch_src = f"{TEST_DATA}/transport-server-snippets/transport-server-snippets.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            patch_src,
            transport_server_setup.namespace,
        )
        wait_before_test()

        conf = get_ts_nginx_template_conf(
            kube_apis.v1,
            transport_server_setup.namespace,
            transport_server_setup.name,
            transport_server_setup.ingress_pod_name,
            ingress_controller_prerequisites.namespace
        )
        print(conf)

        std_src = f"{TEST_DATA}/transport-server-status/standard/transport-server.yaml"
        patch_ts(
            kube_apis.custom_objects,
            transport_server_setup.name,
            std_src,
            transport_server_setup.namespace,
        )

        assert (
            "limit_conn_zone $binary_remote_addr zone=addr:10m;" in conf # stream-snippets
            and "limit_conn addr 1;" in conf # server-snippets
        )
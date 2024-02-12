import json
import re
import subprocess
from datetime import datetime

import pytest
import requests
from common import collect_prom_reload_metrics, run_perf
from suite.utils.resources_utils import wait_before_test

reload = []


@pytest.fixture(scope="class")
def collect(request, kube_apis, ingress_controller_endpoint, test_namespace) -> None:
    def fin():
        with open("reload_vs.json", "w+") as f:
            json.dump(reload, f, ensure_ascii=False, indent=4)

    request.addfinalizer(fin)


@pytest.fixture
def setup_users(request):
    return request.config.getoption("--users")


@pytest.fixture
def setup_rate(request):
    return request.config.getoption("--hatch-rate")


@pytest.fixture
def setup_time(request):
    return request.config.getoption("--time")


@pytest.mark.perf
@pytest.mark.parametrize(
    "crd_ingress_controller, virtual_server_setup",
    [
        (
            {
                "type": "complete",
                "extra_args": [f"-enable-custom-resources", f"-enable-prometheus-metrics"],
            },
            {
                "example": "virtual-server",
                "app_type": "simple",
            },
        )
    ],
    indirect=True,
)
class TestVirtualServerPerf:
    def test_vs_perf(
        self,
        kube_apis,
        ingress_controller_endpoint,
        crd_ingress_controller,
        virtual_server_setup,
        collect,
        setup_rate,
        setup_time,
        setup_users,
    ):
        wait_before_test()
        resp = requests.get(
            virtual_server_setup.backend_1_url,
            headers={"host": virtual_server_setup.vs_host},
        )
        assert resp.status_code == 200
        collect_prom_reload_metrics(
            reload,
            "VS resource",
            ingress_controller_endpoint.public_ip,
            ingress_controller_endpoint.metrics_port,
        )

        run_perf(virtual_server_setup.backend_1_url, setup_users, setup_rate, setup_time, "vs")

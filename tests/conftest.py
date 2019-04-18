"""Describe overall framework configuration."""

import os
import pytest

from kubernetes.config.kube_config import KUBE_CONFIG_DEFAULT_LOCATION
from settings import DEFAULT_IMAGE, DEFAULT_PULL_POLICY, DEFAULT_IC_TYPE, DEFAULT_SERVICE, DEFAULT_DEPLOYMENT_TYPE


def pytest_addoption(parser) -> None:
    """Get cli-arguments.

    :param parser: pytest parser
    :return:
    """
    parser.addoption("--context",
                     action="store", default="", help="The context to use in the kubeconfig file.")
    parser.addoption("--image",
                     action="store", default=DEFAULT_IMAGE, help="The Ingress Controller image.")
    parser.addoption("--image-pull-policy",
                     action="store", default=DEFAULT_PULL_POLICY, help="The pull policy of the Ingress Controller image.")
    parser.addoption("--deployment-type",
                     action="store", default=DEFAULT_DEPLOYMENT_TYPE,
                     help="The type of the IC deployment: deployment or daemon-set.")
    parser.addoption("--ic-type",
                     action="store", default=DEFAULT_IC_TYPE, help="The type of the Ingress Controller: nginx-ingress or nginx-ingress-plus.")
    parser.addoption("--service",
                     action="store",
                     default=DEFAULT_SERVICE,
                     help="The type of the Ingress Controller service: nodeport or loadbalancer.")
    parser.addoption("--node-ip", action="store", help="The public IP of a cluster node. Not required if you use the loadbalancer service (see --service argument).")
    parser.addoption("--kubeconfig",
                     action="store",
                     default=os.path.expanduser(KUBE_CONFIG_DEFAULT_LOCATION),
                     help="An absolute path to a kubeconfig file.")


# import fixtures into pytest global namespace
pytest_plugins = [
    "suite.fixtures"
]


def pytest_collection_modifyitems(config, items) -> None:
    """
    Skip the tests marked with '@pytest.mark.skip_for_nginx_oss' for Nginx OSS runs.

    :param config: pytest config
    :param items: pytest collected test-items
    :return:
    """
    if config.getoption("--ic-type") == "nginx-ingress":
        skip_for_nginx_oss = pytest.mark.skip(reason="Skip a test for Nginx OSS")
        for item in items:
            if "skip_for_nginx_oss" in item.keywords:
                item.add_marker(skip_for_nginx_oss)

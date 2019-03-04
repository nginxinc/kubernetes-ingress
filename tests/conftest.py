"""Describe overall framework configuration."""

import os
import pytest

from kubernetes.config.kube_config import KUBE_CONFIG_DEFAULT_LOCATION
from settings import DEFAULT_IMAGE, DEFAULT_PULL_POLICY, DEFAULT_IC_TYPE, DEFAULT_SERVICE


def pytest_addoption(parser) -> None:
    """Get cli-arguments.

    :param parser: pytest parser
    :return:
    """
    parser.addoption("--context",
                     action="store", default="", help="context name as in the kubeconfig")
    parser.addoption("--image",
                     action="store", default=DEFAULT_IMAGE, help="image with tag (image:tag)")
    parser.addoption("--image-pull-policy",
                     action="store", default=DEFAULT_PULL_POLICY, help="image pull policy")
    parser.addoption("--ic-type",
                     action="store", default=DEFAULT_IC_TYPE, help="provide ic type")
    parser.addoption("--service",
                     action="store",
                     default=DEFAULT_SERVICE,
                     help="service type: nodeport or loadbalancer")
    parser.addoption("--node-ip", action="store", help="public IP of a cluster node")
    parser.addoption("--kubeconfig",
                     action="store",
                     default=os.path.expanduser(KUBE_CONFIG_DEFAULT_LOCATION),
                     help="an absolute path to kubeconfig")


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

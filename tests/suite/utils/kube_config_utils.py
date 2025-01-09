"""Describe methods to work with kubeconfig file."""

import pytest
import yaml


def get_current_context_name(kube_config) -> str:
    """
    Get current-context from kubeconfig.

    :param kube_config: absolute path to kubeconfig
    :return: str
    """
    with open(kube_config) as conf:
        dep = yaml.safe_load(conf)
        return dep["current-context"]


def ensure_context_in_config(kube_config, context_name) -> None:
    """
    Verify that kubeconfig contains specific context and fail if it doesn't.

    :param kube_config: absolute path to kubeconfig
    :param context_name: context name to verify
    :return:
    """
    with open(kube_config) as conf:
        dep = yaml.safe_load(conf)
        for contexts in dep["contexts"]:
            if contexts["name"] == context_name:
                return
    pytest.fail(f"Failed to find context '{context_name}' in the kubeconfig file: {kube_config}")

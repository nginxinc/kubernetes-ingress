"""Describe methods to utilize the Policy resource."""

import yaml
import logging
from kubernetes.client import CustomObjectsApi, ApiextensionsV1Api, CoreV1Api
from kubernetes.client.rest import ApiException
from suite.custom_resources_utils import read_custom_resource
from suite.resources_utils import ensure_item_removal


def read_policy(custom_objects: CustomObjectsApi, namespace, name) -> object:
    """
    Read Policy resource.
    """
    return read_custom_resource(custom_objects, namespace, "policies", name)


def create_policy_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> str:
    """
    Create a Policy based on yaml file.

    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: str
    """
    print("Create a Policy:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)
    try:
        custom_objects.create_namespaced_custom_object(
            "k8s.nginx.org", "v1", namespace, "policies", dep
        )
        print(f"Policy created with name '{dep['metadata']['name']}'")
        return dep["metadata"]["name"]
    except ApiException:
        logging.exception(f"Exception occurred while creating Policy: {dep['metadata']['name']}")
        raise


def delete_policy(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a Policy.

    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete a Policy: {name}")

    custom_objects.delete_namespaced_custom_object(
        "k8s.nginx.org", "v1", namespace, "policies", name
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "k8s.nginx.org",
        "v1",
        namespace,
        "policies",
        name,
    )
    print(f"Policy was removed with name '{name}'")

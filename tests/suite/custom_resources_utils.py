"""Describe methods to utilize the kubernetes-client."""
import pytest
import yaml

from kubernetes.client import CustomObjectsApi, ApiextensionsV1beta1Api
from kubernetes import client

from suite.resources_utils import ensure_item_removal


def create_crd_from_yaml(api_extensions_v1_beta1: ApiextensionsV1beta1Api, yaml_manifest) -> str:
    """
    Create a CRD based on yaml file.

    :param api_extensions_v1_beta1: ApiextensionsV1beta1Api
    :param yaml_manifest: an absolute path to file
    :return: str
    """
    print("Create CRD:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)

    try:
        api_extensions_v1_beta1.create_custom_resource_definition(dep)
    except Exception as ex:
        # https://github.com/kubernetes-client/python/issues/376
        if ex.args[0] == 'Invalid value for `conditions`, must not be `None`':
            print("There was an insignificant exception during the CRD creation. Continue...")
        else:
            pytest.fail(f"An unexpected exception {ex} occurred. Exiting...")
    print(f"CRD created with name '{dep['metadata']['name']}'")
    return dep['metadata']['name']


def delete_crd(api_extensions_v1_beta1: ApiextensionsV1beta1Api, name) -> None:
    """
    Delete a CRD.

    :param api_extensions_v1_beta1: ApiextensionsV1beta1Api
    :param name:
    :return:
    """
    print(f"Delete a CRD: {name}")
    delete_options = client.V1DeleteOptions()
    api_extensions_v1_beta1.delete_custom_resource_definition(name, delete_options)
    ensure_item_removal(api_extensions_v1_beta1.read_custom_resource_definition, name)
    print(f"CRD was removed with name '{name}'")


def create_virtual_server_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> str:
    """
    Create a VirtualServer based on yaml file.

    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: str
    """
    print("Create a VirtualServer:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)

    custom_objects.create_namespaced_custom_object("k8s.nginx.org", "v1alpha1", namespace, "virtualservers", dep)
    print(f"VirtualServer created with name '{dep['metadata']['name']}'")
    return dep['metadata']['name']


def delete_virtual_server(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a VirtualServer.

    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete a VirtualServer: {name}")
    delete_options = client.V1DeleteOptions()
    custom_objects.delete_namespaced_custom_object("k8s.nginx.org", "v1alpha1", namespace, "virtualservers", name, delete_options)
    ensure_item_removal(custom_objects.get_namespaced_custom_object, "k8s.nginx.org", "v1alpha1", namespace, "virtualservers", name)
    print(f"VirtualServer was removed with name '{name}'")


def patch_virtual_server_from_yaml(custom_objects: CustomObjectsApi, name, yaml_manifest, namespace) -> None:
    """
    Update a VS based on yaml manifest

    :param custom_objects: CustomObjectsApi
    :param name:
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return:
    """
    print(f"Update a VirtualServer: {name}")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)

    custom_objects.patch_namespaced_custom_object("k8s.nginx.org", "v1alpha1", namespace, "virtualservers", name, dep)
    print(f"VirtualServer updated with name '{dep['metadata']['name']}'")

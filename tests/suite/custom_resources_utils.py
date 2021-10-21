"""Describe methods to utilize the kubernetes-client."""
import pytest
import time
import yaml
import logging

from pprint import pprint
from kubernetes.client import CustomObjectsApi, ApiextensionsV1Api, CoreV1Api
from kubernetes.client.rest import ApiException

from suite.resources_utils import ensure_item_removal, get_file_contents


def create_crd(api_extensions_v1: ApiextensionsV1Api, body) -> None:
    """
    Create a CRD based on a dict

    :param api_extensions_v1: ApiextensionsV1Api
    :param body: a dict
    """
    try:
        api_extensions_v1.create_custom_resource_definition(body)
    except ApiException as api_ex:
        raise api_ex
    except Exception as ex:
        # https://github.com/kubernetes-client/python/issues/376
        if ex.args[0] == "Invalid value for `conditions`, must not be `None`":
            print("There was an insignificant exception during the CRD creation. Continue...")
        else:
            pytest.fail(f"An unexpected exception {ex} occurred. Exiting...")


def create_crd_from_yaml(
    api_extensions_v1: ApiextensionsV1Api, name, yaml_manifest
) -> None:
    """
    Create a specific CRD based on yaml file.

    :param api_extensions_v1: ApiextensionsV1Api
    :param name: CRD name
    :param yaml_manifest: an absolute path to file
    """
    print(f"Create a CRD with name: {name}")
    with open(yaml_manifest) as f:
        docs = yaml.safe_load_all(f)
        for dep in docs:
            if dep["metadata"]["name"] == name:
                create_crd(api_extensions_v1, dep)
                print("CRD was created")


def delete_crd(api_extensions_v1: ApiextensionsV1Api, name) -> None:
    """
    Delete a CRD.

    :param api_extensions_v1: ApiextensionsV1Api
    :param name:
    :return:
    """
    print(f"Delete a CRD: {name}")
    api_extensions_v1.delete_custom_resource_definition(name)
    ensure_item_removal(api_extensions_v1.read_custom_resource_definition, name)
    print(f"CRD was removed with name '{name}'")


def read_custom_resource(custom_objects: CustomObjectsApi, namespace, plural, name) -> object:
    """
    Get CRD information (kubectl describe output)

    :param custom_objects: CustomObjectsApi
    :param namespace: The custom resource's namespace	
    :param plural: the custom resource's plural name
    :param name: the custom object's name
    :return: object
    """
    print(f"Getting info for {name} in namespace {namespace}")
    try:
        response = custom_objects.get_namespaced_custom_object(
            "k8s.nginx.org", "v1", namespace, plural, name
        )
        pprint(response)
        return response

    except ApiException:
        logging.exception(f"Exception occurred while reading CRD")
        raise


def read_custom_resource_v1alpha1(custom_objects: CustomObjectsApi, namespace, plural, name) -> object:
    """
    Get CRD information (kubectl describe output)

    :param custom_objects: CustomObjectsApi
    :param namespace: The custom resource's namespace
    :param plural: the custom resource's plural name
    :param name: the custom object's name
    :return: object
    """
    print(f"Getting info for v1alpha1 crd {name} in namespace {namespace}")
    try:
        response = custom_objects.get_namespaced_custom_object(
            "k8s.nginx.org", "v1alpha1", namespace, plural, name
        )
        pprint(response)
        return response

    except ApiException:
        logging.exception(f"Exception occurred while reading CRD")
        raise


def read_ts(custom_objects: CustomObjectsApi, namespace, name) -> object:
    """
    Read TransportService resource.
    """
    return read_custom_resource_v1alpha1(custom_objects, namespace, "transportservers", name)


def create_ts_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> dict:
    """
    Create a TransportServer Resource based on yaml file.

    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: a dictionary representing the resource
    """
    return create_resource_from_yaml(custom_objects, yaml_manifest, namespace, "transportservers")


def create_gc_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> dict:
    """
    Create a GlobalConfiguration Resource based on yaml file.

    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: a dictionary representing the resource
    """
    return create_resource_from_yaml(custom_objects, yaml_manifest, namespace, "globalconfigurations")


def create_resource_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace, plural) -> dict:
    """
    Create a Resource based on yaml file.

    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :param plural: the plural of the resource
    :return: a dictionary representing the resource
    """

    with open(yaml_manifest) as f:
        body = yaml.safe_load(f)
    try:
        print("Create a Custom Resource: " + body["kind"])
        group, version = body["apiVersion"].split("/")
        custom_objects.create_namespaced_custom_object(
             group, version, namespace, plural, body
        )
        print(f"Custom resource {body['kind']} created with name '{body['metadata']['name']}'")
        return body
    except ApiException as ex:
        logging.exception(
            f"Exception: {ex} occurred while creating {body['kind']}: {body['metadata']['name']}"
        )
        raise


def delete_ts(custom_objects: CustomObjectsApi, resource, namespace) -> None:
    """
    Delete a TransportServer Resource.

    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param resource: a dictionary representation of the resource yaml
    :param namespace:
    :return:
    """
    return delete_resource(custom_objects, resource, namespace, "transportservers")


def delete_gc(custom_objects: CustomObjectsApi, resource, namespace) -> None:
    """
    Delete a GlobalConfiguration Resource.

    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param resource: a dictionary representation of the resource yaml
    :param namespace:
    :return:
    """
    return delete_resource(custom_objects, resource, namespace, "globalconfigurations")


def delete_resource(custom_objects: CustomObjectsApi, resource, namespace, plural) -> None:
    """
    Delete a Resource.

    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param resource: a dictionary representation of the resource yaml
    :param namespace:
    :param plural: the plural of the resource
    :return:
    """

    name = resource['metadata']['name']
    kind = resource['kind']
    group, version = resource["apiVersion"].split("/")

    print(f"Delete a '{kind}' with name '{name}'")

    custom_objects.delete_namespaced_custom_object(
        group, version, namespace, plural, name
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        group,
        version,
        namespace,
        plural,
        name,
    )
    print(f"Resource '{kind}' was removed with name '{name}'")


def patch_ts(
        custom_objects: CustomObjectsApi, name, yaml_manifest, namespace
) -> None:
    """
    Patch a TransportServer based on yaml manifest
    """
    return patch_custom_resource_v1alpha1(custom_objects, name, yaml_manifest, namespace, "transportservers")


def patch_custom_resource_v1alpha1(custom_objects: CustomObjectsApi, name, yaml_manifest, namespace, plural) -> None:
    """
    Patch a custom resource based on yaml manifest
    """
    print(f"Update a Resource: {name}")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)

    try:
        custom_objects.patch_namespaced_custom_object(
            "k8s.nginx.org", "v1alpha1", namespace, plural, name, dep
        )
    except ApiException:
        logging.exception(f"Failed with exception while patching custom resource: {name}")
        raise


def generate_item_with_upstream_options(yaml_manifest, options) -> dict:
    """
    Generate a VS/VSR item with an upstream option.

    Update all the upstreams in VS/VSR
    :param yaml_manifest: an absolute path to a file
    :param options: dict
    :return: dict
    """
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)
    for upstream in dep["spec"]["upstreams"]:
        upstream.update(options)
    return dep

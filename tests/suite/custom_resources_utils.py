"""Describe methods to utilize the kubernetes-client."""
import pytest
import yaml
import logging

from pprint import pprint
from kubernetes.client import CustomObjectsApi, ApiextensionsV1beta1Api, CoreV1Api
from kubernetes import client
from kubernetes.client.rest import ApiException

from suite.resources_utils import ensure_item_removal, get_file_contents


def create_crd(api_extensions_v1_beta1: ApiextensionsV1beta1Api, body) -> None:
    """
    Create a CRD based on a dict

    :param api_extensions_v1_beta1: ApiextensionsV1beta1Api
    :param body: a dict
    """
    try:
        api_extensions_v1_beta1.create_custom_resource_definition(body)
    except ApiException as api_ex:
        raise api_ex
        # ApiException(f"An unexpected exception occurred: {api_ex}", reason=api_ex.reason)
    except Exception as ex:
        # https://github.com/kubernetes-client/python/issues/376
        if ex.args[0] == "Invalid value for `conditions`, must not be `None`":
            print("There was an insignificant exception during the CRD creation. Continue...")
        else:
            pytest.fail(f"An unexpected exception {ex} occurred. Exiting...")


def create_crd_from_yaml(
    api_extensions_v1_beta1: ApiextensionsV1beta1Api, name, yaml_manifest
) -> None:
    """
    Create a specific CRD based on yaml file.

    :param api_extensions_v1_beta1: ApiextensionsV1beta1Api
    :param name: CRD name
    :param yaml_manifest: an absolute path to file
    """
    print(f"Create a CRD with name: {name}")
    with open(yaml_manifest) as f:
        docs = yaml.safe_load_all(f)
        for dep in docs:
            if dep["metadata"]["name"] == name:
                create_crd(api_extensions_v1_beta1, dep)
                print("CRD was created")


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


def read_crd(custom_objects: CustomObjectsApi, namespace, plural, name) -> object:
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

    except ApiException as ex:
        logging.exception(f"Exception occured: {ex} while getting reading CRD")
        raise


def create_policy_from_yaml(
    custom_objects: CustomObjectsApi, yaml_manifest, namespace
) -> str:
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
            "k8s.nginx.org", "v1alpha1", namespace, "policies", dep
        )
        print(f"Policy created with name '{dep['metadata']['name']}'")
        return dep["metadata"]["name"]
    except ApiException as ex:
        logging.exception(
            f"Exception: {ex} occured while creating Policy: {dep['metadata']['name']}"
        )
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
    delete_options = client.V1DeleteOptions()

    custom_objects.delete_namespaced_custom_object(
        "k8s.nginx.org", "v1alpha1", namespace, "policies", name, delete_options
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "k8s.nginx.org",
        "v1alpha1",
        namespace,
        "policies",
        name,
    )
    print(f"Policy was removed with name '{name}'")


def read_policy(custom_objects: CustomObjectsApi, namespace, name) -> object:
    """
    Get policy information (kubectl describe output)

    :param custom_objects: CustomObjectsApi
    :param namespace: The policy's namespace	
    :param name: policy's name
    :return: object
    """
    print(f"Getting info for policy {name} in namespace {namespace}")
    try:
        response = custom_objects.get_namespaced_custom_object(
            "k8s.nginx.org", "v1alpha1", namespace, "policies", name
        )
        pprint(response)
        return response

    except ApiException as ex:
        logging.exception(f"Exception occured: {ex} while getting reading Policy")
        raise


def create_virtual_server_from_yaml(
    custom_objects: CustomObjectsApi, yaml_manifest, namespace
) -> str:
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
    try:
        custom_objects.create_namespaced_custom_object(
            "k8s.nginx.org", "v1", namespace, "virtualservers", dep
        )
        print(f"VirtualServer created with name '{dep['metadata']['name']}'")
        return dep["metadata"]["name"]
    except ApiException as ex:
        logging.exception(
            f"Exception: {ex} occured while creating VirtualServer: {dep['metadata']['name']}"
        )
        raise


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

    custom_objects.delete_namespaced_custom_object(
        "k8s.nginx.org", "v1", namespace, "virtualservers", name, delete_options
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "k8s.nginx.org",
        "v1",
        namespace,
        "virtualservers",
        name,
    )
    print(f"VirtualServer was removed with name '{name}'")


def patch_virtual_server_from_yaml(
    custom_objects: CustomObjectsApi, name, yaml_manifest, namespace
) -> None:
    """
    Patch a VS based on yaml manifest

    :param custom_objects: CustomObjectsApi
    :param name:
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return:
    """
    print(f"Update a VirtualServer: {name}")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)

    try:
        custom_objects.patch_namespaced_custom_object(
            "k8s.nginx.org", "v1", namespace, "virtualservers", name, dep
        )
        print(f"VirtualServer updated with name '{dep['metadata']['name']}'")
    except ApiException as ex:
        logging.exception(f"Failed with exception {ex} while patching VirtualServer: {name}")
        raise


def patch_virtual_server(custom_objects: CustomObjectsApi, name, namespace, body) -> str:
    """
    Update a VirtualServer based on a dict.

    :param custom_objects: CustomObjectsApi
    :param name:
    :param body: dict
    :param namespace:
    :return: str
    """
    print("Update a VirtualServer:")
    custom_objects.patch_namespaced_custom_object(
        "k8s.nginx.org", "v1", namespace, "virtualservers", name, body
    )
    print(f"VirtualServer updated with a name '{body['metadata']['name']}'")
    return body["metadata"]["name"]


def patch_v_s_route_from_yaml(
    custom_objects: CustomObjectsApi, name, yaml_manifest, namespace
) -> None:
    """
    Update a VirtualServerRoute based on yaml manifest

    :param custom_objects: CustomObjectsApi
    :param name:
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return:
    """
    print(f"Update a VirtualServerRoute: {name}")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)
    try:
        custom_objects.patch_namespaced_custom_object(
            "k8s.nginx.org", "v1", namespace, "virtualserverroutes", name, dep
        )
        print(f"VirtualServerRoute updated with name '{dep['metadata']['name']}'")
    except ApiException as ex:
        logging.exception(f"Failed with exception {ex} while patching VirtualServerRoute: {name}")
        raise


def get_vs_nginx_template_conf(
    v1: CoreV1Api, vs_namespace, vs_name, pod_name, pod_namespace
) -> str:
    """
    Get contents of /etc/nginx/conf.d/vs_{namespace}_{vs_name}.conf in the pod.

    :param v1: CoreV1Api
    :param vs_namespace:
    :param vs_name:
    :param pod_name:
    :param pod_namespace:
    :return: str
    """
    file_path = f"/etc/nginx/conf.d/vs_{vs_namespace}_{vs_name}.conf"
    return get_file_contents(v1, file_path, pod_name, pod_namespace)


def create_v_s_route_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> str:
    """
    Create a VirtualServerRoute based on a yaml file.

    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to a file
    :param namespace:
    :return: str
    """
    print("Create a VirtualServerRoute:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)

    custom_objects.create_namespaced_custom_object(
        "k8s.nginx.org", "v1", namespace, "virtualserverroutes", dep
    )
    print(f"VirtualServerRoute created with a name '{dep['metadata']['name']}'")
    return dep["metadata"]["name"]


def patch_v_s_route(custom_objects: CustomObjectsApi, name, namespace, body) -> str:
    """
    Update a VirtualServerRoute based on a dict.

    :param custom_objects: CustomObjectsApi
    :param name:
    :param body: dict
    :param namespace:
    :return: str
    """
    print("Update a VirtualServerRoute:")
    custom_objects.patch_namespaced_custom_object(
        "k8s.nginx.org", "v1", namespace, "virtualserverroutes", name, body
    )
    print(f"VirtualServerRoute updated with a name '{body['metadata']['name']}'")
    return body["metadata"]["name"]


def delete_v_s_route(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a VirtualServerRoute.

    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete a VirtualServerRoute: {name}")
    delete_options = client.V1DeleteOptions()
    custom_objects.delete_namespaced_custom_object(
        "k8s.nginx.org", "v1", namespace, "virtualserverroutes", name, delete_options
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "k8s.nginx.org",
        "v1",
        namespace,
        "virtualserverroutes",
        name,
    )
    print(f"VirtualServerRoute was removed with the name '{name}'")


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

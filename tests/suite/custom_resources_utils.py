"""Describe methods to utilize the kubernetes-client."""
import pytest
import time
import yaml
import logging

from pprint import pprint
from kubernetes.client import CustomObjectsApi, ApiextensionsV1Api, CoreV1Api
from kubernetes import client
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

def read_vs(custom_objects: CustomObjectsApi, namespace, name) -> object:
    """
    Read VirtualServer resource.
    """
    return read_custom_resource(custom_objects, namespace, "virtualservers", name)

def read_vsr(custom_objects: CustomObjectsApi, namespace, name) -> object:
    """
    Read VirtualServerRoute resource.
    """
    return read_custom_resource(custom_objects, namespace, "virtualserverroutes", name)

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

    return create_virtual_server(custom_objects, dep, namespace)


def create_virtual_server(
    custom_objects: CustomObjectsApi, vs, namespace
) -> str:
    """
    Create a VirtualServer.

    :param custom_objects: CustomObjectsApi
    :param vs: a VirtualServer
    :param namespace:
    :return: str
    """
    print("Create a VirtualServer:")
    try:
        custom_objects.create_namespaced_custom_object(
            "k8s.nginx.org", "v1", namespace, "virtualservers", vs
        )
        print(f"VirtualServer created with name '{vs['metadata']['name']}'")
        return vs["metadata"]["name"]
    except ApiException as ex:
        logging.exception(
            f"Exception: {ex} occurred while creating VirtualServer: {vs['metadata']['name']}"
        )
        raise


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


def create_ap_logconf_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> str:
    """
    Create a logconf for AppProtect based on yaml file.
    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: str
    """
    print("Create Ap logconf:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)
    custom_objects.create_namespaced_custom_object(
        "appprotect.f5.com", "v1beta1", namespace, "aplogconfs", dep
    )
    print(f"AP logconf created with name '{dep['metadata']['name']}'")
    return dep["metadata"]["name"]


def create_ap_policy_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> str:
    """
    Create a policy for AppProtect based on yaml file.
    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: str
    """
    print("Create AP Policy:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)
    custom_objects.create_namespaced_custom_object(
        "appprotect.f5.com", "v1beta1", namespace, "appolicies", dep
    )
    print(f"AP Policy created with name '{dep['metadata']['name']}'")
    return dep["metadata"]["name"]


def create_dos_logconf_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> str:
    """
    Create a logconf for Dos, based on yaml file.
    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: str
    """
    print("Create DOS logconf:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)
    custom_objects.create_namespaced_custom_object(
        "appprotectdos.f5.com", "v1beta1", namespace, "apdoslogconfs", dep
    )
    print(f"DOS logconf created with name '{dep['metadata']['name']}'")
    return dep["metadata"]["name"]


def create_dos_policy_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> str:
    """
    Create a policy for Dos based on yaml file.
    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: str
    """
    print("Create Dos Policy:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)
    custom_objects.create_namespaced_custom_object(
        "appprotectdos.f5.com", "v1beta1", namespace, "apdospolicies", dep
    )
    print(f"DOS Policy created with name '{dep['metadata']['name']}'")
    return dep["metadata"]["name"]


def create_dos_protected_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> str:
    """
    Create a protected resource for Dos based on yaml file.
    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: str
    """
    print("Create Dos Protected:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)
        dep['spec']['dosSecurityLog']['apDosLogConf'] = dep['spec']['dosSecurityLog']['apDosLogConf'].replace("<NAMESPACE>", namespace)
        dep['spec']['dosSecurityLog']['dosLogDest'] = dep['spec']['dosSecurityLog']['dosLogDest'].replace("<NAMESPACE>", namespace)
        dep['spec']['apDosPolicy'] = dep['spec']['apDosPolicy'].replace("<NAMESPACE>", namespace)
    custom_objects.create_namespaced_custom_object(
        "appprotectdos.f5.com", "v1beta1", namespace, "dosprotectedresources", dep
    )
    print(f"DOS Protected resource created with name '{namespace}/{dep['metadata']['name']}'")
    return dep["metadata"]["name"]


def create_ap_usersig_from_yaml(custom_objects: CustomObjectsApi, yaml_manifest, namespace) -> str:
    """
    Create a UserSig for AppProtect based on yaml file.
    :param custom_objects: CustomObjectsApi
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return: str
    """
    print("Create AP UserSig:")
    with open(yaml_manifest) as f:
        dep = yaml.safe_load(f)
    custom_objects.create_namespaced_custom_object(
        "appprotect.f5.com", "v1beta1", namespace, "apusersigs", dep
    )
    print(f"AP UserSig created with name '{dep['metadata']['name']}'")
    return dep["metadata"]["name"]


def delete_and_create_ap_policy_from_yaml(
    custom_objects: CustomObjectsApi, name, yaml_manifest, namespace
) -> None:
    """
    Patch a AP Policy based on yaml manifest
    :param custom_objects: CustomObjectsApi
    :param name:
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return:
    """
    print(f"Update an AP Policy: {name}")

    try:
        delete_ap_policy(custom_objects, name, namespace)
        create_ap_policy_from_yaml(custom_objects, yaml_manifest, namespace)
    except ApiException:
        logging.exception(f"Failed with exception while patching AP Policy: {name}")
        raise


def delete_ap_usersig(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a AppProtect usersig.
    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete AP UserSig: {name}")
    custom_objects.delete_namespaced_custom_object(
        "appprotect.f5.com", "v1beta1", namespace, "apusersigs", name
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "appprotect.f5.com",
        "v1beta1",
        namespace,
        "apusersigs",
        name,
    )
    print(f"AP UserSig was removed with name: {name}")


def delete_ap_logconf(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a AppProtect logconf.
    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete AP logconf: {name}")
    custom_objects.delete_namespaced_custom_object(
        "appprotect.f5.com", "v1beta1", namespace, "aplogconfs", name
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "appprotect.f5.com",
        "v1beta1",
        namespace,
        "aplogconfs",
        name,
    )
    print(f"AP logconf was removed with name: {name}")


def delete_ap_policy(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a AppProtect policy.
    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete a AP policy: {name}")
    custom_objects.delete_namespaced_custom_object(
        "appprotect.f5.com", "v1beta1", namespace, "appolicies", name
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "appprotect.f5.com",
        "v1beta1",
        namespace,
        "appolicies",
        name,
    )
    time.sleep(3)
    print(f"AP policy was removed with name: {name}")


def delete_dos_logconf(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a Dos logconf.
    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete DOS logconf: {name}")
    custom_objects.delete_namespaced_custom_object(
        "appprotectdos.f5.com", "v1beta1", namespace, "apdoslogconfs", name
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "appprotectdos.f5.com",
        "v1beta1",
        namespace,
        "apdoslogconfs",
        name,
    )
    print(f"DOS logconf was removed with name: {name}")


def delete_dos_protected(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a Dos protected.
    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete DOS protected: {name}")
    custom_objects.delete_namespaced_custom_object(
        "appprotectdos.f5.com", "v1beta1", namespace, "dosprotectedresources", name
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "appprotectdos.f5.com",
        "v1beta1",
        namespace,
        "dosprotectedresources",
        name,
    )
    print(f"DOS logconf was removed with name: {name}")


def delete_dos_policy(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a Dos policy.
    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete a DOS policy: {name}")
    custom_objects.delete_namespaced_custom_object(
        "appprotectdos.f5.com", "v1beta1", namespace, "apdospolicies", name
    )
    ensure_item_removal(
        custom_objects.get_namespaced_custom_object,
        "appprotectdos.f5.com",
        "v1beta1",
        namespace,
        "apdospolicies",
        name,
    )
    time.sleep(3)
    print(f"DOS policy was removed with name: {name}")


def delete_virtual_server(custom_objects: CustomObjectsApi, name, namespace) -> None:
    """
    Delete a VirtualServer.

    :param custom_objects: CustomObjectsApi
    :param namespace: namespace
    :param name:
    :return:
    """
    print(f"Delete a VirtualServer: {name}")

    custom_objects.delete_namespaced_custom_object(
        "k8s.nginx.org", "v1", namespace, "virtualservers", name
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
    except ApiException:
        logging.exception(f"Failed with exception while patching VirtualServer: {name}")
        raise


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


def delete_and_create_vs_from_yaml(
    custom_objects: CustomObjectsApi, name, yaml_manifest, namespace
) -> None:
    """
    Perform delete and create for vs with same name based on yaml

    :param custom_objects: CustomObjectsApi
    :param name:
    :param yaml_manifest: an absolute path to file
    :param namespace:
    :return:
    """
    try:
        delete_virtual_server(custom_objects, name, namespace)
        create_virtual_server_from_yaml(custom_objects, yaml_manifest, namespace)
    except ApiException:
        logging.exception(f"Failed with exception while patching VirtualServer: {name}")
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
    except ApiException:
        logging.exception(f"Failed with exception while patching VirtualServerRoute: {name}")
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

    return create_v_s_route(custom_objects, dep, namespace)

def create_v_s_route(custom_objects: CustomObjectsApi, vsr, namespace) -> str:
    """
    Create a VirtualServerRoute.

    :param custom_objects: CustomObjectsApi
    :param vsr: a VirtualServerRoute
    :param namespace:
    :return: str
    """
    print("Create a VirtualServerRoute:")
    custom_objects.create_namespaced_custom_object(
        "k8s.nginx.org", "v1", namespace, "virtualserverroutes", vsr
    )
    print(f"VirtualServerRoute created with a name '{vsr['metadata']['name']}'")
    return vsr["metadata"]["name"]


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
    custom_objects.delete_namespaced_custom_object(
        "k8s.nginx.org", "v1", namespace, "virtualserverroutes", name,
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

import pytest, yaml, tempfile, os
from settings import DEPLOYMENTS, TEST_DATA
from suite.utils.resources_utils import (
    create_items_from_yaml,
    delete_items_from_yaml,
    create_namespace,
    delete_namespace,
    create_ingress,
    patch_deployment,
)
from suite.utils.vs_vsr_resources_utils import create_virtual_server

deployment = f"{TEST_DATA}/test-resources/deployment.yaml"
service = f"{TEST_DATA}/test-resources/service.yaml"
ns = f"{TEST_DATA}/test-resources/ns.yaml"
vs = f"{TEST_DATA}/test-resources/virtual-server.yaml"
secret = f"{TEST_DATA}/test-resources/secret.yaml"

# Ingress specific resources
ingress_secret = f"{TEST_DATA}/test-resources/ingress-secret.yaml"
ingress = f"{TEST_DATA}/test-resources/ingress.yaml"
ingress_master = f"{TEST_DATA}/test-resources/master.yaml"
ingress_minion = f"{TEST_DATA}/test-resources/ingress-minion.yaml"

class TestStuff:
    @pytest.mark.create
    def test_stuff_create(self, request, kube_apis):
        # count = int(request.config.getoption("--num"))

        with open(ns) as f:
            namespace=f"test"
            doc = yaml.safe_load(f)
            doc["metadata"]["name"] = "test"
            with tempfile.NamedTemporaryFile(mode="w+", suffix=".yml", delete=False) as temp:
                temp.write(yaml.safe_dump(doc) + "---\n")
            namespace = create_namespace(kube_apis.v1, doc)
            os.remove(temp.name)

        with open(ingress_master) as f:
            doc = yaml.safe_load(f)
            create_ingress(kube_apis.networking_v1, "test", doc)

        with open(ingress_secret) as f:
            doc = yaml.safe_load(f)
            with tempfile.NamedTemporaryFile(mode="w+", suffix=".yml", delete=False) as temp:
                temp.write(yaml.safe_dump(doc) + "---\n")
            create_items_from_yaml(kube_apis, temp.name, "test")
            os.remove(temp.name)

        for i in range(1, 65+1):
            namespace=f"test"
            with open(ns) as f:
                doc = yaml.safe_load(f)
                doc["metadata"]["name"] = f"ns-{i}"
                with tempfile.NamedTemporaryFile(mode="w+", suffix=".yml", delete=False) as temp:
                    temp.write(yaml.safe_dump(doc) + "---\n")
                namespace = create_namespace(kube_apis.v1, doc)
                os.remove(temp.name)

            with open(deployment) as f:
                doc = yaml.safe_load(f)
                doc["metadata"]["name"] = f"backend-{i}"
                doc["spec"]["selector"]["matchLabels"]["app"] = f"backend-{i}"
                doc["spec"]["template"]["metadata"]["labels"]["app"] = f"backend-{i}"
                doc["metadata"]["name"] = f"backend-{i}"
                with tempfile.NamedTemporaryFile(mode="w+", suffix=".yml", delete=False) as temp:
                    temp.write(yaml.safe_dump(doc) + "---\n")
                create_items_from_yaml(kube_apis, temp.name, namespace)
                os.remove(temp.name)

                # patch_deployment(kube_apis.apps_v1_api, namespace, doc) #update number of pods

            with open(service) as f:
                doc = yaml.safe_load(f)
                doc["metadata"]["name"] = f"backend-svc-{i}"
                doc["spec"]["selector"]["app"] = f"backend-{i}"
                with tempfile.NamedTemporaryFile(mode="w+", suffix=".yml", delete=False) as temp:
                    temp.write(yaml.safe_dump(doc) + "---\n")
                create_items_from_yaml(kube_apis, temp.name, namespace)
                os.remove(temp.name)

            with open(secret) as f:
                doc = yaml.safe_load(f)
                doc["metadata"]["name"] = f"secret-{i}"
                with tempfile.NamedTemporaryFile(mode="w+", suffix=".yml", delete=False) as temp:
                    temp.write(yaml.safe_dump(doc) + "---\n")
                create_items_from_yaml(kube_apis, temp.name, namespace)
                os.remove(temp.name)

            # Ingress Minions
            with open(ingress_minion) as f:
                doc = yaml.safe_load(f)
                doc["metadata"]["name"] = f"cafe-ingress-coffee-minion-{i}"
                doc["spec"]["rules"][0]["http"]["paths"][0]["path"] = f"/backend-{i}"
                doc["spec"]["rules"][0]["http"]["paths"][0]["backend"]["service"]["name"] = f"backend-svc-{i}"
                create_ingress(kube_apis.networking_v1, namespace, doc)

            # VirtualServer
            with open(vs) as f:
                doc = yaml.safe_load(f)
                doc["metadata"]["name"] = f"vs-{i}"
                doc["spec"]["host"] = f"vs-{i}.example.com"
                doc["spec"]["tls"]["secret"] = f"secret-{i}"
                doc["spec"]["upstreams"][0]["name"] = f"backend-{i}"
                doc["spec"]["upstreams"][0]["service"] = f"backend-svc-{i}"
                doc["spec"]["routes"][0]["action"]["pass"] = f"backend-{i}"
                create_virtual_server(kube_apis.custom_objects, doc, namespace)

            # Ingress
            with open(ingress) as f:
                doc = yaml.safe_load(f)
                doc["metadata"]["name"] = f"ingress-{i}"
                doc["spec"]["tls"][0]["hosts"][0] = f"ingress-{i}.example.com"
                doc["spec"]["tls"][0]["secretName"] = f"secret-{i}"
                doc["spec"]["rules"][0]["host"] = f"ingress-{i}.example.com"
                doc["spec"]["rules"][0]["http"]["paths"][0]["path"] = f"/backend-{i}"
                doc["spec"]["rules"][0]["http"]["paths"][0]["backend"]["service"]["name"] = f"backend-svc-{i}"
                create_ingress(kube_apis.networking_v1, namespace, doc)


    @pytest.mark.delete
    def test_stuff_delete(self, request, kube_apis):
        # count = int(request.config.getoption("--num"))
        delete_namespace(kube_apis.v1, "test")
        # delete namespaces
        for i in range(1, 100+1):
            delete_namespace(kube_apis.v1, f"ns-{i}")
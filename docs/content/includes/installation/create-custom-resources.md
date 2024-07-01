---
docs: DOCS-1463
---

To make sure your NGINX Ingress Controller pods reach the `Ready` state, you'll need to create custom resource definitions (CRDs) for various components.

Alternatively, you can disable this requirement by setting the `-enable-custom-resources` command-line argument to `false`.

There are two ways you can install the custom resource definitions:

1. Using a URL to apply a single CRD yaml file, which we recommend.
1. Applying your local copy of the CRD yaml files, which requires you to clone the repository.

{{<tabs name="install-crds">}}

{{%tab name="Install CRDs from single YAML"%}}

This single YAML file creates CRDs for the following resources:

- [VirtualServer and VirtualServerRoute]({{< relref "configuration/virtualserver-and-virtualserverroute-resources.md" >}})
- [TransportServer]({{< relref "configuration/transportserver-resource.md" >}})
- [Policy]({{< relref "configuration/policy-resource.md" >}})
- [GlobalConfiguration]({{< relref "configuration/global-configuration/globalconfiguration-resource.md" >}})

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.6.1/deploy/crds.yaml
```

{{%/tab%}}

{{%tab name="Install CRDs after cloning the repo"%}}

{{< note >}} If you are installing the CRDs this way, ensure you have first cloned the repository. {{< /note >}}

These YAML files create CRDs for the following resources:

- [VirtualServer and VirtualServerRoute]({{< relref "configuration/virtualserver-and-virtualserverroute-resources.md" >}})
- [TransportServer]({{< relref "configuration/transportserver-resource.md" >}})
- [Policy]({{< relref "configuration/policy-resource.md" >}})
- [GlobalConfiguration]({{< relref "configuration/global-configuration/globalconfiguration-resource.md" >}})

```shell
kubectl apply -f config/crd/bases/k8s.nginx.org_virtualservers.yaml
kubectl apply -f config/crd/bases/k8s.nginx.org_virtualserverroutes.yaml
kubectl apply -f config/crd/bases/k8s.nginx.org_transportservers.yaml
kubectl apply -f config/crd/bases/k8s.nginx.org_policies.yaml
kubectl apply -f config/crd/bases/k8s.nginx.org_globalconfigurations.yaml
```

{{%/tab%}}

{{</tabs>}}

---
title: Installation with Manifests
description: "This document describes how to install the NGINX Ingress Controller in your Kubernetes cluster using Kubernetes manifests."
weight: 1900
doctypes: [""]
aliases:
    - /installation/
toc: true
docs: "DOCS-603"
---

## Prerequisites

1. Make sure you have access to an NGINX Ingress Controller image:
    - For NGINX Ingress Controller, use the images from [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress),
    [GitHub Container Registry](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress),
    [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress) or
    [Quay.io](https://quay.io/repository/nginx/nginx-ingress).
    - For NGINX Plus Ingress Controller, see
      [here](/nginx-ingress-controller/installation/pulling-ingress-controller-image) for details on pulling the image
      from the F5 Docker registry.
    - To pull from the F5 Container registry in your Kubernetes cluster, configure a docker registry secret using your
      JWT token from the MyF5 portal by following the instructions from
      [here](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret).
    - You can also build your own image and push it to your private Docker registry by following the instructions from [here](/nginx-ingress-controller/installation/building-ingress-controller-image).
2. All the commands in this document directly apply the YAML files from the repository. If you prefer, you can download
   the files and modify them according to your requirements.

{{<note>}} To perform some of the following steps you must be a cluster admin. Follow the documentation of your
Kubernetes platform to configure the admin access. For Google Kubernetes Engine, see their [Role-Based Access
Control](https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control) documentation.{{</note>}}

---

## 1. Create Custom Resources

{{<note>}}
By default, it is required to create custom resource definitions for VirtualServer, VirtualServerRoute, TransportServer
and Policy. Otherwise, NGINX Ingress Controller pods will not become `Ready`. If you'd like to disable that requirement,
configure
[`-enable-custom-resources`](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments#cmdoption-global-configuration)
command-line argument to `false` and skip this section.
{{</note>}}

1. Create custom resource definitions for [VirtualServer and VirtualServerRoute](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources),
   [TransportServer](/nginx-ingress-controller/configuration/transportserver-resource),
   [Policy](/nginx-ingress-controller/configuration/policy-resource) and
   [GlobalConfiguration](/nginx-ingress-controller/configuration/global-configuration/globalconfiguration-resource)
   resources:

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/crds.yaml
    ```

2. If you would like to use the NGINX App Protect WAF module, you will need to create custom resource definitions for
   `APPolicy`, `APLogConf` and `APUserSig`:

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/crds-nap-waf.yaml
    ```

3. If you would like to use the NGINX App Protect DoS module, you will need to create custom resource definitions for
   `APDosPolicy`, `APDosLogConf` and `DosProtectedResource`:

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/crds-nap-dos.yaml
    ```

---

## 2. Deploying NGINX Ingress Controller

The NGINX Ingress Controller repository contains deployment files with all the resources needed in a single file (except
for the CRDs above). You can run the commands as is or or customize them according to your requirements, for example to
update the [command line arguments](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments)
documentation for more details.

---

### 2.1 Running NGINX Ingress Controller

There are multiple sample deployment files available in the repository. Choose the one that best suits your needs.

{{<tabs name="install-manifests">}}

{{%tab name="Deployment"%}}

This is a default deployment file. It deploys the NGINX Ingress Controller as a Deployment.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/default/deploy.yaml
```

{{%/tab%}}

{{%tab name="DaemonSet"%}}

This is a default daemonset file. It deploys the NGINX Ingress Controller as a DaemonSet.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/default/daemonset.yaml
```

{{%/tab%}}

{{%tab name="Azure"%}}

Deploys NGINX Ingress Controller using a nodeSelector to deploy the controller on Azure nodes.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/azure/deploy.yaml
```

{{%/tab%}}

{{%tab name="AWS NLB"%}}

 Deploys NGINX Ingress Controller using a Service type of `LoadBalancer` to allocate an AWS
  Network Load Balancer (NLB).

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/aws-nlb/deploy.yaml
```

{{%/tab%}}

{{%tab name="OIDC"%}}

Deploys NGINX Ingress Controller with OpenID Connect (OIDC) authentication enabled.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/oidc/deploy.yaml
```

{{%/tab%}}

{{%tab name="NGINX Plus"%}}

Deploys NGINX Ingress Controller with the NGINX Plus. The image is pulled from the
NGINX Plus Docker registry, and the `imagePullSecretName` is the name of the secret to use to pull the image.
The secret must be created in the same namespace as the NGINX Ingress Controller.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/nginx-plus/deploy.yaml
```

{{%/tab%}}

{{%tab name="NGINX App Protect WAF"%}}

Deploys NGINX Ingress Controller with the NGINX App Protect WAF module enabled. The image is pulled from the NGINX Plus
Docker registry, and the `imagePullSecretName` is the name of the secret to use to pull the image. The secret must be
created in the same namespace as the NGINX Ingress Controller.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/appprotect-waf/deploy.yaml
```

{{%/tab%}}

{{%tab name="NGINX App Protect DoS"%}}

Deploys NGINX Ingress Controller with the NGINX App Protect DoS module enabled. The image is pulled from the NGINX Plus
Docker registry, and the `imagePullSecretName` is the name of the secret to use to pull the image. The secret must be
created in the same namespace as the NGINX Ingress Controller.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/appprotect-dos/deploy.yaml
```

{{%/tab%}}

{{%tab name="Read-only filesystem"%}}

Deploys NGINX Ingress Controller with a read-only filesystem.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/read-only-fs/deploy.yaml
```

{{%/tab%}}

{{%tab name="NodePort"%}}

Deploys NGINX Ingress Controller using a Service type of `NodePort`.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/nodeport/deploy.yaml
```

{{%/tab%}}

{{%tab name="Edge"%}}

Deploys NGINX Ingress Controller using the `edge` tag from Docker Hub. See the
[README](https://github.com/nginxinc/kubernetes-ingress/blob/main/README.md#nginx-ingress-controller-releases)
for more information on the different tags.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/edge/deploy.yaml
```

{{%/tab%}}

{{%tab name="Service Insight"%}}

Deploys NGINX Ingress Controller with Service Insight enabled.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/service-insight/deploy.yaml
```

{{%/tab%}}

{{%tab name="External DNS"%}}

Deploys NGINX Ingress Controller with External DNS enabled.

```shell
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/external-dns/deploy.yaml
```

{{%/tab%}}

{{</tabs>}}

---

### 4.2 Check that NGINX Ingress Controller is Running

Run the following command to make sure that the NGINX Ingress Controller pods are running:

```shell
kubectl get pods --namespace=nginx-ingress
```

## 5. Getting Access to NGINX Ingress Controller

If you deployed a DaemonSet, ports 80 and 443 of NGINX Ingress Controller container are mapped to the same ports of the
node where the container is running. To access NGINX Ingress Controller, use those ports and an IP address of any node
of the cluster where the Ingress Controller is running.

If you deployed a Deployment, there are two options for accessing NGINX Ingress Controller pods:

- If the LoadBalancer type is `NodePort`, Kubernetes will randomly allocate two ports on every node of the cluster.
To access the Ingress Controller, use an IP address of any node of the cluster along with the two allocated ports.

{{<note>}} Read more about the type NodePort in the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport). {{</note>}}

- If the LoadBalancer type is `LoadBalancer`:
  - For GCP or Azure, Kubernetes will allocate a cloud load balancer for load balancing the Ingress Controller pods.
    Use the public IP of the load balancer to access NGINX Ingress Controller.
  - For AWS, Kubernetes will allocate a Network Load Balancer (NLB) in TCP mode with the PROXY protocol enabled to pass
    the client's information (the IP address and the port).

    {{<note>}} For AWS, additional options regarding an allocated load balancer are available, such as its type and SSL
    termination. Read the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#type-loadbalancer) to learn more.
    {{</note>}}

    Kubernetes will allocate and configure a cloud load balancer for load balancing the Ingress Controller pods.

  Use the public IP of the load balancer to access NGINX Ingress Controller. To get the public IP:
  - For GCP or Azure, run:

    ```shell
    kubectl get svc nginx-ingress --namespace=nginx-ingress
    ```

  - In case of AWS ELB, the public IP is not reported by `kubectl`, because the ELB IP addresses are not static. In
    general,  you should rely on the ELB DNS name instead of the ELB IP addresses. However, for testing purposes, you
    can get the DNS name of the ELB using `kubectl describe` and then run `nslookup` to find the associated IP address:

    ```shell
    kubectl describe svc nginx-ingress --namespace=nginx-ingress
    ```

    You can resolve the DNS name into an IP address using `nslookup`:

    ```shell
    nslookup <dns-name>
    ```

    The public IP can be reported in the status of an ingress resource. See the [Reporting Resources Status doc](/nginx-ingress-controller/configuration/global-configuration/reporting-resources-status) for more details.

{{<note>}} Learn more about type LoadBalancer in the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#type-loadbalancer). {{</note>}}

## Uninstall NGINX Ingress Controller

1. Delete the `nginx-ingress` namespace to uninstall NGINX Ingress Controller along with all the auxiliary resources
   that were created:

    ```shell
    kubectl delete namespace nginx-ingress
    ```

1. Delete the ClusterRole and ClusterRoleBinding:

    ```shell
    kubectl delete clusterrole nginx-ingress
    kubectl delete clusterrolebinding nginx-ingress
    ```

1. Delete the Custom Resource Definitions:

    {{<note>}} This step will also remove all associated Custom Resources. {{</note>}}

    ```shell
    kubectl delete -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.3.2/deploy/crds.yaml
    ```

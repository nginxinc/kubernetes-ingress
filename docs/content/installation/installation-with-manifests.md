---
title: Installation with Manifests
description: "This document describes how to install the NGINX Ingress Controller in your Kubernetes cluster using Kubernetes manifests."
weight: 1800
doctypes: [""]
aliases:
    - /installation/
toc: true
docs: "DOCS-603"
---

## Prerequisites

1. Make sure you have access to an NGINX Ingress Controller image:
    * For NGINX Ingress Controller, use the image `nginx/nginx-ingress` from [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress).
    * For NGINX Plus Ingress Controller, see [here](/nginx-ingress-controller/installation/pulling-ingress-controller-image) for details on pulling the image from the F5 Docker registry.
    * To pull from the F5 Container registry in your Kubernetes cluster, configure a docker registry secret using your JWT token from the MyF5 portal by following the instructions from [here](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret).
    * You can also build your own image and push it to your private Docker registry by following the instructions from [here](/nginx-ingress-controller/installation/building-ingress-controller-image).
2. Clone the NGINX Ingress Controller repository and change into the deployments folder:

    ```shell
    git clone https://github.com/nginxinc/kubernetes-ingress.git --branch v3.2.0
    cd kubernetes-ingress/deployments
    ```

    {{<note>}}The above command will clone the branch of the latest NGINX Ingress Controller release, and all documentation assumes you are using it.{{</note>}}

---

## 1. Configure RBAC

1. Create a namespace and a service account for NGINX Ingress Controller:

    ```shell
    kubectl apply -f common/ns-and-sa.yaml
    ```

2. Create a cluster role and cluster role binding for the service account:

    ```shell
    kubectl apply -f rbac/rbac.yaml
    ```

3. (App Protect only) Create the App Protect role and role binding:

    ```shell
    kubectl apply -f rbac/ap-rbac.yaml
    ```

4. (App Protect DoS only) Create the App Protect DoS role and role binding:

    ```shell
    kubectl apply -f rbac/apdos-rbac.yaml
    ```

{{<note>}} To perform this step you must be a cluster admin. Follow the documentation of your Kubernetes platform to configure the admin access. For Google Kubernetes Engine, see their [Role-Based Access Control](https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control) documentation.{{</note>}}

---

## 2. Create Common Resources

In this section, we create resources common for most of NGINX Ingress Controller installations:
{{<note>}}
Installing the `default-server-secret.yaml` is optional and is required only if you are using the  [default server TLS secret](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments#cmdoption-default-server-tls-secret) command line argument. It is recommended that users provide their own certificate.
Otherwise, step 1 can be ignored.
{{</note>}}

1. Create a secret with a TLS certificate and a key for the default server in NGINX (below assumes you are in the `kubernetes-ingress/deployment` directory):

    ```console
    kubectl apply -f ../examples/shared-examples/default-server-secret/default-server-secret.yaml
    ```

    {{<note>}} The default server returns the Not Found page with the 404 status code for all requests for domains for which there are no Ingress rules defined. For testing purposes we include a self-signed certificate and key that we generated. However, we recommend that you use your own certificate and key. {{</note>}}

1. Create a config map for customizing NGINX configuration:

    ```console
    kubectl apply -f common/nginx-config.yaml
    ```

1. Create an IngressClass resource:

    ```console
    kubectl apply -f common/ingress-class.yaml
    ```

    If you would like to set this NGINX Ingress Controller instance as the default, uncomment the annotation `ingressclass.kubernetes.io/is-default-class`. With this annotation set to true all the new Ingresses without an ingressClassName field specified will be assigned this IngressClass.

    {{<note>}} NGINX Ingress Controller will fail to start without an IngressClass resource. {{</note>}}

---

## 3. Create Custom Resources

{{<note>}}
By default, it is required to create custom resource definitions for VirtualServer, VirtualServerRoute, TransportServer and Policy. Otherwise, NGINX Ingress Controller pods will not become `Ready`. If you'd like to disable that requirement, configure [`-enable-custom-resources`](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments#cmdoption-global-configuration) command-line argument to `false` and skip this section.
{{</note>}}

1. Create custom resource definitions for [VirtualServer and VirtualServerRoute](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources), [TransportServer](/nginx-ingress-controller/configuration/transportserver-resource) and [Policy](/nginx-ingress-controller/configuration/policy-resource) resources:

    ```console
    kubectl apply -f common/crds/k8s.nginx.org_virtualservers.yaml
    kubectl apply -f common/crds/k8s.nginx.org_virtualserverroutes.yaml
    kubectl apply -f common/crds/k8s.nginx.org_transportservers.yaml
    kubectl apply -f common/crds/k8s.nginx.org_policies.yaml
    ```

2. If you would like to use the TCP and UDP load balancing features, create a custom resource definition for the [GlobalConfiguration](/nginx-ingress-controller/configuration/global-configuration/globalconfiguration-resource) resource:

    ```console
    kubectl apply -f common/crds/k8s.nginx.org_globalconfigurations.yaml
    ```

3. If you would like to use the App Protect WAF module, you will need to create custom resource definitions for `APPolicy`, `APLogConf` and `APUserSig`:

    ```console
    kubectl apply -f common/crds/appprotect.f5.com_aplogconfs.yaml
    kubectl apply -f common/crds/appprotect.f5.com_appolicies.yaml
    kubectl apply -f common/crds/appprotect.f5.com_apusersigs.yaml
    ```

4. If you would like to use the App Protect DoS module, you will need to create custom resource definitions for `APDosPolicy`, `APDosLogConf` and `DosProtectedResource`:

   ```console
   kubectl apply -f common/crds/appprotectdos.f5.com_apdoslogconfs.yaml
   kubectl apply -f common/crds/appprotectdos.f5.com_apdospolicy.yaml
   kubectl apply -f common/crds/appprotectdos.f5.com_dosprotectedresources.yaml
   ```

---

## 4. Deploying NGINX Ingress Controller

There are two options for deploying NGINX Ingress Controller:

* *Deployment*. Use a Deployment if you plan to dynamically change the number of Ingress Controller replicas.
* *DaemonSet*. Use a DaemonSet for deploying the Ingress Controller on every node or a subset of nodes.

Additionally, if you would like to use the NGINX App Protect DoS module, you'll need to deploy the Arbitrator.

{{<note>}} Before creating a Deployment or Daemonset resource, make sure to update the [command-line arguments](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments) of NGINX Ingress Controller container in the corresponding manifest file according to your requirements. {{</note>}}

---

### Deploying Arbitrator for NGINX App Protect DoS

There are two steps for deploying NGINX Ingress Controller with the NGINX App Protect DoS module:

1. Build your own image and push it to your private Docker registry by following the instructions from [here](/nginx-ingress-controller/app-protect-dos/installation#Build-the-app-protect-dos-arb-Docker-Image).

1. Run the Arbitrator by using a Deployment and Service

   ```console
   kubectl apply -f deployment/appprotect-dos-arb.yaml
   kubectl apply -f service/appprotect-dos-arb-svc.yaml
   ```

---

### 4.1 Running NGINX Ingress Controller

#### Using a Deployment
When you run NGINX Ingress Controller by using a Deployment, by default, Kubernetes will create one NGINX Ingress Controller pod.

For NGINX, run:

```console
kubectl apply -f deployment/nginx-ingress.yaml
```

For NGINX Plus, run:

```console
kubectl apply -f deployment/nginx-plus-ingress.yaml
```

{{<note>}} Update the `nginx-plus-ingress.yaml` with the chosen image from the F5 Container registry; or the container image that you have built. {{</note>}}

---

#### Using a DaemonSet
When you run the Ingress Controller by using a DaemonSet, Kubernetes will create an Ingress Controller pod on every node of the cluster.

{{<note>}} Read the Kubernetes [DaemonSet docs](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) to learn how to run NGINX Ingress Controller on a subset of nodes instead of on every node of the cluster.{{</note>}}

For NGINX, run:

```console
kubectl apply -f daemon-set/nginx-ingress.yaml
```

For NGINX Plus, run:

```console
kubectl apply -f daemon-set/nginx-plus-ingress.yaml
```

{{<note>}}Update `nginx-plus-ingress.yaml` with the chosen image from the F5 Container registry; or the container image that you have built.{{</note>}}

---

### 4.2 Check that NGINX Ingress Controller is Running

Run the following command to make sure that the NGINX Ingress Controller pods are running:

```console
kubectl get pods --namespace=nginx-ingress
```

## 5. Getting Access to NGINX Ingress Controller

**If you created a daemonset**, ports 80 and 443 of NGINX Ingress Controller container are mapped to the same ports of the node where the container is running. To access NGINX Ingress Controller, use those ports and an IP address of any node of the cluster where the Ingress Controller is running.

**If you created a deployment**, there are two options for accessing NGINX Ingress Controller pods:

### 5.1 Create a Service for the NGINX Ingress Controller Pods

#### Using a NodePort Service

Create a service with the type *NodePort*:

```console
kubectl create -f service/nodeport.yaml
```

Kubernetes will randomly allocate two ports on every node of the cluster. To access the Ingress Controller, use an IP address of any node of the cluster along with the two allocated ports.

{{<note>}} Read more about the type NodePort in the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport). {{</note>}}

#### Using a LoadBalancer Service

1. Create a service using a manifest for your cloud provider:
    * For GCP or Azure, run:

        ```shell
        kubectl apply -f service/loadbalancer.yaml
        ```

    * For AWS, run:

        ```shell
        kubectl apply -f service/loadbalancer-aws-elb.yaml
        ```

        Kubernetes will allocate a Classic Load Balancer (ELB) in TCP mode with the PROXY protocol enabled to pass the client's information (the IP address and the port). NGINX must be configured to use the PROXY protocol:
        * Add the following keys to the config map file `nginx-config.yaml` from the Step 2:

            ```yaml
            proxy-protocol: "True"
            real-ip-header: "proxy_protocol"
            set-real-ip-from: "0.0.0.0/0"
            ```

        * Update the config map:

            ```shell
            kubectl apply -f common/nginx-config.yaml
            ```

        {{<note>}} For AWS, additional options regarding an allocated load balancer are available, such as its type and SSL termination. Read the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#type-loadbalancer) to learn more. {{</note>}}

    Kubernetes will allocate and configure a cloud load balancer for load balancing the Ingress Controller pods.
2. Use the public IP of the load balancer to access NGINX Ingress Controller. To get the public IP:
    * For GCP or Azure, run:

        ```shell
        kubectl get svc nginx-ingress --namespace=nginx-ingress
        ```

    * In case of AWS ELB, the public IP is not reported by `kubectl`, because the ELB IP addresses are not static. In general, you should rely on the ELB DNS name instead of the ELB IP addresses. However, for testing purposes, you can get the DNS name of the ELB using `kubectl describe` and then run `nslookup` to find the associated IP address:

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

1. Delete the `nginx-ingress` namespace to uninstall NGINX Ingress Controller along with all the auxiliary resources that were created:

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
    kubectl delete -f common/crds/
    ```

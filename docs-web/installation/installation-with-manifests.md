# Installation with Manifests

This document describes how to install the NGINX Ingress Controller in your Kubernetes cluster using Kubernetes manifests.

## Prerequisites

1. Make sure you have access to the Ingress controller image:
    * For NGINX Ingress controller, use the image `nginx/nginx-ingress` from [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress).
    * For NGINX Plus Ingress controller, build your own image and push it to your private Docker registry by following the instructions from [here](/nginx-ingress-controller/installation/building-ingress-controller-image).
1. Clone the Ingress controller repo and change into the deployments folder:
    ```
    $ git clone https://github.com/nginxinc/kubernetes-ingress/
    $ cd kubernetes-ingress/deployments
    ```

## 1. Configure RBAC

1. Create a namespace and a service account for the Ingress controller:
    ```
    $ kubectl apply -f common/ns-and-sa.yaml
    ```
2. Create a cluster role and cluster role binding for the service account:
    ```
    $ kubectl apply -f rbac/rbac.yaml
    ```
**Note**: To perform this step you must be a cluster admin. Follow the documentation of your Kubernetes platform to configure the admin access. For GKE, see the [Role-Based Access Control](https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control) doc.

## 2. Create Common Resources

In this section, we create resources common for most of the Ingress Controller installations:
1. Create a secret with a TLS certificate and a key for the default server in NGINX:
    ```
    $ kubectl apply -f common/default-server-secret.yaml
    ```

    **Note**: The default server returns the Not Found page with the 404 status code for all requests for domains for which there are no Ingress rules defined. For testing purposes we include a self-signed certificate and key that we generated. However, we recommend that you use your own certificate and key.

1. Create a config map for customizing NGINX configuration:
    ```
    $ kubectl apply -f common/nginx-config.yaml
    ```

1. Create custom resource definitions for [VirtualServer and VirtualServerRoute](/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources), [TransportServer](/nginx-ingress-controller/configuration/transportserver-resource) and [Policy](/nginx-ingress-controller/configuration/policy-resource) resources:
    ```
    $ kubectl apply -f common/vs-definition.yaml
    $ kubectl apply -f common/vsr-definition.yaml
    $ kubectl apply -f common/ts-definition.yaml
    $ kubectl apply -f common/policy-definition.yaml
    ```

If you would like to use the TCP and UDP load balancing features of the Ingress Controller, create the following additional resources: 
1. Create a custom resource definition for [GlobalConfiguration](/nginx-ingress-controller/configuration/global-configuration/globalconfiguration-resource) resource:
    ```
    $ kubectl apply -f common/gc-definition.yaml
    ```
1. Create a GlobalConfiguration resource:
    ```
    $ kubectl apply -f common/global-configuration.yaml
    ```
    **Note**: Make sure to references this resource in the [`-global-configuration`](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments#cmdoption-global-configuration) command-line argument.

> **Feature Status**: The TransportServer, GlobalConfiguration and Policy resources are available as a preview feature: it is suitable for experimenting and testing; however, it must be used with caution in production environments. Additionally, while the feature is in preview, we might introduce some backward-incompatible changes to the resources specification in the next releases.

## 3. Deploy the Ingress Controller

We include two options for deploying the Ingress controller:
* *Deployment*. Use a Deployment if you plan to dynamically change the number of Ingress controller replicas.
* *DaemonSet*. Use a DaemonSet for deploying the Ingress controller on every node or a subset of nodes.

> Before creating a Deployment or Daemonset resource, make sure to update the  [command-line arguments](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments) of the Ingress Controller container in the corresponding manifest file according to your requirements.

### 3.1 Run the Ingress Controller
* *Use a Deployment*.
    When you run the Ingress Controller by using a Deployment, by default, Kubernetes will create one Ingress controller pod.

    For NGINX, run:
    ```
    $ kubectl apply -f deployment/nginx-ingress.yaml
    ```

    For NGINX Plus, run:
    ```
    $ kubectl apply -f deployment/nginx-plus-ingress.yaml
    ```

    **Note**: Update the `nginx-plus-ingress.yaml` with the container image that you have built.

* *Use a DaemonSet*:
    When you run the Ingress Controller by using a DaemonSet, Kubernetes will create an Ingress controller pod on every node of the cluster.

    **See also:** See the Kubernetes [DaemonSet docs](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) to learn how to run the Ingress controller on a subset of nodes instead of on every node of the cluster.

    For NGINX, run:
    ```
    $ kubectl apply -f daemon-set/nginx-ingress.yaml
    ```

    For NGINX Plus, run:
    ```
    $ kubectl apply -f daemon-set/nginx-plus-ingress.yaml
    ```

    **Note**: Update the `nginx-plus-ingress.yaml` with the container image that you have built.

### 3.2 Check that the Ingress Controller is Running

Run the following command to make sure that the Ingress controller pods are running:
```
$ kubectl get pods --namespace=nginx-ingress
```

## 4. Get Access to the Ingress Controller

**If you created a daemonset**, ports 80 and 443 of the Ingress controller container are mapped to the same ports of the node where the container is running. To access the Ingress controller, use those ports and an IP address of any node of the cluster where the Ingress controller is running.

**If you created a deployment**, below are two options for accessing the Ingress controller pods.

### 4.1 Create a Service for the Ingress Controller Pods

* *Use a NodePort service*.

    Create a service with the type *NodePort*:
    ```
    $ kubectl create -f service/nodeport.yaml
    ```

    Kubernetes will randomly allocate two ports on every node of the cluster. To access the Ingress controller, use an IP address of any node of the cluster along with the two allocated ports.

    > Read more about the type NodePort in the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport).


* *Use a LoadBalancer service*:

    1. Create a service using a manifest for your cloud provider:
        * For GCP or Azure, run:
            ```
            $ kubectl apply -f service/loadbalancer.yaml
            ```
        * For AWS, run:
            ```
            $ kubectl apply -f service/loadbalancer-aws-elb.yaml
            ```
            Kubernetes will allocate a Classic Load Balancer (ELB) in TCP mode with the PROXY protocol enabled to pass the client's information (the IP address and the port). NGINX must be configured to use the PROXY protocol:
            * Add the following keys to the config map file `nginx-config.yaml` from the Step 2:
                ```
                proxy-protocol: "True"
                real-ip-header: "proxy_protocol"
                set-real-ip-from: "0.0.0.0/0"
                ```
            * Update the config map:
                ```
                kubectl apply -f common/nginx-config.yaml
                ```
            **Note**: For AWS, additional options regarding an allocated load balancer are available, such as the type of a load balancer and SSL termination. Read the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#type-loadbalancer) to learn more.

        Kubernetes will allocate and configure a cloud load balancer for load balancing the Ingress controller pods.
    2. Use the public IP of the load balancer to access the Ingress controller. To get the public IP:
        * For GCP or Azure, run:
            ```
            $ kubectl get svc nginx-ingress --namespace=nginx-ingress
            ```
        * In case of AWS ELB, the public IP is not reported by `kubectl`, because the ELB IP addresses are not static. In general, you should rely on the ELB DNS name instead of the ELB IP addresses. However, for testing purposes, you can get the DNS name of the ELB using `kubectl describe` and then run `nslookup` to find the associated IP address:
            ```
            $ kubectl describe svc nginx-ingress --namespace=nginx-ingress
            ```
            You can resolve the DNS name into an IP address using `nslookup`:
            ```
            $ nslookup <dns-name>
            ```

        The public IP can be reported in the status of an ingress resource. See the [Reporting Resources Status doc](/nginx-ingress-controller/configuration/global-configuration/reporting-resources-status) for more details.

    > Learn more about type LoadBalancer in the [Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/service/#type-loadbalancer).

## Uninstall the Ingress Controller

1. Delete the `nginx-ingress` namespace to uninstall the Ingress controller along with all the auxiliary resources that were created:
    ```
    $ kubectl delete namespace nginx-ingress
    ```
2. Delete the ClusterRole and ClusterRoleBinding created in that step:
    ```
    $ kubectl delete clusterrole nginx-ingress
    $ kubectl delete clusterrolebinding nginx-ingress
    ```

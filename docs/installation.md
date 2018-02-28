# Installing the Ingress Controller

## Prerequisites 

Make sure you have access to the Ingress controller image:

* For NGINX Ingress controller, use the image `nginxdemos/nginx-ingress` from [DockerHub](https://hub.docker.com/r/nginxdemos/nginx-ingress/).
* For NGINX Plus Ingress controller, build your own image and push it to your private Docker registry by following the instructions from [here](../nginx-controller).

The installation manifests are located in the [install](../install) folder. In the steps below we assume that you will be running the commands from that folder.

## 1. Create a Namespace, a SA and the Default Secret.

1. Create a namespace and a service account for the Ingress controller:
    ```
    kubectl apply -f common/ns-and-sa.yaml

    ```

1. Create a secret with a TLS certificate and a key for the default server in NGINX:
    ```
    $ kubectl apply -f common/default-server-secret.yaml
    ```

    **Note**: The default server returns the Not Found page with the 404 status code for all requests for domains for which there are no Ingress rules defined. For testing purposes we include a self-signed certificate and key that we generated. However, we recommend that you use your own certificate and key.

1. *Optional*. Create a config map for customizing NGINX configuration (read more about customization [here](../examples/customization)):
    ```
    $ kubectl apply -f common/nginx-config.yaml
    ```

## 2. Configure RBAC

If RBAC is enabled in your cluster, create a cluster role and bind it to the service account, created in Step 1:
```
$ kubectl apply -f rbac/rbac.yaml
```

**Note**: To perform this step you must be a cluster admin.

## 3. Deploy the Ingress Controller

We include two options for deploying the Ingress controller:
* *Deployment*. Use a Deployment if you plan to dynamically change the number of Ingress controller replicas.
* *DaemonSet*. Use a DaemonSet for deploying the Ingress controller on every node or a subset of nodes. 

### 3.1 Create a Deployment

For NGINX, run:
```
$ kubectl apply -f deployment/nginx-ingress.yaml
```

For NGINX Plus, run:
```
$ kubectl apply -f deployment/nginx-plus-ingress.yaml
```

**Note**: Update the `nginx-plus-ingress.yaml` with the container image that you have built.

Kubernetes will create one Ingress controller pod.


### 3.2 Create a DaemonSet

For NGINX, run:
```
$ kubectl apply -f daemon-set/nginx-ingress.yaml
```

For NGINX Plus, run:
```
$ kubectl apply -f daemon-set/nginx-plus-ingress.yaml
```

**Note**: Update the `nginx-plus-ingress.yaml` with the container image that you have built.

Kubernetes will create an Ingress controller pod on every node of the cluster. Read [this doc](https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/) to learn how to run the Ingress controller on a subset of nodes, instead of every node of the cluster.

### 3.3 Check that the Ingress Controller is Running

Run the following command to make sure that the Ingress controller pods are running:
```
$ kubectl get pods --namespace=nginx-ingress
```

## 4. Get Access to the Ingress Controller

**If you created a daemonset**, ports 80 and 443 of the Ingress controller container are mapped to the same ports of the node where the container is running. To access the Ingress controller, use those ports and an IP address of any node of the cluster where the Ingress controller is running.

**If you created a deployment**, below are two options for accessing the Ingress controller pods.

### 4.1 Service with the Type NodePort

Create a service with the type *NodePort*:
```
$ kubectl create -f service/nodeport.yaml
```
Kubernetes will allocate two ports on every node of the cluster. To access the Ingress controller, use an IP address of any node of the cluster along with two allocated ports. Read more about the type NodePort [here](https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport).

### 4.2 Service with the Type LoadBalancer

Create a service with the type *LoadBalancer*. Kubernetes will allocate and configure a cloud load balancer for load balancing the Ingress controller pods.

Create a service using a manifest for your cloud provider:
* For GCP or Azure, run:
    ```
    $ kubectl apply -f service/loadbalancer.yaml
    ```
* For AWS, run:
    ```
    $ kubectl apply -f service/loadbalancer-aws-elb.yaml
    ```
    Kubernetes will allocate a Classic Load Balancer (ELB) in TCP mode with the PROXY protocol enabled to pass the client's information (the IP address and the port). NGINX must be configured to use the PROXY protocol:
    * Add the following keys to the config map file `nginx-config.yaml` from the Step 1 :
        ```
        proxy-protocol: "True"
        real-ip-header: "proxy_protocol"
        set-real-ip-from: "0.0.0.0/0"
        ```
    * Update the config map:
        ```
        kubectl apply -f common/nginx-config.yaml
        ```
    **Note**: For AWS, additional options regarding an allocated load balancer are available, such as the type of a load balancer and SSL termination. Read [this doc](https://kubernetes.io/docs/concepts/services-networking/service/#type-loadbalancer) to learn more.

Use the public IP of the load balancer to access the Ingress controller. To get the public IP:
* For GCP or Azure, run:
    ```
    $ kubectl get svc nginx-ingress --namespace=nginx-ingress
    ```
* In case of AWS ELB, the public IP is not reported by kubectl, as the IP addresses of the ELB are not static and you should not rely on them, but rely on the ELB DNS name instead. However, you can use them for testing purposes. To get the DNS name of the ELB, run:
    ```
    $ kubectl describe svc nginx-ingress --namespace=nginx-ingress
    ```
    You can resolve the DNS name into an IP address using `nslookup`:
    ```
    $ nslookup <dns-name>
    ```

Read more about the type LoadBalancer [here](https://kubernetes.io/docs/concepts/services-networking/service/#type-loadbalancer).

## 5. Access the Live Activity Monitoring Dashboard

For NGINX Plus, you can access the live activity monitoring dashboard:
1. Use `kubectl port-forward` command to forward connections to port 8080 on your local machine to port 8080 of an NGINX Plus Ingress controller pod (replace <nginx-plus-ingress-pod> with the actual name of a pod):
    ```
    $ kubectl port-forward <nginx-plus-ingress-pod> 8080:8080 --namespace=nginx-ingress 
    ```
1. Open your browser at http://127.0.0.1:8080/dashboard.html to access the dashboard. 

## Support For Prometheus Monitoring

If you are using [Prometheus](https://prometheus.io/), you can deploy the NGINX Plus Ingress controller with the prometheus exporter for NGINX Plus. The exporter will export NGINX Plus metrics into your Prometheus. To deploy the Ingress controller with the exporter, use the modified manifests:
* For a deployment, run:
    ```
    $ kubectl apply -f deployment/nginx-plus-ingress-with-prometheus.yaml
    ```
* For a daemon set, run:
    ```
    $ kubectl apply -f daemon-set/nginx-plus-ingress-with-prometheus.yaml
    ```

**Note**: this is a preview version of the prometheus exporter for NGINX Plus. It is not suitable for using in production environments.

## Uninstall the Ingress Controller

Delete the `nginx-ingress` namespace to uninstall the Ingress controller along with all the auxiliary resources that were created:
```
$ kubectl delete namespace nginx-ingress
```

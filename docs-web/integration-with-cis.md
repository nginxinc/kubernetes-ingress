# Integration with F5 Container Ingress Services

The integration with [F5 Container Ingress Services](https://clouddocs.f5.com/containers/v2/) (CIS) configures an F5 BIG-IP device as a load balancer for NGINX Ingress Controller pods.

> **Feature Status**: The integration with F5 CIS is available as a preview feature: it is suitable for experimenting and testing; however, it must be used with caution in production environments. Additionally, while the feature is in preview, we might introduce some backward-incompatible changes in the next releases.

## Prerequisites 

To enable the integration, the F5 CIS must be deployed in the cluster and configured to support the integration. Follow the instructions on the [CIS documentation portal](#link-to-be-added-later).

## Configuration 

### 1. Install the Ingress Controller with the Integration Enabled

This step depends on how you install the Ingress Controller: using [Manifests](/nginx-ingress-controller/installation/installation-with-manifests) or the [Helm chart](/nginx-ingress-controller/installation/installation-with-helm).

#### Manifests Installation

1. Create a service for the Ingress Controller pods for ports 80 and 443. For example:
    ```yaml
    apiVersion: v1
    kind: Service
    metadata:
      name: nginx-ingress-ingresslink 
      namespace: nginx-ingress
      labels:
        app: ingresslink
    spec:
      ports:
      - port: 80
        targetPort: 80
        protocol: TCP
        name: http
      - port: 443
        targetPort: 443
        protocol: TCP
        name: https
      selector:
        app: nginx-ingress
    ```
    Note the label `app: ingresslink`. We will use it in the Step 2. 
1. In the [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource), enable the PROXY protocol, which the BIG-IP system will use to pass the client IP and port information to NGINX. For the  `set-real-ip-from` key, use the subnet of the IP, which the BIG-IP system uses to send traffic to NGINX:
    ```yaml
    proxy-protocol: "True"
    real-ip-header: "proxy_protocol"
    set-real-ip-from: "0.0.0.0/0"
    ```
1. Deploy the Ingress Controller with additional [command-line arguments](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments):
    ```yaml
    args:
    - -ingresslink=nginx-ingress
    - -report-ingress-status 
    . . .
    ```
    where `ingresslink` references the name of the IngressLink resource from Step 2, and `report-ingress-status` enables [reporting Ingress statuses](/nginx-ingress-controller/configuration/global-configuration/reporting-resources-status#ingress-resources).

#### Helm Installation

Install a helm release with the following values that replicate the Manifest installation above:
```yaml
controller:
  config:
    entries:
      proxy-protocol: "True"
      real-ip-header: "proxy_protocol"
      set-real-ip-from: "0.0.0.0/0"
  reportIngressStatus:
    ingressLink: nginx-ingress
  service:
    type: ClusterIP
    externalTrafficPolicy: Cluster
    extraLabels:
      app: ingresslink 
```
We will use the values for the parameters `ingressLink` and `extraLabels` in Step 2. For the  `set-real-ip-from` key, use the subnet of the IP, which the BIG-IP system uses to send traffic to NGINX.  

### 2. Create an IngressLink Resource

To configure the BIG-IP device to load balance among the Ingress Controller pods, create an IngressLink resource. For example, the following resource will expose the Ingress Controller pods via `192.168.10.5`:
```yaml
apiVersion: "cis.f5.com/v1"
kind: IngressLink
metadata:
  name: nginx-ingress
  namespace: nginx-ingress
spec:
  virtualServerAddress: "192.168.10.5"
  iRules:
  - /Common/Proxy_Protocol_iRule
  selector:
    matchLabels:
      app: ingresslink 
```

The name of the resource and the labels in the selector must match the values you configured in Step 1. The resource must belong to the same namespace as the Ingress Controller pod.

### 3. Test the Integration

Now the Ingress Controller pods are behind the IP configured in Step 2.

If you deploy the [cafe example](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples/complete-example), you will be able to send requests to the Ingress Controller pods using the following command:
```
$ curl --resolve cafe.example.com:192.168.10.5:443 https://cafe.example.com:443/coffee --insecure
Server address: 10.12.0.18:80
Server name: coffee-7586895968-r26zn
...
```

Also, if you check the status of the cafe-ingress, you will see the IP of the BIG-IP system:
```
$ kubectl get ing cafe-ingress
NAME           HOSTS              ADDRESS         PORTS     AGE
cafe-ingress   cafe.example.com   192.168.10.5    80, 443   115s
```
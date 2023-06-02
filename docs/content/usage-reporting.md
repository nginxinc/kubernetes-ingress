---
title: Usage Reporting
description: |
How to enable usage reporting for NGINX Ingress Controller and how to view the usage data through the API.weight: 1800
doctypes: ["concept"]
toc: true
docs: ["???"]
---


This document outlines how to enable usage reporting for NGINX Ingress Controller and how to view the usage data through the API.

## Overview

The  Flexible Consumption Program (FCP) is a new pricing offer for NGINX Ingress Controller. The  pricing model is based on the number of NGINX Ingress Controller instances in a cluster. The number is reported to NGINX Management Suite, which is used to calculate the license cost. 

NGINX Cluster Connector is a Kubernetes controller that connects to the NGINX Management Suite and reports the number of NGINX Ingress Controller nodes in the cluster. The NGINX Cluster Connector is deployed as a Kubernetes Deployment in the same cluster where NGINX Ingress Controller is deployed.

To use the NGINX Cluster Connector, you must have access to NGINX Management Suite. For more information, see [NGINX Management Suite](https://www.nginx.com/products/nginx-management-suite/). 

### Requirements

To deploy the NGINX Cluster Connector, you must have the following:

| Requirements                            | Notes                                                                                                                               |
|-----------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------|
| NGINX Ingress Controller 3.1.0 or later | https://docs.nginx.com/nginx-ingress-controller                                                                                     |
| NGINX Management Suite 2.11 or later    | https://docs.nginx.com/nginx-management-suite                                                                                       |
| Docker 23 or later                      | https://docs.docker.com/get-docker/                                                                                                 |
| Kubernetes 1.22 or later                | Ensure your client can access the Kubernetes API server.                                                                            |
| kubectl 1.22 or later                   | https://kubernetes.io/docs/tasks/tools/#kubectl                                                                                     |
| OpenSSL 3.0.0 or later                  | For converting credentials string to base64. Alternatively, you can use any online tools available. https://www.openssl.org/source/ |

In addition to the software requirements, you must have the following:
- Access to NGINX Management Suite with username and password for basic authentication
- Access to the Kubernetes cluster where the NGINX Ingress Controller is deployed, with the ability to deploy a Kubernetes Deployment and a Kubernetes secret.

### Deploying the NGINX Cluster Connector

1. Basic authentication is required to connect the NGINX Management Suite API. A base64 representation of the username and password is required to create the Kubernetes secret. In this example the username will be `foo` and the password will be `bar`. To obtain the base64 representation of a string, use the following command:
    ```
    > echo -n 'foo' | base64
    Zm9v
    > echo -n 'bar' | base64
    YmFy
    ```
   
2. In order to make the credential available to NGINX Cluster Connector, create a Kubernetes secret by creating a file named `nms-basic-auth.yaml` with the following content, using the base64 representation of the username and password obtained in step 1:
    ```
    apiVersion: v1
    kind: Secret
    metadata:
      name: nms-basic-auth
      namespace: nginx-cluster-connector
    type: kubernetes.io/basic-auth
    data:
      username: Zm9v # base64 representation of 'foo' obtained in step 1
      password: YmFy # base64 representation of 'bar' obtained in step 1
    ```
    In the example, the namespace is `nginx-cluster-connector` and the secret name is `nms-basic-auth`. The namespace is the default namespace for the NGINX Cluster Connector. If you are using a different namespace, please change the namespace in the `metadata` section of the file. Note that the cluster connector only supports basic-auth secret type in `data` format, not `stringData`, with the username and password encoded in base64. 
3. If the namespace `nginx-cluster-connector` does not exist, create the namespace:
    ```
    > kubectl create namespace nginx-cluster-connector
    ```
4. Deploy the Kubernetes secret:
    ```
    > kubectl apply -f nms-basic-auth.yaml
    ```
   To change the username and password, update the secret and apply the changes to the Kubernetes cluster. The NGINX Cluster Connector will automatically detect the changes and use the new username and password without redeploying the NGINX Cluster Connector.

5. Download the [nginx-cluster.yaml](https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.1.1/deployments/deployment/cluster-connector.yaml). Edit the args section  in `deployment.yaml` to edit the following values;
   ```
        args:
        - -nms-server-address=https://k8s-usage.example.com
        - -nms-basic-auth-secret=nginx-cluster-connector/nms-basic-auth
   ```
   The `nms-server-address` should be the address of the NGINX Management Suite host. IPv4 addresses and hostnames are supported. The `nms-basic-auth-secret` should be the namespace/name of the secret created in step 2, i.e `nginx-cluster-connector/nms-basic-auth`.
   For more information on the command-line flags, see [Configuration](#configuration).

6. To deploy the Nginx Cluster Connector,  run the following command to deploy the application to your Kubernetes cluster:
   ```
   > kubectl apply -f cluster-connector.yaml
   ```


## Viewing Usage Data

#TODO: instructions how to get the UUID.
```
curl -k --user "foo:bar" https://k8s-usage.example.com/usage/{uuid}
```


## Uninstall
To remove the Nginx Cluster Connector from your Kubernetes cluster, run the following command:
```
kubectl delete -f deployment.yaml
```

## Cammand-line Arguments
The NGINX Cluster Connector supports several command-line arguments.
Below we describe the available command-line arguments:

### -nms-server-address `<string>`
The address of the NGINX Management Suite host. IPv4 addresses and hostnames are supported. 
Default `apigw.nms.svc.cluster.local`.

### -nms-server-port `<int>`
The port of the NGINX Management Suite host
Default `443`. 

### -nms-basic-auth-secret `<string>`
Secret for basic authentication to the NGINX Management Suite API. The secret must be in `kubernetes.io/basic-auth` format using base64 encoding.
Format `<namespace>/<name>`.

### -cluster-display-name `<string>`
The display name of the Kubernetes cluster.

### -skip-tls-verify
Skip TLS verification for the NMS server. **For testing purposes with NMS server with self-assigned certificate.**

### -min-update-interval `<string>`
The minimum interval between updates to the NMS.
Default `24h`.

### -proxy
Use a proxy server to connect to the Kubernetes API started by the "kubectl proxy" command. **For testing purposes only.**


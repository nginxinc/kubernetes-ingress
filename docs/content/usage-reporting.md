---
title: Usage Reporting
description: |
How to enable usage reporting for NGINX Ingress Controller and view the usage data through the API.
weight: 1800
doctypes: ["concept"]
toc: true
docs: ["???"]
---


This document outlines how to enable usage reporting for NGINX Ingress Controller and how to view the usage data through the API.

## Overview

The Flexible Consumption Program (FCP) is a new pricing offer for NGINX Ingress Controller. The pricing model is based on the number of NGINX Ingress Controller instances in a cluster. The number is reported to NGINX Management Suite, which is used to calculate the license cost.

NGINX Cluster Connector is a Kubernetes controller that connects to the NGINX Management Suite and reports the number of NGINX Ingress Controller nodes in the cluster. The NGINX Cluster Connector is deployed as a Kubernetes Deployment in the same cluster where NGINX Ingress Controller is deployed.

To use the NGINX Cluster Connector, you must have access to NGINX Management Suite. For more information, see [NGINX Management Suite](https://www.nginx.com/products/nginx-management-suite/).

## Requirements

To deploy Usage Reporting, you must have the following:

| Requirements                            | Notes                                                                                                                 |
|-----------------------------------------|-----------------------------------------------------------------------------------------------------------------------|
| NGINX Ingress Controller 3.1.0 or later | https://docs.nginx.com/nginx-ingress-controller                                                                       |
| NGINX Management Suite 2.11 or later    | https://docs.nginx.com/nginx-management-suite                                                                         |
| Docker 23 or later                      | https://docs.docker.com/get-docker/                                                                                   |
| Kubernetes 1.22 or later                | Ensure you can access the Kubernetes API, with the ability to deploy a Kubernetes Deployment and a Kubernetes secret. |
| kubectl 1.22 or later                   | https://kubernetes.io/docs/tasks/tools/#kubectl                                                                       |

In addition to the software requirements, you must have the following:
- Access to NGINX Management Suite with username and password for basic authentication, including the URL, username, and password, with the access to the `/api/platform/v1/k8s-usage` endpoints
- Access to the Kubernetes cluster where the NGINX Ingress Controller is deployed, with the ability to deploy a Kubernetes Deployment and a Kubernetes Secret.
- Access to public internet to pull the image, since the image is hosted in the NGINX container registry at `registry.nginx.com/cluster-connector`. Alternatively, you can pull the image down and push it to a private container registry that your cluster has access to.
[//]: # (  TODO: Update the image and tag after published)

## Setting up user accounts in NGINX Management Suite

The NGINX Cluster Connector uses the NGINX Management Suite API to report the number of NGINX Ingress Controller nodes in the cluster. To use the NGINX Management Suite API, you must have a user account with the correct role.

1. Create a role following the steps in [Create a Role](https://docs.nginx.com/nginx-management-suite/admin-guides/access-control/set-up-rbac/#create-role) section of the NGINX Management Suite documentation. Select these permissions in step 6 for the role:
   - Module: Instance Manager
   - Feature: Nginx Plus Usage
   - Access: CRUD

2. Create a user account following the steps in [Add Users](https://docs.nginx.com/nginx-management-suite/admin-guides/access-control/set-up-rbac/#add-users) section of the NGINX Management Suite documentation. In step 6, assign the user to the role created above. Note that currently only basic auth authentication is supported for usage reporting purposes.

## Deploying the NGINX Cluster Connector

1. Basic authentication is required to connect the NGINX Management Suite API. If you do not have a user account, create one following the above section . A base64 representation of the username and password is required to create the Kubernetes secret. In this example the username will be `foo` and the password will be `bar`. To obtain the base64 representation of a string, use the following command:
    ```
    > echo -n 'foo' | base64
    Zm9v
    > echo -n 'bar' | base64
    YmFy
    ```

2. In order to make the credential available to NGINX Cluster Connector, we need to create a Kubernetes secret. Copying the following content to a text editor, and fill in the username and password under `data`, with base64 representation of the username and password obtained in step 1:
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
    Save this in a file named `nms-basic-auth.yaml`. In the example, the namespace is `nginx-cluster-connector` and the secret name is `nms-basic-auth`. The namespace is the default namespace for the NGINX Cluster Connector. If you are using a different namespace, please change the namespace in the `metadata` section of the file. Note that the cluster connector only supports basic-auth secret type in `data` format, not `stringData`, with the username and password encoded in base64.
3. If the namespace `nginx-cluster-connector` does not exist, create the namespace:
    ```
    > kubectl create namespace nginx-cluster-connector
    ```
4. Deploy the Kubernetes secret created in step 2 to the Kubernetes cluster:
    ```
    > kubectl apply -f nms-basic-auth.yaml
    ```
   If you want to update the basic-auth credentials to NMS server in the future, update the `username` and `password` fields, and apply the changes by running the command again. The NGINX Cluster Connector will automatically detect the changes and use the new username and password without redeploying the NGINX Cluster Connector.

5. Download the deployment file [nginx-cluster.yaml](https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.1.1/deployments/deployment/cluster-connector.yaml). Edit the following under `args` section
   ```
        args:
        - -nms-server-address=https://nms.example.com/api/platform/v1/k8s-usage
        - -nms-basic-auth-secret=nginx-cluster-connector/nms-basic-auth
   ```
   The `-nms-server-address` should be the address of the NGINX Management Suite host. The `nms-basic-auth-secret` should be the namespace/name of the secret created in step 2, i.e `nginx-cluster-connector/nms-basic-auth`.
   For more information on the command-line flags, see [Configuration](#configuration).

6. To deploy the Nginx Cluster Connector, save the file above and run the following command to deploy it to your Kubernetes cluster:
   ```
   > kubectl apply -f cluster-connector.yaml
   ```


## Viewing Usage Data from the NGINX Management Suite API
The NGINX Cluster Connector reports the number of NGINX Ingress Controller instances and nodes in the cluster to NGINX Management Suite. To view the usage data, query the NGINX Management Suite API. The usage data is available in the following endpoint:

```
> curl --user "foo:bar" https://nms.example.com/api/platform/v1/k8s-usage
{
  "items": [
    {
      "metadata": {
        "displayName": "my-cluster",
        "uid": "d290f1ee-6c54-4b01-90e6-d701748f0851",
        "createTime": "2023-01-27T09:12:33.001Z",
        "updateTime": "2023-01-29T10:12:33.001Z",
        "monthReturned": "May"
      },
      "node_count": 4,
      "max_node_count": 5,
      "pod_details": {
        "current_pod_counts": {
          "pod_count": 15,
          "waf_count": 5,
          "dos_count": 0
        },
        "max_pod_counts": {
          "max_pod_count": 25,
          "max_waf_count": 7,
          "max_dos_count": 1
        }
      }
    },
    {
      "metadata": {
        "displayName": "my-cluster2",
        "uid": "12tgb8ug-g8ik-bs7h-gj3j-hjitk672946hb",
        "createTime": "2023-01-25T09:12:33.001Z",
        "updateTime": "2023-01-26T10:12:33.001Z",
        "monthReturned": "May"
      },
      "node_count": 3,
      "max_node_count": 3,
      "pod_details": {
        "current_pod_counts": {
          "pod_count": 5,
          "waf_count": 5,
          "dos_count": 0
        },
        "max_pod_counts": {
          "max_pod_count": 15,
          "max_waf_count": 5,
          "max_dos_count": 0
        }
      }
    }
  ]
}
```
You can specify `displayName` for the cluster in the response  with the `-cluster-display-name` command-line argument when you deploy the Cluster Connector . See [Command-line Arguments](#Command-line Arguments) for more information. From this response, you can see the cluster uid corresponding to the cluster name.

You can also query the usage data for a specific cluster by specifying the cluster uid in the endpoint, for example:
```
> curl --user "foo:bar" https://nms.example.com/api/platform/v1/k8s-usage/d290f1ee-6c54-4b01-90e6-d701748f0851
{
  "metadata": {
    "displayName": "my-cluster",
    "uid": "d290f1ee-6c54-4b01-90e6-d701748f0851",
    "createTime": "2023-01-27T09:12:33.001Z",
    "updateTime": "2023-01-29T10:12:33.001Z",
    "monthReturned": "May"
  },
  "node_count": 4,
  "max_node_count": 5,
  "pod_details": {
    "current_pod_counts": {
      "pod_count": 15,
      "waf_count": 5,
      "dos_count": 0
    },
    "max_pod_counts": {
      "max_pod_count": 25,
      "max_waf_count": 7,
      "max_dos_count": 1
    }
  }
}
```

## Uninstalling the NGINX Cluster Connector
To remove the Nginx Cluster Connector from your Kubernetes cluster, run the following command:
```
kubectl delete -f cluster-connector.yaml
```

## Command-line Arguments
The NGINX Cluster Connector supports several command-line arguments. The command-line arguments can be specified in the `args` section of the Kubernetes deployment file. The following is a list of the supported command-line arguments and their usage:

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

---
title: NGINX Ingress Controller and Open Service Mesh
description: |
  Use NGINX Ingress Controller with Open Service Mesh.
weight: 1800
doctypes: ["concept"]
toc: true
docs: "DOCS-1181"
---

# OIDC Custom Configuration

The F5 NGINX Ingress Controller implements OpenID Connect (OIDC) using the NGINX OpenID Connect Reference implementation: [nginx-openid-connect](https://github.com/nginxinc/nginx-openid-connect).

This guide will walk through how to customise and configure this default implementation.

## Prerequisites
This guide assumes that you have an F5 NGINX Ingress Controller deployed. If not, please follow the installation steps using either the [Manifest](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/) or [HELM](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/) approach.

To customise the NGINX OpenID Connect Reference implementation, we will need to:
1. Create a ConfigMap containing the contents of the default `oidc.conf` file
2. Attach a `Volume` and `VolumeMount` to our deployment of the F5 NGINX Ingress Controller

This setup will allow our custom configuration in our ConfigMap to override the contents of the default `oidc.conf` file.

## Step 1 - Creating the ConfigMap

Run the below command to generate a ConfgMap with the contents of the `oidc.conf` file.
**NOTE** The ConfgMap must be deployed in the same `namespace` as the F5 NGINX Ingress Controller.
```
kubectl create configmap oidc-config-map --from-literal=oidc.conf="$(curl -k https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.1.1/internal/configs/oidc/oidc.conf)"
```

Use the `kubectl describe` command to confirm the contents of the ConfigMap are correct

```
kubectl describe configmap oidc-config-map
```

```
Name:         oidc-config-map
Namespace:    default
Labels:       <none>
Annotations:  <none>

Data
====
oidc.conf:
----
    # Advanced configuration START
    set $internal_error_message "NGINX / OpenID Connect login failure\n";
    set $pkce_id "";
    # resolver 8.8.8.8; # For DNS lookup of IdP endpoints;
    subrequest_output_buffer_size 32k; # To fit a complete tokenset response
    gunzip on; # Decompress IdP responses if necessary
    # Advanced configuration END

    ...
    # Rest of config ammended
```

## Step 2 - Customising the default configuration
Once the contents of the `oidc.conf` file has been added to the ConfigMap, you are free to customise the contents of this ConfigMap.
In this example, we will add a comment to the top of the file to demonstrate it is overwriting the default contents.
This comment will be `# >> Custom Comment for my OIDC file <<`

```
kubectl edit configmap oidc-config-map
```

Add the custom content:
```
# Please edit the object below. Lines beginning with a '#' will be ignored,
# and an empty file will abort the edit. If an error occurs while saving this file will be
# reopened with the relevant failures.
#
apiVersion: v1
data:
  oidc.conf: |2-
        # >> Custom Comment for my OIDC file <<
        # Advanced configuration START
        set $internal_error_message "NGINX / OpenID Connect login failure\n";
        set $pkce_id "";
        # resolver 8.8.8.8; # For DNS lookup of IdP endpoints;
        subrequest_output_buffer_size 32k; # To fit a complete tokenset response
        gunzip on; # Decompress IdP responses if necessary
        # Advanced configuration END

        ...
        # Rest of config ammended
```
> **IMPORTANT**
>
> In Step 3 we will deploy/update an Ingress Controller that will use this ConfigMap. Any changes made to this ConfigMap must be made **before** deploying/updating the Ingress Controller. If an update is applied to the ConfigMap after the Ingress Controller is deployed, it will not get applied. Applying any updates to the data in this ConfigMap will require the Ingress Controller to be re-deployed.

## Step 3 - Add Volume and VolumeMount to the Ingress Controller deployment

In this step we will add a `Volume` and `VolumeMount` to our Ingress Controller deployment.
This will allow us to mount the ConfigMap created in Step 1 and over write the contents of the `oidc.conf` file.

This document will demonstrate how to add the `Volume` and `VolumeMount` using both Manifest and HELM

### Manifest

The below configuration shows where the `Volume` and `VolumeMount` can be added to your Deployment/Daemonset file.

The `VolumeMount` must be added the `spec.template.spec.containers` section.

The `Volume` must be added the `spec.template.spec` section:
```
apiVersion: apps/v1
kind: <Deployment/Daemonset>
metadata:
  name: <name>
  namespace: <ic-namespace>
spec:
  ...
  ...
  template:
    ...
    ...
    spec:
      ...
      ...
      volumes:
      - name: oidc-volume
        configMap:
          name: <config-map-name> # Must match the name of the ConfigMap
      containers:
        ...
        ...
        volumeMounts:
        - name: oidc-volume
          mountPath: /etc/nginx/oidc/oidc.conf
          subPath: oidc.conf # Must match the name in the data filed
          readOnly: true
```

Once the `Volume` and `VolumeMount` has been added the manifest file, apply the changes do the Ingress Controller deployment.

Confirm the `oidc.conf` file has been updated:
```
kubectl exec -it -n <ic-namespace> <ingess-controller-pod> -- cat /etc/nginx/oidc/oidc.conf
```

### HELM

Deployments using HELM will need to edit their existing
Edit your Ingress Controller Deployment/Daemonset yaml to include a `Volume` and `VolumeMount`.

The `Volume` should be within the `spec.template.spec` section.

The `VolumeMount `must be added the `spec.template.spec.containers` section.

For Deployments:
```
kubectl edit deployments <name-of-deployment> -n <ic-namespace>
```

For Daemonsets:
```
kubectl edit daemonset <name-of-daemonset> -n <ic-namespace>
```

```
apiVersion: apps/v1
kind: <Deployment/Daemonset>
metadata:
  name: <name>
  namespace: <ic-namespace>
spec:
  ...
  ...
  template:
    ...
    ...
    spec:
      ...
      ...
      volumes:
      - name: oidc-volume
        configMap:
          name: <config-map-name> # Must match the name of the ConfigMap
      containers:
        ...
        ...
        volumeMounts:
        - name: oidc-volume
          mountPath: /etc/nginx/oidc/oidc.conf
          subPath: oidc.conf # Must match the name in the data filed
          readOnly: true
```

Once the Deployment/Daemonset has been edited, save the file and exit.

Confirm the `oidc.conf` file has been updated:
```
kubectl exec -it -n <ic-namespace> <ingess-controller-pod> -- cat /etc/nginx/oidc/oidc.conf
```

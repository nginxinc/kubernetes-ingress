---
docs: DOCS-000
title: Compile NAP WAF policies using NGINX Instance Manager
weight: 300
---

## Overview

This guide describes how to use F5 NGINX Instance Manager to compile NGINX App WAF Policies for use with NGINX Ingress Controller.

NGINX App Protect WAF uses policies to configure which security features are enabled or disabled. When these policies are changed, they need to be compiled so that the engine can begin to use them. Compiling policies can take a large amount of time and resources (Depending on the size), so the preferred way to do this is with NGINX Instance Manager, reducing the impact on a NGINX Ingress Controller deployment.

By using NGINX Instance Manager to compile WAF policies, the policy bundle can also be used immediately by NGINX Ingress Controller without reloading.

The following steps describe how to use the NGINX Instance Manager API to create a new security policy, compile a bundle, then add it to NGINX Ingress Controller.

## Before you start
### Requirements
- A working [NGINX Management Suite](https://docs.nginx.com/nginx-management-suite/installation/) instance.
- A NGINX Ingress Controller [deployment with NGINX App Protect WAF]({{< relref "/installation/integrations/app-protect-waf/installation" >}}).

## Configure NGINX Instance Manager
<!-- This looks like it corresponds to step 1 of the draft:

- The reader will need a working NMS instance (Covered in the requirements above)
- They will need a user account with API access (Link to NIM/NMS RBAC user creation/auth documentation?) -->

## Create a new security policy

<!-- step 2 of the draft -->

## Create a new security bundle

<!-- step 3 of the draft -->

## Download the security bundle

<!-- step 4 & 5 of the draft - list and download can be one step -->

## Add volumes and volumeMounts to NGINX Ingress Controller

<!-- step 6 of the draft -->

## Upload the security bundle

<!-- step 9 of the draft -->

## Create WAF policy

<!-- step 7 of the draft -->

## Create VirtualServer resource and apply policy

<!-- step 8 of the draft -->

<!-- ROUGH WORK BELOW THIS POINT -->

# Outline of the steps to follow

1. Setup NIM console (install all necessary pieces including authentication)
2. Create a new security policy through the API
    a. To create security policy bundles, send an HTTP POST request to the Security Policies Bundles API endpoint. The specified security policies you’d like to compile into security policy bundles must already exist in Instance Manager. This will be a .json file that will be uploaded (POST) to NIM.
    b. The content of the policy **must** be base64 encoded before uploading.
3. Create a new security bundle through the API.
    a. This has to have specific requirements for the security policy it will be attached to.
4. Once the bundle has been compiled, download the bundle through the API
5. Use new bundle with NAP in NIC

# Step 1.
## Setup and ensure NIM is running and a user account as Access to the API

The first step required when creating a NAP bundle for NIC is to create a new security policy, or use an exisiting security policy within NIM. This is required as we will associate the bundle we are creating with the security policy, allowing us to use a variety of policies and their perspective bundles as needed. In this example, we are going to be using the NIM API to create both the security policy and the bundle. Once those are created, we are then going to download the bundle in a .tgz format, which can then be used with NIC+NAP.

Before we can create our policy and generate the bundle, we need to make sure have proper access. Once you have created the account to be used and has proper RBAC access, we need to Authorize the request user to the NIM API. You can use Basic Auth or OIDC.

# Step 2.
## Create the security policy 

Now we are going to create our security policy using the API.      
https://docs.nginx.com/nginx-management-suite/nim/how-to/app-protect/manage-waf-security-policies/#create-security-policy

We can use a tool like `Postman`, or `curl` to interact with API. 

We will be uploading .json files to create the security policy as well as the bundle in this document.

NOTE: For the policy "content" itself, this **MUST** be base64 encoded or it will fail.

For are example, we are going to use this`simple-policy.json` file with contents below:
```json
{
  "metadata": {
    "name": "Nginxbundletest",
    "displayName": "Nginxbundletest",
    "description": "Ignore cross-site scripting is a security policy that intentionally ignores cross site scripting."
  },
  "content": "ewoJInBvbGljeSI6IHsKCQkibmFtZSI6ICJzaW1wbGUtYmxvY2tpbmctcG9saWN5IiwKCQkic2lnbmF0dXJlcyI6IFsKCQkJewoJCQkJInNpZ25hdHVyZUlkIjogMjAwMDAxODM0LAoJCQkJImVuYWJsZWQiOiBmYWxzZQoJCQl9CgkJXSwKCQkidGVtcGxhdGUiOiB7CgkJCSJuYW1lIjogIlBPTElDWV9URU1QTEFURV9OR0lOWF9CQVNFIgoJCX0sCgkJImFwcGxpY2F0aW9uTGFuZ3VhZ2UiOiAidXRmLTgiLAoJCSJlbmZvcmNlbWVudE1vZGUiOiAiYmxvY2tpbmciCgl9Cn0="
}
```

Next step, we need to POST the new security policy into NIM through the API.

```shell
curl -X POST https://{{NMS_FQDN}}/api/platform/v1/security/policies \
    -H "Authorization: Bearer <access token>" \
    -d @simple-policy.json
```

THe response back from the API, after the policy is successfully created.

```json
{
    "metadata": {
        "created": "2024-06-12T20:28:08.152171922Z",
        "description": "Ignore cross-site scripting is a security policy that intentionally ignores cross site scripting.",
        "displayName": "Nginxbundletest",
        "externalId": "",
        "externalIdType": "",
        "modified": "2024-06-12T20:28:08.152171922Z",
        "name": "Nginxbundletest",
        "revisionTimestamp": "2024-06-12T20:28:08.152171922Z",
        "uid": "6af9f261-658b-4be1-b07a-cebd83e917a1"
    },
    "selfLink": {
        "rel": "/api/platform/v1/security/policies/6af9f261-658b-4be1-b07a-cebd83e917a1"
    }
}
```

NOTE: Take note of the `uid` field as we will need that when we download our bundle.

# Step 3.   
## Create the security bundle

Next, we are going to create the security policy bundle using the API.   
https://docs.nginx.com/nginx-management-suite/nim/how-to/app-protect/manage-waf-security-policies/#create-security-policy-bundles   

A very important item that must be followed is that the bundle you will be creating, must line up with the proper WAF compiler version you are using. If you use the wrong version in your json payload (see above), you will receive an error similar to the below:

```text
{"code":13018,"message":"Error compiling the security policy set: One or more of the specified compiler versions does not exist. Check the compiler versions, then try again."}
```

You can verify which compilers are installed on NIM, by checking which packages have been installed through the operating system (apt, dnf for example):

Here is the contents of our `security-policy-bundles.json` file:

```json
{
  "bundles": [
    {
      "appProtectWAFVersion": "4.815.0",
      "policyName": "Nginxbundletest",
      "policyUID": "",
      "attackSignatureVersionDateTime": "latest",
      "threatCampaignVersionDateTime": "latest"
    }
  ]
}
```

Now, we send our request to create the bundle in NIM:

```
curl -X POST https://{{NMS_FQDN}}/api/platform/v1/security/policies/bundles \
    -H "Authorization: Bearer <access token>" \
    -d @security-policy-bundles.json
```
Response from NIM after successful POST of our bundle:

```json
{
    "items": [
        {
            "compilationStatus": {
                "message": "",
                "status": "compiling"
            },
            "content": "",
            "metadata": {
                "appProtectWAFVersion": "4.815.0",
                "attackSignatureVersionDateTime": "2024.02.21",
                "created": "2024-06-12T13:28:20.023775785-07:00",
                "modified": "2024-06-12T13:28:20.023775785-07:00",
                "policyName": "Nginxbundletest",
                "policyUID": "6af9f261-658b-4be1-b07a-cebd83e917a1",
                "threatCampaignVersionDateTime": "2024.02.25",
                "uid": "cbdf9577-6d81-43d6-8ce1-2e3d4714e8b5"
            }
        }
    ]
}
```

Now we can query for the bundlle that we just created in the above command:

# Step 4.
## List security bundles
Here we are going to list or newly created bundle
Here we can list our security bundles:

```
curl --location 'https://127.0.0.1/api/platform/v1/security/policies/bundles' \
-H 'Authorization: Bearer <access_token>
```
Response showing our bundle we created earlier
```json
{
    "items": [
        {
            "compilationStatus": {
                "message": "",
                "status": "compiled"
            },
            "content": "",
            "metadata": {
                "appProtectWAFVersion": "4.815.0",
                "attackSignatureVersionDateTime": "2024.02.21",
                "created": "2024-06-13T09:09:10.809-07:00",
                "modified": "2024-06-13T09:09:20-07:00",
                "policyName": "Nginxbundletest",
                "policyUID": "ec8681eb-1e25-4b71-93bd-b91f67c5ac99",
                "threatCampaignVersionDateTime": "2024.02.25",
                "uid": "de08b324-99d8-4155-b2eb-fe687b21034e"
            }
        }
    ]
}
```
Take note of the `uid` field. this is the UID for the security bundle which is required when download our bundle once it is compiled.


# Step 5.
## Download our security bundle

```shell
curl -X GET "https://{NMS_FQDN}/api/platform/v1/security/policies/{security-policy-uid}/bundles/{security-policy-bundle-uid}" -H "Authorization: Bearer xxxxx.yyyyy.zzzzz" | jq -r '.content' | base64 -d > security-policy-bundle.tgz
```

In our example, we are using the `seucrity-policy-id` and the `security-policy-bundle-id`
```shell
curl -X GET -k 'https://127.0.0.1/api/platform/v1/security/policies/6af9f261-658b-4be1-b07a-cebd83e917a1/bundles/de08b324-99d8-4155-b2eb-fe687b21034e' \                                                                                                     
    -H "Authorization: Basic YWRtaW46UncxQXBQS3lRRTRuQXRXOFRYa1J4ZFdVSWVTSGtU" \
     | jq -r '.content' | base64 -d > security-policy-bundle.tgz
```


# Step 6. 
## Configure NIC Deployment

Since we are going to use bundles for WAF running on NGINX Ingress controller, we will need to modify the deployment for NIC to add volumes and volumeMounts, where NIC can pick up the bundle when new ones are uploaded to the cluster. This path is specific and must be correct in order for the bundle to be pickedup and used within NIC:
Quick overview of what needs to be added:

```yaml
volumes:
- name: <volume_name>
persistentVolumeClaim:
    claimName: <claim_name>

volumeMounts:
- name: <volume_mount_name>
    mountPath: /etc/nginx/waf/bundles
```

Full example of a deployment file with `volumes` and `volumeMounts` added:

```yaml

apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-ingress
  namespace: nginx-ingress
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-ingress
  template:
    metadata:
      labels:
        app: nginx-ingress
        app.kubernetes.io/name: nginx-ingress
     #annotations:
       #prometheus.io/scrape: "true"
       #prometheus.io/port: "9113"
       #prometheus.io/scheme: http
    spec:
      serviceAccountName: nginx-ingress
      automountServiceAccountToken: true
      securityContext:
        seccompProfile:
          type: RuntimeDefault
      volumes:
      - name: nginx-bundle-mount
        emptydir: {}
      containers:
      - image: <replace>
        imagePullPolicy: IfNotPresent
        name: nginx-ingress
        ports:
        - name: http
          containerPort: 80
        - name: https
          containerPort: 443
        - name: readiness-port
          containerPort: 8081
        - name: prometheus
          containerPort: 9113
        readinessProbe:
          httpGet:
            path: /nginx-ready
            port: readiness-port
          periodSeconds: 1
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
         #limits:
         #  cpu: "1"
         #  memory: "1Gi"
        securityContext:
          allowPrivilegeEscalation: false
          runAsUser: 101 #nginx
          runAsNonRoot: true
          capabilities:
            drop:
            - ALL
            add:
            - NET_BIND_SERVICE
        volumeMounts:
        -  name: bundle-mount
           mountPath: /etc/nginx/waf/bundles
        env:
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        args:
          - -nginx-configmaps=$(POD_NAMESPACE)/nginx-config
          - -report-ingress-status
          - -external-service=nginx-ingress
```

# Step 7.
## Create WAF policy:

Before applying a policy, a WAF policy needs to be created. This WAF policy will use the newly created bundle we did in the previous steps. It must be copied over to `/etc/nginx/waf/bundles` so NIC can load the new bundle into WAF. 

In the below, `spec.waf.apBundle` is the name of the bundle that we downloaded from NIM. 

```yaml
apiVersion: k8s.nginx.org/v1
kind: Policy
metadata:
  name: waf-policy
spec:
  waf:
    enable: true
    apBundle: "<bundle-name>.tgz" ### <-- this is the name of the bundle downloaded from NIM
    securityLogs:
    - enable: true
        apLogConf: "<bundle-name>.tgz"
        logDest: "syslog:server=syslog-svc.default:514"
```

# Step 8.
## Create virtualServer resource and apply WAF policy
Now that we have our WAF policy created, we can now link the policy to our `virtualServer` resource:

```yaml
apiVersion: k8s.nginx.org/v1
kind: VirtualServer
metadata:
  name: webapp
spec:
  host: webapp.example.com
  policies:
  - name: waf-policy
  upstreams:
  - name: webapp
    service: webapp-svc
    port: 80
  routes:
  - path: /
    action:
      pass: webapp
```

# Step 9. 
## Upload the bundle to Kubernetes storage
Upload tarball to kubernetes cluster. `kubectl cp` or another mechanism.    
Once the new bundle is uploaded to the kubernetes cluster, NIC will pick up the new bundle and load in the new WAF policy automatically.

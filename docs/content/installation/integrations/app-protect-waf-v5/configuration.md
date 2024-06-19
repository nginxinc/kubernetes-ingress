---
docs: DOCS-000
doctypes:
- ''
title: Configuration
toc: true
weight: 200
---

This document explains how to use F5 NGINX Ingress Controller to configure [NGINX App Protect WAF v5](https://docs.nginx.com/nginx-app-protect-waf/v5/).

{{< note >}} Check out the complete NGINX Ingress Controller with NGINX App Protect WAF example resources on GitHub [for VirtualServer resources](https://github.com/nginxinc/kubernetes-ingress/tree/v3.6.0/examples/custom-resources/app-protect-waf-v5) and [for Ingress resources](https://github.com/nginxinc/kubernetes-ingress/tree/v3.6.0/examples/ingress-resources/app-protect-waf-v5).{{< /note >}}


## Global Configuration

NGINX Ingress Controller has global configuration parameters that match those in NGINX App Protect WAF. They are found in the [ConfigMap resource]({{< relref "configuration/global-configuration/configmap-resource.md#modules" >}}): the NGINX App Protect WAF parameters are prefixed with `app-protect*`.

## Enable NGINX App Protect WAF v5

NGINX App Protect WAF can be enabled and configured for custom resources (VirtualServer, VirtualServerRoute) or Ingress resources.

- For custom resources, you need to create a Policy Custom Resource referencing a policy bundle, then add it to the VirtualServer definition. Additional detail can be found in the [Policy Resource documentation]({{< relref "configuration/policy-resource.md#waf" >}}).
- For Ingress resources, apply the [`app-protect` annotations]({{< relref "configuration/ingress-resources/advanced-configuration-with-annotations.md#app-protect" >}}) to each desired resource.



## NGINX App Protect WAF Bundles {#waf-bundles}

You define App Protect WAF bundles for VirtualServer custom resources by creating policy bundles and putting them on a mounted volume accessible from NGINX Ingress Controller.

Before applying a policy, a WAF policy bundle must be created, then copied to a volume mounted to `/etc/app_protect/bundles`.

{{< note >}} NGINX Ingress Controller supports `securityLogs` for policy bundles. Log bundles must also be copied to a volume mounted to `/etc/app_protect/bundles`. {{< /note >}}

This example shows how a policy is configured by referencing a generated WAF Policy Bundle:


```yaml
apiVersion: k8s.nginx.org/v1
kind: Policy
metadata:
  name: <policy_name>
spec:
  waf:
    enable: true
    apBundle: "<policy_bundle_name>.tgz"
```

This example shows the same policy as above but with a log bundle used for security log configuration:


```yaml
apiVersion: k8s.nginx.org/v1
kind: Policy
metadata:
  name: <policy_name>
spec:
  waf:
    enable: true
    apBundle: "<policy_bundle_name>.tgz"
    securityLogs:
    - enable: true
      apLogBundle: "<log_bundle_name>.tgz"
      logDest: "syslog:server=syslog-svc.default:514"
```

## NGINX App Protect WAF Logs {#waf-logs}

todo

## NGINX App Protect WAF User Defined Signatures

todo


## OpenAPI Specification in NGINX Ingress Controller

The OpenAPI Specification defines the spec file format needed to describe RESTful APIs. The spec file can be written either in JSON or YAML. Using a spec file simplifies the work of implementing API protection. Refer to the [OpenAPI Specification](https://github.com/OAI/OpenAPI-Specification) (formerly called Swagger) for details.

NGINX Ingress Controller supports OpenAPI Specification versions 2.0 and 3.0.

The simplest way to create an API protection policy is using an OpenAPI Specification file to import the details of the APIs. If you use an OpenAPI Specification file, NGINX App Protect WAF will automatically create a policy for the following properties (depending on what's included in the spec file):

- Methods
- URLs
- Parameters
- JSON profiles

An OpenAPI-ready policy template is provided with the NGINX App Protect WAF packages and is located in: `/etc/app_protect/conf/NginxApiSecurityPolicy.json`

It contains violations related to OpenAPI set to blocking (enforced).

## Configuration in NGINX Plus Ingress Controller using Virtual Server Resource

In this example we deploy NGINX Ingress Controller with NGINX Plus and NGINX App Protect WAF v5, deploy a simple web application, and then configure load balancing and WAF protection for that application using the VirtualServer resource.

{{< note >}} You can find the example, and the files referenced, on [GitHub](https://github.com/nginxinc/kubernetes-ingress/tree/v3.6.0/examples/custom-resources/app-protect-waf/app-protect-waf-v55).{{< /note >}}

## Prerequisites

1. Follow the installation [instructions]({{< relref "installation/integrations/app-protect-waf/installation.md" >}}) to deploy NGINX Ingress Controller with NGINX Plus and NGINX App Protect WAF.

2. Save the public IP address of NGINX Ingress Controller into a shell variable:

   ```shell
    IC_IP=XXX.YYY.ZZZ.III
   ```

3. Save the HTTP port of NGINX Ingress Controller into a shell variable:

   ```shell
    IC_HTTP_PORT=<port number>
   ```

### Step 1. Deploy a Web Application

Create the application deployment and service:

  ```shell
  kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.6.0/examples/custom-resources/app-protect-waf-v5/webapp.yaml
  ```

### Step 2. Deploy the Policy Bundle

Create the syslog service and pod for the NGINX App Protect WAF security logs:

   ```shell
   kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.6.0/examples/custom-resources/app-protect-waf-v5/syslog.yaml
   ```

### Step 3 - Deploy the WAF Policy

Create the WAF policy

 ```shell
  kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.6.0/examples/custom-resources/app-protect-waf-v5/waf.yaml
 ```

  Note the NGINX App Protect WAF configuration settings in the Policy resource. They enable WAF protection by configuring NGINX App Protect WAF with the policy and log configuration created in the previous step.

### Step 4 - Configure Load Balancing

1. Create the VirtualServer Resource:

    ```shell
    kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.6.0/examples/custom-resources/app-protect-waf-v5/virtual-server.yaml
    ```

Note that the VirtualServer references the policy waf-policy bundle created in Step 3.

### Step 5 - Test the Application

To access the application, curl the coffee and the tea services. We'll use the --resolve option to set the Host header of a request with `webapp.example.com`

1. Send a request to the application:

  ```shell
  curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT/
  ```

  ```shell
  Server address: 10.12.0.18:80
  Server name: webapp-7586895968-r26zn
  ```

1. Now, let's try to send a request with a suspicious URL:

  ```shell
  curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP "http://webapp.example.com:$IC_HTTP_PORT/<script>"
  ```
 
  ```shell  
  <html><head><title>Request Rejected</title></head><body>
  ```

1. To check the security logs in the syslog pod:

  ```shell
  kubectl exec -it <SYSLOG_POD> -- cat /var/log/messages
  ```

### Configuration Example of Virtual Server

Refer to GitHub repo for [Virtual Server example](https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.6.0/examples/custom-resources/app-protect-waf-v5/webapp.yaml).

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

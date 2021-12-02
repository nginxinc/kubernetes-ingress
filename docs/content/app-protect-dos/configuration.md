---
title: Configuration

description:
weight: 1900
doctypes: [""]
toc: true
---

This document describes how to configure the NGINX App Protect Dos module
> Check out the complete [NGINX Ingress Controller with App Protect Dos example resources on GitHub](https://github.com/nginxinc/kubernetes-ingress/tree/v1.12.0/examples/appprotect-dos).

## Global Configuration

The NGINX Ingress Controller has a set of global configuration parameters that align with those available in the NGINX App Protect Dos module. See [ConfigMap keys](/nginx-ingress-controller/configuration/global-configuration/configmap-resource/#modules) for the complete list. The App Protect parameters use the `app-protect-dos*` prefix.

## Enable App Protect Dos for Ingress

You can enable and configure NGINX App Protect Dos on a per-Ingress-resource basis. To do so, you can apply the [App Protect Dos annotation](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations/#app-protect-dos) to each desired resource.

## App Protect Dos Protected Resources

An `DosProtectedResource` is a [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) that holds the configuration of a collection of protected resources. 
An Ingress or VirtualServer can be protected by adding a reference to the Dos Protected Resource.

To enable DOS protection to an Ingress:

1. Create an `DosProtectedResource` Custom resource manifest. As an example:
  ```yaml
   apiVersion: appprotectdos.f5.com/v1beta1
   kind: DosProtectedResource
   metadata:
      name: dos-protected
   spec:
      enable: true
      name: "my-dos"
      apDosMonitor: 
         uri: "webapp.example.com"
  ```
2. Add an annotation to an Ingress that refers to that resource by `namespace/name`:
  ```yaml
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
      name: webapp-ingress
      annotations:
         appprotectdos.f5.com/app-protect-dos-resource: "default/dos-protected"
  ```
## Dos Policy configuration

You can set the App Protect Dos Policy configurations by creating an `APDosPolicy` [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) and referencing that in the `DosProtectedResource`.


For example, say you want to use Dos Policy as shown below:

  ```json
  {
   mitigation_mode: "standard",
   signatures: "on",
   bad_actors: "on",
   automation_tools_detection: "on",
   tls_fingerprint: "on",
}
  ```

You would create an `APDosPolicy` resource with the policy defined in the `spec`, as shown below:

  ```yaml
   apiVersion: appprotectdos.f5.com/v1beta1
   kind: APDosPolicy
   metadata:
      name: dospolicy
   spec:
      mitigation_mode: "standard"
      signatures: "on"
      bad_actors: "on"
      automation_tools_detection: "on"
      tls_fingerprint: "on"
  ```

Then add a reference in the `DosProtectedResrouce` to the `ApDosPolicy`:
  ```yaml
   apiVersion: appprotectdos.f5.com/v1beta1
   kind: DosProtectedResource
   metadata:
      name: dos-protected
   spec:
      enable: true
      name: "my-dos"
      apDosMonitor: 
         uri: "webapp.example.com"
      apDosPolicy: "default/dospolicy"
  ```

> Notice how the fields match exactly in name and level. The Ingress Controller will transform the YAML into a valid JSON App Protect Dos policy config.

> **Note**: The relationship between the Policy JSON and the resource spec is 1:1. If you're defining your resources in YAML, as we do in our examples, you'll need to represent the policy as YAML. The fields must match those in the source JSON exactly in name and level.


## App Protect Dos Logs

You can set the [App Protect Dos Log configurations](/nginx-app-protect-dos/logs-overview/types-of-logs/) by creating an `APDosLogConf` [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

To add the App Protect Dos log configurations to an Ingress resource:

1. Create an `APDosLogConf` Custom resource manifest.
2. Add the desired log configuration to the `spec` field in the `APDosLogConf` resource.

   > **Note**: The fields from the JSON must be presented in the YAML *exactly* the same, in name and level. The Ingress Controller will transform the YAML into a valid JSON App Protect Dos log config.

For example, say you want to log state changing requests for your Ingress resources using App Protect Dos. The App Protect Dos log configuration looks like this:

```json
{
    "filter": {
        "request_type": "all"
    },
    "content": {
        "format": "default",
        "max_request_size": "any",
        "max_message_size": "64k"
    }
}
```

You would add define that config in the `spec` of your `APDosLogConf` resource as follows:

```yaml
apiVersion: appprotectdos.f5.com/v1beta1
kind: APDosLogConf
metadata:
   name: doslogconf
spec:
   content:
      format: splunk
      max_message_size: 64k
   filter:
      traffic-mitigation-stats: all
      bad-actors: top 10
      attack-signatures: top 10
```

Then add a reference in the `DosProtectedResrouce` to the `APDosLogConf`:
  ```yaml
   apiVersion: appprotectdos.f5.com/v1beta1
   kind: DosProtectedResource
   metadata:
      name: dos-protected
   spec:
      enable: true
      name: "my-dos"
      apDosMonitor: 
         uri: "webapp.example.com"
      dosSecurityLog:
         enable: true
         apDosLogConf: "doslogconf"
         dosLogDest: "syslog-svc.default.svc.cluster.local:514"
  ```
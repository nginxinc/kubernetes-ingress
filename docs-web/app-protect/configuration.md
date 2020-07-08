# Configuration
This document describes how to configure the NGINX App Protect module
> Check out the complete [NGINX Ingress Controller with App Protect example resources on GitHub](https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/appprotect).

## Global Configuration

The NGINX Ingress Controller has a set of global configuration parameters that align with those available in the NGINX App Protect module. See [ConfigMap keys](/nginx-ingress-controller/configuration/global-configuration/configmap-resource/#modules) for the complete list. The App Protect parameters use the `app-protect*` prefix.

> Check out the complete [NGINX Ingress Controller with App Protect example resources on GitHub](https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/appprotect).

## Enable App Protect for an Ingress Resource

You can enable and configure NGINX App Protect on a per-Ingress-resource basis. To do so, you can apply the [App Protect annotations](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations/#app-protect) to each desired resource.

## App Protect Policies

You can define App Protect policies for your Ingress resources by creating an `APPolicy` [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

To add any [App Protect policy](/nginx-app-protect/policy/#policy) to an Ingress resource:

1. Create an `APPolicy` Custom resource manifest. 
2. Add the desired policy to the `spec` field in the `APPolicy` resource. 
   
   > **Note**: The relationship between the Policy JSON and the resource spec is 1:1. If you're defining your resources in YAML, as we do in our examples, you'll need to represent the policy as YAML. The fields must match those in the source JSON exactly in name and level. 

  For example, say you want to use the [DataGuard policy](/nginx-app-protect/policy/#data-guard) shown below:

  ```json
  {
      "policy": {
          "name": "dataguard_blocking",
          "template": { "name": "POLICY_TEMPLATE_NGINX_BASE" },
          "applicationLanguage": "utf-8",
          "enforcementMode": "blocking",
          "blocking-settings": {
              "violations": [
                  {
                      "name": "VIOL_DATA_GUARD",
                      "alarm": true,
                      "block": true
                  }
              ]
          },
          "data-guard": {
              "enabled": true,
              "maskData": true,
              "creditCardNumbers": true,
              "usSocialSecurityNumbers": true,
              "enforcementMode": "ignore-urls-in-list",
              "enforcementUrls": []            
          }
      }
  }
  ```

  You would create an `APPolicy` resource with the policy defined in the `spec`, as shown below:

  ```yaml
  apiVersion: appprotect.f5.com/v1beta1
  kind: APPolicy
  metadata: 
    name: dataguard-blocking
  spec:
    policy:
      name: dataguard_blocking
      template: 
        name: POLICY_TEMPLATE_NGINX_BASE
      applicationLanguage: utf-8
      enforcementMode: blocking 
      blocking-settings:
        violations:
        - name: VIOL_DATA_GUARD
          alarm: true
          block: true
      data-guard:
        enabled: true
        maskData: true
        creditCardNumbers: true
        usSocialSecurityNumbers: true
        enforcementMode: ignore-urls-in-list
        enforcementUrls: []
  ```

  > Notice how the fields match exactly in name and level. The Ingress Controller will transform the YAML into a valid JSON App Protect policy config.

## App Protect Logs

You can set the [App Protect Log configurations](/nginx-app-protect/nginx-app-protect/troubleshooting/#app-protect-logging-overview) by creating an `APLogConf` [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

To add the [App Protect log configurations](/nginx-app-protect/policy/#policy) to an Ingress resource:

1. Create an `APLogConf` Custom resource manifest. 
2. Add the desired log configuration to the `spec` field in the `APLogConf` resource. 
   
   > **Note**: The fields from the JSON must be presented in the YAML *exactly* the same, in name and level. The Ingress Controller will transform the YAML into a valid JSON App Protect log config.

For example, say you want to [log state changing requests](nginx-app-protect/troubleshooting/#log-state-changing-requests) for your Ingress resources using App Protect. The App Protect log configuration looks like this:

```json
{
    "filter": {
        "request_type": "all"
    },
    "content": {
        "format": "default",
        "max_request_size": "any",
        "max_message_size": "5k"
    }
}
```

You would add define that config in the `spec` of your `APLogConf` resource as follows:

```yaml
apiVersion: appprotect.f5.com/v1beta1
kind: APLogConf
metadata: 
  name: logconf
spec:
  filter: 
    request_types: all
  content: 
    format: default
    max_request_size: any
    max_message_size: 5k
```

---
title: Troubleshooting with NGINX App Protect
description: 
weight: 2000
doctypes: [""]
aliases:
    - /app-protect/troubleshooting/
toc: true
---

This document describes how to troubleshoot problems with the Ingress Controller with the [App Protect](/nginx-app-protect/) module enabled.

For general troubleshooting of the Ingress Controller, check the general [troubleshooting](/nginx-ingress-controller/troubleshooting/) documentation.

For additional troubleshooting of the App Protect module itself, check the [troubleshooting](/nginx-app-protect/troubleshooting/) guide in the App Protect module documentation. 

## Potential Problems

The table below categorizes some potential problems with the Ingress Controller when App Protect module is enabled. It suggests how to troubleshoot those problems, using one or more methods from the next section.

{{% table %}} 
|Problem area | Symptom | Troubleshooting method | Common cause | 
| ---| ---| ---| --- | 
|Start. | The Ingress Controller fails to start. | Check the logs. | Misconfigured APLogConf or APPolicy. | 
|APLogConf, APPolicy or Ingress Resource. | The configuration is not applied. | Check the events of the APLogConf, APPolicy and Ingress Resource, check the logs, replace the policy. | APLogConf or APPolicy is invalid. | 
|NGINX. | The Ingress Controller NGINX verification timeouts while starting for the first time or while reloading after a change. | Check the logs for ``Unable to fetch version: X`` message. Check the Availability of APPolicy External References. | Too many Ingress Resources with App Protect enabled. Check the `NGINX fails to start/reload section <#nginx-fails-to-start-or-reload>`_ of the Known Issues. | 
{{% /table %}} 

## Troubleshooting Methods

### Check the Ingress Controller and App Protect logs

App Protect logs are part of the Ingress Controller logs when the module is enabled. To check the Ingress Controller logs, follow the steps of [Checking the Ingress Controller Logs](/troubleshooting/troubleshoot-ingress-controller/#checking-the-ingress-controller-logs) of the Troubleshooting guide.

For App Protect specific logs, look for messages starting with `APP_PROTECT`, for example:
```
2020/07/10 11:13:20 [notice] 17#17: APP_PROTECT { "event": "configuration_load_success", "software_version": "2.52.1", "completed_successfully":true,"attack_signatures_package":{"revision_datetime":"2020-06-18T10:11:32Z","version":"2020.06.18"}}
```

### Check events of an Ingress Resource

Follow the steps of [Checking the Events of an Ingress Resource](/troubleshooting/troubleshoot-ingress-controller/#checking-the-events-of-an-ingress-resource).

### Check events of APLogConf

After you create or update an APLogConf, you can immediately check if the NGINX configuration was successfully applied by NGINX:
```
$ kubectl describe aplogconf logconf
Name:         logconf
Namespace:    default
. . . 
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  11s   nginx-ingress-controller  AppProtectLogConfig  default/logconf was added or updated
```
Note that in the events section, we have a `Normal` event with the `AddedOrUpdated` reason, which informs us that the configuration was successfully applied.

### Check events of APPolicy

After you create or update an APPolicy, you can immediately check if the NGINX configuration was successfully applied by NGINX:
```
$ kubectl describe appolicy dataguard-alarm
Name:         dataguard-alarm
Namespace:    default
. . . 
Events:
  Type    Reason          Age    From                      Message
  ----    ------          ----   ----                      -------
  Normal  AddedOrUpdated  2m25s  nginx-ingress-controller  AppProtectPolicy default/dataguard-alarm was added or updated
```
Note that in the events section, we have a `Normal` event with the `AddedOrUpdated` reason, which informs us that the configuration was successfully applied.

### Replace the policy

NOTE: This method only applies if using [external references](https://docs.nginx.com/nginx-app-protect/configuration/#external-references) 
If items on the external reference change but the spec of the APPolicy remains unchanged (even when re-applying the policy), kubernetes will not detect the update.
In this case you can force-replace the resource. This will remove the resource and add it again, triggering a reload. For example:

```
kubectl replace appolicy -f your-policy-manifest.yaml --force
```

### Check the Availability of APPolicy External References.

NOTE: This method only applies if you're using [external references](https://docs.nginx.com/nginx-app-protect/configuration/#external-references) in NGINX App Protect policies.

To check what servers host the external references of a policy: 
```
kubectl get appolicy mypolicy -o jsonpath='{.items[*].spec.policy.*.link}' | tr ' ' '\n'

http://192.168.100.100/resources/headersettings.txt
```

You can check the total time a http request takes, in multiple ways eg. using curl:
```
curl -w '%{time_total}' http://192.168.100.100/resources/headersettings.txt
```

## Run App Protect in Debug Mode

When you set the Ingress Controller to use debug mode, the setting also applies to the App Protect module.  See  [Running NGINX in the Debug Mode](/troubleshooting/troubleshoot-ingress-controller/#running-nginx-in-the-debug-mode) for instructions.

## Known Issues

When using the Ingress Controller with the App Protect module, the following issues have been reported. The occurrence of these issues is commonly related to a higher number of Ingress Resources with App Protect being enabled in a cluster.

When you make a change that requires NGINX to apply a new configuration, the Ingress Controller reloads NGINX automatically. Without the App Protect module enabled, usual reload times are around 150ms. If App Protect module is enabled and is being used by any number of Ingress Resources, these reloads might take a few seconds instead. 

### NGINX Configuration Skew

If you are running more than one instance of the Ingress Controller, the extended reload time may cause the NGINX configuration of your instances to be out of sync. This can occur because there is no order imposed on how the Ingress Controller processes the Kubernetes Resources. The configurations will be the same after all instances have completed the reload.

In order to reduce these inconsistencies, we advise that you do not apply changes to multiple resources handled by the Ingress Controller at the same time. 

### NGINX Fails to Start or Reload

The first time the Ingress Controller starts, or whenever there is a change that requires reloading NGINX, the Ingress Controller will verify if the reload was successful. The timeout for this verification is normally 4 seconds. When App Protect is enabled, this timeout is 20 seconds. 

This timeout should be more than enough to verify configurations. However, when numerous Ingress Resources with App Protect enabled are handled by the Ingress Controller at the same time, you may find that you need to extend the timeout further.  Examples of when this might be necessary include:

- You need to apply a large amount of Ingress Resources at once.
- You are running the Ingress Controller for the first time in a cluster where the Ingress Resources with App Protect enabled are already present.

You can increase this timeout by setting the `nginx-reload-timeout` [cli-argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#cmdoption-nginx-reload-timeout).

When using the User Defined Signature feature, an update to an `APUserSig` requires more reload time from NGINX Plus compared with the other AppProtect resources. As a consequence, we recommend increasing the `nginx-reload-timeout` to 30 seconds if you're planning to use this feature.

If you are using external references in your Nginx App Protect policies, verify if the servers hosting the referenced resources are available and that their response time is as short as possible (see the Check the Availability of APPolicy External References section). If the references are not available during the Ingress Controller startup, the pod will fail to start. In case the resources are not available during a reload, the reload will fail, and NGINX Plus will use the previous correct configuration.

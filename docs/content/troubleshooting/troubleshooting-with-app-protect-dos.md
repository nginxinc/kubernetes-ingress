---
title: Troubleshooting with NGINX App Protect Dos
description: "This document describes how to troubleshoot problems with the Ingress Controller with the App Protect Dos module enabled." 
weight: 2000
doctypes: [""]
aliases:
- /app-protect/troubleshooting/
toc: true
docs: "DOCS-620"
---

This document describes how to troubleshoot problems with the Ingress Controller with the App Protect Dos module enabled.

For general troubleshooting of the Ingress Controller, check the general [troubleshooting](/nginx-ingress-controller/troubleshooting/) documentation.

## Potential Problems

The table below categorizes some potential problems with the Ingress Controller when App Protect Dos module is enabled. It suggests how to troubleshoot those problems, using one or more methods from the next section.

{{% table %}}
|Problem area | Symptom | Troubleshooting method | Common cause |
| ---| ---| ---| --- |
|Start | The Ingress Controller fails to start. | Check the Ingress Controller logs. | Misconfigured DosProtectedResource, APDosLogConf or APDosPolicy. |
|DosProtectedResource, APDosLogConf, APDosPolicy or Ingress Resource. | The configuration is not applied. | Check the events of the DosProtectedResource, APDosLogConf, APDosPolicy and Ingress Resource, check the Ingress Controller logs. | DosProtectedResource, APDosLogConf or APDosPolicy is invalid. |
{{% /table %}}

## Troubleshooting Methods

### Check the Ingress Controller and App Protect Dos logs

App Protect Dos logs are part of the Ingress Controller logs when the module is enabled. To check the Ingress Controller logs, follow the steps of [Checking the Ingress Controller Logs](/nginx-ingress-controller/troubleshooting/#checking-the-ingress-controller-logs) of the Troubleshooting guide.

For App Protect Dos specific logs, look for messages starting with `APP_PROTECT_DOS`, for example:
```
2021/06/14 08:17:50 [notice] 242#242: APP_PROTECT_DOS { "event": "shared_memory_connected", "worker_pid": 242, "mode": "operational", "mode_changed": true }
```

### Check events of an Ingress Resource

Follow the steps of [Checking the Events of an Ingress Resource]({{< relref "troubleshooting/troubleshoot-ingress-controller.md#checking-the-events-of-an-ingress-resource" >}}).

### Check events of a VirtualServer Resource

Follow the steps of [Checking the Events of a VirtualServer Resource]({{< relref "troubleshooting/troubleshoot-ingress-controller.md#checking-the-events-of-a-virtualserver-and-virtualserverroute-resources" >}}).

### Check events of DosProtectedResource

After you create or update an DosProtectedResource, you can immediately check if the NGINX configuration was successfully applied by NGINX:
```
$ kubectl describe dosprotectedresource dos-protected
Name:         dos-protected
Namespace:    default
. . . 
Events:
  Type     Reason          Age   From                      Message
  ----     ------          ----  ----                      -------
  Normal   AddedOrUpdated  2s    nginx-ingress-controller  Configuration for default/dos-protected was added or updated
```
Note that in the events section, we have a `Normal` event with the `AddedOrUpdated` reason, which informs us that the configuration was successfully applied.

If the DosProtectedResource refers to a missing resource, you should see a message like the following:
```
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Warning  Rejected  8s    nginx-ingress-controller  dos protected refers (default/dospolicy) to an invalid DosPolicy: DosPolicy default/dospolicy not found
```
This can be fixed by adding the missing resource.

### Check events of APDosLogConf

After you create or update an APDosLogConf, you can immediately check if the NGINX configuration was successfully applied by NGINX:
```
$ kubectl describe apdoslogconf logconf
Name:         logconf
Namespace:    default
. . . 
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  11s   nginx-ingress-controller  AppProtectDosLogConfig  default/logconf was added or updated
```
Note that in the events section, we have a `Normal` event with the `AddedOrUpdated` reason, which informs us that the configuration was successfully applied.

### Check events of APDosPolicy

After you create or update an APDosPolicy, you can immediately check if the NGINX configuration was successfully applied by NGINX:
```
$ kubectl describe apdospolicy dospolicy
Name:         dospolicy
Namespace:    default
. . . 
Events:
  Type    Reason          Age    From                      Message
  ----    ------          ----   ----                      -------
  Normal  AddedOrUpdated  2m25s  nginx-ingress-controller  AppProtectDosPolicy default/dospolicy was added or updated
```
Note that in the events section, we have a `Normal` event with the `AddedOrUpdated` reason, which informs us that the configuration was successfully applied.

## Run App Protect Dos in Debug log Mode

When you set the Ingress Controller to use debug log mode, the setting also applies to the App Protect Dos module.  See  [Running NGINX in the Debug Mode](/nginx-ingress-controller/troubleshooting/#running-nginx-in-the-debug-mode) for instructions.

You can enable debug log mode to App Protect Dos module only by setting the `app-protect-dos-debug` [configmap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource#modules).

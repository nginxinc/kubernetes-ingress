---
title: Troubleshooting with NGINX App Protect Dos
description:
weight: 2000
doctypes: [""]
aliases:
- /app-protect/troubleshooting/
toc: true
---

This document describes how to troubleshoot problems with the Ingress Controller with the App Protect Dos module enabled.

For general troubleshooting of the Ingress Controller, check the general [troubleshooting](/nginx-ingress-controller/troubleshooting/) documentation.

## Potential Problems

The table below categorizes some potential problems with the Ingress Controller when App Protect module is enabled. It suggests how to troubleshoot those problems, using one or more methods from the next section.

```eval_rst
.. list-table::
   :header-rows: 1

   * - Problem area
     - Symptom
     - Troubleshooting method
     - Common cause
   * - Start.
     - The Ingress Controller fails to start.
     - Check the logs.
     - Misconfigured APDosLogConf or APDosPolicy.
   * - APDosLogConf, APDosPolicy or Ingress Resource.
     - The configuration is not applied.
     - Check the events of the APDosLogConf, APDosPolicy and Ingress Resource, check the logs, replace the policy.
     - APDosLogConf or APDosPolicy is invalid.
   * - NGINX.
     - The Ingress Controller NGINX verification timeouts while starting for the first time or while reloading after a change.
     - Check the logs for ``Unable to fetch version: X`` message. Check the Availability of APDosPolicy External References.
     - Too many Ingress Resources with App Protect Dos enabled. Check the `NGINX fails to start/reload section <#nginx-fails-to-start-or-reload>`_ of the Known Issues.
```

## Troubleshooting Methods

### Check the Ingress Controller and App Protect Dos logs

App Protect Dos logs are part of the Ingress Controller logs when the module is enabled. To check the Ingress Controller logs, follow the steps of [Checking the Ingress Controller Logs](/nginx-ingress-controller/troubleshooting/#checking-the-ingress-controller-logs) of the Troubleshooting guide.

For App Protect Dos specific logs, look for messages starting with `APP_PROTECT_DOS`, for example:
```
2021/06/14 08:17:50 [notice] 242#242: APP_PROTECT_DOS { "event": "shared_memory_connected", "worker_pid": 242, "mode": "operational", "mode_changed": true }
```

### Check events of an Ingress Resource

Follow the steps of [Checking the Events of an Ingress Resource](/troubleshooting/#checking-the-events-of-an-ingress-resource).

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

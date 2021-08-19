---
title: "Troubleshooting Common Issues"
date: 2021-07-13T21:01:29-06:00
draft: true
description: ""
# Assign weights in increments of 100
weight: 100
draft: false
toc: true
tags: [ "docs" ]
# Taxonomies
# These are pre-populated with all available terms for your convenience.
# Remove all terms that do not apply.
doctypes: ["troubleshooting"]
---

## Overview

This document describes how to troubleshoot problems with the Ingress Controller.

## Potential Problems

The table below categorizes some potential problems with the Ingress Controller you may encounter and suggests how to troubleshoot those problems using one or more methods from the next section.

{{% table %}} 
| Problem Area | Symptom | Troubleshooting Method | Common Cause |
|-----|-----|-----|-----|
| Start | The Ingress Controller fails to start. | Check the logs. | Misconfigured RBAC, a missing default server TLS Secret.|
| Ingress Resource and Annotations | The configuration is not applied | Check the events of the Ingress resource, check the logs, check the generated config. | Invalid values of annotations. |
| VirtualServer and VirtualServerRoute Resources | The configuration is not applied. | Check the events of the VirtualServer and VirtualServerRoutes, check the logs, check the generated config. | VirtualServer or VirtualServerRoute is invalid. | 
| Policy Resource | The configuration is not applied. | Check the events of the Policy resource as well as the events of the VirtualServers that reference that policy, check the logs, check the generated config. | Policy is invalid. |
| ConfigMap Keys | The configuration is not applied. | Check the events of the ConfigMap, check the logs, check the generated config.  | Invalid values of ConfigMap keys. |
| NGINX | NGINX responds with unexpected responses. | Check the logs, check the generated config, check the live activity dashboard (NGINX Plus only), run NGINX in the debug mode. | Unhealthy backend pods, a misconfigured backend service. |
{{% /table %}} 

## Troubleshooting Methods

Note that the commands in the next sections make the following assumptions:
* The Ingress Controller is deployed in the namespace `nginx-ingress`.
* `<nginx-ingress-pod>` is the name of one of the Ingress Controller pods.

### Checking the Ingress Controller Logs

To check the Ingress Controller logs -- both of the Ingress Controller software and the NGINX access and error logs -- run:
```
$ kubectl logs <nginx-ingress-pod> -n nginx-ingress
```

Controlling the verbosity and format:
* To control the verbosity of the Ingress Controller software logs (from 1 to 4), use the `-v` [command-line argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments). For example, with `-v=3` you will get more information and the content of any new or updated configuration file will be printed in the logs.
* To control the verbosity and the format of the NGINX logs, configure the corresponding [ConfigMap keys](/nginx-ingress-controller/configuration/global-configuration/configmap-resource).

### Checking the Events of an Ingress Resource

After you create or update an Ingress resource, you can immediately check if the NGINX configuration for that Ingress resource was successfully applied by NGINX:
```
$ kubectl describe ing cafe-ingress
Name:             cafe-ingress
Namespace:        default
. . .
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  12s   nginx-ingress-controller  Configuration for default/cafe-ingress was added or updated
```
Note that in the events section, we have a `Normal` event with the `AddedOrUpdated` reason, which informs us that the configuration was successfully applied.

### Checking the Events of a VirtualServer and VirtualServerRoute Resources

After you create or update a VirtualServer resource, you can immediately check if the NGINX configuration for that  resource was successfully applied by NGINX:
```
$ kubectl describe vs cafe
. . .
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  16s   nginx-ingress-controller  Configuration for default/cafe was added or updated
```
Note that in the events section, we have a `Normal` event with the `AddedOrUpdated` reason, which informs us that the configuration was successfully applied.

Checking the events of a VirtualServerRoute is similar:
```
$ kubectl describe vsr coffee 
. . .
Events:
  Type     Reason                 Age   From                      Message
  ----     ------                 ----  ----                      -------
  Normal   AddedOrUpdated         1m    nginx-ingress-controller  Configuration for default/coffee was added or updated
```

### Checking the Events of a Policy Resource

After you create or update a Policy resource, you can use `kubectl describe` to check whether or not the Ingress Controller accepted the Policy:
```
$ kubectl describe pol webapp-policy
. . .
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  11s   nginx-ingress-controller  Policy default/webapp-policy was added or updated
```
Note that in the events section, we have a `Normal` event with the `AddedOrUpdated` reason, which informs us that the policy was successfully accepted.

However, the fact that a policy was accepted doesn't guarantee that the NGINX configuration was successfully applied. To confirm that, check the events of the VirtualServer and VirtualServerRoute resources that reference that policy.

### Checking the Events of the ConfigMap Resource

After you update the [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource) resource, you can immediately check if the configuration was successfully applied by NGINX:
```
$ kubectl describe configmap nginx-config -n nginx-ingress
Name:         nginx-config
Namespace:    nginx-ingress
Labels:       <none>
. . .
Events:
  Type    Reason   Age                From                      Message
  ----    ------   ----               ----                      -------
  Normal  Updated  11s (x2 over 26m)  nginx-ingress-controller  Configuration from nginx-ingress/nginx-config was updated
```
Note that in the events section, we have a `Normal` event with the `Updated` reason, which informs us that the configuration was successfully applied.

### Checking the Generated Config

For each Ingress/VirtualServer resource, the Ingress Controller generates a corresponding NGINX configuration file in the `/etc/nginx/conf.d` folder. Additionally, the Ingress Controller generates the main configuration file `/etc/nginx/nginx.conf`, which includes all the configurations files from `/etc/nginx/conf.d`. The config of a VirtualServerRoute resource is located in the configuration file of the VirtualServer that references the resource.

You can view the content of the main configuration file by running:
```
$ kubectl exec <nginx-ingress-pod> -n nginx-ingress -- cat /etc/nginx/nginx.conf
```

Similarly, you can view the content of any generated configuration file in the `/etc/nginx/conf.d` folder. 

You can also print all NGINX configuration files together:
```
$ kubectl exec <nginx-ingress-pod> -n nginx-ingress -- nginx -T
```
However, this command will fail if any of the configuration files is not valid.

### Checking the Live Activity Monitoring Dashboard

The live activity monitoring dashboard shows the real-time information about NGINX Plus and the applications it is load balancing, which is helpful for troubleshooting. To access the dashboard, follow the steps from [here](/nginx-ingress-controller/logging-and-monitoring/status-page).

### Running NGINX in the Debug Mode

Running NGINX in the [debug mode](https://docs.nginx.com/nginx/admin-guide/monitoring/debugging/) allows us to enable its debug logs, which can help to troubleshoot problems in NGINX. Note that it is highly unlikely that a problem you encounter with the Ingress Controller is caused by a bug in the NGINX code, but it is rather caused by NGINX misconfiguration. Thus, this method is rarely needed.

To enable the debug mode, set the `error-log-level` to `debug` in the [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource) and use the `-nginx-debug` [command-line argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments) when running the Ingress Controller.

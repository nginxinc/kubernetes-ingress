---
title: Performance

description: Performance considerations for using App Protect DOS with Ingress Controller. 
weight: 1900
doctypes: [""]
toc: true
---

## Performance Baselines
This document describes performance considerations and assumptions when using NGINX App Protect Dos module

### Ingress with App Protect Dos

Using the [example configuration](https://github.com/nginxinc/kubernetes-ingress/tree/v2.0.3/examples/appprotect-dos), the number of resources were scaled up. 
A resource group was defined as: 

 * 1 IngressController pod with Dos enabled
 * 1 Ingress resource
 * 1 webapp backend

With each Ingress referencing the same, single dos protected resource.

For ten of these resource groups, it took around 11 seconds to start all the pods.

### VirtualServer with App Protect Dos

Using the [example configuration](https://github.com/nginxinc/kubernetes-ingress/tree/v2.0.3/examples/custom-resources/dos), the number of resources were scaled up.
A resource group was defined as:

* 1 IngressController pod with Dos enabled
* 1 VirtualServer resource
* 1 webapp backend

With each VirtualServer referencing the same, single dos protected resource.

For ten of these resource groups, it took around 12 seconds to start all the pods.


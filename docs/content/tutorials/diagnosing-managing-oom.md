---
title: "Diagnosing and Managing Out of Memory with NGINX Ingress Controller"
date: 2022-03-03
draft: true
description: "Understanding and avoiding OOM with NGINX Ingress Controller"
# Assign weights in increments of 100
weight: 
draft: true
toc: true
tags: [ "docs" ]
# Taxonomies
# These are pre-populated with all available terms for your convenience.
# Remove all terms that do not apply.
categories: ["diagnostics"]
doctypes: ["tutorial"]
versions: []
authors: []
---

An Ingress Controller is one of the critical components of a Kubernetes cluster: it acts as a gateway for the traffic destined for the applications running in the cluster. As a result, the availability and performance of your applications directly depends on the availability and performance of the Ingress Controller.

For each workload, Kubernetes supports configuring its [resource requests and limits][https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/] for CPU and memory. This allows Kubernetes to schedule a workload to a node with enough available resources and prevent the workload from consuming excessive resources, which in turns ensures that the other workloads on the node have enough resources. However, configuring a low memory limit can lead to the container being terminated with an out of memory (OOM) error, which can lead to an outage.

In this tutorial, we will:

1. Show how to the configure memory limit for the NGINX Ingress Controller.
2. Explain how the NGINX Ingress Controller uses memory, so that you are better equipped at configuring the memory limit for it.
3. Give tips on how to prevent OOM errors.

Configuring Memory Limits

Configuring the memory limit helps prevent a container from:

* Consuming available memory of the node and, as a result, affecting the other containers on the node and the stability of the node in extreme cases. For example, this can happen if the application has a memory leak. 
* Being killed with an OOM error: if a node is under memory pressure, containers without specified memory limits [will be killed first][https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/resource-qos.md#qos-classes].

If you use helm to install the NGINX Ingress Controller, you can configure the memory limit for the Ingress Controller via the controller resources parameter. For example:

```yaml
controller:
  resources:
    limits:
      memory: 128Mi
```

For the manifests installation method, you can add the limit to Ingress Controller container specification:
containers:

```yaml
- image: nginx/nginx-ingress:1.12.1
  name: nginx-ingress
  resources:
    limits:
      memory: 128Mi
```

While configuring the memory limit is easy, choosing the value can be challenging. A low value can lead to the Ingress Controller pod being terminated with the OOMKilled reason:

```text
$ kubectl describe pod -n nginx-ingress my-release-nginx-ingress-5bfc87ccbd-zdmwc
. . .
    State:          Waiting
      Reason:       CrashLoopBackOff
    Last State:     Terminated
      Reason:       OOMKilled
      Exit Code:    137
```

NGINX Ingress Controller Memory Usage

NGINX Ingress Controller pod consists of a single container which includes two parts:

* Control-plane: Ingress Controller process, which configures the data plane according to configuration rules, primarily Ingress resources.
* Data-plane: NGINX master process, which manages multiple NGINX worker processes that handle client traffic.

Note: you can read more about the architecture of the Ingress Controller in the How NGINX Ingress Controller Works article.
We will talk about the memory usage of those two parts in isolation and in combination.

NGINX is designed for high performance, conservative usage of memory, doesn’t create unnecessary threads or processes.  You can read more here: 
https://www.nginx.com/blog/inside-nginx-how-we-designed-for-performance-scale/ 

Memory Utilization

There are two main items to consider:

Configuration:

* The amount of configuration / number of configured resources
* Complexity of configuration - the type of work nginx is being asked to do.

Traffic
• traffic
• certain attacks

Ingress Controller
• cache
• features

Combined Ingress Controller with NGINX
• reloads

Tips for Preventing OOMs

1. Configure (determine) correctly
   1. using normal and peak
   2. take in mind NGINX reloads
   3. long-living connections
2. Reduce memory
   1. Horizontal scaling and auto scaling
   2. Separate Ingress Controller
   3. Single namespace
   4. Separate applications into a separate Ingress Controller
3. Monitor
   1. Monitoring system
   2. Monitor worker processes (old). reload errors., readiness probes … 

---
title: How NGINX Ingress Controller Works
description: "This document explains how NGINX Ingress Controller works."
weight: 300
doctypes: [""]
toc: true
docs: "DOCS-609"
---


This document explains how NGINX Ingress Controller works. The target audience includes the following two main groups:

* *Operators* who would like to know how the software works and also better understand how it can fail.
* *Developers* who would like to [contribute](https://github.com/nginxinc/kubernetes-ingress/blob/main/CONTRIBUTING.md) to the project.

We assume that the reader is familiar with core Kubernetes concepts, such as Pod, Deployment, Service, and Endpoints. Additionally, we recommend reading [this blog post](https://www.nginx.com/blog/inside-nginx-how-we-designed-for-performance-scale/) for an overview of the NGINX architecture.

## What is an Ingress Controller?

An Ingress Controller is a component in a Kubernetes cluster that configures an HTTP load balancer according to Ingress resources created by the cluster user.

> If you’d like to read more about the Ingress resource, refer to [the official Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/).

This document is specific to NGINX Ingress Controller, referred to as *Ingress Controller* or *IC*, which is built upon NGINX and NGINX Plus capabilities

## The Ingress Controller at a High Level

Let’s begin with a high-level examination of the Ingress Controller (IC). The following figure depicts an example of how the IC exposes two web applications running in a Kubernetes cluster to clients on the internet:

{{< img title="IC at a high level" src="./img/ic-high-level.png" >}}

The figure shows:

* A *Kubernetes cluster*.
* Cluster users *Admin*, *User A* and *User B*, which use the cluster via the *Kubernetes API*.
* *Clients A* and *Clients B*, which connect to the *Applications A* and *B* deployed by the corresponding users.
* *IC*, [deployed by *Admin*](/nginx-ingress-controller/installation/installation-with-manifests) in a pod in the namespace *nginx-ingress* and configured via the *ConfigMap nginx-ingress*. For simplicity, we depict only one IC pod; however, *Admin* typically deploys at least two pods for redundancy. The *IC* uses the *Kubernetes API* to get the latest Ingress resources created in the cluster and then configures *NGINX* according to those resources.
* *Application A* with two pods deployed in the *namespace A* by *User A*. To expose the application to its clients (*Clients A*) via the host `a.example.com`, *User A* creates *Ingress A*.
* *Application B* with one pod deployed in the *namespace B* by *User B*. To expose the application to its clients (*Clients B*) via the host `b.example.com`, *User B* creates *VirtualServer B*.
* *Public Endpoint*, which fronts the *IC* pod(s). This is typically a TCP load balancer (cloud, software, or hardware) or a combination of such load balancer with a NodePort service. *Clients A* and *B* connect to their applications via the *Public Endpoint*.

The yellow and purple arrows represent connections related to the client traffic, and the black arrows represent access to the Kubernetes API.

> For simplicity, many necessary Kubernetes resources like Deployment and Services aren't shown, which Admin and the users also need to create.

Next, let's explore the IC pod.

## The Ingress Controller Pod

The IC pod consists of a single container, which in turn includes the following:

* *IC process*, which configures NGINX according to Ingress and other resources created in the cluster.
* *NGINX master process*, which controls NGINX worker processes.
* *NGINX worker processes*, which handle the client traffic and load balance the traffic to the backend applications.

The following is an architectural diagram that shows how those processes interact together and with some external processes/entities:

{{< img title="IC pod" src="./img/ic-pod.png" >}}

For brevity, we've omitted the suffix process from the description of the processes.

<!-- TO-DO: after the docs are converted to Hugo, convert this list to a markdown table>
The table below describes each connection:
| Connection # | Type | Description |
| --- | --- | --- |
-->

The numbered list that follows describes each connection with its type in curly brackets:

1. (HTTP) *Prometheus* fetches the IC and NGINX metrics via an HTTP endpoint that the *IC* exposes. The default is ``:9113/metrics``. **Note**: *Prometheus* is not required by the IC, the endpoint can be turned off.
2. (HTTPS) The *IC* reads the *Kubernetes API* to get the latest versions of the resources in the cluster and writes to the API to update the handled resources' statuses and emit events.
3. (HTTP) *Kubelet* probes the *IC* readiness probe (the default is `:8081/nginx-ready`) to consider the IC pod [ready](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions).
4. (File I/O) When the *IC* starts, it reads the *configuration templates* necessary for config generation from the filesystem. The templates are located in the `/` directory of the container and have the `.tmpl` extension.
5. (File I/O) The *IC* writes logs to its *stdout* and *stderr*, which are collected by the container runtime.
6. (File I/O) The *IC* generates NGINX *configuration* based on the resources created in the cluster (refer to [The Ingress Controller is a Kubernetes Controller](#the-ingress-controller-is-a-kubernetes-controller) section for the list of resources) and writes it on the filesystem in the `/etc/nginx` folder. The configuration files have a `.conf` extension.
7. (File I/O) The *IC* writes *TLS certificates* and *keys* from any [TLS Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets) referenced in the Ingress and other resources to the filesystem.
8. (HTTP) The *IC* fetches the [NGINX metrics](https://nginx.org/en/docs/http/ngx_http_stub_status_module.html#stub_status) via the `unix:/var/lib/nginx/nginx-status.sock` UNIX socket and converts it to Prometheus format used in #1.
9. (HTTP) To consider a configuration reload a success, the *IC* ensures that at least one *NGINX worker* has the new configuration. To do that, the *IC* checks a particular endpoint via the `unix:/var/lib/nginx/nginx-config-version.sock` UNIX socket.
10. (N/A) To start NGINX, the *IC* runs the `nginx` command, which launches the *NGINX master*.
11. (Signal) To reload NGINX, the *IC* runs the `nginx -s reload` command, which validates the configuration and sends the [reload signal](https://nginx.org/en/docs/control.html) to the *NGINX master*.
12. (Signal) To shutdown NGINX, the *IC* executes `nginx -s quit` command, which sends the graceful shutdown signal to the *NGINX master*.
13. (File I/O) The *NGINX master* sends logs to its *stdout* and *stderr*, which are collected by the container runtime.
14. (File I/O) The *NGINX master* reads the *TLS cert and keys* referenced in the configuration when it starts or reloads.
15. (File I/O) The *NGINX master* reads *configuration files* when it starts or during a reload.
16. (Signal) The *NGINX master* controls the [lifecycle of *NGINX workers*](https://nginx.org/en/docs/control.html#reconfiguration) it creates workers with the new configuration and shutdowns workers with the old configuration.
17. (File I/O) An *NGINX worker* writes logs to its *stdout* and *stderr*, which are collected by the container runtime.
18. (UDP) An *NGINX worker* sends the HTTP upstream server response latency logs via the Syslog protocol over the UNIX socket `/var/lib/nginx/nginx-syslog.sock` to the *IC*. In turn, the *IC* analyzes and transforms the logs into Prometheus metrics.
19. (HTTP,HTTPS,TCP,UDP) A *client* sends traffic to and receives traffic from any of the *NGINX workers* on ports 80 and 443 and any additional ports exposed by the [GlobalConfiguration resource](/nginx-ingress-controller/configuration/global-configuration/globalconfiguration-resource).
20. (HTTP,HTTPS,TCP,UDP) An *NGINX worker* sends traffic to and receives traffic from the *backends*.
21. (HTTP) *Admin* can connect to the [NGINX stub_status](http://nginx.org/en/docs/http/ngx_http_stub_status_module.html#stub_status) using port 8080 via an *NGINX worker*. **Note**: By default, NGINX only allows connections from `localhost`.

### Differences for NGINX Plus

The preceding diagram depicts the IC with NGINX. The IC also supports NGINX Plus with the following important differences:

* To configure NGINX Plus, Ingress Controller uses [configuration reloads](#reloading-nginx) and the [NGINX Plus API](http://nginx.org/en/docs/http/ngx_http_api_module.html#api). The latter allows the Ingress Controller to dynamically change the upstream servers of an upstream.
* Instead of the stub status metrics, the extended metrics are used, which are available via NGINX Plus API.
* In addition to TLS certs and keys, the IC writes JWKs from the secrets of the type `nginx.org/jwk`, and NGINX workers read them.

## The Ingress Controller Process

This section covers the architecture of the IC process, including:

* How the IC processes a new Ingress resource created by a user.
* The summary of how the IC works and how it relates to Kubernetes Controllers.
* The different components of the IC process.

### Processing a New Ingress Resource

The following diagram depicts how the IC processes a new Ingress resource. We represent the NGINX master and worker processes as a single rectangle *NGINX* for simplicity. Also, note that VirtualServer and VirtualServerRoute resources are processed similarly.

{{< img title="IC process" src="./img/ic-process.png" >}}

Processing a new Ingress resource involves the following steps, where each step corresponds to the arrow on the diagram with the same number:

1. *User* creates a new Ingress resource.
2. The IC process has a *Cache* of the resources in the cluster. The *Cache* includes only the resources the IC is interested in, such as Ingresses. The *Cache* stays in sync with the Kubernetes API by [watching for changes to the resources](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes).
3. Once the *Cache* has the new Ingress resource, it notifies the *Control loop* about the changed resource.
4. The *Control loop* gets the latest version of the Ingress resource from the *Cache*. Because the Ingress resource references other resources, such as TLS Secrets, the *Control loop* gets the latest versions of any referenced resources as well.
5. The *Control loop* generates TLS certificates and keys from the TLS Secrets and writes them to the filesystem.
6. The *Control loop* generates and writes the NGINX *configuration files*, which correspond to the Ingress resource, and writes them to the filesystem.
7. The *Control loop* reloads *NGINX* and waits for *NGINX* to successfully reload. As part of the reload:
    1. *NGINX* reads the *TLS certs and keys*.
    1. *NGINX* reads the *configuration files*.
8. The *Control loop* emits an event for the Ingress resource and updates its status. If the reload fails, the event includes the error message.

### The Ingress Controller is a Kubernetes Controller

Based on the example from the previous section, we can generalize how the IC works:

*The IC constantly processes both new resources and changes to the existing resources in the cluster. As a result, the NGINX configuration stays up-to-date with the resources in the cluster.*

The IC is an example of a [Kubernetes controller](https://kubernetes.io/docs/concepts/architecture/controller/): the IC runs a control loop that ensures NGINX is configured according to the desired state (Ingresses and other resources).

The desired state is concentrated in the following built-in Kubernetes resources and Custom Resources (CRs):

* Layer 7 Load balancing configuration:
  * Ingresses
  * VirtualServers (CR)
  * VirtualServerRoutes (CR)
* Layer 7 policies:
  * Policies (CR)
* Layer 4 load balancing configuration:
  * TransportServers (CR)
* Service discovery:
  * Services
  * Endpoints
  * Pods
* Secret configuration:
  * Secrets
* Global Configuration:
  * ConfigMap (only one resource)
  * GlobalConfiguration (CR, only one resource)

The IC can watch additional Custom Resources, which are less common and not enabled by default:

* [NGINX App Protect resources](/nginx-ingress-controller/app-protect/configuration/) (APPolicies, APLogConfs, APUserSigs)
* IngressLink resource (only one resource)

In the next section, we examine the different components of the IC process.

## Ingress Controller Process Components

In this section, we describe the components of the IC process and how they interact, including:

1. How the IC watches for resources changes.
1. The main components of the IC control loop.
1. How those components process a resource change.
1. A few additional components, which are crucial for processing changes.

The IC is written in [go](https://golang.org/) and relies heavily on the [Go client for Kubernetes](https://github.com/kubernetes/client-go). In the sections next, we include links to the code on GitHub when necessary.

### Resource Caches

In the section [Processing a New Ingress Resource](#processing-a-new-ingress-resource), we mentioned that the IC has a cache of the resources in the cluster that stays in sync with the Kubernetes API by watching for changes to the resources. We also mentioned that once cache is updated, it notifies the control loop about the changed resource.

The cache is actually a collection of *informers*. The following diagram shows how changes to resources are processed by the IC.

{{< img title="IC process components" src="./img/ic-process-components.png" >}}

* For every resource type the IC monitors, it creates an [*Informer*](https://pkg.go.dev/k8s.io/client-go@v0.21.0/tools/cache#SharedInformer). The *Informer* includes a *Store* that holds the resources of that type. To keep the *Store* in sync with the latest versions of the resources in the cluster, the *Informer* calls the Watch and List *Kubernetes APIs* for that resource type (see the arrow *1. Watch and List* on the diagram).
* When a change happens in the cluster (for example, a new resource is created), the *Informer* updates its *Store* and invokes [*Handlers*](https://pkg.go.dev/k8s.io/client-go@v0.21.0/tools/cache#ResourceEventHandler) (see the arrow *2. Invoke*) for that *Informer*.
* The IC registers handlers for every *Informer*. Most of the time, a *Handler* creates an entry for the affected resource in the *Workqueue* where a workqueue element includes the type of the resource and its namespace and name. (See the arrow *3. Put*.)
* The *Workqueue* always tries to drain itself: if there is an element at the front, the queue will remove the element and send it to the *Controller* by calling a callback function. (See the arrow *4. Send*.)
* The *Controller* is the primary component in the IC, which represents the control loop. We explain the components in [The Control Loop](#the-control-loop) section. For now, it suffices to know that to process a workqueue element, the *Controller* component gets the latest version of the resource from the *Store* (see the arrow *5. Get*), reconfigures *NGINX* according to the resource (see the arrow *6. Reconfigure*), updates the resource status, and emits an event via the *Kubernetes API* (see the arrow  *7. Update status and emit event*).

### The Control Loop

This section discusses the main components of the IC, which comprise the control loop:

* [Controller](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/controller.go#L90)
  * Runs the IC control loop.
  * Instantiates *Informers*, *Handlers*, the *Workqueue* and additional helper components.
  * Includes the sync method (see the next section), which is called by the *Workqueue* to process a changed resource.
  * Passes changed resources to *Configurator* to re-configure NGINX.
* [Configurator](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/configs/configurator.go#L95)
  * Generates NGINX configuration files, TLS and cert keys, and JWKs based on the Kubernetes resource.
  * Uses *Manager* to write the generated files and reload NGINX.
* [Manager](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/nginx/manager.go#L52)
  * Controls the lifecycle of NGINX (starting, reloading, quitting). See [Reloading NGINX](#reloading-nginx) for more details about reloading.
  * Manages the configuration files, TLS keys and certs, and JWKs.

The following diagram shows how the three components interact:

{{< img src="./img/control-loop.png" title="Control Loop" >}}

#### The Controller Sync Method

The Controller [sync](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/controller.go#L663) method is called by the *Workqueue* to process a change of a resource. The method determines the *kind* of the resource and calls the appropriate *sync* method (for example, *syncIngress* for Ingresses).

Rather than show how all the various sync methods work, we focus on the most important one -- the *syncIngress* method -- and look at how it processes a new Ingress resource, illustrated in the diagram below.

{{< img src="/img/controller-sync.png" title="Controller sync" >}}

1. The *Workqueue* calls the *sync* method and passes a workqueue element to it that includes the changed resource *kind* and *key* (the key is the resource namespace/name like “default/cafe-ingress”).
2. Using the *kind*, the *sync* method calls the appropriate sync method and passes the resource key. For Ingresses, that method is *syncIngress*.
3. *syncIngress* gets the Ingress resource from the *Ingress Store* using the key. The *Store* is controlled by the *Ingress Informer*, as mentioned in the section [Resource Caches](#resource-caches). **Note**: In the code, we use the helper *storeToIngressLister* type that wraps the *Store*.
4. *syncIngress* calls *AddOrUpdateIngress* of the *Configuration*, passing the Ingress along. The [Configuration](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/configuration.go#L320) is a component that represents a valid collection of load balancing configuration resources (Ingresses, VirtualServers, VirtualServerRoutes, TransportServers), ready to be converted to the NGINX configuration (see the [Configuration section](#configuration) for more details). *AddOrUpdateIngress* returns a list of [ResourceChanges](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/configuration.go#L59), which must be reflected in the NGINX config. Typically, for a new Ingress resource, the *Configuration* returns only a single *ResourceChange*.
5. *syncIngress* calls *processChanges*, which processes the single Ingress *ResourceChange*.
    1. *processChanges* creates an extended Ingress resource (*IngressEx*) that includes the original Ingress resource and its dependencies, such as Endpoints and Secrets, to generate the NGINX configuration. For simplicity, we don’t show this step on the diagram.
    2. *processChanges* calls *AddOrUpdateIngress* of the *Configurator* and passes the extended Ingress resource.
6. *Configurator* generates an NGINX configuration file based on the extended Ingress resource and then:
    1. Calls *Manager’s CreateConfig()* to  update the config for the Ingress resource.
    2. Calls *Manager’s Reload()* to reload NGINX.
7. The reload status is propagated from *Manager* to *processChanges*. The status is either a success or a failure with an error message.
8. *processChanges* calls *updateRegularIngressStatusAndEvent* to update the status of the Ingress resource and emit an event with the status of the reload. Both involve making an API call to the Kubernetes API.

Notes:

* Some details weren't discussed for simplicity. You can view the source code if you want a fuller picture.
* The *syncVirtualServer*, *syncVirtualServerRoute*, and *syncTransportServer* methods are similar to syncIngress. The other sync methods are different. However, those methods typically involve finding the affected Ingress, VirtualServer, and TransportServer resources and regenerating a configuration for them.
* The *Workqueue* has only a single worker thread that calls the sync method synchronously. This means that the control loop processes only one change at a time.

#### Helper Components

There are two additional helper components crucial for processing changes: *Configuration* and *LocalSecretStore*.

##### Configuration

[*Configuration*](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/configuration.go#L320) holds the latest valid state of the IC load balancing configuration resources: Ingresses, VirtualServers, VirtualServerRoutes, TransportServers, and GlobalConfiguration.

The *Configuration* supports add (for add/update) and delete operations on the resources. When you add/update/delete a resource in the Configuration, it performs the following:

1. Validates the object (for add/update)
2. Calculates the changes to the affected resources that are necessary to propagate to the NGINX config, returning the changes to the caller.

For example, when you add a new Ingress resource, the *Configuration* returns a change requiring the IC to add the configuration for that Ingress to the NGINX config files. Another example: if you make an existing Ingress resource invalid, the *Configuration* returns a change requiring the IC to remove the configuration for that Ingress from the NGINX config files.

Additionally, the *Configuration* ensures that only one Ingress/VirtualServer/TransportServer (TLS Passthrough) holds a particular host (for example, cafe.example.com) and only one TransportServer (TCP/UDP) holds a particular listener (for example, port 53 for UDP). This ensures that no host or listener collisions happen in the NGINX config.

Ultimately, the IC ensures the NGINX config on the filesystem reflects the state of the objects in the *Configuration* at any point in time.

##### LocalSecretStore

[*LocalSecretStore*](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/secrets/store.go#L32) (of the *SecretStore* interface) holds the valid Secret resources and keeps the corresponding files on the filesystem in sync with them. Secrets are used to hold TLS certificates and keys (type `kubernetes.io/tls`), CAs (`nginx.org/ca`), JWKs (`nginx.org/jwk`), and client secrets for an OIDC provider (`nginx.org/oidc`).

When *Controller* processes a change to a configuration resource like Ingress, it creates an extended version of a resource that includes the dependencies -- such as Secrets -- necessary to generate the NGINX configuration. *LocalSecretStore* allows *Controller* to get a reference on the filesystem for a secret by the secret key (namespace/name).

## Reloading NGINX

The following section covers reloading NGINX in general and specifically how the Ingress Controller implements it.

### Reloading in General

Reloading NGINX is necessary to apply the new configuration and involves the following actions:
1. The administrator sends a HUP (hangup) signal to the NGINX master process to trigger a reload.
1. The master process brings down the worker processes with the old configuration and starts worker processes with the new configuration.
1. The administrator verifies the reload has successfully finished.
  
> Refer to the [NGINX documentation](https://nginx.org/en/docs/control.html#reconfiguration) for more details about reloading. See also [this blog post](https://www.nginx.com/blog/inside-nginx-how-we-designed-for-performance-scale/) for an overview of the NGINX architecture.

#### How to Reload

The NGINX binary (`nginx`) supports the reload operation with the `-s reload` option. When you run this option:
1. It validates the new NGINX configuration and exits if it is invalid printing the error messages to the stderr.
1. It sends a HUP signal to the NGINX master process and exits.

Alternatively, you can send a HUP signal to the NGINX master process directly.

#### How to Confirm Reloading Success

`nginx -s reload` doesn't wait for NGINX to finish reloading. As a result, it is the responsibility of the administrator to confirm it. There are a few options:
* Check if the master process created new worker processes. For example, by running `ps` or reading the `/proc` file system.
* Send an HTTP request to NGINX, and if a new worker process responds, you'll know the NGINX reloaded successfully. Note: this requires additional NGINX config, see the [Reloading in the Ingress Controller section](#reloading-in-the-ic).

Reloading takes time, usually at least 200ms. The time depends on the configuration size, the number of TLS certificates/keys, enabled modules, configuration details, and the available CPU resources.

#### Potential Problems

Most of the time, if `nginx -s reload` succeeds, the reload will also succeed. In rare cases the reload fails, the NGINX master will print the error message to the error log. For example:
```
2022/07/09 00:56:42 [emerg] 1353#1353: limit_req "one" uses the "$remote_addr" key while previously it used the "$binary_remote_addr" key
```

The operation is graceful; reloading doesn't lead to any traffic loss by NGINX. However, frequent reloads can lead to high memory utilization and potentially NGINX stopping with an OOM (Out-Of-Memory) error, resulting in traffic loss. This can happen if you (1) proxy traffic that utilizes long-lived connections (ex: Websockets, gRPC) and (2) reload frequently. In this case, you can end up with multiple generations of NGINX worker processes that are shutting down (old NGINX workers will not shut down until all connections are terminated either by clients or backends, unless you configure [worker_shutdown_timeout](https://nginx.org/en/docs/ngx_core_module.html#worker_shutdown_timeout) which will force old workers to shut down after the timeout). Eventually, all those worker processes can exhaust the system's available memory.

Since both the old and new NGINX worker processes coexist during a reload, reloading can lead to a spike in memory utilization up to two times. Because of the lack of available memory, the NGINX master process can fail to create new worker processes.

### Reloading in the IC

The Ingress Controller reloads NGINX to apply configuration changes. 

To facilitate reloading, the Ingress Controller configures a server listening on the Unix socket `unix:/var/lib/nginx/nginx-config-version.sock` that responds with the config version for `/configVersion` URI. The Ingress Controller writes the config to  `/etc/nginx/config-version.conf`. 

A reload involves multiple steps:
1. The Ingress Controller updates generated configuration files, including any secrets.
1. The Ingress Controller updates the config version in `/etc/nginx/config-version.conf`. 
1. The Ingress Controller runs `nginx -s reload`. If the command fails, the Ingress Controller logs the error and considers the reload failed.
2. Assuming the command succeeds, the Ingress Controller periodically checks for the config version by sending an HTTP request to the config version server on  `unix:/var/lib/nginx/nginx-config-version.sock`.
3. Once the Ingress Controller sees the correct config version returned by NGINX, it considers the reload successful. If it doesn't see the correct config version after the configurable timeout (see `-nginx-reload-timeout` [cli argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments), the Ingress Controller considers the reload failed.

> The [Ingress Controler Control Loop](#the-control-loop) stops during a reload so that it cannot change any configuration files or reload NGINX until the current reload succeeds or fails.

### When the Ingress Controller Reloads NGINX

The Ingress Controller reloads NGINX every time the Control Loop processes a change that affects the generated NGINX configuration. In general, every time a resource is changed, the Ingress Controller will regenerate the configuration and reload NGINX. A resource could be of any type the Ingress Controller monitors -- see [The Ingress Controller is a Kubernetes Controller](#the-ingress-controller-is-a-kubernetes-controller) section.

There are three special cases:
- *Start*. When the Ingress Controller starts, it processes all resources in the cluster and only then reloads NGINX. This avoids creating a "reload storm" by reloading only once.
- *Batch updates*. When the Ingress Controller receives a number of concurrent requests from the Kubernetes API, it will pause NGINX reloads until the task queue is empty. This reduces the number of reloads to minimize the impact of batch updates and reduce the risk of OOM (Out of Memory) errors.
- *NGINX Plus*. If the Ingress Controller uses NGINX Plus, it will not reload NGINX Plus for changes to the Endpoints resources. In this case, the Ingress Controller will use the NGINX Plus API to update the corresponding upstreams and skip reloading.

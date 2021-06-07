# How NGINX Ingress Controller Works

This document explains how the NGINX Ingress Controller works. The target audience is the two main groups:
* *Operators* who would like to know how the software works and also better understand how it can fail.
* *Developers* who would like to [contribute](https://github.com/nginxinc/kubernetes-ingress/blob/master/CONTRIBUTING.md) to the project.

We assume that the reader is familiar with core Kubernetes concepts, such as Pod, Deployment, Service, Endpoints. Additionally, we recommend reading [this blog post](https://www.nginx.com/blog/inside-nginx-how-we-designed-for-performance-scale/) for an overview of the NGINX architecture.

## Contents

- [How NGINX Ingress Controller Works](#how-nginx-ingress-controller-works)
  - [Contents](#contents)
  - [What is an Ingress Controller?](#what-is-an-ingress-controller)
  - [The Ingress Controller at a High Level](#the-ingress-controller-at-a-high-level)
  - [The Ingress Controller Pod](#the-ingress-controller-pod)
    - [Differences for NGINX Plus](#differences-for-nginx-plus)
  - [The Ingress Controller Process](#the-ingress-controller-process)
    - [Processing a New Ingress Resource](#processing-a-new-ingress-resource)
    - [The Ingress Controller is a Kubernetes Controller](#the-ingress-controller-is-a-kubernetes-controller)
  - [Ingress Controller Process Components](#ingress-controller-process-components)
    - [Resource Caches](#resource-caches)
    - [The Control Loop](#the-control-loop)
      - [The Controller Sync Method](#the-controller-sync-method)
      - [Helper Components](#helper-components)
        - [Configuration](#configuration)
        - [LocalSecretStore](#localsecretstore)

## What is an Ingress Controller?

An Ingress Controller is a Kubernetes component that runs in a Kubernetes cluster and configures an HTTP load balancer according to Ingress resources that the cluster users create.

> If you’d like to read more about the Ingress resource, refer to [the official Kubernetes documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/).

Within the context of this document references are specific to the NGINX Ingress Controller, which is built upon the capabilities of NGINX and NGINX Plus.  In this document, we will refer to it as the *Ingress Controller* or simply the *IC*.

## The Ingress Controller at a High Level 

First, let’s explore the Ingress Controller (IC) at a high level. The figure below depicts an example of how the IC exposes two web application running within a Kubernetes cluster to the clients on the internet:

![IC at a high level](how-nginx-ingress-controller-works-images/ic-high-level.png)

On the figure, we have:
* A *Kubernetes cluster*.
* Cluster users *Admin*, *User A* and *User B*, which use the cluster via the *Kubernetes API*.
* *Clients A* and *Clients B*, which connect to the *Applications A* and *B* deployed by the corresponding users.
* *IC*, [deployed by *Admin*](/nginx-ingress-controller/installation/installation-with-manifests) in a pod in the namespace *nginx-ingress* and configured via the *ConfigMap nginx-ingress*. For simplicity, we only depict one IC pod. However, *Admin* would typically deploy at least two pods for redundancy. The *IC* uses the *Kubernetes API* to get the latest Ingress resources created in the cluster and then configures *NGINX* according to those resources.
* *Application A* with two pods deployed in the *namespace A* by *User A*. To expose the application to its clients (*Clients A*) via the host `a.example.com`,  *User A* creates *Ingress A*.
* *Application B* with one pod deployed in the *namespace B* by *User B*. To expose the application to its clients (*Clients B*) via the host `b.example.com`, *User B* creates *VirtualServer B*.
* *Public Endpoint*, which fronts the IC pod(s). This is typically a TCP load balancer (cloud, software or hardware) or a combination of such load balancer with a NodePort service. *Clients A* and *B* connect to their applications via the *Public Endpoint*.

We use yellow and purple arrows to represent connections related to the client traffic and black arrows to represent access to the Kubernetes API.

> For simplicity, we omitted many necessary Kubernetes resources like Deployment and Services which Admin and the users also need to create.

Next, we will explore the IC pod.

## The Ingress Controller Pod

The IC pod consists of a single container which in turn consists of:
* *IC process*, which configures NGINX according to Ingress and other resources created in the cluster.
* *NGINX master process*, which controls NGINX worker processes.
* *NGINX worker processes*, which handle the client traffic and load balance it to the backend applications.

Below is an architectural diagram that shows how those processes interact with each other and with some external processes/entities: 

![IC pod](how-nginx-ingress-controller-works-images/ic-pod.png)

For brevity, we will omit the suffix process from the description of the processes.

<!-- TO-DO: after the docs are converted to Hugo, convert this list to a markdown table>
The table below describes each connection:
| Connection # | Type | Description |
| --- | --- | --- |
-->

The numbered list below describes each connection with its type in curly brackets:
1. (HTTP) *Prometheus* fetches the IC and NGINX metrics via an HTTP endpoint that the *IC* exposes. The default is ``:9113/metrics``. **Note**: *Prometheus* is not required by the IC, the endpoint can be turned off. 
2. (HTTPS) The *IC* reads the *Kubernetes API* to get the latest versions of the resources in the cluster and writes to the API to update the statuses of the handled resources and also emit events.
3. (HTTP) *Kubelet* probes the *IC* readiness probe (the default is `:8081/nginx-ready`) to consider the IC pod [ready](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-conditions).
4. (File I/O) When the *IC* starts, it reads the *configuration templates* necessary for config generation from the filesystem. The templates are located in the `/` directory of the container and have the `.tmpl` extension.
5. (File I/O) The *IC* writes logs to its *stdout* and *stderr*, which are collected by the container runtime.
6. (File I/O) The *IC* generates NGINX *configuration* based on the resources created in the cluster (see the [The Ingress Controller is a Kubernetes Controller](#the-ingress-controller-is-a-kubernetes-controller) section for the list of resources) and writes it on the file system in the `/etc/nginx` folder. The configuration files have `.conf` extension.
7. (File I/O) The *IC* writes *TLS certificates* and *keys* from any [TLS Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#tls-secrets) referenced in the Ingress and other resources to the file system.
8. (HTTP) The *IC* fetches the [NGINX metrics](https://nginx.org/en/docs/http/ngx_http_stub_status_module.html#stub_status) via the `unix:/var/lib/nginx/nginx-status.sock` unix socket and converts it to Prometheus format used in 1. 
9. (HTTP) To consider a configuration reload successful, the *IC* ensures that at least one *NGINX worker* has the new configuration. To do that, the *IC* checks a special endpoint via the `unix:/var/lib/nginx/nginx-config-version.sock` unix socket.
10. (N/A) To start NGINX, the *IC* executes `nginx` command, which launches the *NGINX master*.
11. (Signal) To reload NGINX, the *IC* executes `nginx -s reload` command, which validates the configuration and sends the [reload signal](https://nginx.org/en/docs/control.html) to the *NGINX master*.
12. (Signal) To shutdown NGINX, the *IC* executes `nginx -s quit` command, which sends the graceful shutdown signal to the *NGINX master*. 
13. (File I/O) The *NGINX master* sends logs its *stdout* and *stderr*, which are collected by the container runtime.
14. (File I/O) The *NGINX master* reads *TLS cert and keys* referenced in the configuration when it starts or during a reload.
15. (File I/O) The *NGINX master* reads *configuration files* when it starts or during a reload.
16. (Signal) The *NGINX master* controls the [lifecycle of *NGINX workers*](https://nginx.org/en/docs/control.html#reconfiguration): it creates workers with the new configuration and shutdowns workers with the old configuration.
17. (File I/O) An *NGINX worker* writes logs to its *stdout* and *stderr*, which are collected by the container runtime.
18. (UDP) An *NGINX worker* sends the HTTP upstream server response latency logs via the Syslog protocol over the unix socket `/var/lib/nginx/nginx-syslog.sock`  to the *IC*. In turn, the IC analyzes and transforms the logs into Prometheus metrics.
19. (HTTP,HTTPS,TCP,UDP) A *client* sends traffic to and receives from any of the *NGINX workers* on ports 80 and 443 and any additional ports exposed by the [GlobalConfiguration resource](/nginx-ingress-controller/configuration/global-configuration/globalconfiguration-resource).
20. (HTTP,HTTPS,TCP,UDP) An *NGINX worker* sends traffic to and receives from the *backends*. 
21. (HTTP) *Admin* can connect to the [NGINX stub_status](http://nginx.org/en/docs/http/ngx_http_stub_status_module.html#stub_status) via port 8080 via an *NGINX worker*. Note that by default, NGINX will only allow connections from `127.0.0.1`.

### Differences for NGINX Plus

The diagram depicts the IC with NGINX. The IC also supports NGINX Plus with the following differences relevant to the diagram:
* To configure NGINX Plus, in addition to configuration reloads, the IC also uses [NGINX Plus API](http://nginx.org/en/docs/http/ngx_http_api_module.html#api), which allows the IC to dynamically change the upstream servers of an upstream.
* Instead of the stub status metrics, the extended metrics are used, which are available via NGINX Plus API.
* In addition to TLS certs and keys, the IC writes JWKs from the secrets of the type `nginx.org/jwk` and NGINX workers read them.

## The Ingress Controller Process

In this section, we will cover the architecture of the IC process:
1. How the IC processes a new Ingress resource created by a user. 
2. The summary of how the IC works and how it relates to Kubernetes Controllers.  
3. The different components of the IC process. 

### Processing a New Ingress Resource

The diagram below depicts how the IC processes a new Ingress resource. Note that for simplicity we depict the NGINX master and worker processes as a single rectangle *NGINX*. Also note that VirtualServer and VirtualServerRoute resources are processed similarly.

![IC process](how-nginx-ingress-controller-works-images/ic-process.png)

Processing a new Ingress resource involves the following steps, where each step corresponds to the arrow on the diagram with the same number:
1. *User* creates a new Ingress resource.
2. The IC process has a *Cache* of the resources in the cluster. The *Cache* only includes the resources the IC is interested in, such as Ingresses. The *Cache* stays in sync with the Kubernetes API by [watching for changes to the resources](https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes).
3. Once the *Cache* has the new Ingress resource, it notifies the *Control loop* about the changed resource.
4. The *Control loop* gets the latest version of the Ingress resource from the *Cache*. Because the Ingress resource references other resources, such as TLS Secrets, the *Control loop* gets the latest versions of  any referenced resources as well. 
5. The *Control loop* generates and writes to the file system TLS certificates and keys from the TLS Secrets.
6. The *Control loop* generates and writes to the file system the NGINX *configuration files*, which correspond to the Ingress resource.
7. The *Control loop* reloads *NGINX* and waits for *NGINX* to be successfully reloaded. As part of the reload:
    1. *NGINX* reads the *TLS certs and keys*.
    1. *NGINX* reads the *configuration files*.
8. The *Control loop* emits an event for the Ingress resource and updates its status.  If the reload failed, the event will include the error message.

### The Ingress Controller is a Kubernetes Controller

Based on the example from the previous section, we can generalize how the IC works:

*The IC constantly processes new resources or changes to the existing resources in the cluster. As a result, the NGINX configuration stays up to date with the resources in the cluster.*

The IC is an example of a [Kubernetes controller](https://kubernetes.io/docs/concepts/architecture/controller/): the IC runs a control loop that ensures that NGINX is configured according to the desired state (Ingresses and other resources).

The desired state is concentrated in the following built-in Kubernetes resources and Custom resources (CRs):
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

The IC can watch additional Custom resources, which are less common and not enabled by default:
* [NGINX App Protect resources](/nginx-ingress-controller/app-protect/configuration/) (APPolicies, APLogConfs, APUserSigs)
* IngressLink resource (only one resource)

Next, we will zoom in and see different components of the IC process.

## Ingress Controller Process Components

In this section, we will describe the components of the IC process and how they interact:
1. How the IC watches for resources changes.
1. The main components of the IC control loop.
1. How those components work to process a resource change.
1. A few additional components, which are crucial for processing changes.

The IC is written in [go](https://golang.org/) and relies heavily on the [Go client for Kubernetes](https://github.com/kubernetes/client-go). In the sections next, we will include links to the code on GitHub when necessary.

### Resource Caches

In the section [Processing a New Ingress Resource](#processing-a-new-ingress-resource) we mentioned that the IC has a cache of the resources in the cluster that stays in sync with the Kubernetes API by watching for changes to the resources. We also mentioned that once cache is updated, it notifies the control loop about the changed resource.

The cache is actually a collection of *informers*. The diagram gives you more details on how changes to resources are processed by the IC. 

![IC process components](how-nginx-ingress-controller-works-images/ic-process-components.png)

* For every resource type the IC is interested in, it creates an [*Informer*](https://pkg.go.dev/k8s.io/client-go@v0.21.0/tools/cache#SharedInformer). The *Informer* includes a *Store* that holds the resources of that type. To keep the store in sync with the latest versions of the resources in the cluster, the *Informer* calls the watch and list *Kubernetes APIs* for that resource type (see the arrow *1. Watches and Lists* on the diagram). 
* When a change happens in the cluster (for example, a new resource is created), the *Informer* will update its *Store* and then invoke [*Handlers*](https://pkg.go.dev/k8s.io/client-go@v0.21.0/tools/cache#ResourceEventHandler) (see the arrow *2. Invokes*) for that informer.
* The IC registers handlers for every informer. Most of the time, a *Handler* creates an entry for the affected resource in the *Workqueue* where a workqueue element includes the type of the resource and its namespace and name. (See the arrow *3. Puts*)
* The *Workqueue* always tries to drain itself: if there is an element at the front, the queue will remove the element and send it to the *Controller* by calling a callback function.  (See the arrow *4. Sends*)
* The *Controller* is the primary component in the IC, which represents the control loop. We will explain the component in the [The Control Loop section](#the-control-loop). For now, it suffices to know that to process a workqueue element, it will get the latest version of the resource from the *Store* (see the arrow *5. Gets*), reconfigure *NGINX* according to the resource (see the arrow *6. Reconfigures*) and update the resource status and emit an event via the *Kubernetes API* (see the arrow  *7. Updates status and emits events*).

### The Control Loop

In this section we’ll describe the main components of the IC, which comprise the control loop:
* [Controller](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/controller.go#L90)
    * Runs the IC Control loop.
    * Instantiates Informers, handlers, the Workqueue and additional helper components.
    * Includes the sync method (see the next section), which is called by the Workqueue to process a changed resource.
    * Passes changed resources to *Configurator* to re-configure NGINX.
* [Configurator](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/configs/configurator.go#L95)
    * Generates NGINX configuration files, TLS and cert keys, JWKs based on Kubernetes resource.
    * Uses *Manager* to write the generated files and reload NGINX.
* [Manager](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/nginx/manager.go#L52)
    * Controls the lifecycle of NGINX (starting, reloading, quitting).
    * Manages the configuration files, TLS keys and certs, JWKs.

The diagram below depicts how the three components interact:

![Control loop](how-nginx-ingress-controller-works-images/control-loop.png)

#### The Controller Sync Method

The Controller [sync](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/controller.go#L663) method is called by the *Workqueue* to process a change of a resource. The method determines the *kind* of the resource and calls the appropriate for that kind *sync* method (for example, *syncIngress* for Ingresses).

Rather than showing how all of the various sync methods work, we will focus on the most important one  – the *syncIngress* - and show how it processes a new Ingress resource, which is illustrated on the diagram below.

![Controller sync](how-nginx-ingress-controller-works-images/controller-sync.png)

1. The *sync* method is called by the *Workqueue*, which passes a workqueue element to it that includes the changed resource *kind* and *key* (the key is the resource namespace/name like “default/cafe-ingress”).
2. Using the *kind*, the *sync* method calls the appropriate for that kind sync method and passes the resource key. For Ingresses, that method is *syncIngress*.
3. *syncIngress* gets the Ingress resource from the *Ingress Store* using the key. The *Store* is controlled by the *Ingress Informer*, as mentioned in the section Resource Caches. Note that in the code we use the helper *storeToIngressLister* type that wraps the Store.
4. *syncIngress* calls *AddOrUpdateIngress* of the *Configuration* passing the Ingress along. The [Configuration](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/configuration.go#L320) is a component that represents a valid collection of load balancing configuration resources (Ingresses, VirtualServers, VirtualServerRoutes, TransportServers), ready to be converted to NGINX configuration (See the [Configuration section](#configuration) for more details). *AddOrUpdateIngress* returns a list of [ResourceChanges](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/configuration.go#L59), which must be reflected in the NGINX config. Typically, for a new Ingress resource, the *Configuration* returns only a single *ResourceChange*.
5. *syncIngress* calls *processChanges*, which will process our single Ingress *ResourceChange*.
    1. *processChanges*  creates an extended Ingress resource (*IngressEx*) that includes the original Ingress resource as well as its dependencies such as Endpoints and Secrets, necessary for generating the NGINX configuration. For simplicity, we don’t show this step on the diagram.
    2. *processChanges* calls *AddOrUpdateIngress* of the *Configurator* passing the extended Ingress resource.
6. *Configurator* generates an NGINX configuration file based on the extended Ingress resource and then:
    1. Calls  *Manager’s CreateConfig()* to  update the config for the Ingress resource.
    2. Calls  *Manager’s Reload()* to reload NGINX.
7. The reload status is propagated from *Manager* to *processChanges*. The status is either a success or a failure with an error message.
8. *processChanges* calls *updateRegularIngressStatusAndEvent* to update the status of the Ingress resource and emit an event with the status of the reload. Both involve making an API call to the Kubernetes API.

Notes:
*  For simplicity, we omitted some of the details. To get the full picture, please refer to the source code.
* *syncVirtualServer*, *syncVirtualServerRoute*, *syncTransportServer* methods are similar to syncIngress. The other sync methods are different. However, those typically involve finding the affected Ingress, VirtualServer and TransportServer resources and regenerating configuration for them.
* The *Workqueue* has only a single worker thread that calls the sync method synchronously. This means that the control loop processes only one change at a time. 

#### Helper Components

There are two additional components crucial for processing changes. 

##### Configuration

[*Configuration*](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/configuration.go#L320) holds the latest valid state of IC load balancing configuration resources: Ingresses, VirtualServers, VirtualServerRoutes, TransportServers and GlobalConfiguration. 

The *Configuration* supports add (for add/update) and delete operations on the resources. When you add/update/delete a resource in the Configuration, it will:
1. Validate the object (for add/update)
2. Calculate the changes to the affected resources that are necessary to propagate to the NGINX config, returning the changes to the caller. 

For example, when you add a new Ingress resource, the *Configuration* will return a change requiring the IC to add the configuration for that Ingress to the NGINX config files. Another example: if you make an existing Ingress resource invalid, the *Configuration* will return a change requiring the IC to remove the configuration for that Ingress from the NGINX config files.

Additionally, the *Configuration* ensures that only one Ingress/VirtualServer/TransportServer(TLS Passthrough) holds a particular host (for example, cafe.example.com) and only one TransportServer(TCP/UDP) holds a particular listener (for example, port 53 for UDP). This will ensure that no host or listener collisions happen in the NGINX config.

Ultimately, the IC needs to ensure that at any point in time the NGINX config on the filesystem reflects the state of the objects in the *Configuration*.

##### LocalSecretStore

[*LocalSecretStore*](https://github.com/nginxinc/kubernetes-ingress/blob/v1.11.0/internal/k8s/secrets/store.go#L32) (of the *SecretStore* interface) holds the valid Secret resources and keeps the corresponding files on the file system in sync with them. Secrets are used to hold TLS certificates and keys (type `kubernetes.io/tls`), CAs (`nginx.org/ca`), JWKs (`nginx.org/jwk`) and client secrets for an OIDC provider (`nginx.org/oidc`).

When *Controller* processes a change of a configuration resource like Ingress, it creates an extended version of a resource that includes its dependencies necessary to generate NGINX configuration. Among those decencies are Secrets. *LocalSecretStore* allows *Controller* to get a reference on the file system for a secret by the secret key (namespace/name).
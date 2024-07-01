---
docs: DOCS-590
doctypes:
- ''
title: Host and Listener collisions
toc: true
weight: 1700
---

This document explains how F5 NGINX Ingress Controller handles host and listener collisions between resources.

---

## Winner Selection Algorithm

If multiple resources contend for the same host or listener, NGINX Ingress Controller will pick the winner based on the `creationTimestamp` of the resources: the oldest resource will win. In case there are more than one oldest resource (their `creationTimestamp` is the same),  NGINX Ingress Controller will choose the resource with the lexicographically smallest `uid`.

{{< note >}} The `creationTimestamp` and `uid` fields are part of the [ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta) resource. {{< /note >}}

---

## Host collisions

A host collision occurs when multiple Ingress, VirtualServer, and TransportServer (configured for TLS Passthrough) resources configure the same `host`. NGINX Ingress Controller has two strategies for handling host collisions:

- Choosing a single "winner" resource to handle the host.
- Merging the configuration of the conflicting resources.

---

### Choosing the winner

Consider the following two resources:

- `cafe-ingress` Ingress:

    ```yaml
    apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: cafe-ingress
    spec:
      ingressClassName: nginx
      rules:
      - host: cafe.example.com
        . . .
    ```

- `cafe-virtual-server` VirtualServer:

    ```yaml
    apiVersion: k8s.nginx.org/v1
    kind: VirtualServer
    metadata:
      name: cafe-virtual-server
    spec:
      host: cafe.example.com
      . . .
    ```

If a user creates both resources in the cluster, a host collision will occur. NGINX Ingress Controller will pick the winner using the [winner selection algorithm](#winner-selection-algorithm).

If `cafe-virtual-server` was created first, it will win the host `cafe.example.com` and NGINX Ingress Controller will reject `cafe-ingress`. This will be reflected in the events and in the resource's status field:

```shell
kubectl describe vs cafe-virtual-server
```
```text
...
Status:
  ...
  Message:  Configuration for default/cafe-virtual-server was added or updated
  Reason:   AddedOrUpdated
  State:    Valid
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  9s    nginx-ingress-controller  Configuration for default/cafe-virtual-server was added or updated
```

```shell
kubectl describe ingress cafe-ingress
```
```text
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Warning  Rejected  66s   nginx-ingress-controller  All hosts are taken by other resources
```

Similarly, if `cafe-ingress` was created first, it will win `cafe.example.com` and NGINX Ingress Controller will reject `cafe-virtual-server`.

{{< note >}} You can configure multiple hosts for Ingress resources, and its possible that an Ingress resource can be the winner for some of its hosts and a loser for the others. 

For example, if `cafe-ingress` had an additional rule host rule for `pub.example.com`, NGINX Ingress Controller would not reject the Ingress. Instead, it would allow `cafe-ingress` to handle `pub.example.com`. {{< /note >}}

---

### Merging configuration for the same host

It is possible to merge configuration for multiple Ingress resources for the same host. One common use case for this approach is distributing resources across multiple namespaces. 

The [Cross-namespace configuration]({{< relref "configuration/ingress-resources/cross-namespace-configuration.md">}}) topic has more information.

It is *not* possible to merge the configurations for multiple VirtualServer resources for the same host. However, you can split the VirtualServers into multiple VirtualServerRoute resources, which a single VirtualServer can then reference. See the [corresponding example](https://github.com/nginxinc/kubernetes-ingress/tree/v3.6.1/examples/custom-resources/cross-namespace-configuration) on GitHub.

It is *not* possible to merge configuration for multiple TransportServer resources.

---

## Listener collisions

Listener collisions occur when multiple TransportServer resources (Configured for TCP/UDP load balancing) target the same `listener`. 

NGINX Ingress Controller will choose the winner, which will own the listener.

---

### Choosing the winner

Consider the following two resources:

- `tcp-1` TransportServer:

    ```yaml
    apiVersion: k8s.nginx.org/v1
    kind: TransportServer
    metadata:
      name: tcp-1
    spec:
      listener:
        name: dns-tcp
        protocol: TCP
        . . .
    ```

- `tcp-2` TransportServer:

    ```yaml
    apiVersion: k8s.nginx.org/v1
    kind: TransportServer
    metadata:
      name: tcp-2
    spec:
      listener:
        name: dns-tcp
        protocol: TCP
        . . .
    ```

If a user creates both resources in the cluster, a listener collision will occur. As a result, NGINX Ingress Controller will pick the winner using the [winner selection algorithm](#winner-selection-algorithm).

In our example, if `tcp-1` was created first, it will win the listener `dns-tcp` and NGINX Ingress Controller will reject `tcp-2`. This will be reflected in the events and in the resource's status field:

```shell
kubectl describe ts tcp-2
```
```text
...
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Warning  Rejected  10s   nginx-ingress-controller  Listener dns-tcp is taken by another resource
```

Similarly, if `tcp-2` was created first, it will win `dns-tcp` and NGINX Ingress Controller will reject `tcp-1`.

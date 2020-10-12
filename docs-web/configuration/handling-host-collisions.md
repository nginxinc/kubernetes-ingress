# Handling Host Collisions

A host collision occurs when multiple resources configure the same `host`. The Ingress Controller supports two options for handling host collisions:
* Choosing the winner so that only one resource handles the host.
* Merging configuration of the conflicting resources.

## Choosing the Winner

Consider the following two resources:
* `cafe-ingress` Ingress:
    ```yaml
    apiVersion: networking.k8s.io/v1beta1
    kind: Ingress
    metadata:
      name: cafe-ingress
      annotations:
        kubernetes.io/ingress.class: "nginx"
    spec:
      rules:
      - host: cafe.example.com
        . . .
    ```
* `cafe-virtual-server` VirtualServer:
    ```yaml
    apiVersion: k8s.nginx.org/v1
    kind: VirtualServer
    metadata:
      name: cafe-virtual-server
    spec:
      host: cafe.example.com
      . . .
    ```

If a user creates both resources in the cluster, a host collision will occur. As a result, the Ingress Controller will pick the winner using the following algorithm:

> If multiple resources contend for the same host, the Ingress Controller will pick the winner based on the `creationTimestamp` of the resources: the oldest resource will win. In case there are more than one oldest resources (their `creationTimestamp` is the same),  the Ingress Controller will choose the resource with the lexicographically smallest `uid`.

> Note: the `creationTimestamp` and `uid` fields are part of the resource [ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta). 

In our example, if `cafe-virtual-server` was created first, it will win the host `cafe.example.com` and the Ingress Controller will reject `cafe-ingress`. This will be reflected in the events and in the resource's status field:
```
$ kubectl describe vs cafe-virtual-server
. . .
Status:
  . . .
  Message:  Configuration for default/cafe-virtual-server was added or updated
  Reason:   AddedOrUpdated
  State:    Valid
Events:
  Type    Reason          Age   From                      Message
  ----    ------          ----  ----                      -------
  Normal  AddedOrUpdated  9s    nginx-ingress-controller  Configuration for default/cafe-virtual-server was added or updated

$ kubectl describe ingress cafe-ingress
. . .
Events:
  Type     Reason    Age   From                      Message
  ----     ------    ----  ----                      -------
  Warning  Rejected  66s   nginx-ingress-controller  All hosts are taken by other resources
```
> Note: You can configure multiple hosts for Ingress resources. As a result, it's possible that an Ingress resource can be the winner for some of its hosts and a loser for the others. For example, if `cafe-ingress` had an additional rule host rule -- `pub.example.com` -- the Ingress Controller would not reject the Ingress. Rather, it would allow `cafe-ingress` to handle `pub.example.com`.

Similarly, if `cafe-ingress` was created first, it will win `cafe.example.com` and the Ingress Controller will reject `cafe-virtual-server`.

## Merging Configuration for the Same Host

It is possible to merge configuration for multiple Ingress resources for the same host. One common use case for this approach is distributing resources across multiple namespaces. See the [Cross-namespace Configuration](/nginx-ingress-controller/configuration/ingress-resources/cross-namespace-configuration/) doc for more information.

It is *not* possible to merge the configurations for multiple VirtualServer resources for the same host. However, you can split the VirtualServers into multiple VirtualServerRoute resources, which a single VirtualServer can then reference. See the [corresponding example](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples-of-custom-resources/cross-namespace-configuration) on GitHub.

# Differences Between nginxinc/kubernetes-ingress and kubernetes/ingress-nginx Ingress Controllers

There are two NGINX-based Ingress controller implementations out there: the one you can find in this repo (nginxinc/kubernetes-ingress) and the one from [kubernetes/ingress-nginx](https://github.com/kubernetes/ingress-nginx) repo. In this document, we explain the key differences between those implementations. This information should help you to choose an appropriate implementation for your requirements or move from one implementation to the other.

## Which One Am I Using?

If you are unsure about which implementation you are using, check the container image of the Ingress controller that is running. For the nginxinc/kubernetes-ingress Ingress controller its Docker image is published on [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/) and available as *nginx/nginx-ingress*.

## The Key Differences

The table below summarizes the key difference between nginxinc/kubernetes-ingress and kubernetes/ingress-nginx Ingress controllers. Note that the table has two columns for the nginxinc/kubernetes-ingress Ingress controller, as it can be used both with NGINX and NGINX Plus. For more information about nginxinc/kubernetes-ingress with NGINX Plus, read [here](nginx-plus.md). 

| Aspect or Feature | kubernetes/ingress-nginx | nginxinc/kubernetes-ingress with NGINX | nginxinc/kubernetes-ingress with NGINX Plus |
| --- | --- | --- | --- |
| **Fundamental** |
| Authors | Kubernetes community | NGINX Inc and community |  NGINX Inc and community |
| NGINX version | [Custom](https://github.com/kubernetes/ingress-nginx/tree/master/images/nginx) NGINX build that includes several third-party modules | NGINX official mainline [build](https://github.com/nginxinc/docker-nginx) | NGINX Plus |
| Commercial support | N/A | N/A | Included |
| **Load balancing configuration via the Ingress resource** |
| Merging Ingress rules with the same host | Supported | Supported via [Mergeable Ingresses](../examples/mergeable-ingress-types) | Supported via [Mergeable Ingresses](../examples/mergeable-ingress-types) |
| HTTP load balancing extensions - Annotations | See the [supported annotations](https://github.com/kubernetes/ingress-nginx/blob/master/docs/user-guide/nginx-configuration/annotations.md) | See the [supported annotations](configmap-and-annotations.md) | See the [supported annotations](configmap-and-annotations.md)|
| HTTP load balancing extensions -- ConfigMap | See the [supported ConfigMap keys](https://github.com/kubernetes/ingress-nginx/blob/master/docs/user-guide/nginx-configuration/configmap.md) | See the [supported ConfigMap keys](configmap-and-annotations.md) | See the [supported ConfigMap keys](configmap-and-annotations.md) |
| TCP/UDP | Supported via a ConfigMap | Supported via a ConfigMap with native NGINX configuration | Supported via a ConfigMap with native NGINX configuration |
| Websocket  | Supported | Supported via an [annotation](../examples/websocket) | Supported via an [annotation](../examples/websocket) |
| TCP SSL Passthrough | Supported via a ConfigMap | Not supported | Not supported |
| JWT validation | Not supported | Not supported | Supported |
| Session persistence | Supported via a third-party module | Not supported | Supported |
| Canary testing (by header, cookie, weight) | Supported via annotations | Supported via custom resources | Supported via custom resources |
| Configuration templates *1 | See the [template](https://github.com/kubernetes/ingress-nginx/blob/master/rootfs/etc/nginx/template/nginx.tmpl) | See the [templates](../internal/configs/version1) | See the [templates](../internal/configs/version1) |
| **Load balancing configuration via Custom Resources** |
| HTTP load balancing | Not supported | See [VirtualServer and VirtualServerRoute](virtualserver-and-virtualserverroute.md) resources. | See [VirtualServer and VirtualServerRoute](virtualserver-and-virtualserverroute.md) resources. |
| **Deployment** |
| Command-line arguments *2 | See the [arguments](https://github.com/kubernetes/ingress-nginx/blob/master/docs/user-guide/cli-arguments.md) | See the [arguments](cli-arguments.md) | See the [arguments](cli-arguments.md) |
| TLS certificate and key for the default server | Required as a command-line argument/ auto-generated | Required as a command-line argument | Required as a command-line argument |
| Helm chart | Supported | Supported | Supported |
| **Operational** |
| Reporting the IP address(es) of the Ingress controller into Ingress resources | Supported | Supported | Supported |
| Extended Status | Supported via a third-party module | Not supported | Supported |
| Prometheus Integration | Supported | Supported | Supported |
| Dynamic reconfiguration of endpoints (no configuration reloading) | Supported with a third-party Lua module | Not supported | Supported |

Notes:

*1 -- The configuration templates that are used by the Ingress controllers to generate NGINX configuration are different. As a result, for the same Ingress resource the generated NGINX configuration files are different from one Ingress controller to the other, which in turn means that in some cases the behavior of NGINX can be different as well.

*2 -- Because the command-line arguments are different, it is not possible to use the same deployment manifest for deploying the Ingress controllers.

## How to Swap an Ingress Controller

If you decide to swap an Ingress controller implementation, be prepared to deal with the differences that were mentioned in the previous section. At minimum, you need to start using a different deployment manifest.

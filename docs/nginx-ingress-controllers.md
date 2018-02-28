# Differences Between nginxinc/kubernetes-ingress and kubernetes/ingress-nginx Ingress Controllers

There are two NGINX-based Ingress controller implementations out there: the one you can find in this repo (nginxinc/kubernetes-ingress) and the one from [kubernetes/ingress-nginx](https://github.com/kubernetes/ingress-nginx) repo. In this document, we explain the key differences between those implementations. This information should help you to choose an appropriate implementation for your requirements or move from one implementation to the other.

## Which One Am I Using?

If you are unsure about which implementation you are using, check the container image of the Ingress controller that is running. For the nginxinc/kubernetes-ingress Ingress controller its Docker image is published on [DockerHub](https://hub.docker.com/r/nginxdemos/nginx-ingress/) and available as *nginxdemos/nginx-ingress*.

## The Key Differences

The table below summarizes the key difference between nginxinc/kubernetes-ingress and kubernetes/ingress-nginx Ingress controllers. Note that the table has two columns for the nginxinc/kubernetes-ingress Ingress controller, as it can be used both with NGINX and NGINX Plus

| Aspect or Feature | kubernetes/ingress-nginx | nginxinc/kubernetes-ingress with NGINX | nginxinc/kubernetes-ingress with NGINX Plus |
| --- | --- | --- | --- |
| **Fundamental** |
| Authors | Kubernetes community | NGINX Inc and community |  NGINX Inc and community |
| NGINX version | [Custom](https://github.com/kubernetes/ingress-nginx/tree/master/images/nginx) NGINX build that includes several third-party modules | NGINX official mainline [build](https://github.com/nginxinc/docker-nginx) | NGINX Plus |
| Commercial support | N/A | N/A | Included |
| **Load balancing configuration** |
| Merging Ingress rules with the same host | Supported | Under consideration | Under consideration |
| HTTP load balancing extensions - Annotations | See the [supported annotations](https://github.com/kubernetes/ingress-nginx/blob/master/docs/user-guide/annotations.md) | See the [supported annotations](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples/customization) | See the [supported annotations](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples/customization)|
| HTTP load balancing extensions -- ConfigMap | See the [supported ConfigMap keys](https://github.com/kubernetes/ingress-nginx/blob/master/docs/user-guide/configmap.md) | See the [supported ConfigMap keys](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples/customization) | See the [supported ConfigMap keys](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples/customization) |
| TCP/UDP | Supported via a ConfigMap | Not supported | Not supported |
| Websocket  | Supported | Supported via an [annotation](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples/websocket) | Supported via an [annotation](https://github.com/nginxinc/kubernetes-ingress/tree/master/examples/websocket) |
| TCP SSL Passthrough | Supported via a ConfigMap | Not supported | Not supported |
| JWT validation | Not supported | Not supported | Supported |
| Session persistence | Supported via a third-party module | Not supported | Supported |
| Configuration templates *1 | See the [template](https://github.com/kubernetes/ingress-nginx/blob/master/rootfs/etc/nginx/template/nginx.tmpl) | See the [templates](https://github.com/nginxinc/kubernetes-ingress/tree/master/nginx-controller/nginx/templates) | See the [templates](https://github.com/nginxinc/kubernetes-ingress/tree/master/nginx-controller/nginx/templates) |
| **Deployment** |
| Command-line arguments *2 | See the [arguments](https://github.com/kubernetes/ingress-nginx/blob/master/docs/user-guide/cli-arguments.md) | See the [arguments](https://github.com/nginxinc/kubernetes-ingress/blob/master/nginx-controller/main.go#L27) | See the [arguments](https://github.com/nginxinc/kubernetes-ingress/blob/master/nginx-controller/main.go#L27) |
| TLS certificate and key for the default server | Required as a command-line argument/ auto-generated | Required as a command-line argument | Required as a command-line argument |
| Helm chart | Supported | Coming soon | Coming soon |
| **Operational** |
| Reporting the IP address(es) of the Ingress controller into Ingress resources | Supported | Coming soon | Coming soon |
| Extended Status | Supported via a third-party module | Not supported | Supported |
| Prometheus Integration | Supported | Not supported | Supported |
| Dynamic reconfiguration of endpoints (no configuration reloading) | Not supported | Not supported | Supported |

Notes:

*1 -- The configuration templates that are used by the Ingress controllers to generate NGINX configuration are different. As a result, for the same Ingress resource the generated NGINX configuration files are different from one Ingress controller to the other, which in turn means that in some cases the behavior of NGINX can be different as well.

*2 -- Because the command-line arguments are different, it is not possible to use the same deployment manifest for deploying the Ingress controllers.

## How to Swap an Ingress Controller

If you decide to swap an Ingress controller implementation, be prepared to deal with the differences that were mentioned in the previous section. At minimum, you need to start using a different deployment manifest.

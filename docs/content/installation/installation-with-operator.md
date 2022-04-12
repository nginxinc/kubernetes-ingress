---
title: Installation with the NGINX Ingress Operator

description: "This document describes how to install the NGINX Ingress Controller in your Kubernetes cluster using the NGINX Ingress Operator."
weight: 2000
doctypes: [""]
toc: true
docs: "DOCS-604"
---

{{< note >}}
An NGINX Ingress Operator version compatible with the 2.2.0 NGINX Ingress Controller release is not available yet. We will update this document and remove this note once we publish a compatible Operator version.
{{< /note >}}

This document describes how to install the NGINX Ingress Controller in your Kubernetes cluster using the NGINX Ingress Operator.

## Prerequisites

1. Make sure you have access to the Ingress Controller image:
    * For NGINX Ingress Controller, use the image `nginx/nginx-ingress` from [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress).
    * For NGINX Plus Ingress Controller, see [here](/nginx-ingress-controller/installation/pulling-ingress-controller-image) for details on how to pull the image from the F5 Docker registry.
    * To pull from the F5 Container registry, configure a docker registry secret using your JWT token from the MyF5 portal by following the instructions from [here](/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret).
    * It is also possible to build your own image and push it to your private Docker registry by following the instructions from [here](/nginx-ingress-controller/installation/building-ingress-controller-image).
1. Install the NGINX Ingress Operator following the [instructions](https://github.com/nginxinc/nginx-ingress-operator/blob/main/docs/installation.md).

## 1. Create the NginxIngressController manifest

Create a manifest `nginx-ingress-controller.yaml` with the following content:

```yaml
apiVersion: k8s.nginx.org/v1alpha1
kind: NginxIngressController
metadata:
  name: my-nginx-ingress-controller
  namespace: default
spec:
  type: deployment
  image:
    repository: nginx/nginx-ingress
    tag: 2.2.0
    pullPolicy: Always
  serviceType: NodePort
  nginxPlus: False
```

**Note:** For NGINX Plus, change the `image.repository` and `image.tag` values and change `nginxPlus` to `True`.

## 2. Create the NginxIngressController

```
$ kubectl apply -f nginx-ingress-controller.yaml
```

A new instance of the NGINX Ingress Controller will be deployed by the NGINX Ingress Operator in the `default` namespace with default parameters.

To configure other parameters of the NginxIngressController resource, check the [documentation](https://github.com/nginxinc/nginx-ingress-operator/blob/main/docs/nginx-ingress-controller.md).

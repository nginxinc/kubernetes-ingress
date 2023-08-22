# Helm Chart Examples

This directory contains examples of Helm charts that can be used to deploy
NGINX Ingress Controller in a Kubernetes cluster.

## Prerequisites

- Helm 3.0+

## Examples

- [Default](./default) - deploys the NGINX Ingress Controller with default parameters.
- [NGINX App Protect DoS](./app-protect-dos) - deploys the NGINX Ingress Controller with the NGINX App Protect DoS
  module enabled. The image is pulled from the NGINX Plus Docker registry, and the `imagePullSecretName` is the name of
  the secret to use to pull the image. The secret must be created in the same namespace as the NGINX Ingress Controller.
- [NGINX App Protect WAF](./app-protect-waf) - deploys the NGINX Ingress Controller with the NGINX App Protect WAF
  module enabled. The image is pulled from the NGINX Plus Docker registry, and the `imagePullSecretName` is the name of
  the secret to use to pull the image. The secret must be created in the same namespace as the NGINX Ingress Controller.
- [AWS NLB](./aws-nlb) - deploys the NGINX Ingress Controller using a Service type of `LoadBalancer` to allocate an AWS
  Network Load Balancer (NLB).
- [Azure](./azure) - deploys the NGINX Ingress Controller using a nodeSelector to deploy the controller on Azure nodes.
- [DaemonSet](./daemonset) - deploys the NGINX Ingress Controller as a DaemonSet.
- [Edge](./edge) - deploys the NGINX Ingress Controller using the `edge` tag from Docker Hub.
  See the [README](../../README.md#nginx-ingress-controller-releases) for more information on the different tags.
- [NGINX Plus](./nginx-plus) - deploys the NGINX Ingress Controller with the NGINX Plus. The image is pulled from the
  NGINX Plus Docker registry, and the `imagePullSecretName` is the name of the secret to use to pull the image.
  The secret must be created in the same namespace as the NGINX Ingress Controller.
- [OIDC](./oidc) - deploys the NGINX Ingress Controller with OpenID Connect (OIDC) authentication enabled.
- [Read-only filesystem](./read-only-filesystem) - deploys the NGINX Ingress Controller with a read-only filesystem.
- [NodePort](./nodeport) - deploys the NGINX Ingress Controller using a Service type of `NodePort`.
- [Service Insight](./service-insight) - deploys the NGINX Ingress Controller with Service Insight enabled.
- [External DNS](./external-dns) - deploys the NGINX Ingress Controller with External DNS enabled.

## Manifests generation

These examples are used to generate manifests for the NGINX Ingress Controller located in the manifest folder
[here](../../deploy).

If you want to generate manifests for a specific example, or need to customize one of the examples, run the following
command from the root of the project:

```shell
helm template nginx-ingress --namespace nginx-ingress --values examples/helm-chart/<example-name>/values.yaml charts/nginx-ingress
```

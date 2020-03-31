# Cross-namespace Configuration

You can spread the Ingress configuration for a common host across multiple Ingress resources using Mergeable Ingress resources. Such resources can belong to the *same* or *different* namespaces. This enables easier management when using a large number of paths.

See the [Mergeable Ingress Resources](https://github.com/nginxinc/kubernetes-ingress/tree/v1.7.0-rc1/examples/mergeable-ingress-types) example on our GitHub.

# Custom NGINX log format

This example lets you set the log-format for NGINX using the configmap resource

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
  namespace: nginx-ingress
data:
  log-format:  '$remote_addr - $remote_user [$time_local] "$request" $status $grpc_status $body_bytes_sent "$http_referer"  "$http_user_agent" "$http_x_forwarded_for" "$resource_name" "$resource_type" "$resource_namespace" "$service"'
```

In addition to the [built-in NGINX variables](https://nginx.org/en/docs/varindex.html), you can also use the variables
that the Ingress Controller configures:

- $resource_type - The type of kubernetes resource that handled the client request.
- $resource_name - The name of the resource
- $resource_namespace - The namespace the resource exists in.
- $service - The name of the service the client request was sent to.
- $grpc_status - the gRPC status code, which is constructed either from the HTTP/2 trailer (grpc_status) returned from
  the backend for normal conditions, or from the HTTP/2 header (grpc_status) set either by the backend or by NGINX
  itself for some error conditions.

**note** These variables are only available for Ingress, VirtualServer and VirtualServerRoute resources.

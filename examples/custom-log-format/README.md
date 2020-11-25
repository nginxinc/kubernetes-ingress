# Custom NGINX log format

This example lets you set the log-format for NGINX using the configmap reosurce 

```yaml 
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
data:
  log-format: |
      compression '$remote_addr - $remote_user [$time_local] '
                       '"$request" $status $bytes_sent '
                       '"$http_referer" "$http_user_agent" "$gzip_ratio"'
```

In addition to the built-in NGINX variables, you can also use the variables that the Ingress Controller configures:

- $resource_type - The type of k8s resource. 
- $resource_name - The name of the k8s resource
- $resource_namespace - The namespace the resource exists in.
- $service - The service that exposes the resource.

# Custom NGINX log format

This example lets you set the log-format for NGINX using the configmap reosurce 

```yaml 
kind: ConfigMap
apiVersion: v1
metadata:
  name: oss-nginx-ingress
data:
  log-format:  $remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\"  \"$http_user_agent\" \"$http_x_forwarded_for\" $resource_name $resource_type $resource_namespace $service;
```

In addition to the built-in NGINX variables, you can also use the variables that the Ingress Controller configures:

- $resource_type - The type of k8s resource. 
- $resource_name - The name of the k8s resource
- $resource_namespace - The namespace the resource exists in.
- $service - The service that exposes the resource.

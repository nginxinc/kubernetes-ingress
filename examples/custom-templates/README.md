# Custom Templates

The Ingress controller allows you to customize your templates through a [ConfigMap](../../docs/configmap-and-annotations.md) via the following keys:
* `main-template` - Sets the main NGINX configuration template.
* `ingress-template` - Sets the Ingress NGINX configuration template for an Ingress resource.

## Example
```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
  namespace: nginx-ingress
data:
  main-template: |
    user  nginx;
    worker_processes  {{.WorkerProcesses}};
    ...
        include /etc/nginx/conf.d/*.conf;
    }
  ingress-template: |
    {{range $upstream := .Upstreams}}
    upstream {{$upstream.Name}} {
      {{if $upstream.LBMethod }}{{$upstream.LBMethod}};{{end}}
    ...
    }{{end}}
```
**Notes:**
* The templates are truncated for the clarity of the example.
* The templates for NGINX (the main `nginx.tmpl` and the Ingress `nginx.ingress.tmpl`) and NGINX Plus (the main `nginx-plus.tmpl` and the Ingress `nginx-plus.ingress.tmpl`) are located at [internal/configs/templates](../../internal/configs/version1/).

## Troubleshooting
* If a custom template contained within the ConfigMap is invalid on startup, the Ingress controller will fail to start, the error will be reported in the Ingress controller logs.

    An example of an error from the logs:
    ```
    Error updating NGINX main template: template: nginxTemplate:98: unexpected EOF
    ```

* If a custom template contained within the ConfigMap is invalid on update, the Ingress controller will not update the NGINX configuration, the error will be reported in the Ingress controller logs and an event with the error will be associated with the ConfigMap.

    An example of an error from the logs:
    ```
    Error when updating config from ConfigMap: Invalid nginx configuration detected, not reloading
    ```

  An example of an event with an error (you can view events associated with the ConfigMap by running `kubectl describe -n nginx-ingress configmap nginx-config`):

    ```
    Events:
      Type     Reason            Age                From                      Message
      ----     ------            ----               ----                      -------
      Normal   Updated           12s (x2 over 25s)  nginx-ingress-controller  Configuration from nginx-ingress/nginx-config was updated
      Warning  UpdatedWithError  10s                nginx-ingress-controller  Configuration from nginx-ingress/nginx-config was updated, but not applied: Error when parsing the main template: template: nginxTemplate:98: unexpected EOF
      Warning  UpdatedWithError  8s                 nginx-ingress-controller  Configuration from nginx-ingress/nginx-config was updated, but not applied: Error when writing main Config
    ```

# Custom Annotations

Custom annotations enable you to quickly extend the Ingress Controller to support many advanced features of NGINX, such as rate limiting, caching, etc.

Let's create a set of custom annotations to support [rate-limiting](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html):
* `custom.nginx.org/rate-limiting` - enables rate-limiting.
* `custom.nginx.org/rate-limiting-rate` - configures the rate of rate-limiting, with the default of `1r/s`.
* `custom.nginx.org/rate-limiting-burst` - configures the maximum bursts size of requests with the default of `3`.

## Prerequisites 

* Read the [custom annotations doc](../../docs/custom-annotations.md) before going through this example first.
* Read about [custom templates](../custom-templates).

## Step 1 - Customize the Template

Customize the template for Ingress resources to include the logic to handle and apply the annotations. 

1. Create a ConfigMap file with the customized template (`nginx-config.yaml`):
    ```yaml
    kind: ConfigMap
    apiVersion: v1
    metadata:
      name: nginx-config
      namespace: nginx-ingress
    data:
      ingress-template: |
        . . .
        # handling custom.nginx.org/rate-limiting` and custom.nginx.org/rate-limiting-rate 

        {{if index $.Ingress.Annotations "custom.nginx.org/rate-limiting"}}
        {{$rate := index $.Ingress.Annotations "custom.nginx.org/rate-limiting-rate"}}
        limit_req_zone $binary_remote_addr zone={{$.Ingress.Namespace}}-{{$.Ingress.Name}}:10m rate={{if $rate}}{{$rate}}{{else}}1r/s{{end}};
        {{end}}

        . . .

        {{range $server := .Servers}}
        server {
        
          . . .

          {{range $location := $server.Locations}}
          location {{$location.Path}} {

            . . .

            # handling custom.nginx.org/rate-limiting and custom.nginx.org/rate-limiting-burst

            {{if index $.Ingress.Annotations "custom.nginx.org/rate-limiting"}}
            {{$burst := index $.Ingress.Annotations "custom.nginx.org/rate-limiting-burst"}}
            limit_req zone={{$.Ingress.Namespace}}-{{$.Ingress.Name}} burst={{if $burst}}{{$burst}}{{else}}3{{end}} nodelay;
            {{end}}
        
            . . .
    ```

    The customization above consists of two parts:
    * handling the `custom.nginx.org/rate-limiting` and `custom.nginx.org/rate-limiting-rate` annotations in the `http` context.
    * handling the `custom.nginx.org/rate-limiting` and `custom.nginx.org/rate-limiting-burst` annotation in the `location` context.

    **Note**: for the brevity, the unimportant for the example parts of the template are replaced with `. . .`.

1. Apply the customized template:
    ```
    $ kubectl apply -f nginx-config.yaml
    ```

1. If the Ingress Controller fails to parse the customized template, it will attach an error event with the corresponding ConfigMap resource. You can see the events by running:
    ```
    $ kubectl describe configmap nginx-config -n nginx-ingress
    . . .
    Events:
    Type     Reason            Age                From                      Message
    ----     ------            ----               ----                      -------
    Normal   Updated           12s (x2 over 25s)  nginx-ingress-controller  Configuration from nginx-ingress/nginx-config was updated
    ```
    In this case, we got the `Updated` event meaning that the template was parsed successfully.

### Step 2 - Use Custom Annotations in an Ingress Resource

1. Create a file with the following Ingress resource (`cafe-ingress.yaml`) and use the custom annotations to enable rate-limiting:
    ```yaml
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: cafe-ingress
      annotations:
        kubernetes.io/ingress.class: "nginx"
        custom.nginx.org/rate-limiting: "on"
        custom.nginx.org/rate-limiting-rate: "5r/s"
        custom.nginx.org/rate-limiting-burst: "1"
    spec:
      rules:
      - host: "cafe.example.com"
        http:
          paths:
          - path: /tea
            backend:
              serviceName: tea-svc
              servicePort: 80
          - path: /coffee
            backend:
              serviceName: coffee-svc
              servicePort: 80
    ```

1. Apply the Ingress resource:
    ```
    $ kubectl apply -f cafe-ingress.yaml
    ```

1. Since it is possible that the value we put in `custom.nginx.org/rate-limiting-rate` or `custom.nginx.org/rate-limiting-burst` annotation might be considered invalid by NGINX, make sure to run the following command to check if the configuration for the Ingress resource was successfully applied. As with the ConfigMap resource, in case of an error, the Ingress Controller will attach an error event to the Ingress resource:
    ```
    $ kubectl describe ingress cafe-ingress
    Events:
    Type    Reason          Age   From                      Message
    ----    ------          ----  ----                      -------
    Normal  AddedOrUpdated  2m    nginx-ingress-controller  Configuration for default/cafe-ingress was added or updated
    ```
    In this case, the config was successfully applied.

### Step 3 -- Take a Look at the Generated NGINX Config

Take a look at the generated config for the cafe-ingress Ingress resource to see how the rate-limiting feature is enabled:
```
$ kubectl exec <nginx-ingress-pod> -n nginx-ingress -- cat /etc/nginx/conf.d/default-cafe-ingress.conf
```

```nginx
# configuration for default/cafe-ingress

. . .

limit_req_zone $binary_remote_addr zone=default-cafe-ingress:10m rate=5r/s;

server {
  listen 80;

  . . .

  location /tea {

    limit_req zone=default-cafe-ingress burst=1 nodelay;

    . . .
  }

  location /coffee {

    limit_req zone=default-cafe-ingress burst=1 nodelay;

    . . .
  }

. . .
}
```
**Note**: the unimportant parts are replaced with `. . .`.
# Custom HTTP Listeners

In this example, we will configure a [VirtualServer](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/) resource with custom HTTP listeners.
This will allow HTTP and/or HTTPs based requests to be made on non-default ports.

## Prerequisites

1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/)
   instructions to deploy the Ingress Controller with custom resources enabled.
2. Ensure the Ingress Controller is configured with the `-global-configuration` argument:

   ```console
   args:
      - -global-configuration=$(POD_NAMESPACE)/nginx-configuration
   ```
3. Save the public IP address of the Ingress Controller into a shell variable:

    ```console
    IC_IP=XXX.YYY.ZZZ.III
    ```

## Step 1 - Deploy the GlobalConfiguration resource
Similar to how listeners are configured in our [basic-tcp-udp](../../examples/custom-resource/basic-tcp-udp) examples,
here we will deploy a GlobalConfiguration resource with the listeners we want to use in our VirtualServer.

```yaml
apiVersion: k8s.nginx.org/v1alpha1
kind: GlobalConfiguration
metadata:
  name: nginx-configuration
  namespace: nginx-ingress
spec:
  listeners:
  - name: http-8083
    port: 8083
    protocol: HTTP
  - name: https-8443
    port: 8443
    protocol: HTTP
    ssl: true
```

```console
kubectl create -f global-configuration.yaml
```

## Step 2 - Save the custom port numbers
Save the custom HTTP and/or HTTPs ports into a shell variables for later:
```console
IC_HTTP_PORT=8083
IC_HTTPS_PORT=8443
```

## Step 3 - Deploy the Cafe Application

Create the coffee and the tea deployments and services:

```console
kubectl create -f cafe.yaml
```

## Step 4 - Deploy the VirtualServer with custom listeners
The VirtualServer in this example is set to use the listeners defined in the GlobalConfiguration resource
that was deployed in Step 1. Below is the yaml of this example VirtualServer:

```yaml
apiVersion: k8s.nginx.org/v1
kind: VirtualServer
metadata:
  name: cafe
spec:
  listener:
    http: http-8083
    https: https-8443
  host: cafe.example.com
  tls:
    secret: cafe-secret
  upstreams:
  - name: tea
    service: tea-svc
    port: 80
  - name: coffee
    service: coffee-svc
    port: 80
  routes:
  - path: /tea
    action:
      pass: tea
  - path: /coffee
    action:
      pass: coffee
```

1. Create the secret with the TLS certificate and key:

    ```console
    kubectl create -f cafe-secret.yaml
    ```

2. Create the VirtualServer resource:

    ```console
    kubectl create -f cafe-virtual-server.yaml
    ```

## Step 5 - Test the Configuration

1. Check that the configuration has been successfully applied by inspecting the events of the VirtualServer:

    ```console
    kubectl describe virtualserver cafe
    ```

   Below you will see the events as well as the new `Listeners` field
    ```console
   . . .
   Spec:
      Host:  cafe.example.com
      Listener:
         Http:   http-8083
         Https:  https-8443
   . . .
   Routes:
    . . .
    Events:
      Type    Reason          Age   From                      Message
      ----    ------          ----  ----                      -------
      Normal  AddedOrUpdated  7s    nginx-ingress-controller  Configuration for default/cafe was added or updated
    ```

2. Access the application using curl. We'll use curl's `--insecure` option to turn off certificate verification of our
   self-signed certificate and `--resolve` option to set the IP address and HTTPS port of the Ingress Controller to the
   domain name of the cafe application.

    To get coffee:

    ```console
    curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP https://cafe.example.com:$IC_HTTPS_PORT/coffee --insecure
    ```

    ```text
    Server address: 10.16.1.182:80
    Server name: coffee-7dbb5795f6-tnbtq
    ...
    ```

    If your prefer tea:

    ```console
    curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP https://cafe.example.com:$IC_HTTPS_PORT/tea --insecure
    ```

    ```text
    Server address: 10.16.0.149:80
    Server name: tea-7d57856c44-zlftd
    ...

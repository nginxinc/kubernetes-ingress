# Custom IPv4 and IPv6 IP Listeners

In this example, we will configure a VirtualServer resource with a custom IPv4 or IPv6 IP using HTTP/HTTPS listeners.
This will allow IPv4 and/or IPv6 IPs using HTTP and/or HTTPS based requests to be made on non-default ports using separate IPs.

## Prerequisites

1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/)
   instructions to deploy the Ingress Controller with custom resources enabled.
2. Ensure the Ingress Controller is configured with the `-global-configuration` argument:

   ```console
   args:
      - -global-configuration=$(POD_NAMESPACE)/nginx-configuration
   ```

3. If you have a NodePort or Loadbalancer service deployed, ensure they are updated to include the custom listener ports.
Example YAML for a LoadBalancer:

   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: nginx-ingress
     namespace: nginx-ingress
   spec:
     type: LoadBalancer
     ports:
     - port: 8083
       targetPort: 8083
       protocol: TCP
       name: ip-listener-1-http
     - port: 8443
       targetPort: 8443
       protocol: TCP
       name: ip-listener-2-https
     selector:
       app: nginx-ingress
   ```

## Step 1 - Deploy the GlobalConfiguration resource

Similar to how listeners are configured in our [custom-listeners](../../custom-listeners) examples,
here we deploy a GlobalConfiguration resource with the listeners we want to use in our VirtualServer.

   ```yaml
apiVersion: k8s.nginx.org/v1
kind: GlobalConfiguration
metadata:
  name: nginx-configuration
  namespace: nginx-ingress
spec:
  listeners:
  - name: ip-listener-1-http
    port: 8083
    protocol: HTTP
    ipv4ip: 127.0.0.1
  - name: ip-listener-2-https
    port: 8443
    protocol: HTTP
    ipv4ip: 127.0.0.2
    ipv6ip: ::1
    ssl: true
   ```

   ```console
   kubectl create -f global-configuration.yaml
   ```

## Step 2 - Deploy the Cafe Application

Create the coffee and the tea deployments and services:

   ```console
   kubectl create -f cafe.yaml
   ```

## Step 3 - Deploy the VirtualServer with custom listeners

The VirtualServer in this example is set to use the listeners defined in the GlobalConfiguration resource
that was deployed in Step 1. Below is the yaml of this example VirtualServer:

   ```yaml
   apiVersion: k8s.nginx.org/v1
   kind: VirtualServer
   metadata:
     name: cafe
   spec:
     listener:
       http: ip-listener-1-http
       https: ip-listener-2-https
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

## Step 4 - Test the Configuration

1. Check that the configuration has been successfully applied by inspecting the events of the VirtualServer and the GlobalConfiguration:

    ```console
    kubectl describe virtualserver cafe
    ```

    Below you will see the events as well as the new `Listeners` field

    ```console
    . . .
    Spec:
      Host:  cafe.example.com
      Listener:
          Http:   ip-listener-1-http
          Https:  ip-listener-2-https
    . . .
    Routes:
    . . .
    Events:
      Type    Reason          Age   From                      Message
      ----    ------          ----  ----                      -------
      Normal  AddedOrUpdated  2s    nginx-ingress-controller  Configuration for default/cafe was added or updated
    ```

    ```console
    kubectl describe globalconfiguration nginx-configuration -n nginx-ingress
    ```

    ```console
    . . .
    Spec:
      Listeners:
        ipv4ip:    127.0.0.1
        Name:      ip-listener-1-http
        Port:      8083
        Protocol:  HTTP
        ipv4ip:    127.0.0.2
        ipv6ip:    ::1
        Name:      ip-listener-2-https
        Port:      8443
        Protocol:  HTTP
        Ssl:       true
    Events:
      Type    Reason   Age   From                      Message
      ----    ------   ----  ----                      -------
      Normal  Updated  14s   nginx-ingress-controller  GlobalConfiguration nginx-ingress/nginx-configuration was added or updated
    ```

2. Since the deployed VirtualServer is using ports `8083` and `8443` in this example. you must explicitly specify these ports
when sending requests to the endpoints of this VirtualServer:

   For `/coffee` on `8083`:

    ```console
    curl -k http://cafe.example.com:8083/coffee
    ```

    ```text
    Server address: 10.32.0.40:8080
    Server name: coffee-7dd75bc79b-qmhmv
    ...
    URI: /coffee
    ...
    ```

   For `/coffee` on `8443`:

    ```console
    curl -k https://cafe.example.com:8443/coffee
    ```

    ```text
    Server address: 10.32.0.40:8080
    Server name: coffee-7dd75bc79b-qmhmv
    ...
    URI: /coffee
    ...
    ```

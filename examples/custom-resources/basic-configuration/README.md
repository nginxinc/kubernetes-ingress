# Basic Configuration

In this example we configure load balancing with TLS termination for a simple web application using the [VirtualServer](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/) resource. The application, called cafe, lets you get either tea via the tea service or coffee via the coffee service. You indicate your drink preference with the URI of your HTTP request: URIs ending with `/tea` get you tea and URIs ending with `/coffee` get you coffee.

The example is similar to the [complete example](../../examples/complete-example/README.md). However, instead of the Ingress resource, we use the VirtualServer.

## Prerequisites  

1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/) instructions to deploy the Ingress Controller with custom resources enabled.
1. Save the public IP address of the Ingress Controller into a shell variable:
    ```
    $ IC_IP=XXX.YYY.ZZZ.III
    ```
1. Save the HTTPS port of the Ingress Controller into a shell variable:
    ```
    $ IC_HTTPS_PORT=<port number>
    ```

## Step 1 - Deploy the Cafe Application

Create the coffee and the tea deployments and services:
```
$ kubectl create -f cafe.yaml
```

## Step 2 - Configure Load Balancing and TLS Termination

1. Create the secret with the TLS certificate and key:
    ```
    $ kubectl create -f cafe-secret.yaml
    ```

2. Create the VirtualServer resource:
    ```
    $ kubectl create -f cafe-virtual-server.yaml
    ```

## Step 3 - Test the Configuration

1. Check that the configuration has been successfully applied by inspecting the events of the VirtualServer:
    ```
    $ kubectl describe virtualserver cafe
    . . .
    Events:
      Type    Reason          Age   From                      Message
      ----    ------          ----  ----                      -------
      Normal  AddedOrUpdated  7s    nginx-ingress-controller  Configuration for default/cafe was added or updated
    ```
1. Access the application using curl. We'll use curl's `--insecure` option to turn off certificate verification of our self-signed certificate and `--resolve` option to set the IP address and HTTPS port of the Ingress Controller to the domain name of the cafe application:
    
    To get coffee:
    ```
    $ curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP https://cafe.example.com:$IC_HTTPS_PORT/coffee --insecure
    Server address: 10.16.1.182:80
    Server name: coffee-7dbb5795f6-tnbtq
    ...
    ```
    If your prefer tea:
    ```
    $ curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP https://cafe.example.com:$IC_HTTPS_PORT/tea --insecure
    Server address: 10.16.0.149:80
    Server name: tea-7d57856c44-zlftd
    ...
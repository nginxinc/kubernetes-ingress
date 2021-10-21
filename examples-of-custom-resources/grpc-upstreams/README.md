# gRPC support

To support a gRPC application using Virtual server resources with NGINX Ingress controllers, you need to add the **type: grpc** field to an upstream.
The protocol defaults to http if left unset.

## Prerequisites

1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/) instructions to deploy the Ingress Controller with custom resources enabled.
2. Save the public IP address of the Ingress Controller into a shell variable:
    ```
    $ IC_IP=XXX.YYY.ZZZ.III
    ```
3. Save the HTTPS port of the Ingress Controller into a shell variable:
    ```
    $ IC_HTTPS_PORT=<port number>
    ```

## Step 0 - Replace the ConfigMap

* HTTP/2 must be enabled. See `http2` ConfigMap key in the [ConfigMap](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/configmap-resource/#listeners)

```
$ kubectl apply -f nginx-config.yaml
```

## Step 1 - Deploy the Cafe Application

Create the greeter and the grpc-svc deployment and service:
```
$ kubectl create -f grpc-app.yaml
```

## Step 2 - Configure Load Balancing and TLS Termination

1. Create the secret with the TLS certificate and key:
    ```
    $ kubectl create -f vs-grpc-secret.yaml
    ```

2. Create the VirtualServer resource:
    ```
    $ kubectl create -f vs-grpc.yaml
    ```

## Step 3 - Test the Configuration

1. Check that the configuration has been successfully applied by inspecting the events of the VirtualServer:
    ```
    $ kubectl describe virtualserver virtual-server
    . . .
    Events:
      Type    Reason          Age   From                      Message
      ----    ------          ----  ----                      -------
      Normal  AddedOrUpdated  7s    nginx-ingress-controller  Configuration for default/virtual-server was added or updated
    ```
2. Access the application using grpcurl
    
    ```
    $ grpcurl -insecure -d '{\"name\": \"world!\"}' -protoset helloworld.protoset $IC_IP:$IC_HTTPS_PORT helloWorld.greeter/SayHello
    Greeting: Hello world!
    ...
    ```

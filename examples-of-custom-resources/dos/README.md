# DOS

In this example we deploy the NGINX Plus Ingress controller with [NGINX App Protect Dos](https://www.nginx.com/products/nginx-app-protect-dos/), a simple web application and then configure load balancing and DOS protection for that application using the VirtualServer resource.

## Prerequisites

1. Follow the installation [instructions](../../docs/installation.md) to deploy the Ingress controller with NGINX App Protect Dos.
1. Save the public IP address of the Ingress Controller into a shell variable:
    ```
    $ IC_IP=XXX.YYY.ZZZ.III
    ```
1. Save the HTTP port of the Ingress Controller into a shell variable:
    ```
    $ IC_HTTP_PORT=<port number>
    ```

## Step 1. Deploy a Web Application

Create the application deployment and service:
```
$ kubectl apply -f webapp.yaml
```

## Step 2 - Deploy the DOS configuration resources

1. Create the syslog service and pod for the App Protect security logs:
    ```
    $ kubectl apply -f syslog.yaml
    ```
2. Create the Dos protected resource configuration:
    ```
    $ kubectl apply -f apdos-protected.yaml
    ```
3. Create the App Protect Dos policy and log configuration:
    ```
    $ kubectl apply -f apdos-policy.yaml
    $ kubectl apply -f apdos-logconf.yaml
    ```

## Step 3 - Configure Load Balancing

1. Create the VirtualServer Resource:
    ```
    $ kubectl apply -f virtual-server.yaml
    ```
Note the reference to the DOS protected resource in the Policy resource. By specifying the resource it enables DOS protection for the VirtualServer.

## Step 4 - Test the Application

To access the application, curl the Webapp service. We'll use the --resolve option to set the Host header of a request with `webapp.example.com`

1. Send a request to the application:
    ```
    $ curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT/
    Server address: 10.12.0.18:80
    Server name: webapp-7586895968-r26zn
    ...
    ```

1. To check the security logs in the syslog pod:
    ```
    $ kubectl exec -it <SYSLOG_POD> -- cat /var/log/messages
    ```

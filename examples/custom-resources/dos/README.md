# DOS

In this example we deploy the NGINX Plus Ingress Controller with [NGINX App Protect DoS](https://www.nginx.com/products/nginx-app-protect-dos/), a simple web application and then configure load balancing and DOS protection for that application using the VirtualServer resource.

## Prerequisites

1. Follow the installation [instructions](https://docs.nginx.com/nginx-ingress-controller/installation) to deploy the Ingress Controller with NGINX App Protect DoS.
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

1. Create the syslog services and pod for the App Protect security and access logs:
    ```
    $ kubectl apply -f syslog.yaml
    $ kubectl apply -f syslog2.yaml
    ```
2. Create the DoS protected resource configuration:
    ```
    $ kubectl apply -f apdos-protected.yaml
    ```
3. Create the App Protect DoS policy and log configuration:
    ```
    $ kubectl apply -f apdos-policy.yaml
    $ kubectl apply -f apdos-logconf.yaml
    ```

## Step 3 - Configure Load Balancing

1. Create the VirtualServer Resource:
    ```
    $ kubectl apply -f virtual-server.yaml
    ```
Note the reference to the DOS protected resource in the VirtualServer resource. By specifying the resource it enables DOS protection for the VirtualServer.

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
2. To check the access logs in the syslog pod:
    ```
    $ kubectl exec -it <SYSLOG_POD_2> -- cat /var/log/messages
    ```

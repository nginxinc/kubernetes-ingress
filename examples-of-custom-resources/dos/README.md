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

## Step 2 - Deploy the AP Dos Policy

1. Create the syslog service and pod for the App Protect security logs:
    ```
    $ kubectl apply -f syslog.yaml
    ```
1. Create the App Protect Dos policy and log configuration:
    ```
    $ kubectl apply -f apdos-policy.yaml
    $ kubectl apply -f apdos-logconf.yaml
    ```

## Step 3 - Deploy the DOS Policy

1. Update the `logDest` field from `dos.yaml` with the ClusterIP of the syslog service. For example, if the IP is `10.101.21.110`:
    ```yaml
    dos:
        ...
        logDest: "10.101.21.110:514"
    ```

1. Create the DOS policy
    ```
    $ kubectl apply -f dos.yaml
    ```

Note the App Protect Dos configuration settings in the Policy resource. They enable DOS protection by configuring App Protect Dos with the policy and log configuration created in the previous step.

## Step 4 - Configure Load Balancing

1. Create the VirtualServer Resource:
    ```
    $ kubectl apply -f virtual-server.yaml
    ```

Note that the VirtualServer references the policy `dos-policy` created in Step 3.

## Step 5 - Test the Application

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

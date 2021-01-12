# WAF

In this example we deploy the NGINX Plus Ingress controller with [NGINX App Protect](https://www.nginx.com/products/nginx-app-protect/), a simple web application and then configure load balancing and WAF protection for that application using the VirtualServer resource.

## Prerequisites

1. Follow the installation [instructions](../../docs/installation.md) to deploy the Ingress controller with NGINX App Protect.
1. Save the public IP address of the Ingress controller into a shell variable:
    ```
    $ IC_IP=XXX.YYY.ZZZ.III
    ```
1. Save the HTTPS port of the Ingress controller into a shell variable:
    ```
    $ IC_HTTPS_PORT=<port number>
    ```

## Step 1. Deploy a Web Application

Create the application deployment and service:
```
$ kubectl apply -f webapp.yaml
```

## Step 2 - Deploy the AP Policy

1. Create the syslog service and pod for the App Protect security logs:
    ```
    $ kubectl apply -f syslog.yaml
    ```
1. Create the App Protect policy, log configuration and user defined signature:
    ```
    $ kubectl apply -f ap-dataguard-alarm-policy.yaml
    $ kubectl apply -f ap-logconf.yaml
    $ kubectl apply -f ap-apple-uds.yaml
    ```

## Step 3 - Configure Load Balancing

Update the `logDest` field from `virtualserver.yaml` with the ClusterIP of the syslog service. For example, if the IP is `10.101.21.110`:
```yaml
waf:
    ...
    logDest: "syslog:server=10.101.21.110:514"
```

Create the VirtualServer Resource:
```
$ kubectl apply -f virtualserver.yaml
```
Note the App Protect configuration settings in the Policy resource. They enable WAF protection by configuring App Protect with the policy and log configuration created in the previous step.

## Step 4 - Test the Application

1. To access the application, curl the coffee and the tea services. We'll use `curl`'s `--insecure` option to turn off certificate verification of our self-signed
certificate and the --resolve option to set the Host header of a request with `webapp.example.com`

    Send a request to the application:
    ```
    $ curl --resolve webapp.example.com:$IC_HTTPS_PORT:$IC_IP https://webapp.example.com:$IC_HTTPS_PORT/ --insecure
    Server address: 10.12.0.18:80
    Server name: webapp-7586895968-r26zn
    ...
    ```

    Now, let's try to send a request with a suspicious URL:
    ```
    $ curl --resolve webapp.example.com:$IC_HTTPS_PORT:$IC_IP "https://webapp.example.com:$IC_HTTPS_PORT/<script>" --insecure
    <html><head><title>Request Rejected</title></head><body>
    ...
    ```
    Lastly, let's try to send some suspicious data that matches the user defined signature.
    ```
    $ curl --resolve webapp.example.com:$IC_HTTPS_PORT:$IC_IP -X POST -d "apple" "https://webapp.example.com:$IC_HTTPS_PORT/" --insecure
    <html><head><title>Request Rejected</title></head><body>
    ...
    ```
    As you can see, the suspicious requests were blocked by App Protect

1. To check the security logs in the syslog pod:
    ```
    $ kubectl exec -it <SYSLOG_POD> -- cat /var/log/messages
    ```

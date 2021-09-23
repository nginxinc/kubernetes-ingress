# NGINX App Protect Dos Support

In this example we deploy the NGINX Plus Ingress controller with [NGINX App Protect Dos](https://www.nginx.com/products/nginx-app-protect-dos/), a simple web application and then configure load balancing and DOS protection for that application using the Ingress resource.

## Running the Example

## 1. Deploy the Ingress Controller

1. Follow the installation [instructions](../../docs/installation.md) to deploy the Ingress controller with NGINX App Protect Dos.

2. Save the public IP address of the Ingress controller into a shell variable:
    ```
    $ IC_IP=XXX.YYY.ZZZ.III
    ```
3. Save the HTTPS port of the Ingress controller into a shell variable:
    ```
    $ IC_HTTPS_PORT=<port number>
    ```

## 2. Deploy the Webapp Application

Create the webapp deployment and service:
```
$ kubectl create -f webapp.yaml
```

## 3. Configure Load Balancing
1. Create the syslog service and pod for the App Protect Dos security logs:
    ```
    $ kubectl create -f syslog.yaml
    ```
2. Create a secret with an SSL certificate and a key:
    ```
    $ kubectl create -f webapp-secret.yaml
    ```
3. Create the App Protect Dos Protected Resource:
    ```
    $ kubectl create -f apdos-protected.yaml
    ```
4. Create the App Protect Dos policy and log configuration:
    ```
    $ kubectl create -f apdos-policy.yaml
    $ kubectl create -f apdos-logconf.yaml
    ```
5. Create an Ingress Resource:

    ```
    $ kubectl create -f webapp-ingress.yaml
    ```
    Note the App Protect Dos annotation in the Ingress resource. This enables DOS protection by specifying the DOS protected resource configuration that applies to this Ingress.

## 4. Test the Application

1. To access the application, curl the Webapp service. We'll use `curl`'s --insecure option to turn off certificate verification of our self-signed
certificate and the --resolve option to set the Host header of a request with `webapp.example.com`

    Send a request to the application::
    ```
    $ curl --resolve webapp.example.com:$IC_HTTPS_PORT:$IC_IP https://webapp.example.com:$IC_HTTPS_PORT/ --insecure
    Server address: 10.12.0.18:80
    Server name: coffee-7586895968-r26zn
    ...
    ```
1. To check the security logs in the syslog pod:
    ```
    $ kubectl exec -it <SYSLOG_POD> -- cat /var/log/messages
    ```

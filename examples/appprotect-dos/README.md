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
3. Create the App Protect Dos policy and log configuration:
    ```
    $ kubectl create -f apdos-policy.yaml
    $ kubectl create -f apdos-logconf.yaml
    ```
4. Create an Ingress Resource:

    Update the `appprotectdos.f5.com/app-protect-dos-security-log-destination` annotation from `webapp-ingress.yaml` with the ClusterIP of the syslog service. For example, if the IP is `10.101.21.110`:
    ```yaml
    . . .
    appprotect.f5.com/app-protect-security-log-destination: "10.101.21.110:514"
    ```
    Create the Ingress Resource:
    ```
    $ kubectl create -f webapp-ingress.yaml
    ```
    Note the App Protect Dos annotations in the Ingress resource. They enable DOS protection by configuring App Protect Dos with the policy and log configuration created in the previous step.

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

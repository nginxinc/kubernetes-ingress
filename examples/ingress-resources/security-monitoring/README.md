# NGINX Security Monitoring

In this example we deploy the NGINX Plus Ingress Controller with [NGINX App
Protect](https://www.nginx.com/products/nginx-app-protect/) and [NGINX Agent](https://docs.nginx.com/nginx-agent/overview/) in order to integrate with [NGINX Management Suite Security Monitoring](https://docs.nginx.com/nginx-management-suite/security/). We also deploy a simple web application and then configure load balancing and WAF protection for that application using the Ingress resource. We configure logging for NGINX App Protect to send logs to the NGINX Agent syslog listener which is then sent to the Security Monitoring dashboard in NGINX Instance Manager.

## Running the Example

## 1. Deploy the Ingress Controller

1. Follow the installation [instructions](https://docs.nginx.com/nginx-ingress-controller/installation) to deploy the
   Ingress Controller with NGINX App Protect and NGINX Agent. Configure NGINX Agent to connect to a deployment of NGINX Instance Manager with Security Monitoring, and verify that your NGINX Ingress Controller deployment is online in NGINX Instance Manager.

2. Save the public IP address of the Ingress Controller into a shell variable:

    ```console
    IC_IP=XXX.YYY.ZZZ.III
    ```

3. Save the HTTPS port of the Ingress Controller into a shell variable:

    ```console
    IC_HTTPS_PORT=<port number>
    ```

## 2. Deploy the Cafe Application

Create the coffee and the tea deployments and services:

```console
kubectl create -f cafe.yaml
```

## 3. Configure Load Balancing

1. . Create a secret with an SSL certificate and a key:

    ```console
    kubectl create -f cafe-secret.yaml
    ```

2. Create the App Protect policy, log configuration and user defined signature:

    ```console
    kubectl create -f ap-dataguard-alarm-policy.yaml
    kubectl create -f ap-logconf.yaml
    kubectl create -f ap-apple-uds.yaml
    ```

    Note the log configuration in `ap-logconf.yaml` is a specific format required by NGINX Agent for integration with Security Monitoring.

3. Create an Ingress Resource:

    ```console
    kubectl create -f cafe-ingress.yaml
    ```

    Note the App Protect annotations in the Ingress resource. They enable WAF protection by configuring App Protect with
    the policy and log configuration created in the previous step.

## 4. Test the Application

1. To access the application, curl the coffee and the tea services. We'll use `curl`'s --insecure option to turn off
certificate verification of our self-signed certificate and the --resolve option to set the Host header of a request
with `cafe.example.com`

    To get coffee:

    ```console
    curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP https://cafe.example.com:$IC_HTTPS_PORT/coffee --insecure
    ```

    ```text
    Server address: 10.12.0.18:80
    Server name: coffee-7586895968-r26zn
    ...
    ```

    If your prefer tea:

    ```console
    curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP https://cafe.example.com:$IC_HTTPS_PORT/tea --insecure
    ```

    ```text
    Server address: 10.12.0.19:80
    Server name: tea-7cd44fcb4d-xfw2x
    ...
    ```

    Now, let's try to send a request with a suspicious URL:

    ```console
    curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP "https://cafe.example.com:$IC_HTTPS_PORT/tea/<script>" --insecure
    ```

    ```text
    <html><head><title>Request Rejected</title></head><body>
    ...
    ```

    Lastly, let's try to send some suspicious data that matches the user defined signature.

    ```console
    curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP -X POST -d "apple" "https://cafe.example.com:$IC_HTTPS_PORT/tea/" --insecure
    ```

    ```text
    <html><head><title>Request Rejected</title></head><body>
    ...
    ```

    As you can see, the suspicious requests were blocked by App Protect

1. Access the Security Monitoring dashboard in your deployment of NGINX Instance Manager to view details for the blocked requests.

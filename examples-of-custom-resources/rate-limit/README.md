# Rate Limit

In this example, we deploy a web application, configure load balancing for it via a VirtualServer, and apply a rate limit policy.

## Prerequisites

1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/) instructions to deploy the Ingress Controller.
1. Save the public IP address of the Ingress Controller into a shell variable:
    ```
    $ IC_IP=XXX.YYY.ZZZ.III
    ```
1. Save the HTTP port of the Ingress Controller into a shell variable:
    ```
    $ IC_HTTP_PORT=<port number>
    ```

## Step 1 - Deploy a Web Application

Create the application deployment and service:
```
$ kubectl apply -f webapp.yaml
```

## Step 2 - Deploy the Rate Limit Policy

In this step, we create a policy with the name `rate-limit-policy` that allows only 1 request per second coming from a single IP address.

Create the policy:
```
$ kubectl apply -f rate-limit.yaml
```

## Step 3 - Configure Load Balancing

Create a VirtualServer resource for the web application:
```
$ kubectl apply -f virtual-server.yaml
```

Note that the VirtualServer references the policy `rate-limit-policy` created in Step 2.

## Step 4 - Test the Configuration

Let's test the configuration. If you access the application at a rate that exceeds one request per second, NGINX will start rejecting your requests:
```
$ curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT/
Server address: 10.8.1.19:8080
Server name: webapp-dc88fc766-zr7f8
. . .

$ curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT/
<html>
<head><title>503 Service Temporarily Unavailable</title></head>
<body>
<center><h1>503 Service Temporarily Unavailable</h1></center>
<hr><center>nginx/1.19.1</center>
</body>
</html>
```

> Note: The command result is truncated for the clarity of the example.

# JWT

In this example, we deploy a web application, configure load balancing for it via a VirtualServer, and apply a JWT policy.

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

## Step 2 - Deploy the JWK Secret

Create a secret with the name `jwk-secret` that will be used for JWT validation:
```
$ kubectl apply -f jwk-secret.yaml
```

## Step 3 - Deploy the JWT Policy

Create a policy with the name `jwt-policy` that references the secret from the previous step and only permits requests to our web application that contain a valid JWT:
```
$ kubectl apply -f jwt.yaml
```

## Step 4 - Configure Load Balancing

Create a VirtualServer resource for the web application:
```
$ kubectl apply -f virtual-server.yaml
```

Note that the VirtualServer references the policy `jwt-policy` created in Step 3.

## Step 5 - Test the Configuration

If you attempt to access the application without providing a valid JWT, NGINX will reject your requests for that VirtualServer:
```
$ curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT/
<html>
<head><title>401 Authorization Required</title></head>
<body>
<center><h1>401 Authorization Required</h1></center>
<hr><center>nginx/1.19.1</center>
</body>
</html>
```

If you provide a valid JWT, your request will succeed:
```
$ curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT/ -H "token: `cat token.jwt`"
Server address: 172.17.0.3:8080
Server name: webapp-7c6d448df9-lcrx6
Date: 10/Sep/2020:18:20:03 +0000
URI: /
Request ID: db2c07ce640755ccbe9f666d16f85620
```

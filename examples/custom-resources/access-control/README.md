# Access Control

In this example, we deploy a web application; configure load balancing for it via a VirtualServer; and apply access
control policies to deny and allow traffic from a specific subnet.

## Prerequisites

1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/)
   instructions to deploy the Ingress Controller.
1. Save the public IP address of the Ingress Controller into a shell variable:

    ```console
    IC_IP=XXX.YYY.ZZZ.III
    ```

1. Save the HTTP port of the Ingress Controller into a shell variable:

    ```console
    IC_HTTP_PORT=<port number>
    ```

## Step 1 - Deploy a Web Application

Create the application deployment and service:

```console
kubectl apply -f webapp.yaml
```

## Step 2 - Deploy an Access Control Policy

In this step, we create a policy with the name `webapp-policy` that denies requests from clients with an IP that belongs
to the subnet `10.0.0.0/8`. This is the subnet that our test client in Steps 4 and 6 will belong to. Make sure to change
the `deny` field of the `access-control-policy-deny.yaml` according to your environment (use the subnet of your
machine).

Create the policy:

```console
kubectl apply -f access-control-policy-deny.yaml
```

## Step 3 - Configure Load Balancing

Create a VirtualServer resource for the web application:

```console
kubectl apply -f virtual-server.yaml
```

Note that the VirtualServer references the policy `webapp-policy` created in Step 2.

## Step 4 - Test the Configuration

Let's access the application:

```console
curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT
```

```text
<html>
<head><title>403 Forbidden</title></head>
<body>
<center><h1>403 Forbidden</h1></center>
<hr><center>nginx/1.17.9</center>
</body>
</html>
```

We got a 403 response from NGINX, which means that our policy successfully blocked our request.

## Step 5 - Update the Policy

In this step, we update the policy to allow requests from clients from the subnet `10.0.0.0/8`. Make sure to change the
`allow` field of the `access-control-policy-allow.yaml` according to your environment.

Update the policy:

```console
kubectl apply -f access-control-policy-allow.yaml
```

## Step 6 - Test the Configuration

Let's access the application again:

```console
curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT
```

```text
Server address: 10.64.0.13:8080
Server name: webapp-5cbbc7bd78-wf85w
```

In contrast with Step 4, we got a 200 response, which means that our updated policy successfully allowed our request.

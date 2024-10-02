---
title: Deploy an Access Control policy
weight: 900
toc: true
docs: DOCS-000
---

This topic describes how to apply and update an Access Control policy with F5 NGINX Ingress Controller.

It demonstrates this using an example application and a [VirtualServer custom resource]({{< ref "/configuration/virtualserver-and-virtualserverroute-resources.md" >}})

---

## Before you begin

You should have a [working NGINX Ingress Controller]({{< ref "/installation/installing-nic/installation-with-helm.md" >}}) instance.

For ease of use in shell commands, set two shell variables:

1. The public IP address for your NGINX Ingress Controller instance.

```shell
IC_IP=<ip-address>
```

2. The HTTP port of the same instance

```shell
IC_HTTP_PORT=<port number>
```

---

## Deploy the example application

Create the file *webapp.yaml* with the following contents:

{{< ghcode "https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/refs/heads/main/examples/custom-resources/access-control/webapp.yaml" >}}

Apply it using *kubectl*:

```shell
kubectl apply -f webapp.yaml
```

---

## Deploy an Access Control policy

Create a file named *access-control-policy-deny.yaml*. The highlighted *deny* field will be used by the example application, and should be changed to the subnet of your machine.

{{< ghcode "https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/refs/heads/main/examples/custom-resources/access-control/access-control-policy-deny.yaml" "hl_lines=7-8" >}}

Apply the policy:

```shell
kubectl apply -f access-control-policy-deny.yaml
```

---

## Configure load balancing

Create a file named *virtual-server.yaml* for the VirtualServer resource. The *policies* field references the example application.

{{< ghcode "https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/refs/heads/main/examples/custom-resources/access-control/virtual-server.yaml" "hl_lines=7-8" >}}

Apply the policy:

```shell
kubectl apply -f virtual-server.yaml
```

---

## Test the example application

Use *curl* to attempt to access the application:

```shell
curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT
```
```text
<html>
<head><title>403 Forbidden</title></head>
<body>
<center><h1>403 Forbidden</h1></center>
</body>
</html>
```

The *403* response is expected, successfully blocking your machine.

---

## Update the policy

Create a new policy with the file *access-control-policy-allow.yaml*, updating the *allow* field to the subnet of your machine.

{{< ghcode "https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/refs/heads/main/examples/custom-resources/access-control/access-control-policy-allow.yaml" "hl_lines=7-8" >}}

Apply the policy:

```shell
kubectl apply -f access-control-policy-allow.yaml
```

----

## Verify the policy update

Attempt to access the application again:

```shell
curl --resolve webapp.example.com:$IC_HTTP_PORT:$IC_IP http://webapp.example.com:$IC_HTTP_PORT
```
```text
Server address: 10.64.0.13:8080
Server name: webapp-5cbbc7bd78-wf85w
```

The successful response demonstrates that the policy has been updated.

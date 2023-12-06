# Support for Backup Directive in Virtual Transport Server

The Ingress Controller supports routing requests to services of the type
[ExternalName](https://kubernetes.io/docs/concepts/services-networking/service/#externalname).

An ExternalName service is defined by an external DNS name that is resolved into the IP addresses, typically external to
the cluster. This enables to use the Ingress Controller to route requests to the destinations outside of the cluster.

**Note:** This feature is only available in NGINX Plus.

## Prerequisites

For illustration purposes, we will run NGINX Ingress Controller (referred to as NIC in the examples) with the
```-watch-namespace=nginx-ingress,default``` option. The option enables NIC to watch selected namespaces.

Any application deployed in other namespaces will be treated as an external service.

We will use the ```examples/custom-resources/tls-passthrough``` application example as our backend app that will be
responding to requests.

## Example NIC OSS

### 1. Deploy the cafe application

1. Deploy the cafe backend application

  ```shell
  kubectl apply -f cafe.yaml
  ```

### 2. Deploy resolver

1. Deploy the resolver

    ```shell
    kubectl apply -f nginx-config.yaml
    ```

### 3. Deploy Backup Service

1. Deploy the backup service of type external name

    ```shell
    kubectl apply -f backup-svc.yaml
    ```

### 4. Deploy NGINX Ingress Controller (OSS)

1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/)
   instructions to deploy the Ingress Controller with custom resources enabled.

1. Save the public IP address of the Ingress Controller into a shell variable:

    ```console
    IC_IP=XXX.YYY.ZZZ.III
    ```

1. Save the HTTPS port of the Ingress Controller into a shell variable:

    ```console
    IC_HTTPS_PORT=<port number>
    ```

    Response

    ```console
    hello from pod secure-app-external-backend-5fbf4fb494-x7bkl
    ```

## Example NIC Plus

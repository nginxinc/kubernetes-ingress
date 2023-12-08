# Support for Backup Directive in in Transport Server

NGINX Ingress Controller supports routing requests to a service specified as `Backup`.
The `backup` service is of type
[ExternalName](https://kubernetes.io/docs/concepts/services-networking/service/#externalname).

## Prerequisites

For illustration purposes, we will run NGINX Ingress Controller (referred to as NIC in the examples) with the following options:

```shell
      ...
    - -enable-custom-resources
    - -enable-tls-passthrough
    - -watch-namespace=nginx-ingress,default
      ...
```

The option `-watch-namespace` enables NIC to watch selected namespaces. Any application deployed in other namespaces
will be treated as an external service.

We will use two ```examples/custom-resources/tls-passthrough``` applications example as our backend app that will be
responding to requests. First application will be deployed in the `default` namespace, second application will
be deployed in the `external-ns` namespace.

## Example NIC Plus

### 1. Deploy ConfigMap with defined resolver

1. Deploy resolver

  ```shell
  kubectl create -f nginx-config-resolver.yaml
  ```

### 2. Deploy Backup ExternalName service

1. Deploy Backup service

  ```shell
  kubectl create -f backup-svc-ts.yaml
  ```

### 3. Deploy TransportServer

1. Deploy TransportServer. Note that the server uses not default load balancing method.

  ```shell
  kubectl create -f transport-server-passthrough.yaml
  ```

### 4. Deploy the tls-passthrough application

1. Deploy tls-passthrough application

  ```shell
  kubectl create -f secure-app.yaml
  ```

Make sure the application works as described in the ```examples/custom-resources/tls-passthrough``` example.

### 5. Deploy the second tls-passthrough aplication to the external namespace

1. Create the external namespace `external-ns`.

  ```shell
  kubectl create -f external-app-ns/external-ns.yaml
  ```

1. Deploy backend application to external namespace (```external-ns```). Note that the NIC is not watching the namespace.

    ```shell
    kubectl apply -f external-app-ns/external-secure-app.yaml
    ```

### 6. Verify the backup service

1. Scale down `secure-app` deployment to 0

    ```shell
    kubectl scale deployment --replicas=0
    ```

1. Verify if the application is working by sending a request and check if the response is coming from the "external
   backend pod" (refer to to the tls-passthrough example)

    ```shell
    curl --resolve app.example.com:$IC_HTTPS_PORT:$IC_IP https://app.example.com:$IC_HTTPS_PORT --insecure
    ```

    Response

    ```shell
    hello from pod secure-app-external-backend-5fbf4fb494-x7bkl
    ```

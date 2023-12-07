# Support for Backup Directive in in Transport Server

The Ingress Controller supports routing requests to a service specified as `Backup`.
The `backup` service is of type
[ExternalName](https://kubernetes.io/docs/concepts/services-networking/service/#externalname).

## Prerequisites

For illustration purposes, we will run NGINX Ingress Controller (referred to as NIC in the examples) with the
```-watch-namespace=nginx-ingress,default``` option. The option enables NIC to watch selected namespaces.

Any application deployed in other namespaces will be treated as an external service.

We will use the ```examples/custom-resources/tls-passthrough``` application example as our backend app that will be
responding to requests.

## Example NIC Plus

### 1. Deploy the tls-passthrough application

1. Deploy the backend application as described in the ```examples/custom-resources/tls-passthrough``` example,
and make sure it is working as described. The application must use one of supported load balancing
 methods required by the backup service.

TODO:

update loadBalancing method - the backup service of type ExternalName does not work with the following
load balancing methods: `hash`, 'round_robin', ..... add here methods

### 2. Deploy the second aplication to external namespace

1. Deploy backend application to external namespace (```external-ns```). Note that the namespace is not being watched by
   ```NIC```.

    ```console
    kubectl apply -f secure-app-external.yaml
    ```

### 3. Setup ExternalName service

1. Create the service of type ```ExternalName```

    ```console
    kubectl apply -f externalname-svc.yaml
    ```

2. Apply the config map

    ```console
    kubectl apply -f nginx-config.yaml
    ```

### 4. Change the Transport Server to point to the ExternalName and verify if it is working correctly

1. Navigate to the tls-passthrough example ```examples/custom-resources/tls-passthrough``` and open the
   ```transport-server-passthrough.yaml``` file.

2. Replace the service name ```secure-app``` with ```externalname-service``` and apply the change.

    ```yaml
    apiVersion: k8s.nginx.org/v1alpha1
    kind: TransportServer
    metadata:
      name: secure-app
    spec:
      listener:
        name: tls-passthrough
        protocol: TLS_PASSTHROUGH
      host: app.example.com
      upstreams:
      - name: secure-app
        service: externalname-service
        port: 8443
      action:
        pass: secure-app
    ```

    ```console
    kubectl apply -f transport-server-passthrough.yaml
    ```

3. Verify if the application is working by sending a request and check if the response is coming from the "external
   backend pod" (refer to to the tls-passthrough example)

    ```console
    curl --resolve app.example.com:$IC_HTTPS_PORT:$IC_IP https://app.example.com:$IC_HTTPS_PORT --insecure
    ```

    Response

    ```console
    hello from pod secure-app-external-backend-5fbf4fb494-x7bkl
    ```

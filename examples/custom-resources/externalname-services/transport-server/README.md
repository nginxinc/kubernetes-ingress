# Support for Type ExternalName Services

The Ingress Controller supports routing requests to services of the type [ExternalName](https://kubernetes.io/docs/concepts/services-networking/service/#externalname).

An ExternalName service is defined by an external DNS name that is resolved into the IP addresses, typically external to the cluster. This enables to use the Ingress Controller to route requests to the destinations outside of the cluster.

**Note:** This feature is only available in NGINX Plus.

# Prerequisites


For the illustration purpose we will run NGINX Ingress Controller (refered as NIC in the examples) with the ```-watch-namespace=nginx-ingress,default``` option. The option enables NIC to watch selected namespaces.

Any application deployed in other namespaces will be treated as an external service.

We will use the ```examples/custom-resources/tls-passthrough``` application example as our backend app that will be responding to requests.

# Example

## 1. Deploy the tls-passthrough application

1. Deploy the backend application as described in the ```examples/custom-resources/tls-passthrough``` example, and make sure it is working as described.

## 2. Deploy external service to external namespace

1. Navigate to the external-name example ```examples/custom-resources/externalname-services/transport-server```

2. Deploy external namespace (```external-ns```) and the backend application. Note that the namespace is not being watched by ```NIC```
    ```
    $ kubectl apply -f transport-server/secure-app-external.yaml
    ```

## 3. Setup ExternalName service

1. Refer the newly created service in the file ```examples/custom-resources/externalname-services/externalname-svc.yaml``` in the spec section
    ```yaml
    kind: Service
    apiVersion: v1
    metadata:
      name: externalname-service
    spec:
      type: ExternalName
      externalName: secure-app-external-backend-svc.external-ns.svc.cluster.local
    ```

2. Create the service of type ```ExternalName```
    ```
    $ kubectl apply -f externalname-svc.yaml
    ```

3. Update config map ```examples/custom-resources/externalname-services/nginx-config.yaml``` with the resolver address
    ```yaml
    kind: ConfigMap
    apiVersion: v1
    metadata:
      name: nginx-config
      namespace: nginx-ingress
    data:
      resolver-addresses: "kube-dns.kube-system.svc.cluster.local"
    ```

4. Apply the change
    ```bash
    $ kubectl apply -f nginx-config.yaml
    ```

## 4. Change the TS to point to the ExternalName and verify if it is working correctly

1. Navigate to the tls-passthrough example ```examples/custom-resources/tls-passthrough``` and open the ```transport-server-passthrough.yaml``` file.

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

    ```
    $ kubectl apply -f transport-server-passthrough.yaml
    ```

3. Verify if the application is working by sending a request and check if the response is coming from the "external backend pod" (refer to to the tls-passthrough example)
    ```bash
    $ curl --resolve app.example.com:$IC_HTTPS_PORT:$IC_IP https://app.example.com:$IC_HTTPS_PORT --insecure
    ```
    Response
    ```
    hello from pod secure-app-external-backend-5fbf4fb494-x7bkl
    ```

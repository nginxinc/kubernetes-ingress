# Support for Type ExternalName Services
The Ingress Controller supports routing requests to services of the type [ExternalName](https://kubernetes.io/docs/concepts/services-networking/service/#externalname).

An ExternalName service is defined by an external DNS name that is resolved into the IP addresses, typically external to the cluster. This enables to use the Ingress Controller to route requests to the destinations outside of the cluster.

**Note:** This feature is only available in NGINX Plus.

# Prerequisites:

For the illustration purpose we will run NGINX Ingress Controller with the ```- -watch-namespace=nginx-ingress,default``` option. The option enables NIC to watch selected namespaces.

We will use the tls-passthrough application example as our backend app that will be responding to requests.

## Steps

- Deploy the backend application as described in the ```examples/custom-resources/tls-passthrough```, and make sure it is working as described.

- Navigate to the external-name example ```examples/custom-resources/externalname-services```

- Deploy backend application to the ```external-ns```. Note that the namespace is not being watched by ```KIC```

```bash
kubectl apply -f transport-server/secure-app-external.yaml
```

- Refer the newly created service in the external name svc ```secure-app-external-backend-svc``` in the spec section

```yaml
kind: Service
apiVersion: v1
metadata:
  name: externalname-service
spec:
  type: ExternalName
  externalName: secure-app-external-backend-svc.external-ns.svc.cluster.local
```

- Create the service

```bash
kubectl apply -f externalname-svc.yaml
```

- Update config map ```nginx-config.yaml``` with the resolver address.

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
  namespace: nginx-ingress
data:
  resolver-addresses: "kube-dns.kube-system.svc.cluster.local"
```

- Apply the change

```bash
kubectl apply -f nginx-config.yaml
```

- Add the ```externalname-service``` to the TransportServer deployed in the tls-passthrough example

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

- Apply the change

```bash
kubectl apply -f transport-server-passthrough.yaml
```

- Send a request to verify the response is comming from the "external" pod (refer to to the tls-passthrough example)

```bash
curl --resolve app.example.com:$IC_HTTPS_PORT:$IC_IP https://app.example.com:$IC_HTTPS_PORT --insecure
```

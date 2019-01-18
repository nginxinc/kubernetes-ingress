# Support for Type ExternalName Services
The Ingress Controller supports routing requests to services of the type [ExternalName](https://kubernetes.io/docs/concepts/services-networking/service/#externalname).
 
An ExternalName service is defined by an external DNS name that is resolved into the IP addresses, typically external to the cluster. This enables to use the Ingress Controller to route requests to the destinations outside of the cluster. 

**Note:** This feature is only available in NGINX Plus.


## Prerequisites
To use ExternalName services, first you need to configure one or more resolvers using the ConfigMap. NGINX Plus will use those resolvers to resolve DNS names of the services.

For example, the following ConfigMap configures one resolver:

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
  namespace: nginx-ingress
data:
  resolver-addresses: "10.0.0.10"
```

Additional resolver parameters, including the caching of DNS records, are available. Check the corresponding [ConfigMap and Annotations](../../docs/configmap-and-annotations.md) section.


## Example
In the following yaml file we define an ExternalName service with the name my-service:

```yaml
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  type: ExternalName
  externalName: my.service.example.com
```

In the following Ingress resource we use my-service:

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: example-ingress
  annotations:
    kubernetes.io/ingress.class: "nginx"
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        backend:
          serviceName: my-service
          servicePort: 80

```

As a result, NGINX Plus will route requests for “example.com” to the IP addresses behind the DNS name my.service.example.com.
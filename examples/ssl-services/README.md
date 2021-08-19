# SSL Services Support

To enable HTTPS or gRPC over SSL when connecting to the endpoints of services, you need to add the **nginx.org/ssl-services** annotation to your Ingress resource definition. The annotation specifies which services require SSL. The annotation syntax is as follows:

```
nginx.org/ssl-services: "service1[,service2,...]"
```

In the following example we load balance three applications, one of which requires HTTPS:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/ssl-services: "ssl-svc"
spec:
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea
        pathType: Prefix
        backend:
          service:
            name: tea-svc
            port:
              number: 80
      - path: /coffee
        pathType: Prefix
        backend:
          service:
            name: coffee-svc
            port:
              number: 80
      - path: /ssl
        pathType: Prefix
        backend:
          service:
            name: ssl-svc
            port:
              number: 443
```
*ssl-svc* is a service for an HTTPS application. The service becomes available at the `/ssl` path. Note how we used the **nginx.org/ssl-services** annotation.

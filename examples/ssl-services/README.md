# SSL Services Support

To enable HTTPS or gRPC over SSL when connecting to the endpoints of services, you need to add the **nginx.org/ssl-services** annotation to your Ingress resource definition. The annotation specifies which services require SSL. The annotation syntax is as follows:

```
nginx.org/ssl-services: "service1[,service2,...]"
```

In the following example we load balance three applications, one of which requires HTTPS:
```yaml
apiVersion: extensions/v1beta1
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
        backend:
          serviceName: tea-svc
          servicePort: 80
      - path: /coffee
        backend:
          serviceName: coffee-svc
          servicePort: 80
      - path: /ssl
        backend:
          serviceName: ssl-svc
          servicePort: 443
```
*ssl-svc* is a service for an HTTPS application. The service becomes available at the `/ssl` path. Note how we used the **nginx.org/ssl-services** annotation.

# WebSocket support

To load balance a WebSocket application with NGINX Ingress Controllers, you need to add the **nginx.org/websocket-services** annotation to your Ingress resource definition. The annotation specifies which services are websocket services. The annotation syntax is as follows:
```
nginx.org/websocket-services: "service1[,service2,...]"
```

In the following example we load balance three applications, one of which is using WebSocket:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/websocket-services: "ws-svc"
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
      - path: /ws
        pathType: Prefix
        backend:
          service:
            name: ws-svc
            port:
              number: 8008
```
*ws-svc* is a service for the WebSocket application. The service becomes available at the `/ws` path. Note how we used the **nginx.org/websocket-services** annotation.

# Support for Active Health Checks

NGINX Plus supports [active health checks](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-health-check/#active-health-checks). To use active health checks in the Ingress controller:

1. Define health checks ([HTTP Readiness Probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/#define-readiness-probes)) in the templates of your application pods.
2. Enable heath checks in the Ingress controller using the annotations.

The Ingress controller provides the following annotations for configuring active health checks:

* Required: `nginx.com/health-checks: "true"` -- enables active health checks. The default is `false`.
* Optional: `nginx.com/health-checks-mandatory: "true"` -- configures active health checks as mandatory. With the default active health checks, when an endpoint is added to NGINX Plus via the API or after a configuration reload, NGINX Plus considers the endpoint to be healthy. With mandatory health checks, when an endpoint is added to NGINX Plus or after a configuration reload, NGINX Plus considers the endpoint to be unhealthy until its health check passes. The default is `false`.
* Optional: `nginx.com/health-checks-mandatory-queue: "500"` -- configures a [queue](http://nginx.org/en/docs/http/ngx_http_upstream_module.html#queue) for temporary storing incoming requests during the time when NGINX Plus is checking the health of the endpoints after a configuration reload. If the queue is not configured or the queue is full, NGINX Plus will drop an incoming request returning the `502` code to the client. The queue is configured only when health checks are mandatory. The timeout parameter of the queue is configured with the value of the timeoutSeconds field of the corresponding Readiness Probe. Choose the size of the queue according with your requirements such as the expected number of requests per second and the timeout. The default is `0`.

# Example

In the following example we enable active health checks in the cafe-ingress Ingress:
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
     kubernetes.io/ingress.class: "nginx"
     nginx.com/health-checks: "true"
spec:
  rules:
  - host: "cafe.example.com"
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
      - path: /beer
        backend:
          serviceName: beer-svc
          servicePort: 80
```

Note that a Readiness Probe must be configured in the pod template:
```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: tea
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tea
  template: 
    metadata:
      labels:
        app: tea
    spec:
      containers:
      - name: tea
        image: nginxdemos/hello:plain-text
        ports:
        - containerPort: 80
        readinessProbe:
          httpGet:
            port: 80
            path: /healthz/tea
            httpHeaders:
            - name: header1
              value: "some value"
            - name: header2
              value: "123"
          initialDelaySeconds: 1
          periodSeconds: 5
          timeoutSeconds: 4
          successThreshold: 2
          failureThreshold: 3
```
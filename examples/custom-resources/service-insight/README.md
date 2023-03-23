# Support for Service Insight

NGINX Plus supports [Service Insight](https://docs.nginx.com/nginx-ingress-controller/logging-and-monitoring/service-insight/). To use the service in the Ingress Controller:

1. [Enable service insight](https://docs.nginx.com/nginx-ingress-controller/logging-and-monitoring/service-insight/#enabling-service-insight-endpoint) in the Ingress Controller in the deployment file.

In the following example we enable service insight in the NGINX Ingress Controller [deployment file](../../../deployments/deployment/nginx-plus-ingress.yaml):

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-ingress
  namespace: nginx-ingress
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-ingress
  template:
    metadata:
      labels:
        app: nginx-ingress
        app.kubernetes.io/name: nginx-ingress
    spec:
      serviceAccountName: nginx-ingress
      automountServiceAccountToken: true
      securityContext:
      ...
      containers:
      - image: nginx-plus-ingress:3.0.2
        imagePullPolicy: IfNotPresent
        name: nginx-plus-ingress
        ports:
        - name: http
          containerPort: 80
        - name: https
          containerPort: 443
        - name: readiness-port
          containerPort: 8081
        - name: prometheus
          containerPort: 9113
        - name: service-insight
          containerPort: 9114
        readinessProbe:
          httpGet:
            path: /nginx-ready
            port: readiness-port
          periodSeconds: 1
        resources:
        ...
        securityContext:
        ...
        env:
        ...
        args:
          - -nginx-plus
          - -nginx-configmaps=$(POD_NAMESPACE)/nginx-config
        ...
          - -enable-service-insight

```

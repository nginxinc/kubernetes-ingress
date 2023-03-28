# Support for Service Insight

  > The Service Insight feature is available only for F5 NGINX Plus.

To use the [Service Insight](https://docs.nginx.com/nginx-ingress-controller/logging-and-monitoring/service-insight/) feature provided by F5 NGINX Ingress Controller you must enable it by setting `serviceInsight.create=true` in your `helm install/upgrade...` command OR  [manifest](../../../deployments/deployment/nginx-plus-ingress.yaml) depending on your preferred installation method.

The following example demonstrates how to enable the Service Insight for NGINX Ingress Controller using [manifests (Deployment)](../../../deployments/deployment/nginx-plus-ingress.yaml):

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

## Deployment

[Install NGINX Ingress Controller](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/), and uncomment the `-enable-service-insight` option: this will allow Service Insight to interact with it.

The examples below use the `nodeport` service.

## Configuration

First, get the nginx-ingress pod id:

```bash
kubectl get pods -n nginx-ingress
```

```
NAME                             READY   STATUS    RESTARTS   AGE
nginx-ingress-5b99f485fb-vflb8   1/1     Running   0          72m
```

Using the id, forward the service insight port (9114) to localhost port 9114:
```bash
kubectl port-forward -n nginx-ingress nginx-ingress-5b99f485fb-vflb8 9114:9114 &
```

## Virtual Servers

### Deployment

Follow the [basic configuration example](../basic-configuration/) to deploy `cafe` app and `cafe virtual server`.

### Testing

Verify that the virtual server is running, and check the hostname:
```bash
kubectl get vs cafe
NAME   STATE   HOST               IP    PORTS   AGE
cafe   Valid   cafe.example.com                 16m
```

Scale down the `tea` and `coffee` deployments:

```bash
kubectl scale deployment tea --replicas=1
```

```bash
kubectl scale deployment coffee --replicas=1
```

Verify `tea` deployment:

```bash
kubectl get deployments.apps tea
```

```bash
NAME   READY   UP-TO-DATE   AVAILABLE   AGE
tea    1/1     1            1           19m
```

Verify `coffee` deployment:

```bash
kubectl get deployments.apps coffee
```

```bash
NAME     READY   UP-TO-DATE   AVAILABLE   AGE
coffee   1/1     1            1           20m
```

Send a `GET` request to the service insight endpoint to check statistics:

Request:

```bash
curl http://localhost:9114/probe/cafe.example.com
```

Response:

```json
{"Total":2,"Up":2,"Unhealthy":0}
```

Scale up deployments:

```bash
kubectl scale deployment tea --replicas=3
```

```bash
kubectl scale deployment coffee --replicas=3
```

Verify deployments:

```bash
kubectl get deployments.apps tea
```

```bash
NAME   READY   UP-TO-DATE   AVAILABLE   AGE
tea    3/3     3            3           31m
```

```bash
kubectl get deployments.apps coffee
```

```bash
NAME     READY   UP-TO-DATE   AVAILABLE   AGE
coffee   3/3     3            3           31m
```

Send a `GET` HTTP request to the service insight endpoint to check statistics:

```bash
curl http://localhost:9114/probe/cafe.example.com
```

Response:

```json
{"Total":6,"Up":6,"Unhealthy":0}
```

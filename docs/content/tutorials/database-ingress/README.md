# Database Load Balancing

NGINX Ingress Controller supports load balancing arbitrary TCP and UDP services using a Custom Resource Definition called a TransportServer.

## Usage

This directory contains a sample using redis; this can easily be adapted to other services.

## Installing NGINX Ingress Controller

This tutorial uses Helm to deploy the Ingress Controller. The Image for the Ingress Controller is typically stored on your own private registry but we will pull it from the NGINX hosted registry which requires authentication. Follow the [docker secret documentation](https://docs.nginx.com/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret/) to configure it. 

Note: Alternatively, you can build the ingress controller following the [building the ingress controller documentation](https://docs.nginx.com/nginx-ingress-controller/installation/building-ingress-controller-image/).

In the [helm-files/values-plus-redis.yaml](helm-files/values-plus-redis.yaml) there is a section for globalConfiguration which will tell NGINX to listen on non-standard ports- see the redis section in this and the extra services. This is also possible with the manifest yaml files.

Create the secret to pull from the private registry:

```
kubectl create secret docker-registry regcred --docker-server=private-registry.nginx.com --docker-username=<NGINX Plus JWT Token> --docker-password=none -n nginx-ingress
```
Complete documentation for creating the secret is here: [https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/]

You will need to add the NGINX Helm repository:

```
helm repo add nginx-stable https://helm.nginx.com/stable
helm repo update
helm install plus nginx-stable/nginx-ingress --namespace nginx-ingress -f values-plus.yaml
```

## TransportServer Configuration

Deploy both the [redis/redis-deployment.yaml](redis/redis-deployment.yaml) and [redis/transportserver-ingress.yaml](redis/transportserver-ingress.yaml) 

```
kubectl apply -f redis
```

This will create a deployment with 10 pods running redis.

For our testing, we will set a simple key to query in our demo:

```
for pod in $(kubectl get pods --selector=app=redis --output=jsonpath={.items..metadata.name}); do echo $pod && kubectl exec -i -t $pod -- redis-cli set pod $pod; done
```

Once done, you can test by finding the LoadBalancer IP of the Ingress Controller and running the redis-cli

```
kubectl get svc -n nginx-ingress plus-nginx-ingress # get the LoadBalancer External-IP
watch -n .5 redis-cli -h LoadBalancerIP get pod # <use this if you have redis-cli installed
# or 
docker run --rm -i -t redis redis-cli -h LoadBalancerIP get pod # <or use this if you have docker
```

View the NGINX Plus dashboard: http://LoadBalancerIP:8080/dashboard.html

![images/transportserver-dashboard.png](images/transportserver-dashboard.png)
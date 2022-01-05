# Database Load Balancing

NGINX Ingress Controller supports load balancing arbitrary TCP and UDP services using a Custom Resource Definition called a TransportServer.

## Usage

This directory contains a sample using redis; this can easily be adapted to other services.

## Installing NGINX Ingress Controller

This tutorial uses Helm to deploy the Ingress Controller. The Image for the Ingress Controller is typically stored on your own private registry but we will pull it from the NGINX hosted registry which requires authentication. Follow the [documentation](https://docs.nginx.com/nginx-ingress-controller/installation/using-the-jwt-token-docker-secret/) to configure it.

In the [values-plus.yaml](values-plus.yaml) there is a section for globalConfiguration which will tell NGINX to listen on non-standard ports- see the redis section in this and the extra services.

You will need to add the NGINX Helm repository to use it:

```
helm repo add nginx-stable https://helm.nginx.com/stable
helm repo update
helm install plus nginx-stable/nginx-ingress --namespace nginx-ingress -f values-plus.yaml
```
Note that the values file expects that you have already created a secret named regcred in the nginx-ingress namespace.

Complete documentation is here: [https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/]

## TransportServer Configuration

Deploy both the [redis-deployment.yaml](redis-deployment.yaml) and [transportserver-ingress.yaml](transportserver-ingress.yaml) 

This will create a deployment with 10 pods running redis.

For our testing, we will set a simple key to query in our demo:

```
for pod in $(kubectl get pods --selector=app=redis --output=jsonpath={.items..metadata.name}); do echo $pod && kubectl exec -i -t $pod -- redis-cli set pod $pod; done
```

Once done, you can test by finding the LoadBalancer IP of the Ingress Controller and running 

```
watch -n .5 redis-cli -h LoadBalancerIP get pod
```

View the NGINX Plus dashboard: http://LoadBalancerIP:8080/dashboard.html

![images/transportserver-dashboard.png](images/transportserver-dashboard.png)
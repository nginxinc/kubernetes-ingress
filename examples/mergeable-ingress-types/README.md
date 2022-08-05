# Mergeable Ingress Types Support

You can spread the Ingress configuration for a common host across multiple Ingress resources using Mergeable Ingress resources. Such resources can belong to the same or different namespaces. This enables
easier management when using a large number of paths.

## Syntax and Rules

A Master is declared using `nginx.org/mergeable-ingress-type: master`. A Master will process all configurations at the
host level, which includes the TLS configuration, and any annotations which will be applied for the complete host. There
can only be one ingress resource on a unique host that contains the master value. Paths cannot be part of the
ingress resource.

Masters cannot contain the following annotations:
* nginx.org/rewrites
* nginx.org/ssl-services
* nginx.org/grpc-services
* nginx.org/websocket-services
* nginx.com/sticky-cookie-services
* nginx.com/health-checks
* nginx.com/health-checks-mandatory
* nginx.com/health-checks-mandatory-queue

A Minion is declared using `nginx.org/mergeable-ingress-type: minion`. A Minion will be used to append different
locations to an ingress resource with the Master value. TLS configurations are not allowed. Multiple minions can be
applied per master as long as they do not have conflicting paths. If a conflicting path is present then the path defined
on the oldest minion will be used.

Minions cannot contain the following annotations:
* nginx.org/proxy-hide-headers
* nginx.org/proxy-pass-headers
* nginx.org/redirect-to-https
* ingress.kubernetes.io/ssl-redirect
* nginx.org/hsts
* nginx.org/hsts-max-age
* nginx.org/hsts-include-subdomains
* nginx.org/server-tokens
* nginx.org/listen-ports
* nginx.org/listen-ports-ssl
* nginx.org/server-snippets

Minions inherent the following annotations from the master, unless they override them:
* nginx.org/proxy-connect-timeout
* nginx.org/proxy-read-timeout
* nginx.org/client-max-body-size
* nginx.org/proxy-buffering
* nginx.org/proxy-buffers
* nginx.org/proxy-buffer-size
* nginx.org/proxy-max-temp-file-size
* nginx.org/location-snippets
* nginx.org/lb-method
* nginx.org/keepalive
* nginx.org/max-fails
* nginx.org/fail-timeout

Note: Ingress Resources with more than one host cannot be used.

## Example

In this example we deploy the NGINX Ingress Controller, a simple web application and then configure
load balancing for that application using Ingress resources with the `nginx.org/mergeable-ingress-type` annotations.

## Running the Example

## 1. Deploy the Ingress Controller

1. Follow the [installation](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/) instructions to deploy the Ingress Controller.

2. Save the public IP address of the Ingress Controller into a shell variable:
    ```
    $ IC_IP=XXX.YYY.ZZZ.III
    ```
3. Save the HTTPS port of the Ingress Controller into a shell variable:
    ```
    $ IC_HTTPS_PORT=<port number>
    ```

## 2. Deploy the Cafe Application

Create the coffee and the tea deployments and services:
```
$ kubectl create -f cafe.yaml
```

## 3. Configure Load Balancing

1. Create a secret with an SSL certificate and a key:
    ```
    $ kubectl create -f cafe-secret.yaml
    ```

2. Create the Master Ingress resource:
    ```
    $ kubectl create -f cafe-master.yaml
    ```

3. Create the Minion Ingress resource for the Coffee Service:
    ```
    $ kubectl create -f coffee-minion.yaml
    ```

4. Create the Minion Ingress resource for the Tea Service:
   ```
    $ kubectl create -f tea-minion.yaml
    ```

## 4. Test the Application

1. To access the application, curl the coffee and the tea services. We'll use ```curl```'s --insecure option to turn off certificate verification of our self-signed
certificate and the --resolve option to set the Host header of a request with ```cafe.example.com```

    To get coffee:
    ```
    $ curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP https://cafe.example.com:$IC_HTTPS_PORT/coffee --insecure
    Server address: 10.12.0.18:80
    Server name: coffee-7586895968-r26zn
    ...
    ```
    If you prefer tea:
    ```
    $ curl --resolve cafe.example.com:$IC_HTTPS_PORT:$IC_IP https://cafe.example.com:$IC_HTTPS_PORT/tea --insecure
    Server address: 10.12.0.19:80
    Server name: tea-7cd44fcb4d-xfw2x
    ...
    ```

## 5. Examine the Configuration

1. Access the NGINX Pod.
```
$ kubectl get pods -n nginx-ingress
NAME                             READY     STATUS    RESTARTS   AGE
nginx-ingress-66bc44674b-hrcx8   1/1       Running   0          4m
```

2. Examine the NGINX Configuration.
```
$ kubectl exec -it nginx-ingress-66bc44674b-hrcx8 -n nginx-ingress -- cat /etc/nginx/conf.d/default-cafe-ingress-master.conf

upstream default-cafe-ingress-coffee-minion-cafe.example.com-coffee-svc {
	server 172.17.0.5:80;
	server 172.17.0.6:80;
}
upstream default-cafe-ingress-tea-minion-cafe.example.com-tea-svc {
	server 172.17.0.7:80;
	server 172.17.0.8:80;
	server 172.17.0.9:80;
}
 # *Master*, configured in Ingress Resource: default-cafe-ingress-master
server {
	listen 80;
	listen 443 ssl;
	ssl_certificate /etc/nginx/secrets/default-cafe-secret;
	ssl_certificate_key /etc/nginx/secrets/default-cafe-secret;
	server_tokens on;
	server_name cafe.example.com;
	if ($scheme = http) {
		return 301 https://$host:443$request_uri;
	}
	 # *Minion*, configured in Ingress Resource: default-cafe-ingress-coffee-minion
	location /coffee {
		proxy_http_version 1.1;
		proxy_connect_timeout 60s;
		proxy_read_timeout 60s;
		client_max_body_size 1m;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Host $host;
		proxy_set_header X-Forwarded-Port $server_port;
		proxy_set_header X-Forwarded-Proto $scheme;
		proxy_buffering on;
		proxy_pass http://default-cafe-ingress-coffee-minion-cafe.example.com-coffee-svc;
	}
	 # *Minion*, configured in Ingress Resource: default-cafe-ingress-tea-minion
	location /tea {
		proxy_http_version 1.1;
		proxy_connect_timeout 60s;
		proxy_read_timeout 60s;
		client_max_body_size 1m;
		proxy_set_header Host $host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Host $host;
		proxy_set_header X-Forwarded-Port $server_port;
		proxy_set_header X-Forwarded-Proto $scheme;
		proxy_buffering on;
		proxy_pass http://default-cafe-ingress-tea-minion-cafe.example.com-tea-svc;
	}
}
```

# Cluster level dynamic rate limiting support

Usually there are multiple Ingress Controller instances in k8s cluster, you want to perform rate limiting  base on cluster level instead of an independent NGINX instance. This example will contain the following implementation references:
* Build NGINX plus sync cluster

* Rate limiting based on svc

* Rate limiting  based on hostname

* Using API to control rate limiting

  

# Cluster Sync setting for Ingress Controller

First of first, you need create NGINX Plus sync cluster. This will let all the instances in the cluster to sync the limiting status and keyval zones. NGINX Plus supports the use of DNS to discover instances in the cluster. Therefore, it is necessary to create a headless svc in k8s and discover all instances through the svc domain name.

## Step 1. Add zone sync port in Ingress Controller deployment

```yaml
apiVersion: apps/v1
kind: Deployment
# omit others
      containers:
      - image: myf5/nginx-plus-ingress:1.7.0
        imagePullPolicy: IfNotPresent
        name: nginx-plus-ingress
        ports:
        - name: ic-cluster-sync
          containerPort: 12345
# omit others
```

## Step 2. Create headless svc for the Ingress Controller deployment

```yaml
kind: Service
apiVersion: v1
metadata:
  namespace: nginx-ingress
  name: nginx-ic-svc
spec:
  selector:
    app: nginx-ingress
  clusterIP: None
  ports:
  - protocol: TCP
    port: 12345
    targetPort: 12345
```

Verify that we can get IC instances by the FQDN name:

```bash
#nslookup nginx-ic-svc.nginx-ingress.svc.cluster.local.

Name:      nginx-ic-svc.nginx-ingress.svc.cluster.local.
Address 1: 10.244.2.68 10-244-2-68.nginx-ic-svc.nginx-ingress.svc.cluster.local
Address 2: 10.244.1.60 10-244-1-60.nginx-ic-svc.nginx-ingress.svc.cluster.local
```

## Step3. Add Stream snippets in the configmap that NGINX Plus using
```
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
  namespace: nginx-ingress
data:
  stream-snippets: |
    resolver 10.96.0.10 valid=5s;
    server {
       listen 12345;
       zone_sync;
       zone_sync_server nginx-ic-svc.nginx-ingress.svc.cluster.local:12345 resolve;
    }
```
> Note: Use k8s dns service cluster ip for for resolver. The resolve option helps to refresh cache.



# Examples:

## 1. Apply same limit threshhold for all svc(location)

### Step 1. Add below http snippets in the configmap
```
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
  namespace: nginx-ingress
data:
  http-snippets: |
    keyval_zone zone=limitreq_uri:64k timeout=2h type=prefix sync;
    keyval $uri $enablelimit zone=limitreq_uri;

    map $enablelimit $limit_key {
        default "";
        1  $binary_remote_addr;
    }

    limit_req_zone $limit_key zone=req_zone_10:1m rate=10r/s sync;
```
The limitreq_uri zone will stores KV, this will be a switch to enable or disable rate limiting for a uri. For below example, rate limiting on /coffee will be enabled, will be disabled on /tea.

We also need decide what factor will be used for rate limiting, here we use $binary_remote_addr. This is controlled by the map configuration.

### Step 2. Enable rate limiting for each svc (location)
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/location-snippets: |
      limit_req zone=req_zone_10 burst=1 nodelay;
      limit_req zone=perserver nodelay;
spec:
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
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
```
Here, we use nginx.org/location-snippets  annotation to insert limit_req configurations.

The last NGINX configuration will be like:
```
	location /tea {
		proxy_http_version 1.1;
		limit_req zone=req_zone_10 burst=1 nodelay;
		limit_req zone=perserver nodelay;

		#omit others

		proxy_pass http://default-cafe-ingress-cafe.example.com-tea-svc-80;
	}
	location /coffee {
		proxy_http_version 1.1;
		limit_req zone=req_zone_10 burst=1 nodelay;
		limit_req zone=perserver nodelay;

		#omit others
	
		proxy_pass http://default-cafe-ingress-cafe.example.com-coffee-svc-80;
	}
```

###  Step 3. Test

Enable rate limiting for /coffee only. Post below json data to the keyval zone  `limitreq_uri`

```
{
  "limitreq_uri": {
    "/coffee": "1",
    "/tea": "0"
  }
}
```

Simulate some traffic to /coffe svc, the test tool output shows many errors:
```
Document Path:          /coffee
Document Length:        158 bytes

Concurrency Level:      100
Time taken for tests:   12.520 seconds
Complete requests:      465
Failed requests:        425
   (Connect: 0, Receive: 0, Length: 425, Exceptions: 0)
Non-2xx responses:      425
```
Simulate some traffic to /tea svc, the test toll output shows no errors:
```
Document Path:          /tea
Document Length:        153 bytes

Concurrency Level:      100
Time taken for tests:   20.719 seconds
Complete requests:      1000
Failed requests:        0
```


##  2. Apply different limit threshhold for all svc(location)

If we need apply different limiting threshhold for different svc, then we need [mergable ingress](https://github.com/nginxinc/kubernetes-ingress/blob/master/examples/mergeable-ingress-types/README.md) 

### Add a new limit_req_zone
Here, add new threshhold 20r/s. 
```
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
  namespace: nginx-ingress
data:
  http-snippets: |
    keyval_zone zone=limitreq_uri:64k timeout=2h type=prefix sync;
    keyval $uri $enablelimit zone=limitreq_uri;

    map $enablelimit $limit_key {
        default "";
        1  $binary_remote_addr;
    }

    limit_req_zone $limit_key zone=req_zone_10:1m rate=10r/s sync;
    limit_req_zone $limit_key zone=req_zone_20:1m rate=20r/s sync;
```

### Apply mergable ingress resources:
```
[root@k8s-master-v1-16 complete-example]# cat cafe-ingress-mergeable.yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: cafe-ingress-master
  annotations:
    kubernetes.io/ingress.class: "nginx"
    nginx.org/mergeable-ingress-type: "master"
spec:
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
  rules:
  - host: cafe.example.com
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: cafe-ingress-teasvc-minion
  annotations:
    kubernetes.io/ingress.class: "nginx"
    nginx.org/mergeable-ingress-type: "minion"
    nginx.org/location-snippets: |
      limit_req zone=req_zone_10 burst=1 nodelay;
      limit_req zone=perserver nodelay;
spec:
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea
        backend:
          serviceName: tea-svc
          servicePort: 80
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: cafe-ingress-coffeesvc-minion
  annotations:
    kubernetes.io/ingress.class: "nginx"
    nginx.org/mergeable-ingress-type: "minion"
    nginx.org/location-snippets: |
      limit_req zone=req_zone_20 burst=1 nodelay;
      limit_req zone=perserver nodelay;
spec:
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /coffee
        backend:
          serviceName: coffee-svc
          servicePort: 80
```

The last nginx configurations will be like:
```
	location /coffee {
		# location for minion default/cafe-ingress-coffeesvc-minion
		proxy_http_version 1.1;
		
		limit_req zone=req_zone_20 burst=1 nodelay;
		limit_req zone=perserver nodelay;

		proxy_pass http://default-cafe-ingress-coffeesvc-minion-cafe.example.com-coffee-svc-80;
	}
	
	location /tea {

		# location for minion default/cafe-ingress-teasvc-minion
		proxy_http_version 1.1;

		limit_req zone=req_zone_10 burst=1 nodelay;
		limit_req zone=perserver nodelay;

		proxy_pass http://default-cafe-ingress-teasvc-minion-cafe.example.com-tea-svc-80;
	}
```



## 3. Limit at hostname level

Sometimes, we may need quickly enable rate limiting on hostname level. Each svc path(location) will be limited once switch on by performing NGINX API. 

To archive this, we need add a new keyval zone,  a new map and a new limit rate. The final NGINS Plus configmap will be like:
```
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
  namespace: nginx-ingress
data:
  http-snippets: |
    keyval_zone zone=limitreq_uri:64k type=prefix;
    keyval $uri $enablelimit zone=limitreq_uri;
 
 		#add keyval zone for per hostname
    keyval_zone  zone=limitper_server:64k;
    keyval $server_name $enableserverlimit zone=limitper_server;
 
    map $enablelimit $limit_key {
        default "";
        1  $binary_remote_addr;
    }
 
 		#add map to let NGINX perform rate limiting base on $server_name factor
    map $enableserverlimit $limit_key_servername {
        default "";
        1 $server_name;
    }
    limit_req_zone $limit_key zone=req_zone_10:1m rate=10r/s;
    limit_req_zone $limit_key zone=req_zone_20:1m rate=20r/s;
    
    #set limit rate to 50r/s
    limit_req_zone $limit_key_servername zone=perserver:10m rate=50r/s;
```

Make sure config `perserver` limit_req zone in ingress annotation:
```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    nginx.org/location-snippets: |
      limit_req zone=req_zone_10 burst=1 nodelay;
      limit_req zone=perserver nodelay;
spec:
  tls:
  - hosts:
    - cafe.example.com
    secretName: cafe-secret
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
```

Disable the `limitreq_uri` and enable `limitper_server` for the keyval zones. The final keyval zone will be:
```
{
  "limitper_server": {
    "cafe.example.com": "1"
  },
  "limitreq_uri": {
    "/coffee": "0",
    "/tea": "0"
  }
}
```

Simulate traffic to /tea and /coffee svc, the test tool output will be like :
```
Document Path:          /tea
Document Length:        153 bytes

Concurrency Level:      100
Time taken for tests:   11.588 seconds
Complete requests:      1000
Failed requests:        914
   (Connect: 0, Receive: 0, Length: 914, Exceptions: 0)
```

```
Document Path:          /coffee
Document Length:        158 bytes

Concurrency Level:      100
Time taken for tests:   20.065 seconds
Complete requests:      1000
Failed requests:        901
   (Connect: 0, Receive: 0, Length: 901, Exceptions: 0)
```



## Summary

By using NGINX Plus zone_sync module, we can limit requests at whole cluster level, each Ingress controller instance will sync the zone info. The below show that the limit zones are synced.
```
curl -X GET "http://172.16.10.212:7777/api/6/stream/zone_sync/" -H "accept: application/json"

{
  "status": {
    "nodes_online": 1,
    "msgs_in": 34,
    "msgs_out": 1,
    "bytes_in": 1434,
    "bytes_out": 68
  },
  "zones": {
    "limitreq_uri": {
      "records_total": 0,
      "records_pending": 0
    },
    "limitper_server": {
      "records_total": 1,
      "records_pending": 0
    },
    "req_zone_10": {
      "records_total": 0,
      "records_pending": 0
    },
    "req_zone_20": {
      "records_total": 0,
      "records_pending": 0
    },
    "perserver": {
      "records_total": 1,
      "records_pending": 0
    }
  }
}
```
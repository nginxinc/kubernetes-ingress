# TransportServer SNI

In this example we create two different TransportServers that listen on the same interface, which are distinguished by their Host field.

1. Create a GlobalConfiguration resource with the following listener.

```yaml
     listeners:
      - name: tcp-listener
        port: 7000 
        protocol: TCP
```

1. Add a custom port to the NIC pod via the helm chart

```yaml
controller.customPorts:
    - name: port
      containerPort: 7000
      protocol: TCP
```

1. Add a custom port to the NIC service

```yaml
controller.service.customPorts:
          - name: tcp-port 
            port: 7000 
            protocol: TCP
            targetPort: 7000 
```

1. Create the cafe-secret, and mongo-secret. These secrets are used for TLS in the TransportServers
`kubectl apply -f cafe-secret.yaml`
`kubectl apply -f mongo-secret.yaml`

1. Create the mongo and tcp echo example applications
`kubectl apply -f mongo.yaml`
`kubectl apply -f tcp-echo-server.yaml`

1. Wait until these are ready.
`kubectl get deploy -w`

1. Create the TransportServers for each application
`kubectl apply -f cafe-transport-server.yaml`
`kubectl apply -f mongo-transport-server.yaml`

1. Ensure they are in valid state
`kubectl get ts`

```bash
NAME       STATE   REASON           AGE
cafe-ts    Valid   AddedOrUpdated   2m
mongo-ts   Valid   AddedOrUpdated   2m
```

1. Set up /etc/hosts or DNS
In this example I am running NIC locally, so I will set
cafe.example.com and mongo.example.com in my /etc/hosts to localhost
If running NIC as a live service, the server names would be set at the DNS layer
`cat /etc/hosts`

```bash
...
127.0.0.1 cafe.example.com
127.0.0.1 mongo.example.com
```

1. Expose port 7000 of the NIC Loadbalancer Service
`k port-forward svc/my-release-nginx-ingress-controller 7000:7000`

1. Use ncat to ping cafe.example.com on port 7000 with ssl
`ncat --ssl cafe.example.com 7000`
When you write a message you should get sent back

```bash
hi
hi
```

And once the connection is closed (ctrl + c) you should see the request and response in the NIC logs are both 2 bytes

```bash
127.0.0.1 [24/Sep/2024:15:48:58 +0000] TCP 200 3 3 2.702 "-
```

1. Use mongosh to connect to the mongodb container, via the TransportServer, also on port 7000
`mongosh --host mongo.example.com --port 7000 --tls --tlsAllowInvalidCertificates`

```bash
test> show dbs
admin   40.00 KiB
config  60.00 KiB
local   40.00 KiB
test>
```

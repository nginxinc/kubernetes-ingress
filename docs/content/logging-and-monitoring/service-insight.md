---
title: Service Insight

description: "The Ingress Controller exposes the Service Insight endpoint."
weight: 2100
doctypes: [""]
aliases:
    - /service-insight/
toc: true
docs: "DOCS-000"
---


The Ingress Controller exposes an endpoint and provides host statistics for Virtual Servers (VS) and Transport Servers (TS).
It exposes data in JSON format and returns HTTP status codes.
The response body holds information about the total, down and the unhealthy number of
upstreams associated with the hostname.
Returned HTTP codes indicate the health of the upstreams (service).

The service is not healthy (HTTP response code different than 200 OK) if all upstreams are unhealthy.
The service is healthy if at least one upstream is healthy. In this case, the endpoint returns HTTP code 200 OK.

## Enabling Service Insight Endpoint

If you're using *Kubernetes manifests* (Deployment or DaemonSet) to install the Ingress Controller, to enable the Service Insight endpoint:

1. Run the Ingress Controller with the `-enable-service-insight` [command-line argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments). This will expose the Ingress Controller endpoint via paths `/probe/{hostname}` for Virtual Servers, and `/probe/ts/{service_name}` for Transport Servers on port `9114` (customizable with the `-service-insight-listen-port` command-line argument). The `service_name` parameter refers to the name of the deployed service (the service specified under `upstreams` in the transport server).
1. To enable TLS for the Service Insight endpoint, configure the `-service-insight-tls-secret` cli argument with the namespace and name of a TLS Secret.
1. Add the Service Insight port to the list of the ports of the Ingress Controller container in the template of the Ingress Controller pod:

    ```yaml
    - name: service-insight
      containerPort: 9114
    ```

If you're using *Helm* to install the Ingress Controller, to enable Service Insight endpoint, configure the `serviceinsight.*` parameters of the Helm chart. See the [Installation with Helm](/nginx-ingress-controller/installation/installation-with-helm) doc.

## Available Statistics and HTTP Response Codes

The Service Insight provides the following statistics:

* Total number of VS and TS
* Number of VS and TS in 'Up' state
* Number of VS and TS in 'Unhealthy' state

These statistics are returned as JSON:

```json
{ "Total": <int>, "Up": <int>, "Unhealthy": <int>  }
```

Response codes:

* HTTP 200 OK - Service is healthy
* HTTP 404 Not Found - No upstreams/VS/TS found for the requested hostname/name
* HTTP 418 I'm a teapot - The service is down (All upstreams/VS/TS are "Unhealthy")

**Note**: wildcards in hostnames are not supported at the moment.

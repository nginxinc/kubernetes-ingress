---
title: Logging

description: 
weight: 1800
doctypes: [""]
toc: true
---


The NGINX Ingress Controller exposes the logs of the Ingress Controller process (the process that generates NGINX configuration and reloads NGINX to apply it) and NGINX access and error logs. All logs are sent to the standard output and error of the Ingress Controller process. To view the logs, you can execute the `kubectl logs` command for an Ingress Controller pod. For example:
```
$ kubectl logs <nginx-ingress-pod> -n nginx-ingress
```

## Ingress Controller Process Logs

The Ingress Controller process logs are configured through the `-v` command-line argument of the Ingress Controller, which sets the log verbosity level. The default value is `1`, for which the minimum amount of logs is reported. The value `3` is useful for troubleshooting: you will be able to see how the Ingress Controller gets updates from the Kubernetes API, generates NGINX configuration and reloads NGINX.

See also the doc about Ingress Controller [command-line arguments](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments).

## NGINX Logs

The NGINX includes two logs:
* *Access log*, where NGINX writes information about client requests in the access log right after the request is processed. The access log is configured via the [logging-related](/nginx-ingress-controller/configuration/global-configuration/configmap-resource#logging) ConfigMap keys:
    * `log-format` for HTTP and HTTPS traffic.
    * `stream-log-format` for TCP, UDP, and TLS Passthrough traffic.

    Additionally, you can disable access logging with the `access-log-off` ConfigMap key.
* *Error log*, where NGINX writes information about encountered issues of different severity levels. It is configured via the `error-log-level` [ConfigMap key](/nginx-ingress-controller/configuration/global-configuration/configmap-resource#logging). To enable debug logging, set the level to `debug` and also set the `-nginx-debug` [command-line argument](/nginx-ingress-controller/configuration/global-configuration/command-line-arguments), so that NGINX is started with the debug binary `nginx-debug`.

See also the doc about [NGINX logs](https://docs.nginx.com/nginx/admin-guide/monitoring/logging/) from NGINX Admin guide.

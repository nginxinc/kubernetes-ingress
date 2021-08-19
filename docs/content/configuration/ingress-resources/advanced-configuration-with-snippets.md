---
title: Advanced Configuration with Snippets
description:
weight: 1800
doctypes: [""]
toc: true
---


Snippets allow you to insert raw NGINX config into different contexts of the NGINX configurations that the Ingress Controller generates. These should be used as a last-resort solution in cases where annotations and ConfigMap entries cannot help. Snippets are intended for advanced NGINX users who need more control over the generated NGINX configuration.

Snippets are also available through the [ConfigMap](/nginx-ingress-controller/configuration/global-configuration/configmap-resource). Annotations take precedence over the ConfigMap.

## Using Snippets

The example below shows how to use snippets to customize the NGINX configuration template using annotations.
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress-with-snippets
  annotations:
    nginx.org/server-snippets: |
      location / {
          return 302 /coffee;
      }
    nginx.org/location-snippets: |
      add_header my-test-header test-value;
spec:
  rules:
  - host: cafe.example.com
    http:
      paths:
      - path: /tea
        pathType: Prefix
        backend:
          service:
            name: tea-svc
            port:
              number: 80
      - path: /coffee
        pathType: Prefix
        backend:
          service:
            name: coffee-svc
            port:
              number: 80
```

Generated NGINX configuration:
```nginx
server {
    listen 80;


    location / {
        return 302 /coffee;
    }


    location /coffee {
        proxy_http_version 1.1;


        add_header my-test-header test-value;
        ...
        proxy_pass http://default-cafe-ingress-with-snippets-cafe.example.com-coffee-svc-80;
    }

    location /tea {
        proxy_http_version 1.1;

        add_header my-test-header test-value;
        ...
        proxy_pass http://default-cafe-ingress-with-snippets-cafe.example.com-tea-svc-80;
    }
}
```
**Note**: The generated configs are truncated for the clarity of the example.

## Summary of Snippets

See the [snippets annotations](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations/#snippets-and-custom-templates) documentation for more information.

## Disadvantages of Using Snippets

Snippets have the following disadvantages:

* *Complexity*. To use snippets, you will need to:
  * Understand NGINX configuration primitives and implement a correct NGINX configuration.
  * Understand how the IC generates NGINX configuration so that a snippet doesn't interfere with the other features in the configuration.
* *Decreased robustness*. An incorrect snippet makes the NGINX config invalid, which causes reload failures. This will prevent any new configuration updates, including updates for the other Ingress resources, until the snippet is fixed.
* *Security implications*. Snippets give access to NGINX configuration primitives and those primitives are not validated by the Ingress Controller. For example, a snippet can configure NGINX to serve the TLS certificates and keys used for TLS termination for Ingress resources.

> **Note**: If the NGINX config includes an invalid snippet, NGINX will continue to operate with the latest valid configuration.

## Troubleshooting

If a snippet includes an invalid NGINX configuration, the Ingress Controller will fail to reload NGINX. The error will be reported in the Ingress Controller logs and an event with the error will be associated with the Ingress resource:

An example of an error from the logs:
```
[emerg] 31#31: unknown directive "badd_header" in /etc/nginx/conf.d/default-cafe-ingress-with-snippets.conf:54
Event(v1.ObjectReference{Kind:"Ingress", Namespace:"default", Name:"cafe-ingress-with-snippets", UID:"f9656dc9-63a6-41dd-a499-525b0e0309bb", APIVersion:"extensions/v1beta1", ResourceVersion:"2322030", FieldPath:""}): type: 'Warning' reason: 'AddedOrUpdatedWithError' Configuration for default/cafe-ingress-with-snippets was added or updated, but not applied: Error reloading NGINX for default/cafe-ingress-with-snippets: nginx reload failed: Command /usr/sbin/nginx -s reload stdout: ""
stderr: "nginx: [emerg] unknown directive \"badd_header\" in /etc/nginx/conf.d/default-cafe-ingress-with-snippets.conf:54\n"
finished with error: exit status 1
```

An example of an event with an error (you can view events associated with the Ingress by running `kubectl describe -n nginx-ingress ingress nginx-ingress`):
```
Events:
Type     Reason                   Age                From                      Message
----     ------                   ----               ----                      -------
Normal   AddedOrUpdated           52m (x3 over 61m)  nginx-ingress-controller  Configuration for default/cafe-ingress-with-snippets was added or updated
finished with error: exit status 1
Warning  AddedOrUpdatedWithError  54s (x2 over 89s)  nginx-ingress-controller  Configuration for default/cafe-ingress-with-snippets was added or updated, but not applied: Error reloading NGINX for default/cafe-ingress-with-snippets: nginx reload failed: Command /usr/sbin/nginx -s reload stdout: ""
stderr: "nginx: [emerg] unknown directive \"badd_header\" in /etc/nginx/conf.d/default-cafe-ingress-with-snippets.conf:54\n"
finished with error: exit status 1
```

Additionally, to help troubleshoot snippets, a number of Prometheus metrics show the stats about failed reloads – `controller_nginx_last_reload_status` and `controller_nginx_reload_errors_total`.

# Basic Configuration

The example below shows a basic Ingress resource definition. It load balances requests for two services -- coffee and tea -- comprising a hypothetical *cafe* app hosted at `cafe.example.com`:
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: cafe-ingress
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

Here is a breakdown of what this Ingress resource definition means:
* The `metadata.name` field defines the name of the resource `cafe‑ingress`.
* In the `spec.tls` field we set up SSL/TLS termination:
    * In the `secretName`, we reference a secret resource by its name, `cafe‑secret`. This resource contains the SSL/TLS certificate and key and it must be deployed prior to the Ingress resource.
    * In the `hosts` field, we apply the certificate and key to our `cafe.example.com` host.
* In the `spec.rules` field, we define a host with domain name `cafe.example.com`.
* In the `paths` field, we define two path‑based rules:
  * The rule with the path `/tea` instructs NGINX to distribute the requests with the `/tea` URI among the pods of the *tea* service, which is deployed with the name `tea‑svc` in the cluster.
  * The rule with the path `/coffee` instructs NGINX to distribute the requests with the `/coffee` URI among the pods of the *coffee* service, which is deployed with the name `coffee‑svc` in the cluster.
  * Both rules instruct NGINX to distribute the requests to `port 80` of the corresponding service (the `servicePort` field).

> For complete instructions on deploying the Ingress and Secret resources in the cluster, see the [complete-example](https://github.com/nginxinc/kubernetes-ingress/tree/v1.8.0/examples/complete-example) in our GitHub repo.

> To learn more about the Ingress resource, see the [Ingress resource documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/) in the Kubernetes docs.

## Restrictions

The NGINX Ingress Controller imposes the following restrictions on Ingress resources:
* When defining an Ingress resource, the `host` field is required.
* The `host` value must be unique for each ingress resource.

## Advanced Configuration

The Ingress resource only allows you to use basic NGINX features -- host and path-based routing and TLS termination. Advanced features like rewriting the request URI or inserting additional response headers are available through annotations. See the [Advanced Configuration with Annotations](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations) doc.

The Ingress Controller generates NGINX configuration by executing a template file that contains configuration options. These options are set via the Ingress resource and the Ingress Controller's ConfigMap. Advanced NGINX users who require more control over the generated NGINX configurations can use snippets to insert raw NGINX config. See [Advanced Configuration with Snippets](/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-snippets) for more information. Additionally, it is possible to customize the template. See [Custom Templates](/nginx-ingress-controller/configuration/global-configuration/custom-templates/) for instructions.

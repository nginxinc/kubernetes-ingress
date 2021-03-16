# OpenTracing

The Ingress Controller supports [OpenTracing](https://opentracing.io/) with the third-party module [opentracing-contrib/nginx-opentracing](https://github.com/opentracing-contrib/nginx-opentracing).

This document explains how to use OpenTracing with the Ingress Controller.

## Prerequisites
1. **Use the Ingress Controller image with OpenTracing.** The default Ingress Controller images don’t include the OpenTracing module. To use OpenTracing, you need to build the image with that module. Follow the build instructions to build the image using `openshift-image` for NGINX or `openshift-image-plus` for NGINX Plus.
By default, the Dockerfiles install Jaeger as a tracer. However, it is possible to replace Jaeger with other supported [tracers](https://github.com/opentracing-contrib/nginx-opentracing#building-from-source). For that, please modify the Dockerfile accordingly:
   1. Change the download line in the tracer-downloader stage of the Dockerfile to download the right tracer.
   1. Edit the COPY line of the final image to copy the previously downloaded tracer to the image

1. **Load the OpenTracing module.** You need to load the module with the configuration for the chosen tracer using the following ConfigMap keys:
   * `opentracing-tracer`: sets the path to the vendor tracer binary plugin. This is the path you used in the COPY line of step *ii* above.
   * `opentracing-tracer-config`: sets the tracer configuration in JSON format.

   Below an example on how to use those keys to load the module with Jaeger tracer:
    ```yaml
    opentracing-tracer: "/usr/local/lib/libjaegertracing_plugin.so"
    opentracing-tracer-config: |
            {
                "service_name": "nginx-ingress",
                "sampler": {
                    "type": "const",
                    "param": 1
                },
                "reporter": {
                    "localAgentHostPort": "jaeger-agent.default.svc.cluster.local:6831"
                }
            }
    ```

## Enable OpenTracing Globally
To enable OpenTracing globally (for all Ingress, VirtualServer and VirtualServerRoute resources), set the `opentracing` ConfigMap key to `True`:

```yaml
opentracing: True
```

## Enable/Disable OpenTracing per Ingress Resource

It is possible to use annotations to enable or disable OpenTracing for a specific Ingress Resource. As mentioned in the prerequisites section, both `opentracing-tracer` and `opentracing-tracer-config` must be configured.

Consider the following two cases:
1. OpenTracing is globally disabled.
   1. To enable OpenTracing for a specific Ingress Resource, use the server snippet annotation:
        ```yaml
        nginx.org/server-snippets: |
            opentracing on;
        ```
   1. To enable OpenTracing for specific paths, (1) you need to use [Mergeable Ingress resources](/nginx-ingress-controller/configuration/ingress-resources/cross-namespace-configuration) and (2) use the location snippets annotation to enable OpenTracing for the paths of a specific Minion Ingress resource:
        ```yaml
        nginx.org/location-snippets: |
            opentracing on;
        ```

2. OpenTracing is globally enabled:
   1. To disable OpenTracing for a specific Ingress Resource, use the server snippet annotation:
        ```yaml
        nginx.org/server-snippets: |
            opentracing off;
        ```

   1. To disable OpenTracing for specific paths, (1) you need to use [Mergeable Ingress resources](/nginx-ingress-controller/configuration/ingress-resources/cross-namespace-configuration) and (2) use the location snippets annotation to disable OpenTracing for the paths of a specific Minion Ingress resource:
        ```yaml
        nginx.org/location-snippets: |
            opentracing off;
        ```

## Customize OpenTracing

You can customize OpenTracing though the supported [OpenTracing module directives](https://github.com/opentracing-contrib/nginx-opentracing/blob/master/doc/Reference.md). Use the snippets ConfigMap keys or annotations to insert those directives into the http, server or location contexts of the generated NGINX configuration.

For example, to propagate the active span context for upstream requests, it is required to set the `opentracing_propagate_context` directive, which you can add to an Ingress resource using the location snippets annotation:

```yaml
nginx.org/location-snippets: |
   opentracing_propagate_context;
```

**Note**: `opentracing_propagate_context` and `opentracing_grpc_propagate_context` directives can be used in http, server or location contexts according to the [module documentation](https://github.com/opentracing-contrib/nginx-opentracing/blob/master/doc/Reference.md#opentracing_propagate_context). However, because of the way the module works and how the Ingress Controller generates the NGINX configuration, it is only possible to use the directive in the location context.

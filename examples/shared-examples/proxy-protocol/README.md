# PROXY Protocol

Proxies and load balancers, such as HAProxy or ELB, can pass the client's information (the IP address and the port) to
the next proxy or load balancer via the PROXY Protocol. To enable NGINX Ingress Controller to receive that information,
use the `proxy-protocol` ConfigMaps configuration key as well as the `real-ip-header` and the `set-real-ip-from` keys.
Once you enable the PROXY Protocol, it is enabled for every Ingress and VirtualServer resource. **NOTE** TransportServer
resource supports PROXY Protocol only when TLS Passthrough is enabled for the Ingress Controller.

## Syntax

The `proxy-protocol` key syntax is as follows:

```yaml
proxy-protocol: "True | False"
```

Additionally, you must configure the following keys:

- **real-ip-header**: Set its value to `proxy_protocol`.
- **set-real-ip-from**: Set its value to the IP address or the subnet of the proxy or the load balancer. See
  [set-real-ip-from](https://nginx.org/en/docs/http/ngx_http_realip_module.html#set_real_ip_from)

## Example

In the example below we configure the PROXY Protocol via a ConfigMaps resource. `set-real-ip-from` is set to
`192.168.0.0/16`. This is the CIDR range of the proxy that sits in front of the Ingress Controller in this example. You
can set this to `0.0.0.0/0` to trust all IPs. After we create the ConfigMaps resource, the client's IP address is
available via the `$remote_addr` variable in the NGINX configuration. By default, NGINX Ingress Controller logs the
value of this variable and also passes the value to the backend service in the `X-Real-IP` header.

The default log format for NGINX is `'$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent
"$http_referer" "$http_user_agent"'`

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  name: nginx-config
data:
  proxy-protocol: "True"
  real-ip-header: "proxy_protocol"
  set-real-ip-from: "192.168.0.0/16"
```

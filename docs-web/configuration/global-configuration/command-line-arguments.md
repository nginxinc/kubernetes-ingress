# Command-line Arguments

The Ingress Controller supports several command-line arguments. Setting the arguments depends on how you install the Ingress Controller:
* If you're using *Kubernetes manifests* (Deployment or DaemonSet) to install the Ingress Controller, to set the command-line arguments, modify those manifests accordingly. See the [Installation with Manifests](/nginx-ingress-controller/installation/installation-with-manifests) doc.
* If you're using *Helm* to install the Ingress Controller, modify the parameters of the Helm chart that correspond to the command-line arguments. See the [Installation with Helm](/nginx-ingress-controller/installation/installation-with-helm) doc. 

Below we describe the available command-line arguments:
```eval_rst
.. option:: -default-server-tls-secret <string>

	Secret with a TLS certificate and key for TLS termination of the default server.

	- If not set, certificate and key in the file "/etc/nginx/secrets/default" are used.
	- If a secret is set, but the Ingress controller is not able to fetch it from Kubernetes API, or if a secret is not set and the file "/etc/nginx/secrets/  default" does not exist, the Ingress controller will fail to start.

	Format: ``<namespace>/<name>``

.. option:: -wildcard-tls-secret <string>

	A Secret with a TLS certificate and key for TLS termination of every Ingress host for which TLS termination is enabled but the Secret is not specified.

	- If the argument is not set, for such Ingress hosts NGINX will break any attempt to establish a TLS connection.

	- If the argument is set, but the Ingress controller is not able to fetch the Secret from Kubernetes API, the Ingress controller will fail to start.

	Format: ``<namespace>/<name>``

.. option:: -enable-custom-resources

	Enables custom resources (default true)

.. option:: -enable-leader-election

	Enables Leader election to avoid multiple replicas of the controller reporting the status of Ingress, VirtualServer and VirtualServerRoute resources -- only one replica will report status (default true).

	See :option:`-report-ingress-status` flag.

.. option:: -enable-tls-passthrough

	Enable TLS Passthrough on port 443.

	Requires :option:`-enable-custom-resources`.	

.. option:: -external-service <string>

	Specifies the name of the service with the type LoadBalancer through which the Ingress controller pods are exposed externally. The external address of the service is used when reporting the status of Ingress, VirtualServer and VirtualServerRoute resources.

	For Ingress resources only: Requires :option:`-report-ingress-status`.

.. option:: -global-configuration <string>

	A GlobalConfiguration resource for global configuration of the Ingress Controller. If the flag is set, but the Ingress Controller is not able to fetch the corresponding resource from Kubernetes API, the Ingress Controller will fail to start.
	
	Format: ``<namespace>/<name>``

	Requires :option:`-enable-custom-resources`.

.. option:: -health-status

	Adds a location "/nginx-health" to the default server. The location responds with the 200 status code for any request.
	Useful for external health-checking of the Ingress controller.

.. option:: -ingress-class <string>

	A class of the Ingress controller. The Ingress controller only processes Ingress resources that belong to its class (i.e. have the annotation "kubernetes.io/ingress.class" or the "ingressClassName" field in VirtualServer/VirtualServerRoute").
	Additionally, the Ingress controller processes Ingress resources that do not have that annotation, which can be disabled by setting the :option:`-use-ingress-class-only` flag (default "nginx").

.. option:: -ingress-template-path <string>

	Path to the ingress NGINX configuration template for an ingress resource. Default for NGINX is "nginx.ingress.tmpl"; default for NGINX Plus is "nginx-plus.ingress.tmpl".

.. option:: -leader-election-lock-name <string>

	Specifies the name of the ConfigMap, within the same namespace as the controller, used as the lock for leader election. Requires :option:`-enable-leader-election`.

.. option:: -log_backtrace_at <value>

	When logging hits line ``file:N``, emit a stack trace

.. option:: -main-template-path <string>

	Path to the main NGINX configuration template.

	- Default for NGINX is "nginx.ingress.tmpl"
	- Default for NGINX Plus is "nginx-plus.ingress.tmpl".

.. option:: -nginx-configmaps <string>

	A ConfigMap resource for customizing NGINX configuration. If a ConfigMap is set, but the Ingress controller is not able to fetch it from Kubernetes API, the Ingress controller will fail to start.

	Format: ``<namespace>/<name>``

.. option:: -nginx-debug

	Enable debugging for NGINX. Uses the nginx-debug binary. Requires 'error-log-level: debug' in the ConfigMap.

.. option:: -nginx-plus

	Enable support for NGINX Plus

.. option:: -nginx-reload-timeout <value>

    Timeout in milliseconds which the Ingress Controller will wait for a successful NGINX reload after a change or at the initial start. (default is 4000. Default is 20000 instead if `enable-app-protect` is true)

.. option:: -nginx-status

	Enable the NGINX stub_status, or the NGINX Plus API. (default true)

.. option:: -nginx-status-allow-cidrs <string>

	Whitelist IPv4 IP/CIDR blocks to allow access to NGINX stub_status or the NGINX Plus API.
	Separate multiple IP/CIDR by commas. (default "127.0.0.1")

.. option:: -nginx-status-port [int]

	Set the port where the NGINX stub_status or the NGINX Plus API is exposed.

	Format: ``[1023 - 65535]`` (default 8080)

.. option:: -proxy <string>

	Use a proxy server to connect to Kubernetes API started by "kubectl proxy" command. **For testing purposes only**.
	The Ingress controller does not start NGINX and does not write any generated NGINX configuration files to disk.

.. option:: -report-ingress-status

	Update the address field in the status of Ingresses resources.
	Requires the :option:`-external-service` flag or the ``external-status-address`` key in the ConfigMap.

.. option:: -transportserver-template-path <string>

	Path to the TransportServer NGINX configuration template for a TransportServer resource.

	- Default for NGINX is "nginx.transportserver.tmpl"
	- Default for NGINX Plus is "nginx-plus.transportserver.tmpl".

.. option:: -use-ingress-class-only

	Ignore Ingress resources without the "kubernetes.io/ingress.class" annotation or the "ingressClassName" field in VirtualServer/VirtualServerRoute.

.. option:: -v <value>

	Log level for V logs

.. option:: -version

	Print the version and git-commit hash and exit

.. option:: -virtualserver-template-path <string>

	Path to the VirtualServer NGINX configuration template for a VirtualServer resource.

	- Default for NGINX is "nginx.ingress.tmpl"
	- Default for NGINX Plus is "nginx-plus.ingress.tmpl".

.. option:: -vmodule <value>

	A comma-separated list of pattern=N settings for file-filtered logging.

.. option:: -watch-namespace <string>

	Namespace to watch for Ingress resources. By default the Ingress controller watches all namespaces.

.. option:: -enable-prometheus-metrics

	Enables exposing NGINX or NGINX Plus metrics in the Prometheus format.

.. option:: -prometheus-metrics-listen-port

	Sets the port where the Prometheus metrics are exposed.

	Format: ``[1023 - 65535]`` (default 9113)

.. option:: -spire-agent-address

	Specifies the address of a running Spire agent. **For use with NGINX Service Mesh only**.

	- If the argument is set, but the Ingress Controller is unable to connect to the Spire Agent, the Ingress Controller will fail to start.
```

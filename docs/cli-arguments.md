# Ingress Controller Command-line Arguments

```
Usage of ./nginx-ingress:
  -alsologtostderr
    	log to standard error as well as files
  -default-server-tls-secret string
    	A Secret with a TLS certificate and key for TLS termination of the default server. Format: <namespace>/<name>.
	If not set, certificate and key in the file "/etc/nginx/secrets/default" are used. If a secret is set,
	but the Ingress controller is not able to fetch it from Kubernetes API or a secret is not set and
	the file "/etc/nginx/secrets/default" does not exist, the Ingress controller will fail to start
  -enable-leader-election
    	Enable Leader election to avoid multiple replicas of the controller reporting the status of Ingress resources -- only one replica will report status. See -report-ingress-status flag.
  -external-service string
    	Specifies the name of the service with the type LoadBalancer through which the Ingress controller pods are exposed externally.
    	The external address of the service is used when reporting the status of Ingress resources. Requires -report-ingress-status.
  -health-status
    	Add a location "/nginx-health" to the default server. The location responds with the 200 status code for any request.
	Useful for external health-checking of the Ingress controller
  -ingress-class string
    	A class of the Ingress controller. The Ingress controller only processes Ingress resources that belong to its class
	- i.e. have the annotation "kubernetes.io/ingress.class" equal to the class. Additionally,
	the Ingress controller processes Ingress resources that do not have that annotation,
	which can be disabled by setting the "-use-ingress-class-only" flag (default "nginx")
  -ingress-template-path string
    	Path to the ingress NGINX configuration template for an ingress resource.
	(default for NGINX "nginx.ingress.tmpl"; default for NGINX Plus "nginx-plus.ingress.tmpl")
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -main-template-path string
    	Path to the main NGINX configuration template. (default for NGINX "nginx.tmpl"; default for NGINX Plus "nginx-plus.tmpl")
  -nginx-configmaps string
    	A ConfigMap resource for customizing NGINX configuration. If a ConfigMap is set,
	but the Ingress controller is not able to fetch it from Kubernetes API, the Ingress controller will fail to start.
	Format: <namespace>/<name>
  -nginx-debug
	Enable debugging for NGINX. Uses the nginx-debug binary. Requires 'error-log-level: debug' in the ConfigMap.
  -nginx-plus
    	Enable support for NGINX Plus
  -nginx-status
    	Enable the NGINX stub_status, or the NGINX Plus API. (default true)
  -nginx-status-allow-cidrs string
        Whitelist IPv4 IP/CIDR blocks to allow access to NGINX stub_status or the NGINX Plus API.
	Separate multiple IP/CIDR by commas. (default "127.0.0.1")
  -nginx-status-port int
    	Set the port where the NGINX stub_status or the NGINX Plus API is exposed. [1023 - 65535] (default 8080)
  -proxy string
        Use a proxy server to connect to Kubernetes API started by "kubectl proxy" command. For testing purposes only.
        The Ingress controller does not start NGINX and does not write any generated NGINX configuration files to disk
  -report-ingress-status
    	Update the address field in the status of Ingresses resources. Requires the -external-service flag, or the 'external-status-address' key in the ConfigMap.
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -use-ingress-class-only
    	Ignore Ingress resources without the "kubernetes.io/ingress.class" annotation
  -v value
    	log level for V logs
  -version
    	Print the version and git-commit hash and exit
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
  -watch-namespace string
    	Namespace to watch for Ingress resources. By default the Ingress controller watches all namespaces
```

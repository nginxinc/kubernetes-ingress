package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	cr_validation "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/validation"
	k8s_nginx "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned"
	conf_scheme "github.com/nginxinc/kubernetes-ingress/pkg/client/clientset/versioned/scheme"
	"github.com/nginxinc/nginx-plus-go-client/client"
	nginxCollector "github.com/nginxinc/nginx-prometheus-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	util_version "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var (

	// Set during build
	version string
	commit  string
	date    string

	healthStatus = flag.Bool("health-status", false,
		`Add a location based on the value of health-status-uri to the default server. The location responds with the 200 status code for any request.
	Useful for external health-checking of the Ingress controller`)

	healthStatusURI = flag.String("health-status-uri", "/nginx-health",
		`Sets the URI of health status location in the default server. Requires -health-status`)

	proxyURL = flag.String("proxy", "",
		`Use a proxy server to connect to Kubernetes API started by "kubectl proxy" command. For testing purposes only.
	The Ingress controller does not start NGINX and does not write any generated NGINX configuration files to disk`)

	watchNamespace = flag.String("watch-namespace", api_v1.NamespaceAll,
		`Namespace to watch for Ingress resources. By default the Ingress controller watches all namespaces`)

	nginxConfigMaps = flag.String("nginx-configmaps", "",
		`A ConfigMap resource for customizing NGINX configuration. If a ConfigMap is set,
	but the Ingress controller is not able to fetch it from Kubernetes API, the Ingress controller will fail to start.
	Format: <namespace>/<name>`)

	nginxPlus = flag.Bool("nginx-plus", false, "Enable support for NGINX Plus")

	appProtect = flag.Bool("enable-app-protect", false, "Enable support for NGINX App Protect. Requires -nginx-plus.")

	ingressClass = flag.String("ingress-class", "nginx",
		`A class of the Ingress controller.

	For Kubernetes >= 1.18, a corresponding IngressClass resource with the name equal to the class must be deployed. Otherwise,
	the Ingress Controller will fail to start.
	The Ingress controller only processes resources that belong to its class - i.e. have the "ingressClassName" field resource equal to the class.

	For Kubernetes < 1.18, the Ingress Controller only processes resources that belong to its class -
	i.e have the annotation "kubernetes.io/ingress.class" (for Ingress resources)
	or field "ingressClassName" equal to the class.
	Additionally, the Ingress Controller processes resources that do not have the class set,
	which can be disabled by setting the "-use-ingress-class-only" flag

	The Ingress Controller processes all the VirtualServer/VirtualServerRoute/TransportServer resources that do not have the "ingressClassName" field for all versions of kubernetes.`)

	useIngressClassOnly = flag.Bool("use-ingress-class-only", false,
		`For kubernetes versions >= 1.18 this flag will be IGNORED.

	Ignore Ingress resources without the "kubernetes.io/ingress.class" annotation`)

	defaultServerSecret = flag.String("default-server-tls-secret", "",
		`A Secret with a TLS certificate and key for TLS termination of the default server. Format: <namespace>/<name>.
	If not set, than the certificate and key in the file "/etc/nginx/secrets/default" are used.
	If "/etc/nginx/secrets/default" doesn't exist, the Ingress Controller will configure NGINX to reject TLS connections to the default server.
	If a secret is set, but the Ingress controller is not able to fetch it from Kubernetes API or it is not set and the Ingress Controller
	fails to read the file "/etc/nginx/secrets/default", the Ingress controller will fail to start.`)

	versionFlag = flag.Bool("version", false, "Print the version, git-commit hash and build date and exit")

	mainTemplatePath = flag.String("main-template-path", "",
		`Path to the main NGINX configuration template. (default for NGINX "nginx.tmpl"; default for NGINX Plus "nginx-plus.tmpl")`)

	ingressTemplatePath = flag.String("ingress-template-path", "",
		`Path to the ingress NGINX configuration template for an ingress resource.
	(default for NGINX "nginx.ingress.tmpl"; default for NGINX Plus "nginx-plus.ingress.tmpl")`)

	virtualServerTemplatePath = flag.String("virtualserver-template-path", "",
		`Path to the VirtualServer NGINX configuration template for a VirtualServer resource.
	(default for NGINX "nginx.virtualserver.tmpl"; default for NGINX Plus "nginx-plus.virtualserver.tmpl")`)

	transportServerTemplatePath = flag.String("transportserver-template-path", "",
		`Path to the TransportServer NGINX configuration template for a TransportServer resource.
	(default for NGINX "nginx.transportserver.tmpl"; default for NGINX Plus "nginx-plus.transportserver.tmpl")`)

	externalService = flag.String("external-service", "",
		`Specifies the name of the service with the type LoadBalancer through which the Ingress controller pods are exposed externally.
	The external address of the service is used when reporting the status of Ingress, VirtualServer and VirtualServerRoute resources. For Ingress resources only: Requires -report-ingress-status.`)

	ingressLink = flag.String("ingresslink", "",
		`Specifies the name of the IngressLink resource, which exposes the Ingress Controller pods via a BIG-IP system.
	The IP of the BIG-IP system is used when reporting the status of Ingress, VirtualServer and VirtualServerRoute resources. For Ingress resources only: Requires -report-ingress-status.`)

	reportIngressStatus = flag.Bool("report-ingress-status", false,
		"Updates the address field in the status of Ingress resources. Requires the -external-service or -ingresslink flag, or the 'external-status-address' key in the ConfigMap.")

	leaderElectionEnabled = flag.Bool("enable-leader-election", true,
		"Enable Leader election to avoid multiple replicas of the controller reporting the status of Ingress, VirtualServer and VirtualServerRoute resources -- only one replica will report status (default true). See -report-ingress-status flag.")

	leaderElectionLockName = flag.String("leader-election-lock-name", "nginx-ingress-leader-election",
		`Specifies the name of the ConfigMap, within the same namespace as the controller, used as the lock for leader election. Requires -enable-leader-election.`)

	nginxStatusAllowCIDRs = flag.String("nginx-status-allow-cidrs", "127.0.0.1", `Add IPv4 IP/CIDR blocks to the allow list for NGINX stub_status or the NGINX Plus API. Separate multiple IP/CIDR by commas.`)

	nginxStatusPort = flag.Int("nginx-status-port", 8080,
		"Set the port where the NGINX stub_status or the NGINX Plus API is exposed. [1024 - 65535]")

	nginxStatus = flag.Bool("nginx-status", true,
		"Enable the NGINX stub_status, or the NGINX Plus API.")

	nginxDebug = flag.Bool("nginx-debug", false,
		"Enable debugging for NGINX. Uses the nginx-debug binary. Requires 'error-log-level: debug' in the ConfigMap.")

	nginxReloadTimeout = flag.Int("nginx-reload-timeout", 60000,
		`The timeout in milliseconds which the Ingress Controller will wait for a successful NGINX reload after a change or at the initial start. (default 60000)`)

	wildcardTLSSecret = flag.String("wildcard-tls-secret", "",
		`A Secret with a TLS certificate and key for TLS termination of every Ingress host for which TLS termination is enabled but the Secret is not specified.
		Format: <namespace>/<name>. If the argument is not set, for such Ingress hosts NGINX will break any attempt to establish a TLS connection.
		If the argument is set, but the Ingress controller is not able to fetch the Secret from Kubernetes API, the Ingress controller will fail to start.`)

	enablePrometheusMetrics = flag.Bool("enable-prometheus-metrics", false,
		"Enable exposing NGINX or NGINX Plus metrics in the Prometheus format")

	prometheusTLSSecretName = flag.String("prometheus-tls-secret", "",
		`A Secret with a TLS certificate and key for TLS termination of the prometheus endpoint.`)

	prometheusMetricsListenPort = flag.Int("prometheus-metrics-listen-port", 9113,
		"Set the port where the Prometheus metrics are exposed. [1024 - 65535]")

	enableCustomResources = flag.Bool("enable-custom-resources", true,
		"Enable custom resources")

	enablePreviewPolicies = flag.Bool("enable-preview-policies", false,
		"Enable preview policies")

	enableSnippets = flag.Bool("enable-snippets", false,
		"Enable custom NGINX configuration snippets in VirtualServer, VirtualServerRoute and TransportServer resources.")

	globalConfiguration = flag.String("global-configuration", "",
		`The namespace/name of the GlobalConfiguration resource for global configuration of the Ingress Controller. Requires -enable-custom-resources. Format: <namespace>/<name>`)

	enableTLSPassthrough = flag.Bool("enable-tls-passthrough", false,
		"Enable TLS Passthrough on port 443. Requires -enable-custom-resources")

	spireAgentAddress = flag.String("spire-agent-address", "",
		`Specifies the address of the running Spire agent. Requires -nginx-plus and is for use with NGINX Service Mesh only. If the flag is set,
			but the Ingress Controller is not able to connect with the Spire Agent, the Ingress Controller will fail to start.`)

	enableInternalRoutes = flag.Bool("enable-internal-routes", false,
		`Enable support for internal routes with NGINX Service Mesh. Requires -spire-agent-address and -nginx-plus. Is for use with NGINX Service Mesh only.`)

	readyStatus = flag.Bool("ready-status", true, "Enables the readiness endpoint '/nginx-ready'. The endpoint returns a success code when NGINX has loaded all the config after the startup")

	readyStatusPort = flag.Int("ready-status-port", 8081, "Set the port where the readiness endpoint is exposed. [1024 - 65535]")

	enableLatencyMetrics = flag.Bool("enable-latency-metrics", false,
		"Enable collection of latency metrics for upstreams. Requires -enable-prometheus-metrics")

	startupCheckFn func() error
)

func main() {
	flag.Parse()

	err := flag.Lookup("logtostderr").Value.Set("true")
	if err != nil {
		glog.Fatalf("Error setting logtostderr to true: %v", err)
	}

	versionInfo := fmt.Sprintf("Version=%v GitCommit=%v Date=%v", version, commit, date)
	if *versionFlag {
		fmt.Println(versionInfo)
		os.Exit(0)
	}
	glog.Infof("Starting NGINX Ingress controller %v PlusFlag=%v", versionInfo, *nginxPlus)

	if startupCheckFn != nil {
		err := startupCheckFn()
		if err != nil {
			glog.Fatalf("Failed startup check: %v", err)
		}
	}

	healthStatusURIValidationError := validateLocation(*healthStatusURI)
	if healthStatusURIValidationError != nil {
		glog.Fatalf("Invalid value for health-status-uri: %v", healthStatusURIValidationError)
	}

	statusLockNameValidationError := validateResourceName(*leaderElectionLockName)
	if statusLockNameValidationError != nil {
		glog.Fatalf("Invalid value for leader-election-lock-name: %v", statusLockNameValidationError)
	}

	statusPortValidationError := validatePort(*nginxStatusPort)
	if statusPortValidationError != nil {
		glog.Fatalf("Invalid value for nginx-status-port: %v", statusPortValidationError)
	}

	metricsPortValidationError := validatePort(*prometheusMetricsListenPort)
	if metricsPortValidationError != nil {
		glog.Fatalf("Invalid value for prometheus-metrics-listen-port: %v", metricsPortValidationError)
	}

	readyStatusPortValidationError := validatePort(*readyStatusPort)
	if readyStatusPortValidationError != nil {
		glog.Fatalf("Invalid value for ready-status-port: %v", readyStatusPortValidationError)
	}

	allowedCIDRs, err := parseNginxStatusAllowCIDRs(*nginxStatusAllowCIDRs)
	if err != nil {
		glog.Fatalf(`Invalid value for nginx-status-allow-cidrs: %v`, err)
	}

	if *enableTLSPassthrough && !*enableCustomResources {
		glog.Fatal("enable-tls-passthrough flag requires -enable-custom-resources")
	}

	if *appProtect && !*nginxPlus {
		glog.Fatal("NGINX App Protect support is for NGINX Plus only")
	}

	if *spireAgentAddress != "" && !*nginxPlus {
		glog.Fatal("spire-agent-address support is for NGINX Plus only")
	}

	if *enableInternalRoutes && *spireAgentAddress == "" {
		glog.Fatal("enable-internal-routes flag requires spire-agent-address")
	}

	if *enableLatencyMetrics && !*enablePrometheusMetrics {
		glog.Warning("enable-latency-metrics flag requires enable-prometheus-metrics, latency metrics will not be collected")
		*enableLatencyMetrics = false
	}

	if *ingressLink != "" && *externalService != "" {
		glog.Fatal("ingresslink and external-service cannot both be set")
	}

	var config *rest.Config
	if *proxyURL != "" {
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{},
			&clientcmd.ConfigOverrides{
				ClusterInfo: clientcmdapi.Cluster{
					Server: *proxyURL,
				},
			}).ClientConfig()
		if err != nil {
			glog.Fatalf("error creating client configuration: %v", err)
		}
	} else {
		if config, err = rest.InClusterConfig(); err != nil {
			glog.Fatalf("error creating client configuration: %v", err)
		}
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v.", err)
	}

	k8sVersion, err := k8s.GetK8sVersion(kubeClient)
	if err != nil {
		glog.Fatalf("error retrieving k8s version: %v", err)
	}

	minK8sVersion := minVersion("1.14.0")
	if !k8sVersion.AtLeast(minK8sVersion) {
		glog.Fatalf("Versions of Kubernetes < %v are not supported, please refer to the documentation for details on supported versions.", minK8sVersion)
	}

	// Ingress V1 is only available from k8s > 1.18
	ingressV1Version := minVersion("1.18.0")
	if k8sVersion.AtLeast(ingressV1Version) {
		*useIngressClassOnly = true
		glog.Warningln("The '-use-ingress-class-only' flag will be deprecated and has no effect on versions of kubernetes >= 1.18.0. Processing ONLY resources that have the 'ingressClassName' field in Ingress equal to the class.")

		ingressClassRes, err := kubeClient.NetworkingV1beta1().IngressClasses().Get(context.TODO(), *ingressClass, meta_v1.GetOptions{})
		if err != nil {
			glog.Fatalf("Error when getting IngressClass %v: %v", *ingressClass, err)
		}

		if ingressClassRes.Spec.Controller != k8s.IngressControllerName {
			glog.Fatalf("IngressClass with name %v has an invalid Spec.Controller %v", ingressClassRes.Name, ingressClassRes.Spec.Controller)
		}
	}

	var dynClient dynamic.Interface
	if *appProtect || *ingressLink != "" {
		dynClient, err = dynamic.NewForConfig(config)
		if err != nil {
			glog.Fatalf("Failed to create dynamic client: %v.", err)
		}
	}
	var confClient k8s_nginx.Interface
	if *enableCustomResources {
		confClient, err = k8s_nginx.NewForConfig(config)
		if err != nil {
			glog.Fatalf("Failed to create a conf client: %v", err)
		}

		// required for emitting Events for VirtualServer
		err = conf_scheme.AddToScheme(scheme.Scheme)
		if err != nil {
			glog.Fatalf("Failed to add configuration types to the scheme: %v", err)
		}
	}

	nginxConfTemplatePath := "nginx.tmpl"
	nginxIngressTemplatePath := "nginx.ingress.tmpl"
	nginxVirtualServerTemplatePath := "nginx.virtualserver.tmpl"
	nginxTransportServerTemplatePath := "nginx.transportserver.tmpl"
	if *nginxPlus {
		nginxConfTemplatePath = "nginx-plus.tmpl"
		nginxIngressTemplatePath = "nginx-plus.ingress.tmpl"
		nginxVirtualServerTemplatePath = "nginx-plus.virtualserver.tmpl"
		nginxTransportServerTemplatePath = "nginx-plus.transportserver.tmpl"
	}

	if *mainTemplatePath != "" {
		nginxConfTemplatePath = *mainTemplatePath
	}
	if *ingressTemplatePath != "" {
		nginxIngressTemplatePath = *ingressTemplatePath
	}
	if *virtualServerTemplatePath != "" {
		nginxVirtualServerTemplatePath = *virtualServerTemplatePath
	}
	if *transportServerTemplatePath != "" {
		nginxTransportServerTemplatePath = *transportServerTemplatePath
	}

	var registry *prometheus.Registry
	var managerCollector collectors.ManagerCollector
	var controllerCollector collectors.ControllerCollector
	var latencyCollector collectors.LatencyCollector
	constLabels := map[string]string{"class": *ingressClass}
	managerCollector = collectors.NewManagerFakeCollector()
	controllerCollector = collectors.NewControllerFakeCollector()
	latencyCollector = collectors.NewLatencyFakeCollector()

	if *enablePrometheusMetrics {
		registry = prometheus.NewRegistry()
		managerCollector = collectors.NewLocalManagerMetricsCollector(constLabels)
		controllerCollector = collectors.NewControllerMetricsCollector(*enableCustomResources, constLabels)
		processCollector := collectors.NewNginxProcessesMetricsCollector(constLabels)
		workQueueCollector := collectors.NewWorkQueueMetricsCollector(constLabels)

		err = managerCollector.Register(registry)
		if err != nil {
			glog.Errorf("Error registering Manager Prometheus metrics: %v", err)
		}

		err = controllerCollector.Register(registry)
		if err != nil {
			glog.Errorf("Error registering Controller Prometheus metrics: %v", err)
		}

		err = processCollector.Register(registry)
		if err != nil {
			glog.Errorf("Error registering NginxProcess Prometheus metrics: %v", err)
		}

		err = workQueueCollector.Register(registry)
		if err != nil {
			glog.Errorf("Error registering WorkQueue Prometheus metrics: %v", err)
		}
	}

	useFakeNginxManager := *proxyURL != ""
	var nginxManager nginx.Manager
	if useFakeNginxManager {
		nginxManager = nginx.NewFakeManager("/etc/nginx")
	} else {
		timeout := time.Duration(*nginxReloadTimeout) * time.Millisecond
		nginxManager = nginx.NewLocalManager("/etc/nginx/", *nginxDebug, managerCollector, timeout)
	}
	nginxVersion := nginxManager.Version()
	isPlus := strings.Contains(nginxVersion, "plus")
	glog.Infof("Using %s", nginxVersion)

	if *nginxPlus && !isPlus {
		glog.Fatal("NGINX Plus flag enabled (-nginx-plus) without NGINX Plus binary")
	} else if !*nginxPlus && isPlus {
		glog.Fatal("NGINX Plus binary found without NGINX Plus flag (-nginx-plus)")
	}

	templateExecutor, err := version1.NewTemplateExecutor(nginxConfTemplatePath, nginxIngressTemplatePath)
	if err != nil {
		glog.Fatalf("Error creating TemplateExecutor: %v", err)
	}

	templateExecutorV2, err := version2.NewTemplateExecutor(nginxVirtualServerTemplatePath, nginxTransportServerTemplatePath)
	if err != nil {
		glog.Fatalf("Error creating TemplateExecutorV2: %v", err)
	}

	var aPPluginDone chan error
	var aPAgentDone chan error

	if *appProtect {
		aPPluginDone = make(chan error, 1)
		aPAgentDone = make(chan error, 1)

		nginxManager.AppProtectAgentStart(aPAgentDone, *nginxDebug)
		nginxManager.AppProtectPluginStart(aPPluginDone)
	}

	var sslRejectHandshake bool

	if *defaultServerSecret != "" {
		secret, err := getAndValidateSecret(kubeClient, *defaultServerSecret)
		if err != nil {
			glog.Fatalf("Error trying to get the default server TLS secret %v: %v", *defaultServerSecret, err)
		}

		bytes := configs.GenerateCertAndKeyFileContent(secret)
		nginxManager.CreateSecret(configs.DefaultServerSecretName, bytes, nginx.TLSSecretFileMode)
	} else {
		_, err := os.Stat(configs.DefaultServerSecretPath)
		if err != nil {
			if os.IsNotExist(err) {
				// file doesn't exist - it is OK! we will reject TLS connections in the default server
				sslRejectHandshake = true
			} else {
				glog.Fatalf("Error checking the default server TLS cert and key in %s: %v", configs.DefaultServerSecretPath, err)
			}
		}
	}

	if *wildcardTLSSecret != "" {
		secret, err := getAndValidateSecret(kubeClient, *wildcardTLSSecret)
		if err != nil {
			glog.Fatalf("Error trying to get the wildcard TLS secret %v: %v", *wildcardTLSSecret, err)
		}

		bytes := configs.GenerateCertAndKeyFileContent(secret)
		nginxManager.CreateSecret(configs.WildcardSecretName, bytes, nginx.TLSSecretFileMode)
	}

	var prometheusSecret *api_v1.Secret
	if *prometheusTLSSecretName != "" {
		prometheusSecret, err = getAndValidateSecret(kubeClient, *prometheusTLSSecretName)
		if err != nil {
			glog.Fatalf("Error trying to get the prometheus TLS secret %v: %v", *prometheusTLSSecretName, err)
		}
	}

	globalConfigurationValidator := createGlobalConfigurationValidator()

	if *globalConfiguration != "" {
		_, _, err := k8s.ParseNamespaceName(*globalConfiguration)
		if err != nil {
			glog.Fatalf("Error parsing the global-configuration argument: %v", err)
		}

		if !*enableCustomResources {
			glog.Fatal("global-configuration flag requires -enable-custom-resources")
		}
	}

	cfgParams := configs.NewDefaultConfigParams()

	if *nginxConfigMaps != "" {
		ns, name, err := k8s.ParseNamespaceName(*nginxConfigMaps)
		if err != nil {
			glog.Fatalf("Error parsing the nginx-configmaps argument: %v", err)
		}
		cfm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(context.TODO(), name, meta_v1.GetOptions{})
		if err != nil {
			glog.Fatalf("Error when getting %v: %v", *nginxConfigMaps, err)
		}
		cfgParams = configs.ParseConfigMap(cfm, *nginxPlus, *appProtect)
		if cfgParams.MainServerSSLDHParamFileContent != nil {
			fileName, err := nginxManager.CreateDHParam(*cfgParams.MainServerSSLDHParamFileContent)
			if err != nil {
				glog.Fatalf("Configmap %s/%s: Could not update dhparams: %v", ns, name, err)
			} else {
				cfgParams.MainServerSSLDHParam = fileName
			}
		}
		if cfgParams.MainTemplate != nil {
			err = templateExecutor.UpdateMainTemplate(cfgParams.MainTemplate)
			if err != nil {
				glog.Fatalf("Error updating NGINX main template: %v", err)
			}
		}
		if cfgParams.IngressTemplate != nil {
			err = templateExecutor.UpdateIngressTemplate(cfgParams.IngressTemplate)
			if err != nil {
				glog.Fatalf("Error updating ingress template: %v", err)
			}
		}
	}
	staticCfgParams := &configs.StaticConfigParams{
		HealthStatus:                   *healthStatus,
		HealthStatusURI:                *healthStatusURI,
		NginxStatus:                    *nginxStatus,
		NginxStatusAllowCIDRs:          allowedCIDRs,
		NginxStatusPort:                *nginxStatusPort,
		StubStatusOverUnixSocketForOSS: *enablePrometheusMetrics,
		TLSPassthrough:                 *enableTLSPassthrough,
		EnableSnippets:                 *enableSnippets,
		NginxServiceMesh:               *spireAgentAddress != "",
		MainAppProtectLoadModule:       *appProtect,
		EnableLatencyMetrics:           *enableLatencyMetrics,
		EnablePreviewPolicies:          *enablePreviewPolicies,
		SSLRejectHandshake:             sslRejectHandshake,
	}

	ngxConfig := configs.GenerateNginxMainConfig(staticCfgParams, cfgParams)
	content, err := templateExecutor.ExecuteMainConfigTemplate(ngxConfig)
	if err != nil {
		glog.Fatalf("Error generating NGINX main config: %v", err)
	}
	nginxManager.CreateMainConfig(content)

	nginxManager.UpdateConfigVersionFile(ngxConfig.OpenTracingLoadModule)

	nginxManager.SetOpenTracing(ngxConfig.OpenTracingLoadModule)

	if ngxConfig.OpenTracingLoadModule {
		err := nginxManager.CreateOpenTracingTracerConfig(cfgParams.MainOpenTracingTracerConfig)
		if err != nil {
			glog.Fatalf("Error creating OpenTracing tracer config file: %v", err)
		}
	}

	if *enableTLSPassthrough {
		var emptyFile []byte
		nginxManager.CreateTLSPassthroughHostsConfig(emptyFile)
	}

	nginxDone := make(chan error, 1)
	nginxManager.Start(nginxDone)

	var plusClient *client.NginxClient

	if *nginxPlus && !useFakeNginxManager {
		httpClient := getSocketClient("/var/lib/nginx/nginx-plus-api.sock")
		plusClient, err = client.NewNginxClient(httpClient, "http://nginx-plus-api/api")
		if err != nil {
			glog.Fatalf("Failed to create NginxClient for Plus: %v", err)
		}
		nginxManager.SetPlusClients(plusClient, httpClient)
	}

	var plusCollector *nginxCollector.NginxPlusCollector
	var syslogListener metrics.SyslogListener
	syslogListener = metrics.NewSyslogFakeServer()
	if *enablePrometheusMetrics {
		upstreamServerVariableLabels := []string{"service", "resource_type", "resource_name", "resource_namespace"}
		upstreamServerPeerVariableLabelNames := []string{"pod_name"}
		if staticCfgParams.NginxServiceMesh {
			upstreamServerPeerVariableLabelNames = append(upstreamServerPeerVariableLabelNames, "pod_owner")
		}
		if *nginxPlus {
			streamUpstreamServerVariableLabels := []string{"service", "resource_type", "resource_name", "resource_namespace"}
			streamUpstreamServerPeerVariableLabelNames := []string{"pod_name"}

			serverZoneVariableLabels := []string{"resource_type", "resource_name", "resource_namespace"}
			streamServerZoneVariableLabels := []string{"resource_type", "resource_name", "resource_namespace"}
			variableLabelNames := nginxCollector.NewVariableLabelNames(upstreamServerVariableLabels, serverZoneVariableLabels, upstreamServerPeerVariableLabelNames,
				streamUpstreamServerVariableLabels, streamServerZoneVariableLabels, streamUpstreamServerPeerVariableLabelNames)
			plusCollector = nginxCollector.NewNginxPlusCollector(plusClient, "nginx_ingress_nginxplus", variableLabelNames, constLabels)
			go metrics.RunPrometheusListenerForNginxPlus(*prometheusMetricsListenPort, plusCollector, registry, prometheusSecret)
		} else {
			httpClient := getSocketClient("/var/lib/nginx/nginx-status.sock")
			client, err := metrics.NewNginxMetricsClient(httpClient)
			if err != nil {
				glog.Errorf("Error creating the Nginx client for Prometheus metrics: %v", err)
			}
			go metrics.RunPrometheusListenerForNginx(*prometheusMetricsListenPort, client, registry, constLabels, prometheusSecret)
		}
		if *enableLatencyMetrics {
			latencyCollector = collectors.NewLatencyMetricsCollector(constLabels, upstreamServerVariableLabels, upstreamServerPeerVariableLabelNames)
			if err := latencyCollector.Register(registry); err != nil {
				glog.Errorf("Error registering Latency Prometheus metrics: %v", err)
			}
			syslogListener = metrics.NewLatencyMetricsListener("/var/lib/nginx/nginx-syslog.sock", latencyCollector)
			go syslogListener.Run()
		}
	}

	isWildcardEnabled := *wildcardTLSSecret != ""
	cnf := configs.NewConfigurator(nginxManager, staticCfgParams, cfgParams, templateExecutor,
		templateExecutorV2, *nginxPlus, isWildcardEnabled, plusCollector, *enablePrometheusMetrics, latencyCollector, *enableLatencyMetrics)
	controllerNamespace := os.Getenv("POD_NAMESPACE")

	transportServerValidator := cr_validation.NewTransportServerValidator(*enableTLSPassthrough, *enableSnippets, *nginxPlus)
	virtualServerValidator := cr_validation.NewVirtualServerValidator(*nginxPlus)

	lbcInput := k8s.NewLoadBalancerControllerInput{
		KubeClient:                   kubeClient,
		ConfClient:                   confClient,
		DynClient:                    dynClient,
		ResyncPeriod:                 30 * time.Second,
		Namespace:                    *watchNamespace,
		NginxConfigurator:            cnf,
		DefaultServerSecret:          *defaultServerSecret,
		AppProtectEnabled:            *appProtect,
		IsNginxPlus:                  *nginxPlus,
		IngressClass:                 *ingressClass,
		UseIngressClassOnly:          *useIngressClassOnly,
		ExternalServiceName:          *externalService,
		IngressLink:                  *ingressLink,
		ControllerNamespace:          controllerNamespace,
		ReportIngressStatus:          *reportIngressStatus,
		IsLeaderElectionEnabled:      *leaderElectionEnabled,
		LeaderElectionLockName:       *leaderElectionLockName,
		WildcardTLSSecret:            *wildcardTLSSecret,
		ConfigMaps:                   *nginxConfigMaps,
		GlobalConfiguration:          *globalConfiguration,
		AreCustomResourcesEnabled:    *enableCustomResources,
		EnablePreviewPolicies:        *enablePreviewPolicies,
		MetricsCollector:             controllerCollector,
		GlobalConfigurationValidator: globalConfigurationValidator,
		TransportServerValidator:     transportServerValidator,
		VirtualServerValidator:       virtualServerValidator,
		SpireAgentAddress:            *spireAgentAddress,
		InternalRoutesEnabled:        *enableInternalRoutes,
		IsPrometheusEnabled:          *enablePrometheusMetrics,
		IsLatencyMetricsEnabled:      *enableLatencyMetrics,
		IsTLSPassthroughEnabled:      *enableTLSPassthrough,
	}

	lbc := k8s.NewLoadBalancerController(lbcInput)

	if *readyStatus {
		go func() {
			port := fmt.Sprintf(":%v", *readyStatusPort)
			s := http.NewServeMux()
			s.HandleFunc("/nginx-ready", ready(lbc))
			glog.Fatal(http.ListenAndServe(port, s))
		}()
	}

	if *appProtect {
		go handleTerminationWithAppProtect(lbc, nginxManager, syslogListener, nginxDone, aPAgentDone, aPPluginDone)
	} else {
		go handleTermination(lbc, nginxManager, syslogListener, nginxDone)
	}

	lbc.Run()

	for {
		glog.Info("Waiting for the controller to exit...")
		time.Sleep(30 * time.Second)
	}
}

func createGlobalConfigurationValidator() *cr_validation.GlobalConfigurationValidator {
	forbiddenListenerPorts := map[int]bool{
		80:  true,
		443: true,
	}

	if *nginxStatus {
		forbiddenListenerPorts[*nginxStatusPort] = true
	}
	if *enablePrometheusMetrics {
		forbiddenListenerPorts[*prometheusMetricsListenPort] = true
	}

	return cr_validation.NewGlobalConfigurationValidator(forbiddenListenerPorts)
}

func handleTermination(lbc *k8s.LoadBalancerController, nginxManager nginx.Manager, listener metrics.SyslogListener, nginxDone chan error) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)

	exitStatus := 0
	exited := false

	select {
	case err := <-nginxDone:
		if err != nil {
			glog.Errorf("nginx command exited with an error: %v", err)
			exitStatus = 1
		} else {
			glog.Info("nginx command exited successfully")
		}
		exited = true
	case <-signalChan:
		glog.Info("Received SIGTERM, shutting down")
	}

	glog.Info("Shutting down the controller")
	lbc.Stop()

	if !exited {
		glog.Info("Shutting down NGINX")
		nginxManager.Quit()
		<-nginxDone
	}
	listener.Stop()

	glog.Infof("Exiting with a status: %v", exitStatus)
	os.Exit(exitStatus)
}

// getSocketClient gets an http.Client with the a unix socket transport.
func getSocketClient(sockPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", sockPath)
			},
		},
	}
}

// validateResourceName validates the name of a resource
func validateResourceName(lock string) error {
	allErrs := validation.IsDNS1123Subdomain(lock)
	if len(allErrs) > 0 {
		return fmt.Errorf("invalid resource name %v: %v", lock, allErrs)
	}
	return nil
}

// validatePort makes sure a given port is inside the valid port range for its usage
func validatePort(port int) error {
	if port < 1024 || port > 65535 {
		return fmt.Errorf("port outside of valid port range [1024 - 65535]: %v", port)
	}
	return nil
}

// parseNginxStatusAllowCIDRs converts a comma separated CIDR/IP address string into an array of CIDR/IP addresses.
// It returns an array of the valid CIDR/IP addresses or an error if given an invalid address.
func parseNginxStatusAllowCIDRs(input string) (cidrs []string, err error) {
	cidrsArray := strings.Split(input, ",")
	for _, cidr := range cidrsArray {
		trimmedCidr := strings.TrimSpace(cidr)
		err := validateCIDRorIP(trimmedCidr)
		if err != nil {
			return cidrs, err
		}
		cidrs = append(cidrs, trimmedCidr)
	}
	return cidrs, nil
}

// validateCIDRorIP makes sure a given string is either a valid CIDR block or IP address.
// It an error if it is not valid.
func validateCIDRorIP(cidr string) error {
	if cidr == "" {
		return fmt.Errorf("invalid CIDR address: an empty string is an invalid CIDR block or IP address")
	}
	_, _, err := net.ParseCIDR(cidr)
	if err == nil {
		return nil
	}
	ip := net.ParseIP(cidr)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %v", cidr)
	}
	return nil
}

// getAndValidateSecret gets and validates a secret.
func getAndValidateSecret(kubeClient *kubernetes.Clientset, secretNsName string) (secret *api_v1.Secret, err error) {
	ns, name, err := k8s.ParseNamespaceName(secretNsName)
	if err != nil {
		return nil, fmt.Errorf("could not parse the %v argument: %w", secretNsName, err)
	}
	secret, err = kubeClient.CoreV1().Secrets(ns).Get(context.TODO(), name, meta_v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get %v: %w", secretNsName, err)
	}
	err = secrets.ValidateTLSSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("%v is invalid: %w", secretNsName, err)
	}
	return secret, nil
}

const (
	locationFmt    = `/[^\s{};]*`
	locationErrMsg = "must start with / and must not include any whitespace character, `{`, `}` or `;`"
)

var locationRegexp = regexp.MustCompile("^" + locationFmt + "$")

func validateLocation(location string) error {
	if location == "" || location == "/" {
		return fmt.Errorf("invalid location format: '%v' is an invalid location", location)
	}
	if !locationRegexp.MatchString(location) {
		msg := validation.RegexError(locationErrMsg, locationFmt, "/path", "/path/subpath-123")
		return fmt.Errorf("invalid location format: %v", msg)
	}
	return nil
}

func handleTerminationWithAppProtect(lbc *k8s.LoadBalancerController, nginxManager nginx.Manager, listener metrics.SyslogListener, nginxDone, agentDone, pluginDone chan error) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)

	select {
	case err := <-nginxDone:
		glog.Fatalf("nginx command exited unexpectedly with status: %v", err)
	case err := <-pluginDone:
		glog.Fatalf("AppProtectPlugin command exited unexpectedly with status: %v", err)
	case err := <-agentDone:
		glog.Fatalf("AppProtectAgent command exited unexpectedly with status: %v", err)
	case <-signalChan:
		glog.Infof("Received SIGTERM, shutting down")
		lbc.Stop()
		nginxManager.Quit()
		<-nginxDone
		nginxManager.AppProtectPluginQuit()
		<-pluginDone
		nginxManager.AppProtectAgentQuit()
		<-agentDone
		listener.Stop()
	}
	glog.Info("Exiting successfully")
	os.Exit(0)
}

func ready(lbc *k8s.LoadBalancerController) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if !lbc.IsNginxReady() {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Ready")
	}
}

func minVersion(min string) (v *util_version.Version) {
	minVer, err := util_version.ParseGeneric(min)
	if err != nil {
		glog.Fatalf("unexpected error parsing minimum supported version: %v", err)
	}

	return minVer
}

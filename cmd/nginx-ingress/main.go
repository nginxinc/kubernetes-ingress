package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
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
	util_version "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func main() {
	parseFlags()

	var config *rest.Config
	var err error
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
	glog.Infof("Kubernetes version: %v", k8sVersion)

	minK8sVersion, err := util_version.ParseGeneric("1.19.0")
	if err != nil {
		glog.Fatalf("unexpected error parsing minimum supported version: %v", err)
	}

	if !k8sVersion.AtLeast(minK8sVersion) {
		glog.Fatalf("Versions of Kubernetes < %v are not supported, please refer to the documentation for details on supported versions and legacy controller support.", minK8sVersion)
	}

	ingressClassRes, err := kubeClient.NetworkingV1().IngressClasses().Get(context.TODO(), *ingressClass, meta_v1.GetOptions{})
	if err != nil {
		glog.Fatalf("Error when getting IngressClass %v: %v", *ingressClass, err)
	}

	if ingressClassRes.Spec.Controller != k8s.IngressControllerName {
		glog.Fatalf("IngressClass with name %v has an invalid Spec.Controller %v; expected %v", ingressClassRes.Name, ingressClassRes.Spec.Controller, k8s.IngressControllerName)
	}

	var dynClient dynamic.Interface
	if *appProtectDos || *appProtect || *ingressLink != "" {
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

	managerCollector, controllerCollector, registry = registerPrometheusMetrics(managerCollector, controllerCollector, constLabels)

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

		nginxManager.AppProtectAgentStart(aPAgentDone, *appProtectLogLevel)
		nginxManager.AppProtectPluginStart(aPPluginDone)
	}

	var aPPDosAgentDone chan error

	if *appProtectDos {
		aPPDosAgentDone = make(chan error, 1)
		nginxManager.AppProtectDosAgentStart(aPPDosAgentDone, *appProtectDosDebug, *appProtectDosMaxDaemons, *appProtectDosMaxWorkers, *appProtectDosMemory)
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

	cfgParams := configs.NewDefaultConfigParams(*nginxPlus)

	if *nginxConfigMaps != "" {
		ns, name, err := k8s.ParseNamespaceName(*nginxConfigMaps)
		if err != nil {
			glog.Fatalf("Error parsing the nginx-configmaps argument: %v", err)
		}
		cfm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(context.TODO(), name, meta_v1.GetOptions{})
		if err != nil {
			glog.Fatalf("Error when getting %v: %v", *nginxConfigMaps, err)
		}
		cfgParams = configs.ParseConfigMap(cfm, *nginxPlus, *appProtect, *appProtectDos)
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
		MainAppProtectDosLoadModule:    *appProtectDos,
		EnableLatencyMetrics:           *enableLatencyMetrics,
		EnableOIDC:                     *enableOIDC,
		SSLRejectHandshake:             sslRejectHandshake,
		EnableCertManager:              *enableCertManager,
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

	var syslogListener metrics.SyslogListener
	syslogListener = metrics.NewSyslogFakeServer()
	plusCollector, syslogListener, latencyCollector := processPrometheusMetrics(managerCollector, controllerCollector,
		latencyCollector, registry, syslogListener, constLabels, kubeClient, plusClient, staticCfgParams.NginxServiceMesh)

	isWildcardEnabled := *wildcardTLSSecret != ""
	cnf := configs.NewConfigurator(nginxManager, staticCfgParams, cfgParams, templateExecutor,
		templateExecutorV2, *nginxPlus, isWildcardEnabled, plusCollector, *enablePrometheusMetrics, latencyCollector, *enableLatencyMetrics)
	controllerNamespace := os.Getenv("POD_NAMESPACE")

	transportServerValidator := cr_validation.NewTransportServerValidator(*enableTLSPassthrough, *enableSnippets, *nginxPlus)
	virtualServerValidator := cr_validation.NewVirtualServerValidator(cr_validation.IsPlus(*nginxPlus), cr_validation.IsDosEnabled(*appProtectDos), cr_validation.IsCertManagerEnabled(*enableCertManager))

	lbcInput := k8s.NewLoadBalancerControllerInput{
		KubeClient:                   kubeClient,
		ConfClient:                   confClient,
		DynClient:                    dynClient,
		RestConfig:                   config,
		ResyncPeriod:                 30 * time.Second,
		Namespace:                    *watchNamespace,
		NginxConfigurator:            cnf,
		DefaultServerSecret:          *defaultServerSecret,
		AppProtectEnabled:            *appProtect,
		AppProtectDosEnabled:         *appProtectDos,
		IsNginxPlus:                  *nginxPlus,
		IngressClass:                 *ingressClass,
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
		EnableOIDC:                   *enableOIDC,
		MetricsCollector:             controllerCollector,
		GlobalConfigurationValidator: globalConfigurationValidator,
		TransportServerValidator:     transportServerValidator,
		VirtualServerValidator:       virtualServerValidator,
		SpireAgentAddress:            *spireAgentAddress,
		InternalRoutesEnabled:        *enableInternalRoutes,
		IsPrometheusEnabled:          *enablePrometheusMetrics,
		IsLatencyMetricsEnabled:      *enableLatencyMetrics,
		IsTLSPassthroughEnabled:      *enableTLSPassthrough,
		SnippetsEnabled:              *enableSnippets,
		CertManagerEnabled:           *enableCertManager,
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

	if *appProtect || *appProtectDos {
		go handleTerminationWithAppProtect(lbc, nginxManager, syslogListener, nginxDone, aPAgentDone, aPPluginDone, aPPDosAgentDone, *appProtect, *appProtectDos)
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

func handleTerminationWithAppProtect(lbc *k8s.LoadBalancerController, nginxManager nginx.Manager, listener metrics.SyslogListener, nginxDone, agentDone, pluginDone, agentDosDone chan error, appProtectEnabled, appProtectDosEnabled bool) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)

	select {
	case err := <-nginxDone:
		glog.Fatalf("nginx command exited unexpectedly with status: %v", err)
	case err := <-pluginDone:
		glog.Fatalf("AppProtectPlugin command exited unexpectedly with status: %v", err)
	case err := <-agentDone:
		glog.Fatalf("AppProtectAgent command exited unexpectedly with status: %v", err)
	case err := <-agentDosDone:
		glog.Fatalf("AppProtectDosAgent command exited unexpectedly with status: %v", err)
	case <-signalChan:
		glog.Infof("Received SIGTERM, shutting down")
		lbc.Stop()
		nginxManager.Quit()
		<-nginxDone
		if appProtectEnabled {
			nginxManager.AppProtectPluginQuit()
			<-pluginDone
			nginxManager.AppProtectAgentQuit()
			<-agentDone
		}
		if appProtectDosEnabled {
			nginxManager.AppProtectDosAgentQuit()
			<-agentDosDone
		}
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

func registerPrometheusMetrics(
	mc collectors.ManagerCollector,
	cc collectors.ControllerCollector,
	constLabels map[string]string,
) (collectors.ManagerCollector, collectors.ControllerCollector, *prometheus.Registry) {
	var registry *prometheus.Registry
	var err error

	if *enablePrometheusMetrics {
		registry = prometheus.NewRegistry()
		mc = collectors.NewLocalManagerMetricsCollector(constLabels)
		cc = collectors.NewControllerMetricsCollector(*enableCustomResources, constLabels)
		processCollector := collectors.NewNginxProcessesMetricsCollector(constLabels)
		workQueueCollector := collectors.NewWorkQueueMetricsCollector(constLabels)

		err = mc.Register(registry)
		if err != nil {
			glog.Errorf("Error registering Manager Prometheus metrics: %v", err)
		}

		err = cc.Register(registry)
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
	return mc, cc, registry
}

func processPrometheusMetrics(
	mc collectors.ManagerCollector,
	cc collectors.ControllerCollector,
	lc collectors.LatencyCollector,
	registry *prometheus.Registry,
	syslogListener metrics.SyslogListener,
	constLabels map[string]string,
	kubeClient *kubernetes.Clientset,
	plusClient *client.NginxClient,
	isMesh bool,
) (*nginxCollector.NginxPlusCollector, metrics.SyslogListener, collectors.LatencyCollector) {
	var prometheusSecret *api_v1.Secret
	var err error
	if *prometheusTLSSecretName != "" {
		prometheusSecret, err = getAndValidateSecret(kubeClient, *prometheusTLSSecretName)
		if err != nil {
			glog.Fatalf("Error trying to get the prometheus TLS secret %v: %v", *prometheusTLSSecretName, err)
		}
	}

	var plusCollector *nginxCollector.NginxPlusCollector
	if *enablePrometheusMetrics {
		upstreamServerVariableLabels := []string{"service", "resource_type", "resource_name", "resource_namespace"}
		upstreamServerPeerVariableLabelNames := []string{"pod_name"}
		if isMesh {
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
			lc = collectors.NewLatencyMetricsCollector(constLabels, upstreamServerVariableLabels, upstreamServerPeerVariableLabelNames)
			if err := lc.Register(registry); err != nil {
				glog.Errorf("Error registering Latency Prometheus metrics: %v", err)
			}
			syslogListener = metrics.NewLatencyMetricsListener("/var/lib/nginx/nginx-syslog.sock", lc)
			go syslogListener.Run()
		}
	}

	return plusCollector, syslogListener, lc
}

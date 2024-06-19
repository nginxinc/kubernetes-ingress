package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/healthcheck"
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
	"github.com/prometheus/common/promlog"
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

// Injected during build
var (
	version           string
	telemetryEndpoint string
)

const (
	nginxVersionLabel        = "app.nginx.org/version"
	versionLabel             = "app.kubernetes.io/version"
	appProtectVersionLabel   = "appprotect.f5.com/version"
	agentVersionLabel        = "app.nginx.org/agent-version"
	appProtectVersionPath    = "/opt/app_protect/RELEASE"
	appProtectv4BundleFolder = "/etc/nginx/waf/bundles/"
	appProtectv5BundleFolder = "/etc/app_protect/bundles/"
)

func main() {
	commitHash, commitTime, dirtyBuild := getBuildInfo()
	fmt.Printf("NGINX Ingress Controller Version=%v Commit=%v Date=%v DirtyState=%v Arch=%v/%v Go=%v\n", version, commitHash, commitTime, dirtyBuild, runtime.GOOS, runtime.GOARCH, runtime.Version())

	parseFlags()
	parsedFlags := os.Args[1:]

	config, kubeClient := createConfigAndKubeClient()

	kubernetesVersionInfo(kubeClient)

	validateIngressClass(kubeClient)

	checkNamespaces(kubeClient)

	dynClient, confClient := createCustomClients(config)

	constLabels := map[string]string{"class": *ingressClass}

	managerCollector, controllerCollector, registry := createManagerAndControllerCollectors(constLabels)

	nginxManager, useFakeNginxManager := createNginxManager(managerCollector)

	nginxVersion := getNginxVersionInfo(nginxManager)

	var appProtectVersion string
	var appProtectV5 bool
	appProtectBundlePath := appProtectv4BundleFolder
	if *appProtect {
		appProtectVersion = getAppProtectVersionInfo()

		r := regexp.MustCompile("^5.*")
		if r.MatchString(appProtectVersion) {
			appProtectV5 = true
			appProtectBundlePath = appProtectv5BundleFolder
		}
	}

	var agentVersion string
	if *agent {
		agentVersion = getAgentVersionInfo(nginxManager)
	}

	go updateSelfWithVersionInfo(kubeClient, version, appProtectVersion, agentVersion, nginxVersion, 10, time.Second*5)

	templateExecutor, templateExecutorV2 := createTemplateExecutors()

	sslRejectHandshake := processDefaultServerSecret(kubeClient, nginxManager)

	isWildcardEnabled := processWildcardSecret(kubeClient, nginxManager)

	globalConfigurationValidator := createGlobalConfigurationValidator()

	processGlobalConfiguration()

	cfgParams := configs.NewDefaultConfigParams(*nginxPlus)
	cfgParams = processConfigMaps(kubeClient, cfgParams, nginxManager, templateExecutor)

	staticCfgParams := &configs.StaticConfigParams{
		DisableIPV6:                    *disableIPV6,
		DefaultHTTPListenerPort:        *defaultHTTPListenerPort,
		DefaultHTTPSListenerPort:       *defaultHTTPSListenerPort,
		HealthStatus:                   *healthStatus,
		HealthStatusURI:                *healthStatusURI,
		NginxStatus:                    *nginxStatus,
		NginxStatusAllowCIDRs:          allowedCIDRs,
		NginxStatusPort:                *nginxStatusPort,
		StubStatusOverUnixSocketForOSS: *enablePrometheusMetrics,
		TLSPassthrough:                 *enableTLSPassthrough,
		TLSPassthroughPort:             *tlsPassthroughPort,
		EnableSnippets:                 *enableSnippets,
		NginxServiceMesh:               *spireAgentAddress != "",
		MainAppProtectLoadModule:       *appProtect,
		MainAppProtectV5LoadModule:     appProtectV5,
		MainAppProtectDosLoadModule:    *appProtectDos,
		MainAppProtectV5EnforcerAddr:   *appProtectEnforcerAddress,
		EnableLatencyMetrics:           *enableLatencyMetrics,
		EnableOIDC:                     *enableOIDC,
		SSLRejectHandshake:             sslRejectHandshake,
		EnableCertManager:              *enableCertManager,
		DynamicSSLReload:               *enableDynamicSSLReload,
		DynamicWeightChangesReload:     *enableDynamicWeightChangesReload,
		StaticSSLPath:                  nginxManager.GetSecretsDir(),
		NginxVersion:                   nginxVersion,
		AppProtectBundlePath:           appProtectBundlePath,
	}

	processNginxConfig(staticCfgParams, cfgParams, templateExecutor, nginxManager)

	if *enableTLSPassthrough {
		var emptyFile []byte
		nginxManager.CreateTLSPassthroughHostsConfig(emptyFile)
	}

	process := startChildProcesses(nginxManager, appProtectV5)

	plusClient := createPlusClient(*nginxPlus, useFakeNginxManager, nginxManager)

	plusCollector, syslogListener, latencyCollector := createPlusAndLatencyCollectors(registry, constLabels, kubeClient, plusClient, staticCfgParams.NginxServiceMesh)
	cnf := configs.NewConfigurator(configs.ConfiguratorParams{
		NginxManager:                        nginxManager,
		StaticCfgParams:                     staticCfgParams,
		Config:                              cfgParams,
		TemplateExecutor:                    templateExecutor,
		TemplateExecutorV2:                  templateExecutorV2,
		LatencyCollector:                    latencyCollector,
		LabelUpdater:                        plusCollector,
		IsPlus:                              *nginxPlus,
		IsWildcardEnabled:                   isWildcardEnabled,
		IsPrometheusEnabled:                 *enablePrometheusMetrics,
		IsLatencyMetricsEnabled:             *enableLatencyMetrics,
		IsDynamicSSLReloadEnabled:           *enableDynamicSSLReload,
		IsDynamicWeightChangesReloadEnabled: *enableDynamicWeightChangesReload,
		NginxVersion:                        nginxVersion,
	})

	controllerNamespace := os.Getenv("POD_NAMESPACE")

	transportServerValidator := cr_validation.NewTransportServerValidator(*enableTLSPassthrough, *enableSnippets, *nginxPlus)
	virtualServerValidator := cr_validation.NewVirtualServerValidator(
		cr_validation.IsPlus(*nginxPlus),
		cr_validation.IsDosEnabled(*appProtectDos),
		cr_validation.IsCertManagerEnabled(*enableCertManager),
		cr_validation.IsExternalDNSEnabled(*enableExternalDNS),
	)

	if *enableServiceInsight {
		createHealthProbeEndpoint(kubeClient, plusClient, cnf)
	}

	lbcInput := k8s.NewLoadBalancerControllerInput{
		KubeClient:                   kubeClient,
		ConfClient:                   confClient,
		DynClient:                    dynClient,
		RestConfig:                   config,
		ResyncPeriod:                 30 * time.Second,
		Namespace:                    watchNamespaces,
		SecretNamespace:              watchSecretNamespaces,
		NginxConfigurator:            cnf,
		DefaultServerSecret:          *defaultServerSecret,
		AppProtectEnabled:            *appProtect,
		AppProtectDosEnabled:         *appProtectDos,
		AppProtectVersion:            appProtectVersion,
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
		TLSPassthroughPort:           *tlsPassthroughPort,
		SnippetsEnabled:              *enableSnippets,
		CertManagerEnabled:           *enableCertManager,
		ExternalDNSEnabled:           *enableExternalDNS,
		IsIPV6Disabled:               *disableIPV6,
		WatchNamespaceLabel:          *watchNamespaceLabel,
		EnableTelemetryReporting:     *enableTelemetryReporting,
		TelemetryReportingEndpoint:   telemetryEndpoint,
		NICVersion:                   version,
		DynamicWeightChangesReload:   *enableDynamicWeightChangesReload,
		InstallationFlags:            parsedFlags,
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

	go handleTermination(lbc, nginxManager, syslogListener, process)

	lbc.Run()

	for {
		glog.Info("Waiting for the controller to exit...")
		time.Sleep(30 * time.Second)
	}
}

func createConfigAndKubeClient() (*rest.Config, *kubernetes.Clientset) {
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

	return config, kubeClient
}

func kubernetesVersionInfo(kubeClient kubernetes.Interface) {
	k8sVersion, err := k8s.GetK8sVersion(kubeClient)
	if err != nil {
		glog.Fatalf("error retrieving k8s version: %v", err)
	}
	glog.Infof("Kubernetes version: %v", k8sVersion)

	minK8sVersion, err := util_version.ParseGeneric("1.22.0")
	if err != nil {
		glog.Fatalf("unexpected error parsing minimum supported version: %v", err)
	}

	if !k8sVersion.AtLeast(minK8sVersion) {
		glog.Fatalf("Versions of Kubernetes < %v are not supported, please refer to the documentation for details on supported versions and legacy controller support.", minK8sVersion)
	}
}

func validateIngressClass(kubeClient kubernetes.Interface) {
	ingressClassRes, err := kubeClient.NetworkingV1().IngressClasses().Get(context.TODO(), *ingressClass, meta_v1.GetOptions{})
	if err != nil {
		glog.Fatalf("Error when getting IngressClass %v: %v", *ingressClass, err)
	}

	if ingressClassRes.Spec.Controller != k8s.IngressControllerName {
		glog.Fatalf("IngressClass with name %v has an invalid Spec.Controller %v; expected %v", ingressClassRes.Name, ingressClassRes.Spec.Controller, k8s.IngressControllerName)
	}
}

func checkNamespaces(kubeClient kubernetes.Interface) {
	if *watchNamespaceLabel != "" {
		// bootstrap the watched namespace list
		var newWatchNamespaces []string
		nsList, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), meta_v1.ListOptions{LabelSelector: *watchNamespaceLabel})
		if err != nil {
			glog.Errorf("error when getting Namespaces with the label selector %v: %v", watchNamespaceLabel, err)
		}
		for _, ns := range nsList.Items {
			newWatchNamespaces = append(newWatchNamespaces, ns.Name)
		}
		watchNamespaces = newWatchNamespaces
		glog.Infof("Namespaces watched using label %v: %v", *watchNamespaceLabel, watchNamespaces)
	} else {
		checkNamespaceExists(kubeClient, watchNamespaces)
	}
	checkNamespaceExists(kubeClient, watchSecretNamespaces)
}

func checkNamespaceExists(kubeClient kubernetes.Interface, namespaces []string) {
	for _, ns := range namespaces {
		if ns != "" {
			_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), ns, meta_v1.GetOptions{})
			if err != nil {
				glog.Warningf("Error when getting Namespace %v: %v", ns, err)
			}
		}
	}
}

func createCustomClients(config *rest.Config) (dynamic.Interface, k8s_nginx.Interface) {
	var dynClient dynamic.Interface
	var err error
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
	return dynClient, confClient
}

func createPlusClient(nginxPlus bool, useFakeNginxManager bool, nginxManager nginx.Manager) *client.NginxClient {
	var plusClient *client.NginxClient
	var err error

	if nginxPlus && !useFakeNginxManager {
		httpClient := getSocketClient("/var/lib/nginx/nginx-plus-api.sock")
		plusClient, err = client.NewNginxClient("http://nginx-plus-api/api", client.WithHTTPClient(httpClient))
		if err != nil {
			glog.Fatalf("Failed to create NginxClient for Plus: %v", err)
		}
		nginxManager.SetPlusClients(plusClient, httpClient)
	}
	return plusClient
}

func createTemplateExecutors() (*version1.TemplateExecutor, *version2.TemplateExecutor) {
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

	templateExecutor, err := version1.NewTemplateExecutor(nginxConfTemplatePath, nginxIngressTemplatePath)
	if err != nil {
		glog.Fatalf("Error creating TemplateExecutor: %v", err)
	}

	templateExecutorV2, err := version2.NewTemplateExecutor(nginxVirtualServerTemplatePath, nginxTransportServerTemplatePath)
	if err != nil {
		glog.Fatalf("Error creating TemplateExecutorV2: %v", err)
	}

	return templateExecutor, templateExecutorV2
}

func createNginxManager(managerCollector collectors.ManagerCollector) (nginx.Manager, bool) {
	useFakeNginxManager := *proxyURL != ""
	var nginxManager nginx.Manager
	if useFakeNginxManager {
		nginxManager = nginx.NewFakeManager("/etc/nginx")
	} else {
		timeout := time.Duration(*nginxReloadTimeout) * time.Millisecond
		nginxManager = nginx.NewLocalManager("/etc/nginx/", *nginxDebug, managerCollector, timeout)
	}
	return nginxManager, useFakeNginxManager
}

func getNginxVersionInfo(nginxManager nginx.Manager) nginx.Version {
	nginxInfo := nginxManager.Version()
	glog.Infof("Using %s", nginxInfo.String())

	if *nginxPlus && !nginxInfo.IsPlus {
		glog.Fatal("NGINX Plus flag enabled (-nginx-plus) without NGINX Plus binary")
	} else if !*nginxPlus && nginxInfo.IsPlus {
		glog.Fatal("NGINX Plus binary found without NGINX Plus flag (-nginx-plus)")
	}
	return nginxInfo
}

func getAppProtectVersionInfo() string {
	v, err := os.ReadFile(appProtectVersionPath)
	if err != nil {
		glog.Fatalf("Cannot detect the AppProtect version, %s", err.Error())
	}
	version := strings.TrimSpace(string(v))
	glog.Infof("Using AppProtect Version %s", version)
	return version
}

func getAgentVersionInfo(nginxManager nginx.Manager) string {
	return nginxManager.AgentVersion()
}

type childProcesses struct {
	nginxDone      chan error
	aPPluginEnable bool
	aPPluginDone   chan error
	aPDosEnable    bool
	aPDosDone      chan error
	agentEnable    bool
	agentDone      chan error
}

// newChildProcesses starts the several child processes based on flags set.
// AppProtect. AppProtectDos, Agent.
func startChildProcesses(nginxManager nginx.Manager, appProtectV5 bool) childProcesses {
	var aPPluginDone chan error

	// Do not start AppProtect Plugins when using v5.
	if *appProtect && !appProtectV5 {
		aPPluginDone = make(chan error, 1)
		nginxManager.AppProtectPluginStart(aPPluginDone, *appProtectLogLevel)
	}

	var aPPDosAgentDone chan error

	if *appProtectDos {
		aPPDosAgentDone = make(chan error, 1)
		nginxManager.AppProtectDosAgentStart(aPPDosAgentDone, *appProtectDosDebug, *appProtectDosMaxDaemons, *appProtectDosMaxWorkers, *appProtectDosMemory)
	}

	nginxDone := make(chan error, 1)
	nginxManager.Start(nginxDone)

	var agentDone chan error
	if *agent {
		agentDone = make(chan error, 1)
		nginxManager.AgentStart(agentDone, *agentInstanceGroup)
	}

	return childProcesses{
		nginxDone:      nginxDone,
		aPPluginEnable: *appProtect,
		aPPluginDone:   aPPluginDone,
		aPDosEnable:    *appProtectDos,
		aPDosDone:      aPPDosAgentDone,
		agentEnable:    *agent,
		agentDone:      agentDone,
	}
}

func processDefaultServerSecret(kubeClient *kubernetes.Clientset, nginxManager nginx.Manager) bool {
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
	return sslRejectHandshake
}

func processWildcardSecret(kubeClient *kubernetes.Clientset, nginxManager nginx.Manager) bool {
	if *wildcardTLSSecret != "" {
		secret, err := getAndValidateSecret(kubeClient, *wildcardTLSSecret)
		if err != nil {
			glog.Fatalf("Error trying to get the wildcard TLS secret %v: %v", *wildcardTLSSecret, err)
		}

		bytes := configs.GenerateCertAndKeyFileContent(secret)
		nginxManager.CreateSecret(configs.WildcardSecretName, bytes, nginx.TLSSecretFileMode)
	}
	return *wildcardTLSSecret != ""
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

	if *enableServiceInsight {
		forbiddenListenerPorts[*serviceInsightListenPort] = true
	}

	if *enableTLSPassthrough {
		forbiddenListenerPorts[*tlsPassthroughPort] = true
	}

	return cr_validation.NewGlobalConfigurationValidator(forbiddenListenerPorts)
}

func processNginxConfig(staticCfgParams *configs.StaticConfigParams, cfgParams *configs.ConfigParams, templateExecutor *version1.TemplateExecutor, nginxManager nginx.Manager) {
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
}

// getSocketClient gets a http.Client with a unix socket transport.
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

func handleTermination(lbc *k8s.LoadBalancerController, nginxManager nginx.Manager, listener metrics.SyslogListener, cpcfg childProcesses) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)

	select {
	case err := <-cpcfg.nginxDone:
		if err != nil {
			glog.Fatalf("nginx command exited unexpectedly with status: %v", err)
		} else {
			glog.Info("nginx command exited successfully")
		}
	case err := <-cpcfg.aPPluginDone:
		glog.Fatalf("AppProtectPlugin command exited unexpectedly with status: %v", err)
	case err := <-cpcfg.aPDosDone:
		glog.Fatalf("AppProtectDosAgent command exited unexpectedly with status: %v", err)
	case <-signalChan:
		glog.Infof("Received SIGTERM, shutting down")
		lbc.Stop()
		nginxManager.Quit()
		<-cpcfg.nginxDone
		if cpcfg.aPPluginEnable {
			nginxManager.AppProtectPluginQuit()
			<-cpcfg.aPPluginDone
		}
		if cpcfg.aPDosEnable {
			nginxManager.AppProtectDosAgentQuit()
			<-cpcfg.aPDosDone
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

func createManagerAndControllerCollectors(constLabels map[string]string) (collectors.ManagerCollector, collectors.ControllerCollector, *prometheus.Registry) {
	var err error

	var registry *prometheus.Registry
	var mc collectors.ManagerCollector
	var cc collectors.ControllerCollector
	mc = collectors.NewManagerFakeCollector()
	cc = collectors.NewControllerFakeCollector()

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

func createPlusAndLatencyCollectors(
	registry *prometheus.Registry,
	constLabels map[string]string,
	kubeClient *kubernetes.Clientset,
	plusClient *client.NginxClient,
	isMesh bool,
) (*nginxCollector.NginxPlusCollector, metrics.SyslogListener, collectors.LatencyCollector) {
	var prometheusSecret *api_v1.Secret
	var err error
	var lc collectors.LatencyCollector
	lc = collectors.NewLatencyFakeCollector()
	var syslogListener metrics.SyslogListener
	syslogListener = metrics.NewSyslogFakeServer()

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
				streamUpstreamServerVariableLabels, streamServerZoneVariableLabels, streamUpstreamServerPeerVariableLabelNames, nil, nil)
			infoLevel := new(promlog.AllowedLevel)
			err := infoLevel.Set("info")
			if err != nil {
				glog.Error("Error setting prometheus exporter log level")
			}
			promlogConfig := &promlog.Config{
				Level: infoLevel,
			}
			logger := promlog.New(promlogConfig)
			plusCollector = nginxCollector.NewNginxPlusCollector(plusClient, "nginx_ingress_nginxplus", variableLabelNames, constLabels, logger)
			go metrics.RunPrometheusListenerForNginxPlus(*prometheusMetricsListenPort, plusCollector, registry, prometheusSecret)
		} else {
			httpClient := getSocketClient("/var/lib/nginx/nginx-status.sock")
			client := metrics.NewNginxMetricsClient(httpClient)
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

func createHealthProbeEndpoint(kubeClient *kubernetes.Clientset, plusClient *client.NginxClient, cnf *configs.Configurator) {
	if !*enableServiceInsight {
		return
	}
	var serviceInsightSecret *api_v1.Secret
	var err error

	if *serviceInsightTLSSecretName != "" {
		serviceInsightSecret, err = getAndValidateSecret(kubeClient, *serviceInsightTLSSecretName)
		if err != nil {
			glog.Fatalf("Error trying to get the service insight TLS secret %v: %v", *serviceInsightTLSSecretName, err)
		}
	}
	go healthcheck.RunHealthCheck(*serviceInsightListenPort, plusClient, cnf, serviceInsightSecret)
}

func processGlobalConfiguration() {
	if *globalConfiguration != "" {
		_, _, err := k8s.ParseNamespaceName(*globalConfiguration)
		if err != nil {
			glog.Fatalf("Error parsing the global-configuration argument: %v", err)
		}

		if !*enableCustomResources {
			glog.Fatal("global-configuration flag requires -enable-custom-resources")
		}
	}
}

func processConfigMaps(kubeClient *kubernetes.Clientset, cfgParams *configs.ConfigParams, nginxManager nginx.Manager, templateExecutor *version1.TemplateExecutor) *configs.ConfigParams {
	if *nginxConfigMaps != "" {
		ns, name, err := k8s.ParseNamespaceName(*nginxConfigMaps)
		if err != nil {
			glog.Fatalf("Error parsing the nginx-configmaps argument: %v", err)
		}
		cfm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(context.TODO(), name, meta_v1.GetOptions{})
		if err != nil {
			glog.Fatalf("Error when getting %v: %v", *nginxConfigMaps, err)
		}
		cfgParams = configs.ParseConfigMap(cfm, *nginxPlus, *appProtect, *appProtectDos, *enableTLSPassthrough)
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
	return cfgParams
}

func updateSelfWithVersionInfo(kubeClient *kubernetes.Clientset, version, appProtectVersion, agentVersion string, nginxVersion nginx.Version, maxRetries int, waitTime time.Duration) {
	podUpdated := false

	for i := 0; (i < maxRetries || maxRetries == 0) && !podUpdated; i++ {
		if i > 0 {
			time.Sleep(waitTime)
		}
		pod, err := kubeClient.CoreV1().Pods(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), os.Getenv("POD_NAME"), meta_v1.GetOptions{})
		if err != nil {
			glog.Errorf("Error getting pod on attempt %d of %d: %v", i+1, maxRetries, err)
			continue
		}

		// Copy pod and update the labels.
		newPod := pod.DeepCopy()
		labels := newPod.ObjectMeta.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		labels[nginxVersionLabel] = nginxVersion.Format()
		labels[versionLabel] = strings.TrimPrefix(version, "v")
		if appProtectVersion != "" {
			labels[appProtectVersionLabel] = appProtectVersion
		}
		if agentVersion != "" {
			labels[agentVersionLabel] = agentVersion
		}
		newPod.ObjectMeta.Labels = labels

		_, err = kubeClient.CoreV1().Pods(newPod.ObjectMeta.Namespace).Update(context.TODO(), newPod, meta_v1.UpdateOptions{})
		if err != nil {
			glog.Errorf("Error updating pod with labels on attempt %d of %d: %v", i+1, maxRetries, err)
			continue
		}

		glog.Infof("Pod label updated: %s", pod.ObjectMeta.Name)
		podUpdated = true
	}

	if !podUpdated {
		glog.Errorf("Failed to update pod labels after %d attempts", maxRetries)
	}
}

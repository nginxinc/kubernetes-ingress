package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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

// KubernetesAPI - type to abstract the Kubernetes interface
type KubernetesAPI struct {
	Client kubernetes.Interface
}

// FileHandle - Interface to read a file
type FileHandle interface {
	ReadFile(filename string) ([]byte, error)
}

// OSFileHandle - Struct to hold the interface to the reading a file
type OSFileHandle struct{}

// ReadFile - Actual implementation of the interface abstraction
func (o *OSFileHandle) ReadFile(filename string) ([]byte, error) {
	_, err := os.Open(filepath.Clean(filename))
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filename)
}

// This is the start of the program
func main() {
	commitHash, commitTime, dirtyBuild := getBuildInfo()
	fmt.Printf("NGINX Ingress Controller Version=%v Commit=%v Date=%v DirtyState=%v Arch=%v/%v Go=%v\n", version, commitHash, commitTime, dirtyBuild, runtime.GOOS, runtime.GOARCH, runtime.Version())

	parseFlags()
	parsedFlags := os.Args[1:]

	config, err := getClientConfig()
	if err != nil {
		glog.Fatalf("error creating client configuration: %v", err)
	}

	kubeClient, err := getKubeClient(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v.", err)
	}

	if err := confirmMinimumK8sVersionCriteria(kubeClient); err != nil {
		glog.Fatal(err)
	}

	if err := validateIngressClass(kubeClient); err != nil {
		glog.Fatal(err)
	}

	checkNamespaces(kubeClient)

	dynClient, err := createDynamicClient(config)
	if err != nil {
		glog.Fatal(err)
	}
	confClient, err := createConfigClient(config)
	if err != nil {
		glog.Fatal(err)
	}

	constLabels := map[string]string{"class": *ingressClass}

	managerCollector, controllerCollector, registry := createManagerAndControllerCollectors(constLabels)

	nginxManager, useFakeNginxManager := createNginxManager(managerCollector)

	nginxVersion, err := getNginxVersionInfo(nginxManager)
	if err != nil {
		glog.Fatal(err)
	}

	var appProtectVersion string
	var appProtectV5 bool
	appProtectBundlePath := appProtectv4BundleFolder
	if *appProtect {
		osFileHandle := &OSFileHandle{}
		appProtectVersion, err = getAppProtectVersionInfo(osFileHandle)
		if err != nil {
			glog.Fatal(err)
    }
		r := regexp.MustCompile("^5.*")
		if r.MatchString(appProtectVersion) {
			appProtectV5 = true
			appProtectBundlePath = appProtectv5BundleFolder
		}
	}

	var agentVersion string
	if *agent {
		agentVersion = nginxManager.AgentVersion()
	}

	go updateSelfWithVersionInfo(kubeClient, version, appProtectVersion, agentVersion, nginxVersion, 10, time.Second*5)

	templateExecutorV1, err := createV1TemplateExecutors()
	if err != nil {
		glog.Fatal(err)
	}

	templateExecutorV2, err := createV2TemplateExecutors()
	if err != nil {
		glog.Fatal(err)
	}

	kAPI := &KubernetesAPI{
		Client: kubeClient,
	}
	sslRejectHandshake, err := kAPI.processDefaultServerSecret(nginxManager)
	if err != nil {
		glog.Fatal(err)
	}

	isWildcardEnabled, err := kAPI.processWildcardSecret(nginxManager)
	if err != nil {
		glog.Fatal(err)
	}

	globalConfigurationValidator := createGlobalConfigurationValidator()

	if err := processGlobalConfiguration(); err != nil {
		glog.Fatal(err)
	}

	cfgParams := configs.NewDefaultConfigParams(*nginxPlus)
	cfgParams, err = kAPI.processConfigMaps(cfgParams, nginxManager, templateExecutorV1)
	if err != nil {
		glog.Fatal(err)
	}

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

	if err := processNginxConfig(staticCfgParams, cfgParams, templateExecutorV1, nginxManager); err != nil {
		glog.Fatal(err)
	}

	if *enableTLSPassthrough {
		var emptyFile []byte
		nginxManager.CreateTLSPassthroughHostsConfig(emptyFile)
	}

	process := startChildProcesses(nginxManager, appProtectV5)

	plusClient, err := createPlusClient(*nginxPlus, useFakeNginxManager, nginxManager)
	if err != nil {
		glog.Fatal(err)
	}

	plusCollector, syslogListener, latencyCollector := createPlusAndLatencyCollectors(registry, constLabels, kubeClient, plusClient, staticCfgParams.NginxServiceMesh)
	cnf := configs.NewConfigurator(configs.ConfiguratorParams{
		NginxManager:                        nginxManager,
		StaticCfgParams:                     staticCfgParams,
		Config:                              cfgParams,
		TemplateExecutor:                    templateExecutorV1,
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

	if err := createHealthProbeEndpoint(kubeClient, plusClient, cnf); err != nil {
		glog.Fatal(err)
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

// This function returns a k8s client object configuration
func getClientConfig() (config *rest.Config, err error) {
	if *proxyURL != "" {
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{},
			&clientcmd.ConfigOverrides{
				ClusterInfo: clientcmdapi.Cluster{
					Server: *proxyURL,
				},
			}).ClientConfig()
	} else {
		config, err = rest.InClusterConfig()
	}

	return config, err
}

// This returns a k8s client with the provided client config for interacting with the k8s API
func getKubeClient(config *rest.Config) (kubeClient *kubernetes.Clientset, err error) {
	kubeClient, err = kubernetes.NewForConfig(config)
	return kubeClient, err
}

// This function checks that NIC is running on at least a prescribed minimum k8s version or higher for supportability
// Anything lower throws than the prescribed version, an error is returned to the caller
func confirmMinimumK8sVersionCriteria(kubeClient kubernetes.Interface) (err error) {
	k8sVersion, err := k8s.GetK8sVersion(kubeClient)
	if err != nil {
		return fmt.Errorf("error retrieving k8s version: %w", err)
	}
	glog.Infof("Kubernetes version: %v", k8sVersion)

	minK8sVersion, err := util_version.ParseGeneric("1.22.0")
	if err != nil {
		return fmt.Errorf("unexpected error parsing minimum supported version: %w", err)
	}

	if !k8sVersion.AtLeast(minK8sVersion) {
		return fmt.Errorf("versions of kubernetes < %v are not supported, please refer to the documentation for details on supported versions and legacy controller support", minK8sVersion)
	}
	return err
}

// An Ingress resource can target a specific Ingress controller instance.
// This is useful when running multiple ingress controllers in the same cluster.
// Targeting an Ingress controller means only a specific controller should handle/implement the ingress resource.
// This can be done using either the IngressClassName field or the ingress.class annotation
// This function confirms that the Ingress resource is meant to be handled by NGINX Ingress Controller.
// Otherwise an error is returned to the caller
// This is defined in the const k8s.IngressControllerName
func validateIngressClass(kubeClient kubernetes.Interface) (err error) {
	ingressClassRes, err := kubeClient.NetworkingV1().IngressClasses().Get(context.TODO(), *ingressClass, meta_v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error when getting IngressClass %v: %w", *ingressClass, err)
	}

	if ingressClassRes.Spec.Controller != k8s.IngressControllerName {
		return fmt.Errorf("ingressClass with name %v has an invalid Spec.Controller %v; expected %v", ingressClassRes.Name, ingressClassRes.Spec.Controller, k8s.IngressControllerName)
	}

	return err
}

// The objective of this function is to confirm the presence of the list of namepsaces in the k8s cluster.
// The list is provided via -watch-namespace and -watch-namespace-label cmdline options
// The function may log an error if it failed to get the namespace(s) via the label selector
// The secrets namespace is watched in the same vein specified via -watch-secret-namespace cmdline option
func checkNamespaces(kubeClient kubernetes.Interface) {
	if *watchNamespaceLabel != "" {
		watchNamespaces = getWatchedNamespaces(kubeClient)
	} else {
		_ = checkNamespaceExists(kubeClient, watchNamespaces)
	}
	_ = checkNamespaceExists(kubeClient, watchSecretNamespaces)
}

// This is a helper function for fetching the all the namespaces in the cluster
func getWatchedNamespaces(kubeClient kubernetes.Interface) (newWatchNamespaces []string) {
	// bootstrap the watched namespace list
	nsList, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), meta_v1.ListOptions{LabelSelector: *watchNamespaceLabel})
	if err != nil {
		glog.Errorf("error when getting Namespaces with the label selector %v: %v", watchNamespaceLabel, err)
	}
	for _, ns := range nsList.Items {
		newWatchNamespaces = append(newWatchNamespaces, ns.Name)
	}
	glog.Infof("Namespaces watched using label %v: %v", *watchNamespaceLabel, watchNamespaces)

	return newWatchNamespaces
}

// This is a helper function for confirming the presence of input  namespaces
func checkNamespaceExists(kubeClient kubernetes.Interface, namespaces []string) bool {
	hasErrors := false
	for _, ns := range namespaces {
		if ns != "" {
			_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), ns, meta_v1.GetOptions{})
			if err != nil {
				glog.Warningf("Error when getting Namespace %v: %v", ns, err)
			}
			hasErrors = hasErrors || err != nil
		}
	}
	return hasErrors
}

func createConfigClient(config *rest.Config) (configClient k8s_nginx.Interface, err error) {
	if *enableCustomResources {
		configClient, err = k8s_nginx.NewForConfig(config)
		if err != nil {
			return configClient, fmt.Errorf("failed to create a conf client: %w", err)
		}

		// required for emitting Events for VirtualServer
		err = conf_scheme.AddToScheme(scheme.Scheme)
		if err != nil {
			return configClient, fmt.Errorf("failed to add configuration types to the scheme: %w", err)
		}
	}
	return configClient, err
}

// Creates a new dynamic client or returns an error
func createDynamicClient(config *rest.Config) (dynClient dynamic.Interface, err error) {
	if *appProtectDos || *appProtect || *ingressLink != "" {
		dynClient, err = dynamic.NewForConfig(config)
		if err != nil {
			return dynClient, fmt.Errorf("failed to create dynamic client: %w", err)
		}
	}
	return dynClient, err
}

// Returns a NGINX plus client config to talk to the N+ API
func createPlusClient(nginxPlus bool, useFakeNginxManager bool, nginxManager nginx.Manager) (plusClient *client.NginxClient, err error) {
	if nginxPlus && !useFakeNginxManager {
		httpClient := getSocketClient("/var/lib/nginx/nginx-plus-api.sock")
		plusClient, err = client.NewNginxClient("http://nginx-plus-api/api", client.WithHTTPClient(httpClient))
		if err != nil {
			return plusClient, fmt.Errorf("failed to create NginxClient for Plus: %w", err)
		}
		nginxManager.SetPlusClients(plusClient, httpClient)
	}
	return plusClient, nil
}

// Returns a version 1 of the template
func createV1TemplateExecutors() (templateExecutor *version1.TemplateExecutor, err error) {
	nginxConfTemplatePath := "nginx.tmpl"
	nginxIngressTemplatePath := "nginx.ingress.tmpl"

	if *nginxPlus {
		nginxConfTemplatePath = "nginx-plus.tmpl"
		nginxIngressTemplatePath = "nginx-plus.ingress.tmpl"
	}

	if *mainTemplatePath != "" {
		nginxConfTemplatePath = *mainTemplatePath
	}
	if *ingressTemplatePath != "" {
		nginxIngressTemplatePath = *ingressTemplatePath
	}

	templateExecutor, err = version1.NewTemplateExecutor(nginxConfTemplatePath, nginxIngressTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("error creating TemplateExecutor: %w", err)
	}

	return templateExecutor, nil
}

// Returns a version 2 of the template
func createV2TemplateExecutors() (templateExecutorV2 *version2.TemplateExecutor, err error) {
	nginxVirtualServerTemplatePath := "nginx.virtualserver.tmpl"
	nginxTransportServerTemplatePath := "nginx.transportserver.tmpl"
	if *nginxPlus {
		nginxVirtualServerTemplatePath = "nginx-plus.virtualserver.tmpl"
		nginxTransportServerTemplatePath = "nginx-plus.transportserver.tmpl"
	}

	if *virtualServerTemplatePath != "" {
		nginxVirtualServerTemplatePath = *virtualServerTemplatePath
	}
	if *transportServerTemplatePath != "" {
		nginxTransportServerTemplatePath = *transportServerTemplatePath
	}

	templateExecutorV2, err = version2.NewTemplateExecutor(nginxVirtualServerTemplatePath, nginxTransportServerTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("error creating TemplateExecutorV2: %w", err)
	}

	return templateExecutorV2, nil
}

// Returns a handle to a manager interface for managing the configuration of NGINX
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

// Returns the NGINX version depending on OSS or Plus versions
func getNginxVersionInfo(nginxManager nginx.Manager) (nginxInfo nginx.Version, err error) {
	nginxInfo = nginxManager.Version()
	glog.Infof("Using %s", nginxInfo.String())

	if *nginxPlus && !nginxInfo.IsPlus {
		return nginxInfo, fmt.Errorf("the NGINX Plus flag is enabled (-nginx-plus) without NGINX Plus binary")
	} else if !*nginxPlus && nginxInfo.IsPlus {
		return nginxInfo, fmt.Errorf("found NGINX Plus binary but without NGINX Plus flag (-nginx-plus)")
	}
	return nginxInfo, err
}

// Returns the version of App-Protect running on the system
func getAppProtectVersionInfo(fd FileHandle) (version string, err error) {
	v, err := fd.ReadFile(appProtectVersionPath)
	if err != nil {
		return version, fmt.Errorf("cannot detect the AppProtect version, %s", err.Error())
	}
	version = strings.TrimSpace(string(v))
	glog.Infof("Using AppProtect Version %s", version)
	return version, err
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

// Applies the server secret config as provided via the cmdline option -default-server-tls-secret or the default
// Returns a boolean for rejecting the SSL handshake
func (kAPI KubernetesAPI) processDefaultServerSecret(nginxManager nginx.Manager) (sslRejectHandshake bool, err error) {
	sslRejectHandshake = false
	if *defaultServerSecret != "" {
		secret, err := kAPI.getAndValidateSecret(*defaultServerSecret)
		if err != nil {
			return sslRejectHandshake, fmt.Errorf("error trying to get the default server TLS secret %v: %w", *defaultServerSecret, err)
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
				return sslRejectHandshake, fmt.Errorf("error checking the default server TLS cert and key in %s: %w", configs.DefaultServerSecretPath, err)
			}
		}
	}
	return sslRejectHandshake, nil
}

// Applies the wildcard server secret config as provided via the cmdline option -wildcard-tls-secret or the default
// Returns a boolean for rejecting the SSL handshake
func (kAPI KubernetesAPI) processWildcardSecret(nginxManager nginx.Manager) (isWildcardTLSSecret bool, err error) {
	isWildcardTLSSecret = false
	if *wildcardTLSSecret != "" {
		secret, err := kAPI.getAndValidateSecret(*wildcardTLSSecret)
		if err != nil {
			return isWildcardTLSSecret, fmt.Errorf("error trying to get the wildcard TLS secret %v: %w", *wildcardTLSSecret, err)
		}

		bytes := configs.GenerateCertAndKeyFileContent(secret)
		nginxManager.CreateSecret(configs.WildcardSecretName, bytes, nginx.TLSSecretFileMode)
	}
	isWildcardTLSSecret = *wildcardTLSSecret != ""
	return isWildcardTLSSecret, nil
}

// This returns a list of ports and corresponding boolean on the status of the service is enabled
// This depends on
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

// Generates the main NGINX config and the open tracing config
func processNginxConfig(staticCfgParams *configs.StaticConfigParams, cfgParams *configs.ConfigParams, templateExecutor *version1.TemplateExecutor, nginxManager nginx.Manager) (err error) {
	ngxConfig := configs.GenerateNginxMainConfig(staticCfgParams, cfgParams)
	content, err := templateExecutor.ExecuteMainConfigTemplate(ngxConfig)
	if err != nil {
		return fmt.Errorf("error generating NGINX main config: %w", err)
	}
	nginxManager.CreateMainConfig(content)

	nginxManager.UpdateConfigVersionFile(ngxConfig.OpenTracingLoadModule)

	nginxManager.SetOpenTracing(ngxConfig.OpenTracingLoadModule)

	if ngxConfig.OpenTracingLoadModule {
		err := nginxManager.CreateOpenTracingTracerConfig(cfgParams.MainOpenTracingTracerConfig)
		if err != nil {
			return fmt.Errorf("error creating OpenTracing tracer config file: %w", err)
		}
	}
	return err
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
func (kAPI KubernetesAPI) getAndValidateSecret(secretNsName string) (secret *api_v1.Secret, err error) {
	ns, name, err := k8s.ParseNamespaceName(secretNsName)
	if err != nil {
		return nil, fmt.Errorf("could not parse the %v argument: %w", secretNsName, err)
	}
	secret, err = kAPI.Client.CoreV1().Secrets(ns).Get(context.TODO(), name, meta_v1.GetOptions{})
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

// Handler/callback function when KIC's NGINX software declares itself ready via the -ready-status cmdline option
// This function is called by http.NewServeMux which probes the /nginx-ready endpoint via an HTTP request.
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

// This procedure creates various managers for KIC operation
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

// Creates an NGINX Plus and Latency Collector
func createPlusAndLatencyCollectors(
	registry *prometheus.Registry,
	constLabels map[string]string,
	kubeClient *kubernetes.Clientset,
	plusClient *client.NginxClient,
	isMesh bool,
) (plusCollector *nginxCollector.NginxPlusCollector, syslogListener metrics.SyslogListener, latencyCollector collectors.LatencyCollector) {
	if *enablePrometheusMetrics {
		upstreamServerVariableLabels := []string{"service", "resource_type", "resource_name", "resource_namespace"}
		upstreamServerPeerVariableLabelNames := []string{"pod_name"}
		if isMesh {
			upstreamServerPeerVariableLabelNames = append(upstreamServerPeerVariableLabelNames, "pod_owner")
		}

		plusCollector = createNginxPlusCollector(registry, constLabels, kubeClient, plusClient, upstreamServerVariableLabels, upstreamServerPeerVariableLabelNames)
		syslogListener, latencyCollector = createLatencyCollector(registry, constLabels, upstreamServerVariableLabels, upstreamServerPeerVariableLabelNames)
	}

	return plusCollector, syslogListener, latencyCollector
}

// Helper function to creates an NGINX Plus Collector
func createNginxPlusCollector(registry *prometheus.Registry, constLabels map[string]string, kubeClient *kubernetes.Clientset, plusClient *client.NginxClient, upstreamServerVariableLabels []string, upstreamServerPeerVariableLabelNames []string) *nginxCollector.NginxPlusCollector {
	var plusCollector *nginxCollector.NginxPlusCollector
	var prometheusSecret *api_v1.Secret
	var err error

	if *prometheusTLSSecretName != "" {
		kAPI := &KubernetesAPI{
			Client: kubeClient,
		}
		prometheusSecret, err = kAPI.getAndValidateSecret(*prometheusTLSSecretName)
		if err != nil {
			glog.Fatalf("Error trying to get the prometheus TLS secret %v: %v", *prometheusTLSSecretName, err)
		}
	}

	if *nginxPlus {
		streamUpstreamServerVariableLabels := []string{"service", "resource_type", "resource_name", "resource_namespace"}
		streamUpstreamServerPeerVariableLabelNames := []string{"pod_name"}

		serverZoneVariableLabels := []string{"resource_type", "resource_name", "resource_namespace"}
		streamServerZoneVariableLabels := []string{"resource_type", "resource_name", "resource_namespace"}
		variableLabelNames := nginxCollector.NewVariableLabelNames(upstreamServerVariableLabels, serverZoneVariableLabels, upstreamServerPeerVariableLabelNames,
			streamUpstreamServerVariableLabels, streamServerZoneVariableLabels, streamUpstreamServerPeerVariableLabelNames, nil, nil)

		promlogConfig := &promlog.Config{}
		logger := promlog.New(promlogConfig)
		plusCollector = nginxCollector.NewNginxPlusCollector(plusClient, "nginx_ingress_nginxplus", variableLabelNames, constLabels, logger)
		go metrics.RunPrometheusListenerForNginxPlus(*prometheusMetricsListenPort, plusCollector, registry, prometheusSecret)
	} else {
		httpClient := getSocketClient("/var/lib/nginx/nginx-status.sock")
		client := metrics.NewNginxMetricsClient(httpClient)
		go metrics.RunPrometheusListenerForNginx(*prometheusMetricsListenPort, client, registry, constLabels, prometheusSecret)
	}
	return plusCollector
}

// Helper that returns a latency metrics collect via syslog
func createLatencyCollector(registry *prometheus.Registry, constLabels map[string]string, upstreamServerVariableLabels []string, upstreamServerPeerVariableLabelNames []string) (metrics.SyslogListener, collectors.LatencyCollector) {
	var lc collectors.LatencyCollector
	lc = collectors.NewLatencyFakeCollector()
	var syslogListener metrics.SyslogListener
	syslogListener = metrics.NewSyslogFakeServer()

	if *enablePrometheusMetrics {
		if *enableLatencyMetrics {
			lc = collectors.NewLatencyMetricsCollector(constLabels, upstreamServerVariableLabels, upstreamServerPeerVariableLabelNames)
			if err := lc.Register(registry); err != nil {
				glog.Errorf("Error registering Latency Prometheus metrics: %v", err)
			}
			syslogListener = metrics.NewLatencyMetricsListener("/var/lib/nginx/nginx-syslog.sock", lc)
			go syslogListener.Run()
		}
	}

	return syslogListener, lc
}

// This function starts a go routine (a lightweight thread of execution) for health checks against service insight listener ports
// An error is returned if there is a problem with the TLS secret for the Insight service
func createHealthProbeEndpoint(kubeClient *kubernetes.Clientset, plusClient *client.NginxClient, cnf *configs.Configurator) (err error) {
	if !*enableServiceInsight {
		return nil
	}
	var serviceInsightSecret *api_v1.Secret

	if *serviceInsightTLSSecretName != "" {
		kAPI := &KubernetesAPI{
			Client: kubeClient,
		}
		serviceInsightSecret, err = kAPI.getAndValidateSecret(*serviceInsightTLSSecretName)
		if err != nil {
			return fmt.Errorf("error trying to get the service insight TLS secret %v: %w", *serviceInsightTLSSecretName, err)
		}
	}
	go healthcheck.RunHealthCheck(*serviceInsightListenPort, plusClient, cnf, serviceInsightSecret)
	return nil
}

// Parses the cmdline option -global-configuration (requires -enable-custom-resources)
// Returns an error either if there is a problem with -global-configuration or -enable-custom-resources is not set
func processGlobalConfiguration() (err error) {
	if *globalConfiguration != "" {
		_, _, err := k8s.ParseNamespaceName(*globalConfiguration)
		if err != nil {
			return fmt.Errorf("error parsing the global-configuration argument: %w", err)
		}

		if !*enableCustomResources {
			return fmt.Errorf("global-configuration flag requires -enable-custom-resources")
		}
	}
	return nil
}

// Parses a ConfigMap resource for customizing NGINX configuration provided via the cmdline option -nginx-configmaps
// Returns an error if unable to parse the ConfigMap
func (kAPI KubernetesAPI) processConfigMaps(cfgParams *configs.ConfigParams, nginxManager nginx.Manager, templateExecutor *version1.TemplateExecutor) (*configs.ConfigParams, error) {
	if *nginxConfigMaps != "" {
		ns, name, err := k8s.ParseNamespaceName(*nginxConfigMaps)
		if err != nil {
			return nil, fmt.Errorf("error parsing the nginx-configmaps argument: %w", err)
		}
		cfm, err := kAPI.Client.CoreV1().ConfigMaps(ns).Get(context.TODO(), name, meta_v1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("error when getting %v: %w", *nginxConfigMaps, err)
		}
		cfgParams = configs.ParseConfigMap(cfm, *nginxPlus, *appProtect, *appProtectDos, *enableTLSPassthrough)
		if cfgParams.MainServerSSLDHParamFileContent != nil {
			fileName, err := nginxManager.CreateDHParam(*cfgParams.MainServerSSLDHParamFileContent)
			if err != nil {
				return nil, fmt.Errorf("configmap %s/%s: Could not update dhparams: %w", ns, name, err)
			} else {
				cfgParams.MainServerSSLDHParam = fileName
			}
		}
		if cfgParams.MainTemplate != nil {
			err = templateExecutor.UpdateMainTemplate(cfgParams.MainTemplate)
			if err != nil {
				return nil, fmt.Errorf("error updating NGINX main template: %w", err)
			}
		}
		if cfgParams.IngressTemplate != nil {
			err = templateExecutor.UpdateIngressTemplate(cfgParams.IngressTemplate)
			if err != nil {
				return nil, fmt.Errorf("error updating ingress template: %w", err)
			}
		}
	}
	return cfgParams, nil
}

// This function updates the labels of the NGINX Ingress Controller
// An error is returned if its unable to retrieve pod info and other problems along the way
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

package main

import (
	"context"
	"flag"
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
	"github.com/nginxinc/kubernetes-ingress/internal/k8s"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var (
	// Set during build
	version   string
	gitCommit string

	healthStatus = flag.Bool("health-status", false,
		`Add a location "/nginx-health" to the default server. The location responds with the 200 status code for any request.
	Useful for external health-checking of the Ingress controller`)

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

	ingressClass = flag.String("ingress-class", "nginx",
		`A class of the Ingress controller. The Ingress controller only processes Ingress resources that belong to its class
	- i.e. have the annotation "kubernetes.io/ingress.class" equal to the class. Additionally,
	the Ingress controller processes Ingress resources that do not have that annotation,
	which can be disabled by setting the "-use-ingress-class-only" flag`)

	useIngressClassOnly = flag.Bool("use-ingress-class-only", false,
		`Ignore Ingress resources without the "kubernetes.io/ingress.class" annotation`)

	defaultServerSecret = flag.String("default-server-tls-secret", "",
		`A Secret with a TLS certificate and key for TLS termination of the default server. Format: <namespace>/<name>.
	If not set, certificate and key in the file "/etc/nginx/secrets/default" are used. If a secret is set,
	but the Ingress controller is not able to fetch it from Kubernetes API or a secret is not set and
	the file "/etc/nginx/secrets/default" does not exist, the Ingress controller will fail to start`)

	versionFlag = flag.Bool("version", false, "Print the version and git-commit hash and exit")

	mainTemplatePath = flag.String("main-template-path", "",
		`Path to the main NGINX configuration template. (default for NGINX "nginx.tmpl"; default for NGINX Plus "nginx-plus.tmpl")`)

	ingressTemplatePath = flag.String("ingress-template-path", "",
		`Path to the ingress NGINX configuration template for an ingress resource.
	(default for NGINX "nginx.ingress.tmpl"; default for NGINX Plus "nginx-plus.ingress.tmpl")`)

	externalService = flag.String("external-service", "",
		`Specifies the name of the service with the type LoadBalancer through which the Ingress controller pods are exposed externally.
The external address of the service is used when reporting the status of Ingress resources. Requires -report-ingress-status.`)

	reportIngressStatus = flag.Bool("report-ingress-status", false,
		"Update the address field in the status of Ingresses resources. Requires the -external-service flag, or the 'external-status-address' key in the ConfigMap.")

	leaderElectionEnabled = flag.Bool("enable-leader-election", false,
		"Enable Leader election to avoid multiple replicas of the controller reporting the status of Ingress resources -- only one replica will report status. See -report-ingress-status flag.")

	nginxStatusAllowCIDRs = flag.String("nginx-status-allow-cidrs", "127.0.0.1", `Whitelist IPv4 IP/CIDR blocks to allow access to NGINX stub_status or the NGINX Plus API. Separate multiple IP/CIDR by commas.`)

	nginxStatusPort = flag.Int("nginx-status-port", 8080,
		"Set the port where the NGINX stub_status or the NGINX Plus API is exposed. [1023 - 65535]")

	nginxStatus = flag.Bool("nginx-status", true,
		"Enable the NGINX stub_status, or the NGINX Plus API.")

	nginxDebug = flag.Bool("nginx-debug", false,
		"Enable debugging for NGINX. Uses the nginx-debug binary. Requires 'error-log-level: debug' in the ConfigMap.")

	wildcardTLSSecret = flag.String("wildcard-tls-secret", "",
		`A Secret with a TLS certificate and key for TLS termination of every Ingress host for which TLS termination is enabled but the Secret is not specified. 
		Format: <namespace>/<name>. If the argument is not set, for such Ingress hosts NGINX will break any attempt to establish a TLS connection. 
		If the argument is set, but the Ingress controller is not able to fetch the Secret from Kubernetes API, the Ingress controller will fail to start.`)

	enablePrometheusMetrics = flag.Bool("enable-prometheus-metrics", false,
		"Enable exposing NGINX or NGINX Plus metrics in the Prometheus format")

	prometheusMetricsListenPort = flag.Int("prometheus-metrics-listen-port", 9113,
		"Set the port where the Prometheus metrics are exposed. [1023 - 65535]")
)

func main() {
	flag.Parse()

	err := flag.Lookup("logtostderr").Value.Set("true")
	if err != nil {
		glog.Fatalf("Error setting logtostderr to true: %v", err)
	}

	if *versionFlag {
		fmt.Printf("Version=%v GitCommit=%v\n", version, gitCommit)
		os.Exit(0)
	}

	statusPortValidationError := validatePort(*nginxStatusPort)
	if statusPortValidationError != nil {
		glog.Fatalf("Invalid value for nginx-status-port: %v", statusPortValidationError)
	}

	metricsPortValidationError := validatePort(*prometheusMetricsListenPort)
	if metricsPortValidationError != nil {
		glog.Fatalf("Invalid value for prometheus-metrics-listen-port: %v", metricsPortValidationError)
	}

	allowedCIDRs, err := parseNginxStatusAllowCIDRs(*nginxStatusAllowCIDRs)
	if err != nil {
		glog.Fatalf(`Invalid value for nginx-status-allow-cidrs: %v`, err)
	}

	glog.Infof("Starting NGINX Ingress controller Version=%v GitCommit=%v\n", version, gitCommit)

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

	local := *proxyURL != ""

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

	nginxBinaryPath := "/usr/sbin/nginx"
	if *nginxDebug {
		nginxBinaryPath = "/usr/sbin/nginx-debug"
	}

	templateExecutor, err := configs.NewTemplateExecutor(nginxConfTemplatePath, nginxIngressTemplatePath, *healthStatus, *nginxStatus, allowedCIDRs, *nginxStatusPort, *enablePrometheusMetrics)
	if err != nil {
		glog.Fatalf("Error creating TemplateExecutor: %v", err)
	}
	ngxc := nginx.NewNginxController("/etc/nginx/", nginxBinaryPath, local)

	if *defaultServerSecret != "" {
		secret, err := getAndValidateSecret(kubeClient, *defaultServerSecret)
		if err != nil {
			glog.Fatalf("Error trying to get the default server TLS secret %v: %v", *defaultServerSecret, err)
		}

		bytes := configs.GenerateCertAndKeyFileContent(secret)
		ngxc.AddOrUpdateSecretFile(configs.DefaultServerSecretName, bytes, nginx.TLSSecretFileMode)
	} else {
		_, err = os.Stat("/etc/nginx/secrets/default")
		if os.IsNotExist(err) {
			glog.Fatalf("A TLS cert and key for the default server is not found")
		}
	}

	if *wildcardTLSSecret != "" {
		secret, err := getAndValidateSecret(kubeClient, *wildcardTLSSecret)
		if err != nil {
			glog.Fatalf("Error trying to get the wildcard TLS secret %v: %v", *wildcardTLSSecret, err)
		}

		bytes := configs.GenerateCertAndKeyFileContent(secret)
		ngxc.AddOrUpdateSecretFile(configs.WildcardSecretName, bytes, nginx.TLSSecretFileMode)
	}

	cfg := configs.NewDefaultConfig()
	if *nginxConfigMaps != "" {
		ns, name, err := k8s.ParseNamespaceName(*nginxConfigMaps)
		if err != nil {
			glog.Fatalf("Error parsing the nginx-configmaps argument: %v", err)
		}
		cfm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(name, meta_v1.GetOptions{})
		if err != nil {
			glog.Fatalf("Error when getting %v: %v", *nginxConfigMaps, err)
		}
		cfg = configs.ParseConfigMap(cfm, *nginxPlus)
		if cfg.MainServerSSLDHParamFileContent != nil {
			fileName, err := ngxc.AddOrUpdateDHParam(*cfg.MainServerSSLDHParamFileContent)
			if err != nil {
				glog.Fatalf("Configmap %s/%s: Could not update dhparams: %v", ns, name, err)
			} else {
				cfg.MainServerSSLDHParam = fileName
			}
		}
		if cfg.MainTemplate != nil {
			err = templateExecutor.UpdateMainTemplate(cfg.MainTemplate)
			if err != nil {
				glog.Fatalf("Error updating NGINX main template: %v", err)
			}
		}
		if cfg.IngressTemplate != nil {
			err = templateExecutor.UpdateIngressTemplate(cfg.IngressTemplate)
			if err != nil {
				glog.Fatalf("Error updating ingress template: %v", err)
			}
		}
	}

	ngxConfig := configs.GenerateNginxMainConfig(cfg)
	content, err := templateExecutor.ExecuteMainConfigTemplate(ngxConfig)
	if err != nil {
		glog.Fatalf("Error generating NGINX main config: %v", err)
	}
	ngxc.UpdateMainConfigFile(content)
	ngxc.UpdateConfigVersionFile()

	nginxDone := make(chan error, 1)
	ngxc.Start(nginxDone)

	var nginxAPI *nginx.NginxAPIController
	if *nginxPlus {
		httpClient := getSocketClient("/var/run/nginx-plus-api.sock")
		nginxAPI, err = nginx.NewNginxAPIController(&httpClient, "http://nginx-plus-api/api", local)
		if err != nil {
			glog.Fatalf("Failed to create NginxAPIController: %v", err)
		}
	}
	isWildcardEnabled := *wildcardTLSSecret != ""
	cnf := configs.NewConfigurator(ngxc, cfg, nginxAPI, templateExecutor, isWildcardEnabled)
	controllerNamespace := os.Getenv("POD_NAMESPACE")

	lbcInput := k8s.NewLoadBalancerControllerInput{
		KubeClient:              kubeClient,
		ResyncPeriod:            30 * time.Second,
		Namespace:               *watchNamespace,
		NginxConfigurator:       cnf,
		DefaultServerSecret:     *defaultServerSecret,
		IsNginxPlus:             *nginxPlus,
		IngressClass:            *ingressClass,
		UseIngressClassOnly:     *useIngressClassOnly,
		ExternalServiceName:     *externalService,
		ControllerNamespace:     controllerNamespace,
		ReportIngressStatus:     *reportIngressStatus,
		IsLeaderElectionEnabled: *leaderElectionEnabled,
		WildcardTLSSecret:       *wildcardTLSSecret,
		ConfigMaps:              *nginxConfigMaps,
	}

	lbc := k8s.NewLoadBalancerController(lbcInput)

	if *enablePrometheusMetrics {
		if *nginxPlus {
			go metrics.RunPrometheusListenerForNginxPlus(*prometheusMetricsListenPort, nginxAPI.GetClientPlus())
		} else {
			httpClient := getSocketClient("/var/run/nginx-status.sock")
			client, err := metrics.NewNginxMetricsClient(&httpClient)
			if err != nil {
				glog.Fatalf("Error creating the Nginx client for Prometheus metrics: %v", err)
			}
			go metrics.RunPrometheusListenerForNginx(*prometheusMetricsListenPort, client)
		}
	}

	go handleTermination(lbc, ngxc, nginxDone)
	lbc.Run()

	for {
		glog.Info("Waiting for the controller to exit...")
		time.Sleep(30 * time.Second)
	}
}

func handleTermination(lbc *k8s.LoadBalancerController, ngxc *nginx.Controller, nginxDone chan error) {
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
		glog.Infof("Received SIGTERM, shutting down")
	}

	glog.Infof("Shutting down the controller")
	lbc.Stop()

	if !exited {
		glog.Infof("Shutting down NGINX")
		ngxc.Quit()
		<-nginxDone
	}

	glog.Infof("Exiting with a status: %v", exitStatus)
	os.Exit(exitStatus)
}

// getSocketClient gets an http.Client with the a unix socket transport.
func getSocketClient(sockPath string) http.Client {
	return http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", sockPath)
			},
		},
	}
}

// validatePort makes sure a given port is inside the valid port range for its usage
func validatePort(port int) error {
	if port < 1023 || port > 65535 {
		return fmt.Errorf("port outside of valid port range [1023 - 65535]: %v", port)
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
		return nil, fmt.Errorf("could not parse the %v argument: %v", secretNsName, err)
	}
	secret, err = kubeClient.CoreV1().Secrets(ns).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get %v: %v", secretNsName, err)
	}
	err = configs.ValidateTLSSecret(secret)
	if err != nil {
		return nil, fmt.Errorf("%v is invalid: %v", secretNsName, err)
	}
	return secret, nil
}

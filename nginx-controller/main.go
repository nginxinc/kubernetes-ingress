package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"

	"github.com/nginxinc/kubernetes-ingress/nginx-controller/controller"
	"github.com/nginxinc/kubernetes-ingress/nginx-controller/nginx"
	"github.com/nginxinc/kubernetes-ingress/nginx-controller/nginx/plus"
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
		`If present, the default server listening on port 80 with the health check
		location "/nginx-health" gets added to the main nginx configuration.`)

	proxyURL = flag.String("proxy", "",
		`If specified, the controller assumes a kubctl proxy server is running on the
		given url and creates a proxy client. Regenerated NGINX configuration files
    	are not written to the disk, instead they are printed to stdout. Also NGINX
    	is not getting invoked. This flag is for testing.`)

	watchNamespace = flag.String("watch-namespace", api_v1.NamespaceAll,
		`Namespace to watch for Ingress/Services/Endpoints. By default the controller
		watches acrosss all namespaces`)

	nginxConfigMaps = flag.String("nginx-configmaps", "",
		`Specifies a configmaps resource that can be used to customize NGINX
		configuration. The value must follow the following format: <namespace>/<name>`)

	nginxPlus = flag.Bool("nginx-plus", false,
		`Enables support for NGINX Plus.`)

	ingressClass = flag.String("ingress-class", "nginx",
		`Specifies a class of ingress. Only processes Ingresses with this value in annotations. Can be used with --use-ingress-class-only. Default 'nginx'`)

	useIngressClassOnly = flag.Bool("use-ingress-class-only", false,
		`If true, ingress resource will handled by ingress controller with class which specifed by value of ingress-class. Default false`)

	defaultServerSecret = flag.String("default-server-tls-secret", "",
		`Specifies a secret with a TLS certificate and key for SSL termination of
		the default server. The value must follow the following format: <namespace>/<name>.
		If not specified, the key and the cert from /etc/nginx/secrets/default is used.`)
)

func main() {
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")

	glog.Infof("Starting NGINX Ingress controller Version=%v GitCommit=%v\n", version, gitCommit)

	var err error
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
	ngxc, _ := nginx.NewNginxController("/etc/nginx/", local, *healthStatus, nginxConfTemplatePath, nginxIngressTemplatePath)

	if *defaultServerSecret != "" {
		ns, name, err := controller.ParseNamespaceName(*defaultServerSecret)
		if err != nil {
			glog.Fatalf("Error parsing the default-server-tls-secret argument: %v", err)
		}
		secret, err := kubeClient.CoreV1().Secrets(ns).Get(name, meta_v1.GetOptions{})
		if err != nil {
			glog.Fatalf("Error when getting %v: %v", *defaultServerSecret, err)
		}
		err = nginx.ValidateTLSSecret(secret)
		if err != nil {
			glog.Fatalf("%v is invalid: %v", *defaultServerSecret, err)
		}

		bytes := nginx.GenerateCertAndKeyFileContent(secret)
		ngxc.AddOrUpdateSecretFile(nginx.DefaultServerSecretName, bytes, nginx.TLSSecretFileMode)
	} else {
		_, err = os.Stat("/etc/nginx/secrets/default")
		if os.IsNotExist(err) {
			glog.Fatalf("A TLS cert and key for the default server is not found")
		}
	}

	nginxDone := make(chan error, 1)
	ngxc.Start(nginxDone)

	nginxConfig := nginx.NewDefaultConfig()
	var nginxAPI *plus.NginxAPIController
	if *nginxPlus {
		time.Sleep(500 * time.Millisecond)
		nginxAPI, err = plus.NewNginxAPIController(&http.Client{}, "http://127.0.0.1:8080/api", local)
		if err != nil {
			glog.Fatalf("Failed to create NginxAPIController: %v", err)
		}
	}
	cnf := nginx.NewConfigurator(ngxc, nginxConfig, nginxAPI)

	lbc, _ := controller.NewLoadBalancerController(kubeClient, 30*time.Second, *watchNamespace, cnf, *nginxConfigMaps, *defaultServerSecret, *nginxPlus, *ingressClass, *useIngressClassOnly)
	go handleTermination(lbc, ngxc, nginxDone)
	lbc.Run()

	for {
		glog.Info("Waiting for the controller to exit...")
		time.Sleep(30 * time.Second)
	}
}

func handleTermination(lbc *controller.LoadBalancerController, ngxc *nginx.NginxController, nginxDone chan error) {
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
			glog.Info("nginx command exited succesfully")
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

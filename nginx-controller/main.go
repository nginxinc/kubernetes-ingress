package main

import (
	"flag"
	"time"

	"github.com/golang/glog"

	"github.com/nginxinc/kubernetes-ingress/nginx-controller/controller"
	"github.com/nginxinc/kubernetes-ingress/nginx-controller/nginx"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

var (
	// Set during build
	version string

	proxyURL = flag.String("proxy", "",
		`If specified, the controller assumes a kubctl proxy server is running on the
		given url and creates a proxy client. Regenerated NGINX configuration files
    are not written to the disk, instead they are printed to stdout. Also NGINX
    is not getting invoked. This flag is for testing.`)

	watchNamespace = flag.String("watch-namespace", api.NamespaceAll,
		`Namespace to watch for Ingress/Services/Endpoints. By default the controller
		watches acrosss all namespaces`)

	nginxConfigMaps = flag.String("nginx-configmaps", "",
		`Specifies a configmaps resource that can be used to customize NGINX
		configuration. The value must follow the following format: <namespace>/<name>`)

	enableMerging = flag.Bool("enable-merging", false,
		`Enables merging of ingress rules in multiple ingress objects targeting the same host.
		By default referencing the same host in multiple ingress objects will result in an error,
		and only the first ingress object will be used by the ingress controller.
		If this flag is enabled these rules will be merged using the server generated of
		the oldest ingress object as a base and adding the locations and settings of
		every ingress object in descending oder of their age.
		This is similar to the behavoir of other nginx ingress controller.`)
)

func main() {
	flag.Parse()

	glog.Infof("Starting NGINX Ingress controller Version %v\n", version)

	var kubeClient *client.Client
	var local = false

	if *proxyURL != "" {
		kubeClient = client.NewOrDie(&client.Config{
			Host: *proxyURL,
		})
		local = true
	} else {
		var err error
		kubeClient, err = client.NewInCluster()
		if err != nil {
			glog.Fatalf("Failed to create client: %v.", err)
		}
	}

	ngxc, _ := nginx.NewNginxController("/etc/nginx/", local)
	ngxc.Start()
	config := nginx.NewDefaultConfig()
	var merger nginx.Merger
	if *enableMerging {
		merger = nginx.NewOldestFirstMerger()
	} else {
		merger = nginx.NewNeverMerger()
	}
	cnf := nginx.NewConfigurator(ngxc, merger, config)
	lbc, _ := controller.NewLoadBalancerController(kubeClient, 30*time.Second, *watchNamespace, cnf, *nginxConfigMaps)
	lbc.Run()
}

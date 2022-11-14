package healthcheck

import (
	"strconv"

	apiv1 "k8s.io/api/core/v1"

	"github.com/golang/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/nginx-plus-go-client/client"
)

// RunHealthCheck starts the deep healthcheck service.
//
// If the server encounters an error and it can't start
// it exits with code 255 (glog.Fatal).
func RunHealthCheck(port int, plusClient *client.NginxClient, cnf *configs.Configurator, healthProbeTLSSecret *apiv1.Secret) {
	err := RunHealtcheckServer(strconv.Itoa(port), plusClient, cnf, healthProbeTLSSecret)
	if err != nil {
		glog.Fatal(err)
	}
}

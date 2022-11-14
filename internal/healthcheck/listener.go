package healthcheck

import (
	apiv1 "k8s.io/api/core/v1"
	"strconv"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/nginx-plus-go-client/client"
)

func RunHealthCheck(port int, plusClient *client.NginxClient, cnf *configs.Configurator, healthProbeTLSSecret *apiv1.Secret) {
	RunHealtcheckServer(strconv.Itoa(port), plusClient, cnf, healthProbeTLSSecret)
}

package healthcheck

import (
	"fmt"
	"path"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/nginx-plus-go-client/client"
	"k8s.io/utils/strings/slices"

	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

const healthEndpoint = "/healthcheck"

func RunHealthCheck(plusClient *client.NginxClient, cnf *configs.Configurator) {
	glog.Infof("Running Health Endpoint at port 9000")
	runServer(strconv.Itoa(9000), plusClient, cnf)
}

func runServer(port string, plusClient *client.NginxClient, cnf *configs.Configurator) {
	http.HandleFunc("/healthcheck/", func(w http.ResponseWriter, r *http.Request) {
		hostname := path.Base(r.URL.Path)
		glog.Infof("Path: %s", hostname)

		upstreamNames := cnf.GetUpstreamsforHost(hostname)
		glog.Infof("Upstream Names: %s", upstreamNames)

		upstreams, err := plusClient.GetUpstreams()
		if err != nil {
			// handle error
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		var states []string

		for name, u := range *upstreams {
			if slices.Contains(upstreamNames, name) {
				for _, p := range u.Peers {
					glog.Infof("Peer ID: %v, Name: %v, State: %v", p.ID, p.Name, p.State)
					states = append(states, "IP: "+p.Name+", State: "+p.State)
				}
			}
		}

		w.WriteHeader(http.StatusCreated)

		resp := `<html>
			<head><title>NGINX Ingress Controller</title></head>
			<body>
			<h1>NGINX Ingress Controller</h1><p>Hostname: ` + hostname +
			`</p><p>upstreams: ` + strings.Join(upstreamNames[:], ", ") +
			`</p><p>states: ` + strings.Join(states[:], ", ") +
			`</p></body></html>`
		_, err = w.Write([]byte(resp))
		if err != nil {
			glog.Warningf("Error while sending a response for the '/' path: %v", err)
		}
	})
	address := fmt.Sprintf(":%v", port)
	glog.Infof("Starting Healthcheck listener on: %v%v", address, healthEndpoint)
	glog.Fatal("Error in Healthcheck listener server: ", http.ListenAndServe(address, nil))

}

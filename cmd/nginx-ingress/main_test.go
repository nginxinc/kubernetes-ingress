package main

import (
	"flag"
	"os"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
)

func TestGetNginxVersionInfo(t *testing.T) {
	os.Args = append(os.Args, "-nginx-plus")
	os.Args = append(os.Args, "-proxy")
	os.Args = append(os.Args, "test-proxy")
	flag.Parse()
	constLabels := map[string]string{"class": *ingressClass}
	mc := collectors.NewLocalManagerMetricsCollector(constLabels)
	nginxManager, _ := createNginxManager(mc)
	nginxInfo, _ := getNginxVersionInfo(nginxManager)
	if nginxInfo.String() == "" {
		t.Errorf("Error when getting nginx version, empty string")
	}
}

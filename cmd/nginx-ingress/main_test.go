package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
)

// Test for getNginxVersionInfo()
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

func TestGetAppProtectVersionInfo(t *testing.T) {
	dataToWrite := "1.2.3\n"
	f, err := os.CreateTemp("/var/tmp/", "dat1")
	if err != nil {
		fmt.Println(err)
		return
	}
	l, err := f.WriteString(dataToWrite)
	if err != nil {
		fmt.Println(err)
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}

		return
	}
	fmt.Println(l, "bytes written successfully")
	if err := f.Close(); err != nil {
		fmt.Println(err)
		return
	}
	version, err := getAppProtectVersionInfo()
	if err != nil {
		t.Errorf("Error reading AppProtect Version file #{err}\n")
	}

	if version == "" {
		t.Errorf("Error with AppProtect Version, is empty")
	}
}

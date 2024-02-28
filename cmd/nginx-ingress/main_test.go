package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

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

func TestCreateGlobalConfigurationValidator(t *testing.T) {
	globalConfiguration := conf_v1.GlobalConfiguration{
		Spec: conf_v1.GlobalConfigurationSpec{
			Listeners: []conf_v1.Listener{
				{
					Name:     "tcp-listener",
					Port:     53,
					Protocol: "TCP",
				},
				{
					Name:     "udp-listener",
					Port:     53,
					Protocol: "UDP",
				},
			},
		},
	}

	gcv := createGlobalConfigurationValidator()

	if err := gcv.ValidateGlobalConfiguration(&globalConfiguration); err != nil {
		t.Errorf("ValidateGlobalConfiguration() returned error %v for valid input", err)
	}

	incorrectGlobalConf := conf_v1.GlobalConfiguration{
		Spec: conf_v1.GlobalConfigurationSpec{
			Listeners: []conf_v1.Listener{
				{
					Name:     "tcp-listener",
					Port:     53,
					Protocol: "TCPT",
				},
				{
					Name:     "udp-listener",
					Port:     53,
					Protocol: "UDP",
				},
			},
		},
	}

	if err := gcv.ValidateGlobalConfiguration(&incorrectGlobalConf); err == nil {
		t.Errorf("ValidateGlobalConfiguration() returned error %v for invalid input", err)
	}
}

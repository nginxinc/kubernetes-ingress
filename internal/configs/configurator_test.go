package configs

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
)

func createTestStaticConfigParams() *StaticConfigParams {
	return &StaticConfigParams{
		HealthStatus:                   true,
		NginxStatus:                    true,
		NginxStatusAllowCIDRs:          []string{"127.0.0.1"},
		NginxStatusPort:                8080,
		StubStatusOverUnixSocketForOSS: false,
	}
}

func createTestConfigurator() (*Configurator, error) {
	templateExecutor, err := version1.NewTemplateExecutor("version1/nginx-plus.tmpl", "version1/nginx-plus.ingress.tmpl")
	if err != nil {
		return nil, err
	}

	manager := nginx.NewFakeManager("/etc/nginx")

	return NewConfigurator(manager, createTestStaticConfigParams(), NewDefaultConfigParams(), templateExecutor, false, false), nil
}

func createTestConfiguratorInvalidIngressTemplate() (*Configurator, error) {
	templateExecutor, err := version1.NewTemplateExecutor("version1/nginx-plus.tmpl", "version1/nginx-plus.ingress.tmpl")
	if err != nil {
		return nil, err
	}

	invalidIngressTemplate := "{{.Upstreams.This.Field.Does.Not.Exist}}"
	if err := templateExecutor.UpdateIngressTemplate(&invalidIngressTemplate); err != nil {
		return nil, err
	}

	manager := nginx.NewFakeManager("/etc/nginx")

	return NewConfigurator(manager, createTestStaticConfigParams(), NewDefaultConfigParams(), templateExecutor, false, false), nil
}

func TestAddOrUpdateIngress(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()

	err = cnf.AddOrUpdateIngress(&ingress)
	if err != nil {
		t.Errorf("AddOrUpdateIngress returned:  \n%v, but expected: \n%v", err, nil)
	}

	cnfHasIngress := cnf.HasIngress(ingress.Ingress)
	if !cnfHasIngress {
		t.Errorf("AddOrUpdateIngress didn't add ingress successfully. HasIngress returned %v, expected %v", cnfHasIngress, true)
	}
}

func TestAddOrUpdateMergeableIngress(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngess := createMergeableCafeIngress()

	err = cnf.AddOrUpdateMergeableIngress(mergeableIngess)
	if err != nil {
		t.Errorf("AddOrUpdateMergeableIngress returned \n%v, expected \n%v", err, nil)
	}

	cnfHasMergeableIngress := cnf.HasIngress(mergeableIngess.Master.Ingress)
	if !cnfHasMergeableIngress {
		t.Errorf("AddOrUpdateMergeableIngress didn't add mergeable ingress successfully. HasIngress returned %v, expected %v", cnfHasMergeableIngress, true)
	}
}

func TestAddOrUpdateIngressFailsWithInvalidIngressTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()

	err = cnf.AddOrUpdateIngress(&ingress)
	if err == nil {
		t.Errorf("AddOrUpdateIngressFailsWithInvalidTemplate returned \n%v,  but expected \n%v", nil, "template execution error")
	}
}

func TestAddOrUpdateMergeableIngressFailsWithInvalidIngressTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngess := createMergeableCafeIngress()

	err = cnf.AddOrUpdateMergeableIngress(mergeableIngess)
	if err == nil {
		t.Errorf("AddOrUpdateMergeableIngress returned \n%v, but expected \n%v", nil, "template execution error")
	}
}

func TestUpdateEndpoints(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()
	ingresses := []*IngressEx{&ingress}

	err = cnf.UpdateEndpoints(ingresses)
	if err != nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", err, nil)
	}

	err = cnf.UpdateEndpoints(ingresses)
	if err != nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", err, nil)
	}
}

func TestUpdateEndpointsMergeableIngress(t *testing.T) {
	cnf, err := createTestConfigurator()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngress := createMergeableCafeIngress()
	mergeableIngresses := []*MergeableIngresses{mergeableIngress}

	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
	if err != nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", err, nil)
	}

	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
	if err != nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", err, nil)
	}
}

func TestUpdateEndpointsFailsWithInvalidTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	ingress := createCafeIngressEx()
	ingresses := []*IngressEx{&ingress}

	err = cnf.UpdateEndpoints(ingresses)
	if err == nil {
		t.Errorf("UpdateEndpoints returned\n%v, but expected \n%v", nil, "template execution error")
	}
}

func TestUpdateEndpointsMergeableIngressFailsWithInvalidTemplate(t *testing.T) {
	cnf, err := createTestConfiguratorInvalidIngressTemplate()
	if err != nil {
		t.Errorf("Failed to create a test configurator: %v", err)
	}

	mergeableIngress := createMergeableCafeIngress()
	mergeableIngresses := []*MergeableIngresses{mergeableIngress}

	err = cnf.UpdateEndpointsMergeableIngress(mergeableIngresses)
	if err == nil {
		t.Errorf("UpdateEndpointsMergeableIngress returned \n%v, but expected \n%v", nil, "template execution error")
	}
}

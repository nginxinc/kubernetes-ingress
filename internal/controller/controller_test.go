package controller

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx/plus"
	"k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestIsNginxIngress(t *testing.T) {
	ingressClass := "ing-ctrl"

	var testsWithoutIngressClassOnly = []struct {
		lbc      *LoadBalancerController
		ing      *extensions.Ingress
		expected bool
	}{
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: false,
			},
			&extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: ""},
				},
			},
			true,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: false,
			},
			&extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: "gce"},
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: false,
			},
			&extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: ingressClass},
				},
			},
			true,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: false,
			},
			&extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			true,
		},
	}

	var testsWithIngressClassOnly = []struct {
		lbc      *LoadBalancerController
		ing      *extensions.Ingress
		expected bool
	}{
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true,
			},
			&extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: ""},
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true,
			},
			&extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: "gce"},
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true,
			},
			&extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: ingressClass},
				},
			},
			true,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true,
			},
			&extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			false,
		},
	}

	for _, test := range testsWithoutIngressClassOnly {
		if result := test.lbc.isNginxIngress(test.ing); result != test.expected {
			classAnnotation := "N/A"
			if class, exists := test.ing.Annotations[ingressClassKey]; exists {
				classAnnotation = class
			}
			t.Errorf("lbc.isNginxIngress(ing), lbc.ingressClass=%v, lbc.useIngressClassOnly=%v, ing.Annotations['%v']=%v; got %v, expected %v",
				test.lbc.ingressClass, test.lbc.useIngressClassOnly, ingressClassKey, classAnnotation, result, test.expected)
		}
	}

	for _, test := range testsWithIngressClassOnly {
		if result := test.lbc.isNginxIngress(test.ing); result != test.expected {
			classAnnotation := "N/A"
			if class, exists := test.ing.Annotations[ingressClassKey]; exists {
				classAnnotation = class
			}
			t.Errorf("lbc.isNginxIngress(ing), lbc.ingressClass=%v, lbc.useIngressClassOnly=%v, ing.Annotations['%v']=%v; got %v, expected %v",
				test.lbc.ingressClass, test.lbc.useIngressClassOnly, ingressClassKey, classAnnotation, result, test.expected)
		}
	}

}

func TestCreateMergableIngresses(t *testing.T) {
	cafeMaster, coffeeMinion, teaMinion, lbc := getMergableDefaults()

	lbc.ingLister.Add(&cafeMaster)
	lbc.ingLister.Add(&coffeeMinion)
	lbc.ingLister.Add(&teaMinion)

	mergeableIngresses, err := lbc.createMergableIngresses(&cafeMaster)
	if err != nil {
		t.Errorf("Error creating Mergable Ingresses: %v", err)
	}
	if mergeableIngresses.Master.Ingress.Name != cafeMaster.Name && mergeableIngresses.Master.Ingress.Namespace != cafeMaster.Namespace {
		t.Errorf("Master %s not set properly", cafeMaster.Name)
	}

	if len(mergeableIngresses.Minions) != 2 {
		t.Errorf("Invalid amount of minions in mergeableIngresses: %v", mergeableIngresses.Minions)
	}

	coffeeCount := 0
	teaCount := 0
	for _, minion := range mergeableIngresses.Minions {
		if minion.Ingress.Name == coffeeMinion.Name {
			coffeeCount++
		} else if minion.Ingress.Name == teaMinion.Name {
			teaCount++
		} else {
			t.Errorf("Invalid Minion %s exists", minion.Ingress.Name)
		}
	}

	if coffeeCount != 1 {
		t.Errorf("Invalid amount of coffee Minions, amount %d", coffeeCount)
	}

	if teaCount != 1 {
		t.Errorf("Invalid amount of tea Minions, amount %d", teaCount)
	}
}

func TestCreateMergableIngressesInvalidMaster(t *testing.T) {
	cafeMaster, _, _, lbc := getMergableDefaults()

	// Test Error when Master has a Path
	cafeMaster.Spec.Rules = []extensions.IngressRule{
		extensions.IngressRule{
			Host: "ok.com",
			IngressRuleValue: extensions.IngressRuleValue{
				HTTP: &extensions.HTTPIngressRuleValue{
					Paths: []extensions.HTTPIngressPath{
						extensions.HTTPIngressPath{
							Path: "/coffee",
							Backend: extensions.IngressBackend{
								ServiceName: "coffee-svc",
								ServicePort: intstr.IntOrString{
									StrVal: "80",
								},
							},
						},
					},
				},
			},
		},
	}
	lbc.ingLister.Add(&cafeMaster)

	expected := fmt.Errorf("Ingress Resource %v/%v with the 'nginx.org/mergeable-ingress-type' annotation set to 'master' cannot contain Paths", cafeMaster.Namespace, cafeMaster.Name)
	_, err := lbc.createMergableIngresses(&cafeMaster)
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Error Validating the Ingress Resource: \n Expected: %s \n Obtained: %s", expected, err)
	}
}

func TestFindMasterForMinion(t *testing.T) {
	cafeMaster, coffeeMinion, teaMinion, lbc := getMergableDefaults()

	// Makes sure there is an empty path assigned to a master, to allow for lbc.createIngress() to pass
	cafeMaster.Spec.Rules[0].HTTP = &extensions.HTTPIngressRuleValue{
		Paths: []extensions.HTTPIngressPath{},
	}

	lbc.ingLister.Add(&cafeMaster)
	lbc.ingLister.Add(&coffeeMinion)
	lbc.ingLister.Add(&teaMinion)

	master, err := lbc.findMasterForMinion(&coffeeMinion)
	if err != nil {
		t.Errorf("Error finding master for %s(Minion): %v", coffeeMinion.Name, err)
	}
	if master.Name != cafeMaster.Name && master.Namespace != cafeMaster.Namespace {
		t.Errorf("Invalid Master found. Obtained %+v, Expected %+v", master, cafeMaster)
	}

	master, err = lbc.findMasterForMinion(&teaMinion)
	if err != nil {
		t.Errorf("Error finding master for %s(Minion): %v", teaMinion.Name, err)
	}
	if master.Name != cafeMaster.Name && master.Namespace != cafeMaster.Namespace {
		t.Errorf("Invalid Master found. Obtained %+v, Expected %+v", master, cafeMaster)
	}
}

func TestFindMasterForMinionNoMaster(t *testing.T) {
	_, coffeeMinion, teaMinion, lbc := getMergableDefaults()

	lbc.ingLister.Add(&coffeeMinion)
	lbc.ingLister.Add(&teaMinion)

	expected := fmt.Errorf("Could not find a Master for Minion: '%v/%v'", coffeeMinion.Namespace, coffeeMinion.Name)
	_, err := lbc.findMasterForMinion(&coffeeMinion)
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Expected: %s \nObtained: %s", expected, err)
	}

	expected = fmt.Errorf("Could not find a Master for Minion: '%v/%v'", teaMinion.Namespace, teaMinion.Name)
	_, err = lbc.findMasterForMinion(&teaMinion)
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Error master found for %s(Minion): %v", teaMinion.Name, err)
	}
}

func TestFindMasterForMinionInvalidMinion(t *testing.T) {
	cafeMaster, coffeeMinion, _, lbc := getMergableDefaults()

	// Makes sure there is an empty path assigned to a master, to allow for lbc.createIngress() to pass
	cafeMaster.Spec.Rules[0].HTTP = &extensions.HTTPIngressRuleValue{
		Paths: []extensions.HTTPIngressPath{},
	}

	coffeeMinion.Spec.Rules = []extensions.IngressRule{
		extensions.IngressRule{
			Host: "ok.com",
		},
	}

	lbc.ingLister.Add(&cafeMaster)
	lbc.ingLister.Add(&coffeeMinion)

	master, err := lbc.findMasterForMinion(&coffeeMinion)
	if err != nil {
		t.Errorf("Error finding master for %s(Minion): %v", coffeeMinion.Name, err)
	}
	if master.Name != cafeMaster.Name && master.Namespace != cafeMaster.Namespace {
		t.Errorf("Invalid Master found. Obtained %+v, Expected %+v", master, cafeMaster)
	}
}

func TestGetMinionsForMaster(t *testing.T) {
	cafeMaster, coffeeMinion, teaMinion, lbc := getMergableDefaults()

	// Makes sure there is an empty path assigned to a master, to allow for lbc.createIngress() to pass
	cafeMaster.Spec.Rules[0].HTTP = &extensions.HTTPIngressRuleValue{
		Paths: []extensions.HTTPIngressPath{},
	}

	lbc.ingLister.Add(&cafeMaster)
	lbc.ingLister.Add(&coffeeMinion)
	lbc.ingLister.Add(&teaMinion)

	cafeMasterIngEx, err := lbc.createIngress(&cafeMaster)
	if err != nil {
		t.Errorf("Error creating %s(Master): %v", cafeMaster.Name, err)
	}

	minions, err := lbc.getMinionsForMaster(cafeMasterIngEx)
	if err != nil {
		t.Errorf("Error getting Minions for %s(Master): %v", cafeMaster.Name, err)
	}

	if len(minions) != 2 {
		t.Errorf("Invalid amount of minions: %+v", minions)
	}

	coffeeCount := 0
	teaCount := 0
	for _, minion := range minions {
		if minion.Ingress.Name == coffeeMinion.Name {
			coffeeCount++
		} else if minion.Ingress.Name == teaMinion.Name {
			teaCount++
		} else {
			t.Errorf("Invalid Minion %s exists", minion.Ingress.Name)
		}
	}

	if coffeeCount != 1 {
		t.Errorf("Invalid amount of coffee Minions, amount %d", coffeeCount)
	}

	if teaCount != 1 {
		t.Errorf("Invalid amount of tea Minions, amount %d", teaCount)
	}
}

func TestGetMinionsForMasterInvalidMinion(t *testing.T) {
	cafeMaster, coffeeMinion, teaMinion, lbc := getMergableDefaults()

	// Makes sure there is an empty path assigned to a master, to allow for lbc.createIngress() to pass
	cafeMaster.Spec.Rules[0].HTTP = &extensions.HTTPIngressRuleValue{
		Paths: []extensions.HTTPIngressPath{},
	}

	teaMinion.Spec.Rules = []extensions.IngressRule{
		extensions.IngressRule{
			Host: "ok.com",
		},
	}

	lbc.ingLister.Add(&cafeMaster)
	lbc.ingLister.Add(&coffeeMinion)
	lbc.ingLister.Add(&teaMinion)

	cafeMasterIngEx, err := lbc.createIngress(&cafeMaster)
	if err != nil {
		t.Errorf("Error creating %s(Master): %v", cafeMaster.Name, err)
	}

	minions, err := lbc.getMinionsForMaster(cafeMasterIngEx)
	if err != nil {
		t.Errorf("Error getting Minions for %s(Master): %v", cafeMaster.Name, err)
	}

	if len(minions) != 1 {
		t.Errorf("Invalid amount of minions: %+v", minions)
	}

	coffeeCount := 0
	teaCount := 0
	for _, minion := range minions {
		if minion.Ingress.Name == coffeeMinion.Name {
			coffeeCount++
		} else if minion.Ingress.Name == teaMinion.Name {
			teaCount++
		} else {
			t.Errorf("Invalid Minion %s exists", minion.Ingress.Name)
		}
	}

	if coffeeCount != 1 {
		t.Errorf("Invalid amount of coffee Minions, amount %d", coffeeCount)
	}

	if teaCount != 0 {
		t.Errorf("Invalid amount of tea Minions, amount %d", teaCount)
	}
}

func TestGetMinionsForMasterConflictingPaths(t *testing.T) {
	cafeMaster, coffeeMinion, teaMinion, lbc := getMergableDefaults()

	// Makes sure there is an empty path assigned to a master, to allow for lbc.createIngress() to pass
	cafeMaster.Spec.Rules[0].HTTP = &extensions.HTTPIngressRuleValue{
		Paths: []extensions.HTTPIngressPath{},
	}

	coffeeMinion.Spec.Rules[0].HTTP.Paths = append(coffeeMinion.Spec.Rules[0].HTTP.Paths, extensions.HTTPIngressPath{
		Path: "/tea",
		Backend: extensions.IngressBackend{
			ServiceName: "tea-svc",
			ServicePort: intstr.IntOrString{
				StrVal: "80",
			},
		},
	})

	lbc.ingLister.Add(&cafeMaster)
	lbc.ingLister.Add(&coffeeMinion)
	lbc.ingLister.Add(&teaMinion)

	cafeMasterIngEx, err := lbc.createIngress(&cafeMaster)
	if err != nil {
		t.Errorf("Error creating %s(Master): %v", cafeMaster.Name, err)
	}

	minions, err := lbc.getMinionsForMaster(cafeMasterIngEx)
	if err != nil {
		t.Errorf("Error getting Minions for %s(Master): %v", cafeMaster.Name, err)
	}

	if len(minions) != 2 {
		t.Errorf("Invalid amount of minions: %+v", minions)
	}

	coffeePathCount := 0
	teaPathCount := 0
	for _, minion := range minions {
		for _, path := range minion.Ingress.Spec.Rules[0].HTTP.Paths {
			if path.Path == "/coffee" {
				coffeePathCount++
			} else if path.Path == "/tea" {
				teaPathCount++
			} else {
				t.Errorf("Invalid Path %s exists", path.Path)
			}
		}
	}

	if coffeePathCount != 1 {
		t.Errorf("Invalid amount of coffee paths, amount %d", coffeePathCount)
	}

	if teaPathCount != 1 {
		t.Errorf("Invalid amount of tea paths, amount %d", teaPathCount)
	}
}

func getMergableDefaults() (cafeMaster, coffeeMinion, teaMinion extensions.Ingress, lbc LoadBalancerController) {
	cafeMaster = extensions.Ingress{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-master",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
		Spec: extensions.IngressSpec{
			Rules: []extensions.IngressRule{
				extensions.IngressRule{
					Host: "ok.com",
				},
			},
		},
		Status: extensions.IngressStatus{},
	}
	coffeeMinion = extensions.Ingress{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: extensions.IngressSpec{
			Rules: []extensions.IngressRule{
				extensions.IngressRule{
					Host: "ok.com",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								extensions.HTTPIngressPath{
									Path: "/coffee",
									Backend: extensions.IngressBackend{
										ServiceName: "coffee-svc",
										ServicePort: intstr.IntOrString{
											StrVal: "80",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Status: extensions.IngressStatus{},
	}
	teaMinion = extensions.Ingress{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tea-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: extensions.IngressSpec{
			Rules: []extensions.IngressRule{
				extensions.IngressRule{
					Host: "ok.com",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								extensions.HTTPIngressPath{
									Path: "/tea",
								},
							},
						},
					},
				},
			},
		},
		Status: extensions.IngressStatus{},
	}

	ingExMap := make(map[string]*nginx.IngressEx)
	cafeMasterIngEx, _ := lbc.createIngress(&cafeMaster)
	ingExMap["default-cafe-master"] = cafeMasterIngEx

	cnf := nginx.NewConfigurator(&nginx.NginxController{}, &nginx.Config{}, &plus.NginxAPIController{}, &nginx.TemplateExecutor{})

	// edit private field ingresses to use in testing
	pointerVal := reflect.ValueOf(cnf)
	val := reflect.Indirect(pointerVal)

	field := val.FieldByName("ingresses")
	ptrToField := unsafe.Pointer(field.UnsafeAddr())
	realPtrToField := (*map[string]*nginx.IngressEx)(ptrToField)
	*realPtrToField = ingExMap

	fakeClient := fake.NewSimpleClientset()
	lbc = LoadBalancerController{
		client:       fakeClient,
		ingressClass: "nginx",
		cnf:          cnf,
	}
	lbc.svcLister, _ = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.ExtensionsV1beta1().RESTClient(), "services", "default", fields.Everything()),
		&extensions.Ingress{}, time.Duration(1), nil)
	lbc.ingLister.Store, _ = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.ExtensionsV1beta1().RESTClient(), "ingresses", "default", fields.Everything()),
		&extensions.Ingress{}, time.Duration(1), nil)
	coffeeService := v1.Service{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee-svc",
			Namespace: "default",
		},
		Spec:   v1.ServiceSpec{},
		Status: v1.ServiceStatus{},
	}
	teaService := v1.Service{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee-svc",
			Namespace: "default",
		},
		Spec:   v1.ServiceSpec{},
		Status: v1.ServiceStatus{},
	}
	lbc.svcLister.Add(coffeeService)
	lbc.svcLister.Add(teaService)

	return
}

func TestComparePorts(t *testing.T) {
	scenarios := []struct {
		sp       v1.ServicePort
		cp       v1.ContainerPort
		expected bool
	}{
		{
			// match TargetPort.strval and Protocol
			v1.ServicePort{
				TargetPort: intstr.FromString("name"),
				Protocol:   v1.ProtocolTCP,
			},
			v1.ContainerPort{
				Name:          "name",
				Protocol:      v1.ProtocolTCP,
				ContainerPort: 80,
			},
			true,
		},
		{
			// don't match Name and Protocol
			v1.ServicePort{
				Name:     "name",
				Protocol: v1.ProtocolTCP,
			},
			v1.ContainerPort{
				Name:          "name",
				Protocol:      v1.ProtocolTCP,
				ContainerPort: 80,
			},
			false,
		},
		{
			// TargetPort intval mismatch, don't match by TargetPort.Name
			v1.ServicePort{
				Name:       "name",
				TargetPort: intstr.FromInt(80),
			},
			v1.ContainerPort{
				Name:          "name",
				ContainerPort: 81,
			},
			false,
		},
		{
			// match by TargetPort intval
			v1.ServicePort{
				TargetPort: intstr.IntOrString{
					IntVal: 80,
				},
			},
			v1.ContainerPort{
				ContainerPort: 80,
			},
			true,
		},
		{
			// Fall back on ServicePort.Port if TargetPort is empty
			v1.ServicePort{
				Name: "name",
				Port: 80,
			},
			v1.ContainerPort{
				Name:          "name",
				ContainerPort: 80,
			},
			true,
		},
		{
			// TargetPort intval mismatch
			v1.ServicePort{
				TargetPort: intstr.FromInt(80),
			},
			v1.ContainerPort{
				ContainerPort: 81,
			},
			false,
		},
		{
			// don't match empty ports
			v1.ServicePort{},
			v1.ContainerPort{},
			false,
		},
	}

	for _, scen := range scenarios {
		if scen.expected != compareContainerPortAndServicePort(scen.cp, scen.sp) {
			t.Errorf("Expected: %v, ContainerPort: %v, ServicePort: %v", scen.expected, scen.cp, scen.sp)
		}
	}
}

func TestFindProbeForPods(t *testing.T) {
	pods := []v1.Pod{
		v1.Pod{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						ReadinessProbe: &v1.Probe{
							Handler: v1.Handler{
								HTTPGet: &v1.HTTPGetAction{
									Path: "/",
									Host: "asdf.com",
									Port: intstr.IntOrString{
										IntVal: 80,
									},
								},
							},
							PeriodSeconds: 42,
						},
						Ports: []v1.ContainerPort{
							v1.ContainerPort{
								Name:          "name",
								ContainerPort: 80,
								Protocol:      v1.ProtocolTCP,
								HostIP:        "1.2.3.4",
							},
						},
					},
				},
			},
		},
	}
	svcPort := v1.ServicePort{
		TargetPort: intstr.FromInt(80),
	}
	probe := findProbeForPods(pods, &svcPort)
	if probe == nil || probe.PeriodSeconds != 42 {
		t.Errorf("ServicePort.TargetPort as int match failed: %+v", probe)
	}

	svcPort = v1.ServicePort{
		TargetPort: intstr.FromString("name"),
		Protocol:   v1.ProtocolTCP,
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe == nil || probe.PeriodSeconds != 42 {
		t.Errorf("ServicePort.TargetPort as string failed: %+v", probe)
	}

	svcPort = v1.ServicePort{
		TargetPort: intstr.FromInt(80),
		Protocol:   v1.ProtocolTCP,
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe == nil || probe.PeriodSeconds != 42 {
		t.Errorf("ServicePort.TargetPort as int failed: %+v", probe)
	}

	svcPort = v1.ServicePort{
		Port: 80,
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe == nil || probe.PeriodSeconds != 42 {
		t.Errorf("ServicePort.Port should match if TargetPort is not set: %+v", probe)
	}

	svcPort = v1.ServicePort{
		TargetPort: intstr.FromString("wrong_name"),
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe != nil {
		t.Errorf("ServicePort.TargetPort should not have matched string: %+v", probe)
	}

	svcPort = v1.ServicePort{
		TargetPort: intstr.FromInt(22),
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe != nil {
		t.Errorf("ServicePort.TargetPort should not have matched int: %+v", probe)
	}

	svcPort = v1.ServicePort{
		Port: 22,
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe != nil {
		t.Errorf("ServicePort.Port mismatch: %+v", probe)
	}

}

func TestGetServicePortForIngressPort(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	cnf := nginx.NewConfigurator(&nginx.NginxController{}, &nginx.Config{}, &plus.NginxAPIController{}, &nginx.TemplateExecutor{})
	lbc := LoadBalancerController{
		client:       fakeClient,
		ingressClass: "nginx",
		cnf:          cnf,
	}
	svc := v1.Service{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee-svc",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "foo",
					Port:       80,
					TargetPort: intstr.FromInt(22),
				},
			},
		},
		Status: v1.ServiceStatus{},
	}
	ingSvcPort := intstr.FromString("foo")
	svcPort := lbc.getServicePortForIngressPort(ingSvcPort, &svc)
	if svcPort == nil || svcPort.Port != 80 {
		t.Errorf("TargetPort string match failed: %+v", svcPort)
	}

	ingSvcPort = intstr.FromInt(80)
	svcPort = lbc.getServicePortForIngressPort(ingSvcPort, &svc)
	if svcPort == nil || svcPort.Port != 80 {
		t.Errorf("TargetPort int match failed: %+v", svcPort)
	}

	ingSvcPort = intstr.FromInt(22)
	svcPort = lbc.getServicePortForIngressPort(ingSvcPort, &svc)
	if svcPort != nil {
		t.Errorf("Mismatched ints should not return port: %+v", svcPort)
	}
	ingSvcPort = intstr.FromString("bar")
	svcPort = lbc.getServicePortForIngressPort(ingSvcPort, &svc)
	if svcPort != nil {
		t.Errorf("Mismatched strings should not return port: %+v", svcPort)
	}
}

func TestFindIngressesForSecret(t *testing.T) {
	testCases := []struct {
		secret         v1.Secret
		ingress        extensions.Ingress
		expectedToFind bool
		desc           string
	}{
		{
			secret: v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-tls-secret",
					Namespace: "namespace-1",
				},
			},
			ingress: extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-ingress",
					Namespace: "namespace-1",
				},
				Spec: extensions.IngressSpec{
					TLS: []extensions.IngressTLS{
						extensions.IngressTLS{
							SecretName: "my-tls-secret",
						},
					},
				},
			},
			expectedToFind: true,
			desc:           "an Ingress references a TLS Secret that exists in the Ingress namespace",
		},
		{
			secret: v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-tls-secret",
					Namespace: "namespace-1",
				},
			},
			ingress: extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-ingress",
					Namespace: "namespace-2",
				},
				Spec: extensions.IngressSpec{
					TLS: []extensions.IngressTLS{
						extensions.IngressTLS{
							SecretName: "my-tls-secret",
						},
					},
				},
			},
			expectedToFind: false,
			desc:           "an Ingress references a TLS Secret that exists in a different namespace",
		},
		{
			secret: v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-jwk-secret",
					Namespace: "namespace-1",
				},
			},
			ingress: extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-ingress",
					Namespace: "namespace-1",
					Annotations: map[string]string{
						nginx.JWTKeyAnnotation: "my-jwk-secret",
					},
				},
			},
			expectedToFind: true,
			desc:           "an Ingress references a JWK Secret that exists in the Ingress namespace",
		},
		{
			secret: v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-jwk-secret",
					Namespace: "namespace-1",
				},
			},
			ingress: extensions.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-ingress",
					Namespace: "namespace-2",
					Annotations: map[string]string{
						nginx.JWTKeyAnnotation: "my-jwk-secret",
					},
				},
			},
			expectedToFind: false,
			desc:           "an Ingress references a JWK secret that exists in a different namespace",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset()

			templateExecutor, err := nginx.NewTemplateExecutor("../nginx/templates/nginx-plus.tmpl", "../nginx/templates/nginx-plus.ingress.tmpl", true, true, 8080)
			if err != nil {
				t.Fatalf("templateExecuter could not start: %v", err)
			}
			ngxc := nginx.NewNginxController("/etc/nginx", true)
			apiCtrl, err := plus.NewNginxAPIController(&http.Client{}, "", true)
			if err != nil {
				t.Fatalf("NGINX API Controller could not start: %v", err)
			}

			cnf := nginx.NewConfigurator(ngxc, &nginx.Config{}, apiCtrl, templateExecutor)
			lbc := LoadBalancerController{
				client:       fakeClient,
				ingressClass: "nginx",
				cnf:          cnf,
				nginxPlus:    true,
			}

			lbc.ingLister.Store, _ = cache.NewInformer(
				cache.NewListWatchFromClient(lbc.client.ExtensionsV1beta1().RESTClient(), "ingresses", "default", fields.Everything()),
				&extensions.Ingress{}, time.Duration(1), nil)

			lbc.secrLister.Store, lbc.secrController = cache.NewInformer(
				cache.NewListWatchFromClient(lbc.client.Core().RESTClient(), "secrets", "default", fields.Everything()),
				&v1.Secret{}, time.Duration(1), nil)

			ngxIngress := &nginx.IngressEx{
				Ingress: &test.ingress,
				TLSSecrets: map[string]*v1.Secret{
					test.secret.ObjectMeta.Name: &test.secret,
				},
			}

			err = cnf.AddOrUpdateIngress(ngxIngress)
			if err != nil {
				t.Fatalf("Ingress was not added: %v", err)
			}

			lbc.ingLister.Add(&test.ingress)
			lbc.secrLister.Add(&test.secret)

			nonMinions, minions, err := lbc.findIngressesForSecret(test.secret.ObjectMeta.Namespace, test.secret.ObjectMeta.Name)
			if err != nil {
				t.Fatalf("Couldn't find Ingress resource: %v", err)
			}

			if len(minions) > 0 {
				t.Fatalf("Expected 0 minions. Got: %v", len(minions))
			}

			if len(nonMinions) > 0 {
				if !test.expectedToFind {
					t.Fatalf("Expected 0 non-Minions. Got: %v", len(nonMinions))
				}
				if len(nonMinions) != 1 {
					t.Fatalf("Expected 1 non-Minion. Got: %v", len(nonMinions))
				}
				if nonMinions[0].Name != test.ingress.ObjectMeta.Name || nonMinions[0].Namespace != test.ingress.ObjectMeta.Namespace {
					t.Fatalf("Expected: %v/%v. Got: %v/%v.", test.ingress.ObjectMeta.Namespace, test.ingress.ObjectMeta.Name, nonMinions[0].Namespace, nonMinions[0].Name)
				}
			} else if test.expectedToFind {
				t.Fatal("Expected 1 non-Minion. Got: 0")
			}
		})
	}
}

package k8s

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestHasCorrectIngressClass(t *testing.T) {
	ingressClass := "ing-ctrl"
	incorrectIngressClass := "gce"
	emptyClass := ""

	var testsWithoutIngressClassOnly = []struct {
		lbc      *LoadBalancerController
		ing      *networking.Ingress
		expected bool
	}{
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: false,
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: emptyClass},
				},
			},
			true,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: false,
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: incorrectIngressClass},
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: false,
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
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
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			true,
		},
	}

	var testsWithIngressClassOnly = []struct {
		lbc      *LoadBalancerController
		ing      *networking.Ingress
		expected bool
	}{
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true,
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: emptyClass},
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true,
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: incorrectIngressClass},
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true,
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
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
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true, // always true for k8s >= 1.18
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				Spec: networking.IngressSpec{
					IngressClassName: &incorrectIngressClass,
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true, // always true for k8s >= 1.18
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				Spec: networking.IngressSpec{
					IngressClassName: &emptyClass,
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true, // always true for k8s >= 1.18
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Annotations: map[string]string{ingressClassKey: incorrectIngressClass},
				},
				Spec: networking.IngressSpec{
					IngressClassName: &ingressClass,
				},
			},
			false,
		},
		{
			&LoadBalancerController{
				ingressClass:        ingressClass,
				useIngressClassOnly: true, // always true for k8s >= 1.18
				metricsCollector:    collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				Spec: networking.IngressSpec{
					IngressClassName: &ingressClass,
				},
			},
			true,
		},
	}

	for _, test := range testsWithoutIngressClassOnly {
		if result := test.lbc.HasCorrectIngressClass(test.ing); result != test.expected {
			classAnnotation := "N/A"
			if class, exists := test.ing.Annotations[ingressClassKey]; exists {
				classAnnotation = class
			}
			t.Errorf("lbc.HasCorrectIngressClass(ing), lbc.ingressClass=%v, lbc.useIngressClassOnly=%v, ing.Annotations['%v']=%v; got %v, expected %v",
				test.lbc.ingressClass, test.lbc.useIngressClassOnly, ingressClassKey, classAnnotation, result, test.expected)
		}
	}

	for _, test := range testsWithIngressClassOnly {
		if result := test.lbc.HasCorrectIngressClass(test.ing); result != test.expected {
			classAnnotation := "N/A"
			if class, exists := test.ing.Annotations[ingressClassKey]; exists {
				classAnnotation = class
			}
			t.Errorf("lbc.HasCorrectIngressClass(ing), lbc.ingressClass=%v, lbc.useIngressClassOnly=%v, ing.Annotations['%v']=%v; got %v, expected %v",
				test.lbc.ingressClass, test.lbc.useIngressClassOnly, ingressClassKey, classAnnotation, result, test.expected)
		}
	}

}

func TestHasCorrectIngressClassVS(t *testing.T) {
	ingressClass := "ing-ctrl"
	lbcIngOnlyTrue := &LoadBalancerController{
		ingressClass:        ingressClass,
		useIngressClassOnly: true,
		metricsCollector:    collectors.NewControllerFakeCollector(),
	}

	var testsWithIngressClassOnlyVS = []struct {
		lbc      *LoadBalancerController
		ing      *conf_v1.VirtualServer
		expected bool
	}{
		{
			lbcIngOnlyTrue,
			&conf_v1.VirtualServer{
				Spec: conf_v1.VirtualServerSpec{
					IngressClass: "",
				},
			},
			true,
		},
		{
			lbcIngOnlyTrue,
			&conf_v1.VirtualServer{
				Spec: conf_v1.VirtualServerSpec{
					IngressClass: "gce",
				},
			},
			false,
		},
		{
			lbcIngOnlyTrue,
			&conf_v1.VirtualServer{
				Spec: conf_v1.VirtualServerSpec{
					IngressClass: ingressClass,
				},
			},
			true,
		},
		{
			lbcIngOnlyTrue,
			&conf_v1.VirtualServer{},
			true,
		},
	}

	lbcIngOnlyFalse := &LoadBalancerController{
		ingressClass:        ingressClass,
		useIngressClassOnly: false,
		metricsCollector:    collectors.NewControllerFakeCollector(),
	}
	var testsWithoutIngressClassOnlyVS = []struct {
		lbc      *LoadBalancerController
		ing      *conf_v1.VirtualServer
		expected bool
	}{
		{
			lbcIngOnlyFalse,
			&conf_v1.VirtualServer{
				Spec: conf_v1.VirtualServerSpec{
					IngressClass: "",
				},
			},
			true,
		},
		{
			lbcIngOnlyFalse,
			&conf_v1.VirtualServer{
				Spec: conf_v1.VirtualServerSpec{
					IngressClass: "gce",
				},
			},
			false,
		},
		{
			lbcIngOnlyFalse,
			&conf_v1.VirtualServer{
				Spec: conf_v1.VirtualServerSpec{
					IngressClass: ingressClass,
				},
			},
			true,
		},
		{
			lbcIngOnlyFalse,
			&conf_v1.VirtualServer{},
			true,
		},
	}

	for _, test := range testsWithIngressClassOnlyVS {
		if result := test.lbc.HasCorrectIngressClass(test.ing); result != test.expected {
			t.Errorf("lbc.HasCorrectIngressClass(ing), lbc.ingressClass=%v, lbc.useIngressClassOnly=%v, ingressClassKey=%v, ing.IngressClass=%v; got %v, expected %v",
				test.lbc.ingressClass, test.lbc.useIngressClassOnly, ingressClassKey, test.ing.Spec.IngressClass, result, test.expected)
		}
	}

	for _, test := range testsWithoutIngressClassOnlyVS {
		if result := test.lbc.HasCorrectIngressClass(test.ing); result != test.expected {
			t.Errorf("lbc.HasCorrectIngressClass(ing), lbc.ingressClass=%v, lbc.useIngressClassOnly=%v, ingressClassKey=%v, ing.IngressClass=%v; got %v, expected %v",
				test.lbc.ingressClass, test.lbc.useIngressClassOnly, ingressClassKey, test.ing.Spec.IngressClass, result, test.expected)
		}
	}
}

func TestCreateMergableIngresses(t *testing.T) {
	cafeMaster, coffeeMinion, teaMinion, lbc := getMergableDefaults()

	err := lbc.ingressLister.Add(&cafeMaster)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &cafeMaster.Name, err)
	}

	err = lbc.ingressLister.Add(&coffeeMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &coffeeMinion.Name, err)
	}

	err = lbc.ingressLister.Add(&teaMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &teaMinion.Name, err)
	}

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
	cafeMaster.Spec.Rules = []networking.IngressRule{
		{
			Host: "ok.com",
			IngressRuleValue: networking.IngressRuleValue{
				HTTP: &networking.HTTPIngressRuleValue{
					Paths: []networking.HTTPIngressPath{
						{
							Path: "/coffee",
							Backend: networking.IngressBackend{
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
	err := lbc.ingressLister.Add(&cafeMaster)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &cafeMaster.Name, err)
	}

	expected := fmt.Errorf("Ingress Resource %v/%v with the 'nginx.org/mergeable-ingress-type' annotation set to 'master' cannot contain Paths", cafeMaster.Namespace, cafeMaster.Name)
	_, err = lbc.createMergableIngresses(&cafeMaster)
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Error Validating the Ingress Resource: \n Expected: %s \n Obtained: %s", expected, err)
	}
}

func TestFindMasterForMinion(t *testing.T) {
	cafeMaster, coffeeMinion, teaMinion, lbc := getMergableDefaults()

	// Makes sure there is an empty path assigned to a master, to allow for lbc.createIngress() to pass
	cafeMaster.Spec.Rules[0].HTTP = &networking.HTTPIngressRuleValue{
		Paths: []networking.HTTPIngressPath{},
	}

	err := lbc.ingressLister.Add(&cafeMaster)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &cafeMaster.Name, err)
	}

	err = lbc.ingressLister.Add(&coffeeMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &coffeeMinion.Name, err)
	}

	err = lbc.ingressLister.Add(&teaMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &teaMinion.Name, err)
	}

	master, err := lbc.FindMasterForMinion(&coffeeMinion)
	if err != nil {
		t.Errorf("Error finding master for %s(Minion): %v", coffeeMinion.Name, err)
	}
	if master.Name != cafeMaster.Name && master.Namespace != cafeMaster.Namespace {
		t.Errorf("Invalid Master found. Obtained %+v, Expected %+v", master, cafeMaster)
	}

	master, err = lbc.FindMasterForMinion(&teaMinion)
	if err != nil {
		t.Errorf("Error finding master for %s(Minion): %v", teaMinion.Name, err)
	}
	if master.Name != cafeMaster.Name && master.Namespace != cafeMaster.Namespace {
		t.Errorf("Invalid Master found. Obtained %+v, Expected %+v", master, cafeMaster)
	}
}

func TestFindMasterForMinionNoMaster(t *testing.T) {
	_, coffeeMinion, teaMinion, lbc := getMergableDefaults()

	err := lbc.ingressLister.Add(&coffeeMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &coffeeMinion.Name, err)
	}

	err = lbc.ingressLister.Add(&teaMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &teaMinion.Name, err)
	}

	expected := fmt.Errorf("Could not find a Master for Minion: '%v/%v'", coffeeMinion.Namespace, coffeeMinion.Name)
	_, err = lbc.FindMasterForMinion(&coffeeMinion)
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Expected: %s \nObtained: %s", expected, err)
	}

	expected = fmt.Errorf("Could not find a Master for Minion: '%v/%v'", teaMinion.Namespace, teaMinion.Name)
	_, err = lbc.FindMasterForMinion(&teaMinion)
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Error master found for %s(Minion): %v", teaMinion.Name, err)
	}
}

func TestFindMasterForMinionInvalidMinion(t *testing.T) {
	cafeMaster, coffeeMinion, _, lbc := getMergableDefaults()

	// Makes sure there is an empty path assigned to a master, to allow for lbc.createIngress() to pass
	cafeMaster.Spec.Rules[0].HTTP = &networking.HTTPIngressRuleValue{
		Paths: []networking.HTTPIngressPath{},
	}

	coffeeMinion.Spec.Rules = []networking.IngressRule{
		{
			Host: "ok.com",
		},
	}

	err := lbc.ingressLister.Add(&cafeMaster)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &cafeMaster.Name, err)
	}

	err = lbc.ingressLister.Add(&coffeeMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &coffeeMinion.Name, err)
	}

	master, err := lbc.FindMasterForMinion(&coffeeMinion)
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
	cafeMaster.Spec.Rules[0].HTTP = &networking.HTTPIngressRuleValue{
		Paths: []networking.HTTPIngressPath{},
	}

	err := lbc.ingressLister.Add(&cafeMaster)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &cafeMaster.Name, err)
	}

	err = lbc.ingressLister.Add(&coffeeMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &coffeeMinion.Name, err)
	}

	err = lbc.ingressLister.Add(&teaMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &teaMinion.Name, err)
	}

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
	cafeMaster.Spec.Rules[0].HTTP = &networking.HTTPIngressRuleValue{
		Paths: []networking.HTTPIngressPath{},
	}

	teaMinion.Spec.Rules = []networking.IngressRule{
		{
			Host: "ok.com",
		},
	}

	err := lbc.ingressLister.Add(&cafeMaster)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &cafeMaster.Name, err)
	}

	err = lbc.ingressLister.Add(&coffeeMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &coffeeMinion.Name, err)
	}

	err = lbc.ingressLister.Add(&teaMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &teaMinion.Name, err)
	}

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
	cafeMaster.Spec.Rules[0].HTTP = &networking.HTTPIngressRuleValue{
		Paths: []networking.HTTPIngressPath{},
	}

	coffeeMinion.Spec.Rules[0].HTTP.Paths = append(coffeeMinion.Spec.Rules[0].HTTP.Paths, networking.HTTPIngressPath{
		Path: "/tea",
		Backend: networking.IngressBackend{
			ServiceName: "tea-svc",
			ServicePort: intstr.IntOrString{
				StrVal: "80",
			},
		},
	})

	err := lbc.ingressLister.Add(&cafeMaster)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &cafeMaster.Name, err)
	}

	err = lbc.ingressLister.Add(&coffeeMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &coffeeMinion.Name, err)
	}

	err = lbc.ingressLister.Add(&teaMinion)
	if err != nil {
		t.Errorf("Error adding Ingress %v to the ingress lister: %v", &teaMinion.Name, err)
	}

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

func getMergableDefaults() (cafeMaster, coffeeMinion, teaMinion networking.Ingress, lbc LoadBalancerController) {
	cafeMaster = networking.Ingress{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-master",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "ok.com",
				},
			},
		},
		Status: networking.IngressStatus{},
	}
	coffeeMinion = networking.Ingress{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "ok.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: "/coffee",
									Backend: networking.IngressBackend{
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
		Status: networking.IngressStatus{},
	}
	teaMinion = networking.Ingress{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tea-minion",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "minion",
			},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "ok.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path: "/tea",
								},
							},
						},
					},
				},
			},
		},
		Status: networking.IngressStatus{},
	}

	ingExMap := make(map[string]*configs.IngressEx)
	cafeMasterIngEx, _ := lbc.createIngress(&cafeMaster)
	ingExMap["default-cafe-master"] = cafeMasterIngEx

	cnf := configs.NewConfigurator(&nginx.LocalManager{}, &configs.StaticConfigParams{}, &configs.ConfigParams{}, &configs.GlobalConfigParams{}, &version1.TemplateExecutor{}, &version2.TemplateExecutor{}, false, false, nil, false, nil, false)

	// edit private field ingresses to use in testing
	pointerVal := reflect.ValueOf(cnf)
	val := reflect.Indirect(pointerVal)

	field := val.FieldByName("ingresses")
	ptrToField := unsafe.Pointer(field.UnsafeAddr())
	realPtrToField := (*map[string]*configs.IngressEx)(ptrToField)
	*realPtrToField = ingExMap

	fakeClient := fake.NewSimpleClientset()
	lbc = LoadBalancerController{
		client:           fakeClient,
		ingressClass:     "nginx",
		configurator:     cnf,
		metricsCollector: collectors.NewControllerFakeCollector(),
	}
	lbc.svcLister, _ = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.NetworkingV1beta1().RESTClient(), "services", "default", fields.Everything()),
		&networking.Ingress{}, time.Duration(1), nil)
	lbc.ingressLister.Store, _ = cache.NewInformer(
		cache.NewListWatchFromClient(lbc.client.NetworkingV1beta1().RESTClient(), "ingresses", "default", fields.Everything()),
		&networking.Ingress{}, time.Duration(1), nil)

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
	pods := []*v1.Pod{
		{
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
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
							{
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
	cnf := configs.NewConfigurator(&nginx.LocalManager{}, &configs.StaticConfigParams{}, &configs.ConfigParams{}, &configs.GlobalConfigParams{}, &version1.TemplateExecutor{}, &version2.TemplateExecutor{}, false, false, nil, false, nil, false)
	lbc := LoadBalancerController{
		client:           fakeClient,
		ingressClass:     "nginx",
		configurator:     cnf,
		metricsCollector: collectors.NewControllerFakeCollector(),
	}
	svc := v1.Service{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee-svc",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
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
		ingress        networking.Ingress
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
			ingress: networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-ingress",
					Namespace: "namespace-1",
				},
				Spec: networking.IngressSpec{
					TLS: []networking.IngressTLS{
						{
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
			ingress: networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-ingress",
					Namespace: "namespace-2",
				},
				Spec: networking.IngressSpec{
					TLS: []networking.IngressTLS{
						{
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
			ingress: networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-ingress",
					Namespace: "namespace-1",
					Annotations: map[string]string{
						configs.JWTKeyAnnotation: "my-jwk-secret",
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
			ingress: networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-ingress",
					Namespace: "namespace-2",
					Annotations: map[string]string{
						configs.JWTKeyAnnotation: "my-jwk-secret",
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

			templateExecutor, err := version1.NewTemplateExecutor("../configs/version1/nginx-plus.tmpl", "../configs/version1/nginx-plus.ingress.tmpl")
			if err != nil {
				t.Fatalf("templateExecutor could not start: %v", err)
			}

			templateExecutorV2, err := version2.NewTemplateExecutor("../configs/version2/nginx-plus.virtualserver.tmpl", "../configs/version2/nginx-plus.transportserver.tmpl")
			if err != nil {
				t.Fatalf("templateExecutorV2 could not start: %v", err)
			}

			manager := nginx.NewFakeManager("/etc/nginx")

			cnf := configs.NewConfigurator(manager, &configs.StaticConfigParams{}, &configs.ConfigParams{}, &configs.GlobalConfigParams{}, templateExecutor, templateExecutorV2, false, false, nil, false, nil, false)
			lbc := LoadBalancerController{
				client:           fakeClient,
				ingressClass:     "nginx",
				configurator:     cnf,
				isNginxPlus:      true,
				metricsCollector: collectors.NewControllerFakeCollector(),
			}

			lbc.ingressLister.Store, _ = cache.NewInformer(
				cache.NewListWatchFromClient(lbc.client.NetworkingV1beta1().RESTClient(), "ingresses", "default", fields.Everything()),
				&networking.Ingress{}, time.Duration(1), nil)

			lbc.secretLister.Store, lbc.secretController = cache.NewInformer(
				cache.NewListWatchFromClient(lbc.client.CoreV1().RESTClient(), "secrets", "default", fields.Everything()),
				&v1.Secret{}, time.Duration(1), nil)

			ngxIngress := &configs.IngressEx{
				Ingress: &test.ingress,
				TLSSecrets: map[string]*v1.Secret{
					test.secret.Name: &test.secret,
				},
			}

			err = cnf.AddOrUpdateIngress(ngxIngress)
			if err != nil {
				t.Fatalf("Ingress was not added: %v", err)
			}

			err = lbc.ingressLister.Add(&test.ingress)
			if err != nil {
				t.Errorf("Error adding Ingress %v to the ingress lister: %v", &test.ingress.Name, err)
			}

			err = lbc.secretLister.Add(&test.secret)
			if err != nil {
				t.Errorf("Error adding Secret %v to the secret lister: %v", &test.secret.Name, err)
			}

			ings, err := lbc.findIngressesForSecret(test.secret.Namespace, test.secret.Name)
			if err != nil {
				t.Fatalf("Couldn't find Ingress resource: %v", err)
			}

			if len(ings) > 0 {
				if !test.expectedToFind {
					t.Fatalf("Expected 0 ingresses. Got: %v", len(ings))
				}
				if len(ings) != 1 {
					t.Fatalf("Expected 1 ingress. Got: %v", len(ings))
				}
				if ings[0].Name != test.ingress.Name || ings[0].Namespace != test.ingress.Namespace {
					t.Fatalf("Expected: %v/%v. Got: %v/%v.", test.ingress.Namespace, test.ingress.Name, ings[0].Namespace, ings[0].Name)
				}
			} else if test.expectedToFind {
				t.Fatal("Expected 1 ingress. Got: 0")
			}
		})
	}
}

func TestFindIngressesForSecretWithMinions(t *testing.T) {
	testCases := []struct {
		secret         v1.Secret
		ingress        networking.Ingress
		expectedToFind bool
		desc           string
	}{
		{
			secret: v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-jwk-secret",
					Namespace: "default",
				},
			},
			ingress: networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "cafe-ingress-tea-minion",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class":      "nginx",
						"nginx.org/mergeable-ingress-type": "minion",
						configs.JWTKeyAnnotation:           "my-jwk-secret",
					},
				},
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host: "cafe.example.com",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path: "/tea",
											Backend: networking.IngressBackend{
												ServiceName: "tea-svc",
												ServicePort: intstr.FromString("80"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedToFind: true,
			desc:           "a minion Ingress references a JWK Secret that exists in the Ingress namespace",
		},
		{
			secret: v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "my-jwk-secret",
					Namespace: "namespace-1",
				},
			},
			ingress: networking.Ingress{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "cafe-ingress-tea-minion",
					Namespace: "default",
					Annotations: map[string]string{
						"kubernetes.io/ingress.class":      "nginx",
						"nginx.org/mergeable-ingress-type": "minion",
						configs.JWTKeyAnnotation:           "my-jwk-secret",
					},
				},
				Spec: networking.IngressSpec{
					Rules: []networking.IngressRule{
						{
							Host: "cafe.example.com",
							IngressRuleValue: networking.IngressRuleValue{
								HTTP: &networking.HTTPIngressRuleValue{
									Paths: []networking.HTTPIngressPath{
										{
											Path: "/tea",
											Backend: networking.IngressBackend{
												ServiceName: "tea-svc",
												ServicePort: intstr.FromString("80"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedToFind: false,
			desc:           "a Minion references a JWK secret that exists in a different namespace",
		},
	}

	master := networking.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "cafe-ingress-master",
			Namespace: "default",
			Annotations: map[string]string{
				"kubernetes.io/ingress.class":      "nginx",
				"nginx.org/mergeable-ingress-type": "master",
			},
		},
		Spec: networking.IngressSpec{
			Rules: []networking.IngressRule{
				{
					Host: "cafe.example.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{ // HTTP must not be nil for Master
							Paths: []networking.HTTPIngressPath{},
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			fakeClient := fake.NewSimpleClientset()

			templateExecutor, err := version1.NewTemplateExecutor("../configs/version1/nginx-plus.tmpl", "../configs/version1/nginx-plus.ingress.tmpl")
			if err != nil {
				t.Fatalf("templateExecutor could not start: %v", err)
			}

			templateExecutorV2, err := version2.NewTemplateExecutor("../configs/version2/nginx-plus.virtualserver.tmpl", "../configs/version2/nginx-plus.transportserver.tmpl")
			if err != nil {
				t.Fatalf("templateExecutorV2 could not start: %v", err)
			}

			manager := nginx.NewFakeManager("/etc/nginx")

			cnf := configs.NewConfigurator(manager, &configs.StaticConfigParams{}, &configs.ConfigParams{}, &configs.GlobalConfigParams{}, templateExecutor, templateExecutorV2, false, false, nil, false, nil, false)
			lbc := LoadBalancerController{
				client:           fakeClient,
				ingressClass:     "nginx",
				configurator:     cnf,
				isNginxPlus:      true,
				metricsCollector: collectors.NewControllerFakeCollector(),
			}

			lbc.ingressLister.Store, _ = cache.NewInformer(
				cache.NewListWatchFromClient(lbc.client.NetworkingV1beta1().RESTClient(), "ingresses", "default", fields.Everything()),
				&networking.Ingress{}, time.Duration(1), nil)

			lbc.secretLister.Store, lbc.secretController = cache.NewInformer(
				cache.NewListWatchFromClient(lbc.client.CoreV1().RESTClient(), "secrets", "default", fields.Everything()),
				&v1.Secret{}, time.Duration(1), nil)

			mergeable := &configs.MergeableIngresses{
				Master: &configs.IngressEx{
					Ingress: &master,
				},
				Minions: []*configs.IngressEx{
					{
						Ingress: &test.ingress,
						JWTKey: configs.JWTKey{
							Name: test.secret.Name,
						},
					},
				},
			}

			err = cnf.AddOrUpdateMergeableIngress(mergeable)
			if err != nil {
				t.Fatalf("Ingress was not added: %v", err)
			}

			err = lbc.ingressLister.Add(&master)
			if err != nil {
				t.Errorf("Error adding Ingress %v to the ingress lister: %v", &master.Name, err)
			}

			err = lbc.ingressLister.Add(&test.ingress)
			if err != nil {
				t.Errorf("Error adding Ingress %v to the ingress lister: %v", &test.ingress.Name, err)
			}

			err = lbc.secretLister.Add(&test.secret)
			if err != nil {
				t.Errorf("Error adding Secret %v to the secret lister: %v", &test.secret.Name, err)
			}

			ings, err := lbc.findIngressesForSecret(test.secret.Namespace, test.secret.Name)
			if err != nil {
				t.Fatalf("Couldn't find Ingress resource: %v", err)
			}

			if len(ings) > 0 {
				if !test.expectedToFind {
					t.Fatalf("Expected 0 ingresses. Got: %v", len(ings))
				}
				if len(ings) != 1 {
					t.Fatalf("Expected 1 ingress. Got: %v", len(ings))
				}
				if ings[0].Name != test.ingress.Name || ings[0].Namespace != test.ingress.Namespace {
					t.Fatalf("Expected: %v/%v. Got: %v/%v.", test.ingress.Namespace, test.ingress.Name, ings[0].Namespace, ings[0].Name)
				}
			} else if test.expectedToFind {
				t.Fatal("Expected 1 ingress. Got: 0")
			}
		})
	}
}

func TestFindVirtualServersForService(t *testing.T) {
	vs1 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-1",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			Upstreams: []conf_v1.Upstream{
				{
					Service: "test-service",
				},
			},
		},
	}
	vs2 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-2",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			Upstreams: []conf_v1.Upstream{
				{
					Service: "some-service",
				},
			},
		},
	}
	vs3 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-3",
			Namespace: "ns-2",
		},
		Spec: conf_v1.VirtualServerSpec{
			Upstreams: []conf_v1.Upstream{
				{
					Service: "test-service",
				},
			},
		},
	}
	virtualServers := []*conf_v1.VirtualServer{&vs1, &vs2, &vs3}

	service := v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "test-service",
			Namespace: "ns-1",
		},
	}

	expected := []*conf_v1.VirtualServer{&vs1}

	result := findVirtualServersForService(virtualServers, &service)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("findVirtualServersForService returned %v but expected %v", result, expected)
	}
}

func TestFindVirtualServerRoutesForService(t *testing.T) {
	vsr1 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vsr-1",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerRouteSpec{
			Upstreams: []conf_v1.Upstream{
				{
					Service: "test-service",
				},
			},
		},
	}
	vsr2 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vsr-2",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerRouteSpec{
			Upstreams: []conf_v1.Upstream{
				{
					Service: "some-service",
				},
			},
		},
	}
	vsr3 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vrs-3",
			Namespace: "ns-2",
		},
		Spec: conf_v1.VirtualServerRouteSpec{
			Upstreams: []conf_v1.Upstream{
				{
					Service: "test-service",
				},
			},
		},
	}
	virtualServerRoutes := []*conf_v1.VirtualServerRoute{&vsr1, &vsr2, &vsr3}

	service := v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "test-service",
			Namespace: "ns-1",
		},
	}

	expected := []*conf_v1.VirtualServerRoute{&vsr1}

	result := findVirtualServerRoutesForService(virtualServerRoutes, &service)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("findVirtualServerRoutesForService returned %v but expected %v", result, expected)
	}
}

func TestFindOrphanedVirtualServerRoute(t *testing.T) {
	vsr1 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vsr-1",
			Namespace: "ns-1",
		},
	}

	vsr2 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vsr-2",
			Namespace: "ns-1",
		},
	}

	vsr3 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vsr-3",
			Namespace: "ns-2",
		},
	}

	vsr4 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vsr-4",
			Namespace: "ns-1",
		},
	}

	vsr5 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vsr-5",
			Namespace: "ns-3",
		},
	}

	vsrs := []*conf_v1.VirtualServerRoute{&vsr1, &vsr2, &vsr3, &vsr4, &vsr5}

	handledVSRs := []*conf_v1.VirtualServerRoute{&vsr3}

	expected := []*conf_v1.VirtualServerRoute{&vsr1, &vsr2, &vsr4, &vsr5}

	result := findOrphanedVirtualServerRoutes(vsrs, handledVSRs)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("findOrphanedVirtualServerRoutes return %v but expected %v", result, expected)
	}
}

func TestFindTransportServersForService(t *testing.T) {
	ts1 := conf_v1alpha1.TransportServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "ts-1",
			Namespace: "ns-1",
		},
		Spec: conf_v1alpha1.TransportServerSpec{
			Upstreams: []conf_v1alpha1.Upstream{
				{
					Service: "test-service",
				},
			},
		},
	}
	ts2 := conf_v1alpha1.TransportServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "ts-2",
			Namespace: "ns-1",
		},
		Spec: conf_v1alpha1.TransportServerSpec{
			Upstreams: []conf_v1alpha1.Upstream{
				{
					Service: "some-service",
				},
			},
		},
	}
	ts3 := conf_v1alpha1.TransportServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "ts-3",
			Namespace: "ns-2",
		},
		Spec: conf_v1alpha1.TransportServerSpec{
			Upstreams: []conf_v1alpha1.Upstream{
				{
					Service: "test-service",
				},
			},
		},
	}
	transportServers := []*conf_v1alpha1.TransportServer{&ts1, &ts2, &ts3}

	service := v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "test-service",
			Namespace: "ns-1",
		},
	}

	expected := []*conf_v1alpha1.TransportServer{&ts1}

	result := findTransportServersForService(transportServers, &service)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("findTransportServersForService returned %v but expected %v", result, expected)
	}
}

func TestFindVirtualServersForSecret(t *testing.T) {
	vs1 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-1",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			TLS: nil,
		},
	}
	vs2 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-2",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			TLS: &conf_v1.TLS{
				Secret: "",
			},
		},
	}
	vs3 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-3",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			TLS: &conf_v1.TLS{
				Secret: "some-secret",
			},
		},
	}
	vs4 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-4",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			TLS: &conf_v1.TLS{
				Secret: "test-secret",
			},
		},
	}
	vs5 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-5",
			Namespace: "ns-2",
		},
		Spec: conf_v1.VirtualServerSpec{
			TLS: &conf_v1.TLS{
				Secret: "test-secret",
			},
		},
	}

	virtualServers := []*conf_v1.VirtualServer{&vs1, &vs2, &vs3, &vs4, &vs5}

	expected := []*conf_v1.VirtualServer{&vs4}

	result := findVirtualServersForSecret(virtualServers, "ns-1", "test-secret")
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("findVirtualServersForSecret returned %v but expected %v", result, expected)
	}
}

func TestFindVirtualServersForPolicy(t *testing.T) {
	vs1 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-1",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			Policies: []conf_v1.PolicyReference{
				{
					Name:      "test-policy",
					Namespace: "ns-1",
				},
			},
		},
	}
	vs2 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-2",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			Policies: []conf_v1.PolicyReference{
				{
					Name:      "some-policy",
					Namespace: "ns-1",
				},
			},
		},
	}
	vs3 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-3",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			Routes: []conf_v1.Route{
				{
					Policies: []conf_v1.PolicyReference{
						{
							Name:      "test-policy",
							Namespace: "ns-1",
						},
					},
				},
			},
		},
	}
	vs4 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-4",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			Routes: []conf_v1.Route{
				{
					Policies: []conf_v1.PolicyReference{
						{
							Name:      "some-policy",
							Namespace: "ns-1",
						},
					},
				},
			},
		},
	}

	virtualServers := []*conf_v1.VirtualServer{&vs1, &vs2, &vs3, &vs4}

	expected := []*conf_v1.VirtualServer{&vs1, &vs3}

	result := findVirtualServersForPolicy(virtualServers, "ns-1", "test-policy")
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("findVirtualServersForPolicy() returned %v but expected %v", result, expected)
	}
}

func TestIsPolicyIsReferenced(t *testing.T) {
	tests := []struct {
		policies          []conf_v1.PolicyReference
		resourceNamespace string
		policyNamespace   string
		policyName        string
		expected          bool
		msg               string
	}{
		{
			policies: []conf_v1.PolicyReference{
				{
					Name: "test-policy",
				},
			},
			resourceNamespace: "ns-1",
			policyNamespace:   "ns-1",
			policyName:        "test-policy",
			expected:          true,
			msg:               "reference with implicit namespace",
		},
		{
			policies: []conf_v1.PolicyReference{
				{
					Name:      "test-policy",
					Namespace: "ns-1",
				},
			},
			resourceNamespace: "ns-1",
			policyNamespace:   "ns-1",
			policyName:        "test-policy",
			expected:          true,
			msg:               "reference with explicit namespace",
		},
		{
			policies: []conf_v1.PolicyReference{
				{
					Name: "test-policy",
				},
			},
			resourceNamespace: "ns-2",
			policyNamespace:   "ns-1",
			policyName:        "test-policy",
			expected:          false,
			msg:               "wrong namespace with implicit namespace",
		},
		{
			policies: []conf_v1.PolicyReference{
				{
					Name:      "test-policy",
					Namespace: "ns-2",
				},
			},
			resourceNamespace: "ns-2",
			policyNamespace:   "ns-1",
			policyName:        "test-policy",
			expected:          false,
			msg:               "wrong namespace with explicit namespace",
		},
	}

	for _, test := range tests {
		result := isPolicyReferenced(test.policies, test.resourceNamespace, test.policyNamespace, test.policyName)
		if result != test.expected {
			t.Errorf("isPolicyReferenced() returned %v but expected %v for the case of %s", result,
				test.expected, test.msg)
		}
	}
}

func TestFindVirtualServerRoutesForPolicy(t *testing.T) {
	vsr1 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vsr-1",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerRouteSpec{
			Subroutes: []conf_v1.Route{
				{
					Policies: []conf_v1.PolicyReference{
						{
							Name:      "test-policy",
							Namespace: "ns-1",
						},
					},
				},
			},
		},
	}
	vsr2 := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vsr-2",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerRouteSpec{
			Subroutes: []conf_v1.Route{
				{
					Policies: []conf_v1.PolicyReference{
						{
							Name:      "some-policy",
							Namespace: "ns-1",
						},
					},
				},
			},
		},
	}

	virtualServerRoutes := []*conf_v1.VirtualServerRoute{&vsr1, &vsr2}

	expected := []*conf_v1.VirtualServerRoute{&vsr1}

	result := findVirtualServerRoutesForPolicy(virtualServerRoutes, "ns-1", "test-policy")
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("findVirtualServerRoutesForPolicy() returned %v but expected %v", result, expected)
	}
}

func TestFindVirtualServersForVirtualServerRoute(t *testing.T) {
	vs1 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-1",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			Routes: []conf_v1.Route{
				{
					Path:  "/",
					Route: "default/test",
				},
			},
		},
	}
	vs2 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-2",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			Routes: []conf_v1.Route{
				{
					Path:  "/",
					Route: "some-ns/test",
				},
			},
		},
	}
	vs3 := conf_v1.VirtualServer{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "vs-3",
			Namespace: "ns-1",
		},
		Spec: conf_v1.VirtualServerSpec{
			Routes: []conf_v1.Route{
				{
					Path:  "/",
					Route: "default/test",
				},
			},
		},
	}

	vsr := conf_v1.VirtualServerRoute{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
	}

	virtualServers := []*conf_v1.VirtualServer{&vs1, &vs2, &vs3}

	expected := []*conf_v1.VirtualServer{&vs1, &vs3}

	result := findVirtualServersForVirtualServerRoute(virtualServers, &vsr)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("findVirtualServersForVirtualServerRoute returned %v but expected %v", result, expected)
	}
}

func TestFormatWarningsMessages(t *testing.T) {
	warnings := []string{"Test warning", "Test warning 2"}

	expected := "Test warning; Test warning 2"
	result := formatWarningMessages(warnings)

	if result != expected {
		t.Errorf("formatWarningMessages(%v) returned %v but expected %v", warnings, result, expected)
	}
}

func TestGetEndpointsBySubselectedPods(t *testing.T) {
	boolPointer := func(b bool) *bool { return &b }
	tests := []struct {
		desc        string
		targetPort  int32
		svcEps      v1.Endpoints
		expectedEps []podEndpoint
	}{
		{
			desc:       "find one endpoint",
			targetPort: 80,
			expectedEps: []podEndpoint{
				{
					Address: "1.2.3.4:80",
					MeshPodOwner: configs.MeshPodOwner{
						OwnerType: "deployment",
						OwnerName: "deploy-1",
					},
				},
			},
		},
		{
			desc:        "targetPort mismatch",
			targetPort:  21,
			expectedEps: nil,
		},
	}

	pods := []*v1.Pod{
		{
			ObjectMeta: meta_v1.ObjectMeta{
				OwnerReferences: []meta_v1.OwnerReference{
					{
						Kind:       "Deployment",
						Name:       "deploy-1",
						Controller: boolPointer(true),
					},
				},
			},
			Status: v1.PodStatus{
				PodIP: "1.2.3.4",
			},
		},
	}

	svcEps := v1.Endpoints{
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP:       "1.2.3.4",
						Hostname: "asdf.com",
					},
				},
				Ports: []v1.EndpointPort{
					{
						Port: 80,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotEndps := getEndpointsBySubselectedPods(test.targetPort, pods, svcEps)
			if !reflect.DeepEqual(gotEndps, test.expectedEps) {
				t.Errorf("getEndpointsBySubselectedPods() = %v, want %v", gotEndps, test.expectedEps)
			}
		})
	}
}

func TestGetStatusFromEventTitle(t *testing.T) {
	tests := []struct {
		eventTitle string
		expected   string
	}{
		{
			eventTitle: "",
			expected:   "",
		},
		{
			eventTitle: "AddedOrUpdatedWithError",
			expected:   "Invalid",
		},
		{
			eventTitle: "Rejected",
			expected:   "Invalid",
		},
		{
			eventTitle: "NoVirtualServersFound",
			expected:   "Invalid",
		},
		{
			eventTitle: "Missing Secret",
			expected:   "Invalid",
		},
		{
			eventTitle: "UpdatedWithError",
			expected:   "Invalid",
		},
		{
			eventTitle: "AddedOrUpdatedWithWarning",
			expected:   "Warning",
		},
		{
			eventTitle: "UpdatedWithWarning",
			expected:   "Warning",
		},
		{
			eventTitle: "AddedOrUpdated",
			expected:   "Valid",
		},
		{
			eventTitle: "Updated",
			expected:   "Valid",
		},
		{
			eventTitle: "New State",
			expected:   "",
		},
	}

	for _, test := range tests {
		result := getStatusFromEventTitle(test.eventTitle)
		if result != test.expected {
			t.Errorf("getStatusFromEventTitle(%v) returned %v but expected %v", test.eventTitle, result, test.expected)
		}
	}
}

func TestGetPolicies(t *testing.T) {
	validPolicy := &conf_v1alpha1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-policy",
			Namespace: "default",
		},
		Spec: conf_v1alpha1.PolicySpec{
			AccessControl: &conf_v1alpha1.AccessControl{
				Allow: []string{"127.0.0.1"},
			},
		},
	}

	invalidPolicy := &conf_v1alpha1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-policy",
			Namespace: "default",
		},
		Spec: conf_v1alpha1.PolicySpec{},
	}

	lbc := LoadBalancerController{
		isNginxPlus: true,
		policyLister: &cache.FakeCustomStore{
			GetByKeyFunc: func(key string) (item interface{}, exists bool, err error) {
				switch key {
				case "default/valid-policy":
					return validPolicy, true, nil
				case "default/invalid-policy":
					return invalidPolicy, true, nil
				case "nginx-ingress/valid-policy":
					return nil, false, nil
				default:
					return nil, false, errors.New("GetByKey error")
				}
			},
		},
	}

	policyRefs := []conf_v1.PolicyReference{
		{
			Name: "valid-policy",
			// Namespace is implicit here
		},
		{
			Name:      "invalid-policy",
			Namespace: "default",
		},
		{
			Name:      "valid-policy", // doesn't exist
			Namespace: "nginx-ingress",
		},
		{
			Name:      "some-policy", // will make lister return error
			Namespace: "nginx-ingress",
		},
	}

	expectedPolicies := []*conf_v1alpha1.Policy{validPolicy}
	expectedErrors := []error{
		errors.New("Policy default/invalid-policy is invalid: spec: Invalid value: \"\": must specify exactly one of: `accessControl`, `rateLimit`, `ingressMTLS`, `jwt`"),
		errors.New("Policy nginx-ingress/valid-policy doesn't exist"),
		errors.New("Failed to get policy nginx-ingress/some-policy: GetByKey error"),
	}

	result, errors := lbc.getPolicies(policyRefs, "default")
	if !reflect.DeepEqual(result, expectedPolicies) {
		t.Errorf("lbc.getPolicies() returned \n%v but \nexpected %v", result, expectedPolicies)
	}
	if !reflect.DeepEqual(errors, expectedErrors) {
		t.Errorf("lbc.getPolicies() returned \n%v but expected \n%v", errors, expectedErrors)
	}
}

func TestCreatePolicyMap(t *testing.T) {
	policies := []*conf_v1alpha1.Policy{
		{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "policy-1",
				Namespace: "default",
			},
		},
		{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "policy-2",
				Namespace: "default",
			},
		},
		{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "policy-1",
				Namespace: "default",
			},
		},
		{
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "policy-1",
				Namespace: "nginx-ingress",
			},
		},
	}

	expected := map[string]*conf_v1alpha1.Policy{
		"default/policy-1": {
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "policy-1",
				Namespace: "default",
			},
		},
		"default/policy-2": {
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "policy-2",
				Namespace: "default",
			},
		},
		"nginx-ingress/policy-1": {
			ObjectMeta: meta_v1.ObjectMeta{
				Name:      "policy-1",
				Namespace: "nginx-ingress",
			},
		},
	}

	result := createPolicyMap(policies)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("createPolicyMap() returned \n%s but expected \n%s", policyMapToString(result), policyMapToString(expected))
	}
}

func TestGetPodOwnerTypeAndName(t *testing.T) {
	tests := []struct {
		desc    string
		expType string
		expName string
		pod     *v1.Pod
	}{
		{
			desc:    "deployment",
			expType: "deployment",
			expName: "deploy-name",
			pod:     &v1.Pod{ObjectMeta: createTestObjMeta("Deployment", "deploy-name", true)},
		},
		{
			desc:    "stateful set",
			expType: "statefulset",
			expName: "statefulset-name",
			pod:     &v1.Pod{ObjectMeta: createTestObjMeta("StatefulSet", "statefulset-name", true)},
		},
		{
			desc:    "daemon set",
			expType: "daemonset",
			expName: "daemonset-name",
			pod:     &v1.Pod{ObjectMeta: createTestObjMeta("DaemonSet", "daemonset-name", true)},
		},
		{
			desc:    "replica set with no pod hash",
			expType: "deployment",
			expName: "replicaset-name",
			pod:     &v1.Pod{ObjectMeta: createTestObjMeta("ReplicaSet", "replicaset-name", false)},
		},
		{
			desc:    "replica set with pod hash",
			expType: "deployment",
			expName: "replicaset-name",
			pod: &v1.Pod{
				ObjectMeta: createTestObjMeta("ReplicaSet", "replicaset-name-67c6f7c5fd", true),
			},
		},
		{
			desc:    "nil controller should use default values",
			expType: "deployment",
			expName: "deploy-name",
			pod: &v1.Pod{
				ObjectMeta: meta_v1.ObjectMeta{
					OwnerReferences: []meta_v1.OwnerReference{
						{
							Name:       "deploy-name",
							Controller: nil,
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			actualType, actualName := getPodOwnerTypeAndName(test.pod)
			if actualType != test.expType {
				t.Errorf("getPodOwnerTypeAndName() returned %s for owner type but expected %s", actualType, test.expType)
			}
			if actualName != test.expName {
				t.Errorf("getPodOwnerTypeAndName() returned %s for owner name but expected %s", actualName, test.expName)
			}
		})
	}
}

func createTestObjMeta(kind, name string, podHashLabel bool) meta_v1.ObjectMeta {
	controller := true
	meta := meta_v1.ObjectMeta{
		OwnerReferences: []meta_v1.OwnerReference{
			{
				Kind:       kind,
				Name:       name,
				Controller: &controller,
			},
		},
	}
	if podHashLabel {
		meta.Labels = map[string]string{
			"pod-template-hash": "67c6f7c5fd",
		}
	}
	return meta
}

func policyMapToString(policies map[string]*conf_v1alpha1.Policy) string {
	var keys []string
	for k := range policies {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder

	b.WriteString("[ ")
	for _, k := range keys {
		fmt.Fprintf(&b, "%q: '%s/%s', ", k, policies[k].Namespace, policies[k].Name)
	}
	b.WriteString("]")

	return b.String()
}

func TestRemoveDuplicateVirtualServers(t *testing.T) {
	tests := []struct {
		virtualServers []*conf_v1.VirtualServer
		expected       []*conf_v1.VirtualServer
	}{
		{
			[]*conf_v1.VirtualServer{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-1",
						Namespace: "ns-1",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-2",
						Namespace: "ns-1",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-2",
						Namespace: "ns-1",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-3",
						Namespace: "ns-1",
					},
				},
			},
			[]*conf_v1.VirtualServer{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-1",
						Namespace: "ns-1",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-2",
						Namespace: "ns-1",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-3",
						Namespace: "ns-1",
					},
				},
			},
		},
		{
			[]*conf_v1.VirtualServer{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-3",
						Namespace: "ns-2",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-3",
						Namespace: "ns-1",
					},
				},
			},
			[]*conf_v1.VirtualServer{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-3",
						Namespace: "ns-2",
					},
				},
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "vs-3",
						Namespace: "ns-1",
					},
				},
			},
		},
	}
	for _, test := range tests {
		result := removeDuplicateVirtualServers(test.virtualServers)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("removeDuplicateVirtualServers() returned \n%v but expected \n%v", result, test.expected)
		}
	}
}

func TestFindPoliciesForSecret(t *testing.T) {
	jwtPol1 := &conf_v1alpha1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "jwt-policy",
			Namespace: "default",
		},
		Spec: conf_v1alpha1.PolicySpec{
			JWTAuth: &conf_v1alpha1.JWTAuth{
				Secret: "jwk-secret",
			},
		},
	}

	jwtPol2 := &conf_v1alpha1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "jwt-policy",
			Namespace: "ns-1",
		},
		Spec: conf_v1alpha1.PolicySpec{
			JWTAuth: &conf_v1alpha1.JWTAuth{
				Secret: "jwk-secret",
			},
		},
	}

	ingTLSPol := &conf_v1alpha1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "ingress-tmls-policy",
			Namespace: "default",
		},
		Spec: conf_v1alpha1.PolicySpec{
			IngressMTLS: &conf_v1alpha1.IngressMTLS{
				ClientCertSecret: "ingress-mtls-secret",
			},
		},
	}

	tests := []struct {
		policies        []*conf_v1alpha1.Policy
		secretNamespace string
		secretName      string
		expected        []*conf_v1alpha1.Policy
		msg             string
	}{
		{
			policies:        []*conf_v1alpha1.Policy{jwtPol1},
			secretNamespace: "default",
			secretName:      "jwk-secret",
			expected:        []*v1alpha1.Policy{jwtPol1},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1alpha1.Policy{jwtPol2},
			secretNamespace: "default",
			secretName:      "jwk-secret",
			expected:        nil,
			msg:             "Ignore policies in other namespaces",
		},
		{
			policies:        []*conf_v1alpha1.Policy{jwtPol1, jwtPol2},
			secretNamespace: "default",
			secretName:      "jwk-secret",
			expected:        []*v1alpha1.Policy{jwtPol1},
			msg:             "Find policy in default ns, ignore other",
		},
		{
			policies:        []*conf_v1alpha1.Policy{ingTLSPol},
			secretNamespace: "default",
			secretName:      "ingress-mtls-secret",
			expected:        []*v1alpha1.Policy{ingTLSPol},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1alpha1.Policy{jwtPol1, ingTLSPol},
			secretNamespace: "default",
			secretName:      "ingress-mtls-secret",
			expected:        []*v1alpha1.Policy{ingTLSPol},
			msg:             "Find policy in default ns, ignore other types",
		},
	}
	for _, test := range tests {
		result := findPoliciesForSecret(test.policies, test.secretNamespace, test.secretName)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("findPoliciesForSecret() returned \n%v but expected \n%v for the case of %s", result, test.expected, test.msg)
		}
	}
}

func TestAddJWTSecrets(t *testing.T) {
	validSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-jwk-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{"jwk": nil},
	}

	invalidSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-jwk-secret",
			Namespace: "default",
		},
		Data: nil,
	}

	tests := []struct {
		policies        []*conf_v1alpha1.Policy
		expectedJWTKeys map[string]*v1.Secret
		wantErr         bool
		msg             string
	}{
		{
			policies: []*conf_v1alpha1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1alpha1.PolicySpec{
						JWTAuth: &conf_v1alpha1.JWTAuth{
							Secret: "valid-jwk-secret",
							Realm:  "My API",
						},
					},
				},
			},
			expectedJWTKeys: map[string]*v1.Secret{
				"default/valid-jwk-secret": validSecret,
			},
			wantErr: false,
			msg:     "test getting valid secret",
		},
		{
			policies:        []*conf_v1alpha1.Policy{},
			expectedJWTKeys: map[string]*v1.Secret{},
			wantErr:         false,
			msg:             "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1alpha1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1alpha1.PolicySpec{
						AccessControl: &conf_v1alpha1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedJWTKeys: map[string]*v1.Secret{},
			wantErr:         false,
			msg:             "test getting valid secret with wrong policy",
		},
		{
			policies: []*conf_v1alpha1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1alpha1.PolicySpec{
						JWTAuth: &conf_v1alpha1.JWTAuth{
							Secret: "non-existing-jwk-secret",
							Realm:  "My API",
						},
					},
				},
			},
			expectedJWTKeys: map[string]*v1.Secret{},
			wantErr:         true,
			msg:             "test getting secret that does not exist",
		},
		{
			policies: []*conf_v1alpha1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1alpha1.PolicySpec{
						JWTAuth: &conf_v1alpha1.JWTAuth{
							Secret: "invalid-jwk-secret",
							Realm:  "My API",
						},
					},
				},
			},
			expectedJWTKeys: map[string]*v1.Secret{},
			wantErr:         true,
			msg:             "test getting invalid secret",
		},
	}

	for _, test := range tests {
		lbc := LoadBalancerController{
			secretLister: storeToSecretLister{
				&cache.FakeCustomStore{
					GetByKeyFunc: func(key string) (item interface{}, exists bool, err error) {
						switch key {
						case "default/valid-jwk-secret":
							return validSecret, true, nil
						case "default/invalid-jwk-secret":
							return invalidSecret, true, errors.New("secret is missing jwk key in data")
						default:
							return nil, false, errors.New("GetByKey error")
						}
					},
				},
			},
		}

		jwtKeys := make(map[string]*v1.Secret)

		err := lbc.addJWTSecrets(test.policies, jwtKeys)
		if (err != nil) != test.wantErr {
			t.Errorf("addJWTSecrets() returned %v, for the case of %v", err, test.msg)
		}

		if !reflect.DeepEqual(jwtKeys, test.expectedJWTKeys) {
			t.Errorf("addJWTSecrets() returned \n%+v but expected \n%+v", jwtKeys, test.expectedJWTKeys)
		}

	}
}

func TestGetIngressMTLSSecret(t *testing.T) {
	validSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-ingress-mtls-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{"ca.crt": nil},
	}

	invalidSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-ingress-mtls-secret",
			Namespace: "default",
		},
		Data: nil,
	}

	tests := []struct {
		policies                  []*conf_v1alpha1.Policy
		expectedIngressMTLSSecret *v1.Secret
		wantErr                   bool
		msg                       string
	}{
		{
			policies: []*conf_v1alpha1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1alpha1.PolicySpec{
						IngressMTLS: &conf_v1alpha1.IngressMTLS{
							ClientCertSecret: "valid-ingress-mtls-secret",
						},
					},
				},
			},
			expectedIngressMTLSSecret: validSecret,
			wantErr:                   false,
			msg:                       "test getting valid secret",
		},
		{
			policies:                  []*conf_v1alpha1.Policy{},
			expectedIngressMTLSSecret: nil,
			wantErr:                   false,
			msg:                       "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1alpha1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1alpha1.PolicySpec{
						AccessControl: &conf_v1alpha1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedIngressMTLSSecret: nil,
			wantErr:                   false,
			msg:                       "test getting valid secret with wrong policy",
		},
		{
			policies: []*conf_v1alpha1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1alpha1.PolicySpec{
						IngressMTLS: &conf_v1alpha1.IngressMTLS{
							ClientCertSecret: "non-existing-ingress-mtls-secret",
						},
					},
				},
			},
			expectedIngressMTLSSecret: nil,
			wantErr:                   true,
			msg:                       "test getting secret that does not exist",
		},
		{
			policies: []*conf_v1alpha1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1alpha1.PolicySpec{
						IngressMTLS: &conf_v1alpha1.IngressMTLS{
							ClientCertSecret: "invalid-ingress-mtls-secret",
						},
					},
				},
			},
			expectedIngressMTLSSecret: nil,
			wantErr:                   true,
			msg:                       "test getting invalid secret",
		},
	}

	for _, test := range tests {
		lbc := LoadBalancerController{
			secretLister: storeToSecretLister{
				&cache.FakeCustomStore{
					GetByKeyFunc: func(key string) (item interface{}, exists bool, err error) {
						switch key {
						case "default/valid-ingress-mtls-secret":
							return validSecret, true, nil
						case "default/invalid-ingress-mtls-secret":
							return invalidSecret, true, errors.New("secret is missing ingress-mtls key in data")
						default:
							return nil, false, errors.New("GetByKey error")
						}
					},
				},
			},
		}

		secret, err := lbc.getIngressMTLSSecret(test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("getIngressMTLSSecret() returned %v, for the case of %v", err, test.msg)
		}
		if !reflect.DeepEqual(secret, test.expectedIngressMTLSSecret) {
			t.Errorf("getIngressMTLSSecret() returned \n%+v but expected \n%+v", secret, test.expectedIngressMTLSSecret)
		}

	}
}

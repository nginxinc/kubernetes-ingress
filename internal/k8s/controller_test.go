package k8s

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/appprotect"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	conf_v1alpha1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1alpha1"
	api_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	"k8s.io/client-go/tools/cache"
)

func TestHasCorrectIngressClass(t *testing.T) {
	ingressClass := "ing-ctrl"
	incorrectIngressClass := "gce"
	emptyClass := ""

	testsWithoutIngressClassOnly := []struct {
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

	testsWithIngressClassOnly := []struct {
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

func deepCopyWithIngressClass(obj interface{}, class string) interface{} {
	switch obj := obj.(type) {
	case *conf_v1.VirtualServer:
		objCopy := obj.DeepCopy()
		objCopy.Spec.IngressClass = class
		return objCopy
	case *conf_v1.VirtualServerRoute:
		objCopy := obj.DeepCopy()
		objCopy.Spec.IngressClass = class
		return objCopy
	case *conf_v1alpha1.TransportServer:
		objCopy := obj.DeepCopy()
		objCopy.Spec.IngressClass = class
		return objCopy
	default:
		panic(fmt.Sprintf("unknown type %T", obj))
	}
}

func TestIngressClassForCustomResources(t *testing.T) {
	ctrl := &LoadBalancerController{
		ingressClass:        "nginx",
		useIngressClassOnly: false,
	}

	ctrlWithUseICOnly := &LoadBalancerController{
		ingressClass:        "nginx",
		useIngressClassOnly: true,
	}

	tests := []struct {
		lbc             *LoadBalancerController
		objIngressClass string
		expected        bool
		msg             string
	}{
		{
			lbc:             ctrl,
			objIngressClass: "nginx",
			expected:        true,
			msg:             "Ingress Controller handles a resource that matches its IngressClass",
		},
		{
			lbc:             ctrlWithUseICOnly,
			objIngressClass: "nginx",
			expected:        true,
			msg:             "Ingress Controller with useIngressClassOnly handles a resource that matches its IngressClass",
		},
		{
			lbc:             ctrl,
			objIngressClass: "",
			expected:        true,
			msg:             "Ingress Controller handles a resource with an empty IngressClass",
		},
		{
			lbc:             ctrlWithUseICOnly,
			objIngressClass: "",
			expected:        true,
			msg:             "Ingress Controller with useIngressClassOnly handles a resource with an empty IngressClass",
		},
		{
			lbc:             ctrl,
			objIngressClass: "gce",
			expected:        false,
			msg:             "Ingress Controller doesn't handle a resource that doesn't match its IngressClass",
		},
		{
			lbc:             ctrlWithUseICOnly,
			objIngressClass: "gce",
			expected:        false,
			msg:             "Ingress Controller with useIngressClassOnly doesn't handle a resource that doesn't match its IngressClass",
		},
	}

	resources := []interface{}{
		&conf_v1.VirtualServer{},
		&conf_v1.VirtualServerRoute{},
		&conf_v1alpha1.TransportServer{},
	}

	for _, r := range resources {
		for _, test := range tests {
			obj := deepCopyWithIngressClass(r, test.objIngressClass)

			result := test.lbc.HasCorrectIngressClass(obj)
			if result != test.expected {
				t.Errorf("HasCorrectIngressClass() returned %v but expected %v for the case of %q for %T", result, test.expected, test.msg, obj)
			}
		}
	}
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
	cnf := configs.NewConfigurator(&nginx.LocalManager{}, &configs.StaticConfigParams{}, &configs.ConfigParams{}, &version1.TemplateExecutor{}, &version2.TemplateExecutor{}, false, false, nil, false, nil, false)
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
	validPolicy := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			AccessControl: &conf_v1.AccessControl{
				Allow: []string{"127.0.0.1"},
			},
		},
	}

	validPolicyIngressClass := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-policy-ingress-class",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			IngressClass: "test-class",
			AccessControl: &conf_v1.AccessControl{
				Allow: []string{"127.0.0.1"},
			},
		},
	}

	invalidPolicy := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{},
	}

	lbc := LoadBalancerController{
		isNginxPlus: true,
		policyLister: &cache.FakeCustomStore{
			GetByKeyFunc: func(key string) (item interface{}, exists bool, err error) {
				switch key {
				case "default/valid-policy":
					return validPolicy, true, nil
				case "default/valid-policy-ingress-class":
					return validPolicyIngressClass, true, nil
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
		{
			Name:      "valid-policy-ingress-class",
			Namespace: "default",
		},
	}

	expectedPolicies := []*conf_v1.Policy{validPolicy}
	expectedErrors := []error{
		errors.New("Policy default/invalid-policy is invalid: spec: Invalid value: \"\": must specify exactly one of: `accessControl`, `rateLimit`, `ingressMTLS`, `egressMTLS`, `jwt`, `oidc`, `waf`"),
		errors.New("Policy nginx-ingress/valid-policy doesn't exist"),
		errors.New("Failed to get policy nginx-ingress/some-policy: GetByKey error"),
		errors.New("referenced policy default/valid-policy-ingress-class has incorrect ingress class: test-class (controller ingress class: )"),
	}

	result, errors := lbc.getPolicies(policyRefs, "default")
	if !reflect.DeepEqual(result, expectedPolicies) {
		t.Errorf("lbc.getPolicies() returned \n%v but \nexpected %v", result, expectedPolicies)
	}
	if diff := cmp.Diff(expectedErrors, errors, cmp.Comparer(errorComparer)); diff != "" {
		t.Errorf("lbc.getPolicies() mismatch (-want +got):\n%s", diff)
	}
}

func TestCreatePolicyMap(t *testing.T) {
	policies := []*conf_v1.Policy{
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

	expected := map[string]*conf_v1.Policy{
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

func policyMapToString(policies map[string]*conf_v1.Policy) string {
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

type testResource struct {
	keyWithKind string
}

func (*testResource) GetObjectMeta() *meta_v1.ObjectMeta {
	return nil
}

func (t *testResource) GetKeyWithKind() string {
	return t.keyWithKind
}

func (*testResource) AcquireHost(string) {
}

func (*testResource) ReleaseHost(string) {
}

func (*testResource) Wins(Resource) bool {
	return false
}

func (*testResource) IsSame(Resource) bool {
	return false
}

func (*testResource) AddWarning(string) {
}

func (*testResource) IsEqual(Resource) bool {
	return false
}

func (t *testResource) String() string {
	return t.keyWithKind
}

func TestRemoveDuplicateResources(t *testing.T) {
	tests := []struct {
		resources []Resource
		expected  []Resource
	}{
		{
			resources: []Resource{
				&testResource{keyWithKind: "VirtualServer/ns-1/vs-1"},
				&testResource{keyWithKind: "VirtualServer/ns-1/vs-2"},
				&testResource{keyWithKind: "VirtualServer/ns-1/vs-2"},
				&testResource{keyWithKind: "VirtualServer/ns-1/vs-3"},
			},
			expected: []Resource{
				&testResource{keyWithKind: "VirtualServer/ns-1/vs-1"},
				&testResource{keyWithKind: "VirtualServer/ns-1/vs-2"},
				&testResource{keyWithKind: "VirtualServer/ns-1/vs-3"},
			},
		},
		{
			resources: []Resource{
				&testResource{keyWithKind: "VirtualServer/ns-2/vs-3"},
				&testResource{keyWithKind: "VirtualServer/ns-1/vs-3"},
			},
			expected: []Resource{
				&testResource{keyWithKind: "VirtualServer/ns-2/vs-3"},
				&testResource{keyWithKind: "VirtualServer/ns-1/vs-3"},
			},
		},
	}

	for _, test := range tests {
		result := removeDuplicateResources(test.resources)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("removeDuplicateResources() returned \n%v but expected \n%v", result, test.expected)
		}
	}
}

func TestFindPoliciesForSecret(t *testing.T) {
	jwtPol1 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "jwt-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			JWTAuth: &conf_v1.JWTAuth{
				Secret: "jwk-secret",
			},
		},
	}

	jwtPol2 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "jwt-policy",
			Namespace: "ns-1",
		},
		Spec: conf_v1.PolicySpec{
			JWTAuth: &conf_v1.JWTAuth{
				Secret: "jwk-secret",
			},
		},
	}

	ingTLSPol := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "ingress-mtls-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			IngressMTLS: &conf_v1.IngressMTLS{
				ClientCertSecret: "ingress-mtls-secret",
			},
		},
	}
	egTLSPol := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "egress-mtls-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			EgressMTLS: &conf_v1.EgressMTLS{
				TLSSecret: "egress-mtls-secret",
			},
		},
	}
	egTLSPol2 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "egress-trusted-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			EgressMTLS: &conf_v1.EgressMTLS{
				TrustedCertSecret: "egress-trusted-secret",
			},
		},
	}
	oidcPol := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "oidc-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			OIDC: &conf_v1.OIDC{
				ClientSecret: "oidc-secret",
			},
		},
	}

	tests := []struct {
		policies        []*conf_v1.Policy
		secretNamespace string
		secretName      string
		expected        []*conf_v1.Policy
		msg             string
	}{
		{
			policies:        []*conf_v1.Policy{jwtPol1},
			secretNamespace: "default",
			secretName:      "jwk-secret",
			expected:        []*conf_v1.Policy{jwtPol1},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{jwtPol2},
			secretNamespace: "default",
			secretName:      "jwk-secret",
			expected:        nil,
			msg:             "Ignore policies in other namespaces",
		},
		{
			policies:        []*conf_v1.Policy{jwtPol1, jwtPol2},
			secretNamespace: "default",
			secretName:      "jwk-secret",
			expected:        []*conf_v1.Policy{jwtPol1},
			msg:             "Find policy in default ns, ignore other",
		},
		{
			policies:        []*conf_v1.Policy{ingTLSPol},
			secretNamespace: "default",
			secretName:      "ingress-mtls-secret",
			expected:        []*conf_v1.Policy{ingTLSPol},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{jwtPol1, ingTLSPol},
			secretNamespace: "default",
			secretName:      "ingress-mtls-secret",
			expected:        []*conf_v1.Policy{ingTLSPol},
			msg:             "Find policy in default ns, ignore other types",
		},
		{
			policies:        []*conf_v1.Policy{egTLSPol},
			secretNamespace: "default",
			secretName:      "egress-mtls-secret",
			expected:        []*conf_v1.Policy{egTLSPol},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{jwtPol1, egTLSPol},
			secretNamespace: "default",
			secretName:      "egress-mtls-secret",
			expected:        []*conf_v1.Policy{egTLSPol},
			msg:             "Find policy in default ns, ignore other types",
		},
		{
			policies:        []*conf_v1.Policy{egTLSPol2},
			secretNamespace: "default",
			secretName:      "egress-trusted-secret",
			expected:        []*conf_v1.Policy{egTLSPol2},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{egTLSPol, egTLSPol2},
			secretNamespace: "default",
			secretName:      "egress-trusted-secret",
			expected:        []*conf_v1.Policy{egTLSPol2},
			msg:             "Find policy in default ns, ignore other types",
		},
		{
			policies:        []*conf_v1.Policy{oidcPol},
			secretNamespace: "default",
			secretName:      "oidc-secret",
			expected:        []*conf_v1.Policy{oidcPol},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{ingTLSPol, oidcPol},
			secretNamespace: "default",
			secretName:      "oidc-secret",
			expected:        []*conf_v1.Policy{oidcPol},
			msg:             "Find policy in default ns, ignore other types",
		},
	}
	for _, test := range tests {
		result := findPoliciesForSecret(test.policies, test.secretNamespace, test.secretName)
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("findPoliciesForSecret() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func errorComparer(e1, e2 error) bool {
	if e1 == nil || e2 == nil {
		return errors.Is(e1, e2)
	}

	return e1.Error() == e2.Error()
}

func TestAddJWTSecrets(t *testing.T) {
	invalidErr := errors.New("invalid")
	validJWKSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-jwk-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeJWK,
	}
	invalidJWKSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-jwk-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeJWK,
	}

	tests := []struct {
		policies           []*conf_v1.Policy
		expectedSecretRefs map[string]*secrets.SecretReference
		wantErr            bool
		msg                string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Secret: "valid-jwk-secret",
							Realm:  "My API",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-jwk-secret": {
					Secret: validJWKSecret,
					Path:   "/etc/nginx/secrets/default-valid-jwk-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid secret",
		},
		{
			policies:           []*conf_v1.Policy{},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting invalid secret with wrong policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Secret: "invalid-jwk-secret",
							Realm:  "My API",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-jwk-secret": {
					Secret: invalidJWKSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid secret",
		},
	}

	lbc := LoadBalancerController{
		secretStore: secrets.NewFakeSecretsStore(map[string]*secrets.SecretReference{
			"default/valid-jwk-secret": {
				Secret: validJWKSecret,
				Path:   "/etc/nginx/secrets/default-valid-jwk-secret",
			},
			"default/invalid-jwk-secret": {
				Secret: invalidJWKSecret,
				Error:  invalidErr,
			},
		}),
	}

	for _, test := range tests {
		result := make(map[string]*secrets.SecretReference)

		err := lbc.addJWTSecretRefs(result, test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("addJWTSecretRefs() returned %v, for the case of %v", err, test.msg)
		}

		if diff := cmp.Diff(test.expectedSecretRefs, result, cmp.Comparer(errorComparer)); diff != "" {
			t.Errorf("addJWTSecretRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddIngressMTLSSecret(t *testing.T) {
	invalidErr := errors.New("invalid")
	validSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-ingress-mtls-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeCA,
	}
	invalidSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-ingress-mtls-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeCA,
	}

	tests := []struct {
		policies           []*conf_v1.Policy
		expectedSecretRefs map[string]*secrets.SecretReference
		wantErr            bool
		msg                string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "valid-ingress-mtls-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-ingress-mtls-secret": {
					Secret: validSecret,
					Path:   "/etc/nginx/secrets/default-valid-ingress-mtls-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid secret",
		},
		{
			policies:           []*conf_v1.Policy{},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with wrong policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "invalid-ingress-mtls-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-ingress-mtls-secret": {
					Secret: invalidSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid secret",
		},
	}

	lbc := LoadBalancerController{
		secretStore: secrets.NewFakeSecretsStore(map[string]*secrets.SecretReference{
			"default/valid-ingress-mtls-secret": {
				Secret: validSecret,
				Path:   "/etc/nginx/secrets/default-valid-ingress-mtls-secret",
			},
			"default/invalid-ingress-mtls-secret": {
				Secret: invalidSecret,
				Error:  invalidErr,
			},
		}),
	}

	for _, test := range tests {
		result := make(map[string]*secrets.SecretReference)

		err := lbc.addIngressMTLSSecretRefs(result, test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("addIngressMTLSSecretRefs() returned %v, for the case of %v", err, test.msg)
		}

		if diff := cmp.Diff(test.expectedSecretRefs, result, cmp.Comparer(errorComparer)); diff != "" {
			t.Errorf("addIngressMTLSSecretRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddEgressMTLSSecrets(t *testing.T) {
	invalidErr := errors.New("invalid")
	validMTLSSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-egress-mtls-secret",
			Namespace: "default",
		},
		Type: api_v1.SecretTypeTLS,
	}
	validTrustedSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-egress-trusted-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeCA,
	}
	invalidMTLSSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-egress-mtls-secret",
			Namespace: "default",
		},
		Type: api_v1.SecretTypeTLS,
	}
	invalidTrustedSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-egress-trusted-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeCA,
	}

	tests := []struct {
		policies           []*conf_v1.Policy
		expectedSecretRefs map[string]*secrets.SecretReference
		wantErr            bool
		msg                string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret: "valid-egress-mtls-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-egress-mtls-secret": {
					Secret: validMTLSSecret,
					Path:   "/etc/nginx/secrets/default-valid-egress-mtls-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid TLS secret",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-egress-trusted-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TrustedCertSecret: "valid-egress-trusted-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-egress-trusted-secret": {
					Secret: validTrustedSecret,
					Path:   "/etc/nginx/secrets/default-valid-egress-trusted-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid TrustedCA secret",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret:         "valid-egress-mtls-secret",
							TrustedCertSecret: "valid-egress-trusted-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-egress-mtls-secret": {
					Secret: validMTLSSecret,
					Path:   "/etc/nginx/secrets/default-valid-egress-mtls-secret",
				},
				"default/valid-egress-trusted-secret": {
					Secret: validTrustedSecret,
					Path:   "/etc/nginx/secrets/default-valid-egress-trusted-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid secrets",
		},
		{
			policies:           []*conf_v1.Policy{},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with wrong policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret: "invalid-egress-mtls-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-egress-mtls-secret": {
					Secret: invalidMTLSSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid TLS secret",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TrustedCertSecret: "invalid-egress-trusted-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-egress-trusted-secret": {
					Secret: invalidTrustedSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid TrustedCA secret",
		},
	}

	lbc := LoadBalancerController{
		secretStore: secrets.NewFakeSecretsStore(map[string]*secrets.SecretReference{
			"default/valid-egress-mtls-secret": {
				Secret: validMTLSSecret,
				Path:   "/etc/nginx/secrets/default-valid-egress-mtls-secret",
			},
			"default/valid-egress-trusted-secret": {
				Secret: validTrustedSecret,
				Path:   "/etc/nginx/secrets/default-valid-egress-trusted-secret",
			},
			"default/invalid-egress-mtls-secret": {
				Secret: invalidMTLSSecret,
				Error:  invalidErr,
			},
			"default/invalid-egress-trusted-secret": {
				Secret: invalidTrustedSecret,
				Error:  invalidErr,
			},
		}),
	}

	for _, test := range tests {
		result := make(map[string]*secrets.SecretReference)

		err := lbc.addEgressMTLSSecretRefs(result, test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("addEgressMTLSSecretRefs() returned %v, for the case of %v", err, test.msg)
		}
		if diff := cmp.Diff(test.expectedSecretRefs, result, cmp.Comparer(errorComparer)); diff != "" {
			t.Errorf("addEgressMTLSSecretRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddOidcSecret(t *testing.T) {
	invalidErr := errors.New("invalid")
	validSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-oidc-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"client-secret": nil,
		},
		Type: secrets.SecretTypeOIDC,
	}
	invalidSecret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-oidc-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeOIDC,
	}

	tests := []struct {
		policies           []*conf_v1.Policy
		expectedSecretRefs map[string]*secrets.SecretReference
		wantErr            bool
		msg                string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientSecret: "valid-oidc-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-oidc-secret": {
					Secret: validSecret,
				},
			},
			wantErr: false,
			msg:     "test getting valid secret",
		},
		{
			policies:           []*conf_v1.Policy{},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with wrong policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientSecret: "invalid-oidc-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-oidc-secret": {
					Secret: invalidSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid secret",
		},
	}

	lbc := LoadBalancerController{
		secretStore: secrets.NewFakeSecretsStore(map[string]*secrets.SecretReference{
			"default/valid-oidc-secret": {
				Secret: validSecret,
			},
			"default/invalid-oidc-secret": {
				Secret: invalidSecret,
				Error:  invalidErr,
			},
		}),
	}

	for _, test := range tests {
		result := make(map[string]*secrets.SecretReference)

		err := lbc.addOIDCSecretRefs(result, test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("addOIDCSecretRefs() returned %v, for the case of %v", err, test.msg)
		}

		if diff := cmp.Diff(test.expectedSecretRefs, result, cmp.Comparer(errorComparer)); diff != "" {
			t.Errorf("addOIDCSecretRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddWAFPolicyRefs(t *testing.T) {
	apPol := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "default",
				"name":      "ap-pol",
			},
		},
	}

	logConf := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "default",
				"name":      "log-conf",
			},
		},
	}

	tests := []struct {
		policies            []*conf_v1.Policy
		expectedApPolRefs   map[string]*unstructured.Unstructured
		expectedLogConfRefs map[string]*unstructured.Unstructured
		wantErr             bool
		msg                 string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "waf-pol",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						WAF: &conf_v1.WAF{
							Enable:   true,
							ApPolicy: "default/ap-pol",
							SecurityLog: &conf_v1.SecurityLog{
								Enable:    true,
								ApLogConf: "log-conf",
							},
						},
					},
				},
			},
			expectedApPolRefs: map[string]*unstructured.Unstructured{
				"default/ap-pol": apPol,
			},
			expectedLogConfRefs: map[string]*unstructured.Unstructured{
				"default/log-conf": logConf,
			},
			wantErr: false,
			msg:     "base test",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "waf-pol",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						WAF: &conf_v1.WAF{
							Enable:   true,
							ApPolicy: "non-existing-ap-pol",
						},
					},
				},
			},
			wantErr:             true,
			expectedApPolRefs:   make(map[string]*unstructured.Unstructured),
			expectedLogConfRefs: make(map[string]*unstructured.Unstructured),
			msg:                 "apPol doesn't exist",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "waf-pol",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						WAF: &conf_v1.WAF{
							Enable:   true,
							ApPolicy: "ap-pol",
							SecurityLog: &conf_v1.SecurityLog{
								Enable:    true,
								ApLogConf: "non-existing-log-conf",
							},
						},
					},
				},
			},
			wantErr: true,
			expectedApPolRefs: map[string]*unstructured.Unstructured{
				"default/ap-pol": apPol,
			},
			expectedLogConfRefs: make(map[string]*unstructured.Unstructured),
			msg:                 "logConf doesn't exist",
		},
	}

	lbc := LoadBalancerController{
		appProtectConfiguration: appprotect.NewFakeConfiguration(),
	}
	lbc.appProtectConfiguration.AddOrUpdatePolicy(apPol)
	lbc.appProtectConfiguration.AddOrUpdateLogConf(logConf)

	for _, test := range tests {
		resApPolicy := make(map[string]*unstructured.Unstructured)
		resLogConf := make(map[string]*unstructured.Unstructured)

		if err := lbc.addWAFPolicyRefs(resApPolicy, resLogConf, test.policies); (err != nil) != test.wantErr {
			t.Errorf("LoadBalancerController.addWAFPolicyRefs() error = %v, wantErr %v", err, test.wantErr)
		}
		if diff := cmp.Diff(test.expectedApPolRefs, resApPolicy); diff != "" {
			t.Errorf("LoadBalancerController.addWAFPolicyRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
		if diff := cmp.Diff(test.expectedLogConfRefs, resLogConf); diff != "" {
			t.Errorf("LoadBalancerController.addWAFPolicyRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestGetWAFPoliciesForAppProtectPolicy(t *testing.T) {
	apPol := &conf_v1.Policy{
		Spec: conf_v1.PolicySpec{
			WAF: &conf_v1.WAF{
				Enable:   true,
				ApPolicy: "ns1/apPol",
			},
		},
	}

	apPolNs2 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "ns1",
		},
		Spec: conf_v1.PolicySpec{
			WAF: &conf_v1.WAF{
				Enable:   true,
				ApPolicy: "ns2/apPol",
			},
		},
	}

	apPolNoNs := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			WAF: &conf_v1.WAF{
				Enable:   true,
				ApPolicy: "apPol",
			},
		},
	}

	policies := []*conf_v1.Policy{
		apPol, apPolNs2, apPolNoNs,
	}

	tests := []struct {
		pols []*conf_v1.Policy
		key  string
		want []*conf_v1.Policy
		msg  string
	}{
		{
			pols: policies,
			key:  "ns1/apPol",
			want: []*conf_v1.Policy{apPol},
			msg:  "WAF pols that ref apPol which has a namepace",
		},
		{
			pols: policies,
			key:  "default/apPol",
			want: []*conf_v1.Policy{apPolNoNs},
			msg:  "WAF pols that ref apPol which has no namepace",
		},
		{
			pols: policies,
			key:  "ns2/apPol",
			want: []*conf_v1.Policy{apPolNs2},
			msg:  "WAF pols that ref apPol which is in another ns",
		},
		{
			pols: policies,
			key:  "ns1/apPol-with-no-valid-refs",
			want: nil,
			msg:  "WAF pols where there is no valid ref",
		},
	}
	for _, test := range tests {
		got := getWAFPoliciesForAppProtectPolicy(test.pols, test.key)
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("getWAFPoliciesForAppProtectPolicy() returned unexpected result for the case of: %v (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestGetWAFPoliciesForAppProtectLogConf(t *testing.T) {
	logConf := &conf_v1.Policy{
		Spec: conf_v1.PolicySpec{
			WAF: &conf_v1.WAF{
				Enable: true,
				SecurityLog: &conf_v1.SecurityLog{
					Enable:    true,
					ApLogConf: "ns1/logConf",
				},
			},
		},
	}

	logConfNs2 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "ns1",
		},
		Spec: conf_v1.PolicySpec{
			WAF: &conf_v1.WAF{
				Enable: true,
				SecurityLog: &conf_v1.SecurityLog{
					Enable:    true,
					ApLogConf: "ns2/logConf",
				},
			},
		},
	}

	logConfNoNs := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			WAF: &conf_v1.WAF{
				Enable: true,
				SecurityLog: &conf_v1.SecurityLog{
					Enable:    true,
					ApLogConf: "logConf",
				},
			},
		},
	}

	policies := []*conf_v1.Policy{
		logConf, logConfNs2, logConfNoNs,
	}

	tests := []struct {
		pols []*conf_v1.Policy
		key  string
		want []*conf_v1.Policy
		msg  string
	}{
		{
			pols: policies,
			key:  "ns1/logConf",
			want: []*conf_v1.Policy{logConf},
			msg:  "WAF pols that ref logConf which has a namepace",
		},
		{
			pols: policies,
			key:  "default/logConf",
			want: []*conf_v1.Policy{logConfNoNs},
			msg:  "WAF pols that ref logConf which has no namepace",
		},
		{
			pols: policies,
			key:  "ns2/logConf",
			want: []*conf_v1.Policy{logConfNs2},
			msg:  "WAF pols that ref logConf which is in another ns",
		},
		{
			pols: policies,
			key:  "ns1/logConf-with-no-valid-refs",
			want: nil,
			msg:  "WAF pols where there is no valid logConf ref",
		},
	}
	for _, test := range tests {
		got := getWAFPoliciesForAppProtectLogConf(test.pols, test.key)
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("getWAFPoliciesForAppProtectLogConf() returned unexpected result for the case of: %v (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestPreSyncSecrets(t *testing.T) {
	lbc := LoadBalancerController{
		isNginxPlus: true,
		secretStore: secrets.NewEmptyFakeSecretsStore(),
		secretLister: &cache.FakeCustomStore{
			ListFunc: func() []interface{} {
				return []interface{}{
					&api_v1.Secret{
						ObjectMeta: meta_v1.ObjectMeta{
							Name:      "supported-secret",
							Namespace: "default",
						},
						Type: api_v1.SecretTypeTLS,
					},
					&api_v1.Secret{
						ObjectMeta: meta_v1.ObjectMeta{
							Name:      "unsupported-secret",
							Namespace: "default",
						},
						Type: api_v1.SecretTypeOpaque,
					},
				}
			},
		},
	}

	lbc.preSyncSecrets()

	supportedKey := "default/supported-secret"
	ref := lbc.secretStore.GetSecret(supportedKey)
	if ref.Error != nil {
		t.Errorf("GetSecret(%q) returned a reference with an unexpected error %v", supportedKey, ref.Error)
	}

	unsupportedKey := "default/unsupported-secret"
	ref = lbc.secretStore.GetSecret(unsupportedKey)
	if ref.Error == nil {
		t.Errorf("GetSecret(%q) returned a reference without an expected error", unsupportedKey)
	}
}

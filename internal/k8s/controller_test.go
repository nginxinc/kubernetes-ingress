package k8s

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/nginxinc/kubernetes-ingress/internal/telemetry"

	discovery_v1 "k8s.io/api/discovery/v1"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	api_v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestHasCorrectIngressClass(t *testing.T) {
	t.Parallel()
	ingressClass := "ing-ctrl"
	incorrectIngressClass := "gce"
	emptyClass := ""

	tests := []struct {
		lbc      *LoadBalancerController
		ing      *networking.Ingress
		expected bool
	}{
		{
			&LoadBalancerController{
				ingressClass:     ingressClass,
				metricsCollector: collectors.NewControllerFakeCollector(),
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
				ingressClass:     ingressClass,
				metricsCollector: collectors.NewControllerFakeCollector(),
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
				ingressClass:     ingressClass,
				metricsCollector: collectors.NewControllerFakeCollector(),
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
				ingressClass:     ingressClass,
				metricsCollector: collectors.NewControllerFakeCollector(),
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
				ingressClass:     ingressClass,
				metricsCollector: collectors.NewControllerFakeCollector(),
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
				ingressClass:     ingressClass,
				metricsCollector: collectors.NewControllerFakeCollector(),
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
				ingressClass:     ingressClass,
				metricsCollector: collectors.NewControllerFakeCollector(),
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
				ingressClass:     ingressClass,
				metricsCollector: collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				Spec: networking.IngressSpec{
					IngressClassName: &ingressClass,
				},
			},
			true,
		},
		{
			&LoadBalancerController{
				ingressClass:     ingressClass,
				metricsCollector: collectors.NewControllerFakeCollector(),
			},
			&networking.Ingress{
				Spec: networking.IngressSpec{},
			},
			false,
		},
	}

	for _, test := range tests {
		if result := test.lbc.HasCorrectIngressClass(test.ing); result != test.expected {
			classAnnotation := "N/A"
			if class, exists := test.ing.Annotations[ingressClassKey]; exists {
				classAnnotation = class
			}
			t.Errorf("lbc.HasCorrectIngressClass(ing), lbc.ingressClass=%v, ing.Annotations['%v']=%v; got %v, expected %v",
				test.lbc.ingressClass, ingressClassKey, classAnnotation, result, test.expected)
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
	case *conf_v1.TransportServer:
		objCopy := obj.DeepCopy()
		objCopy.Spec.IngressClass = class
		return objCopy
	default:
		panic(fmt.Sprintf("unknown type %T", obj))
	}
}

func TestIngressClassForCustomResources(t *testing.T) {
	t.Parallel()
	ctrl := &LoadBalancerController{
		ingressClass: "nginx",
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
			lbc:             ctrl,
			objIngressClass: "",
			expected:        true,
			msg:             "Ingress Controller handles a resource with an empty IngressClass",
		},
		{
			lbc:             ctrl,
			objIngressClass: "gce",
			expected:        false,
			msg:             "Ingress Controller doesn't handle a resource that doesn't match its IngressClass",
		},
	}

	resources := []interface{}{
		&conf_v1.VirtualServer{},
		&conf_v1.VirtualServerRoute{},
		&conf_v1.TransportServer{},
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

func TestGetServicePortForIngressPort(t *testing.T) {
	t.Parallel()
	fakeClient := fake.NewSimpleClientset()

	cnf := configs.NewConfigurator(configs.ConfiguratorParams{
		NginxManager:            &nginx.LocalManager{},
		StaticCfgParams:         &configs.StaticConfigParams{},
		Config:                  &configs.ConfigParams{},
		TemplateExecutor:        &version1.TemplateExecutor{},
		TemplateExecutorV2:      &version2.TemplateExecutor{},
		LatencyCollector:        nil,
		LabelUpdater:            nil,
		IsPlus:                  false,
		IsWildcardEnabled:       false,
		IsPrometheusEnabled:     false,
		IsLatencyMetricsEnabled: false,
	})
	lbc := LoadBalancerController{
		client:           fakeClient,
		ingressClass:     "nginx",
		configurator:     cnf,
		metricsCollector: collectors.NewControllerFakeCollector(),
	}
	svc := api_v1.Service{
		TypeMeta: meta_v1.TypeMeta{},
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "coffee-svc",
			Namespace: "default",
		},
		Spec: api_v1.ServiceSpec{
			Ports: []api_v1.ServicePort{
				{
					Name:       "foo",
					Port:       80,
					TargetPort: intstr.FromInt(22),
				},
			},
		},
		Status: api_v1.ServiceStatus{},
	}
	backendPort := networking.ServiceBackendPort{
		Name: "foo",
	}
	svcPort := lbc.getServicePortForIngressPort(backendPort, &svc)
	if svcPort == nil || svcPort.Port != 80 {
		t.Errorf("TargetPort string match failed: %+v", svcPort)
	}

	backendPort = networking.ServiceBackendPort{
		Number: 80,
	}
	svcPort = lbc.getServicePortForIngressPort(backendPort, &svc)
	if svcPort == nil || svcPort.Port != 80 {
		t.Errorf("TargetPort int match failed: %+v", svcPort)
	}

	backendPort = networking.ServiceBackendPort{
		Number: 22,
	}
	svcPort = lbc.getServicePortForIngressPort(backendPort, &svc)
	if svcPort != nil {
		t.Errorf("Mismatched ints should not return port: %+v", svcPort)
	}
	backendPort = networking.ServiceBackendPort{
		Name: "bar",
	}
	svcPort = lbc.getServicePortForIngressPort(backendPort, &svc)
	if svcPort != nil {
		t.Errorf("Mismatched strings should not return port: %+v", svcPort)
	}
}

func TestFormatWarningsMessages(t *testing.T) {
	t.Parallel()
	warnings := []string{"Test warning", "Test warning 2"}

	expected := "Test warning; Test warning 2"
	result := formatWarningMessages(warnings)

	if result != expected {
		t.Errorf("formatWarningMessages(%v) returned %v but expected %v", warnings, result, expected)
	}
}

func TestGetEndpointsFromEndpointSlices_DuplicateEndpointsInOneEndpointSlice(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)

	lbc := LoadBalancerController{
		isNginxPlus: true,
	}

	backendServicePort := networking.ServiceBackendPort{
		Number: 8080,
		Name:   "foo",
	}

	endpointReady := true

	tests := []struct {
		desc              string
		svc               api_v1.Service
		svcEndpointSlices []discovery_v1.EndpointSlice
		expectedEndpoints []podEndpoint
	}{
		{
			desc: "duplicate endpoints in an endpointslice",
			svc: api_v1.Service{
				TypeMeta: meta_v1.TypeMeta{},
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee-svc",
					Namespace: "default",
				},
				Spec: api_v1.ServiceSpec{
					Ports: []api_v1.ServicePort{
						{
							Name:       "foo",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						},
					},
				},
				Status: api_v1.ServiceStatus{},
			},
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test // address gosec G601
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints, err := lbc.getEndpointsForPortFromEndpointSlices(test.svcEndpointSlices, backendServicePort, &test.svc)
			if err != nil {
				t.Fatal(err)
			}
			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("lbc.getEndpointsForPortFromEndpointSlices() got %v, want %v",
					gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointsFromEndpointSlices_TwoDifferentEndpointsInOnEndpointSlice(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)

	lbc := LoadBalancerController{
		isNginxPlus: true,
	}

	backendServicePort := networking.ServiceBackendPort{
		Number: 8080,
		Name:   "foo",
	}
	endpointReady := true

	tests := []struct {
		desc              string
		svc               api_v1.Service
		svcEndpointSlices []discovery_v1.EndpointSlice
		expectedEndpoints []podEndpoint
	}{
		{
			desc: "two different endpoints in one endpoint slice",
			svc: api_v1.Service{
				TypeMeta: meta_v1.TypeMeta{},
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee-svc",
					Namespace: "default",
				},
				Spec: api_v1.ServiceSpec{
					Ports: []api_v1.ServicePort{
						{
							Name:       "foo",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						},
					},
				},
				Status: api_v1.ServiceStatus{},
			},
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
				},
				{
					Address: "5.6.7.8:8080",
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
						{
							Addresses: []string{
								"5.6.7.8",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test // address gosec G601
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints, err := lbc.getEndpointsForPortFromEndpointSlices(test.svcEndpointSlices, backendServicePort, &test.svc)
			if err != nil {
				t.Fatal(err)
			}
			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("lbc.getEndpointsForPortFromEndpointSlices() got %v, want %v",
					gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointsFromEndpointSlices_DuplicateEndpointsAcrossTwoEndpointSlices(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)

	lbc := LoadBalancerController{
		isNginxPlus: true,
	}

	backendServicePort := networking.ServiceBackendPort{
		Number: 8080,
		Name:   "foo",
	}

	endpointReady := true

	tests := []struct {
		desc              string
		svc               api_v1.Service
		svcEndpointSlices []discovery_v1.EndpointSlice
		expectedEndpoints []podEndpoint
	}{
		{
			desc: "duplicate endpoints across two endpointslices",
			svc: api_v1.Service{
				TypeMeta: meta_v1.TypeMeta{},
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee-svc",
					Namespace: "default",
				},
				Spec: api_v1.ServiceSpec{
					Ports: []api_v1.ServicePort{
						{
							Name:       "foo",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						},
					},
				},
				Status: api_v1.ServiceStatus{},
			},
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
				},
				{
					Address: "5.6.7.8:8080",
				},
				{
					Address: "10.0.0.1:8080",
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
						{
							Addresses: []string{
								"5.6.7.8",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
						{
							Addresses: []string{
								"10.0.0.1",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test // address gosec G601
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints, err := lbc.getEndpointsForPortFromEndpointSlices(test.svcEndpointSlices, backendServicePort, &test.svc)
			if err != nil {
				t.Fatal(err)
			}
			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("lbc.getEndpointsForPortFromEndpointSlices() got %v, want %v",
					gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointsFromEndpointSlices_TwoDifferentEndpointsInOnEndpointSliceOneEndpointNotReady(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)

	lbc := LoadBalancerController{
		isNginxPlus: true,
	}

	backendServicePort := networking.ServiceBackendPort{
		Number: 8080,
		Name:   "foo",
	}
	endpointReadyTrue := true
	endpointReadyFalse := false

	tests := []struct {
		desc              string
		svc               api_v1.Service
		svcEndpointSlices []discovery_v1.EndpointSlice
		expectedEndpoints []podEndpoint
	}{
		{
			desc: "two different endpoints in one endpoint slice",
			svc: api_v1.Service{
				TypeMeta: meta_v1.TypeMeta{},
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee-svc",
					Namespace: "default",
				},
				Spec: api_v1.ServiceSpec{
					Ports: []api_v1.ServicePort{
						{
							Name:       "foo",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						},
					},
				},
				Status: api_v1.ServiceStatus{},
			},
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReadyTrue,
							},
						},
						{
							Addresses: []string{
								"5.6.7.8",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReadyFalse,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test // address gosec G601
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints, err := lbc.getEndpointsForPortFromEndpointSlices(test.svcEndpointSlices, backendServicePort, &test.svc)
			if err != nil {
				t.Fatal(err)
			}
			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("lbc.getEndpointsForPortFromEndpointSlices() got %v, want %v",
					gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointsFromEndpointSlices_TwoDifferentEndpointsAcrossTwoEndpointSlicesOneEndpointNotReady(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)

	lbc := LoadBalancerController{
		isNginxPlus: true,
	}

	backendServicePort := networking.ServiceBackendPort{
		Number: 8080,
		Name:   "foo",
	}

	endpointReadyTrue := true
	endpointReadyFalse := false

	tests := []struct {
		desc              string
		svc               api_v1.Service
		svcEndpointSlices []discovery_v1.EndpointSlice
		expectedEndpoints []podEndpoint
	}{
		{
			desc: "duplicate endpoints across two endpointslices",
			svc: api_v1.Service{
				TypeMeta: meta_v1.TypeMeta{},
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee-svc",
					Namespace: "default",
				},
				Spec: api_v1.ServiceSpec{
					Ports: []api_v1.ServicePort{
						{
							Name:       "foo",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						},
					},
				},
				Status: api_v1.ServiceStatus{},
			},
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReadyTrue,
							},
						},
					},
				},
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"10.0.0.1",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReadyFalse,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test // address gosec G601
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints, err := lbc.getEndpointsForPortFromEndpointSlices(test.svcEndpointSlices, backendServicePort, &test.svc)
			if err != nil {
				t.Fatal(err)
			}
			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("lbc.getEndpointsForPortFromEndpointSlices() got %v, want %v",
					gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointsFromEndpointSlices_ErrorsOnInvalidTargetPort(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)

	lbc := LoadBalancerController{
		isNginxPlus: true,
	}

	backendServicePort := networking.ServiceBackendPort{
		Number: 8080,
		Name:   "foo",
	}

	tests := []struct {
		desc              string
		svc               api_v1.Service
		svcEndpointSlices []discovery_v1.EndpointSlice
	}{
		{
			desc: "Target Port should be 0",
			svc: api_v1.Service{
				TypeMeta: meta_v1.TypeMeta{},
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee-svc",
					Namespace: "default",
				},
				Spec: api_v1.ServiceSpec{
					Ports: []api_v1.ServicePort{
						{
							Name:       "foo",
							Port:       0,
							TargetPort: intstr.FromInt(0),
						},
					},
				},
				Status: api_v1.ServiceStatus{},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
						},
						{
							Addresses: []string{
								"5.6.7.8",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test // address gosec G601
		t.Run(test.desc, func(t *testing.T) {
			_, err := lbc.getEndpointsForPortFromEndpointSlices(test.svcEndpointSlices, backendServicePort, &test.svc)
			if err == nil {
				t.Logf("%s but was %+v\n", test.desc, test.svc.Spec.Ports[0].TargetPort.IntVal)
				t.Fatal("want error, got nil")
			}
		})
	}
}

func TestGetEndpointsFromEndpointSlices_ErrorsOnNoEndpointSlicesFound(t *testing.T) {
	t.Parallel()
	lbc := LoadBalancerController{
		isNginxPlus: true,
	}

	backendServicePort := networking.ServiceBackendPort{
		Number: 8080,
		Name:   "foo",
	}

	tests := []struct {
		desc              string
		svc               api_v1.Service
		svcEndpointSlices []discovery_v1.EndpointSlice
	}{
		{
			desc: "No EndpointSlices should be found",
			svc: api_v1.Service{
				TypeMeta: meta_v1.TypeMeta{},
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "coffee-svc",
					Namespace: "default",
				},
				Spec: api_v1.ServiceSpec{
					Ports: []api_v1.ServicePort{
						{
							Name:       "foo",
							Port:       80,
							TargetPort: intstr.FromInt(8080),
						},
					},
				},
				Status: api_v1.ServiceStatus{},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{},
		},
	}

	for _, test := range tests {
		test := test // address gosec G601
		t.Run(test.desc, func(t *testing.T) {
			_, err := lbc.getEndpointsForPortFromEndpointSlices(test.svcEndpointSlices, backendServicePort, &test.svc)
			if err == nil {
				t.Logf("%s but got %+v\n", test.desc, test.svcEndpointSlices)
				t.Fatal("want error, got nil")
			}
		})
	}
}

func TestGetEndpointSlicesBySubselectedPods_FindOnePodInOneEndpointSlice(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)
	endpointReady := true
	boolPointer := func(b bool) *bool { return &b }
	tests := []struct {
		desc              string
		targetPort        int32
		svcEndpointSlices []discovery_v1.EndpointSlice
		pods              []*api_v1.Pod
		expectedEndpoints []podEndpoint
	}{
		{
			desc:       "find one pod in one endpointslice",
			targetPort: 8080,
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
					MeshPodOwner: configs.MeshPodOwner{
						OwnerType: "deployment",
						OwnerName: "deploy-1",
					},
				},
			},
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints := getEndpointsFromEndpointSlicesForSubselectedPods(test.targetPort, test.pods, test.svcEndpointSlices)

			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("getEndpointsFromEndpointSlicesForSubselectedPods() = got %v, want %v", gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointSlicesBySubselectedPods_GetsEndpointsOnNilValues(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)
	endpointReady := true
	boolPointer := func(b bool) *bool { return &b }

	tests := []struct {
		desc              string
		targetPort        int32
		svcEndpointSlices []discovery_v1.EndpointSlice
		pods              []*api_v1.Pod
		want              []podEndpoint
	}{
		{
			desc:       "no endpoints selected on nil endpoint port",
			targetPort: 8080,
			want:       []podEndpoint{},
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: nil,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
			},
		},
		{
			desc:       "no endpoints selected on nil endpoint condition",
			targetPort: 8080,
			want:       []podEndpoint{},
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: nil,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := getEndpointsFromEndpointSlicesForSubselectedPods(test.targetPort, test.pods, test.svcEndpointSlices)
			if !cmp.Equal(got, test.want) {
				t.Error(cmp.Diff(got, test.want))
			}
		})
	}
}

func TestGetEndpointSlicesBySubselectedPods_FindOnePodInTwoEndpointSlicesWithDuplicateEndpoints(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)
	endpointReady := true
	boolPointer := func(b bool) *bool { return &b }
	tests := []struct {
		desc              string
		targetPort        int32
		svcEndpointSlices []discovery_v1.EndpointSlice
		pods              []*api_v1.Pod
		expectedEndpoints []podEndpoint
	}{
		{
			desc:       "find one pod in two endpointslices with duplicate endpoints",
			targetPort: 8080,
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
					MeshPodOwner: configs.MeshPodOwner{
						OwnerType: "deployment",
						OwnerName: "deploy-1",
					},
				},
			},
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints := getEndpointsFromEndpointSlicesForSubselectedPods(test.targetPort, test.pods, test.svcEndpointSlices)

			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("getEndpointsFromEndpointSlicesForSubselectedPods() = got %v, want %v", gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointSlicesBySubselectedPods_FindTwoPodsInOneEndpointSlice(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)
	endpointReady := true
	boolPointer := func(b bool) *bool { return &b }
	tests := []struct {
		desc              string
		targetPort        int32
		svcEndpointSlices []discovery_v1.EndpointSlice
		pods              []*api_v1.Pod
		expectedEndpoints []podEndpoint
	}{
		{
			desc:       "find two pods in one endpointslice",
			targetPort: 8080,
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
					MeshPodOwner: configs.MeshPodOwner{
						OwnerType: "deployment",
						OwnerName: "deploy-1",
					},
				},
				{
					Address: "5.6.7.8:8080",
					MeshPodOwner: configs.MeshPodOwner{
						OwnerType: "deployment",
						OwnerName: "deploy-1",
					},
				},
			},
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
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
					Status: api_v1.PodStatus{
						PodIP: "5.6.7.8",
					},
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
						{
							Addresses: []string{
								"5.6.7.8",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints := getEndpointsFromEndpointSlicesForSubselectedPods(test.targetPort, test.pods, test.svcEndpointSlices)

			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("getEndpointsFromEndpointSlicesForSubselectedPods() = got %v, want %v", gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointSlicesBySubselectedPods_FindTwoPodsInTwoEndpointSlices(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)
	endpointReady := true
	boolPointer := func(b bool) *bool { return &b }
	tests := []struct {
		desc              string
		targetPort        int32
		svcEndpointSlices []discovery_v1.EndpointSlice
		pods              []*api_v1.Pod
		expectedEndpoints []podEndpoint
	}{
		{
			desc:       "find two pods in two endpointslices",
			targetPort: 8080,
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
					MeshPodOwner: configs.MeshPodOwner{
						OwnerType: "deployment",
						OwnerName: "deploy-1",
					},
				},
				{
					Address: "5.6.7.8:8080",
					MeshPodOwner: configs.MeshPodOwner{
						OwnerType: "deployment",
						OwnerName: "deploy-1",
					},
				},
			},
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
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
					Status: api_v1.PodStatus{
						PodIP: "5.6.7.8",
					},
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"5.6.7.8",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints := getEndpointsFromEndpointSlicesForSubselectedPods(test.targetPort, test.pods, test.svcEndpointSlices)

			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("getEndpointsFromEndpointSlicesForSubselectedPods() = got %v, want %v", gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointSlicesBySubselectedPods_FindOnePodEndpointInOneEndpointSliceWithOneEndpointNotReady(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)
	endpointReadyTrue := true
	endpointReadyFalse := false
	boolPointer := func(b bool) *bool { return &b }
	tests := []struct {
		desc              string
		targetPort        int32
		svcEndpointSlices []discovery_v1.EndpointSlice
		pods              []*api_v1.Pod
		expectedEndpoints []podEndpoint
	}{
		{
			desc:       "find two pods in one endpointslice",
			targetPort: 8080,
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
					MeshPodOwner: configs.MeshPodOwner{
						OwnerType: "deployment",
						OwnerName: "deploy-1",
					},
				},
			},
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
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
					Status: api_v1.PodStatus{
						PodIP: "5.6.7.8",
					},
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReadyTrue,
							},
						},
						{
							Addresses: []string{
								"5.6.7.8",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReadyFalse,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints := getEndpointsFromEndpointSlicesForSubselectedPods(test.targetPort, test.pods, test.svcEndpointSlices)

			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("getEndpointsFromEndpointSlicesForSubselectedPods() = got %v, want %v", gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointSlicesBySubselectedPods_FindOnePodEndpointInTwoEndpointSlicesWithOneEndpointNotReady(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)
	endpointReadyTrue := true
	endpointReadyFalse := false
	boolPointer := func(b bool) *bool { return &b }
	tests := []struct {
		desc              string
		targetPort        int32
		svcEndpointSlices []discovery_v1.EndpointSlice
		pods              []*api_v1.Pod
		expectedEndpoints []podEndpoint
	}{
		{
			desc:       "find two pods in two endpointslices",
			targetPort: 8080,
			expectedEndpoints: []podEndpoint{
				{
					Address: "1.2.3.4:8080",
					MeshPodOwner: configs.MeshPodOwner{
						OwnerType: "deployment",
						OwnerName: "deploy-1",
					},
				},
			},
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
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
					Status: api_v1.PodStatus{
						PodIP: "5.6.7.8",
					},
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReadyTrue,
							},
						},
					},
				},
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"5.6.7.8",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReadyFalse,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints := getEndpointsFromEndpointSlicesForSubselectedPods(test.targetPort, test.pods, test.svcEndpointSlices)

			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("getEndpointsFromEndpointSlicesForSubselectedPods() = got %v, want %v", gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointSlicesBySubselectedPods_FindNoPods(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)
	endpointReady := true
	boolPointer := func(b bool) *bool { return &b }
	tests := []struct {
		desc              string
		targetPort        int32
		svcEndpointSlices []discovery_v1.EndpointSlice
		pods              []*api_v1.Pod
		expectedEndpoints []podEndpoint
	}{
		{
			desc:              "find no pods",
			targetPort:        8080,
			expectedEndpoints: nil,
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
			},
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"5.4.3.2",
							},
							Conditions: discovery_v1.EndpointConditions{
								Ready: &endpointReady,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints := getEndpointsFromEndpointSlicesForSubselectedPods(test.targetPort, test.pods, test.svcEndpointSlices)

			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("getEndpointsFromEndpointSlicesForSubselectedPods() = got %v, want %v", gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func TestGetEndpointSlicesBySubselectedPods_TargetPortMismatch(t *testing.T) {
	t.Parallel()
	endpointPort := int32(8080)

	boolPointer := func(b bool) *bool { return &b }
	tests := []struct {
		desc              string
		targetPort        int32
		svcEndpointSlices []discovery_v1.EndpointSlice
		pods              []*api_v1.Pod
		expectedEndpoints []podEndpoint
	}{
		{
			desc:       "targetPort mismatch",
			targetPort: 21,
			svcEndpointSlices: []discovery_v1.EndpointSlice{
				{
					Ports: []discovery_v1.EndpointPort{
						{
							Port: &endpointPort,
						},
					},
					Endpoints: []discovery_v1.Endpoint{
						{
							Addresses: []string{
								"1.2.3.4",
							},
						},
					},
				},
			},
			pods: []*api_v1.Pod{
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
					Status: api_v1.PodStatus{
						PodIP: "1.2.3.4",
					},
				},
			},
			expectedEndpoints: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			gotEndpoints := getEndpointsFromEndpointSlicesForSubselectedPods(test.targetPort, test.pods, test.svcEndpointSlices)

			if result := unorderedEqual(gotEndpoints, test.expectedEndpoints); !result {
				t.Errorf("getEndpointsFromEndpointSlicesForSubselectedPods() = got %v, want %v", gotEndpoints, test.expectedEndpoints)
			}
		})
	}
}

func unorderedEqual(got, want []podEndpoint) bool {
	if len(got) != len(want) {
		return false
	}
	exists := make(map[string]bool)
	for _, value := range got {
		exists[value.Address] = true
	}
	for _, value := range want {
		if !exists[value.Address] {
			return false
		}
	}
	return true
}

func TestGetStatusFromEventTitle(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

	policyLister := &cache.FakeCustomStore{
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
	}

	nsi := make(map[string]*namespacedInformer)
	nsi[""] = &namespacedInformer{policyLister: policyLister}

	lbc := LoadBalancerController{
		isNginxPlus:         true,
		namespacedInformers: nsi,
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
		errors.New("policy default/invalid-policy is invalid: spec: Invalid value: \"\": must specify exactly one of: `accessControl`, `rateLimit`, `ingressMTLS`, `egressMTLS`, `basicAuth`, `apiKey`, `jwt`, `oidc`, `waf`"),
		errors.New("policy nginx-ingress/valid-policy doesn't exist"),
		errors.New("failed to get policy nginx-ingress/some-policy: GetByKey error"),
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
	t.Parallel()
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
	t.Parallel()
	tests := []struct {
		desc    string
		expType string
		expName string
		pod     *api_v1.Pod
	}{
		{
			desc:    "deployment",
			expType: "deployment",
			expName: "deploy-name",
			pod:     &api_v1.Pod{ObjectMeta: createTestObjMeta("Deployment", "deploy-name", true)},
		},
		{
			desc:    "stateful set",
			expType: "statefulset",
			expName: "statefulset-name",
			pod:     &api_v1.Pod{ObjectMeta: createTestObjMeta("StatefulSet", "statefulset-name", true)},
		},
		{
			desc:    "daemon set",
			expType: "daemonset",
			expName: "daemonset-name",
			pod:     &api_v1.Pod{ObjectMeta: createTestObjMeta("DaemonSet", "daemonset-name", true)},
		},
		{
			desc:    "replica set with no pod hash",
			expType: "deployment",
			expName: "replicaset-name",
			pod:     &api_v1.Pod{ObjectMeta: createTestObjMeta("ReplicaSet", "replicaset-name", false)},
		},
		{
			desc:    "replica set with pod hash",
			expType: "deployment",
			expName: "replicaset-name",
			pod: &api_v1.Pod{
				ObjectMeta: createTestObjMeta("ReplicaSet", "replicaset-name-67c6f7c5fd", true),
			},
		},
		{
			desc:    "nil controller should use default values",
			expType: "deployment",
			expName: "deploy-name",
			pod: &api_v1.Pod{
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
	t.Parallel()
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

func errorComparer(e1, e2 error) bool {
	if e1 == nil || e2 == nil {
		return errors.Is(e1, e2)
	}

	return e1.Error() == e2.Error()
}

func TestNewTelemetryCollector(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testCase          string
		input             NewLoadBalancerControllerInput
		collectorConfig   telemetry.CollectorConfig
		expectedCollector telemetry.Collector
	}{
		{
			testCase: "New Telemetry Collector with default values",
			input: NewLoadBalancerControllerInput{
				KubeClient:               fake.NewSimpleClientset(),
				EnableTelemetryReporting: true,
			},
			expectedCollector: telemetry.Collector{
				Config: telemetry.CollectorConfig{
					Period: 24 * time.Hour,
				},
				Exporter: &telemetry.StdoutExporter{},
			},
		},
		{
			testCase: "New Telemetry Collector with Telemetry Reporting set to false",
			input: NewLoadBalancerControllerInput{
				KubeClient:               fake.NewSimpleClientset(),
				EnableTelemetryReporting: false,
			},
			expectedCollector: telemetry.Collector{},
		},
	}

	for _, tc := range testCases {
		lbc := NewLoadBalancerController(tc.input)
		if reflect.DeepEqual(tc.expectedCollector, lbc.telemetryCollector) {
			t.Fatalf("Expected %v, but got %v", tc.expectedCollector, lbc.telemetryCollector)
		}
	}
}

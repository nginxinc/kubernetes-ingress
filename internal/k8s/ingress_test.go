/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8s

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/configs"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version1"
	"github.com/nginxinc/kubernetes-ingress/internal/configs/version2"
	"github.com/nginxinc/kubernetes-ingress/internal/metrics/collectors"
	"github.com/nginxinc/kubernetes-ingress/internal/nginx"
	api_v1 "k8s.io/api/core/v1"
	discovery_v1 "k8s.io/api/discovery/v1"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

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

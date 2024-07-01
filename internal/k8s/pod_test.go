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

	api_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestComparePorts(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		sp       api_v1.ServicePort
		cp       api_v1.ContainerPort
		expected bool
	}{
		{
			// match TargetPort.strval and Protocol
			api_v1.ServicePort{
				TargetPort: intstr.FromString("name"),
				Protocol:   api_v1.ProtocolTCP,
			},
			api_v1.ContainerPort{
				Name:          "name",
				Protocol:      api_v1.ProtocolTCP,
				ContainerPort: 80,
			},
			true,
		},
		{
			// don't match Name and Protocol
			api_v1.ServicePort{
				Name:     "name",
				Protocol: api_v1.ProtocolTCP,
			},
			api_v1.ContainerPort{
				Name:          "name",
				Protocol:      api_v1.ProtocolTCP,
				ContainerPort: 80,
			},
			false,
		},
		{
			// TargetPort intval mismatch, don't match by TargetPort.Name
			api_v1.ServicePort{
				Name:       "name",
				TargetPort: intstr.FromInt(80),
			},
			api_v1.ContainerPort{
				Name:          "name",
				ContainerPort: 81,
			},
			false,
		},
		{
			// match by TargetPort intval
			api_v1.ServicePort{
				TargetPort: intstr.IntOrString{
					IntVal: 80,
				},
			},
			api_v1.ContainerPort{
				ContainerPort: 80,
			},
			true,
		},
		{
			// Fall back on ServicePort.Port if TargetPort is empty
			api_v1.ServicePort{
				Name: "name",
				Port: 80,
			},
			api_v1.ContainerPort{
				Name:          "name",
				ContainerPort: 80,
			},
			true,
		},
		{
			// TargetPort intval mismatch
			api_v1.ServicePort{
				TargetPort: intstr.FromInt(80),
			},
			api_v1.ContainerPort{
				ContainerPort: 81,
			},
			false,
		},
		{
			// don't match empty ports
			api_v1.ServicePort{},
			api_v1.ContainerPort{},
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
	t.Parallel()
	pods := []*api_v1.Pod{
		{
			Spec: api_v1.PodSpec{
				Containers: []api_v1.Container{
					{
						ReadinessProbe: &api_v1.Probe{
							ProbeHandler: api_v1.ProbeHandler{
								HTTPGet: &api_v1.HTTPGetAction{
									Path: "/",
									Host: "asdf.com",
									Port: intstr.IntOrString{
										IntVal: 80,
									},
								},
							},
							PeriodSeconds: 42,
						},
						Ports: []api_v1.ContainerPort{
							{
								Name:          "name",
								ContainerPort: 80,
								Protocol:      api_v1.ProtocolTCP,
								HostIP:        "1.2.3.4",
							},
						},
					},
				},
			},
		},
	}
	svcPort := api_v1.ServicePort{
		TargetPort: intstr.FromInt(80),
	}
	probe := findProbeForPods(pods, &svcPort)
	if probe == nil || probe.PeriodSeconds != 42 {
		t.Errorf("ServicePort.TargetPort as int match failed: %+v", probe)
	}

	svcPort = api_v1.ServicePort{
		TargetPort: intstr.FromString("name"),
		Protocol:   api_v1.ProtocolTCP,
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe == nil || probe.PeriodSeconds != 42 {
		t.Errorf("ServicePort.TargetPort as string failed: %+v", probe)
	}

	svcPort = api_v1.ServicePort{
		TargetPort: intstr.FromInt(80),
		Protocol:   api_v1.ProtocolTCP,
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe == nil || probe.PeriodSeconds != 42 {
		t.Errorf("ServicePort.TargetPort as int failed: %+v", probe)
	}

	svcPort = api_v1.ServicePort{
		Port: 80,
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe == nil || probe.PeriodSeconds != 42 {
		t.Errorf("ServicePort.Port should match if TargetPort is not set: %+v", probe)
	}

	svcPort = api_v1.ServicePort{
		TargetPort: intstr.FromString("wrong_name"),
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe != nil {
		t.Errorf("ServicePort.TargetPort should not have matched string: %+v", probe)
	}

	svcPort = api_v1.ServicePort{
		TargetPort: intstr.FromInt(22),
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe != nil {
		t.Errorf("ServicePort.TargetPort should not have matched int: %+v", probe)
	}

	svcPort = api_v1.ServicePort{
		Port: 22,
	}
	probe = findProbeForPods(pods, &svcPort)
	if probe != nil {
		t.Errorf("ServicePort.Port mismatch: %+v", probe)
	}
}

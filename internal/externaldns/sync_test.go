package externaldns

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	vsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	extdnsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/externaldns/v1"
	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/externaldns/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetValidTargets(t *testing.T) {
	t.Parallel()
	tt := []struct {
		name        string
		wantTargets extdnsapi.Targets
		wantRecord  string
		endpoints   []vsapi.ExternalEndpoint
	}{
		{
			name:        "from external endpoint with ipv4",
			wantTargets: extdnsapi.Targets{"10.23.4.5"},
			wantRecord:  "A",
			endpoints: []vsapi.ExternalEndpoint{
				{
					IP: "10.23.4.5",
				},
			},
		},
		{
			name:        "from external endpoint with ipv6",
			wantTargets: extdnsapi.Targets{"2001:db8:0:0:0:0:2:1"},
			wantRecord:  "AAAA",
			endpoints: []vsapi.ExternalEndpoint{
				{
					IP: "2001:db8:0:0:0:0:2:1",
				},
			},
		},
		{
			name:        "from external endpoint with a hostname",
			wantTargets: extdnsapi.Targets{"tea.com"},
			wantRecord:  "CNAME",
			endpoints: []vsapi.ExternalEndpoint{
				{
					Hostname: "tea.com",
				},
			},
		},
		{
			name:        "from external endpoint with multiple targets",
			wantTargets: extdnsapi.Targets{"2001:db8:0:0:0:0:2:1", "10.2.3.4"},
			wantRecord:  "A",
			endpoints: []vsapi.ExternalEndpoint{
				{
					IP: "2001:db8:0:0:0:0:2:1",
				},
				{
					IP: "10.2.3.4",
				},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			targets, recordType, err := getValidTargets(tc.endpoints)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(tc.wantTargets, targets) {
				t.Errorf(cmp.Diff(tc.wantTargets, targets))
			}
			if recordType != tc.wantRecord {
				t.Errorf(cmp.Diff(tc.wantRecord, recordType))
			}
		})
	}
}

// Implementation of interfaces for testing purposes

type DNSEndpointListerExpansion interface{}

type EPNamespaceLister interface {
	List(selector labels.Selector)
	Get(name string) (*v1.DNSEndpoint, error)
	DNSEndpointListerExpansion
}

type DNSEndpointLister interface {
	List(selector labels.Selector) ([]*v1.DNSEndpoint, error)
	DNSEndpoints(namespace string) EPNamespaceLister
	DNSEndpointListerExpansion
}

type EPLister struct{}

func (EPLister) List(labels.Selector) ([]*v1.DNSEndpoint, error) {
	return nil, nil
}

func (EPLister) DNSEndpoints(namespace string) EPNamespaceLister {
	return nil
}

func TestSync_NotRunningOnExternalDNSDisabled(t *testing.T) {
	t.Parallel()
	vs := &vsapi.VirtualServer{
		Spec: vsapi.VirtualServerSpec{
			ExternalDNS: vsapi.ExternalDNS{
				Enable: false,
			},
		},
	}
	fn := SyncFnFor(nil, nil, nil)
	err := fn(context.TODO(), vs)
	if err != nil {
		t.Errorf("want nil got %v", err)
	}
}

// EventRecorder implements EventRecorder interface.
// It's dummy implementation purpose is for testing only.
type EventRecorder struct{}

func (EventRecorder) Event(runtime.Object, string, string, string)                  {}
func (EventRecorder) Eventf(runtime.Object, string, string, string, ...interface{}) {}
func (EventRecorder) AnnotatedEventf(runtime.Object, map[string]string, string, string, string, ...interface{}) {
}

func TestSync_ReturnsErrorOnNilExternalEndpoints(t *testing.T) {
	t.Parallel()
	vs := &vsapi.VirtualServer{
		Spec: vsapi.VirtualServerSpec{
			ExternalDNS: vsapi.ExternalDNS{
				Enable: true,
			},
		},
		Status: vsapi.VirtualServerStatus{},
	}

	rec := EventRecorder{}
	fn := SyncFnFor(rec, nil, nil)
	err := fn(context.TODO(), vs)
	if err == nil {
		t.Errorf("want error got nil")
	}
}

func TestSync_ReturnsErrorOnInvalidTargetsInExternalEndpoints(t *testing.T) {
	t.Parallel()
	tt := []struct {
		name  string
		input *vsapi.VirtualServer
	}{
		{
			name: "when missing ip and Hostname",
			input: &vsapi.VirtualServer{
				Spec: vsapi.VirtualServerSpec{
					ExternalDNS: vsapi.ExternalDNS{
						Enable: true,
					},
				},
				Status: vsapi.VirtualServerStatus{
					ExternalEndpoints: []vsapi.ExternalEndpoint{
						{
							IP:       "",
							Hostname: "",
						},
					},
				},
			},
		},
		{
			name: "when missing hostname",
			input: &vsapi.VirtualServer{
				Spec: vsapi.VirtualServerSpec{
					ExternalDNS: vsapi.ExternalDNS{
						Enable: true,
					},
				},
				Status: vsapi.VirtualServerStatus{
					ExternalEndpoints: []vsapi.ExternalEndpoint{
						{
							Hostname: "",
						},
					},
				},
			},
		},
		{
			name: "when invalid ipv4 address",
			input: &vsapi.VirtualServer{
				Spec: vsapi.VirtualServerSpec{
					ExternalDNS: vsapi.ExternalDNS{
						Enable: true,
					},
				},
				Status: vsapi.VirtualServerStatus{
					ExternalEndpoints: []vsapi.ExternalEndpoint{
						{
							IP:       "10.23.23..3",
							Hostname: "",
						},
					},
				},
			},
		},
		{
			name: "when invalid ipv6 address",
			input: &vsapi.VirtualServer{
				Spec: vsapi.VirtualServerSpec{
					ExternalDNS: vsapi.ExternalDNS{
						Enable: true,
					},
				},
				Status: vsapi.VirtualServerStatus{
					ExternalEndpoints: []vsapi.ExternalEndpoint{
						{
							IP:       "2001:::db8:0:0:0:0:2:1",
							Hostname: "",
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rec := EventRecorder{}
			fn := SyncFnFor(rec, nil, nil)
			err := fn(context.TODO(), tc.input)
			if err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

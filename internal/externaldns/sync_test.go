package externaldns_test

import (
	"testing"

	"github.com/nginxinc/kubernetes-ingress/internal/externaldns"
	vsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
)

func TestGetEndpointAndRecordType(t *testing.T) {
	t.Parallel()
	tt := []struct {
		name        string
		wantEnpoint string
		wantRecord  string
		endpoint    vsapi.ExternalEndpoint
	}{
		{
			name:        "from external endpoint with IPv4",
			wantEnpoint: "10.23.4.5",
			wantRecord:  "A",
			endpoint: vsapi.ExternalEndpoint{
				IP: "10.23.4.5",
			},
		},
		{
			name:        "from external endpoint with IPv6",
			wantEnpoint: "2001:db8:0:0:0:0:2:1",
			wantRecord:  "AAAA",
			endpoint: vsapi.ExternalEndpoint{
				IP: "2001:db8:0:0:0:0:2:1",
			},
		},
		{
			name:        "from external endpoint with a hostname",
			wantEnpoint: "tea.com",
			wantRecord:  "CNAME",
			endpoint: vsapi.ExternalEndpoint{
				Hostname: "tea.com",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotEndpoint, gotRecord, err := externaldns.GetEndpointAndRecordType(tc.endpoint)
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantEnpoint != gotEndpoint {
				t.Errorf("want %s, got %s", tc.wantEnpoint, gotEndpoint)
			}
			if tc.wantRecord != gotRecord {
				t.Errorf("want %s, got %s", tc.wantRecord, gotRecord)
			}
		})
	}
}

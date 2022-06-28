package externaldns

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	vsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	extdnsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/externaldns/v1"
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
			name:        "from external endpoint with IPv4",
			wantTargets: extdnsapi.Targets{"10.23.4.5"},
			wantRecord:  "A",
			endpoints: []vsapi.ExternalEndpoint{
				{
					IP: "10.23.4.5",
				},
			},
		},
		{
			name:        "from external endpoint with IPv6",
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

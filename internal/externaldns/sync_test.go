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
		name      string
		want      extdnsapi.Targets
		endpoints []vsapi.ExternalEndpoint
	}{
		{
			name: "from external endpoint with IPv4",
			want: extdnsapi.Targets{"10.23.4.5"},
			endpoints: []vsapi.ExternalEndpoint{
				{
					IP: "10.23.4.5",
				},
			},
		},
		{
			name: "from external endpoint with IPv6",
			want: extdnsapi.Targets{"2001:db8:0:0:0:0:2:1"},
			endpoints: []vsapi.ExternalEndpoint{
				{
					IP: "2001:db8:0:0:0:0:2:1",
				},
			},
		},
		{
			name: "from external endpoint with a hostname",
			want: extdnsapi.Targets{"tea.com"},
			endpoints: []vsapi.ExternalEndpoint{
				{
					Hostname: "tea.com",
				},
			},
		},
		{
			name: "from external endpoint with multiple targets",
			want: extdnsapi.Targets{"2001:db8:0:0:0:0:2:1", "10.2.3.4"},
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
			got, err := getValidTargets(tc.endpoints)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(tc.want, got) {
				t.Errorf(cmp.Diff(tc.want, got))
			}
		})
	}
}

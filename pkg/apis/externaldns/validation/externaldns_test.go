package validation_test

import (
	"errors"
	"strings"
	"testing"

	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/externaldns/v1"
	"github.com/nginxinc/kubernetes-ingress/pkg/apis/externaldns/validation"
)

func TestValidateTargetsAndDetermineRecordType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		targets    []string
		wantErr    bool
		wantErrMsg string
		wantType   string
	}{
		{
			name:     "single IPv4 target",
			targets:  []string{"10.10.10.10"},
			wantType: "A",
		},
		{
			name:     "single IPv6 target",
			targets:  []string{"2001:db8::ff00:42:8329"},
			wantType: "AAAA",
		},
		{
			name:     "single CNAME (hostname) target",
			targets:  []string{"example.com"},
			wantType: "CNAME",
		},
		{
			name:     "multiple IPv4 targets",
			targets:  []string{"192.168.1.1", "10.10.10.10"},
			wantType: "A",
		},
		{
			name:     "multiple IPv6 targets",
			targets:  []string{"2001:db8::1", "2001:db8::2"},
			wantType: "AAAA",
		},
		{
			name:     "multiple hostnames",
			targets:  []string{"foo.example.com", "bar.example.com"},
			wantType: "CNAME",
		},
		{
			name:       "mixed IPv4 and IPv6",
			targets:    []string{"192.168.1.1", "2001:db8::1"},
			wantErr:    true,
			wantErrMsg: "multiple record types",
		},
		{
			name:       "mixed IPv4 and CNAME",
			targets:    []string{"192.168.1.1", "example.com"},
			wantErr:    true,
			wantErrMsg: "multiple record types",
		},
		{
			name:       "invalid hostname",
			targets:    []string{"not_a_valid_hostname"},
			wantErr:    true,
			wantErrMsg: "invalid",
		},
		{
			name:       "empty targets",
			targets:    []string{},
			wantErr:    true,
			wantErrMsg: "determine record type",
		},
		{
			name:       "duplicate targets",
			targets:    []string{"example.com", "example.com"},
			wantErr:    true,
			wantErrMsg: "expected unique targets",
		},
	}

	for _, tc := range tests {
		tc := tc // address gosec G601
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, recordType, err := validation.ValidateTargetsAndDetermineRecordType(tc.targets)
			if tc.wantErr && err == nil {
				t.Fatalf("expected an error, got none")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantErr && err != nil && tc.wantErrMsg != "" && !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Errorf("expected error message containing %q, got %q", tc.wantErrMsg, err.Error())
			}
			if !tc.wantErr && recordType != tc.wantType {
				t.Errorf("expected record type %q, got %q", tc.wantType, recordType)
			}
		})
	}
}

func TestValidateDNSEndpoint(t *testing.T) {
	t.Parallel()
	tt := []struct {
		name     string
		endpoint v1.DNSEndpoint
	}{
		{
			name: "with a single valid endpoint",
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"10.2.2.3"},
							RecordType: "A",
							RecordTTL:  600,
						},
					},
				},
			},
		},
		{
			name: "with a single IPv6 target",
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"2001:db8:0:0:0:0:2:1"},
							RecordType: "A",
							RecordTTL:  600,
						},
					},
				},
			},
		},
		{
			name: "with multiple valid endpoints",
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"10.2.2.3"},
							RecordType: "A",
							RecordTTL:  600,
						},
						{
							DNSName:    "example.co.uk",
							Targets:    v1.Targets{"10.2.2.3"},
							RecordType: "CNAME",
							RecordTTL:  900,
						},
						{
							DNSName:    "example.ie",
							Targets:    v1.Targets{"2001:db8:0:0:0:0:2:1"},
							RecordType: "AAAA",
							RecordTTL:  900,
						},
					},
				},
			},
		},
		{
			name: "with multiple valid endpoints and multiple targets",
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"example.ie", "example.io"},
							RecordType: "CNAME",
							RecordTTL:  600,
						},
						{
							DNSName:    "example.co.uk",
							Targets:    v1.Targets{"10.2.2.3", "192.123.23.4"},
							RecordType: "A",
							RecordTTL:  900,
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		tc := tc // address gosec G601
		t.Run(tc.name, func(t *testing.T) {
			if err := validation.ValidateDNSEndpoint(&tc.endpoint); err != nil {
				t.Errorf("want no error on %v, got %v", tc.endpoint, err)
			}
		})
	}
}

func TestValidateDNSEndpoint_ReturnsErrorOn(t *testing.T) {
	t.Parallel()
	tt := []struct {
		name     string
		want     error
		endpoint v1.DNSEndpoint
	}{
		{
			name: "not supported DNS record type",
			want: validation.ErrTypeNotSupported,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"10.2.2.3"},
							RecordType: "bogusRecordType",
							RecordTTL:  600,
						},
					},
				},
			},
		},
		{
			name: "bogus target hostname",
			want: validation.ErrTypeInvalid,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"bogusTargetName"},
							RecordType: "A",
							RecordTTL:  600,
						},
					},
				},
			},
		},
		{
			name: "bogus target IPv6 address",
			want: validation.ErrTypeInvalid,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"2001:::0:0:0:0:2:1"},
							RecordType: "A",
							RecordTTL:  600,
						},
					},
				},
			},
		},
		{
			name: "duplicated target",
			want: validation.ErrTypeDuplicated,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"acme.com", "10.2.2.3", "acme.com"},
							RecordType: "A",
							RecordTTL:  600,
						},
					},
				},
			},
		},
		{
			name: "bogus ttl record",
			want: validation.ErrTypeNotInRange,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"10.2.2.3", "acme.com"},
							RecordType: "A",
							RecordTTL:  -1,
						},
					},
				},
			},
		},
		{
			name: "bogus dns name",
			want: validation.ErrTypeInvalid,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "bogusDNSName",
							Targets:    v1.Targets{"acme.com"},
							RecordType: "A",
							RecordTTL:  1800,
						},
					},
				},
			},
		},
		{
			name: "empty dns name",
			want: validation.ErrTypeInvalid,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "",
							Targets:    v1.Targets{"acme.com"},
							RecordType: "A",
							RecordTTL:  1800,
						},
					},
				},
			},
		},
		{
			name: "bogus target name",
			want: validation.ErrTypeInvalid,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"acme."},
							RecordType: "A",
							RecordTTL:  1800,
						},
					},
				},
			},
		},
		{
			name: "empty target name",
			want: validation.ErrTypeInvalid,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{""},
							RecordType: "A",
							RecordTTL:  1800,
						},
					},
				},
			},
		},
		{
			name: "bogus target name",
			want: validation.ErrTypeInvalid,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{
						{
							DNSName:    "example.com",
							Targets:    v1.Targets{"&$$.*&^"},
							RecordType: "A",
							RecordTTL:  1800,
						},
					},
				},
			},
		},
		{
			name: "empty slice of endpoints",
			want: validation.ErrTypeRequired,
			endpoint: v1.DNSEndpoint{
				Spec: v1.DNSEndpointSpec{
					Endpoints: []*v1.Endpoint{},
				},
			},
		},
	}

	for _, tc := range tt {
		tc := tc // address gosec G601
		t.Run(tc.name, func(t *testing.T) {
			err := validation.ValidateDNSEndpoint(&tc.endpoint)
			if !errors.Is(err, tc.want) {
				t.Errorf("want %s, got %v", tc.want, err)
			}
		})
	}
}

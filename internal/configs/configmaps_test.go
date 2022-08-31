package configs

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestParseConfigMapWithAppProtectCompressedRequestsAction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		action string
		expect string
		msg    string
	}{
		{
			action: "pass",
			expect: "pass",
			msg:    "valid action pass",
		},
		{
			action: "drop",
			expect: "drop",
			msg:    "valid action drop",
		},
		{
			action: "invalid",
			expect: "",
			msg:    "invalid action",
		},
		{
			action: "",
			expect: "",
			msg:    "empty action",
		},
	}
	nginxPlus := true
	hasAppProtect := true
	hasAppProtectDos := false
	hasTLSPassthrough := false
	for _, test := range tests {
		cm := &v1.ConfigMap{
			Data: map[string]string{
				"app-protect-compressed-requests-action": test.action,
			},
		}
		result := ParseConfigMap(cm, nginxPlus, hasAppProtect, hasAppProtectDos, hasTLSPassthrough)
		if result.MainAppProtectCompressedRequestsAction != test.expect {
			t.Errorf("ParseConfigMap() returned %q but expected %q for the case %s", result.MainAppProtectCompressedRequestsAction, test.expect, test.msg)
		}
	}
}

func TestParseConfigMapWithAppProtectReconnectPeriod(t *testing.T) {
	tests := []struct {
		period string
		expect string
		msg    string
	}{
		{
			period: "25",
			expect: "25",
			msg:    "valid period 25",
		},
		{
			period: "13.875",
			expect: "13.875",
			msg:    "valid period 13.875",
		},
		{
			period: "0.125",
			expect: "0.125",
			msg:    "valid period 0.125",
		},
		{
			period: "60",
			expect: "60",
			msg:    "valid period 60",
		},
		{
			period: "60.1",
			expect: "",
			msg:    "invalid period 60.1",
		},
		{
			period: "100",
			expect: "",
			msg:    "invalid period 100",
		},
		{
			period: "0",
			expect: "",
			msg:    "invalid period 0",
		},
		{
			period: "-5",
			expect: "",
			msg:    "invalid period -5",
		},
		{
			period: "",
			expect: "",
			msg:    "empty period",
		},
	}
	nginxPlus := true
	hasAppProtect := true
	hasAppProtectDos := false
	hasTLSPassthrough := false
	for _, test := range tests {
		cm := &v1.ConfigMap{
			Data: map[string]string{
				"app-protect-reconnect-period-seconds": test.period,
			},
		}
		result := ParseConfigMap(cm, nginxPlus, hasAppProtect, hasAppProtectDos, hasTLSPassthrough)
		if result.MainAppProtectReconnectPeriod != test.expect {
			t.Errorf("ParseConfigMap() returned %q but expected %q for the case %s", result.MainAppProtectReconnectPeriod, test.expect, test.msg)
		}
	}
}

func TestParseConfigMapWithTLSPassthroughProxyProtocol(t *testing.T) {
	t.Parallel()
	tests := []struct {
		realIPheader string
		want         string
		msg          string
	}{
		{
			realIPheader: "proxy_protocol",
			want:         "",
			msg:          "valid realIPheader proxy_protocol, ignored when TLS Passthrough is enabled",
		},
		{
			realIPheader: "X-Forwarded-For",
			want:         "",
			msg:          "invalid realIPheader X-Forwarded-For, ignored when TLS Passthrough is enabled",
		},
		{
			realIPheader: "",
			want:         "",
			msg:          "empty real-ip-header",
		},
	}
	nginxPlus := true
	hasAppProtect := true
	hasAppProtectDos := false
	hasTLSPassthrough := true
	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			cm := &v1.ConfigMap{
				Data: map[string]string{
					"real-ip-header": test.realIPheader,
				},
			}
			result := ParseConfigMap(cm, nginxPlus, hasAppProtect, hasAppProtectDos, hasTLSPassthrough)
			if result.RealIPHeader != test.want {
				t.Errorf("want %q, got %q", test.want, result.RealIPHeader)
			}
		})
	}
}

func TestParseConfigMapWithoutTLSPassthroughProxyProtocol(t *testing.T) {
	t.Parallel()
	tests := []struct {
		realIPheader string
		want         string
		msg          string
	}{
		{
			realIPheader: "proxy_protocol",
			want:         "proxy_protocol",
			msg:          "valid real-ip-header proxy_protocol",
		},
		{
			realIPheader: "X-Forwarded-For",
			want:         "X-Forwarded-For",
			msg:          "valid real-ip-header X-Forwarded-For",
		},
		{
			realIPheader: "",
			want:         "",
			msg:          "empty real-ip-header",
		},
	}
	nginxPlus := true
	hasAppProtect := true
	hasAppProtectDos := false
	hasTLSPassthrough := false
	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			cm := &v1.ConfigMap{
				Data: map[string]string{
					"real-ip-header": test.realIPheader,
				},
			}
			result := ParseConfigMap(cm, nginxPlus, hasAppProtect, hasAppProtectDos, hasTLSPassthrough)
			if result.RealIPHeader != test.want {
				t.Errorf("want %q, got %q", test.want, result.RealIPHeader)
			}
		})
	}
}

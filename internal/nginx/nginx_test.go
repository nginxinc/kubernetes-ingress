package nginx

import (
	"testing"
)

func TestGetNginxCommand(t *testing.T) {
	tests := []struct {
		cmd             string
		nginxBinaryPath string
		expected        string
	}{
		{"reload", "/usr/sbin/nginx", "/usr/sbin/nginx -s reload"},
		{"stop", "/usr/sbin/nginx-debug", "/usr/sbin/nginx-debug -s stop"},
	}
	for _, test := range tests {
		t.Run(test.cmd, func(t *testing.T) {
			nginx := NewNginxController("/etc/nginx", test.nginxBinaryPath, true)

			if got := nginx.getNginxCommand(test.cmd); got != test.expected {
				t.Errorf("getNginxCommand returned \n%v, but expected \n%v", got, test.expected)
			}
		})
	}
}

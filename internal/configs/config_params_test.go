package configs

import (
	"context"
	"testing"
)

func TestNewDefaultConfigParamsUpstreamZoneSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		isPlus   bool
		expected string
	}{
		{
			isPlus:   false,
			expected: "256k",
		},
		{
			isPlus:   true,
			expected: "512k",
		},
	}

	for _, test := range tests {
		cfgParams := NewDefaultConfigParams(context.Background(), test.isPlus)
		if cfgParams == nil {
			t.Fatalf("NewDefaultConfigParams(context.Background(), %v) returned nil", test.isPlus)
		}

		if cfgParams.UpstreamZoneSize != test.expected {
			t.Errorf("NewDefaultConfigParams(context.Background(), %v) returned %s but expected %s", test.isPlus, cfgParams.UpstreamZoneSize, test.expected)
		}
	}
}

func TestParseConfigOnZoneSyncNotPresent(t *testing.T) {
	t.Parallel()
	// no zone sync k/v in the nginx-config
}

func TestParseConfigOnZoneSyncDisabled(t *testing.T) {
	t.Parallel()
	// explicit `false` for zone-sync
}

func TestParseConfigOnZoneSyncEnabledWithoutPort(t *testing.T) {
	t.Parallel()
	// explicit `true` for zone-sync
	// NIC applies default port
}

func TestParseConfigOnZoneSyncEnabledWithPort(t *testing.T) {
	t.Parallel()
	// explicit `true` for zone-sync
	// NIC applies specified port
}

package configs

import "testing"

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
		cfgParams := NewDefaultConfigParams(test.isPlus)
		if cfgParams == nil {
			t.Fatalf("NewDefaultConfigParams(%v) returned nil", test.isPlus)
		}

		if cfgParams.UpstreamZoneSize != test.expected {
			t.Errorf("NewDefaultConfigParams(%v) returned %s but expected %s", test.isPlus, cfgParams.UpstreamZoneSize, test.expected)
		}
	}
}

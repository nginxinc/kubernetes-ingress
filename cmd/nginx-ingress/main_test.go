package main

import (
	"testing"
)

func TestValidateStatusPort(t *testing.T) {
	badPorts := []int{80, 443, 1, 1022, 65536}
	for _, badPort := range badPorts {
		err := validateStatusPort(badPort)
		if err == nil {
			t.Errorf("Expected error for port %v\n", badPort)
		}
	}

	goodPorts := []int{8080, 8081, 8082, 1023, 65535}
	for _, goodPort := range goodPorts {
		err := validateStatusPort(goodPort)
		if err != nil {
			t.Errorf("Error for valid port:  %v err: %v\n", goodPort, err)
		}
	}

}

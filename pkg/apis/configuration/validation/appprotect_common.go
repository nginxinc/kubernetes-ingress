package validation

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ValidateRequiredSlices(obj *unstructured.Unstructured, fieldsList [][]string) error {
	for _, fields := range fieldsList {
		field, found, err := unstructured.NestedSlice(obj.Object, fields...)
		if err != nil {
			return fmt.Errorf("Error checking for required field %v: %w", field, err)
		}
		if !found {
			return fmt.Errorf("Required field %v not found", field)
		}
	}
	return nil
}

func ValidateRequiredFields(obj *unstructured.Unstructured, fieldsList [][]string) error {
	for _, fields := range fieldsList {
		field, found, err := unstructured.NestedMap(obj.Object, fields...)
		if err != nil {
			return fmt.Errorf("Error checking for required field %v: %w", field, err)
		}
		if !found {
			return fmt.Errorf("Required field %v not found", field)
		}
	}
	return nil
}

var (
	logDstEx = regexp.MustCompile(`(?:syslog:server=((?:\d{1,3}\.){3}\d{1,3}|localhost):\d{1,5})|stderr`)
)

// ValidateAppProtectLogDestination validates destination for log configuration
func ValidateAppProtectLogDestination(dstAntn string) error {
	errormsg := "Error parsing App Protect Log config: Destination must follow format: syslog:server=<ip-address | localhost>:<port> or stderr"
	if !logDstEx.MatchString(dstAntn) {
		return fmt.Errorf("%s Log Destination did not follow format", errormsg)
	}
	if dstAntn == "stderr" {
		return nil
	}

	dstchunks := strings.Split(dstAntn, ":")

	// // This error can be ignored since the regex check ensures this string will be parsable
	port, _ := strconv.Atoi(dstchunks[2])

	if port > 65535 || port < 1 {
		return fmt.Errorf("Error parsing port: %v not a valid port number", port)
	}

	ipstr := strings.Split(dstchunks[1], "=")[1]
	if ipstr == "localhost" {
		return nil
	}

	if net.ParseIP(ipstr) == nil {
		return fmt.Errorf("Error parsing host: %v is not a valid ip address", ipstr)
	}

	return nil
}

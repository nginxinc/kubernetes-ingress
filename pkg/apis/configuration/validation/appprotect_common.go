package validation

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ValidateRequiredSlices validates the required slices.
func ValidateRequiredSlices(obj *unstructured.Unstructured, fieldsList [][]string) error {
	for _, fields := range fieldsList {
		field, found, err := unstructured.NestedSlice(obj.Object, fields...)
		if err != nil {
			return fmt.Errorf("error checking for required field %v: %w", field, err)
		}
		if !found {
			return fmt.Errorf("required field %v not found", field)
		}
	}
	return nil
}

// ValidateRequiredFields validates the required fields.
func ValidateRequiredFields(obj *unstructured.Unstructured, fieldsList [][]string) error {
	for _, fields := range fieldsList {
		field, found, err := unstructured.NestedMap(obj.Object, fields...)
		if err != nil {
			return fmt.Errorf("error checking for required field %v: %w", field, err)
		}
		if !found {
			return fmt.Errorf("required field %v not found", field)
		}
	}
	return nil
}

var (
	logDstEx     = regexp.MustCompile(`(?:syslog:server=((?:\d{1,3}\.){3}\d{1,3}|localhost|[a-zA-Z0-9._-]+):\d{1,5})|stderr|(?:\/[\S]+)+`)
	logDstFileEx = regexp.MustCompile(`(?:\/[\S]+)+`)
	logDstFQDNEx = regexp.MustCompile(`(?:[a-zA-Z0-9_-]+\.)+[a-zA-Z0-9_-]+`)
)

// ValidateAppProtectLogDestination validates destination for log configuration
func ValidateAppProtectLogDestination(dstAntn string) error {
	errormsg := "error parsing App Protect Log config: Destination must follow format: syslog:server=<ip-address | localhost>:<port> or fqdn or stderr or absolute path to file"
	if !logDstEx.MatchString(dstAntn) {
		return fmt.Errorf("%s Log Destination did not follow format", errormsg)
	}
	if dstAntn == "stderr" {
		return nil
	}

	if logDstFileEx.MatchString(dstAntn) {
		return nil
	}

	dstchunks := strings.Split(dstAntn, ":")

	// This error can be ignored since the regex check ensures this string will be parsable
	port, _ := strconv.Atoi(dstchunks[2])

	if port > 65535 || port < 1 {
		return fmt.Errorf("error parsing port: %v not a valid port number", port)
	}

	ipstr := strings.Split(dstchunks[1], "=")[1]
	if ipstr == "localhost" {
		return nil
	}

	if logDstFQDNEx.MatchString(ipstr) {
		return nil
	}

	if net.ParseIP(ipstr) == nil {
		return fmt.Errorf("error parsing host: %v is not a valid ip address or host name", ipstr)
	}

	return nil
}

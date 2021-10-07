package validation

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var appProtectDosPolicyRequiredFields = [][]string{
	{"spec"},
}

var appProtectDosLogConfRequiredFields = [][]string{
	{"spec", "content"},
	{"spec", "filter"},
}

const MaxNameLength = 63

// ValidateAppProtectDosLogConf validates LogConfiguration resource
func ValidateAppProtectDosLogConf(logConf *unstructured.Unstructured) error {
	lcName := logConf.GetName()
	err := ValidateRequiredFields(logConf, appProtectDosLogConfRequiredFields)
	if err != nil {
		return fmt.Errorf("Error validating App Protect Dos Log Configuration %v: %w", lcName, err)
	}

	return nil
}

var accessLog = regexp.MustCompile(`^(((\d{1,3}\.){3}\d{1,3}):\d{1,5})$`)

// ValidateAppProtectDosAccessLogDest validates destination for access log configuration
func ValidateAppProtectDosAccessLogDest(accessLogDest string) error {
	errormsg := "Error parsing App Protect Dos Access Log Dest config: Destination must follow format: <ip-address>:<port>"
	if !accessLog.MatchString(accessLogDest) {
		return fmt.Errorf("%s Log Destination did not follow format", errormsg)
	}

	dstchunks := strings.Split(accessLogDest, ":")

	// This error can be ignored since the regex check ensures this string will be parsable
	port, _ := strconv.Atoi(dstchunks[1])

	if port > 65535 || port < 1 {
		return fmt.Errorf("Error parsing port: %v not a valid port number", port)
	}

	if net.ParseIP(dstchunks[0]) == nil {
		return fmt.Errorf("Error parsing host: %v is not a valid ip address", dstchunks[0])
	}

	return nil
}

// ValidateAppProtectDosName validates name of App Protect Dos
func ValidateAppProtectDosName(name string) error {
	if len(name) > MaxNameLength {
		return fmt.Errorf("App Protect Dos Name max length is %v", MaxNameLength)
	}

	return nil
}

// ValidateAppProtectDosMonitor validates monitor value of App Protect Dos
func ValidateAppProtectDosMonitor(monitor string) error {
	_, err := url.Parse(monitor)
	if err != nil {
		return fmt.Errorf("App Protect Dos Monitor must have valid URL")
	}

	return nil
}

// ValidateAppProtectDosPolicy validates Policy resource
func ValidateAppProtectDosPolicy(policy *unstructured.Unstructured) error {
	polName := policy.GetName()

	err := ValidateRequiredFields(policy, appProtectDosPolicyRequiredFields)
	if err != nil {
		return fmt.Errorf("Error validating App Protect Dos Policy %v: %w", polName, err)
	}

	return nil
}

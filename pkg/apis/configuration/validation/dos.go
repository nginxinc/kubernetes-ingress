package validation

import (
	"fmt"
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

var (
	validDnsRegex       = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9-]{1,62}\.)([A-Za-z0-9-]{1,63}\.)*[A-Za-z]{2,6}:\d{1,5}$`)
	validIpRegex        = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}:\d{1,5}$`)
	validLocalhostRegex = regexp.MustCompile(`^localhost:\d{1,5}$`)
)

// ValidateAppProtectDosLogDest validates destination for log configuration
func ValidateAppProtectDosLogDest(dstAntn string) error {
	if validIpRegex.MatchString(dstAntn) || validDnsRegex.MatchString(dstAntn) || validLocalhostRegex.MatchString(dstAntn) {
		chunks := strings.Split(dstAntn, ":")
		err := validatePort(chunks[1])
		if err != nil {
			return fmt.Errorf("invalid log destination: %w", err)
		}
		return nil
	}
	if dstAntn == "stderr" {
		return nil
	}

	return fmt.Errorf("invalid log destination: %s, must follow format: <ip-address | localhost | dns name>:<port> or stderr", dstAntn)
}

func validatePort(value string) error {
	port, _ := strconv.Atoi(value)
	if port > 65535 || port < 1 {
		return fmt.Errorf("error parsing port: %v not a valid port number", port)
	}
	return nil
}

// ValidateAppProtectDosName validates name of App Protect Dos
func ValidateAppProtectDosName(name string) error {
	if len(name) > MaxNameLength {
		return fmt.Errorf("App Protect Dos Name max length is %v", MaxNameLength)
	}

	if err := validateEscapedString(name, "protected-object-one"); err != nil {
		return err
	}

	return nil
}

// ValidateAppProtectDosMonitor validates monitor value of App Protect Dos
func ValidateAppProtectDosMonitor(monitor string) error {
	_, err := url.Parse(monitor)
	if err != nil {
		return fmt.Errorf("App Protect Dos Monitor must have valid URL")
	}

	if err := validateEscapedString(monitor, "http://www.example.com"); err != nil {
		return err
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

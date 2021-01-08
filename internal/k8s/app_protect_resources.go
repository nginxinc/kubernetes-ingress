package k8s

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var appProtectPolicyRequiredFields = [][]string{
	{"spec", "policy"},
}

var appProtectLogConfRequiredFields = [][]string{
	{"spec", "content"},
	{"spec", "filter"},
}

var appProtectUserSigRequiredSlices = [][]string{
	{"spec", "signatures"},
}

func validateRequiredFields(policy *unstructured.Unstructured, fieldsList [][]string) error {
	for _, fields := range fieldsList {
		field, found, err := unstructured.NestedMap(policy.Object, fields...)
		if err != nil {
			return fmt.Errorf("Error checking for required field %v: %v", field, err)
		}
		if !found {
			return fmt.Errorf("Required field %v not found", field)
		}
	}
	return nil
}

func validateRequiredSlices(policy *unstructured.Unstructured, fieldsList [][]string) error {
	for _, fields := range fieldsList {
		field, found, err := unstructured.NestedSlice(policy.Object, fields...)
		if err != nil {
			return fmt.Errorf("Error checking for required field %v: %v", field, err)
		}
		if !found {
			return fmt.Errorf("Required field %v not found", field)
		}
	}
	return nil
}

func validateRequiredStrings(policy *unstructured.Unstructured, fieldsList [][]string) error {
	for _, fields := range fieldsList {
		field, found, err := unstructured.NestedString(policy.Object, fields...)
		if err != nil {
			return fmt.Errorf("Error checking for required field %v: %v", field, err)
		}
		if !found {
			return fmt.Errorf("Required field %v not found", field)
		}
	}
	return nil
}

// ValidateAppProtectPolicy validates Policy resource
func ValidateAppProtectPolicy(policy *unstructured.Unstructured) error {
	polName := policy.GetName()

	err := validateRequiredFields(policy, appProtectPolicyRequiredFields)
	if err != nil {
		return fmt.Errorf("Error validating App Protect Policy %v: %v", polName, err)
	}

	return nil
}

// ValidateAppProtectLogConf validates LogConfiguration resource
func ValidateAppProtectLogConf(logConf *unstructured.Unstructured) error {
	lcName := logConf.GetName()
	err := validateRequiredFields(logConf, appProtectLogConfRequiredFields)
	if err != nil {
		return fmt.Errorf("Error validating App Protect Log Configuration %v: %v", lcName, err)
	}

	return nil
}

var logDstEx = regexp.MustCompile(`(?:syslog:server=((?:\d{1,3}\.){3}\d{1,3}|localhost):\d{1,5})|stderr|(?:\/[\S]+)+`)
var logDstFileEx = regexp.MustCompile(`(?:\/[\S]+)+`)

// ValidateAppProtectLogDestinationAnnotation validates annotation for log destination configuration
func ValidateAppProtectLogDestinationAnnotation(dstAntn string) error {
	errormsg := "Error parsing App Protect Log config: Destination Annotation must follow format: syslog:server=<ip-address | localhost>:<port> or stderr or absolute path to file"
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

	// This error can be ingored since the regex check ensures this string will be parsable
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

// ParseResourceReferenceAnnotation returns a namespace/name string
func ParseResourceReferenceAnnotation(ns, antn string) string {
	if !strings.Contains(antn, "/") {
		return ns + "/" + antn
	}
	return antn
}

func validateAppProtectUserSig(userSig *unstructured.Unstructured) error {
	sigName := userSig.GetName()
	err := validateRequiredSlices(userSig, appProtectUserSigRequiredSlices)
	if err != nil {
		return fmt.Errorf("Error validating App Protect User Signature %v: %v", sigName, err)
	}

	return nil
}

func getNsName(obj *unstructured.Unstructured) string {
	return obj.GetNamespace() + "/" + obj.GetName()
}

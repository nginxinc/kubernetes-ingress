package configs

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// There seems to be no composite interface in the kubernetes api package,
// so we have to declare our own.
type apiObject interface {
	v1.Object
	runtime.Object
}

// GetMapKeyAsBool searches the map for the given key and parses the key as bool.
func GetMapKeyAsBool(m map[string]string, key string, context apiObject) (bool, bool, error) {
	if str, exists := m[key]; exists {
		b, err := strconv.ParseBool(str)
		if err != nil {
			return false, exists, fmt.Errorf("%s %v/%v '%s' contains invalid bool: %v, ignoring", context.GetObjectKind().GroupVersionKind().Kind, context.GetNamespace(), context.GetName(), key, err)
		}

		return b, exists, nil
	}

	return false, false, nil
}

// GetMapKeyAsInt tries to find and parse a key in a map as int.
func GetMapKeyAsInt(m map[string]string, key string, context apiObject) (int, bool, error) {
	if str, exists := m[key]; exists {
		i, err := strconv.Atoi(str)
		if err != nil {
			return 0, exists, fmt.Errorf("%s %v/%v '%s' contains invalid integer: %v, ignoring", context.GetObjectKind().GroupVersionKind().Kind, context.GetNamespace(), context.GetName(), key, err)
		}

		return i, exists, nil
	}

	return 0, false, nil
}

// GetMapKeyAsInt64 tries to find and parse a key in a map as int64.
func GetMapKeyAsInt64(m map[string]string, key string, context apiObject) (int64, bool, error) {
	if str, exists := m[key]; exists {
		i, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return 0, exists, fmt.Errorf("%s %v/%v '%s' contains invalid integer: %v, ignoring", context.GetObjectKind().GroupVersionKind().Kind, context.GetNamespace(), context.GetName(), key, err)
		}

		return i, exists, nil
	}

	return 0, false, nil
}

// GetMapKeyAsUint64 tries to find and parse a key in a map as uint64.
func GetMapKeyAsUint64(m map[string]string, key string, context apiObject, nonZero bool) (uint64, bool, error) {
	if str, exists := m[key]; exists {
		i, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return 0, exists, fmt.Errorf("%s %v/%v '%s' contains invalid uint64: %v, ignoring", context.GetObjectKind().GroupVersionKind().Kind, context.GetNamespace(), context.GetName(), key, err)
		}

		if nonZero && i == 0 {
			return 0, exists, fmt.Errorf("%s %v/%v '%s' must be greater than 0, ignoring", context.GetObjectKind().GroupVersionKind().Kind, context.GetNamespace(), context.GetName(), key)
		}

		return i, exists, nil
	}

	return 0, false, nil
}

// GetMapKeyAsStringSlice tries to find and parse a key in the map as string slice splitting it on delimiter.
func GetMapKeyAsStringSlice(m map[string]string, key string, context apiObject, delimiter string) ([]string, bool, error) {
	if str, exists := m[key]; exists {
		slice := strings.Split(str, delimiter)
		return slice, exists, nil
	}

	return nil, false, nil
}

// ParseLBMethod parses method and matches it to a corresponding load balancing method in NGINX. An error is returned if method is not valid.
func ParseLBMethod(method string) (string, error) {
	method = strings.TrimSpace(method)

	if method == "round_robin" {
		return "", nil
	}

	if strings.HasPrefix(method, "hash") {
		method, err := validateHashLBMethod(method)
		return method, err
	}

	if _, exists := nginxLBValidInput[method]; exists {
		return method, nil
	}

	return "", fmt.Errorf("Invalid load balancing method: %q", method)
}

var nginxLBValidInput = map[string]bool{
	"least_conn":            true,
	"ip_hash":               true,
	"random":                true,
	"random two":            true,
	"random two least_conn": true,
}

var nginxPlusLBValidInput = map[string]bool{
	"least_conn":                      true,
	"ip_hash":                         true,
	"random":                          true,
	"random two":                      true,
	"random two least_conn":           true,
	"random two least_time=header":    true,
	"random two least_time=last_byte": true,
	"least_time header":               true,
	"least_time last_byte":            true,
	"least_time header inflight":      true,
	"least_time last_byte inflight":   true,
}

// ParseLBMethodForPlus parses method and matches it to a corresponding load balancing method in NGINX Plus. An error is returned if method is not valid.
func ParseLBMethodForPlus(method string) (string, error) {
	method = strings.TrimSpace(method)

	if method == "round_robin" {
		return "", nil
	}

	if strings.HasPrefix(method, "hash") {
		method, err := validateHashLBMethod(method)
		return method, err
	}

	if _, exists := nginxPlusLBValidInput[method]; exists {
		return method, nil
	}

	return "", fmt.Errorf("Invalid load balancing method: %q", method)
}

func validateHashLBMethod(method string) (string, error) {
	keyWords := strings.Split(method, " ")

	if keyWords[0] == "hash" {
		if len(keyWords) == 2 || len(keyWords) == 3 && keyWords[2] == "consistent" {
			return method, nil
		}
	}

	return "", fmt.Errorf("Invalid load balancing method: %q", method)
}

// http://nginx.org/en/docs/syntax.html
var validTimeSuffixes = []string{
	"ms",
	"s",
	"m",
	"h",
	"d",
	"w",
	"M",
	"y",
}

var durationEscaped = strings.Join(validTimeSuffixes, "|")
var validNginxTime = regexp.MustCompile(`^([0-9]+([` + durationEscaped + `]?){0,1} *)+$`)

// ParseTime ensures that the string value in the annotation is a valid time.
func ParseTime(s string) (string, error) {
	s = strings.TrimSpace(s)

	if validNginxTime.MatchString(s) {
		return s, nil
	}
	return "", errors.New("Invalid time string")
}

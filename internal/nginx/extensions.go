package nginx

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ParseLBMethod parses method and matches it to a corresponding load balancing method in NGINX. An error is returned if method is not valid
func ParseLBMethod(method string) (string, error) {
	method = strings.TrimSpace(method)
	if method == "round_robin" {
		return "", nil
	}
	if strings.HasPrefix(method, "hash") {
		method, err := validateHashLBMethod(method)
		return method, err
	}

	if method == "least_conn" || method == "ip_hash" {
		return method, nil
	}
	return "", fmt.Errorf("Invalid load balancing method: %q", method)
}

var nginxPlusLBValidInput = map[string]bool{
	"least_time":                    true,
	"last_byte":                     true,
	"least_conn":                    true,
	"ip_hash":                       true,
	"least_time header":             true,
	"least_time last_byte":          true,
	"least_time header inflight":    true,
	"least_time last_byte inflight": true,
}

// ParseLBMethodForPlus parses method and matches it to a corresponding load balancing method in NGINX Plus. An error is returned if method is not valid
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

// ParseSlowStart ensures that the slow_start value in the annotation is valid.
func ParseSlowStart(s string) (string, error) {
	s = strings.TrimSpace(s)

	if validNginxTime.MatchString(s) {
		return s, nil
	}
	return "", errors.New("Invalid time string")
}

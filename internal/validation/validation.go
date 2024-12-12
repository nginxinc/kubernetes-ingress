package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	validDNSRegex       = regexp.MustCompile(`^(?:[A-Za-z0-9][A-Za-z0-9-]{1,62}\.)([A-Za-z0-9-]{1,63}\.)*[A-Za-z]{2,6}(?::\d{1,5})?$`)
	validIPRegex        = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}(?::\d{1,5})?$`)
	validLocalhostRegex = regexp.MustCompile(`^localhost(?::\d{1,5})?$`)
)

func ValidatePort(value string) error {
	port, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("error parsing port number: %w", err)
	}
	if port > 65535 || port < 1 {
		return fmt.Errorf("error parsing port: %v not a valid port number", port)
	}
	return nil
}

func ValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("error parsing host: empty host")
	}

	if validIPRegex.MatchString(host) || validDNSRegex.MatchString(host) || validLocalhostRegex.MatchString(host) {
		chunks := strings.Split(host, ":")
		err := ValidatePort(chunks[1])
		if err != nil {
			return fmt.Errorf("invalid port: %w", err)
		}
		return nil
	}
	return fmt.Errorf("invalid host: %s", host)
}

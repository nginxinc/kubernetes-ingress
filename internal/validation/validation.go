package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	validDNSRegex      = regexp.MustCompile(`^(?:[A-Za-z0-9][A-Za-z0-9-]{1,62}\.)([A-Za-z0-9-]{1,63}\.)*[A-Za-z]{2,6}(?::\d{1,5})?$`)
	validIPRegex       = regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])(?::\d{1,5})?$`)
	validHostnameRegex = regexp.MustCompile(`^[a-z][A-Za-z0-9-]{1,62}(?::\d{1,5})?$`)
)

// ValidatePort ensure port matches rfc6335 https://www.rfc-editor.org/rfc/rfc6335.html
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

// ValidateHost ensures the host is a valid hostname/IP address or FQDN with an optional :port appended
func ValidateHost(host string) error {
	if host == "" {
		return fmt.Errorf("error parsing host: empty host")
	}

	if validIPRegex.MatchString(host) || validDNSRegex.MatchString(host) || validHostnameRegex.MatchString(host) {
		chunks := strings.Split(host, ":")
		if len(chunks) > 1 {
			err := ValidatePort(chunks[1])
			if err != nil {
				return fmt.Errorf("invalid port: %w", err)
			}
		}
		return nil
	}
	return fmt.Errorf("error parsing host: %s not a valid host", host)
}

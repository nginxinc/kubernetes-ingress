package validation

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slices"

	v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/externaldns/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// ValidateDNSEnpoint validates if all DNSEndpoint fields are valid.
func ValidateDNSEndpoint(dnsendpoint *v1.DNSEndpoint) error {
	if err := validateDNSEndpointSpec(&dnsendpoint.Spec); err != nil {
		return err
	}
	return nil
}

func validateDNSEndpointSpec(es *v1.DNSEndpointSpec) error {
	if len(es.Endpoints) == 0 {
		return fmt.Errorf("%w: no endpoints supplied, expected a list of endpoints", ErrTypeRequired)
	}
	for _, endpoint := range es.Endpoints {
		if err := validateEndpoint(endpoint); err != nil {
			return err
		}
	}
	return nil
}

func validateEndpoint(e *v1.Endpoint) error {
	if err := validateDNSName(e.DNSName); err != nil {
		return err
	}
	if err := validateTargets(e.Targets); err != nil {
		return err
	}
	if err := validateDNSRecordType(e.RecordType); err != nil {
		return err
	}
	if err := validateTTL(e.RecordTTL); err != nil {
		return err
	}
	return nil
}

func validateDNSName(name string) error {
	if issues := validation.IsDNS1123Subdomain(name); len(issues) > 0 {
		return fmt.Errorf("%w: name %s, %s", ErrTypeInvalid, name, strings.Join(issues, ", "))
	}
	return nil
}

func validateTargets(targets v1.Targets) error {
	for _, target := range targets {
		if err := isFullyQualifiedDomainName(target); err != nil {
			return fmt.Errorf("%w: target %q is invalid, it should be a valid IP address or hostname", ErrTypeInvalid, target)
		}
	}
	return isUnique(targets)
}

func isUnique(targets v1.Targets) error {
	occured := make(map[string]bool)
	for _, target := range targets {
		if occured[target] {
			return fmt.Errorf("%w: target %s, expected unique targets", ErrTypeDuplicated, target)
		}
		occured[target] = true
	}
	return nil
}

func validateDNSRecordType(record string) error {
	if !slices.Contains(validRecords, record) {
		return fmt.Errorf("%w: record %s, %s", ErrTypeNotSupported, record, strings.Join(validRecords, ","))
	}
	return nil
}

func validateTTL(ttl v1.TTL) error {
	if ttl <= 0 {
		return fmt.Errorf("%w: ttl %d, ttl value should be > 0", ErrTypeNotInRange, ttl)
	}
	return nil
}

func isFullyQualifiedDomainName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: name not provided", ErrTypeInvalid)
	}
	name = strings.TrimSuffix(name, ".")
	if issues := validation.IsDNS1123Subdomain(name); len(issues) > 0 {
		return fmt.Errorf("%w: name %s is not valid subdomain, %s", ErrTypeInvalid, name, strings.Join(issues, ", "))
	}
	if len(strings.Split(name, ".")) < 2 {
		return fmt.Errorf("%w: name %s should be a domain with at least two segments separated by dots", ErrTypeInvalid, name)
	}
	for _, label := range strings.Split(name, ".") {
		if issues := validation.IsDNS1123Label(label); len(issues) > 0 {
			return fmt.Errorf("%w: label %s should conform to the definition of label in DNS (RFC1123), %s", ErrTypeInvalid, label, strings.Join(issues, ", "))
		}
	}
	return nil
}

var (
	// validRecords represents allowed DNS record names
	//
	// NGINX ingress controller at the moment supports
	// a subset of DNS record types listed in the external-dns project.
	validRecords = []string{"A", "CNAME"}

	// validation error types based on k8s validators
	ErrTypeNotSupported = errors.New("type not supported")
	ErrTypeInvalid      = errors.New("type invalid")
	ErrTypeDuplicated   = errors.New("type duplicated")
	ErrTypeRequired     = errors.New("type required")
	ErrTypeNotInRange   = errors.New("type not in range")
)

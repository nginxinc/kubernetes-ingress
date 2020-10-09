package k8s

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

// JWTKeyKey is the key of the data field of a Secret where the JWK must be stored.
const JWTKeyKey = "jwk"

// CAKey is the key of the data field of a Secret where the certificate authority must be stored.
const CAKey = "ca.crt"

const (
	// TLS Secret
	TLS = iota + 1
	// JWK Secret
	JWK
	// CA Secret
	CA
)

// ValidateTLSSecret validates the secret. If it is valid, the function returns nil.
func ValidateTLSSecret(secret *v1.Secret) error {
	_, certExists := secret.Data[v1.TLSCertKey]
	_, keyExists := secret.Data[v1.TLSPrivateKeyKey]

	if certExists && keyExists {
		return nil
	}
	if !certExists {
		return fmt.Errorf("Secret doesn't have %v", v1.TLSCertKey)
	}

	if !keyExists {
		return fmt.Errorf("Secret doesn't have %v", v1.TLSPrivateKeyKey)
	}

	return nil
}

// ValidateJWKSecret validates the secret. If it is valid, the function returns nil.
func ValidateJWKSecret(secret *v1.Secret) error {
	if _, exists := secret.Data[JWTKeyKey]; !exists {
		return fmt.Errorf("Secret doesn't have %v", JWTKeyKey)
	}

	return nil
}

// ValidateCASecret validates the secret. If it is valid, the function returns nil.
func ValidateCASecret(secret *v1.Secret) error {
	if _, exists := secret.Data[CAKey]; !exists {
		return fmt.Errorf("Secret doesn't have %v", CAKey)
	}

	return nil
}

// GetSecretKind returns the kind of the Secret.
func GetSecretKind(secret *v1.Secret) (int, error) {
	if err := ValidateTLSSecret(secret); err == nil {
		return TLS, nil
	}
	if err := ValidateJWKSecret(secret); err == nil {
		return JWK, nil
	}
	if err := ValidateCASecret(secret); err == nil {
		return CA, nil
	}

	return 0, fmt.Errorf("Unknown Secret")
}

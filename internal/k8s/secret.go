package k8s

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

// JWTKeyKey is the key of the data field of a Secret where the JWK must be stored.
const JWTKeyKey = "jwk"

const (
	// TLS Secret
	TLS = iota
	// JWK Secret
	JWK
)

// ValidateTLSSecret validates the secret. If it is valid, the function returns nil.
func ValidateTLSSecret(secret *v1.Secret) error {
	if _, exists := secret.Data[v1.TLSCertKey]; !exists {
		return fmt.Errorf("Secret doesn't have %v", v1.TLSCertKey)
	}

	if _, exists := secret.Data[v1.TLSPrivateKeyKey]; !exists {
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

// GetSecretKind returns the kind of the Secret.
func GetSecretKind(secret *v1.Secret) (int, error) {
	if err := ValidateTLSSecret(secret); err == nil {
		return TLS, nil
	}
	if err := ValidateJWKSecret(secret); err == nil {
		return JWK, nil
	}

	return 0, fmt.Errorf("Unknown Secret")
}

package nginx

import (
	"fmt"

	api_v1 "k8s.io/api/core/v1"
)

const (
	// TLS Secret
	TLS = iota
	// JWK Secret
	JWK
)

// ValidateTLSSecret validates the secret. If it is valid, the function returns nil.
func ValidateTLSSecret(secret *api_v1.Secret) error {
	if _, exists := secret.Data[api_v1.TLSCertKey]; !exists {
		return fmt.Errorf("Secret doesn't have %v", api_v1.TLSCertKey)
	}

	if _, exists := secret.Data[api_v1.TLSPrivateKeyKey]; !exists {
		return fmt.Errorf("Secret doesn't have %v", api_v1.TLSPrivateKeyKey)
	}

	return nil
}

// ValidateJWKSecret validates the secret. If it is valid, the function returns nil.
func ValidateJWKSecret(secret *api_v1.Secret) error {
	if _, exists := secret.Data[JWTKey]; !exists {
		return fmt.Errorf("Secret doesn't have %v", JWTKey)
	}

	return nil
}

// GetSecretKind returns the kind of the Secret.
func GetSecretKind(secret *api_v1.Secret) (int, error) {
	if err := ValidateTLSSecret(secret); err == nil {
		return TLS, nil
	}
	if err := ValidateJWKSecret(secret); err == nil {
		return JWK, nil
	}

	return 0, fmt.Errorf("Unknown Secret")
}

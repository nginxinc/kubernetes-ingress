package k8s

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

// JWTKeyKey is the key of the data field of a Secret where the JWK must be stored.
const JWTKeyKey = "jwk"

// CAKey is the key of the data field of a Secret where the certificate authority must be stored.
const CAKey = "ca.crt"

// SecretTypeCA contains a certificate authority for TLS certificate verification.
const SecretTypeCA v1.SecretType = "nginx.org/ca"

// SecretTypeJWK contains a JWK (JSON Web Key) for validating JWTs (JSON Web Tokens).
const SecretTypeJWK v1.SecretType = "nginx.org/jwk"

// ValidateTLSSecret validates the secret. If it is valid, the function returns nil.
func ValidateTLSSecret(secret *v1.Secret) error {
	if secret.Type != v1.SecretTypeTLS {
		return fmt.Errorf("TLS Secret must be of the type %v", v1.SecretTypeTLS)
	}

	// Kubernetes ensures that 'tls.crt' and 'tls.key' are present for secrets of v1.SecretTypeTLS type
	// no need to validate that

	return nil
}

// ValidateJWKSecret validates the secret. If it is valid, the function returns nil.
func ValidateJWKSecret(secret *v1.Secret) error {
	if secret.Type != SecretTypeJWK {
		return fmt.Errorf("JWK secret must be of the type %v", SecretTypeJWK)
	}

	if _, exists := secret.Data[JWTKeyKey]; !exists {
		return fmt.Errorf("JWK secret must have the data field %v", JWTKeyKey)
	}

	return nil
}

// ValidateCASecret validates the secret. If it is valid, the function returns nil.
func ValidateCASecret(secret *v1.Secret) error {
	if secret.Type != SecretTypeCA {
		return fmt.Errorf("CA secret must be of the type %v", SecretTypeCA)
	}

	if _, exists := secret.Data[CAKey]; !exists {
		return fmt.Errorf("CA secret must have the data field %v", CAKey)
	}

	return nil
}

// IsSupportedSecretType checks if the secret type is supported.
func IsSupportedSecretType(secretType v1.SecretType) bool {
	return secretType == v1.SecretTypeTLS || secretType == SecretTypeCA || secretType == SecretTypeJWK
}

// ValidateSecret validates the secret. If it is valid, the function returns nil.
func ValidateSecret(secret *v1.Secret) error {
	switch secret.Type {
	case v1.SecretTypeTLS:
		return ValidateTLSSecret(secret)
	case SecretTypeJWK:
		return ValidateJWKSecret(secret)
	case SecretTypeCA:
		return ValidateCASecret(secret)
	}

	return fmt.Errorf("Secret is of the unsupported type %v", secret.Type)
}

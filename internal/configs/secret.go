package configs

import api_v1 "k8s.io/api/core/v1"

// The constants below are defined because we can't use the same constants from the internal/k8s package
// because that will lead to import cycles
// TO-DO: Consider refactoring the internal/k8s package, so that we only define those constants once

// SecretTypeCA contains a certificate authority for TLS certificate verification.
const SecretTypeCA api_v1.SecretType = "nginx.org/ca"

// SecretTypeJWK contains a JWK (JSON Web Key) for validating JWTs (JSON Web Tokens).
const SecretTypeJWK api_v1.SecretType = "nginx.org/jwk"

// SecretReference holds a reference to a secret stored on the file system.
type SecretReference struct {
	Type  api_v1.SecretType
	Path  string
	Error error
}

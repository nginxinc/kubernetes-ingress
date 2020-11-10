package k8s

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateJWKSecret(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "jwk-secret",
			Namespace: "default",
		},
		Type: SecretTypeJWK,
		Data: map[string][]byte{
			"jwk": nil,
		},
	}

	err := ValidateJWKSecret(secret)
	if err != nil {
		t.Errorf("ValidateJWKSecret() returned error %v", err)
	}
}

func TestValidateJWKSecretFails(t *testing.T) {
	tests := []struct {
		secret *v1.Secret
		msg    string
	}{
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "jwk-secret",
					Namespace: "default",
				},
				Type: "some-type",
				Data: map[string][]byte{
					"jwk": nil,
				},
			},
			msg: "Incorrect type for JWK secret",
		},
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "jwk-secret",
					Namespace: "default",
				},
				Type: SecretTypeJWK,
			},
			msg: "Missing jwk for JWK secret",
		},
	}

	for _, test := range tests {
		err := ValidateJWKSecret(test.secret)
		if err == nil {
			t.Errorf("ValidateJWKSecret() returned no error for the case of %s", test.msg)
		}
	}
}

func TestValidateCASecret(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "ingress-mtls-secret",
			Namespace: "default",
		},
		Type: SecretTypeCA,
		Data: map[string][]byte{
			"ca.crt": nil,
		},
	}

	err := ValidateCASecret(secret)
	if err != nil {
		t.Errorf("ValidateCASecret() returned error %v", err)
	}
}

func TestValidateCASecretFails(t *testing.T) {
	tests := []struct {
		secret *v1.Secret
		msg    string
	}{
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "ingress-mtls-secret",
					Namespace: "default",
				},
				Type: "some-type",
				Data: map[string][]byte{
					"ca.crt": nil,
				},
			},
			msg: "Incorrect type for CA secret",
		},
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "ingress-mtls-secret",
					Namespace: "default",
				},
				Type: SecretTypeCA,
			},
			msg: "Missing ca.crt for CA secret",
		},
	}

	for _, test := range tests {
		err := ValidateCASecret(test.secret)
		if err == nil {
			t.Errorf("ValidateCASecret() returned no error for the case of %s", test.msg)
		}
	}
}

func TestValidateTLSSecret(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tls-secret",
			Namespace: "default",
		},
		Type: v1.SecretTypeTLS,
	}

	err := ValidateTLSSecret(secret)
	if err != nil {
		t.Errorf("ValidateTLSSecret() returned error %v", err)
	}
}

func TestValidateTLSSecretFails(t *testing.T) {
	secret := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tls-secret",
			Namespace: "default",
		},
		Type: "some type",
	}

	err := ValidateTLSSecret(secret)
	if err == nil {
		t.Errorf("ValidateTLSSecret() returned no error")
	}
}

func TestValidateSecret(t *testing.T) {
	tests := []struct {
		secret *v1.Secret
		msg    string
	}{
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "tls-secret",
					Namespace: "default",
				},
				Type: v1.SecretTypeTLS,
			},
			msg: "Valid TLS secret",
		},
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "ingress-mtls-secret",
					Namespace: "default",
				},
				Type: SecretTypeCA,
				Data: map[string][]byte{
					"ca.crt": nil,
				},
			},
			msg: "Valid CA secret",
		}, {
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "jwk-secret",
					Namespace: "default",
				},
				Type: SecretTypeJWK,
				Data: map[string][]byte{
					"jwk": nil,
				},
			},
			msg: "Valid JWK secret",
		},
	}

	for _, test := range tests {
		err := ValidateSecret(test.secret)
		if err != nil {
			t.Errorf("ValidateSecret() returned error %v for the case of %s", err, test.msg)
		}
	}
}

func TestValidateSecretFails(t *testing.T) {
	tests := []struct {
		secret *v1.Secret
		msg    string
	}{
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "tls-secret",
					Namespace: "default",
				},
			},
			msg: "Missing type for TLS secret",
		},
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "ingress-mtls-secret",
					Namespace: "default",
				},
				Type: SecretTypeCA,
			},
			msg: "Missing ca.crt for CA secret",
		}, {
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "jwk-secret",
					Namespace: "default",
				},
				Type: SecretTypeJWK,
			},
			msg: "Missing jwk for JWK secret",
		},
	}

	for _, test := range tests {
		err := ValidateSecret(test.secret)
		if err == nil {
			t.Errorf("ValidateSecret() returned no error for the case of %s", test.msg)
		}
	}
}

func TestHasCorrectSecretType(t *testing.T) {
	tests := []struct {
		secretType v1.SecretType
		expected   bool
	}{
		{
			secretType: v1.SecretTypeTLS,
			expected:   true,
		},
		{
			secretType: SecretTypeCA,
			expected:   true,
		},
		{
			secretType: SecretTypeJWK,
			expected:   true,
		},
		{
			secretType: "some-type",
			expected:   false,
		},
	}

	for _, test := range tests {
		result := IsSupportedSecretType(test.secretType)
		if result != test.expected {
			t.Errorf("IsSupportedSecretType(%v) returned %v but expected %v", test.secretType, result, test.expected)
		}
	}
}

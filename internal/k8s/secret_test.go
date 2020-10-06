package k8s

import (
	"errors"
	"testing"

	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSecretKind(t *testing.T) {

	var tests = []struct {
		secret   *v1.Secret
		expected int
	}{
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "tls-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"tls.key": nil,
					"tls.crt": nil,
				},
			},
			expected: TLS,
		},
		{
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "ingress-mtls-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"ca.crt": nil,
				},
			},
			expected: IngressMTLS,
		}, {
			secret: &v1.Secret{
				ObjectMeta: meta_v1.ObjectMeta{
					Name:      "jwk-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"jwk": nil,
				},
			},
			expected: JWK,
		},
	}

	for _, test := range tests {

		secret, err := GetSecretKind(test.secret)
		if err != nil {
			t.Errorf("GetSecretKind() returned an unexpected error: %v", err)
		}
		if secret != test.expected {
			t.Errorf("GetSecretKind() return %v but expected %v", secret, test.expected)
		}
	}
}

func TestGetSecretKindUnkown(t *testing.T) {
	s := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "foo-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"foo.bar": nil,
		},
	}
	e := errors.New("Unknown Secret")

	secret, err := GetSecretKind(s)
	if secret != 0 {
		t.Errorf("GetSecretKind() returned an unexpected secret: %v", secret)
	}
	if err.Error() != e.Error() {
		t.Errorf("GetSecretKind() return %v but expected %v", err, e)
	}

}

func TestValidateTLSSecretFail(t *testing.T) {

	s := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "tls-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"tls.crt": nil,
		},
	}
	e := errors.New("Secret doesn't have tls.key")

	err := ValidateTLSSecret(s)
	if err.Error() != e.Error() {
		t.Errorf("ValidateTLSSecret() return %v but expected %v", err, e)
	}
}

func TestValidateIngressMTLSSecretFail(t *testing.T) {

	s := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "mtls-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.cert": nil,
		},
	}
	e := errors.New("Secret doesn't have ca.crt")

	err := ValidateIngressMTLSSecret(s)
	if err.Error() != e.Error() {
		t.Errorf("ValidateIngressMTLSSecret() return %v but expected %v", err, e)
	}
}

func TestValidateJWKSecretFail(t *testing.T) {

	s := &v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "mtls-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"jwka": nil,
		},
	}
	e := errors.New("Secret doesn't have jwk")

	err := ValidateJWKSecret(s)
	if err.Error() != e.Error() {
		t.Errorf("ValidateJWKSecret() return %v but expected %v", err, e)
	}
}

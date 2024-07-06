package k8s

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nginxinc/kubernetes-ingress/internal/k8s/secrets"
	conf_v1 "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func TestFindPoliciesForSecret(t *testing.T) {
	t.Parallel()
	jwtPol1 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "jwt-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			JWTAuth: &conf_v1.JWTAuth{
				Secret: "jwk-secret",
			},
		},
	}

	jwtPol2 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "jwt-policy",
			Namespace: "ns-1",
		},
		Spec: conf_v1.PolicySpec{
			JWTAuth: &conf_v1.JWTAuth{
				Secret: "jwk-secret",
			},
		},
	}

	basicPol1 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "basic-auth-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			BasicAuth: &conf_v1.BasicAuth{
				Secret: "basic-auth-secret",
			},
		},
	}

	basicPol2 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "basic-auth-policy",
			Namespace: "ns-1",
		},
		Spec: conf_v1.PolicySpec{
			BasicAuth: &conf_v1.BasicAuth{
				Secret: "basic-auth-secret",
			},
		},
	}

	ingTLSPol := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "ingress-mtls-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			IngressMTLS: &conf_v1.IngressMTLS{
				ClientCertSecret: "ingress-mtls-secret",
			},
		},
	}
	egTLSPol := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "egress-mtls-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			EgressMTLS: &conf_v1.EgressMTLS{
				TLSSecret: "egress-mtls-secret",
			},
		},
	}
	egTLSPol2 := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "egress-trusted-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			EgressMTLS: &conf_v1.EgressMTLS{
				TrustedCertSecret: "egress-trusted-secret",
			},
		},
	}
	oidcPol := &conf_v1.Policy{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "oidc-policy",
			Namespace: "default",
		},
		Spec: conf_v1.PolicySpec{
			OIDC: &conf_v1.OIDC{
				ClientSecret: "oidc-secret",
			},
		},
	}

	tests := []struct {
		policies        []*conf_v1.Policy
		secretNamespace string
		secretName      string
		expected        []*conf_v1.Policy
		msg             string
	}{
		{
			policies:        []*conf_v1.Policy{jwtPol1},
			secretNamespace: "default",
			secretName:      "jwk-secret",
			expected:        []*conf_v1.Policy{jwtPol1},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{jwtPol2},
			secretNamespace: "default",
			secretName:      "jwk-secret",
			expected:        nil,
			msg:             "Ignore policies in other namespaces",
		},
		{
			policies:        []*conf_v1.Policy{jwtPol1, jwtPol2},
			secretNamespace: "default",
			secretName:      "jwk-secret",
			expected:        []*conf_v1.Policy{jwtPol1},
			msg:             "Find policy in default ns, ignore other",
		},
		{
			policies:        []*conf_v1.Policy{basicPol1},
			secretNamespace: "default",
			secretName:      "basic-auth-secret",
			expected:        []*conf_v1.Policy{basicPol1},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{basicPol2},
			secretNamespace: "default",
			secretName:      "basic-auth-secret",
			expected:        nil,
			msg:             "Ignore policies in other namespaces",
		},
		{
			policies:        []*conf_v1.Policy{basicPol1, basicPol2},
			secretNamespace: "default",
			secretName:      "basic-auth-secret",
			expected:        []*conf_v1.Policy{basicPol1},
			msg:             "Find policy in default ns, ignore other",
		},
		{
			policies:        []*conf_v1.Policy{ingTLSPol},
			secretNamespace: "default",
			secretName:      "ingress-mtls-secret",
			expected:        []*conf_v1.Policy{ingTLSPol},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{jwtPol1, ingTLSPol},
			secretNamespace: "default",
			secretName:      "ingress-mtls-secret",
			expected:        []*conf_v1.Policy{ingTLSPol},
			msg:             "Find policy in default ns, ignore other types",
		},
		{
			policies:        []*conf_v1.Policy{egTLSPol},
			secretNamespace: "default",
			secretName:      "egress-mtls-secret",
			expected:        []*conf_v1.Policy{egTLSPol},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{jwtPol1, egTLSPol},
			secretNamespace: "default",
			secretName:      "egress-mtls-secret",
			expected:        []*conf_v1.Policy{egTLSPol},
			msg:             "Find policy in default ns, ignore other types",
		},
		{
			policies:        []*conf_v1.Policy{egTLSPol2},
			secretNamespace: "default",
			secretName:      "egress-trusted-secret",
			expected:        []*conf_v1.Policy{egTLSPol2},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{egTLSPol, egTLSPol2},
			secretNamespace: "default",
			secretName:      "egress-trusted-secret",
			expected:        []*conf_v1.Policy{egTLSPol2},
			msg:             "Find policy in default ns, ignore other types",
		},
		{
			policies:        []*conf_v1.Policy{oidcPol},
			secretNamespace: "default",
			secretName:      "oidc-secret",
			expected:        []*conf_v1.Policy{oidcPol},
			msg:             "Find policy in default ns",
		},
		{
			policies:        []*conf_v1.Policy{ingTLSPol, oidcPol},
			secretNamespace: "default",
			secretName:      "oidc-secret",
			expected:        []*conf_v1.Policy{oidcPol},
			msg:             "Find policy in default ns, ignore other types",
		},
	}
	for _, test := range tests {
		result := findPoliciesForSecret(test.policies, test.secretNamespace, test.secretName)
		if diff := cmp.Diff(test.expected, result); diff != "" {
			t.Errorf("findPoliciesForSecret() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddJWTSecrets(t *testing.T) {
	t.Parallel()
	invalidErr := errors.New("invalid")
	validJWKSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-jwk-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeJWK,
	}
	invalidJWKSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-jwk-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeJWK,
	}

	tests := []struct {
		policies           []*conf_v1.Policy
		expectedSecretRefs map[string]*secrets.SecretReference
		wantErr            bool
		msg                string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Secret: "valid-jwk-secret",
							Realm:  "My API",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-jwk-secret": {
					Secret: validJWKSecret,
					Path:   "/etc/nginx/secrets/default-valid-jwk-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid secret",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Realm:    "My API",
							JwksURI:  "https://idp.com/token",
							KeyCache: "1h",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid policy using JwksUri",
		},
		{
			policies:           []*conf_v1.Policy{},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting invalid secret with wrong policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "jwt-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						JWTAuth: &conf_v1.JWTAuth{
							Secret: "invalid-jwk-secret",
							Realm:  "My API",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-jwk-secret": {
					Secret: invalidJWKSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid secret",
		},
	}

	lbc := LoadBalancerController{
		secretStore: secrets.NewFakeSecretsStore(map[string]*secrets.SecretReference{
			"default/valid-jwk-secret": {
				Secret: validJWKSecret,
				Path:   "/etc/nginx/secrets/default-valid-jwk-secret",
			},
			"default/invalid-jwk-secret": {
				Secret: invalidJWKSecret,
				Error:  invalidErr,
			},
		}),
	}

	for _, test := range tests {
		result := make(map[string]*secrets.SecretReference)

		err := lbc.addJWTSecretRefs(result, test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("addJWTSecretRefs() returned %v, for the case of %v", err, test.msg)
		}

		if diff := cmp.Diff(test.expectedSecretRefs, result, cmp.Comparer(errorComparer)); diff != "" {
			t.Errorf("addJWTSecretRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddBasicSecrets(t *testing.T) {
	t.Parallel()
	invalidErr := errors.New("invalid")
	validBasicSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-basic-auth-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeJWK,
	}
	invalidBasicSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-basic-auth-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeJWK,
	}

	tests := []struct {
		policies           []*conf_v1.Policy
		expectedSecretRefs map[string]*secrets.SecretReference
		wantErr            bool
		msg                string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "basic-auth-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						BasicAuth: &conf_v1.BasicAuth{
							Secret: "valid-basic-auth-secret",
							Realm:  "My API",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-basic-auth-secret": {
					Secret: validBasicSecret,
					Path:   "/etc/nginx/secrets/default-valid-basic-auth-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid secret",
		},
		{
			policies:           []*conf_v1.Policy{},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "basic-auth-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting invalid secret with wrong policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "basic-auth-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						BasicAuth: &conf_v1.BasicAuth{
							Secret: "invalid-basic-auth-secret",
							Realm:  "My API",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-basic-auth-secret": {
					Secret: invalidBasicSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid secret",
		},
	}

	lbc := LoadBalancerController{
		secretStore: secrets.NewFakeSecretsStore(map[string]*secrets.SecretReference{
			"default/valid-basic-auth-secret": {
				Secret: validBasicSecret,
				Path:   "/etc/nginx/secrets/default-valid-basic-auth-secret",
			},
			"default/invalid-basic-auth-secret": {
				Secret: invalidBasicSecret,
				Error:  invalidErr,
			},
		}),
	}

	for _, test := range tests {
		result := make(map[string]*secrets.SecretReference)

		err := lbc.addBasicSecretRefs(result, test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("addBasicSecretRefs() returned %v, for the case of %v", err, test.msg)
		}

		if diff := cmp.Diff(test.expectedSecretRefs, result, cmp.Comparer(errorComparer)); diff != "" {
			t.Errorf("addBasicSecretRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddIngressMTLSSecret(t *testing.T) {
	t.Parallel()
	invalidErr := errors.New("invalid")
	validSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-ingress-mtls-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeCA,
	}
	invalidSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-ingress-mtls-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeCA,
	}

	tests := []struct {
		policies           []*conf_v1.Policy
		expectedSecretRefs map[string]*secrets.SecretReference
		wantErr            bool
		msg                string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "valid-ingress-mtls-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-ingress-mtls-secret": {
					Secret: validSecret,
					Path:   "/etc/nginx/secrets/default-valid-ingress-mtls-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid secret",
		},
		{
			policies:           []*conf_v1.Policy{},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with wrong policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						IngressMTLS: &conf_v1.IngressMTLS{
							ClientCertSecret: "invalid-ingress-mtls-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-ingress-mtls-secret": {
					Secret: invalidSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid secret",
		},
	}

	lbc := LoadBalancerController{
		secretStore: secrets.NewFakeSecretsStore(map[string]*secrets.SecretReference{
			"default/valid-ingress-mtls-secret": {
				Secret: validSecret,
				Path:   "/etc/nginx/secrets/default-valid-ingress-mtls-secret",
			},
			"default/invalid-ingress-mtls-secret": {
				Secret: invalidSecret,
				Error:  invalidErr,
			},
		}),
	}

	for _, test := range tests {
		result := make(map[string]*secrets.SecretReference)

		err := lbc.addIngressMTLSSecretRefs(result, test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("addIngressMTLSSecretRefs() returned %v, for the case of %v", err, test.msg)
		}

		if diff := cmp.Diff(test.expectedSecretRefs, result, cmp.Comparer(errorComparer)); diff != "" {
			t.Errorf("addIngressMTLSSecretRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddEgressMTLSSecrets(t *testing.T) {
	t.Parallel()
	invalidErr := errors.New("invalid")
	validMTLSSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-egress-mtls-secret",
			Namespace: "default",
		},
		Type: api_v1.SecretTypeTLS,
	}
	validTrustedSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-egress-trusted-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeCA,
	}
	invalidMTLSSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-egress-mtls-secret",
			Namespace: "default",
		},
		Type: api_v1.SecretTypeTLS,
	}
	invalidTrustedSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-egress-trusted-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeCA,
	}

	tests := []struct {
		policies           []*conf_v1.Policy
		expectedSecretRefs map[string]*secrets.SecretReference
		wantErr            bool
		msg                string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret: "valid-egress-mtls-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-egress-mtls-secret": {
					Secret: validMTLSSecret,
					Path:   "/etc/nginx/secrets/default-valid-egress-mtls-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid TLS secret",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-egress-trusted-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TrustedCertSecret: "valid-egress-trusted-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-egress-trusted-secret": {
					Secret: validTrustedSecret,
					Path:   "/etc/nginx/secrets/default-valid-egress-trusted-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid TrustedCA secret",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret:         "valid-egress-mtls-secret",
							TrustedCertSecret: "valid-egress-trusted-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-egress-mtls-secret": {
					Secret: validMTLSSecret,
					Path:   "/etc/nginx/secrets/default-valid-egress-mtls-secret",
				},
				"default/valid-egress-trusted-secret": {
					Secret: validTrustedSecret,
					Path:   "/etc/nginx/secrets/default-valid-egress-trusted-secret",
				},
			},
			wantErr: false,
			msg:     "test getting valid secrets",
		},
		{
			policies:           []*conf_v1.Policy{},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "ingress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with wrong policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TLSSecret: "invalid-egress-mtls-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-egress-mtls-secret": {
					Secret: invalidMTLSSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid TLS secret",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "egress-mtls-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						EgressMTLS: &conf_v1.EgressMTLS{
							TrustedCertSecret: "invalid-egress-trusted-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-egress-trusted-secret": {
					Secret: invalidTrustedSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid TrustedCA secret",
		},
	}

	lbc := LoadBalancerController{
		secretStore: secrets.NewFakeSecretsStore(map[string]*secrets.SecretReference{
			"default/valid-egress-mtls-secret": {
				Secret: validMTLSSecret,
				Path:   "/etc/nginx/secrets/default-valid-egress-mtls-secret",
			},
			"default/valid-egress-trusted-secret": {
				Secret: validTrustedSecret,
				Path:   "/etc/nginx/secrets/default-valid-egress-trusted-secret",
			},
			"default/invalid-egress-mtls-secret": {
				Secret: invalidMTLSSecret,
				Error:  invalidErr,
			},
			"default/invalid-egress-trusted-secret": {
				Secret: invalidTrustedSecret,
				Error:  invalidErr,
			},
		}),
	}

	for _, test := range tests {
		result := make(map[string]*secrets.SecretReference)

		err := lbc.addEgressMTLSSecretRefs(result, test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("addEgressMTLSSecretRefs() returned %v, for the case of %v", err, test.msg)
		}
		if diff := cmp.Diff(test.expectedSecretRefs, result, cmp.Comparer(errorComparer)); diff != "" {
			t.Errorf("addEgressMTLSSecretRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestAddOidcSecret(t *testing.T) {
	t.Parallel()
	invalidErr := errors.New("invalid")
	validSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "valid-oidc-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"client-secret": nil,
		},
		Type: secrets.SecretTypeOIDC,
	}
	invalidSecret := &api_v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "invalid-oidc-secret",
			Namespace: "default",
		},
		Type: secrets.SecretTypeOIDC,
	}

	tests := []struct {
		policies           []*conf_v1.Policy
		expectedSecretRefs map[string]*secrets.SecretReference
		wantErr            bool
		msg                string
	}{
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientSecret: "valid-oidc-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/valid-oidc-secret": {
					Secret: validSecret,
				},
			},
			wantErr: false,
			msg:     "test getting valid secret",
		},
		{
			policies:           []*conf_v1.Policy{},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with no policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						AccessControl: &conf_v1.AccessControl{
							Allow: []string{"127.0.0.1"},
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{},
			wantErr:            false,
			msg:                "test getting valid secret with wrong policy",
		},
		{
			policies: []*conf_v1.Policy{
				{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "oidc-policy",
						Namespace: "default",
					},
					Spec: conf_v1.PolicySpec{
						OIDC: &conf_v1.OIDC{
							ClientSecret: "invalid-oidc-secret",
						},
					},
				},
			},
			expectedSecretRefs: map[string]*secrets.SecretReference{
				"default/invalid-oidc-secret": {
					Secret: invalidSecret,
					Error:  invalidErr,
				},
			},
			wantErr: true,
			msg:     "test getting invalid secret",
		},
	}

	lbc := LoadBalancerController{
		secretStore: secrets.NewFakeSecretsStore(map[string]*secrets.SecretReference{
			"default/valid-oidc-secret": {
				Secret: validSecret,
			},
			"default/invalid-oidc-secret": {
				Secret: invalidSecret,
				Error:  invalidErr,
			},
		}),
	}

	for _, test := range tests {
		result := make(map[string]*secrets.SecretReference)

		err := lbc.addOIDCSecretRefs(result, test.policies)
		if (err != nil) != test.wantErr {
			t.Errorf("addOIDCSecretRefs() returned %v, for the case of %v", err, test.msg)
		}

		if diff := cmp.Diff(test.expectedSecretRefs, result, cmp.Comparer(errorComparer)); diff != "" {
			t.Errorf("addOIDCSecretRefs() '%v' mismatch (-want +got):\n%s", test.msg, diff)
		}
	}
}

func TestPreSyncSecrets(t *testing.T) {
	t.Parallel()
	secretLister := &cache.FakeCustomStore{
		ListFunc: func() []interface{} {
			return []interface{}{
				&api_v1.Secret{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "supported-secret",
						Namespace: "default",
					},
					Type: api_v1.SecretTypeTLS,
				},
				&api_v1.Secret{
					ObjectMeta: meta_v1.ObjectMeta{
						Name:      "unsupported-secret",
						Namespace: "default",
					},
					Type: api_v1.SecretTypeOpaque,
				},
			}
		},
	}
	nsi := make(map[string]*namespacedInformer)
	nsi[""] = &namespacedInformer{secretLister: secretLister, isSecretsEnabledNamespace: true}

	lbc := LoadBalancerController{
		isNginxPlus:         true,
		secretStore:         secrets.NewEmptyFakeSecretsStore(),
		namespacedInformers: nsi,
	}

	lbc.preSyncSecrets()

	supportedKey := "default/supported-secret"
	ref := lbc.secretStore.GetSecret(supportedKey)
	if ref.Error != nil {
		t.Errorf("GetSecret(%q) returned a reference with an unexpected error %v", supportedKey, ref.Error)
	}

	unsupportedKey := "default/unsupported-secret"
	ref = lbc.secretStore.GetSecret(unsupportedKey)
	if ref.Error == nil {
		t.Errorf("GetSecret(%q) returned a reference without an expected error", unsupportedKey)
	}
}

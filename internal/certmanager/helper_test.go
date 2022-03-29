/*
Copyright 2020 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package certmanager

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/jetstack/cert-manager/test/unit/gen"
	vsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
)

func Test_translateVsSpec(t *testing.T) {
	type testCase struct {
		crt           *cmapi.Certificate
		cmspec        *vsapi.CertManager
		check         func(*assert.Assertions, *cmapi.Certificate)
		expectedError error
	}

	validSpec := vsapi.CertManager{
		CommonName:  "www.example.com",
		Duration:    "168h", // 1 week
		RenewBefore: "24h",
		Usages:      "server auth,signing",
	}

	invalidDuration := vsapi.CertManager{
		Duration: "un-parsable duration",
	}

	invalidRenewBefore := vsapi.CertManager{
		RenewBefore: "un-parsable duration",
	}

	invalidUsages := vsapi.CertManager{
		Usages: "playing ping pong",
	}

	invalidUsageList := vsapi.CertManager{
		Usages: "server auth,,signing",
	}

	tests := map[string]testCase{
		"success": {
			crt:    gen.Certificate("example-cert"),
			cmspec: &validSpec,
			check: func(a *assert.Assertions, crt *cmapi.Certificate) {
				a.Equal("www.example.com", crt.Spec.CommonName)
				a.Equal(&metav1.Duration{Duration: time.Hour * 24 * 7}, crt.Spec.Duration)
				a.Equal(&metav1.Duration{Duration: time.Hour * 24}, crt.Spec.RenewBefore)
				a.Equal([]cmapi.KeyUsage{cmapi.UsageServerAuth, cmapi.UsageSigning}, crt.Spec.Usages)
			},
		},
		"nil cm spec": {
			crt:    gen.Certificate("example-cert"),
			cmspec: nil,
		},
		"nil certificate": {
			crt:           nil,
			cmspec:        &validSpec,
			expectedError: errNilCertificate,
		},
		"bad duration": {
			crt:           gen.Certificate("example-cert"),
			cmspec:        &invalidDuration,
			expectedError: errInvalidCertManagerField,
		},
		"bad renewBefore": {
			crt:           gen.Certificate("example-cert"),
			cmspec:        &invalidRenewBefore,
			expectedError: errInvalidCertManagerField,
		},
		"bad usages": {
			crt:           gen.Certificate("example-cert"),
			cmspec:        &invalidUsages,
			expectedError: errInvalidCertManagerField,
		},
		"bad usage list": {
			crt:           gen.Certificate("example-cert"),
			cmspec:        &invalidUsageList,
			expectedError: errInvalidCertManagerField,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			crt := tc.crt.DeepCopy()

			err := translateVsSpec(crt, tc.cmspec)

			if tc.expectedError != nil {
				assertErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
			if tc.check != nil {
				tc.check(assert.New(t), crt)
			}
		})
	}
}

// assertErrorIs checks that the supplied error has the target error in its chain.
// TODO Upgrade to next release of testify package which has this built in.
func assertErrorIs(t *testing.T, err, target error) {
	if assert.Error(t, err) {
		assert.Truef(t, errors.Is(err, target), "unexpected error type. err: %v, target: %v", err, target)
	}
}

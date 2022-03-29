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

// Package certmanager provides a controller for creating and managing
// certificates for VS resources.
package certmanager

import (
	"errors"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiutil "github.com/jetstack/cert-manager/pkg/api/util"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	vsapi "github.com/nginxinc/kubernetes-ingress/pkg/apis/configuration/v1"
)

var (
	errNilCertificate          = errors.New("the supplied Certificate pointer was nil")
	errInvalidCertManagerField = errors.New("invalid cert manager field")
	clusterIssuerCmField       = "tls.cert-manager.cluster-issuer"
	durationCmField            = "tls.cert-manager.duration"
	issuerCmField              = "tls.cert-manager.issuer"
	issuerGroupCmField         = "tls.cert-manager.issuer-group"
	issuerKindCmField          = "tls.cert-manager.issuer-kind"
	renewBeforeCmField         = "tls.cert-manager.renew-before"
	usagesCmField              = "tls.cert-manager.usages"
)

// translateVsSpec updates the Certificate spec using the VS TLS Cert-Manager
// fields. For example, the following VirtualServer:
//
// apiVersion: k8s.nginx.org/v1
// kind: VirtualServer
// spec:
//   <...>
//   tls:
//     <...>
//     cert-manager:
//       common-name: example.com
//       duration: 2160h
//       renew-before: 1440h
//       usages: "digital signature,key encipherment"
//
// is mapped to the following Certificate:
//
//   kind: Certificate
//   spec:
//     commonName: example.com
//     duration: 2160h
//     renewBefore: 1440h
//     usages:
//       - digital signature
//       - key encipherment
func translateVsSpec(crt *cmapi.Certificate, vsCmSpec *vsapi.CertManager) error {
	if crt == nil {
		return errNilCertificate
	}

	if vsCmSpec == nil {
		return nil
	}

	if vsCmSpec.CommonName != "" {
		crt.Spec.CommonName = vsCmSpec.CommonName
	}

	if vsCmSpec.Duration != "" {
		duration, err := time.ParseDuration(vsCmSpec.Duration)
		if err != nil {
			return fmt.Errorf("%w %q: %v", errInvalidCertManagerField, durationCmField, err)
		}
		crt.Spec.Duration = &metav1.Duration{Duration: duration}
	}

	if vsCmSpec.RenewBefore != "" {
		duration, err := time.ParseDuration(vsCmSpec.RenewBefore)
		if err != nil {
			return fmt.Errorf("%w %q: %v", errInvalidCertManagerField, renewBeforeCmField, err)
		}
		crt.Spec.RenewBefore = &metav1.Duration{Duration: duration}
	}

	if vsCmSpec.Usages != "" {
		var newUsages []cmapi.KeyUsage
		for _, usageName := range strings.Split(vsCmSpec.Usages, ",") {
			usage := cmapi.KeyUsage(strings.Trim(usageName, " "))
			_, isKU := apiutil.KeyUsageType(usage)
			_, isEKU := apiutil.ExtKeyUsageType(usage)
			if !isKU && !isEKU {
				return fmt.Errorf("%w %q: invalid key usage name %q", errInvalidCertManagerField, usagesCmField, usageName)
			}
			newUsages = append(newUsages, usage)
		}
		crt.Spec.Usages = newUsages
	}
	return nil
}

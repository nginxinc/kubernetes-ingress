package workload

import (
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/spiffe/go-spiffe/internal"
	"github.com/spiffe/go-spiffe/proto/spiffe/workload"
)

func protoToX509SVIDs(protoSVIDs *workload.X509SVIDResponse) (*X509SVIDs, error) {
	if len(protoSVIDs.GetSvids()) == 0 {
		return nil, errors.New("workload response contains no svids")
	}

	federatedBundles := make(map[string][]*x509.Certificate)
	for federatedDomainID, federatedBundleDER := range protoSVIDs.FederatedBundles {
		federatedBundle, err := x509.ParseCertificates(federatedBundleDER)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bundle for federated domain %q: %v", federatedDomainID, err)
		}
		if len(federatedBundle) == 0 {
			return nil, fmt.Errorf("no certificates in bundle for federated domain %q", federatedDomainID)
		}
		federatedBundles[federatedDomainID] = federatedBundle
	}

	svids := new(X509SVIDs)
	for i, protoSVID := range protoSVIDs.GetSvids() {
		svid, err := protoToX509SVID(protoSVID, federatedBundles)
		if err != nil {
			// TODO(tjulian): Probably support partial success
			return nil, fmt.Errorf("failed to parse svid entry %d for spiffe id %q: %v", i, protoSVID.GetSpiffeId(), err)
		}
		svids.SVIDs = append(svids.SVIDs, svid)
	}

	return svids, nil
}

func protoToX509SVID(svid *workload.X509SVID, allFederatedBundles map[string][]*x509.Certificate) (*X509SVID, error) {
	certificates, err := x509.ParseCertificates(svid.GetX509Svid())
	if err != nil {
		return nil, err
	}
	if len(certificates) == 0 {
		return nil, errors.New("no certificates found")
	}
	privateKey, err := x509.ParsePKCS8PrivateKey(svid.GetX509SvidKey())
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}
	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("private key is type %T, not crypto.Signer", privateKey)
	}
	trustBundle, err := x509.ParseCertificates(svid.GetBundle())
	if err != nil {
		return nil, fmt.Errorf("failed to parse trust bundle: %v", err)
	}
	if len(trustBundle) == 0 {
		return nil, errors.New("no certificates in trust bundle")
	}
	trustBundlePool := internal.CertPoolFromCerts(trustBundle)

	federatedTrustBundles := make(map[string][]*x509.Certificate)
	federatedTrustBundlePools := make(map[string]*x509.CertPool)
	for _, federatesWith := range svid.GetFederatesWith() {
		bundle, ok := allFederatedBundles[federatesWith]
		if !ok {
			return nil, fmt.Errorf("missing bundle for federated domain %q", federatesWith)
		}
		federatedTrustBundles[federatesWith] = bundle
		federatedTrustBundlePools[federatesWith] = internal.CertPoolFromCerts(bundle)
	}

	return &X509SVID{
		SPIFFEID:                  svid.GetSpiffeId(),
		PrivateKey:                signer,
		Certificates:              certificates,
		TrustBundle:               trustBundle,
		TrustBundlePool:           trustBundlePool,
		FederatedTrustBundles:     federatedTrustBundles,
		FederatedTrustBundlePools: federatedTrustBundlePools,
	}, nil
}

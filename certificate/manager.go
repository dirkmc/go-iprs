package iprs_cert

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"

	ld "github.com/dirkmc/go-iprs/ipld"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
	//u "github.com/ipfs/go-ipfs-util"
)

const CertType = "cert"
const certPrefix = "/" + CertType + "/"
const certPrefixLen = len(certPrefix)
const CertFetchTimeout = time.Second * 10
const CertPutTimeout = time.Second * 10

var log = logging.Logger("iprs.cert")

type CertificateManager struct {
	dag node.NodeGetter
}

func NewCertificateManager(dag node.NodeGetter) *CertificateManager {
	return &CertificateManager{dag}
}

/*
func GetCertPath(certHash string) string {
	return certPrefix + certHash
}

func (m *CertificateManager) PutCertificate(ctx context.Context, cert *x509.Certificate) (string, error) {
	pemBytes, err := MarshalCertificate(cert)
	if err != nil {
		log.Warningf("Failed to marshall certificate: %s", err)
		return "", err
	}

	certHash := getCertificateHashFromBytes(pemBytes)
	certKey := GetCertPath(certHash)
	log.Debugf("Putting certificate at %s", certKey)

	timectx, cancel := context.WithTimeout(ctx, CertPutTimeout)
	defer cancel()

	if err := m.routing.PutValue(timectx, certKey, pemBytes); err != nil {
		log.Warningf("Failed to put certificate at %s: %s", certKey, err)
		return "", err
	}
	return certHash, nil
}
*/
func (m *CertificateManager) GetCertificate(ctx context.Context, certCid *cid.Cid) (*x509.Certificate, error) {
	log.Debugf("CertificateManager get certificate %s", certCid)
	if certCid.Type() != ld.CodecCertRaw {
		return nil, fmt.Errorf("Cid Codec %d is not CodecCertRaw in Cid %s", ld.CodecCertRaw, certCid)
	}

	log.Debugf("Fetching certificate at %s", certCid)

	timectx, cancel := context.WithTimeout(ctx, CertFetchTimeout)
	defer cancel()

	n, err := m.dag.Get(timectx, certCid)
	if err != nil {
		log.Warningf("Failed to fetch certificate at %s: %s", certCid, err)
		return nil, err
	}

	cert, err := UnmarshalCertificate(n.RawData())
	if err != nil {
		log.Warningf("Failed to unmarshal certificate at %s: %s", certCid, err)
		return nil, err
	}

	return cert, nil
}

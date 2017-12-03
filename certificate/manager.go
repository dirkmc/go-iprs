package recordstore_cert

import (
	"context"
	"crypto/x509"
	"time"
	logging "github.com/ipfs/go-log"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	u "github.com/ipfs/go-ipfs-util"
)

const CertType = "cert"
const certPrefix = "/" + CertType + "/"
const certPrefixLen = len(certPrefix)
const CertFetchTimeout = time.Second*10
const CertPutTimeout = time.Second*10

var log = logging.Logger("recordstore.cert")

type CertificateManager struct {
	routing routing.ValueStore
}

func NewCertificateManager(r routing.ValueStore) *CertificateManager {
	return &CertificateManager{
		routing: r,
	}
}

func getCertPath(certHash string) string {
	return certPrefix + certHash
}

func (m *CertificateManager) PutCertificate(ctx context.Context, cert *x509.Certificate) (string, error) {
	pemBytes := MarshalCertificate(cert)
	certHash := string(u.Hash(pemBytes))

	certKey := getCertPath(certHash)
	log.Debugf("Putting certificate at %s", certKey)

	timectx, cancel := context.WithTimeout(ctx, CertPutTimeout)
	defer cancel()

	if err := m.routing.PutValue(timectx, certKey, pemBytes); err != nil {
		log.Warningf("Failed to put certificate at %s: %s", certKey, err)
		return "", err
	}
	return certHash, nil
}

func (m *CertificateManager) GetCertificate(ctx context.Context, certHash string) (*x509.Certificate, error) {
	certKey := getCertPath(certHash)
	log.Debugf("Fetching certificate at %s", certKey)

	timectx, cancel := context.WithTimeout(ctx, CertFetchTimeout)
	defer cancel()

	val, err := m.routing.GetValue(timectx, certKey)
	if err != nil {
		log.Warningf("Failed to fetch certificate at %s: %s", certKey, err)
		return nil, err
	}

	cert, err := UnmarshalCertificate(val)
	if err != nil {
		log.Warningf("Failed to unmarshal certificate at %s: %s", certKey, err)
		return nil, err
	}

	return cert, nil
}

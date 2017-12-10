package recordstore_record

import (
	"context"
	"crypto/x509"
	"crypto/rsa"
	"fmt"
	"strings"
	"time"
	pb "github.com/dirkmc/go-iprs/pb"
	proto "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/proto"
	path "github.com/ipfs/go-ipfs/path"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	rsp "github.com/dirkmc/go-iprs/path"
	u "github.com/ipfs/go-ipfs-util"
	c "github.com/dirkmc/go-iprs/certificate"
)

// ***** CertRecordManager ***** //
type CertRecordManager struct {
	routing routing.ValueStore
	certManager *c.CertificateManager
}

func NewCertRecordManager(r routing.ValueStore, m *c.CertificateManager) *CertRecordManager {
	return &CertRecordManager{
		routing: r,
		certManager: m,
	}
}

func (m *CertRecordManager) NewRecord(pk *rsa.PrivateKey, cert *x509.Certificate, val path.Path, eol time.Time) *CertRecord {
	return &CertRecord{
		m: m,
		cert: cert,
		pk: pk,
		val: val,
		eol: eol,
	}
}

func (m *CertRecordManager) PublishRecord(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry, cert *x509.Certificate) error {
	// Put the certificate and the record itself to routing
	resp := make(chan error, 2)

	go func() {
		_, err := m.certManager.PutCertificate(ctx, cert)
		resp <- err
	}()
	go func() {
		resp <- PutEntryToRouting(ctx, m.routing, iprsKey, entry)
	}()

	var err error
	for i := 0; i < 2; i++ {
		err = <-resp
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *CertRecordManager) VerifyRecord(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	log.Debugf("Verifying record %s", iprsKey)

	certHash, issuerCertHash, err := getCertEntryHashes(iprsKey, entry)
	if err != nil {
		return err
	}

	// Hashes should be X509 certificates retrievable from ipfs
	cert, issuerCert, err := m.getCerts(ctx, certHash, issuerCertHash)
	if err != nil {
		return err
	}

	// Check that issuer issued the certificate
	if err = c.CheckSignatureFrom(cert, issuerCert); err != nil {
		log.Warningf("Check signature parent failed for cert [%s] issued by cert [%s]: %v", certHash, issuerCertHash, err)
		return err
	}

	// Check signature with certificate
	if err = c.CheckSignature(cert, RecordDataForSig(entry), entry.GetSignature()); err != nil {
		return fmt.Errorf("Check signature failed for cert [%s]: %v", certHash, err)
	}

	// Success
	log.Debugf("Record verification successful %s", iprsKey)
	return nil
}

func (m *CertRecordManager) getCerts(ctx context.Context, certHash, issuerCertHash string) (*x509.Certificate, *x509.Certificate, error) {
	// TODO: Optimize for the case where the two cert hashes are the same
	var cert, issuerCert *x509.Certificate
	resp := make(chan error, 2)

	getCert := func(hash string, cType string, cPtr **x509.Certificate) {
		c, err := m.certManager.GetCertificate(ctx, hash)
		if err != nil {
			log.Warningf("Failed to get %s [%s]", cType, hash)
			resp <- err
			return
		}

		*cPtr = c
		resp <- nil
	}

	go getCert(certHash, "Certificate", &cert)
	go getCert(issuerCertHash, "Issuer Certificate", &issuerCert)

	var err error
	for i := 0; i < 2; i++ {
		err = <-resp
		if err != nil {
			return nil, nil, err
		}
	}

	return cert, issuerCert, nil
}


// ***** CertRecord ***** //

type CertRecord struct {
	m *CertRecordManager
	cert *x509.Certificate
	pk *rsa.PrivateKey
	val path.Path
	eol time.Time
}

func (r *CertRecord) Publish(ctx context.Context, iprsKey rsp.IprsPath, seq uint64) error {
	entry := new(pb.IprsEntry)

	entry.Value = []byte(r.val)
	typ := pb.IprsEntry_Cert
	entry.ValidityType = &typ
	entry.Sequence = proto.Uint64(seq)
	entry.Validity = marshallCertEntryValidity(r.eol, r.cert)

	sig, err := c.Sign(r.pk, RecordDataForSig(entry))
	if err != nil {
		return err
	}
	entry.Signature = sig

	return r.m.PublishRecord(ctx, iprsKey, entry, r.cert)
}


// ***** CertRecordChecker ***** //
type CertRecordChecker struct {}

func NewCertRecordChecker() *CertRecordChecker {
	return &CertRecordChecker{}
}

func (v *CertRecordChecker) SelectRecord(recs []*pb.IprsEntry, vals [][]byte) (int, error) {
	return EolSelectRecord(recs, vals, getCertEntryEol)
}

func (v *CertRecordChecker) ValidateRecord(iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	_, err := getCertEntryEol(entry)
	return err
}

func marshallCertEntryValidity(eol time.Time, cert *x509.Certificate) []byte {
	return []byte(u.FormatRFC3339(eol) + "\n" + c.GetCertificateHash(cert))
}

func getCertEntryHashes(iprsKey rsp.IprsPath, entry *pb.IprsEntry) (string, string, error) {
	// The validity is in the format
	// <EOL>\n<issuer cert hash>
	validity := string(entry.GetValidity())
	valParts := strings.SplitN(validity, "\n", 2)
	if len(valParts) != 2 {
		return "", "", fmt.Errorf("Unrecognized validity format: [%s]", validity)
	}

	certHash := valParts[1]
	issuerCertHash := iprsKey.GetHashString()

	return certHash, issuerCertHash, nil
}

func getCertEntryEol(e *pb.IprsEntry) (string, error) {
	// The validity is in the format
	// <EOL>\n<issuer cert hash>
	validity := string(e.GetValidity())
	valParts := strings.SplitN(validity, "\n", 2)
	if len(valParts) != 2 {
		return "", fmt.Errorf("Unrecognized validity format: [%s]", validity)
	}
	return valParts[0], nil
}

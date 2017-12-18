package iprs_record

import (
	"context"
	"fmt"
	"crypto/x509"
	"crypto/rsa"
	c "github.com/dirkmc/go-iprs/certificate"
	pb "github.com/dirkmc/go-iprs/pb"
	rsp "github.com/dirkmc/go-iprs/path"
)

type CertRecordSigner struct {
	m *c.CertificateManager
	cert *x509.Certificate
	pk *rsa.PrivateKey
}

// TODO: Whitelist
func NewCertRecordSigner(m *c.CertificateManager, cert *x509.Certificate, pk *rsa.PrivateKey) *CertRecordSigner {
	return &CertRecordSigner{
		m: m,
		cert: cert,
		pk: pk,
	}
}

func (s *CertRecordSigner) BasePath() (rsp.IprsPath, error) {
	h, err := c.GetCertificateHash(s.cert)
	if err != nil {
		return rsp.NilPath, err
	}
	return rsp.FromString("/iprs/" + h)
}

func (s *CertRecordSigner) VerificationType() *pb.IprsEntry_VerificationType {
	t := pb.IprsEntry_Cert
	return &t
}

func (s *CertRecordSigner) Verification() ([]byte, error) {
	h, err := c.GetCertificateHash(s.cert)
	if err != nil {
		return nil, err
	}
	return []byte(h), nil
}

func (s *CertRecordSigner) PublishVerification(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	// TODO: Check iprsKey is valid for this type of RecordSigner
	_, err := s.m.PutCertificate(ctx, s.cert)
	return err
}

func (s *CertRecordSigner) SignRecord(entry *pb.IprsEntry) error {
	sig, err := c.Sign(s.pk, RecordDataForSig(entry))
	if err != nil {
		return err
	}
	entry.Signature = sig

	return nil
}

type CertRecordVerifier struct {
	m *c.CertificateManager
}

func NewCertRecordVerifier(m *c.CertificateManager) *CertRecordVerifier {
	return &CertRecordVerifier{ m }
}

func (v *CertRecordVerifier) VerifyRecord(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	certHash := string(entry.GetVerification())
	issuerCertHash := iprsKey.GetHashString()

	// Hashes should be X509 certificates retrievable from ipfs
	cert, issuerCert, err := v.getCerts(ctx, certHash, issuerCertHash)
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
	return nil
}

func (v *CertRecordVerifier) getCerts(ctx context.Context, certHash, issuerCertHash string) (*x509.Certificate, *x509.Certificate, error) {
	// The issuer can use her own cert to sign records
	if certHash == issuerCertHash {
		ct, err := v.m.GetCertificate(ctx, certHash)
		if err != nil {
			log.Warningf("Failed to get Certificate [%s]", certHash)
			return nil, nil, err
		}
		return ct, ct, nil
	}

	// If the issuer and cert hash are different, get them in parallel
	var cert, issuerCert *x509.Certificate
	resp := make(chan error, 2)

	getCert := func(hash string, cType string, cPtr **x509.Certificate) {
		ct, err := v.m.GetCertificate(ctx, hash)
		if err != nil {
			log.Warningf("Failed to get %s [%s]", cType, hash)
			resp <- err
			return
		}

		*cPtr = ct
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

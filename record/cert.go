package iprs_record

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"

	c "github.com/dirkmc/go-iprs/certificate"
	ld "github.com/dirkmc/go-iprs/ipld"
	rsp "github.com/dirkmc/go-iprs/path"
	node "gx/ipfs/QmNwUEK7QbwSqyKBu3mMtToo8SUc6wQJ7gdZq4gGGJqfnf/go-ipld-format"
	cid "gx/ipfs/QmeSrf6pzut73u6zLQkRFQ3ygt3k6XFT2kjdYP8Tnkwwyg/go-cid"
)

type CertRecordSigner struct {
	cert     *x509.Certificate
	pk       *rsa.PrivateKey
	certNode node.Node
}

// TODO: Include a whitelist of certificates that are allowed to change IPRS records
// under the issuing certificate's path (so that permission can be revoked)
func NewCertRecordSigner(cert *x509.Certificate, pk *rsa.PrivateKey) *CertRecordSigner {
	return &CertRecordSigner{
		cert: cert,
		pk:   pk,
	}
}

func (s *CertRecordSigner) VerificationType() ld.IprsVerificationType {
	return ld.VerificationType_Cert
}

// Cache the Certificate node
func (s *CertRecordSigner) getCertNode() (node.Node, error) {
	if s.certNode != nil {
		return s.certNode, nil
	}

	b, err := c.MarshalCertificate(s.cert)
	if err != nil {
		return nil, err
	}
	s.certNode = ld.Certificate(b)

	return s.certNode, nil
}

func (s *CertRecordSigner) Nodes() ([]node.Node, error) {
	n, err := s.getCertNode()
	if err != nil {
		return nil, err
	}
	return []node.Node{n}, nil
}

func (s *CertRecordSigner) BasePath(id string) (rsp.IprsPath, error) {
	n, err := s.getCertNode()
	if err != nil {
		return rsp.NilPath, err
	}
	return rsp.FromString("/iprs/" + n.Cid().String() + "/" + id)
}

func (s *CertRecordSigner) SignRecord(data []byte) ([]byte, error) {
	return c.Sign(s.pk, data)
}

func (s *CertRecordSigner) Verification() (interface{}, error) {
	n, err := s.getCertNode()
	if err != nil {
		return nil, err
	}
	return n.Cid(), nil
}

func prepareCertSig(o interface{}) ([]byte, error) {
	c, err := toCid(o)
	if err != nil {
		return nil, err
	}
	return c.Bytes(), nil
}

// The CBOR encoder encodes CIDs as links, so depending on where we
// are in the process it may still be a CID or it may be a link
func toCid(o interface{}) (*cid.Cid, error) {
	c, ok := o.(*cid.Cid)
	if ok {
		return c, nil
	}
	l, ok := o.(*node.Link)
	if ok {
		return l.Cid, nil
	}
	return nil, fmt.Errorf("Unrecognized verification data type %T. Expected Link or Cid", o)
}

type CertRecordVerifier struct {
	m *c.CertificateManager
}

func NewCertRecordVerifier(m *c.CertificateManager) *CertRecordVerifier {
	return &CertRecordVerifier{m}
}

func (v *CertRecordVerifier) VerifyRecord(ctx context.Context, iprsKey rsp.IprsPath, record *Record) error {
	certCid, err := toCid(record.Validity.Verification)
	if err != nil {
		return err
	}
	issuerCertCid := iprsKey.Cid()

	// Hashes should be X509 certificates retrievable from ipfs
	cert, issuerCert, err := v.getCerts(ctx, certCid, issuerCertCid)
	if err != nil {
		return err
	}

	// Check that issuer issued the certificate
	if err = c.CheckSignatureFrom(cert, issuerCert); err != nil {
		log.Warningf("Check signature parent failed for cert [%s] issued by cert [%s]: %v", certCid, issuerCertCid, err)
		return err
	}

	// Check signature with certificate
	sigd, err := dataForSig(record.Value, record.Validity)
	if err != nil {
		return fmt.Errorf("Failed to marshall data for signature for cert [%s]: %v", certCid, err)
	}
	if err = c.CheckSignature(cert, sigd, record.Signature); err != nil {
		return fmt.Errorf("Check signature failed for cert [%s]: %v", certCid, err)
	}

	// Success
	return nil
}

func (v *CertRecordVerifier) getCerts(ctx context.Context, certCid, issuerCertCid *cid.Cid) (*x509.Certificate, *x509.Certificate, error) {
	// The issuer can use her own cert to sign records
	if certCid.Equals(issuerCertCid) {
		ct, err := v.m.GetCertificate(ctx, certCid)
		if err != nil {
			log.Warningf("Failed to get Certificate [%s]", certCid)
			return nil, nil, err
		}
		return ct, ct, nil
	}

	// If the issuer and cert cids are different, get them in parallel
	var cert, issuerCert *x509.Certificate
	resp := make(chan error, 2)

	getCert := func(cid *cid.Cid, cType string, cPtr **x509.Certificate) {
		ct, err := v.m.GetCertificate(ctx, cid)
		if err != nil {
			log.Warningf("Failed to get %s [%s]", cType, cid)
			resp <- err
			return
		}

		*cPtr = ct
		resp <- nil
	}

	go getCert(certCid, "Certificate", &cert)
	go getCert(issuerCertCid, "Issuer Certificate", &issuerCert)

	var err error
	for i := 0; i < 2; i++ {
		err = <-resp
		if err != nil {
			return nil, nil, err
		}
	}

	return cert, issuerCert, nil
}

func init() {
	VerificationSigPreparer[ld.VerificationType_Cert] = prepareCertSig
}

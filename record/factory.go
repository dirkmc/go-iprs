package iprs_record

import (
	"context"
	"crypto/x509"
	"crypto/rsa"
	"fmt"
	"time"
	c "github.com/dirkmc/go-iprs/certificate"
	ci "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	path "github.com/ipfs/go-ipfs/path"
	pb "github.com/dirkmc/go-iprs/pb"
	routing "gx/ipfs/QmPR2JzfKd9poHx9XBhzoFeBBC31ZM3W5iUPKJZWyaoZZm/go-libp2p-routing"
	rsp "github.com/dirkmc/go-iprs/path"
)

type RecordFactory struct {
	r routing.ValueStore
	pkm *PublicKeyManager
	certm *c.CertificateManager
	verifiers map[pb.IprsEntry_VerificationType]RecordVerifier
}

func NewRecordFactory(r routing.ValueStore) *RecordFactory {
	pkm := NewPublicKeyManager(r)
	certm := c.NewCertificateManager(r)

	verifiers := make(map[pb.IprsEntry_VerificationType]RecordVerifier)
	verifiers[pb.IprsEntry_Key] = NewKeyRecordVerifier(pkm)
	verifiers[pb.IprsEntry_Cert] = NewCertRecordVerifier(certm)

	return &RecordFactory{
		r: r,
		pkm: pkm,
		certm: certm,
		verifiers: verifiers,
	}
}

// Verifies that the given record is correctly signed etc
func (f *RecordFactory) Verify(ctx context.Context, iprsKey rsp.IprsPath, entry *pb.IprsEntry) error {
	verifier, ok := f.verifiers[entry.GetVerificationType()]
	if !ok {
		return fmt.Errorf("Unrecognized validity type %s", entry.GetVerificationType().String())
	}
	return verifier.VerifyRecord(ctx, iprsKey, entry)
}

func (f *RecordFactory) NewKeyRecordSigner(pk ci.PrivKey) *KeyRecordSigner {
	return NewKeyRecordSigner(f.pkm, pk)
}

func (f *RecordFactory) NewCertRecordSigner(cert *x509.Certificate, pk *rsa.PrivateKey) *CertRecordSigner {
	return NewCertRecordSigner(f.certm, cert, pk)
}

func (f *RecordFactory) NewRecord(vl RecordValidity, s RecordSigner, p path.Path) *Record {
	return NewRecord(f.r, vl, s, p)
}

func (f *RecordFactory) NewEolKeyRecord(p path.Path, pk ci.PrivKey, eol time.Time) *Record {
	vl := NewEolRecordValidity(eol)
	s := f.NewKeyRecordSigner(pk)
	return f.NewRecord(vl, s, p)
}

func (f *RecordFactory) NewEolCertRecord(p path.Path, cert *x509.Certificate, pk *rsa.PrivateKey, eol time.Time) *Record {
	vl := NewEolRecordValidity(eol)
	s := f.NewCertRecordSigner(cert, pk)
	return f.NewRecord(vl, s, p)
}

func (f *RecordFactory) NewRangeKeyRecord(p path.Path, pk ci.PrivKey, start, end *time.Time) (*Record, error) {
	vl, err := NewRangeRecordValidity(start, end)
	if err != nil {
		return nil, err
	}
	s := f.NewKeyRecordSigner(pk)
	return f.NewRecord(vl, s, p), nil
}

func (f *RecordFactory) NewRangeCertRecord(p path.Path, cert *x509.Certificate, pk *rsa.PrivateKey, start, end *time.Time) (*Record, error) {
	vl, err := NewRangeRecordValidity(start, end)
	if err != nil {
		return nil, err
	}
	s := f.NewCertRecordSigner(cert, pk)
	return f.NewRecord(vl, s, p), nil
}
